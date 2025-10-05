package entity

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	CrosswordGame string = "CrosswordGame"

	LargeTime int = 1000000000
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
	// TimeBank is an array of time bank per player, in milliseconds.
	// Used for correspondence/league games.
	TimeBank []int64 `json:"tb,omitempty"`
	// ResetToIncrementAfterTurn resets the timer to increment_seconds after each turn.
	// Used for correspondence games where each player has a fixed time per turn.
	ResetToIncrementAfterTurn bool `json:"rtiat,omitempty"`
}

func (t *Timers) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (t *Timers) Scan(value interface{}) error {
	if value == nil {
		// Leave timers in zero state for NULL database values
		return nil
	}

	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unexpected type %T for timers", value)
	}

	return json.Unmarshal(b, &t)
}

// Nower is an interface for determining the current time
type Nower interface {
	// Now returns a timestamp in milliseconds
	Now() int64
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

// Sleep simulates a sleep.
func (f *FakeNower) Sleep(t int64) {
	f.fakeMeow += t
}

// Quickdata represents data that we might need quick access to, for the purposes
// of aggregating large numbers of games rapidly. This should get saved in
// its own blob in the store, as opposed to being buried within a game history.
type Quickdata struct {
	OriginalRequestId string           `json:"o"`
	FinalScores       []int32          `json:"s"`
	PlayerInfo        []*pb.PlayerInfo `json:"pi"`
	OriginalRatings   []float64
	NewRatings        []float64
}

func (q *Quickdata) Value() (driver.Value, error) {
	return json.Marshal(q)
}

func (q *Quickdata) Scan(value interface{}) error {
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unexpected type %T for quickdata", value)
	}

	return json.Unmarshal(b, &q)
}

// TournamentData holds the tournament data for a game.
// This is nil if the game is not a tournament game.
type TournamentData struct {
	Id        string
	Division  string `json:"d"`
	Round     int    `json:"r"`
	GameIndex int    `json:"i"`
}

func (t *TournamentData) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (t *TournamentData) Scan(value interface{}) error {
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	case nil:
		*t = TournamentData{}
		return nil
	default:
		return fmt.Errorf("unexpected type %T for tournament-data", value)
	}

	return json.Unmarshal(b, &t)
}

// MetaEventData holds a list of meta events, such as requesting aborts, adjourns, etc.
type MetaEventData struct {
	Events []*pb.GameMetaEvent `json:"events"`
}

func (m *MetaEventData) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *MetaEventData) Scan(value interface{}) error {
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	case nil:
		*m = MetaEventData{}
		return nil
	default:
		return fmt.Errorf("unexpected type %T for tournament-data", value)
	}

	return json.Unmarshal(b, &m)
}

type GameHistory struct {
	macondopb.GameHistory
}

func (h *GameHistory) Value() (driver.Value, error) {
	return proto.Marshal(&h.GameHistory)
}

func (h *GameHistory) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return proto.Unmarshal(b, &h.GameHistory)
}

type GameRequest struct {
	*pb.GameRequest
}

func (g *GameRequest) Value() (driver.Value, error) {
	if g.GameRequest == nil {
		return nil, nil
	}
	log.Debug().Msg("Marshaling GameRequest to jsonb column in Value")
	return protojson.Marshal(g.GameRequest)
}

func (g *GameRequest) Scan(value interface{}) error {
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	case nil:
		*g = GameRequest{GameRequest: &pb.GameRequest{}}
		return nil
	default:
		return fmt.Errorf("unexpected type %T for GameRequest", value)
	}

	g.GameRequest = &pb.GameRequest{}
	log.Debug().Msg("Unmarshaling GameRequest from jsonb column in Scan")
	return protojson.Unmarshal(b, g.GameRequest)
}

// A Game should be saved to the database or store. It wraps a macondo.Game,
// and we should save most of the included fields here, especially the
// macondo.game.History (which can be exported as GCG, etc in the future)
type Game struct {
	sync.RWMutex
	game.Game

	DBID        uint
	Type        pb.GameType
	PlayerDBIDs [2]uint // needed to associate the games to the player IDs in the db.

	GameReq *GameRequest
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
	MetaEvents     *MetaEventData
	CreatedAt      time.Time
}

// GameTimer uses the standard library's `time` package to determine how much time
// has elapsed in a game.
type GameTimer struct{}

// Now returns the current timestamp in milliseconds.
func (g GameTimer) Now() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

// NewGame takes in a Macondo game that was just "started". Note that
// Macondo games when they start do not log any time, they just deal tiles.
// The time of start must be logged later, when both players are in the table
// and ready.
func NewGame(mcg *game.Game, req *pb.GameRequest) *Game {
	ms := 0
	mom := 0
	if req != nil {
		ms = int(req.InitialTimeSeconds * 1000)
		mom = int(req.MaxOvertimeMinutes)
	}
	return &Game{
		Game: *mcg,
		Timers: Timers{
			TimeRemaining: []int{ms, ms},
			MaxOvertime:   mom,
		},
		GameReq:   &GameRequest{req},
		nower:     &GameTimer{},
		Quickdata: &Quickdata{},
	}
}

