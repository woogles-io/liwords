package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"

	macondo "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Print games that would be updated without actually doing it")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	ctx := context.Background()

	cfg := &config.Config{}
	cfg.Load(nil)

	pool, err := pgxpool.New(ctx, cfg.DBConnDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	queries := models.New(pool)

	// Find all completed analysis jobs for league games.
	rows, err := pool.Query(ctx, `
		SELECT aj.id, aj.game_id, aj.result
		FROM analysis_jobs aj
		JOIN games g ON g.uuid = aj.game_id
		WHERE aj.status = 'completed'
		  AND aj.result IS NOT NULL
		  AND g.league_division_id IS NOT NULL
		ORDER BY aj.completed_at DESC
	`)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to query completed analysis jobs")
	}
	defer rows.Close()

	type jobRow struct {
		id     uuid.UUID
		gameID string
		result []byte
	}

	var jobs []jobRow
	for rows.Next() {
		var j jobRow
		if err := rows.Scan(&j.id, &j.gameID, &j.result); err != nil {
			log.Fatal().Err(err).Msg("failed to scan row")
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		log.Fatal().Err(err).Msg("error iterating rows")
	}

	log.Info().Int("count", len(jobs)).Bool("dry_run", *dryRun).Msg("completed analysis jobs for league games")

	if *dryRun {
		const preview = 10
		for i, j := range jobs {
			if i >= preview {
				fmt.Printf("... and %d more\n", len(jobs)-preview)
				break
			}
			fmt.Printf("job %s game %s\n", j.id, j.gameID)
		}
		return
	}

	updated := 0
	skipped := 0
	for _, j := range jobs {
		var result macondo.GameAnalysisResult
		if err := protojson.Unmarshal(j.result, &result); err != nil {
			log.Error().Err(err).Str("game_id", j.gameID).Msg("failed to unmarshal result, skipping")
			skipped++
			continue
		}

		if len(result.PlayerSummaries) != 2 {
			log.Warn().Str("game_id", j.gameID).Int("summaries", len(result.PlayerSummaries)).Msg("unexpected player summary count, skipping")
			skipped++
			continue
		}

		gameInfo, err := queries.GetGameLeagueInfo(ctx, pgtype.Text{String: j.gameID, Valid: true})
		if err != nil {
			log.Error().Err(err).Str("game_id", j.gameID).Msg("failed to get game league info, skipping")
			skipped++
			continue
		}
		if !gameInfo.LeagueDivisionID.Valid {
			skipped++
			continue
		}

		divisionID, err := uuid.FromBytes(gameInfo.LeagueDivisionID.Bytes[:])
		if err != nil {
			log.Error().Err(err).Str("game_id", j.gameID).Msg("failed to parse division UUID, skipping")
			skipped++
			continue
		}

		players := []struct {
			playerID     pgtype.Int4
			mistakeIndex float64
		}{
			{gameInfo.Player0ID, result.PlayerSummaries[0].GetMistakeIndex()},
			{gameInfo.Player1ID, result.PlayerSummaries[1].GetMistakeIndex()},
		}

		for _, p := range players {
			if !p.playerID.Valid {
				continue
			}
			if err := queries.IncrementStandingMistakeIndex(ctx, models.IncrementStandingMistakeIndexParams{
				DivisionID:        divisionID,
				UserID:            p.playerID.Int32,
				TotalMistakeIndex: pgtype.Float8{Float64: p.mistakeIndex, Valid: true},
			}); err != nil {
				log.Error().Err(err).
					Str("game_id", j.gameID).
					Int32("user_id", p.playerID.Int32).
					Msg("failed to update mistake index")
			}
		}
		updated++

		log.Debug().
			Str("game_id", j.gameID).
			Str("division_id", divisionID.String()).
			Float64("p0_mistake_index", result.PlayerSummaries[0].GetMistakeIndex()).
			Float64("p1_mistake_index", result.PlayerSummaries[1].GetMistakeIndex()).
			Msg("updated standings")
	}

	log.Info().Int("updated", updated).Int("skipped", skipped).Int("total", len(jobs)).Msg("done")
}
