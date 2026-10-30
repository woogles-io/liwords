package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

const (
	// A game that ended this recently may still have its archive in flight.
	archiveSweepGrace = 10 * time.Minute
	// Keeps one run short; whatever is left over waits for the next hour.
	archiveSweepMaxGames = 200
)

// PendingArchivalSweep archives finished games that still have turn rows and
// no S3 archive: an upload that failed, or that never ran because the server
// stopped first.
func PendingArchivalSweep() error {
	log.Info().Msg("starting pending archival sweep")

	ctx := log.Logger.WithContext(context.Background())
	cfg := &config.Config{}
	cfg.Load(nil)

	allStores, err := initLeagueStores(ctx, cfg)
	if err != nil {
		return err
	}
	if allStores.GameHistoryArchiver == nil {
		return errors.New("game history archiver is not configured")
	}

	ids, err := allStores.Queries.ListStalePendingArchival(ctx, models.ListStalePendingArchivalParams{
		EndedBefore: pgtype.Timestamptz{Time: time.Now().Add(-archiveSweepGrace), Valid: true},
		MaxGames:    archiveSweepMaxGames,
	})
	if err != nil {
		return err
	}

	archived := 0
	for _, id := range ids {
		g, err := allStores.GameStore.Get(ctx, id.String)
		if err != nil {
			log.Error().Err(err).Str("gameID", id.String).Msg("archive-sweep-load-failed")
			continue
		}
		if err := allStores.GameHistoryArchiver.ArchiveAndCleanup(ctx, g); err != nil {
			log.Error().Err(err).Str("gameID", id.String).Msg("archive-sweep-failed")
			continue
		}
		archived++
	}
	log.Info().Int("pending", len(ids)).Int("archived", archived).Msg("completed pending archival sweep")
	return nil
}
