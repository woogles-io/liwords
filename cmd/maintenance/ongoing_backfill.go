package main

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
)

// OngoingGamesBackfill writes the ongoing_games row for every live game that
// does not have one: games that were already live when WRITE_ONGOING_GAMES
// went on and have not been saved since.
//
// Run it once, after the flag is on in the API. Before that, a game that ends
// in between keeps a row it should not have, because the save that ended it
// did not know to delete it. Requiring the flag here as well is how the run
// says it knows that.
//
// It is safe to run again: a game that has a row, or has ended, is skipped,
// and each game is written under its own lock, the one every move holds.
func OngoingGamesBackfill() error {
	ctx := log.Logger.WithContext(context.Background())
	cfg := &config.Config{}
	cfg.Load(nil)

	if !cfg.WriteOngoingGames {
		return errors.New("WRITE_ONGOING_GAMES is not set: turn it on in the API first, then run this with it set")
	}
	ctx = cfg.WithContext(ctx)

	allStores, err := initLeagueStores(ctx, cfg)
	if err != nil {
		return err
	}

	ids, err := allStores.GameStore.ListLiveGamesWithoutOngoingRow(ctx)
	if err != nil {
		return err
	}
	log.Info().Int("games", len(ids)).Msg("ongoing-backfill-starting")

	wrote, skipped, failed := 0, 0, 0
	for _, id := range ids {
		ok, err := allStores.GameStore.BackfillOngoingGame(ctx, id)
		switch {
		case err != nil:
			failed++
			log.Error().Err(err).Str("gameID", id).Msg("ongoing-backfill-failed")
		case ok:
			wrote++
		default:
			skipped++ // ended, or a save wrote the row first
		}
	}

	remaining, err := allStores.GameStore.ListLiveGamesWithoutOngoingRow(ctx)
	if err != nil {
		return err
	}
	log.Info().Int("games", len(ids)).Int("wrote", wrote).Int("skipped", skipped).
		Int("failed", failed).Int("remaining", len(remaining)).Msg("ongoing-backfill-done")
	if failed > 0 {
		return errors.New("some games failed; see ongoing-backfill-failed, then run again")
	}
	return nil
}
