package bus

// Shutting down without dropping work in progress.
//
// Moves, and everything else the socket server sends, arrive over NATS, and
// ProcessMessages handles each in its own goroutine. Before this, shutdown
// cancelled the bus while those goroutines ran: a move being handled was cut
// off after its save, so its players never heard of it, and at the end of a
// game the ratings, standings and lobby updates after it never ran. The bus
// also stayed subscribed until then, so new moves kept arriving at a server
// that was about to drop them.
//
// A message's way through the bus:
//
//	NATS → subscription buffer → loop takes it → handler goroutine
//	     → event channel → loop takes it → published to NATS
//
// Drain stops new work and waits until nothing is left anywhere on it:
//
//  1. Stop the subscriptions. Messages NATS already sent us still arrive; new
//     ones go to the other servers in the queue groups, or, with one server,
//     nowhere (as they did while the server was down anyway).
//  2. Wait until the loop has taken everything buffered.
//  3. Wait until every handler is done and the loop has published every
//     event. Handlers still publish: the NATS connection stays open, so it
//     must not be drained as a whole (that closes it).
//
// Twice the loop holds a message that is in no buffer and not yet counted:
// just after taking it from a subscription, and while publishing an event. So
// the conditions of steps 2 and 3 are checked by the loop itself, between
// iterations, where it holds nothing: Drain sends each as a drainCheck and
// waits for the loop's answer (waitInLoop). Checked from outside, they
// could hold while the loop was still in the middle of one.
//
// ProcessMessages must keep running until Drain returns.
//
// Jobs (pkg/jobs) such as adjudication also send events through the bus, so
// the job runner must be drained first.
//
// Drain records a "bus.drain" span with an event as each step finishes. The
// messages it waited for are the spans from the same host (host.name) inside
// its time window.

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// subscription is what Drain needs of a NATS subscription.
type subscription interface {
	// Drain stops new messages and delivers the ones already received.
	Drain() error
	// IsValid is false once the subscription has finished draining.
	IsValid() bool
}

// drainCheckEvery is how often an idle loop checks Drain's conditions.
const drainCheckEvery = 20 * time.Millisecond

// Drain stops taking new messages and waits until everything already taken
// has been handled and its events published, or until ctx ends.
func (b *Bus) Drain(ctx context.Context) (err error) {
	ctx, span := otel.Tracer("bus-drain").Start(ctx, "bus.drain")
	defer span.End()
	span.SetAttributes(
		attribute.Int("bus.buffered", b.buffered()),
		attribute.Int64("bus.handlers_in_flight", b.inFlight.Load()),
		attribute.Int("bus.events_queued", b.eventsQueued()))
	step := "stop-subscriptions"
	defer func() {
		if err != nil {
			// Where it was stuck, and with how much left.
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.SetAttributes(
				attribute.String("bus.drain.stuck_at", step),
				attribute.Int("bus.buffered_left", b.buffered()),
				attribute.Int64("bus.handlers_left", b.inFlight.Load()),
				attribute.Int("bus.events_left", b.eventsQueued()))
		}
	}()

	for _, sub := range b.subscriptions {
		if err := sub.Drain(); err != nil {
			return err
		}
	}

	step = "take-buffered"
	if err := b.waitInLoop(ctx, func() bool {
		for _, sub := range b.subscriptions {
			if sub.IsValid() {
				return false
			}
		}
		return b.buffered() == 0
	}); err != nil {
		return err
	}
	// Nothing new will be taken, and everything taken is counted.
	span.AddEvent("buffers-taken", trace.WithAttributes(
		attribute.Int64("bus.handlers_in_flight", b.inFlight.Load()),
		attribute.Int("bus.events_queued", b.eventsQueued())))

	step = "finish-handlers-and-events"
	if err := b.waitInLoop(ctx, func() bool {
		return b.inFlight.Load() == 0 && b.eventsQueued() == 0
	}); err != nil {
		return err
	}
	span.AddEvent("handlers-and-events-done")
	return nil
}

// waitInLoop returns once ProcessMessages, checking between its iterations,
// finds cond true, or when ctx ends.
func (b *Bus) waitInLoop(ctx context.Context, cond func() bool) error {
	c := drainCheck{cond: cond, done: make(chan struct{})}
	select {
	case b.drainChecks <- c:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-c.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// drainCheck asks ProcessMessages to close done once cond holds. Drain sends
// them on Bus.drainChecks (see waitInLoop).
type drainCheck struct {
	cond func() bool
	done chan struct{}
}

// pendingDrainChecks are the drain checks ProcessMessages hasn't answered
// yet. Only the loop uses it.
type pendingDrainChecks struct {
	pending []drainCheck
	ticker  *time.Ticker
}

func (p *pendingDrainChecks) add(c drainCheck) {
	p.pending = append(p.pending, c)
	if p.ticker == nil {
		p.ticker = time.NewTicker(drainCheckEvery)
	}
}

// tick wakes an idle loop while something is pending; otherwise it is nil,
// which never fires.
func (p *pendingDrainChecks) tick() <-chan time.Time {
	if p.ticker == nil {
		return nil
	}
	return p.ticker.C
}

// check answers the checks whose condition holds.
func (p *pendingDrainChecks) check() {
	if len(p.pending) == 0 {
		return
	}
	left := p.pending[:0]
	for _, c := range p.pending {
		if c.cond() {
			close(c.done)
		} else {
			left = append(left, c)
		}
	}
	p.pending = left
	if len(p.pending) == 0 {
		p.stop()
	}
}

func (p *pendingDrainChecks) stop() {
	if p.ticker != nil {
		p.ticker.Stop()
		p.ticker = nil
	}
}

// trackHandler counts a goroutine whose work Drain must wait for; it
// returns the function the goroutine defers.
func (b *Bus) trackHandler() func() {
	b.inFlight.Add(1)
	return func() { b.inFlight.Add(-1) }
}

// buffered is how many received messages ProcessMessages hasn't taken yet.
func (b *Bus) buffered() int {
	n := 0
	for _, ch := range b.subchans {
		n += len(ch)
	}
	return n
}

func (b *Bus) eventsQueued() int {
	return len(b.gameEventChan) + len(b.tournamentEventChan) + len(b.genericEventChan)
}
