// Package game should know nothing about protocols or databases.
// It is mostly a pass-through interface to a Macondo game,
// but also implements a timer and other related logic.
// This is a use-case in the clean architecture hierarchy.
package gameplay

import (
	"context"
	"errors"
	"fmt"

	"github.com/domino14/macondo/alphabet"
	"github.com/rs/zerolog/log"

	"github.com/domino14/macondo/move"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/domino14/crosswords/pkg/config"
	"github.com/domino14/crosswords/pkg/entity"
	pb "github.com/domino14/crosswords/rpc/api/proto"
	"github.com/domino14/macondo/board"
	"github.com/domino14/macondo/game"
)

const (
	CrosswordGame string = "CrosswordGame"
)

var (
	errGameNotActive = errors.New("game is not currently active")
	errNotOnTurn     = errors.New("player not on turn")
)

// GameStore is an interface for getting a full game.
type GameStore interface {
	Get(ctx context.Context, id string) (*entity.Game, error)
	Set(context.Context, *entity.Game) error
}

// type Game interface {
// }

// InstantiateNewGame instantiates a game and returns it.
func InstantiateNewGame(ctx context.Context, gameStore GameStore, cfg *config.Config,
	players []*macondopb.PlayerInfo, req *pb.GameRequest) (*entity.Game, error) {

	var bd []string
	if req.Rules == nil {
		return nil, errors.New("no rules")
	}
	switch req.Rules.BoardLayoutName {
	case CrosswordGame:
		bd = board.CrosswordGameBoard
	default:
		return nil, errors.New("unsupported board layout")
	}

	rules, err := game.NewGameRules(&cfg.MacondoConfig, bd,
		req.Lexicon, req.Rules.LetterDistributionName)

	if err != nil {
		return nil, err
	}
	g, err := game.NewGame(rules, players)
	if err != nil {
		return nil, err
	}
	// StartGame creates a new history Uid and deals tiles, etc.
	g.StartGame()
	g.SetBackupMode(game.InteractiveGameplayMode)
	g.SetChallengeRule(req.ChallengeRule)

	entGame := entity.NewGame(g, req)
	// Save the game to the store.
	if err = gameStore.Set(ctx, entGame); err != nil {
		return nil, err
	}
	return entGame, nil
	// We return the instantiated game. Although the tiles have technically been
	// dealt out, we need to call StartGame to actually start the timer
	// and forward game events to the right channels.
}

// func StartGameInstance(entGame *entity.Game, eventChan chan<- *entity.EventWrapper) error {
// 	if err := entGame.RegisterChangeHook(eventChan); err != nil {
// 		return err
// 	}
// 	entGame.
// 	entGame.SendChange(entity.WrapEvent(entGame.HistoryRefresherEvent(), pb.MessageType_GAME_HISTORY_REFRESHER,
// 		entGame.GameID()))

// 	return nil
// }

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
	}
	return nil, errors.New("client gameplay event not handled")
}

func StartGame(ctx context.Context, gameStore GameStore, eventChan chan<- *entity.EventWrapper, id string) error {
	// Note that StartGame does _not_ start the Macondo game, which
	// has already started, but we don't "know" that. It is _this_
	// function that will actually start the game in the user's eyes.
	// It needs to reset the timer to now.
	entGame, err := gameStore.Get(ctx, id)
	if err != nil {
		return err
	}
	// This should be True, see comment above.
	if entGame.Game.Playing() != game.StatePlaying {
		return errGameNotActive
	}
	log.Debug().Str("gameid", id).Msg("reset timers (and start)")
	entGame.ResetTimers()

	// Save the game back to the store always.
	if err := gameStore.Set(ctx, entGame); err != nil {
		return err
	}
	if err := entGame.RegisterChangeHook(eventChan); err != nil {
		return err
	}

	entGame.SendChange(
		entity.WrapEvent(entGame.HistoryRefresherEvent(),
			pb.MessageType_GAME_HISTORY_REFRESHER,
			entGame.GameID()))

	return nil
}

