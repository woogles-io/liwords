package entity

import "testing"

func TestTimersRanOut(t *testing.T) {
	const now = int64(10_000_000)
	realTime := func(remaining, sinceUpdate, maxOvertimeMinutes int) Timers {
		return Timers{
			TimeOfLastUpdate: now - int64(sinceUpdate),
			TimeRemaining:    []int{remaining, 60_000},
			MaxOvertime:      maxOvertimeMinutes,
		}
	}
	correspondence := func(sinceUpdate int64, bank int64) Timers {
		return Timers{
			TimeOfLastUpdate:          now - sinceUpdate,
			TimeRemaining:             []int{0, 0},
			TimeBank:                  []int64{bank, 0},
			ResetToIncrementAfterTurn: true,
		}
	}

	cases := []struct {
		name      string
		timers    Timers
		idx       int
		increment int32
		want      bool
	}{
		{"real-time, time left", realTime(60_000, 30_000, 0), 0, 0, false},
		{"real-time, no overtime, just out", realTime(60_000, 60_001, 0), 0, 0, true},
		{"real-time, in overtime, within the limit", realTime(60_000, 120_000, 1), 0, 0, false},
		{"real-time, past the overtime limit", realTime(60_000, 120_001, 1), 0, 0, true},
		{"real-time, player index out of range", realTime(60_000, 120_001, 0), 2, 0, false},
		{"correspondence, within the turn", correspondence(30_000, 0), 0, 60, false},
		{"correspondence, over, no bank", correspondence(120_000, 0), 0, 60, true},
		{"correspondence, over, bank covers it", correspondence(120_000, 60_000), 0, 60, false},
		{"correspondence, over, bank exhausted", correspondence(240_000, 120_000), 0, 60, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.timers.RanOut(c.idx, c.increment, now); got != c.want {
				t.Errorf("RanOut = %v, want %v", got, c.want)
			}
		})
	}
}
