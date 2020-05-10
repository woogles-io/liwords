// Package game should know nothing about protocols or databases.
// It is mostly a pass-through interface to a Macondo game,
// but also implements a timer and other related logic.
// This is a use-case in the clean architecture hierarchy.
package game

import (
	"context"
	"errors"

	"github.com/domino14/crosswords/pkg/entity"
	pb "github.com/domino14/crosswords/rpc/api/proto"
	"github.com/domino14/macondo/board"
	"github.com/domino14/macondo/config"
	"github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

const (
	CrosswordGame string = "CrosswordGame"
)

// Store is an interface for getting a full game.
type store interface {
	Get(ctx context.Context, id string) (*entity.Game, error)
	Set(context.Context, *entity.Game) error
	// RegisterChangeHook registers a channel with the store. Events will
	// be sent down this channel.
	RegisterChangeHook(gameID string, eventChan chan *entity.EventWrapper) error
}

// type Game interface {
// }

// StartNewGame instantiates a game and starts the timer.
func StartNewGame(ctx context.Context, gameStore store, cfg *config.Config,
	players []*macondopb.PlayerInfo, req *pb.GameRequest,
	eventChan chan *entity.EventWrapper) (string, error) {

	var bd []string
	switch req.Rules.BoardLayoutName {
	case CrosswordGame:
		bd = board.CrosswordGameBoard
	default:
		return "", errors.New("unsupported board layout")
	}

	rules, err := game.NewGameRules(cfg, bd,
		req.Lexicon, req.Rules.LetterDistributionName)

	if err != nil {
		return "", err
	}
	g, err := game.NewGame(rules, players)
	if err != nil {
		return "", err
	}
	// StartGame sets a new history Uid and actually starts the game.
	g.StartGame()
	entGame := entity.NewGame(g, int(req.InitialTimeSeconds), int(req.IncrementSeconds))
	if err = gameStore.Set(ctx, entGame); err != nil {
		return "", err
	}
	gameID := g.History().Uid
	if err = gameStore.RegisterChangeHook(gameID, eventChan); err != nil {
		return "", err
	}
	eventChan <- entity.NewGameHistoryEvent(entGame.HistoryRefresher())

	return gameID, nil
}

func PlayMove(ctx context.Context, gameStore store, gameID string, player string,
	evt *pb.UserGameplayEvent) error {

}
