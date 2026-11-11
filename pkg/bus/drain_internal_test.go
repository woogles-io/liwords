package bus

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/nats-io/nats.go"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
)

// fakeSub finishes draining a little after Drain, as a NATS subscription
// does once the messages already sent to it have arrived.
type fakeSub struct {
	mu      sync.Mutex
	drained time.Time
}

func (f *fakeSub) Drain() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.drained = time.Now().Add(50 * time.Millisecond)
	return nil
}

func (f *fakeSub) IsValid() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.drained.IsZero() || time.Now().Before(f.drained)
}

// A bus with no NATS connection or stores: enough for ProcessMessages to
// run while nothing it receives needs them.
func drainTestBus(t *testing.T) (*Bus, []*fakeSub) {
	subs := []*fakeSub{{}, {}}
	b := &Bus{
		config:              config.DefaultConfig(),
		subchans:            map[string]chan *nats.Msg{"user.>": make(chan *nats.Msg, 8)},
		syncReq:             make(chan chan struct{}),
		gameEventChan:       make(chan *entity.EventWrapper, 8),
		tournamentEventChan: make(chan *entity.EventWrapper, 8),
		genericEventChan:    make(chan *entity.EventWrapper, 8),
		gameEventAPIServer:  NewEventApiServer(nil, nil),
	}
	for _, s := range subs {
		b.subscriptions = append(b.subscriptions, s)
	}
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		b.ProcessMessages(ctx)
		close(stopped)
	}()
	t.Cleanup(func() {
		cancel()
		<-stopped
	})
	return b, subs
}

// Drain stops the subscriptions, takes what they already delivered, and
// returns only once the handlers in progress are done.
func TestDrainWaitsForHandlers(t *testing.T) {
	is := is.New(t)
	b, subs := drainTestBus(t)

	var finished atomic.Bool
	b.handlers.Add(1)
	go func() {
		defer b.handlers.Done()
		time.Sleep(200 * time.Millisecond) // a move being handled
		finished.Store(true)
	}()
	b.subchans["user.>"] <- &nats.Msg{Subject: "user.x"} // delivered, not yet taken

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	is.NoErr(b.Drain(ctx))
	is.True(finished.Load())
	is.Equal(len(b.subchans["user.>"]), 0)
	for _, s := range subs {
		is.True(!s.IsValid()) // stopped
	}
}

// A handler that doesn't finish doesn't hold the shutdown past its deadline.
func TestDrainGivesUpAtDeadline(t *testing.T) {
	is := is.New(t)
	b, _ := drainTestBus(t)

	release := make(chan struct{})
	defer close(release)
	b.handlers.Add(1)
	go func() {
		defer b.handlers.Done()
		<-release
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := b.Drain(ctx)
	is.True(errors.Is(err, context.DeadlineExceeded))
	is.True(time.Since(start) < time.Second)
}

// Against a real NATS server, when one is running locally (CI has none): a
// drained channel subscription stops being valid, which Drain waits for, and
// messages published after it go to the other member of the queue group.
func TestDrainRealSubscription(t *testing.T) {
	is := is.New(t)
	nc, err := nats.Connect(nats.DefaultURL, nats.Timeout(time.Second))
	if err != nil {
		t.Skip("no local NATS server:", err)
	}
	defer nc.Close()

	subject := "draintest." + time.Now().Format("150405.000000")
	mine, other := make(chan *nats.Msg, 8), make(chan *nats.Msg, 8)
	sub, err := nc.ChanQueueSubscribe(subject, "q", mine)
	is.NoErr(err)
	otherSub, err := nc.ChanQueueSubscribe(subject, "q", other)
	is.NoErr(err)
	defer otherSub.Unsubscribe()
	is.NoErr(nc.Flush())

	is.NoErr(sub.Drain())
	deadline := time.Now().Add(2 * time.Second)
	for sub.IsValid() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	is.True(!sub.IsValid())

	for range 4 {
		is.NoErr(nc.Publish(subject, []byte("m")))
	}
	is.NoErr(nc.Flush())
	time.Sleep(100 * time.Millisecond)
	is.Equal(len(mine), 0)
	is.Equal(len(other), 4)
}
