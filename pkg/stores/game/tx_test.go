package game

// The lock has to actually serialize across connections, because the whole
// point is that a process-local mutex did not.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

func lockTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := common.OpenTestingDB(pkg)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// Two holders of the same game id must not overlap. This is the property that
// Cache.LockGame provided within one process and could not provide between
// app servers.
func TestWithGameLockSerializesSameGame(t *testing.T) {
	is := is.New(t)
	pool := lockTestPool(t)
	ctx := context.Background()

	var mu sync.Mutex
	inside := 0
	maxInside := 0

	run := func() error {
		return WithGameLock(ctx, pool, "lock-test-game", func(ctx context.Context, q *models.Queries) error {
			mu.Lock()
			inside++
			if inside > maxInside {
				maxInside = inside
			}
			mu.Unlock()

			// Hold it long enough that an unserialized second caller would
			// overlap.
			time.Sleep(60 * time.Millisecond)

			mu.Lock()
			inside--
			mu.Unlock()
			return nil
		})
	}

	var wg sync.WaitGroup
	errs := make([]error, 4)
	for i := range errs {
		wg.Add(1)
		go func(i int) { defer wg.Done(); errs[i] = run() }(i)
	}
	wg.Wait()

	for _, err := range errs {
		is.NoErr(err)
	}
	is.Equal(maxInside, 1) // never two at once
}

// Different games must not block each other, or every move on the server
// queues behind every other move.
func TestWithGameLockAllowsDifferentGames(t *testing.T) {
	is := is.New(t)
	pool := lockTestPool(t)
	ctx := context.Background()

	start := time.Now()
	var wg sync.WaitGroup
	errs := make([]error, 4)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = WithGameLock(ctx, pool, "lock-test-"+string(rune('a'+i)),
				func(context.Context, *models.Queries) error {
					time.Sleep(60 * time.Millisecond)
					return nil
				})
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		is.NoErr(err)
	}
	// Serialized would be ~240ms; concurrent should be well under.
	is.True(time.Since(start) < 180*time.Millisecond)
}

// A failing body must roll back and, critically, release the lock -- otherwise
// one bad move wedges a game until the connection is recycled.
func TestWithGameLockReleasesOnError(t *testing.T) {
	is := is.New(t)
	pool := lockTestPool(t)
	ctx := context.Background()

	boom := errors.New("body failed")
	err := WithGameLock(ctx, pool, "lock-test-err", func(context.Context, *models.Queries) error {
		return boom
	})
	is.Equal(err, boom)

	// The lock is free again immediately.
	done := make(chan error, 1)
	go func() {
		done <- WithGameLock(ctx, pool, "lock-test-err", func(context.Context, *models.Queries) error {
			return nil
		})
	}()
	select {
	case err := <-done:
		is.NoErr(err)
	case <-time.After(3 * time.Second):
		t.Fatal("lock was not released after the body failed")
	}
}

// And the same for a panic, since a handler panicking must not wedge a game
// either.
func TestWithGameLockReleasesOnPanic(t *testing.T) {
	is := is.New(t)
	pool := lockTestPool(t)
	ctx := context.Background()

	func() {
		defer func() { _ = recover() }()
		_ = WithGameLock(ctx, pool, "lock-test-panic", func(context.Context, *models.Queries) error {
			panic("boom")
		})
	}()

	done := make(chan error, 1)
	go func() {
		done <- WithGameLock(ctx, pool, "lock-test-panic", func(context.Context, *models.Queries) error {
			return nil
		})
	}()
	select {
	case err := <-done:
		is.NoErr(err)
	case <-time.After(3 * time.Second):
		t.Fatal("lock was not released after a panic")
	}
}

func TestWithGameLockRejectsBadInput(t *testing.T) {
	is := is.New(t)
	noop := func(context.Context, *models.Queries) error { return nil }
	is.True(WithGameLock(context.Background(), nil, "g", noop) != nil)
	is.True(WithGameLock(context.Background(), lockTestPool(t), "", noop) != nil)
}