// SetTimerModule sets the timer for a game to the given Nower.
func (g *Game) SetTimerModule(n Nower) {
	g.nower = n
}

// TimerModule gets the Nower for this game.
func (g *Game) TimerModule() Nower {
	return g.nower
}

// Reset timers to _now_. The game is actually starting.
func (g *Game) ResetTimersAndStart() {
	log.Debug().Msg("reset-timers")
	ts := g.nower.Now()
	g.Timers.TimeOfLastUpdate = ts
	g.Timers.TimeStarted = ts
	g.Started = true

	// Initialize correspondence-specific fields
	if g.IsCorrespondence() {
		// Initialize time bank for both players
		timeBankMs := int64(g.GameReq.MaxOvertimeMinutes) * 60 * 1000
		g.Timers.TimeBank = []int64{timeBankMs, timeBankMs}
		// Enable reset-to-increment behavior for correspondence games
		g.Timers.ResetToIncrementAfterTurn = true
	}
}

func (g *Game) RatingKey() (VariantKey, error) {
	req := g.GameReq
	timefmt, variant, err := VariantFromGameReq(req.GameRequest)
	if err != nil {
		return "", err
	}
	return ToVariantKey(req.Lexicon, variant, timefmt), nil
}

// TimeRemaining calculates the time remaining, but does NOT update it.
func (g *Game) TimeRemaining(idx int) int {
	if g.Type == pb.GameType_ANNOTATED {
		return LargeTime
	}
	if g.Game.PlayerOnTurn() == idx {
		now := g.nower.Now()
		return g.Timers.TimeRemaining[idx] - int(now-g.Timers.TimeOfLastUpdate)
	}
	// If the player is not on turn just return whatever the "cache" says.
	return g.Timers.TimeRemaining[idx]
}

func (g *Game) CachedTimeRemaining(idx int) int {
	if g.Type == pb.GameType_ANNOTATED {
		return LargeTime
	}
	return g.Timers.TimeRemaining[idx]
}

