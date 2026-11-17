package maintenance

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const pbActive = ipc.SeasonStatus_SEASON_ACTIVE

func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// CancelledGamesCleanup deletes cancelled games more than 2 days old.
func (t *Tasks) CancelledGamesCleanup(ctx context.Context) error {
	// game_end_reason 7 is CANCELLED.
	res, err := t.stores.DBPool.Exec(ctx, `
		DELETE FROM games
		WHERE game_end_reason = 7
		  AND created_at < NOW() - INTERVAL '2 days'`)
	if err != nil {
		return err
	}
	log.Ctx(ctx).Info().Int64("deleted", res.RowsAffected()).Msg("cancelled-games-cleanup")
	return nil
}

// MonitoringStreamsCleanup deletes the monitoring stream keys of finished
// tournaments, and any more than a month old.
func (t *Tasks) MonitoringStreamsCleanup(ctx context.Context) error {
	res, err := t.stores.DBPool.Exec(ctx, `
		DELETE FROM monitoring_streams
		WHERE tournament_id IN (SELECT uuid FROM tournaments WHERE is_finished = true)
		   OR created_at < NOW() - INTERVAL '1 month'`)
	if err != nil {
		return err
	}
	log.Ctx(ctx).Info().Int64("deleted", res.RowsAffected()).Msg("monitoring-streams-cleanup")
	return nil
}

// ExpiredSessionsCleanup deletes sessions that have expired.
func (t *Tasks) ExpiredSessionsCleanup(ctx context.Context) error {
	res, err := t.stores.DBPool.Exec(ctx, `DELETE FROM db_sessions WHERE expires_at < NOW()`)
	if err != nil {
		return err
	}
	log.Ctx(ctx).Info().Int64("deleted", res.RowsAffected()).Msg("expired-sessions-cleanup")
	return nil
}

// unverifiedUsers selects users who haven't verified their email within 48
// hours of signing up.
const unverifiedUsers = `SELECT id FROM users WHERE verified = false AND created_at < NOW() - INTERVAL '48 hours'`

// UnverifiedUsersCleanup deletes users who haven't verified their email
// within 48 hours, with the rows that point at them, in one transaction.
func (t *Tasks) UnverifiedUsersCleanup(ctx context.Context) error {
	tx, err := t.stores.DBPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// The foreign keys to users from blockings, followings and user_actions
	// are RESTRICT or NO ACTION, so a single leftover row would abort the
	// transaction and delete nothing. Both sides of each are covered: an
	// unverified account can be the blocker, the follower, or the applier or
	// remover of someone else's action.
	steps := []struct{ name, sql string }{
		{"blockings", `DELETE FROM blockings WHERE user_id IN (` + unverifiedUsers + `) OR blocker_id IN (` + unverifiedUsers + `)`},
		{"followings", `DELETE FROM followings WHERE user_id IN (` + unverifiedUsers + `) OR follower_id IN (` + unverifiedUsers + `)`},
		{"user_actions", `DELETE FROM user_actions WHERE user_id IN (` + unverifiedUsers + `) OR applier_id IN (` + unverifiedUsers + `) OR remover_id IN (` + unverifiedUsers + `)`},
		{"profiles", `DELETE FROM profiles WHERE user_id IN (` + unverifiedUsers + `)`},
		{"users", `DELETE FROM users WHERE verified = false AND created_at < NOW() - INTERVAL '48 hours'`},
	}
	ev := log.Ctx(ctx).Info()
	for _, s := range steps {
		res, err := tx.Exec(ctx, s.sql)
		if err != nil {
			return err
		}
		ev = ev.Int64(s.name, res.RowsAffected())
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	ev.Msg("unverified-users-cleanup")
	return nil
}
