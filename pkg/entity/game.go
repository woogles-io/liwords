package entity

import (
	"time"

	gameservicepb "github.com/domino14/liwords/rpc/api/proto/game_service"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
)

const (
	CrosswordGame string = "CrosswordGame"
)

type Timers struct {
	// TimeOfLastUpdate is the timestamp of the last update, in milliseconds.
	// If no update has been made, this defaults to timeStarted.
	TimeOfLastUpdate int64 `json:"lu"`
	// TimeStarted is a unix timestamp, in milliseconds.
	TimeStarted int64 `json:"ts"`
	// TimeRemaining is an array of remaining time per player, in milliseconds.
	TimeRemaining []int `json:"tr"`
	// MaxOvertime is in minutes. All others are in milliseconds.
	MaxOvertime int `json:"mo"`
	// Nower is the name of the timer module. Should only change for tests.
	Nower string `json:"n,omitempty"`
}

// Nower is an interface for determining the current time
type Nower interface {
	// Now returns a timestamp in milliseconds
	Now() int64
	Name() string
}

// FakeNower uses a fake timer. It is used for tests so we don't actually sleep.
type FakeNower struct {
	fakeMeow int64
}

func NewFakeNower(f int64) *FakeNower {
	return &FakeNower{f}
}

// Now returns now's value
func (f FakeNower) Now() int64 {
	return f.fakeMeow
}

func (f FakeNower) Name() string {
	return "FakeNower"
}

// Sleep simulates a sleep.
func (f *FakeNower) Sleep(t int64) {
	f.fakeMeow += t
}

// Quickdata represents data that we might need quick access to, for the purposes
// of aggregating large numbers of games rapidly. This should get saved in
// its own blob in the store, as opposed to being buried within a game history.
type Quickdata struct {
	OriginalRequestId string                      `json:"o"`
	FinalScores       []int32                     `json:"s"`
	PlayerInfo        []*gameservicepb.PlayerInfo `json:"pi"`
	OriginalRatings   []float64
	NewRatings        []float64
}

// Holds the tournament data for a game.
// This is nil if the game is not a tournament game.
type TournamentData struct {
	Id        string
	Division  string `json:"d"`
	Round     int    `json:"r"`
	GameIndex int    `json:"i"`
}

// A Game should be saved to the database or store. It wraps a macondo.Game,
// and we should save most of the included fields here, especially the
// macondo.game.History (which can be exported as GCG, etc in the future)
type Game struct {
	game.Game

	PlayerDBIDs [2]uint // needed to associate the games to the player IDs in the db.

	GameReq *pb.GameRequest
	// started is set when the game actually starts (when the game timers start).
	// Note that the internal game.Game may have started a few seconds before,
	// but there should be no information about it given until _this_ started
	// is true.
	Started bool
	Timers  Timers

	GameEndReason pb.GameEndReason
	// if 0 or 1, that player won
	// if -1, it was a tie!
	WinnerIdx int
	LoserIdx  int

	Stats *Stats

	ChangeHook chan<- *EventWrapper
	nower      Nower

	Quickdata      *Quickdata
	TournamentData *TournamentData
	CreatedAt      time.Time
}

// GameTimer uses the standard library's `time` package to determine how much time
// has elapsed in a game.
type GameTimer struct{}

// Now returns the current timestamp in milliseconds.
func (g GameTimer) Now() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

func (g GameTimer) Name() string {
	return ""
}

// NewGame takes in a Macondo game that was just "started". Note that
// Macondo games when they start do not log any time, they just deal tiles.
// The time of start must be logged later, when both players are in the table
// and ready.
func NewGame(mcg *game.Game, req *pb.GameRequest) *Game {
	ms := int(req.InitialTimeSeconds * 1000)
	return &Game{
		Game: *mcg,
		Timers: Timers{
			TimeRemaining: []int{ms, ms},
			MaxOvertime:   int(req.MaxOvertimeMinutes),
		},
		GameReq:   req,
		nower:     &GameTimer{},
		Quickdata: &Quickdata{},
	}
}

// SetTimerModule sets the timer for a game to the given Nower.
func (g *Game) SetTimerModule(n Nower) {
	g.nower = n
	g.Timers.Nower = n.Name()
}

// Reset timers to _now_. The game is actually starting.
func (g *Game) ResetTimersAndStart() {
	log.Debug().Msg("reset-timers")
	ts := g.nower.Now()
	g.Timers.TimeOfLastUpdate = ts
	g.Timers.TimeStarted = ts
	g.Started = true
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
		now := g.nower.Now()
		return g.Timers.TimeRemaining[idx] - int(now-g.Timers.TimeOfLastUpdate)
	}
	// If the player is not on turn just return whatever the "cache" says.
	return g.Timers.TimeRemaining[idx]
}

