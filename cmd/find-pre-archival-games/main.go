// find-pre-archival-games identifies active correspondence games whose
// game_turns row count is less than the number of events in their bytea history.
// These games predate or crossed the dual-write cutover; when they finish,
// ArchiveAndCleanup falls back to uploading the bytea history directly.
//
// Output: CSV to --csv-out (default stdout), one line per flagged game:
//
//	uuid,created_at,updated_at,turns_count,event_count,missing
//
// Requires the same DB env vars as liwords-api: DB_CONN_DSN.
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

func main() {
	batchSize := flag.Int("batch-size", 500, "Number of rows per DB fetch")
	workers := flag.Int("workers", 8, "Concurrent unmarshal workers")
	csvOut := flag.String("csv-out", "", "CSV output path (default: stdout)")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := &config.Config{}
	cfg.Load(nil)

	poolCfg, err := pgxpool.ParseConfig(cfg.DBConnUri)
	if err != nil {
		log.Fatal().Err(err).Msg("db-parse-config-failed")
	}
	poolCfg.MaxConns = int32(*workers) + 4
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("db-connect-failed")
	}
	defer pool.Close()

	var out *os.File
	if *csvOut == "" {
		out = os.Stdout
	} else {
		out, err = os.Create(*csvOut)
		if err != nil {
			log.Fatal().Err(err).Str("path", *csvOut).Msg("csv-create-failed")
		}
		defer out.Close()
	}

	w := csv.NewWriter(out)
	if err := w.Write([]string{"uuid", "created_at", "updated_at", "turns_count", "event_count", "missing"}); err != nil {
		log.Fatal().Err(err).Msg("csv-header-failed")
	}

	queries := models.New(pool)

	type result struct {
		uuid       string
		createdAt  time.Time
		updatedAt  time.Time
		turnsCount int
		eventCount int
	}

	var (
		mu          sync.Mutex
		results     []result
		totalScanned int64
		zeroTurns   int64
		partialTurns int64
	)

	sem := make(chan struct{}, *workers)
	var wg sync.WaitGroup
	afterUUID := ""

	for {
		if ctx.Err() != nil {
			log.Info().Msg("find-pre-archival-games: interrupted")
			break
		}

		rows, err := queries.ListActiveCorrespondenceForArchivalAudit(ctx, models.ListActiveCorrespondenceForArchivalAuditParams{
			AfterUuid: pgtype.Text{String: afterUUID, Valid: true},
			Lim:       int32(*batchSize),
		})
		if err != nil {
			log.Fatal().Err(err).Str("cursor", afterUUID).Msg("db-list-failed")
		}
		if len(rows) == 0 {
			break
		}

		for _, r := range rows {
			row := r
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()

				hist := &macondopb.GameHistory{}
				if err := proto.Unmarshal(row.History, hist); err != nil {
					log.Error().Err(err).Str("uuid", row.Uuid.String).Msg("unmarshal-failed")
					return
				}
				eventCount := len(hist.Events)
				turnsCount := int(row.TurnsCount)

				mu.Lock()
				totalScanned++
				if turnsCount < eventCount {
					res := result{
						uuid:       row.Uuid.String,
						createdAt:  row.CreatedAt.Time,
						updatedAt:  row.UpdatedAt.Time,
						turnsCount: turnsCount,
						eventCount: eventCount,
					}
					if turnsCount == 0 {
						zeroTurns++
					} else {
						partialTurns++
					}
					results = append(results, res)
				}
				mu.Unlock()
			}()
		}
		wg.Wait()

		afterUUID = rows[len(rows)-1].Uuid.String
	}

	// Write CSV sorted by creation date (results are in UUID order; sort by createdAt).
	for _, r := range results {
		missing := r.eventCount - r.turnsCount
		if err := w.Write([]string{
			r.uuid,
			r.createdAt.UTC().Format(time.RFC3339),
			r.updatedAt.UTC().Format(time.RFC3339),
			strconv.Itoa(r.turnsCount),
			strconv.Itoa(r.eventCount),
			strconv.Itoa(missing),
		}); err != nil {
			log.Error().Err(err).Str("uuid", r.uuid).Msg("csv-write-failed")
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		log.Error().Err(err).Msg("csv-flush-failed")
	}

	flagged := int64(len(results))
	log.Info().
		Int64("scanned", totalScanned).
		Int64("flagged", flagged).
		Int64("zero_turns", zeroTurns).
		Int64("partial_turns", partialTurns).
		Msg("find-pre-archival-games-done")

	if flagged > 0 {
		var oldestFlagged, newestFlagged time.Time
		for _, r := range results {
			if oldestFlagged.IsZero() || r.createdAt.Before(oldestFlagged) {
				oldestFlagged = r.createdAt
			}
			if r.createdAt.After(newestFlagged) {
				newestFlagged = r.createdAt
			}
		}
		log.Info().
			Time("oldest_flagged_created_at", oldestFlagged).
			Time("newest_flagged_created_at", newestFlagged).
			Msg("flagged-range")
	}
}