// TimeRanOut calculates if time ran out for the given player. Assumes player is
// on turn, otherwise it always returns false.
func (g *Game) TimeRanOut(idx int) bool {
	if g.Type == pb.GameType_ANNOTATED {
		// time never runs out for annotated games
		return false
	}
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
	if g.Type == pb.GameType_ANNOTATED {
		return
	}
	if g.Game.PlayerOnTurn() == pidx {
		// For correspondence games with reset-to-increment, check time bank and reset
		if accountForIncrement && g.Timers.ResetToIncrementAfterTurn {
			// Check if player went over their allowed turn time
			turnTime := now - g.Timers.TimeOfLastUpdate
			allowedTime := int64(g.GameReq.IncrementSeconds) * 1000

			if turnTime > allowedTime {
				// Player took too long, deduct from time bank
				deficit := turnTime - allowedTime
				if len(g.Timers.TimeBank) > pidx && g.Timers.TimeBank[pidx] > 0 {
					if g.Timers.TimeBank[pidx] >= deficit {
						g.Timers.TimeBank[pidx] -= deficit
					} else {
						// Time bank exhausted
						g.Timers.TimeBank[pidx] = 0
					}
				}
			}

			// Reset time to full increment for next turn
			g.Timers.TimeRemaining[pidx] = int(g.GameReq.IncrementSeconds) * 1000
			g.Timers.TimeOfLastUpdate = now
			return
		}

		// Time has passed since this was calculated.
		g.Timers.TimeRemaining[pidx] -= int(now - g.Timers.TimeOfLastUpdate)
		if accountForIncrement {
			g.Timers.TimeRemaining[pidx] += (int(g.GameReq.IncrementSeconds) * 1000)
		}

		// Handle time bank deduction if time went negative
		if g.Timers.TimeRemaining[pidx] < 0 && len(g.Timers.TimeBank) > pidx {
			deficit := -int64(g.Timers.TimeRemaining[pidx])
			if g.Timers.TimeBank[pidx] >= deficit {
				// Time bank can cover the deficit
				g.Timers.TimeBank[pidx] -= deficit
				g.Timers.TimeRemaining[pidx] = 0
			} else {
				// Time bank cannot cover the deficit, time bank exhausted
				g.Timers.TimeRemaining[pidx] = -int(deficit - g.Timers.TimeBank[pidx])
				g.Timers.TimeBank[pidx] = 0
			}
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

	// Update timer with increment (works for all game types, including time bank logic)
	g.calculateAndSetTimeRemaining(idx, now, true)
}

// LastOutstandingMetaRequest returns the last meta request that has not yet been responded to.
// If a user ID is passed in, it only returns that user's last request, if it exists.
// If no such event exists, it returns nil.
func LastOutstandingMetaRequest(evts []*pb.GameMetaEvent, uid string, now int64) *pb.GameMetaEvent {
	var lastReq *pb.GameMetaEvent
	var lastReqID string
	for _, e := range evts {

		switch e.Type {
		case pb.GameMetaEvent_REQUEST_ABORT,
			pb.GameMetaEvent_REQUEST_ADJUDICATION,
			pb.GameMetaEvent_REQUEST_UNDO,
			pb.GameMetaEvent_REQUEST_ADJOURN:

			if uid != "" && e.PlayerId != uid {
				// not our event
				break
			}

			lastReqID = e.OrigEventId
			lastReq = e

		case pb.GameMetaEvent_ABORT_ACCEPTED,
			pb.GameMetaEvent_ABORT_DENIED,
			pb.GameMetaEvent_ADJUDICATION_ACCEPTED,
			pb.GameMetaEvent_ADJUDICATION_DENIED,
			pb.GameMetaEvent_TIMER_EXPIRED:

			if e.OrigEventId == lastReqID {
				// We found a match, so clear the last request
				lastReq = nil
				lastReqID = ""
			}
		}
	}
	if lastReq != nil && lastReq.Timestamp != nil {
		// convert to milliseconds, as `now` is in milliseconds.
		sinceBeginning := now - (lastReq.Timestamp.AsTime().UnixNano() / int64(time.Millisecond))
		// calculate lastReq's expiry as of _now_ (but we're not saving it back,
		// this is just for FE purposes)
		lastReq = proto.Clone(lastReq).(*pb.GameMetaEvent)
		lastReq.Expiry -= int32(sinceBeginning)
	}
	log.Debug().Interface("lastReq", lastReq).Msg("returning last outstanding req")

	return lastReq
}

func (g *Game) HistoryRefresherEvent() *pb.GameHistoryRefresher {
	now := g.nower.Now()

	g.calculateAndSetTimeRemaining(0, now, false)
	g.calculateAndSetTimeRemaining(1, now, false)
	var outstandingEvent *pb.GameMetaEvent
	if g.Playing() != macondopb.PlayState_GAME_OVER {
		outstandingEvent = LastOutstandingMetaRequest(g.MetaEvents.Events, "", now)
	}

	return &pb.GameHistoryRefresher{
		History:            g.History(),
		TimePlayer1:        int32(g.TimeRemaining(0)),
		TimePlayer2:        int32(g.TimeRemaining(1)),
		MaxOvertimeMinutes: g.GameReq.MaxOvertimeMinutes,
		OutstandingEvent:   outstandingEvent,
	}
}

func (g *Game) ChallengeRule() macondopb.ChallengeRule {
	return g.GameReq.ChallengeRule
}

func (g *Game) RatingMode() pb.RatingMode {
	return g.GameReq.RatingMode
}

// RegisterChangeHook registers a channel with the game. Events will
// be sent down this channel.
func (g *Game) RegisterChangeHook(eventChan chan<- *EventWrapper) error {
	log.Debug().Msg("register-change-hook")
	g.ChangeHook = eventChan
	return nil
}

// SendChange sends an event via the registered hook.
func (g *Game) SendChange(e *EventWrapper) {
	log.Debug().Interface("evt", e.Event).
		Interface("aud", e.Audience()).
		Int("chan-length", len(g.ChangeHook)).Msg("send-change")
	if g.ChangeHook == nil {
		// This should never happen in actual operation; consider making it a Fatal.
		// XXX: This happens all the time in production, because we are calling
		// SendChange for a NewActiveGameEntry, and the game has not started by then.
		// This channel only gets initialized when a game actually starts.
		// (how is it working?)
		log.Error().Msg("change hook is closed!")
		return
	}
	g.ChangeHook <- e
	log.Debug().Msg("change sent")
}

func (g *Game) NewActiveGameEntry(gameStillActive bool) *EventWrapper {
	ttl := int64(0) // seconds
	if gameStillActive {
		// Ideally we would set this based on time remaining (and round it up).
		// But since we don't want to refresh every turn, we can just set a very long expiry here.
		// A 60min + 60sec increment game can take 12 hours.
		// - Both players start with 60 mins each.
		// - Game lasts 600 turns (5 passes, play one tile, repeat 100 times).
		ttl = 12 * 60 * 60
	}
	players := g.History().Players
	activeGamePlayers := make([]*pb.ActiveGamePlayer, 0, len(players))
	for _, player := range players {
		activeGamePlayers = append(activeGamePlayers, &pb.ActiveGamePlayer{
			Username: player.Nickname,
			UserId:   player.UserId,
		})
	}
	return WrapEvent(&pb.ActiveGameEntry{
		Id:     g.GameID(),
		Player: activeGamePlayers,
		Ttl:    ttl,
	}, pb.MessageType_ACTIVE_GAME_ENTRY)
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

// IsCorrespondence returns true if this is a correspondence game
func (g *Game) IsCorrespondence() bool {
	if g.GameReq == nil || g.GameReq.GameRequest == nil {
		return false
	}
	return g.GameReq.GameMode == pb.GameMode_CORRESPONDENCE
}
