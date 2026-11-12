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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
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
		drainChecks:         make(chan drainCheck),
		publish:             func(string, []byte) error { return nil },
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
	done := b.trackHandler()
	go func() {
		defer done()
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
	done := b.trackHandler()
	go func() {
		defer done()
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

// recordSpans sends the spans started during the test to a recorder.
func recordSpans(t *testing.T) *tracetest.SpanRecorder {
	rec := tracetest.NewSpanRecorder()
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	t.Cleanup(func() { otel.SetTracerProvider(prev) })
	return rec
}

func endedSpan(is *is.I, rec *tracetest.SpanRecorder, name string) sdktrace.ReadOnlySpan {
	for _, s := range rec.Ended() {
		if s.Name() == name {
			return s
		}
	}
	is.Fail() // no such span
	return nil
}

func attr(s sdktrace.ReadOnlySpan, key string) (attribute.Value, bool) {
	for _, kv := range s.Attributes() {
		if string(kv.Key) == key {
			return kv.Value, true
		}
	}
	return attribute.Value{}, false
}

// Drain's span shows each step, and what was in flight when it started.
func TestDrainSpan(t *testing.T) {
	is := is.New(t)
	rec := recordSpans(t)
	b, _ := drainTestBus(t)

	done := b.trackHandler()
	go func() {
		defer done()
		time.Sleep(100 * time.Millisecond)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	is.NoErr(b.Drain(ctx))

	span := endedSpan(is, rec, "bus.drain")
	inFlight, _ := attr(span, "bus.handlers_in_flight")
	is.Equal(inFlight.AsInt64(), int64(1))
	events := []string{}
	for _, e := range span.Events() {
		events = append(events, e.Name)
	}
	is.Equal(events, []string{"buffers-taken", "handlers-and-events-done"})
	is.Equal(span.Status().Code, codes.Unset)
}

// A drain that runs out of time says where it was stuck.
func TestDrainSpanAtDeadline(t *testing.T) {
	is := is.New(t)
	rec := recordSpans(t)
	b, _ := drainTestBus(t)

	release := make(chan struct{})
	defer close(release)
	done := b.trackHandler()
	go func() {
		defer done()
		<-release
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	is.True(errors.Is(b.Drain(ctx), context.DeadlineExceeded))

	span := endedSpan(is, rec, "bus.drain")
	is.Equal(span.Status().Code, codes.Error)
	stuck, _ := attr(span, "bus.drain.stuck_at")
	is.Equal(stuck.AsString(), "finish-handlers-and-events")
	left, _ := attr(span, "bus.handlers_left")
	is.Equal(left.AsInt64(), int64(1))
}

func chatEvent(audience string) *entity.EventWrapper {
	e := entity.WrapEvent(&pb.ChatMessage{Message: "m"}, pb.MessageType_CHAT_MESSAGE)
	e.SetAudience(audience)
	return e
}

// slowPublisher stands in for NATS: each publish takes a while, and is
// recorded once it has completed.
type slowPublisher struct {
	mu        sync.Mutex
	delay     time.Duration
	published []string
}

func (p *slowPublisher) publish(subject string, data []byte) error {
	time.Sleep(p.delay)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.published = append(p.published, subject)
	return nil
}

func (p *slowPublisher) subjects() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.published...)
}

// Drain returns only after the loop has finished publishing the last event,
// not as soon as the event channel is empty: the loop takes the event out of
// the channel first, then spends a while publishing it.
func TestDrainWaitsForEventBeingPublished(t *testing.T) {
	is := is.New(t)
	b, _ := drainTestBus(t)
	pub := &slowPublisher{delay: 300 * time.Millisecond}
	b.publish = pub.publish

	b.genericEventChan <- chatEvent("chat.lobby")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	is.NoErr(b.Drain(ctx))
	is.Equal(pub.subjects(), []string{"chat.lobby"})
}

// The events a handler sends on its way out are published before Drain
// returns, even when the handler finishes after Drain started.
func TestDrainPublishesHandlersEvents(t *testing.T) {
	is := is.New(t)
	b, _ := drainTestBus(t)
	pub := &slowPublisher{delay: 50 * time.Millisecond}
	b.publish = pub.publish

	done := b.trackHandler()
	go func() {
		defer done()
		time.Sleep(100 * time.Millisecond) // a move being handled
		for i := range 3 {
			b.genericEventChan <- chatEvent("chat." + string(rune('a'+i)))
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	is.NoErr(b.Drain(ctx))
	is.Equal(pub.subjects(), []string{"chat.a", "chat.b", "chat.c"})
}
