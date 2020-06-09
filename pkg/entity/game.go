package entity

import (
	"time"

	pb "github.com/domino14/liwords/rpc/api/proto"
	"github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
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
	gamereq          *pb.GameRequest

	changeHook chan<- *EventWrapper
}

func msTimestamp() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

// NewGame takes in a Macondo game that was just "started". Note that
// Macondo games when they start do not log any time, they just deal tiles.
// The time of start must be logged later, when both players are in the table
// and ready.
func NewGame(mcg *game.Game, req *pb.GameRequest) *Game {
	ms := int(req.InitialTimeSeconds * 1000)
	return &Game{
		Game:          *mcg,
		timeRemaining: []int{ms, ms},
		// timeOfLastMove:   started,
		perTurnIncrement: int(req.IncrementSeconds),
		// timeStarted:      started,
		gamereq: req,
	}
}

// Reset timers to _now_. The game is actually starting.
func (g *Game) ResetTimers() {
	ts := msTimestamp()
	g.timeOfLastMove = ts
	g.timeStarted = ts
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

func (g *Game) GameID() string {
	return g.Game.History().Uid
}

// calculateTimeRemaining calculates the remaining time for the given player.
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

func (g *Game) HistoryRefresherEvent() *pb.GameHistoryRefresher {
	g.calculateTimeRemaining(0)
	g.calculateTimeRemaining(1)
	return &pb.GameHistoryRefresher{
		History:     g.History(),
		TimePlayer1: int32(g.TimeRemaining(0)),
		TimePlayer2: int32(g.TimeRemaining(1)),
	}
}

func (g *Game) GameEndedEvent() *pb.GameEndedEvent {
	return &pb.GameEndedEvent{
		Scores: map[string]int32{
			g.History().Players[0].Nickname: int32(g.PointsFor(0)),
			g.History().Players[1].Nickname: int32(g.PointsFor(1))},
	}
}

func (g *Game) ChallengeRule() macondopb.ChallengeRule {
	return g.gamereq.ChallengeRule
}

func (g *Game) RatingMode() pb.RatingMode {
	return g.gamereq.RatingMode
}

// RegisterChangeHook registers a channel with the game. Events will
// be sent down this channel.
func (g *Game) RegisterChangeHook(eventChan chan<- *EventWrapper) error {
	g.changeHook = eventChan
	return nil
}

// SendChange sends an event via the registered hook.
func (g *Game) SendChange(e *EventWrapper) {
	g.changeHook <- e
}
