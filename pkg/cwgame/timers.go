package cwgame

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const UntimedTime = 0

// Nower is an interface for determining the current time
type Nower interface {
	// Now returns a timestamp in milliseconds
	Now() int64
}

// GameTimer uses the standard library's `time` package to determine how much time
// has elapsed in a game.
type GameTimer struct{}

// Now returns the current timestamp in milliseconds.
func (g GameTimer) Now() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
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

func getTimeRemaining(gdoc *ipc.GameDocument, nower Nower, onTurn uint32) int64 {
	if gdoc.Timers.Untimed {
		return UntimedTime
	}
	if gdoc.PlayerOnTurn == onTurn {
		return gdoc.Timers.TimeRemaining[onTurn] - (nower.Now() - gdoc.Timers.TimeOfLastUpdate)
	}
	// Otherwise just return whatever the object says
	return int64(gdoc.Timers.TimeRemaining[onTurn])
}

func timeRanOut(gdoc *ipc.GameDocument, nower Nower, pidx uint32) bool {
	if gdoc.Timers.Untimed {
		return false
	}
	if gdoc.PlayerOnTurn != pidx {
		return false
	}
	tr := gdoc.Timers.TimeRemaining[pidx] - (nower.Now() - gdoc.Timers.TimeOfLastUpdate)

	// Check time bank if main time expired
	if tr < 0 && len(gdoc.Timers.TimeBank) > int(pidx) {
		timeBank := gdoc.Timers.TimeBank[pidx]
		// Only ran out if both main time and time bank are exhausted
		totalTime := tr + timeBank
		return totalTime < 0
	}

	return tr < (-int64(gdoc.Timers.MaxOvertime) * 60000)
}

func recordTimeOfMove(gdoc *ipc.GameDocument, nower Nower, pidx uint32, applyIncrement bool) {
	now := nower.Now()
	calculateAndSetTimeRemaining(gdoc, now, pidx, applyIncrement)
}

func calculateAndSetTimeRemaining(gdoc *ipc.GameDocument, now int64, pidx uint32, applyIncrement bool) {
	if gdoc.PlayerOnTurn != pidx {
		// player is not on turn, so their time should not have changed
		return
	}
	if gdoc.Timers.Untimed {
		return
	}

	if applyIncrement && gdoc.Timers.ResetToIncrementAfterTurn {
		gdoc.Timers.TimeRemaining[pidx] = int64(gdoc.Timers.IncrementSeconds * 1000)
		gdoc.Timers.TimeOfLastUpdate = now
		return
	}

	gdoc.Timers.TimeRemaining[pidx] -= (now - gdoc.Timers.TimeOfLastUpdate)
	if applyIncrement {
		gdoc.Timers.TimeRemaining[pidx] += (int64(gdoc.Timers.IncrementSeconds) * 1000)
	}

	// Handle time bank deduction if time went negative
	if gdoc.Timers.TimeRemaining[pidx] < 0 && len(gdoc.Timers.TimeBank) > int(pidx) {
		deficit := -gdoc.Timers.TimeRemaining[pidx]
		if gdoc.Timers.TimeBank[pidx] >= deficit {
			// Time bank can cover the deficit
			gdoc.Timers.TimeBank[pidx] -= deficit
			gdoc.Timers.TimeRemaining[pidx] = 0
		} else {
			// Time bank cannot cover the deficit, time bank exhausted
			gdoc.Timers.TimeRemaining[pidx] = -(deficit - gdoc.Timers.TimeBank[pidx])
			gdoc.Timers.TimeBank[pidx] = 0
		}
	}

	// Cap the overtime, because auto-passing always happens after time has expired.
	maxOvertimeMs := gdoc.Timers.MaxOvertime * 60000
	if gdoc.Timers.TimeRemaining[pidx] < int64(-maxOvertimeMs) {
		log.Debug().Int64("proposed-remaining", gdoc.Timers.TimeRemaining[pidx]).
			Msg("calculate-and-set-remaining-capped")
		gdoc.Timers.TimeRemaining[pidx] = int64(-maxOvertimeMs)
	}
	gdoc.Timers.TimeOfLastUpdate = now
}

func resetTimersAndStart(gdoc *ipc.GameDocument, nower Nower) {
	now := nower.Now()
	gdoc.Timers.TimeOfLastUpdate = now
	gdoc.Timers.TimeStarted = now
	gdoc.TimersStarted = true
}
