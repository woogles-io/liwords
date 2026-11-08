// backfill-ongoing-games writes the ongoing_games row for live games that do
// not have one: games that were already live when WRITE_ONGOING_GAMES went on
// and have not been saved since. Run it once, by hand, after the flag is on in
// the API.
//
// Never before: a game that ends in between would keep its row, because the
// save that ended it did not know to delete it. To say that you know this, it
// refuses to run unless WRITE_ONGOING_GAMES=1 is set for it too.
//
// Each game is written under its own game lock, the one every move holds, and
// skipped if it has ended or already has a row, so it is safe to run again.
//
// Requires the same env vars as liwords-api for the database (DB_HOST,
// DB_PORT, DB_NAME, DB_USER, DB_PASSWORD, DB_SSL_MODE) and MACONDO_DATA_PATH:
// each game is loaded, which needs its lexicon and letter distribution.
//
//	WRITE_ONGOING_GAMES=1 ./backfill-ongoing-games --dry-run
//	WRITE_ONGOING_GAMES=1 ./backfill-ongoing-games --all
//	WRITE_ONGOING_GAMES=1 ./backfill-ongoing-games --game-id <uuid>
package main

import (
	"context"
	"flag"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	gamestore "github.com/woogles-io/liwords/pkg/stores/game"
	userstore "github.com/woogles-io/liwords/pkg/stores/user"
)

func main() {
	gameID := flag.String("game-id", "", "backfill a single game by UUID")
	all := flag.Bool("all", false, "backfill every live game that has no row")
	dryRun := flag.Bool("dry-run", false, "list the games that have no row, and write nothing")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if *gameID == "" && !*all && !*dryRun {
		log.Fatal().Msg("specify --dry-run, --all or --game-id <uuid>")
	}

	cfg := &config.Config{}
	cfg.Load(nil)
	if !cfg.WriteOngoingGames {
		log.Fatal().Msg("WRITE_ONGOING_GAMES is not set: turn it on in the API first, then run this with it set")
	}
	ctx := cfg.WithContext(log.Logger.WithContext(context.Background()))

	pool, err := pgxpool.New(ctx, cfg.DBConnUri)
	if err != nil {
		log.Fatal().Err(err).Msg("db-connect-failed")
	}
	defer pool.Close()

	userStore, err := userstore.NewDBStore(pool)
	if err != nil {
		log.Fatal().Err(err).Msg("user-store-failed")
	}
	store, err := gamestore.NewDBStore(cfg, userStore, pool)
	if err != nil {
		log.Fatal().Err(err).Msg("game-store-failed")
	}

	ids := []string{*gameID}
	if *gameID == "" {
		ids, err = store.ListLiveGamesWithoutOngoingRow(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("list-failed")
		}
	}
	log.Info().Int("games", len(ids)).Msg("live games without a row")
	if *dryRun {
		for _, id := range ids {
			log.Info().Str("gameID", id).Msg("would backfill")
		}
		return
	}

	wrote, skipped, failed := 0, 0, 0
	for _, id := range ids {
		ok, err := store.BackfillOngoingGame(ctx, id)
		switch {
		case err != nil:
			failed++
			log.Error().Err(err).Str("gameID", id).Msg("backfill-failed")
		case ok:
			wrote++
		default:
			skipped++ // ended, or a save wrote the row first
		}
	}

	remaining, err := store.ListLiveGamesWithoutOngoingRow(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("list-failed")
	}
	log.Info().Int("wrote", wrote).Int("skipped", skipped).Int("failed", failed).
		Int("remaining", len(remaining)).Msg("done")
	if failed > 0 {
		log.Warn().Msg("some games failed (a busy lock is the usual reason); run again")
		os.Exit(1)
	}
}
