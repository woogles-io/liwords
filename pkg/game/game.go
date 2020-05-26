// Package game should know nothing about protocols or databases.
// It is mostly a pass-through interface to a Macondo game,
// but also implements a timer and other related logic.
// This is a use-case in the clean architecture hierarchy.
package game

import (
	"context"
	"errors"

	"github.com/domino14/macondo/alphabet"

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
	// StartGame creates a new history Uid and actually starts the game.
	g.StartGame()
	entGame := entity.NewGame(g, req)
	if err = gameStore.Set(ctx, entGame); err != nil {
		return nil, err
	}
	return entGame, nil
	// We return the instantiated game. Although the tiles have technically been
	// dealt out, we need to call StartGameInstance to actually start the timer
	// and forward game events to the right channels.

}

func StartGameInstance(entGame *entity.Game, eventChan chan<- *entity.EventWrapper) error {
	if err := entGame.RegisterChangeHook(eventChan); err != nil {
		return err
	}
	entGame.SendChange(entity.WrapEvent(entGame.HistoryRefresherEvent(), entGame.GameID()))

	return nil
}

func clientEventToGameEvent(cge *pb.ClientGameplayEvent, g *game.Game) (*macondopb.GameEvent, error) {
	playerid := g.PlayerOnTurn()
	rack := g.RackFor(playerid)

	switch cge.Type {
	case pb.ClientGameplayEvent_TILE_PLACEMENT:
		m, err := g.CreateAndScorePlacementMove(cge.PositionCoords, cge.Tiles, rack.String())
		if err != nil {
			return nil, err
		}
		// Note that we don't validate the move here, but we do so later.
		return g.EventFromMove(m), nil

	case pb.ClientGameplayEvent_CHALLENGE_PLAY:
		// XXX: THIS NEEDS TO BE HANDLED IN SOME OTHER WAY.
	case pb.ClientGameplayEvent_PASS:
		m := move.NewPassMove(rack.TilesOn(), g.Alphabet())
		return g.EventFromMove(m), nil
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
		return g.EventFromMove(m), nil
	}
	return nil, errors.New("client gameplay event not handled")
}

func PlayMove(ctx context.Context, gameStore GameStore, player string,
	cge *pb.ClientGameplayEvent) error {

	// XXX: VERIFY THAT THE CLIENT GAME ID CORRESPONDS TO THE GAME
	// THE PLAYER IS PLAYING!
	entGame, err := gameStore.Get(ctx, cge.GameId)
	if err != nil {
		return err
	}
	if !entGame.Game.Playing() {
		return errGameNotActive
	}
	onTurn := entGame.Game.PlayerOnTurn()

	// Ensure that it is actually the correct player's turn
	if entGame.Game.NickOnTurn() != player {
		return errNotOnTurn
	}

	// Turn the event into a macondo GameEvent.
	evt, err := clientEventToGameEvent(cge, &entGame.Game)
	if err != nil {
		return err
	}

	m := game.MoveFromEvent(evt, entGame.Game.Alphabet(), entGame.Game.Board())

	wordsFormed, err := entGame.Game.ValidateMove(m)
	if err != nil {
		return err
	}
	entGame.SetLastPlayedWords(wordsFormed)

	// XXX: Depending on the challenge rule, there can be two validation steps:
	// 1. validate that the play is legal in the game (connects to tiles, etc)
	// 2. validate that the play creates actual words if the challenge rule is VOID.
	if entGame.ChallengeRule() == pb.ChallengeRule_VOID {
		// validate wordsFormed here and return an error if any word is
		// invalid in the game's lexicon.
	}
	// Don't back up the move, but add to history
	err = entGame.Game.PlayMove(m, false, true)
	if err != nil {
		return err
	}

	// Register time.
	entGame.RecordTimeOfMove(onTurn)

	// Create a ServerGameplayEvent to send back.
	sge := &pb.ServerGameplayEvent{}
	sge.Event = evt
	sge.GameId = cge.GameId
	sge.TimeRemaining = int32(entGame.TimeRemaining(onTurn))
	sge.NewRack = entGame.Game.RackLettersFor(onTurn)
	// Since the move was successful, we assume the user gameplay event is valid.
	// Re-send it, but overwrite the time remaining and new rack properly.
	playing := entGame.Game.Playing()

	err = gameStore.Set(ctx, entGame)
	if err != nil {
		return err
	}

	entGame.SendChange(entity.WrapEvent(sge, entGame.GameID()))
	if !playing {
		performEndgameDuties(entGame, pb.GameEndReason_WENT_OUT, player)
	}
	return nil
}

func performEndgameDuties(g *entity.Game, reason pb.GameEndReason, player string) {
	// figure out ratings later lol
	// if g.RatingMode() == pb.RatingMode_RATED {
	// 	ratings :=
	// }

	g.SendChange(
		entity.WrapEvent(g.GameEndedEvent(pb.GameEndReason_WENT_OUT, player), g.GameID()))

}