func handleChallenge(ctx context.Context, entGame *entity.Game, gameStore GameStore) error {
	if entGame.ChallengeRule() == macondopb.ChallengeRule_VOID {
		// The front-end shouldn't even show the button.
		return errors.New("challenges not acceptable in void")
	}
	valid, err := entGame.Game.ChallengeEvent(0)
	if err != nil {
		return err
	}

	turns := entGame.Game.History().Turns
	if len(turns) == 0 {
		return errors.New("unexpected turn length")
	}
	events := entGame.Game.History().Turns[len(turns)-1].Events

	if !valid {
		// play is invalid, tiles and board go back to previous state.
		if len(events) != 2 {
			return fmt.Errorf("(play invalid) mismatch: expected event length 2, got %d", len(events))
		}
		// onTurn is the challenger at this point.
		onTurn := entGame.Game.PlayerOnTurn()
		sge := &pb.ServerGameplayEvent{
			Event:   events[1], // This is the "phony tiles returned" event.
			GameId:  entGame.GameID(),
			NewRack: events[1].Rack,
			// This is going to be the time remaining for the _challenger_
			TimeRemaining: int32(entGame.TimeRemaining(onTurn)),
		}
		entGame.SendChange(entity.WrapEvent(sge, pb.MessageType_SERVER_GAMEPLAY_EVENT,
			entGame.GameID()))
	} else {
		// The play is valid, there was either some bonus awarded, or someone
		// lost their turn, etc.
		// if challenge rule is double, then the challenger loses their turn
		// (which would create a new turn and event)
		// otherwise, it is still the challenger's turn, and we should fetch
		// the bonus.
		// note that in _both_ cases, we are fetching everything after the first event.

		// XXX: No -- I have to handle the special cases when the player goes out.
		// (the turn doesn't switch, etc)
		// Seems illogical to handle in both places (macondo and here)
		// Consider having a gameplay event instead be a full snapshot of the board.
		// seems aliunde though, but maybe do this only in special cases.
		// (after challenges?)
		onTurn := entGame.Game.PlayerOnTurn()
		if entGame.ChallengeRule() == macondopb.ChallengeRule_DOUBLE {

			if len(events) != 1 {
				return fmt.Errorf("(play valid) mismatch: expected event length 1, got %d", len(events))
			}
		} else {
			if len(events) != 2 {
				return fmt.Errorf("(play valid) mismatch: expected event length 2, got %d", len(events))
			}
		}
		entGame.NickOnTurn()

		sge := &pb.ServerGameplayEvent{
			Event:  events[len(events)-1],
			GameId: entGame.GameID(),
			// The front end should ignore this:
			NewRack: "",
			// This is going to be the time remaining for either player,
			// whoever ends up being on turn after the challenge. Front
			// end should disambiguate this.
			TimeRemaining: int32(entGame.TimeRemaining(onTurn)),
		}
		entGame.SendChange(entity.WrapEvent(sge, pb.MessageType_SERVER_GAMEPLAY_EVENT,
			entGame.GameID()))
	}

	err = gameStore.Set(ctx, entGame)
	if err != nil {
		return err
	}

	return nil
}

func PlayMove(ctx context.Context, gameStore GameStore, player string,
	cge *pb.ClientGameplayEvent) error {

	// XXX: VERIFY THAT THE CLIENT GAME ID CORRESPONDS TO THE GAME
	// THE PLAYER IS PLAYING!
	entGame, err := gameStore.Get(ctx, cge.GameId)
	if err != nil {
		return err
	}
	if entGame.Game.Playing() == game.StateGameOver {
		return errGameNotActive
	}
	onTurn := entGame.Game.PlayerOnTurn()

	// Ensure that it is actually the correct player's turn
	if entGame.Game.NickOnTurn() != player {
		return errNotOnTurn
	}

	log.Debug().Msg("going to turn into a macondo gameevent")

	if cge.Type == pb.ClientGameplayEvent_CHALLENGE_PLAY {
		// Handle in another way
		return handleChallenge(ctx, gameStore, entGame)
	}

	// Turn the event into a macondo GameEvent.
	m, err := clientEventToMove(cge, &entGame.Game)
	if err != nil {
		return err
	}
	// m := game.MoveFromEvent(evt, entGame.Game.Alphabet(), entGame.Game.Board())
	log.Debug().Msg("validating")

	_, err = entGame.Game.ValidateMove(m)
	if err != nil {
		return err
	}

	// Don't back up the move, but add to history
	log.Debug().Msg("playing the move")
	err = entGame.Game.PlayMove(m, true)
	if err != nil {
		return err
	}
	// Get the turn that we _just_ appended to the history
	turnLength := len(entGame.Game.History().Turns)
	turn := entGame.Game.History().Turns[turnLength-1]
	// Register time.
	entGame.RecordTimeOfMove(onTurn)

	// Create a set of ServerGameplayEvents to send back.

	evts := make([]*pb.ServerGameplayEvent, len(turn.Events))

	for idx, evt := range turn.Events {

		sge := &pb.ServerGameplayEvent{}
		sge.Event = evt
		sge.GameId = cge.GameId
		// note that `onTurn` is correct as it was saved up there before
		// we played the turn.
		sge.TimeRemaining = int32(entGame.TimeRemaining(onTurn))
		sge.NewRack = entGame.Game.RackLettersFor(onTurn)
		evts[idx] = sge
	}
	// Since the move was successful, we assume the user gameplay event is valid.
	// Re-send it, but overwrite the time remaining and new rack properly.
	playing := entGame.Game.Playing()

	err = gameStore.Set(ctx, entGame)
	if err != nil {
		return err
	}
	for _, sge := range evts {
		entGame.SendChange(entity.WrapEvent(sge, pb.MessageType_SERVER_GAMEPLAY_EVENT,
			entGame.GameID()))
	}
	if playing == game.StateGameOver {
		performEndgameDuties(entGame, pb.GameEndReason_WENT_OUT, player)
	}
	return nil
}

func performEndgameDuties(g *entity.Game, reason pb.GameEndReason, player string) {
	g.SendChange(
		entity.WrapEvent(g.GameEndedEvent(pb.GameEndReason_WENT_OUT, player),
			pb.MessageType_GAME_ENDED_EVENT, g.GameID()))

}
