package omgsvc

import (
	"context"
	"errors"
	"time"

	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/rs/zerolog/log"
)

type CtxKey string

const nowerCtxKey CtxKey = CtxKey("nower")

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

// GameTimer uses the standard library's `time` package to determine how much time
// has elapsed in a game.
type GameTimer struct{}

// Now returns the current timestamp in milliseconds.
func (g GameTimer) Now() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

func timeRemaining(timers *pb.Timers, nower Nower, onturn, pidx int) int32 {

	if onturn == pidx {
		now := nower.Now()
		return timers.TimeRemaining[pidx] - int32(now-timers.TimeOfLastUpdate)
	}
	return timers.TimeRemaining[pidx]
}

func timeRanOut(timers *pb.Timers, nower Nower, onturn, pidx int) bool {

	if onturn != pidx {
		return false
	}

	now := nower.Now()
	tr := timers.TimeRemaining[pidx] - int32(now-timers.TimeOfLastUpdate)
	return tr < (-timers.MaxOvertime * 60000)
}

func getNower(ctx context.Context) (Nower, error) {
	ctxTimer, ok := ctx.Value(nowerCtxKey).(string)
	if !ok || ctxTimer == "timer" {
		return &GameTimer{}, nil
	}
	if ctxTimer == "fakeNower" {
		log.Info().Msg("using-fake-nower")
		return &FakeNower{}, nil
	}
	return nil, errors.New("ctxTimer not found: " + ctxTimer)
}
