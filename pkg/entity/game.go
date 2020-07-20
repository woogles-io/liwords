package entity

import (
	"sync"
	"time"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
)

// A Game should be saved to the database or store. It wraps a macondo.Game,
// and we should save most of the included fields here, especially the
// macondo.game.History (which can be exported as GCG, etc in the future)
type Game struct {
	sync.Mutex
	game.Game

	// timeOfLastUpdate is the timestamp of the last update, in milliseconds.
	// If no update has been made, this defaults to timeStarted.
	timeOfLastUpdate int64
	// timeRemaining is an array of remaining time per player, in milliseconds.
	timeRemaining []int
	// timeStarted is a unix timestamp, in milliseconds.
	timeStarted int64

	// perTurnIncrement, in seconds
	perTurnIncrement int
	// maxOvertime, in minutes
	maxOvertime int
	gamereq     *pb.GameRequest
	// started is set when the game actually starts (when the game timers start).
	// Note that the internal game.Game may have started a few seconds before,
	// but there should be no information about it given until _this_ started
	// is true.
	started bool

	gameEndReason pb.GameEndReason
	// if 0 or 1, that player won
	// if -1, it was a tie!
	winnerIdx int
	loserIdx  int

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
		Game:             *mcg,
		timeRemaining:    []int{ms, ms},
		perTurnIncrement: int(req.IncrementSeconds),
		maxOvertime:      int(req.MaxOvertimeMinutes),
		gamereq:          req,
	}
}

// Reset timers to _now_. The game is actually starting.
func (g *Game) ResetTimersAndStart() {
	log.Debug().Msg("reset-timers")
	ts := msTimestamp()
	g.timeOfLastUpdate = ts
	g.timeStarted = ts
	g.started = true
}

func (g *Game) RatingKey() (VariantKey, error) {
	req := g.CreationRequest()
	timefmt, variant, err := VariantFromGameReq(req)
	if err != nil {
		return "", err
	}
	return ToVariantKey(req.Lexicon, variant, timefmt), nil
}

// TimeRemaining calculates the time remaining, but does NOT update it.
func (g *Game) TimeRemaining(idx int) int {
	if g.Game.PlayerOnTurn() == idx {
		now := msTimestamp()
		return g.timeRemaining[idx] - int(now-g.timeOfLastUpdate)
	}
	// If the player is not on turn just return whatever the "cache" says.
	return g.timeRemaining[idx]
}

func (g *Game) CachedTimeRemaining(idx int) int {
	return g.timeRemaining[idx]
}

// TimeRanOut calculates if time ran out for the given player. Assumes player is
// on turn, otherwise it always returns false.
func (g *Game) TimeRanOut(idx int) bool {
	if g.Game.PlayerOnTurn() != idx {
		return false
	}
	now := msTimestamp()
	tr := g.timeRemaining[idx] - int(now-g.timeOfLastUpdate)
	return tr < (-g.maxOvertime * 60000)
}

func (g *Game) TimeStarted() int64 {
	return g.timeStarted
}

func (g *Game) Started() bool {
	return g.started
}

func (g *Game) PerTurnIncrement() int {
	return g.perTurnIncrement
}

func (g *Game) GameID() string {
	return g.Game.History().Uid
}

// calculateTimeRemaining calculates the remaining time for the given player.
func (g *Game) calculateAndSetTimeRemaining(pidx int, now int64) {
	log.Debug().
		Int64("started", g.timeStarted).
		Int64("now", now).
		Int64("lastupdate", g.timeOfLastUpdate).
		Int("player", pidx).
		Int("remaining", g.timeRemaining[pidx]).
		Msg("calculate-and-set-remaining")

	if g.Game.PlayerOnTurn() == pidx {
		// Time has passed since this was calculated.
		g.timeRemaining[pidx] -= int(now - g.timeOfLastUpdate)
		g.timeOfLastUpdate = now
		log.Debug().Int("actual-remaining", g.timeRemaining[pidx]).
			Msg("player-on-turn")
	}
	// Otherwise, the player is not on turn, so their time should not
	// have changed. Do nothing.

}

func (g *Game) RecordTimeOfMove(idx int) {
	now := msTimestamp()
	g.calculateAndSetTimeRemaining(idx, now)
}

// Return a HistoryRefresherEvent for the given user ID. If sanitize is
// true, opponent racks are stripped.
func (g *Game) HistoryRefresherEvent( /*userID string, sanitize bool*/ ) *pb.GameHistoryRefresher {
	now := msTimestamp()

	g.calculateAndSetTimeRemaining(0, now)
	g.calculateAndSetTimeRemaining(1, now)

	return &pb.GameHistoryRefresher{
		History:     g.History(),
		TimePlayer1: int32(g.TimeRemaining(0)),
		TimePlayer2: int32(g.TimeRemaining(1)),
	}
}

func (g *Game) ChallengeRule() macondopb.ChallengeRule {
	return g.gamereq.ChallengeRule
}

func (g *Game) RatingMode() pb.RatingMode {
	return g.gamereq.RatingMode
}

func (g *Game) CreationRequest() *pb.GameRequest {
	return g.gamereq
}

// RegisterChangeHook registers a channel with the game. Events will
// be sent down this channel.
func (g *Game) RegisterChangeHook(eventChan chan<- *EventWrapper) error {
	g.changeHook = eventChan
	return nil
}

// SendChange sends an event via the registered hook.
func (g *Game) SendChange(e *EventWrapper) {
	log.Debug().Interface("evt", e.Event).Interface("aud", e.Audience()).Msg("send-change")
	g.changeHook <- e
}

func (g *Game) GameEndReason() pb.GameEndReason {
	return g.gameEndReason
}

func (g *Game) SetGameEndReason(r pb.GameEndReason) {
	g.gameEndReason = r
}

func (g *Game) SetWinnerIdx(pidx int) {
	g.winnerIdx = pidx
}

func (g *Game) SetLoserIdx(pidx int) {
	g.loserIdx = pidx
}

func (g *Game) GetWinnerIdx() int {
	return g.winnerIdx
}

func (g *Game) WinnerWasSet() bool {
	// This is the only case in which the winner has not yet been set,
	// when both winnerIdx and loserIdx are 0.
	return !(g.winnerIdx == 0 && g.loserIdx == 0)
}
