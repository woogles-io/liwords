// backfill-meta-events copies the meta events saved in games.meta_events into
// game_meta_events, for the games saved before the API started writing that
// table. Run it once, by hand, after that release is deployed.
//
// Not before: a game saved by the previous release after its batch has run
// would keep events the table doesn't have. (Those heal on the game's next
// save, which writes its whole list, but a finished game has no next save.)
//
// It walks games by id in batches, and each batch is one INSERT ... SELECT in
// the database, skipping events already copied, so it is safe to stop and run
// again; --from resumes where a run stopped. Most games have no meta events,
// so most batches insert nothing.
//
// It reads the whole games heap (18 GB in November 2026): on production a
// 50,000-id batch took ~20 s, almost all of it disk reads, so the whole run is
// about an hour and a half of steady I/O. Run it at a quiet hour, and not
// during the daily pg_dump.
//
// Requires the same env vars as liwords-api for the database (DB_HOST,
// DB_PORT, DB_NAME, DB_USER, DB_PASSWORD, DB_SSL_MODE).
//
//	./backfill-meta-events --dry-run
//	./backfill-meta-events
//	./backfill-meta-events --from 6000000 --batch 5000 --pause 1s
package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	gamestore "github.com/woogles-io/liwords/pkg/stores/game"
	userstore "github.com/woogles-io/liwords/pkg/stores/user"
)

func main() {
	from := flag.Int64("from", 0, "first games.id to copy")
	batch := flag.Int64("batch", 10000, "games ids per batch (~4 s each on production)")
	pause := flag.Duration("pause", 500*time.Millisecond, "sleep between batches, to leave the database room")
	dryRun := flag.Bool("dry-run", false, "print the id range and batch count, and write nothing")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if *batch <= 0 {
		log.Fatal().Msg("--batch must be positive")
	}

	cfg := &config.Config{}
	cfg.Load(nil)
	ctx := log.Logger.WithContext(context.Background())

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

	maxID, err := store.MaxGameID(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("max-id-failed")
	}
	nBatches := (maxID - *from + *batch) / *batch
	log.Info().Int64("from", *from).Int64("maxID", maxID).Int64("batches", nBatches).Msg("plan")
	if *dryRun {
		return
	}

	var inserted int64
	start := time.Now()
	for lo := *from; lo <= maxID; lo += *batch {
		hi := lo + *batch
		n, err := store.BackfillMetaEvents(ctx, int32(lo), int32(hi))
		if err != nil {
			log.Fatal().Err(err).Int64("from", lo).Msg("batch-failed; rerun with this --from")
		}
		inserted += n
		log.Info().Int64("from", lo).Int64("to", hi).Int64("inserted", n).
			Int64("total", inserted).Dur("elapsed", time.Since(start)).Msg("batch")
		time.Sleep(*pause)
	}
	log.Info().Int64("inserted", inserted).Dur("elapsed", time.Since(start)).Msg("done")
}
