package game

// The lock has to serialize across connections, because the whole point is that
// a process-local mutex did not.
//
// Each test builds its own DBStore over its own pool, which is as close as a
// test gets to two app servers: separate pools mean separate connections, and a
// separate connection is a separate Postgres session, which is the level
// advisory locks work at.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/common"
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

func lockTestStore(t *testing.T) *DBStore {
	t.Helper()
	return &DBStore{dbPool: lockTestPool(t)}
}

// Two holders of the same game id must never overlap.
func TestLockGameSerializesSameGame(t *testing.T) {
	is := is.New(t)
	store := lockTestStore(t)
	ctx := context.Background()

	var mu sync.Mutex
	inside, maxInside := 0, 0

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lock, err := store.LockGame(ctx, "serialize-me")
			if err != nil {
				t.Error(err)
				return
			}
			defer lock.Unlock(ctx)

			mu.Lock()
			inside++
			if inside > maxInside {
				maxInside = inside
			}
			mu.Unlock()

			time.Sleep(40 * time.Millisecond)

			mu.Lock()
			inside--
			mu.Unlock()
		}()
	}
	wg.Wait()
	is.Equal(maxInside, 1)
}

// Two app servers, modelled as two pools. This is the case the mutex map could
// never cover.
func TestLockGameSerializesAcrossPools(t *testing.T) {
	is := is.New(t)
	a, b := lockTestStore(t), lockTestStore(t)
	ctx := context.Background()

	lock, err := a.LockGame(ctx, "two-servers")
	is.NoErr(err)

	// b must not get it while a holds it.
	short, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
	defer cancel()
	_, err = b.LockGame(short, "two-servers")
	is.True(err != nil)

	// ...and must get it once a lets go.
	lock.Unlock(ctx)
	lock2, err := b.LockGame(ctx, "two-servers")
	is.NoErr(err)
	lock2.Unlock(ctx)
}

// Different games must not wait on each other.
func TestLockGameAllowsDifferentGames(t *testing.T) {
	is := is.New(t)
	store := lockTestStore(t)
	ctx := context.Background()

	l1, err := store.LockGame(ctx, "game-one")
	is.NoErr(err)
	defer l1.Unlock(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		l2, err := store.LockGame(ctx, "game-two")
		if err != nil {
			t.Error(err)
			return
		}
		l2.Unlock(ctx)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a lock on one game blocked a lock on another")
	}
}

// A lock must be released even when the request it belonged to has already been
// cancelled, which is the ordinary shape of a failed request: the deferred
// Unlock runs with a dead context.
func TestUnlockWorksWithACancelledContext(t *testing.T) {
	is := is.New(t)
	store := lockTestStore(t)

	ctx, cancel := context.WithCancel(context.Background())
	lock, err := store.LockGame(ctx, "cancelled-request")
	is.NoErr(err)
	cancel()
	lock.Unlock(ctx)

	// Freely available again.
	again, err := store.LockGame(context.Background(), "cancelled-request")
	is.NoErr(err)
	again.Unlock(context.Background())
}

// Unlocking twice must not release someone else's lock.
func TestUnlockIsIdempotent(t *testing.T) {
	is := is.New(t)
	store := lockTestStore(t)
	ctx := context.Background()

	lock, err := store.LockGame(ctx, "double-unlock")
	is.NoErr(err)
	lock.Unlock(ctx)

	other, err := store.LockGame(ctx, "double-unlock")
	is.NoErr(err)
	defer other.Unlock(ctx)

	lock.Unlock(ctx) // second call: must be a no-op

	// other still holds it, so a third party must still be excluded.
	short, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	_, err = store.LockGame(short, "double-unlock")
	is.True(err != nil)
}

// Waiting is bounded. A stuck holder must produce an error, not a hung request.
func TestLockGameGivesUp(t *testing.T) {
	is := is.New(t)
	store := lockTestStore(t)
	ctx := context.Background()

	held, err := store.LockGame(ctx, "stuck")
	is.NoErr(err)
	defer held.Unlock(ctx)

	start := time.Now()
	short, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	_, err = store.LockGame(short, "stuck")
	is.True(err != nil)
	is.True(errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrGameLockBusy))
	is.True(time.Since(start) < 2*time.Second)
}

// Releasing must give the connection back, or the pool drains and the process
// wedges after a few hundred moves.
func TestLockGameReturnsItsConnection(t *testing.T) {
	is := is.New(t)
	pool := lockTestPool(t)
	store := &DBStore{dbPool: pool}
	ctx := context.Background()

	before := pool.Stat().AcquiredConns()
	for i := range 50 {
		lock, err := store.LockGame(ctx, "recycle-me")
		is.NoErr(err)
		_ = i
		lock.Unlock(ctx)
	}
	is.Equal(pool.Stat().AcquiredConns(), before)
}

// The bound on simultaneous holders is what keeps a burst from deadlocking the
// process: a lock ties up a connection, and its holder needs a second one to do
// any work. If every connection in the pool were held as a lock, nobody could
// proceed and nobody would ever let go.
//
// Half the pool, so N/2 holders each still find a connection for their queries.
func TestLockGameLeavesConnectionsForWork(t *testing.T) {
	is := is.New(t)
	pool := lockTestPool(t)
	store := &DBStore{dbPool: pool}
	ctx := context.Background()

	maxConns := int(pool.Config().MaxConns)
	if maxConns < 4 {
		t.Skipf("pool too small to be meaningful: %d", maxConns)
	}

	// Hold as many locks as the semaphore permits, on distinct games so they
	// never wait on each other.
	var locks []*GameLock
	for i := range maxConns / 2 {
		l, err := store.LockGame(ctx, fmt.Sprintf("burst-%d", i))
		is.NoErr(err)
		locks = append(locks, l)
	}
	defer func() {
		for _, l := range locks {
			l.Unlock(ctx)
		}
	}()

	// With every permitted lock held, ordinary work must still get a
	// connection. This is the query a save would need to run.
	short, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var n int
	is.NoErr(pool.QueryRow(short, "SELECT 1").Scan(&n))
	is.Equal(n, 1)

	// And one more lock must wait for a slot rather than take the last
	// connections out from under that work.
	shorter, cancel2 := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel2()
	_, err := store.LockGame(shorter, "burst-overflow")
	is.True(err != nil)
}