func (g *Game) CachedTimeRemaining(idx int) int {
	return g.Timers.TimeRemaining[idx]
}

// TimeRanOut calculates if time ran out for the given player. Assumes player is
// on turn, otherwise it always returns false.
func (g *Game) TimeRanOut(idx int) bool {
	if g.Game.PlayerOnTurn() != idx {
		return false
	}
	now := g.nower.Now()
	tr := g.Timers.TimeRemaining[idx] - int(now-g.Timers.TimeOfLastUpdate)
	return tr < (-g.Timers.MaxOvertime * 60000)
}

func (g *Game) TimeStarted() int64 {
	return g.Timers.TimeStarted
}

// func (g *Game) PerTurnIncrement() int {
// 	return g.perTurnIncrement
// }

func (g *Game) GameID() string {
	return g.Game.History().Uid
}

// calculateTimeRemaining calculates the remaining time for the given player.
func (g *Game) calculateAndSetTimeRemaining(pidx int, now int64, accountForIncrement bool) {
	log.Debug().
		Int64("started", g.Timers.TimeStarted).
		Int64("now", now).
		Int64("lastupdate", g.Timers.TimeOfLastUpdate).
		Int("player", pidx).
		Int("remaining", g.Timers.TimeRemaining[pidx]).
		Msg("calculate-and-set-remaining")

	if g.Game.PlayerOnTurn() == pidx {
		// Time has passed since this was calculated.
		g.Timers.TimeRemaining[pidx] -= int(now - g.Timers.TimeOfLastUpdate)
		if accountForIncrement {
			g.Timers.TimeRemaining[pidx] += (int(g.GameReq.IncrementSeconds) * 1000)
		}

		// Cap the overtime, because auto-passing always happens after time has expired.
		maxOvertimeMs := g.Timers.MaxOvertime * 60000
		if g.Timers.TimeRemaining[pidx] < -maxOvertimeMs {
			log.Debug().Int("proposed-remaining", g.Timers.TimeRemaining[pidx]).
				Msg("calculate-and-set-remaining-capped")
			g.Timers.TimeRemaining[pidx] = -maxOvertimeMs
		}

		g.Timers.TimeOfLastUpdate = now
		log.Debug().Int("actual-remaining", g.Timers.TimeRemaining[pidx]).
			Msg("player-on-turn")
	}
	// Otherwise, the player is not on turn, so their time should not
	// have changed. Do nothing.

}

func (g *Game) RecordTimeOfMove(idx int) {
	now := g.nower.Now()
	g.calculateAndSetTimeRemaining(idx, now, true)
}

func (g *Game) HistoryRefresherEvent() *pb.GameHistoryRefresher {
	now := g.nower.Now()

	g.calculateAndSetTimeRemaining(0, now, false)
	g.calculateAndSetTimeRemaining(1, now, false)

	return &pb.GameHistoryRefresher{
		History:            g.History(),
		TimePlayer1:        int32(g.TimeRemaining(0)),
		TimePlayer2:        int32(g.TimeRemaining(1)),
		MaxOvertimeMinutes: g.GameReq.MaxOvertimeMinutes,
	}
}

func (g *Game) ChallengeRule() macondopb.ChallengeRule {
	return g.GameReq.ChallengeRule
}

func (g *Game) RatingMode() pb.RatingMode {
	return g.GameReq.RatingMode
}

func (g *Game) CreationRequest() *pb.GameRequest {
	return g.GameReq
}

// RegisterChangeHook registers a channel with the game. Events will
// be sent down this channel.
func (g *Game) RegisterChangeHook(eventChan chan<- *EventWrapper) error {
	log.Debug().Msgf("register-change-hook: %v", eventChan)
	g.ChangeHook = eventChan
	return nil
}

// SendChange sends an event via the registered hook.
func (g *Game) SendChange(e *EventWrapper) {
	log.Debug().Interface("evt", e.Event).Interface("aud", e.Audience()).Msg("send-change")
	if g.ChangeHook == nil {
		// This should never happen in actual operation; consider making it a Fatal.
		log.Error().Msg("change hook is closed!")
		return
	}
	g.ChangeHook <- e
	log.Debug().Msg("change sent")
}

func (g *Game) SetGameEndReason(r pb.GameEndReason) {
	g.GameEndReason = r
}

func (g *Game) SetWinnerIdx(pidx int) {
	g.WinnerIdx = pidx
}

func (g *Game) SetLoserIdx(pidx int) {
	g.LoserIdx = pidx
}

func (g *Game) GetWinnerIdx() int {
	return g.WinnerIdx
}

func (g *Game) WinnerWasSet() bool {
	// This is the only case in which the winner has not yet been set,
	// when both winnerIdx and loserIdx are 0.
	return !(g.WinnerIdx == 0 && g.LoserIdx == 0)
}
