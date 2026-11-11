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
// Drain stops new work and waits for the work already taken:
//
//  1. Stop the subscriptions. Messages NATS already sent us still arrive; new
//     ones go to the other servers in the queue groups, or, with one server,
//     nowhere (as they did while the server was down anyway).
//  2. Let ProcessMessages take every message still buffered, and stop
//     starting adjudication and seek expiry.
//  3. Wait for the handlers, which still publish their results: the NATS
//     connection stays open, so it must not be drained as a whole (that
//     closes it).
//  4. Let ProcessMessages publish the events the handlers left behind.
//
// ProcessMessages must keep running until Drain returns.

import (
	"context"
	"time"
)

// subscription is what Drain needs of a NATS subscription.
type subscription interface {
	// Drain stops new messages and delivers the ones already received.
	Drain() error
	// IsValid is false once the subscription has finished draining.
	IsValid() bool
}

const drainPoll = 20 * time.Millisecond

// Drain stops taking new messages and waits until everything already taken
// has been handled and its events published, or until ctx ends.
func (b *Bus) Drain(ctx context.Context) error {
	b.draining.Store(true)
	for _, sub := range b.subscriptions {
		if err := sub.Drain(); err != nil {
			return err
		}
	}
	if err := b.waitFor(ctx, func() bool {
		for _, sub := range b.subscriptions {
			if sub.IsValid() {
				return false
			}
		}
		for _, ch := range b.subchans {
			if len(ch) > 0 {
				return false
			}
		}
		return true
	}); err != nil {
		return err
	}
	// A message ProcessMessages took just before the buffers emptied may not
	// be tracked yet; once the loop answers, it is. From here on the loop
	// starts no handlers, so waiting on them is safe.
	if err := b.sync(ctx); err != nil {
		return err
	}
	handled := make(chan struct{})
	go func() {
		b.handlers.Wait()
		close(handled)
	}()
	select {
	case <-handled:
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := b.waitFor(ctx, func() bool {
		return len(b.gameEventChan) == 0 && len(b.tournamentEventChan) == 0 &&
			len(b.genericEventChan) == 0
	}); err != nil {
		return err
	}
	return b.sync(ctx)
}

// sync returns once ProcessMessages has finished whatever it was doing.
func (b *Bus) sync(ctx context.Context) error {
	done := make(chan struct{})
	select {
	case b.syncReq <- done:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *Bus) waitFor(ctx context.Context, cond func() bool) error {
	t := time.NewTicker(drainPoll)
	defer t.Stop()
	for !cond() {
		select {
		case <-t.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
