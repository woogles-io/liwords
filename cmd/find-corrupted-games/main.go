// find-corrupted-games scans active correspondence games and calls
// NewFromHistory on each one, reporting any games where history replay fails.
// These games cannot be loaded by the server and are effectively invisible
// to players. Output: one line per corrupted game: uuid, error.
//
// Requires the same DB env vars as liwords-api: DB_CONN_DSN.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

func main() {
	batchSize := flag.Int("batch-size", 200, "Number of rows per DB fetch")
	workers := flag.Int("workers", 4, "Concurrent workers")
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
	poolCfg.MaxConns = int32(*workers) + 2
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("db-connect-failed")
	}
	defer pool.Close()

	queries := models.New(pool)

	var (
		mu           sync.Mutex
		totalScanned int64
		corrupted    []string
	)

	sem := make(chan struct{}, *workers)
	var wg sync.WaitGroup
	afterUUID := ""

	for {
		if ctx.Err() != nil {
			log.Info().Msg("interrupted")
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
					mu.Lock()
					totalScanned++
					log.Error().Err(err).Str("uuid", row.Uuid.String).Msg("unmarshal-failed")
					corrupted = append(corrupted, row.Uuid.String+"\tunmarshal: "+err.Error())
					mu.Unlock()
					return
				}

				lexicon := hist.GetLexicon()
				boardLayout := hist.GetBoardLayout()
				letterDist := hist.GetLetterDistribution()
				variant := hist.GetVariant()

				if boardLayout == "" {
					boardLayout = "CrosswordGame"
				}
				if letterDist == "" {
					letterDist = "english"
				}
				if variant == "" {
					variant = "classic"
				}

				rules, err := macondogame.NewBasicGameRules(
					cfg.MacondoConfig(), lexicon, boardLayout,
					letterDist, macondogame.CrossScoreOnly,
					macondogame.Variant(variant))
				if err != nil {
					mu.Lock()
					totalScanned++
					log.Error().Err(err).Str("uuid", row.Uuid.String).Str("lexicon", lexicon).Msg("rules-failed")
					corrupted = append(corrupted, row.Uuid.String+"\trules: "+err.Error())
					mu.Unlock()
					return
				}

				_, replayErr := macondogame.NewFromHistory(hist, rules, len(hist.Events))

				mu.Lock()
				totalScanned++
				if replayErr != nil {
					log.Error().Err(replayErr).Str("uuid", row.Uuid.String).Msg("corrupted-game")
					corrupted = append(corrupted, row.Uuid.String+"\t"+replayErr.Error())
				}
				mu.Unlock()
			}()
		}
		wg.Wait()

		afterUUID = rows[len(rows)-1].Uuid.String
	}

	log.Info().
		Int64("scanned", totalScanned).
		Int("corrupted", len(corrupted)).
		Msg("scan-complete")

	for _, line := range corrupted {
		os.Stdout.WriteString(line + "\n")
	}
}
