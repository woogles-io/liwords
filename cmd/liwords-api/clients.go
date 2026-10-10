package main

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/user"
)

const (
	clientRecordRetention = 180 * 24 * time.Hour
	clientPruneInterval   = 24 * time.Hour
)

// pruneClientRecords periodically deletes client records that haven't been
// seen within the retention window. Safe to run on every API node.
func pruneClientRecords(ctx context.Context, us user.Store) {
	ticker := time.NewTicker(clientPruneInterval)
	defer ticker.Stop()
	for {
		n, err := us.PruneClients(ctx, time.Now().Add(-clientRecordRetention))
		if err != nil {
			log.Err(err).Msg("prune-client-records")
		} else {
			log.Info().Int64("deleted", n).Msg("pruned-client-records")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
