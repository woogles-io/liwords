package game

// Writing several tables as one unit.
//
// Serialization between writers is not this file's job -- that is the session
// lock in lock.go, held across the whole load-modify-save of a move. This is
// only about atomicity: a move touches game_turns and games, and a reader must
// never see one without the other.
//
// Deliberately no advisory lock here. An earlier version took
// pg_advisory_xact_lock inside the save, and that is a deadlock rather than a
// safety net once callers hold a session lock for the same game: the
// transaction runs on a different pooled connection, which is a different
// session, so it waits for a lock its own caller is holding and never gets it.
// Verified: session lock on one connection, xact lock on another for the same
// key, blocks until lock_timeout. See lock.go.

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

// withGameTx runs fn in a transaction and commits if it returns nil.
//
// fn receives queries bound to that transaction; anything it does through them
// commits or rolls back as one unit, and anything it does outside them does
// not.
func withGameTx(ctx context.Context, pool *pgxpool.Pool,
	fn func(context.Context, *models.Queries) error) error {

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	// No-op once the transaction has been committed.
	defer tx.Rollback(ctx)

	if err := fn(ctx, models.New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
