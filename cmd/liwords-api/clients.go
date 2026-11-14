package main

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/woogles-io/liwords/pkg/user"
)

const clientRecordRetention = 180 * 24 * time.Hour

// pruneClientRecords deletes client records that haven't been seen within
// the retention window. Run daily, as a job.
func pruneClientRecords(us user.Store) func(context.Context) error {
	return func(ctx context.Context) error {
		n, err := us.PruneClients(ctx, time.Now().Add(-clientRecordRetention))
		if err != nil {
			return err
		}
		zerolog.Ctx(ctx).Info().Int64("deleted", n).Msg("pruned-client-records")
		return nil
	}
}
