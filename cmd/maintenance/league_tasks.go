package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/league"
	"github.com/woogles-io/liwords/pkg/stores"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// GameplayAdapter adapts the gameplay package functions to the league.GameCreator interface
type GameplayAdapter struct {
	stores    *stores.Stores
	cfg       *config.Config
	eventChan chan<- *entity.EventWrapper
}

func (a *GameplayAdapter) InstantiateNewGame(ctx context.Context, users [2]*entity.User,
	req *pb.GameRequest, tdata *entity.TournamentData) (*entity.Game, error) {
	return gameplay.InstantiateNewGame(ctx, a.stores.GameStore, a.cfg, users, req, tdata)
}

func (a *GameplayAdapter) StartGame(ctx context.Context, game *entity.Game) error {
	return gameplay.StartGame(ctx, a.stores, a.eventChan, game)
}

// initLeagueStores initializes the necessary stores for league maintenance tasks
func initLeagueStores(ctx context.Context, cfg *config.Config) (*stores.Stores, error) {
	dbPool, err := pgxpool.New(ctx, cfg.DBConnDSN)
	if err != nil {
		return nil, err
	}

	redisPool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(cfg.RedisURL)
		},
	}

	return stores.NewInitializedStores(dbPool, redisPool, cfg)
}

// LeagueRegistrationOpener opens registration for the next season on Day 15
// This creates a new season with status REGISTRATION_OPEN
func LeagueRegistrationOpener() error {
	log.Info().Msg("starting league registration opener maintenance task")

	ctx := context.Background()
	cfg := &config.Config{}
	cfg.Load(nil)

	allStores, err := initLeagueStores(ctx, cfg)
	if err != nil {
		return err
	}

	lifecycleMgr := league.NewSeasonLifecycleManager(allStores)

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	now := time.Now()
	registrationsOpened := 0

	for _, dbLeague := range leagues {
		result, err := lifecycleMgr.OpenRegistrationForNextSeason(ctx, dbLeague.Uuid, now)
		if err != nil {
			log.Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("failed to open registration")
			continue
		}

		if result != nil {
			log.Info().
				Str("leagueID", result.LeagueID.String()).
				Str("seasonID", result.NextSeasonID.String()).
				Int32("seasonNumber", result.NextSeasonNumber).
				Time("startDate", result.StartDate).
				Msg("successfully opened registration for next season")
			registrationsOpened++
		}
	}

	log.Info().Int("registrationsOpened", registrationsOpened).Msg("completed league registration opener")
	return nil
}

// LeagueSeasonCloser closes the current season on Day 20 at midnight
// This force-finishes unfinished games, marks season outcomes, and prepares next season divisions
func LeagueSeasonCloser() error {
	log.Info().Msg("starting league season closer maintenance task")

	ctx := context.Background()
	cfg := &config.Config{}
	cfg.Load(nil)

	allStores, err := initLeagueStores(ctx, cfg)
	if err != nil {
		return err
	}

	lifecycleMgr := league.NewSeasonLifecycleManager(allStores)

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	now := time.Now()
	seasonsClosed := 0

	for _, dbLeague := range leagues {
		result, err := lifecycleMgr.CloseCurrentSeason(ctx, dbLeague.Uuid, now)
		if err != nil {
			log.Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("failed to close season")
			continue
		}

		if result != nil {
			log.Info().
				Str("currentSeasonID", result.CurrentSeasonID.String()).
				Str("leagueID", result.LeagueID.String()).
				Int("forceFinished", result.ForceFinishedGames).
				Msg("successfully closed season")
			seasonsClosed++
		}
	}

	log.Info().Int("seasonsClosed", seasonsClosed).Msg("completed league season closer")
	return nil
}

