package game

// Holding a game still while it is played.
//
// A move is a read-modify-write: load the position, play into it, save it. The
// lock has to span all three. If it only covers the save -- which is where it
// started out -- then two servers can load the same position, each play a legal
// move onto it, and each save atomically. Both saves succeed, both are
// internally consistent, and the second silently discards the first. Nothing
// looks broken; a move just vanishes.
//
// So the lock is taken before the load and released after the save, which means
// it has to outlive any single transaction, which means session scope. That has
// one real cost: a session-level advisory lock lives on a *connection*, so
// holding one means holding a pooled connection for as long as the move takes.
// GameLock owns that connection and is the only thing that may release it.
//
// The thing to know before adding another lock anywhere near this: advisory
// locks are per session, and pool connections are different sessions. Code that
// runs while a GameLock is held must NOT take pg_advisory_xact_lock on the same
// game -- it would run on a different pooled connection and wait for a lock its
// own caller is holding, which is a deadlock every single time, not a race.
// That is why Set and CommitArchival use a plain transaction: they are always
// called underneath one of these.

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

// Holding a lock ties up a pooled connection for the whole move, and the move
// needs a second connection for its own queries. So the number of simultaneous
// lock holders has to be bounded below the pool size, or a burst deadlocks the
// process outright: every connection held by a waiter that cannot proceed
// without one more.
//
// The bound is half the pool. At most N/2 connections are held as locks, which
// leaves at least N/2 for the N/2 holders to run their queries -- one each,
// which is all a save needs. That is a proof rather than a guess, and it is
// derived from the pool's own configuration rather than a constant that would
// drift away from it.
func (s *DBStore) lockSemaphore() chan struct{} {
	s.lockSlotsOnce.Do(func() {
		n := 1
		if s.dbPool != nil {
			if half := int(s.dbPool.Config().MaxConns) / 2; half > n {
				n = half
			}
		}
		s.lockSlots = make(chan struct{}, n)
	})
	return s.lockSlots
}

// ErrGameLockBusy reports that a game stayed locked for longer than we are
// willing to wait.
//
// Returned rather than waited out, because a request that hangs indefinitely on
// a stuck holder is worse than one that fails and can be retried.
var ErrGameLockBusy = errors.New("game: could not acquire the game lock in time")

const (
	// GameLockMaxWait bounds how long LockGame will keep trying when the
	// context carries no earlier deadline.
	GameLockMaxWait = 10 * time.Second
	// The poll interval grows from min to max. Contention on one game is rare
	// -- it needs two writers on the same game at the same moment -- so the
	// uncontended path is a single round trip and never sleeps.
	gameLockMinPoll = 5 * time.Millisecond
	gameLockMaxPoll = 60 * time.Millisecond
)

// GameLock is a held session-level advisory lock, together with the connection
// holding it.
//
// The connection is part of the handle on purpose. An unlock keyed only by game
// id can be issued on a different pooled connection, where it silently does
// nothing -- pg_advisory_unlock returns false and the real lock stays held
// until the process dies. Keeping them together makes that unexpressible.
type GameLock struct {
	conn   *pgxpool.Conn
	slots  chan struct{}
	gameID string
	once   sync.Once
}

// GameID is the game this lock is held for.
func (l *GameLock) GameID() string { return l.gameID }

// LockGame takes the game's lock, waiting for it if another writer holds it.
//
// The caller must release it, on every path:
//
//	lock, err := store.LockGame(ctx, gameID)
//	if err != nil {
//	    return err
//	}
//	defer lock.Unlock(ctx)
func (s *DBStore) LockGame(ctx context.Context, gameID string) (*GameLock, error) {
	if gameID == "" {
		return nil, fmt.Errorf("game: LockGame needs a game id")
	}
	deadline := time.Now().Add(GameLockMaxWait)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}

	// Take a slot before a connection, which is the whole point of the bound.
	slots := s.lockSemaphore()
	select {
	case slots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	release := func() { <-slots }

	conn, err := s.dbPool.Acquire(ctx)
	if err != nil {
		release()
		return nil, err
	}
	q := models.New(conn)

	// pg_try_advisory_lock rather than pg_advisory_lock, so that waiting is
	// this function's decision rather than the database's. Blocking would need
	// a lock_timeout, and lock_timeout is session state that would outlive the
	// request on a pooled connection unless it were carefully reset -- more
	// moving parts than a poll loop, for a lock that is almost never contended.
	poll := gameLockMinPoll
	for {
		got, err := q.TryLockGameSession(ctx, gameID)
		if err != nil {
			conn.Release()
			release()
			return nil, err
		}
		if got {
			return &GameLock{conn: conn, slots: slots, gameID: gameID}, nil
		}
		if time.Now().After(deadline) {
			conn.Release()
			release()
			return nil, fmt.Errorf("%w: %s", ErrGameLockBusy, gameID)
		}
		// Jittered, so that several waiters do not retry in lockstep.
		jitter := time.Duration(rand.Int64N(int64(poll / 2)))
		select {
		case <-ctx.Done():
			conn.Release()
			release()
			return nil, ctx.Err()
		case <-time.After(poll + jitter):
		}
		if poll < gameLockMaxPoll {
			poll *= 2
		}
	}
}

// Unlock releases the lock and returns the connection to the pool. Safe to call
// more than once; only the first call does anything.
func (l *GameLock) Unlock(ctx context.Context) {
	if l == nil {
		return
	}
	l.once.Do(func() {
		// The connection goes back to the pool no matter what happens to the
		// unlock. Leaking it would be worse than leaking the lock: a lock dies
		// with the connection, and a connection that is never released is gone
		// for the life of the process.
		defer func() {
			l.conn.Release()
			if l.slots != nil {
				<-l.slots
			}
		}()

		// A caller's context may already be cancelled by the time a deferred
		// unlock runs -- that is the normal shape of a failed request -- and
		// the unlock still has to happen.
		if ctx == nil {
			ctx = context.Background()
		}
		if ctx.Err() != nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
		}
		released, err := models.New(l.conn).UnlockGameSession(ctx, l.gameID)
		if err != nil {
			// Destroy the connection rather than return a possibly still-locked
			// one to the pool, where it would block this game forever.
			l.conn.Conn().Close(ctx)
			logUnlockFailure(l.gameID, err)
			return
		}
		if !released {
			logUnlockFailure(l.gameID, errors.New("lock was not held by this connection"))
		}
	})
}

// logUnlockFailure records a lock that may still be held. It is rare enough and
// serious enough -- every later move on that game waits out GameLockMaxWait and
// then fails -- that it is never thinned.
func logUnlockFailure(gameID string, err error) {
	log.Error().Err(err).Str("gameID", gameID).Msg("game-unlock-failed")
}
