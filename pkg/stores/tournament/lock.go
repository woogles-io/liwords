package tournament

// Changing a tournament.
//
// A change is a read-modify-write of the whole tournament: load it, change
// it, save it. With more than one API server, two changes to the same
// tournament can run at once, and without a lock across all three steps the
// second save silently discards the first (a result, a pairing, a player).
// So every change goes through GetForUpdate, which holds the tournament until
// the change is saved or abandoned:
//
//  1. an in-process lock on the tournament ID, so that changes on one server
//     wait in Go, holding no database connection;
//  2. a transaction, holding a transaction-scoped advisory lock on the
//     tournament ID, so that changes on different servers wait for each
//     other in Postgres;
//  3. the load, inside the transaction; then the caller's change; then the
//     save and the commit, which ends the advisory lock.
//
// Why the in-process lock first. At the end of a round, many games of one
// tournament end at once, and each records its result while holding its
// game's lock (pkg/stores/game/lock.go). If they all waited for the advisory
// lock inside open transactions, each would hold a connection while waiting:
// up to one per game-lock slot, on top of the game locks' own, leaving the
// rest of the server almost none until the queue drained. Waiting in Go
// first, a server holds at most one connection per tournament for this.
//
// The rule that keeps this safe, the same one the game lock states: between
// GetForUpdate and Save or Release, run no queries on the pool. The holder
// already has its transaction's connection; asking the pool for another while
// holding it is how a burst of holders runs the pool dry. Look up users and
// anything else the change needs before calling GetForUpdate, and send socket
// messages after Save, not before (the event channel can block).
//
// It is an advisory lock rather than SELECT ... FOR UPDATE on the row because
// the lock is on the tournament's ID, not on where it is stored: it stays the
// same when divisions move to their own tables, and it can cover them and any
// other table a change writes.

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/woogles-io/liwords/pkg/entity"
	tl "github.com/woogles-io/liwords/pkg/tournament"
)

// LockTimeout bounds the wait for another server's change to the same
// tournament. A change takes tens of milliseconds, so reaching it means
// something is stuck; the caller gets an error rather than queueing forever.
const LockTimeout = 10 * time.Second

// releaseTimeout bounds the rollback after an abandoned change. It doesn't
// use the caller's context, which may be why the change was abandoned.
const releaseTimeout = 5 * time.Second

// keyedMutex is a set of mutexes keyed by tournament ID. An entry exists
// only while some goroutine holds or waits for it.
type keyedMutex struct {
	mu      sync.Mutex
	entries map[string]*keyedEntry
}

type keyedEntry struct {
	held chan struct{} // capacity one: full while held
	refs int           // holders and waiters
}

// lock waits for the key's mutex, or for ctx to end. The returned function
// releases it.
func (k *keyedMutex) lock(ctx context.Context, key string) (func(), error) {
	k.mu.Lock()
	if k.entries == nil {
		k.entries = map[string]*keyedEntry{}
	}
	e := k.entries[key]
	if e == nil {
		e = &keyedEntry{held: make(chan struct{}, 1)}
		k.entries[key] = e
	}
	e.refs++
	k.mu.Unlock()

	select {
	case e.held <- struct{}{}:
	case <-ctx.Done():
		k.drop(key, e)
		return nil, ctx.Err()
	}
	return func() {
		<-e.held
		k.drop(key, e)
	}, nil
}

func (k *keyedMutex) drop(key string, e *keyedEntry) {
	k.mu.Lock()
	e.refs--
	if e.refs == 0 {
		delete(k.entries, key)
	}
	k.mu.Unlock()
}

// lockedTournament is a tournament loaded by GetForUpdate.
type lockedTournament struct {
	s      *DBStore
	tx     pgx.Tx
	tm     *entity.Tournament
	unlock func()
	done   bool
}

// GetForUpdate loads the tournament for a change, under its lock (see the
// top of this file). The caller must call Release, normally deferred, and
// Save to keep the change:
//
//	t, lk, err := ts.GetForUpdate(ctx, id)
//	if err != nil {
//		return err
//	}
//	defer lk.Release()
//	// change t, with no pool queries
//	if err := lk.Save(ctx); err != nil {
//		return err
//	}
//	// send messages
func (s *DBStore) GetForUpdate(ctx context.Context, id string) (*entity.Tournament, tl.TournamentLock, error) {
	ctx, span := startSpan(ctx, "GetForUpdate", id)
	defer span.End()

	unlock, err := s.local.lock(ctx, id)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, err
	}
	span.AddEvent("local-lock")

	tm, tx, err := s.lockAndLoad(ctx, id)
	if err != nil {
		unlock()
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, err
	}
	return tm, &lockedTournament{s: s, tx: tx, tm: tm, unlock: unlock}, nil
}

func (s *DBStore) lockAndLoad(ctx context.Context, id string) (*entity.Tournament, pgx.Tx, error) {
	tx, err := s.dbPool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	fail := func(err error) (*entity.Tournament, pgx.Tx, error) {
		rollback(tx)
		return nil, nil, err
	}
	// SET LOCAL takes no parameters, so the timeout is formatted in.
	if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '"+LockTimeout.String()+"'"); err != nil {
		return fail(err)
	}
	q := s.queries.WithTx(tx)
	if err := q.LockTournament(ctx, id); err != nil {
		return fail(err)
	}
	row, err := q.GetTournamentByUUID(ctx, text(id))
	if err != nil {
		return fail(err)
	}
	tm, err := s.dbObjToEntity(&row)
	if err != nil {
		return fail(err)
	}
	return tm, tx, nil
}

func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), releaseTimeout)
	defer cancel()
	_ = tx.Rollback(ctx)
}

var errReleased = errors.New("tournament lock already released")

// Save writes the tournament, commits, and releases the lock.
func (l *lockedTournament) Save(ctx context.Context) error {
	if l.done {
		return errReleased
	}
	ctx, span := startSpan(ctx, "Save", l.tm.UUID)
	defer span.End()
	defer l.Release()
	if err := l.s.update(ctx, l.s.queries.WithTx(l.tx), l.tm); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err := l.tx.Commit(ctx); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetAttributes(attribute.Bool("committed", true))
	return nil
}

// Release abandons the change if it wasn't saved, and releases the lock.
// It does nothing after Save, so it can always be deferred.
func (l *lockedTournament) Release() {
	if l.done {
		return
	}
	l.done = true
	rollback(l.tx) // a no-op after a commit
	l.unlock()
}