// LeagueDivisionPreparer prepares divisions for seasons that are in REGISTRATION_OPEN status
// This should run on Day 21 at 7:45 AM (15 minutes before season start at 8:00 AM)
// It closes registration, creates divisions, and transitions seasons to SCHEDULED
func LeagueDivisionPreparer() error {
	log.Info().Msg("starting league division preparer maintenance task")

	ctx := context.Background()
	cfg := &config.Config{}
	cfg.Load(nil)

	allStores, err := initLeagueStores(ctx, cfg)
	if err != nil {
		return err
	}

	lifecycleMgr := league.NewSeasonLifecycleManager(allStores)

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	now := time.Now()
	seasonsPrepared := 0

	for _, dbLeague := range leagues {
		// Get all seasons for this league
		seasons, err := allStores.LeagueStore.GetSeasonsByLeague(ctx, dbLeague.Uuid)
		if err != nil {
			log.Warn().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("failed to get seasons")
			continue
		}

		for _, season := range seasons {
			// Only process seasons in REGISTRATION_OPEN or SCHEDULED status
			if season.Status != int32(pb.SeasonStatus_SEASON_REGISTRATION_OPEN) &&
				season.Status != int32(pb.SeasonStatus_SEASON_SCHEDULED) {
				continue // Skip ACTIVE/COMPLETED seasons
			}

			result, err := lifecycleMgr.PrepareAndScheduleSeason(ctx, dbLeague.Uuid, season.Uuid, now)
			if err != nil {
				log.Err(err).
					Str("seasonID", season.Uuid.String()).
					Str("leagueID", dbLeague.Uuid.String()).
					Msg("failed to prepare and schedule season")
				continue
			}

			if result != nil {
				log.Info().
					Str("leagueID", result.LeagueID.String()).
					Str("seasonID", result.SeasonID.String()).
					Int32("seasonNumber", result.SeasonNumber).
					Int("totalRegistrations", result.DivisionPreparation.TotalRegistrations).
					Int("regularDivisions", result.DivisionPreparation.RegularDivisionsUsed).
					Msg("successfully prepared and scheduled season")
				seasonsPrepared++
			}
		}
	}

	log.Info().Int("seasonsPrepared", seasonsPrepared).Msg("completed league division preparer")
	return nil
}

// LeagueSeasonStarter starts seasons that are SCHEDULED
// This task should be scheduled by the periodic task system (e.g., Day 21 at 8 AM)
// It also creates ALL games for the season upfront using round-robin pairing
func LeagueSeasonStarter() error {
	log.Info().Msg("starting league season starter maintenance task")

	ctx := context.Background()
	cfg := &config.Config{}
	cfg.Load(nil)

	allStores, err := initLeagueStores(ctx, cfg)
	if err != nil {
		return err
	}

	lifecycleMgr := league.NewSeasonLifecycleManager(allStores)

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	now := time.Now()
	seasonsStarted := 0

	// Create event channel for game events (will be nil for maintenance, but required)
	eventChan := make(chan *entity.EventWrapper, 100)
	defer close(eventChan)

	// Drain events in background
	go func() {
		for range eventChan {
			// Discard events in maintenance context
		}
	}()

	gameCreator := &GameplayAdapter{
		stores:    allStores,
		cfg:       cfg,
		eventChan: eventChan,
	}

	for _, dbLeague := range leagues {
		// Parse league settings to get game configuration
		leagueSettings, err := parseLeagueSettings(dbLeague.Settings)
		if err != nil {
			log.Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("failed to parse league settings")
			continue
		}

		// Get all seasons for this league
		seasons, err := allStores.LeagueStore.GetSeasonsByLeague(ctx, dbLeague.Uuid)
		if err != nil {
			log.Warn().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("failed to get seasons")
			continue
		}

		for _, season := range seasons {
			// Only process seasons in SCHEDULED status
			if season.Status != int32(pb.SeasonStatus_SEASON_SCHEDULED) {
				continue // Skip non-SCHEDULED seasons silently
			}

			// Step 1: Start the season (changes status to ACTIVE)
			result, err := lifecycleMgr.StartScheduledSeason(ctx, dbLeague.Uuid, season.Uuid, now)
			if err != nil {
				log.Info().
					Str("seasonID", season.Uuid.String()).
					Str("leagueID", dbLeague.Uuid.String()).
					Err(err).
					Msg("season not ready to start (skipping)")
				continue
			}

			// Step 2: Create ALL games for the season using SeasonStartManager
			startMgr := league.NewSeasonStartManager(allStores.LeagueStore, allStores, cfg, gameCreator)
			gameResult, err := startMgr.CreateGamesForSeason(ctx, dbLeague.Uuid, season.Uuid, leagueSettings)
			if err != nil {
				// Roll back the season status to SCHEDULED since game creation failed
				rollbackErr := lifecycleMgr.RollbackSeasonToScheduled(ctx, season.Uuid)
				if rollbackErr != nil {
					log.Err(rollbackErr).
						Str("seasonID", season.Uuid.String()).
						Msg("failed to rollback season status after game creation failure")
				}
				log.Err(err).
					Str("seasonID", season.Uuid.String()).
					Str("leagueID", dbLeague.Uuid.String()).
					Msg("failed to create games for season - rolled back season to SCHEDULED")
				continue
			}

			log.Info().
				Str("leagueID", result.LeagueID.String()).
				Str("seasonID", result.SeasonID.String()).
				Str("leagueName", result.LeagueName).
				Int("totalGames", gameResult.TotalGamesCreated).
				Interface("gamesPerDivision", gameResult.GamesPerDivision).
				Msg("successfully started league season and created all games")

			seasonsStarted++
		}
	}

	log.Info().Int("seasonsStarted", seasonsStarted).Msg("completed league season starter")
	return nil
}

