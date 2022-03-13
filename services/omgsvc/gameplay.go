package omgsvc

import (
	"context"
	"errors"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/macondo/alphabet"
	mconfig "github.com/domino14/macondo/config"
	"github.com/domino14/macondo/game"
	mgame "github.com/domino14/macondo/game"
	"github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/macondo/move"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/ipc"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
)

var (
	errGameNotActive   = errors.New("game is not currently active")
	errNotOnTurn       = errors.New("player not on turn")
	errTimeDidntRunOut = errors.New("got time ran out, but it did not actually")
)

func clientEventToMove(cge *pb.ClientGameplayEvent, g *game.Game) (*move.Move, error) {
	playerid := g.PlayerOnTurn()
	rack := g.RackFor(playerid)

	switch cge.Type {
	case pb.ClientGameplayEvent_TILE_PLACEMENT:
		m, err := g.CreateAndScorePlacementMove(cge.PositionCoords, cge.Tiles, rack.String())
		if err != nil {
			return nil, err
		}
		log.Debug().Msg("got a client gameplay event tile placement")
		// Note that we don't validate the move here, but we do so later.
		return m, nil

	case pb.ClientGameplayEvent_PASS:
		m := move.NewPassMove(rack.TilesOn(), g.Alphabet())
		return m, nil
	case pb.ClientGameplayEvent_EXCHANGE:
		tiles, err := alphabet.ToMachineWord(cge.Tiles, g.Alphabet())
		if err != nil {
			return nil, err
		}
		leaveMW, err := game.Leave(rack.TilesOn(), tiles)
		if err != nil {
			return nil, err
		}
		m := move.NewExchangeMove(tiles, leaveMW, g.Alphabet())
		return m, nil

	case pb.ClientGameplayEvent_CHALLENGE_PLAY:
		m := move.NewChallengeMove(rack.TilesOn(), g.Alphabet())
		return m, nil
	}
	return nil, errors.New("client gameplay event not handled")
}

func loadGameFromResponse(cfg *mconfig.Config, resp *pb.GetGameResponse) (*mgame.Game, error) {
	rules, err := mgame.NewBasicGameRules(cfg, resp.Lexicon,
		resp.BoardLayoutName, resp.LetterDistributionName,
		mgame.CrossScoreOnly, mgame.Variant(resp.VariantName))
	if err != nil {
		return nil, err
	}

	// There's a chance the game is over, so we want to get that state before
	// the following function modifies it.
	hist := resp.GameHistoryRefresher.History
	histPlayState := hist.GetPlayState()
	// This function modifies the history. (XXX it probably shouldn't)
	// It modifies the play state as it plays the game from the beginning.
	mcg, err := mgame.NewFromHistory(hist, rules, len(hist.Events))
	if err != nil {
		return nil, err
	}
	// XXX: We should probably move this to `NewFromHistory`:
	mcg.SetBackupMode(mgame.InteractiveGameplayMode)
	// Note: we don't need to set the stack length here, as NewFromHistory
	// above does it.

	// Finally, restore the play state from the passed-in history. This
	// might immediately end the game (for example, the game could have timed
	// out, but the NewFromHistory function doesn't actually handle that).
	// We could consider changing NewFromHistory, but we want it to be as
	// flexible as possible for things like analysis mode.
	mcg.SetPlaying(histPlayState)
	mcg.History().PlayState = histPlayState
	return mcg, nil
}

func handleGameplayEvent(ctx context.Context, b ipc.Publisher, userID string, data []byte,
	reply string) error {

	cge := &pb.ClientGameplayEvent{}
	err := proto.Unmarshal(data, cge)
	if err != nil {
		return err
	}
	// get the "nower" - the module that determines what "Now" is.
	nower, err := getNower(ctx)
	if err != nil {
		return err
	}

	g, timers, err := gameFromGameplayEvent(ctx, cge, b)
	if err != nil {
		return err
	}

	log := zerolog.Ctx(ctx).With().Str("gameID", cge.GameId).Logger()

	if g.Playing() == macondo.PlayState_GAME_OVER {
		return errGameNotActive
	}

	onTurn := g.PlayerOnTurn()
	// Ensure that it is actually the correct player's turn
	if cge.Type != pb.ClientGameplayEvent_RESIGN && g.PlayerIDOnTurn() != userID {
		log.Info().Interface("client-event", cge).Msg("not on turn")
		return errNotOnTurn
	}
	tr := timeRemaining(timers, nower, onTurn, onTurn)
	gameUpdates := &pb.SetGame{}

	if !(g.Playing() == macondo.PlayState_WAITING_FOR_FINAL_PASS &&
		cge.Type == pb.ClientGameplayEvent_PASS &&
		timeRanOut(timers, nower, onTurn, onTurn)) {
		log.Info().Msg("got-move-too-late")

		if g.Playing() == macondo.PlayState_WAITING_FOR_FINAL_PASS {
			log.Info().Msg("timed out, so passing instead of processing the submitted move")
			cge = &pb.ClientGameplayEvent{
				Type:   pb.ClientGameplayEvent_PASS,
				GameId: cge.GameId,
			}
		} else {
			return setTimedOut(ctx, g, onTurn)
		}
	}

	log.Debug().Msg("going to turn into a macondo gameevent")

	if cge.Type == pb.ClientGameplayEvent_RESIGN {
		// XXX: set game end reason to RESIGNED
		// XXX: Record time of move

		winner := 1 - onTurn
		// If opponent is the one who resigned, current player wins.
		if g.PlayerIDOnTurn() != userID {
			winner = onTurn
		}
		// XXX: set history winner to winner
		// XXX: set winner idx and loser idx
		// XXX: perform end game duties
	} else {
		m, err := clientEventToMove(cge, g)
		if err != nil {
			return err
		}
		err = PlayMove(ctx, g, userID, onTurn, timeRemaining, m)
		if err != nil {
			return err
		}
	}

	// if endgame reason == pb.GameEndReason_NONE {
	// XXX: cancel  meta events
	//}

	return nil
}

func gameFromGameplayEvent(ctx context.Context, cge *pb.ClientGameplayEvent,
	b ipc.Publisher) (*mgame.Game, *pb.Timers, error) {

	macondoConfig, err := config.GetMacondoConfig(ctx)
	if err != nil {
		return nil, nil, err
	}

	getGame := &pb.GetGame{Id: cge.GameId}
	getGameResponse := &pb.GetGameResponse{}

	err = ipc.RequestProto("storesvc.omgwords.getgame", b, getGame, getGameResponse)
	if err != nil {
		return nil, nil, err
	}
	g, err := loadGameFromResponse(macondoConfig, getGameResponse)
	if err != nil {
		return nil, nil, err
	}
	return g, getGameResponse.Timers, nil
}
