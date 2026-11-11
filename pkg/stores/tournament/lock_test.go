package tournament

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// Changes from two stores (two servers: each has its own in-process lock,
// so only the advisory lock keeps them apart) all land.
func TestLockForChangeSerializesAcrossServers(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	pool, a := recreateDB(t)
	b, err := NewDBStore(pool, nil)
	is.NoErr(err)
	is.NoErr(a.Create(ctx, testTournament(t, 0)))

	const perServer = 10
	var wg sync.WaitGroup
	errs := make(chan error, 2*perServer)
	for i := range 2 * perServer {
		store := a
		if i%2 == 1 {
			store = b
		}
		wg.Go(func() {
			tm, lk, err := store.LockForChange(ctx, "tid1")
			if err != nil {
				errs <- err
				return
			}
			defer lk.Release()
			tm.Description += fmt.Sprintf("[%d]", i)
			errs <- lk.Save(ctx)
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		is.NoErr(err)
	}

	got, err := a.Get(ctx, "tid1")
	is.NoErr(err)
	for i := range 2 * perServer {
		is.True(strings.Contains(got.Description, fmt.Sprintf("[%d]", i))) // no change lost
	}
}

// A change that isn't saved is discarded, and releases the lock on both
// levels: the next change, on this server or another, goes ahead.
func TestReleaseWithoutSave(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	pool, a := recreateDB(t)
	b, err := NewDBStore(pool, nil)
	is.NoErr(err)
	is.NoErr(a.Create(ctx, testTournament(t, 0)))

	tm, lk, err := a.LockForChange(ctx, "tid1")
	is.NoErr(err)
	tm.Description = "not kept"
	lk.Release()
	is.Equal(lk.Save(ctx), errReleased)

	got, err := a.Get(ctx, "tid1")
	is.NoErr(err)
	is.Equal(got.Description, "a description")

	short, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	for _, store := range []*DBStore{a, b} {
		_, lk, err := store.LockForChange(short, "tid1")
		is.NoErr(err)
		lk.Release()
	}
	is.Equal(len(a.local.entries), 0)
}

// Changes waiting on this server hold no connection while they wait, so a
// queue of them (the end of a round) can't use up the pool. A waiter whose
// context ends gives up without taking the lock.
func TestWaitersHoldNoConnection(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	pool, store := recreateDB(t)
	is.NoErr(store.Create(ctx, testTournament(t, 0)))

	_, held, err := store.LockForChange(ctx, "tid1")
	is.NoErr(err)
	is.Equal(pool.Stat().AcquiredConns(), int32(1))

	const waiters = 5
	waitCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	errs := make(chan error, waiters)
	for range waiters {
		wg.Go(func() {
			_, lk, err := store.LockForChange(waitCtx, "tid1")
			if err == nil {
				lk.Release()
			}
			errs <- err
		})
	}
	time.Sleep(100 * time.Millisecond)
	is.Equal(pool.Stat().AcquiredConns(), int32(1)) // only the holder's

	cancel()
	wg.Wait()
	close(errs)
	for err := range errs {
		is.True(errors.Is(err, context.Canceled))
	}
	held.Release()
	is.Equal(pool.Stat().AcquiredConns(), int32(0))
	is.Equal(len(store.local.entries), 0)
}

// A change waiting on another server's lock gives up when its context ends,
// and releases this server's lock as it does.
func TestWaitOnOtherServerHonorsContext(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	pool, a := recreateDB(t)
	b, err := NewDBStore(pool, nil)
	is.NoErr(err)
	is.NoErr(a.Create(ctx, testTournament(t, 0)))

	_, held, err := a.LockForChange(ctx, "tid1")
	is.NoErr(err)

	short, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	_, _, err = b.LockForChange(short, "tid1")
	is.True(err != nil)
	is.Equal(len(b.local.entries), 0)

	held.Release()
	_, lk, err := b.LockForChange(ctx, "tid1")
	is.NoErr(err)
	lk.Release()
}

// The LockForChange span splits into the two lock waits and the load, and
// LockForChange and Save both record the size of the divisions.
func TestLockSpans(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	rec := tracetest.NewSpanRecorder()
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	defer otel.SetTracerProvider(prev)

	_, store := recreateDB(t)
	is.NoErr(store.Create(ctx, testTournament(t, 0)))
	_, lk, err := store.LockForChange(ctx, "tid1")
	is.NoErr(err)
	is.NoErr(lk.Save(ctx))

	spans := map[string]sdktrace.ReadOnlySpan{}
	for _, s := range rec.Ended() {
		spans[s.Name()] = s
	}
	get, save := spans["tournament.LockForChange"], spans["tournament.Save"]
	is.True(get != nil)
	is.True(save != nil)
	events := []string{}
	for _, e := range get.Events() {
		events = append(events, e.Name)
	}
	is.Equal(events, []string{"local-lock", "advisory-lock"})
	for _, s := range []sdktrace.ReadOnlySpan{get, save} {
		size := int64(0)
		for _, a := range s.Attributes() {
			if a.Key == "tournament.divisions_bytes" {
				size = a.Value.AsInt64()
			}
		}
		is.True(size > 0) // divisions size recorded
	}
}
