package game

// Cross-node serialization for a game's write path.
//
// Every write that touches a game takes an advisory lock on its id for the
// duration of one transaction. This exists for two reasons, both of which are
// load-bearing.
//
// First, it is what makes reconstructing a position from game_turns safe. The
// read path was disabled because AppendTurns commits a terminal event before
// Set commits game_end_reason: a load landing between those two commits sees a
// finished game that still claims to be in progress. Putting both writes in one
// locked transaction closes the window rather than narrowing it.
//
// Second, it replaces Cache.LockGame, which is a process-local map of mutexes.
// That serializes writers inside one process and nothing at all between app
// servers, which was fine when there was one server and is not fine now.
//
// pg_advisory_xact_lock releases on commit or rollback, so there is no lock to
// leak if a handler panics or a request is cancelled mid-transaction.

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

// WithGameLock runs fn inside a transaction holding the advisory lock for
// gameUUID, and commits if fn returns nil.
//
// fn receives queries bound to that transaction; anything it does through them
// is covered by the lock, and anything it does outside them is not. Nested
// calls for the same game on the same connection are safe -- advisory locks are
// reentrant for the holder -- but a second call on a different connection will
// block until the first transaction ends, which is the point.
func WithGameLock(ctx context.Context, pool *pgxpool.Pool, gameUUID string,
	fn func(context.Context, *models.Queries) error) error {

	if pool == nil {
		return fmt.Errorf("game: WithGameLock needs a connection pool")
	}
	if gameUUID == "" {
		return fmt.Errorf("game: WithGameLock needs a game id")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	// No-op once the transaction has been committed.
	defer tx.Rollback(ctx)

	qtx := models.New(tx)
	if err := qtx.LockGameForWrite(ctx, gameUUID); err != nil {
		return fmt.Errorf("game: locking %s: %w", gameUUID, err)
	}
	if err := fn(ctx, qtx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
