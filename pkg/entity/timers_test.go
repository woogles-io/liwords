package entity

import (
	"encoding/json"
	"testing"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

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
			c.timers.IncrementSeconds = c.increment
			if got := c.timers.RanOut(c.idx, now); got != c.want {
				t.Errorf("RanOut = %v, want %v", got, c.want)
			}
		})
	}
}

// Timers written before the increment was stored have no "inc" key. Loading
// them fills it from the request; loading new ones leaves it alone; and it is
// always written, even when zero, so SQL can tell the two kinds apart.
func TestTimersIncrement(t *testing.T) {
	var old Timers
	if err := json.Unmarshal([]byte(`{"lu":1,"mo":0,"tb":[5,5],"tr":[9,9],"ts":1,"rtiat":true}`), &old); err != nil {
		t.Fatal(err)
	}
	if old.IncrementSeconds != 0 {
		t.Fatalf("old timers decoded with an increment: %d", old.IncrementSeconds)
	}
	old.FillIncrement(&pb.GameRequest{IncrementSeconds: 432000})
	if old.IncrementSeconds != 432000 {
		t.Errorf("FillIncrement = %d, want 432000", old.IncrementSeconds)
	}

	stored := Timers{IncrementSeconds: 36000}
	stored.FillIncrement(&pb.GameRequest{IncrementSeconds: 1})
	if stored.IncrementSeconds != 36000 {
		t.Errorf("FillIncrement overwrote a stored increment: %d", stored.IncrementSeconds)
	}

	var none Timers
	none.FillIncrement(nil)
	b, err := json.Marshal(&none)
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]any
	if err := json.Unmarshal(b, &keys); err != nil {
		t.Fatal(err)
	}
	if v, ok := keys["inc"]; !ok || v != float64(0) {
		t.Errorf(`a zero increment must be written as "inc":0, got %s`, b)
	}
}
