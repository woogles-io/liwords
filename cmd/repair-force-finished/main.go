package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/league"
	"github.com/woogles-io/liwords/pkg/stores"
)

func main() {
	// Parse command line flags
	seasonIDStr := flag.String("season-id", "", "Season UUID to repair (required)")
	flag.Parse()

	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if *seasonIDStr == "" {
		fmt.Println("Error: --season-id is required")
		fmt.Println()
		printUsage()
		os.Exit(1)
	}

	// Parse season ID
	seasonID, err := uuid.Parse(*seasonIDStr)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid season ID")
	}

	ctx := context.Background()

	// Run repair
	err = runRepair(ctx, seasonID)
	if err != nil {
		log.Fatal().Err(err).Msg("repair failed")
	}

	log.Info().Msg("repair completed successfully")
}

func printUsage() {
	fmt.Println("Repair Force-Finished Games - Backfill missing game_players rows for adjudicated games")
	fmt.Println()
	fmt.Println("Usage: go run cmd/repair-force-finished --season-id <uuid>")
	fmt.Println()
	fmt.Println("This tool:")
	fmt.Println("  1. Finds force-finished or adjudicated games missing game_players rows")
	fmt.Println("  2. Inserts missing game_players entries for those games")
	fmt.Println("  3. Migrates old FORCE_FORFEIT (8) games to ADJUDICATED (9)")
	fmt.Println("  4. Recalculates league standings for the season")
	fmt.Println()
	fmt.Println("This tool is idempotent - safe to run multiple times.")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  go run cmd/repair-force-finished --season-id a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
}

func runRepair(ctx context.Context, seasonID uuid.UUID) error {
	log.Info().Str("seasonID", seasonID.String()).Msg("starting repair")

	// Initialize stores
	allStores, err := initStores(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize stores: %w", err)
	}

	// Create force finish manager
	ffm := league.NewForceFinishManager(allStores)

	// Run repair
	result, err := ffm.RepairForceFinishedGames(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("repair failed: %w", err)
	}

	// Print results
	log.Info().
		Int("total_games_found", result.TotalGamesFound).
		Int("games_repaired", result.GamesRepaired).
		Int("errors", len(result.Errors)).
		Msg("repair complete")

	if len(result.Errors) > 0 {
		log.Warn().Msg("errors encountered during repair:")
		for _, errMsg := range result.Errors {
			log.Warn().Msg(errMsg)
		}
	}

	if result.TotalGamesFound == 0 {
		log.Info().Msg("no broken games found - season is already in good state")
	} else if result.GamesRepaired == result.TotalGamesFound {
		log.Info().Msg("all broken games successfully repaired")
	} else {
		log.Warn().
			Int("failed", result.TotalGamesFound-result.GamesRepaired).
			Msg("some games could not be repaired - see errors above")
	}

	return nil
}

func initStores(ctx context.Context) (*stores.Stores, error) {
	cfg := &config.Config{}
	cfg.Load(nil)

	dbPool, err := pgxpool.New(ctx, cfg.DBConnDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize Redis pool (needed for some store operations)
	redisPool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(cfg.RedisURL)
		},
	}

	allStores, err := stores.NewInitializedStores(dbPool, redisPool, cfg)
	if err != nil {
		return nil, err
	}

	// Wire up the league standings updater to avoid circular dependencies
	allStores.SetLeagueStandingsUpdater(league.NewStandingsUpdaterImpl(allStores.LeagueStore))

	return allStores, nil
}
