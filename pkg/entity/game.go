package entity

import (
	"time"

	pb "github.com/domino14/crosswords/rpc/api/proto"
	"github.com/domino14/macondo/game"
	"github.com/rs/zerolog/log"
)

type Game struct {
	game.Game
	// timeOfLastMove is the timestamp of the last made move, in milliseconds.
	// If no move has been made, this defaults to timeStarted.
	timeOfLastMove int64
	// timeRemaining is an array of remaining time per player, in milliseconds.
	timeRemaining []int
	// timeStarted is a unix timestamp, in milliseconds.
	timeStarted int64

	// perTurnIncrement, in seconds
	perTurnIncrement int
}

func msTimestamp() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

// NewGame takes in a Macondo game that _just started_. So it must log the
// current timestamp asap.
func NewGame(mcg *game.Game, perPlayerTimeSecs, perTurnIncrement int) *Game {
	started := msTimestamp()
	ms := perPlayerTimeSecs * 1000
	return &Game{
		Game:             *mcg,
		timeRemaining:    []int{ms, ms},
		timeOfLastMove:   started,
		perTurnIncrement: perTurnIncrement,
		timeStarted:      started,
	}
}

func (g *Game) TimeRemaining(idx int) int {
	return g.timeRemaining[idx]
}

func (g *Game) TimeStarted() int64 {
	return g.timeStarted
}

func (g *Game) PerTurnIncrement() int {
	return g.perTurnIncrement
}

// calculateTimeRemaining calculates the remaining time for the given player.
// It must be called after every move!
func (g *Game) calculateTimeRemaining(idx int) {
	now := msTimestamp()
	if g.Game.PlayerOnTurn() == idx {
		// Time has passed since this was calculated.
		g.timeRemaining[idx] -= int(now - g.timeOfLastMove)
	}
	// Otherwise, the player is not on turn, so their time should not
	// have changed. Do nothing.
}

func (g *Game) RecordTimeOfMove(idx int) {
	now := msTimestamp()
	log.Debug().Int64("started", g.timeStarted).Int64("now", now).Int("player", idx).Msg("record time of move")
	// How much time passed since the last made move?
	elapsed := int(now - g.timeOfLastMove)
	g.timeRemaining[idx] -= elapsed
	g.timeOfLastMove = now
}

func (g *Game) HistoryRefresherEvent() *EventWrapper {
	g.calculateTimeRemaining(0)
	g.calculateTimeRemaining(1)
	return NewEventWrapper("GameHistoryRefresher", &pb.GameHistoryRefresher{
		History:     g.History(),
		TimePlayer1: int32(g.TimeRemaining(0)),
		TimePlayer2: int32(g.TimeRemaining(1)),
	})
}
