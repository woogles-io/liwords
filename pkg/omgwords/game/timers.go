package game

import (
	"time"

	"github.com/woogles-io/liwords/pkg/omgwords/game/gamestate"
)

const UntimedTime uint64 = 0

// Nower is an interface for determining the current time
type Nower interface {
	// Now returns a timestamp in milliseconds
	Now() uint64
}

// GameTimer uses the standard library's `time` package to determine how much time
// has elapsed in a game.
type GameTimer struct{}

// Now returns the current timestamp in milliseconds.
func (g GameTimer) Now() uint64 {
	return uint64(time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)))
}

// FakeNower uses a fake timer. It is used for tests so we don't actually sleep.
type FakeNower struct {
	fakeMeow uint64
}

func NewFakeNower(f uint64) *FakeNower {
	return &FakeNower{f}
}

// Now returns now's value
func (f FakeNower) Now() uint64 {
	return f.fakeMeow
}

// Sleep simulates a sleep.
func (f *FakeNower) Sleep(t uint64) {
	f.fakeMeow += t
}

func getTimeRemaining(timers *gamestate.Timers, nower Nower, pidx, onTurn uint8) int64 {

	if pidx == onTurn {
		return timers.TimeRemainingMs(int(pidx)) - int64(nower.Now()-timers.TimeOfLastUpdateMs())
	}
	// Otherwise just return whatever the object says
	return timers.TimeRemainingMs(int(onTurn))
}

func timeRanOut(timers *gamestate.Timers, nower Nower, onTurn uint8) bool {
	tr := timers.TimeRemainingMs(int(onTurn)) - int64(nower.Now()-timers.TimeOfLastUpdateMs())
	return tr < (-int64(timers.MaxOvertimeMinutes()) * 60000)
}

// recordTimeOfMove records the time of a move. It should only be called for players
// currently on turn.
func recordTimeOfMove(timers *gamestate.Timers, nower Nower, pidx uint32, applyIncrement bool) {
	now := nower.Now()
	calculateAndSetTimeRemaining(timers, now, pidx, applyIncrement)
}

func calculateAndSetTimeRemaining(timers *gamestate.Timers, now uint64, pidx uint32, applyIncrement bool) {
	if applyIncrement && timers.ResetToIncrementAfterTurn() {
		timers.MutateTimeRemainingMs(int(pidx), int64(timers.IncrementSeconds()*1000))
		timers.MutateTimeOfLastUpdateMs(now)
		return
	}

	timers.MutateTimeRemainingMs(int(pidx),
		int64(timers.TimeRemainingMs(int(pidx))-int64(now-timers.TimeOfLastUpdateMs())))

	if applyIncrement {
		timers.MutateTimeRemainingMs(int(pidx),
			int64(timers.TimeRemainingMs(int(pidx))+int64(timers.IncrementSeconds()*1000)))
	}
	// Cap the overtime, because auto-passing always happens after time has expired.
	maxOvertimeMs := timers.MaxOvertimeMinutes() * 60000

	if timers.TimeRemainingMs(int(pidx)) < int64(-maxOvertimeMs) {
		timers.MutateTimeRemainingMs(int(pidx), int64(-maxOvertimeMs))
	}

	timers.MutateTimeOfLastUpdateMs(now)
}

func resetTimersAndStart(timers *gamestate.Timers, nower Nower) {
	now := nower.Now()

	timers.MutateTimeOfLastUpdateMs(now)
	timers.MutateTimeStartedMs(now)
}