// parseLeagueSettings parses the JSONB settings from the database
func parseLeagueSettings(settingsJSON []byte) (*pb.LeagueSettings, error) {
	var settings pb.LeagueSettings
	err := json.Unmarshal(settingsJSON, &settings)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// LeagueMidnightRunner runs at midnight daily and checks if it should:
// 1. Close the current season (Day 21)
// 2. Prepare divisions for the next season
func LeagueMidnightRunner(forceRun bool) error {
	log.Info().Bool("force", forceRun).Msg("starting league midnight runner maintenance task")

	if forceRun {
		log.Warn().Msg("⚠️  FORCE MODE ENABLED - Skipping time checks")
	}

	ctx := context.Background()
	cfg := &config.Config{}
	cfg.Load(nil)

	allStores, err := initLeagueStores(ctx, cfg)
	if err != nil {
		return err
	}

	lifecycleMgr := league.NewSeasonLifecycleManager(allStores)

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	now := time.Now()
	tasksRun := 0

	for _, dbLeague := range leagues {
		log.Info().Str("league", dbLeague.Name).Msg("Checking league...")

		// Get current active season
		currentSeason, err := allStores.LeagueStore.GetCurrentSeason(ctx, dbLeague.Uuid)
		if err != nil {
			log.Warn().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("No current season, skipping")
			continue
		}

		// Check if we should close the season
		shouldRun, reason := league.ShouldRunTask(&currentSeason, "close-season", forceRun, now)
		log.Info().
			Bool("shouldRun", shouldRun).
			Str("reason", reason).
			Str("seasonID", currentSeason.Uuid.String()).
			Msg("Close season check")

		if !shouldRun {
			continue
		}

		// PHASE 1: Close current season
		log.Info().Str("leagueID", dbLeague.Uuid.String()).Msg("Closing current season...")
		closeResult, err := lifecycleMgr.CloseCurrentSeason(ctx, dbLeague.Uuid, now)
		if err != nil {
			log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to close season")
			continue // Don't proceed to phase 2 if phase 1 fails
		}

		if closeResult != nil {
			log.Info().
				Str("currentSeasonID", closeResult.CurrentSeasonID.String()).
				Str("leagueID", closeResult.LeagueID.String()).
				Int("forceFinished", closeResult.ForceFinishedGames).
				Msg("✓ Successfully closed season")
		}

		// PHASE 2: Prepare divisions for next season
		log.Info().Str("leagueID", dbLeague.Uuid.String()).Msg("Preparing divisions for next season...")

		// Find and process REGISTRATION_OPEN season
		allSeasons, err := allStores.LeagueStore.GetSeasonsByLeague(ctx, dbLeague.Uuid)
		if err != nil {
			log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to get seasons")
			continue
		}

		foundRegOpenSeason := false
		for _, season := range allSeasons {
			if season.Status != int32(pb.SeasonStatus_SEASON_REGISTRATION_OPEN) {
				continue
			}

			foundRegOpenSeason = true
			prepareResult, err := lifecycleMgr.PrepareAndScheduleSeason(ctx, dbLeague.Uuid, season.Uuid, now)
			if err != nil {
				log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to prepare season")
				continue
			}

			if prepareResult != nil {
				log.Info().
					Str("leagueID", prepareResult.LeagueID.String()).
					Str("seasonID", prepareResult.SeasonID.String()).
					Int32("seasonNumber", prepareResult.SeasonNumber).
					Int("totalRegistrations", prepareResult.DivisionPreparation.TotalRegistrations).
					Int("regularDivisions", prepareResult.DivisionPreparation.RegularDivisionsUsed).
					Msg("✓ Successfully prepared season")
			}
			break // Only process first REGISTRATION_OPEN season
		}

		if !foundRegOpenSeason {
			log.Warn().Str("leagueID", dbLeague.Uuid.String()).Msg("No REGISTRATION_OPEN season found to prepare")
			continue
		}

		log.Info().Str("league", dbLeague.Name).Msg("✓ Midnight tasks completed successfully")
		tasksRun++
	}

	log.Info().Int("tasksRun", tasksRun).Msg("completed league midnight runner")
	return nil
}

// LeagueMorningRunner runs at 8am daily and checks if it should:
// 1. Open registration for next season (Day 14)
// 2. Start any scheduled seasons (Day 1 of new season)
func LeagueMorningRunner(forceRun bool) error {
	log.Info().Bool("force", forceRun).Msg("starting league morning runner maintenance task")

	if forceRun {
		log.Warn().Msg("⚠️  FORCE MODE ENABLED - Skipping time checks")
	}

	ctx := context.Background()
	cfg := &config.Config{}
	cfg.Load(nil)

	allStores, err := initLeagueStores(ctx, cfg)
	if err != nil {
		return err
	}

	lifecycleMgr := league.NewSeasonLifecycleManager(allStores)

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	now := time.Now()
	tasksRun := 0

	// Create event channel for game events (needed for season starter)
	eventChan := make(chan *entity.EventWrapper, 100)
	defer close(eventChan)

	// Drain events in background
	go func() {
		for range eventChan {
			// Discard events in maintenance context
		}
	}()

	gameCreator := &GameplayAdapter{
		stores:    allStores,
		cfg:       cfg,
		eventChan: eventChan,
	}

	for _, dbLeague := range leagues {
		log.Info().Str("league", dbLeague.Name).Msg("Checking league...")

		// Parse league settings (needed for game creation)
		leagueSettings, err := parseLeagueSettings(dbLeague.Settings)
		if err != nil {
			log.Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("failed to parse league settings")
			continue
		}

		// TASK 1: Check if we should open registration
		currentSeason, err := allStores.LeagueStore.GetCurrentSeason(ctx, dbLeague.Uuid)
		if err == nil {
			shouldRun, reason := league.ShouldRunTask(&currentSeason, "open-registration", forceRun, now)
			log.Info().
				Bool("shouldRun", shouldRun).
				Str("reason", reason).
				Str("seasonID", currentSeason.Uuid.String()).
				Msg("Open registration check")

			if shouldRun {
				log.Info().Str("leagueID", dbLeague.Uuid.String()).Msg("Opening registration for next season...")
				result, err := lifecycleMgr.OpenRegistrationForNextSeason(ctx, dbLeague.Uuid, now)
				if err != nil {
					log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to open registration")
					// Continue to check for season start anyway
				} else if result != nil {
					log.Info().
						Str("leagueID", result.LeagueID.String()).
						Str("seasonID", result.NextSeasonID.String()).
						Int32("seasonNumber", result.NextSeasonNumber).
						Time("startDate", result.StartDate).
						Msg("✓ Registration opened successfully")
					tasksRun++
				}
			}
		}

		// TASK 2: Check if we should start any scheduled seasons
		allSeasons, err := allStores.LeagueStore.GetSeasonsByLeague(ctx, dbLeague.Uuid)
		if err != nil {
			log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to get seasons")
			continue
		}

		for _, season := range allSeasons {
			if season.Status != int32(pb.SeasonStatus_SEASON_SCHEDULED) {
				continue
			}

			shouldRun, reason := league.ShouldRunTask(&season, "start-season", forceRun, now)
			log.Info().
				Bool("shouldRun", shouldRun).
				Str("reason", reason).
				Str("seasonID", season.Uuid.String()).
				Msg("Start season check")

			if !shouldRun {
				continue
			}

			log.Info().Str("seasonID", season.Uuid.String()).Msg("Starting season...")

			// Step 1: Start the season (changes status to ACTIVE)
			result, err := lifecycleMgr.StartScheduledSeason(ctx, dbLeague.Uuid, season.Uuid, now)
			if err != nil {
				log.Error().Err(err).Str("seasonID", season.Uuid.String()).Msg("Failed to start season")
				continue
			}

			// Step 2: Create ALL games for the season using SeasonStartManager
			startMgr := league.NewSeasonStartManager(allStores.LeagueStore, allStores, cfg, gameCreator)
			gameResult, err := startMgr.CreateGamesForSeason(ctx, dbLeague.Uuid, season.Uuid, leagueSettings)
			if err != nil {
				// Roll back the season status to SCHEDULED since game creation failed
				rollbackErr := lifecycleMgr.RollbackSeasonToScheduled(ctx, season.Uuid)
				if rollbackErr != nil {
					log.Err(rollbackErr).
						Str("seasonID", season.Uuid.String()).
						Msg("failed to rollback season status after game creation failure")
				}
				log.Err(err).
					Str("seasonID", season.Uuid.String()).
					Str("leagueID", dbLeague.Uuid.String()).
					Msg("failed to create games for season - rolled back season to SCHEDULED")
				continue
			}

			log.Info().
				Str("leagueID", result.LeagueID.String()).
				Str("seasonID", result.SeasonID.String()).
				Str("leagueName", result.LeagueName).
				Int("totalGames", gameResult.TotalGamesCreated).
				Interface("gamesPerDivision", gameResult.GamesPerDivision).
				Msg("✓ Successfully started league season and created all games")

			tasksRun++
		}
	}

	log.Info().Int("tasksRun", tasksRun).Msg("completed league morning runner")
	return nil
}

