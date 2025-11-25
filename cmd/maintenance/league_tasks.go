package main

import (
	"context"
	"encoding/json"
	"fmt"
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

	clock := league.NewClockFromEnv()
	lifecycleMgr := league.NewSeasonLifecycleManager(allStores, clock)

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	registrationsOpened := 0

	for _, dbLeague := range leagues {
		result, err := lifecycleMgr.OpenRegistrationForNextSeason(ctx, dbLeague.Uuid)
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

	clock := league.NewClockFromEnv()
	lifecycleMgr := league.NewSeasonLifecycleManager(allStores, clock)

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	seasonsClosed := 0

	for _, dbLeague := range leagues {
		result, err := lifecycleMgr.CloseCurrentSeason(ctx, dbLeague.Uuid)
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

	clock := league.NewClockFromEnv()
	lifecycleMgr := league.NewSeasonLifecycleManager(allStores, clock)

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

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

			result, err := lifecycleMgr.PrepareAndScheduleSeason(ctx, dbLeague.Uuid, season.Uuid)
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

	clock := league.NewClockFromEnv()
	lifecycleMgr := league.NewSeasonLifecycleManager(allStores, clock)

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

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
			result, err := lifecycleMgr.StartScheduledSeason(ctx, dbLeague.Uuid, season.Uuid)
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

// LeagueSeasonStartingSoon sends "season starting soon" notifications
// This runs on the last day of current season (1 day before next season starts)
func LeagueSeasonStartingSoon() error {
	log.Info().Msg("starting league season starting soon maintenance task")

	ctx := context.Background()
	cfg := &config.Config{}
	cfg.Load(nil)

	allStores, err := initLeagueStores(ctx, cfg)
	if err != nil {
		return err
	}

	clock := league.NewClockFromEnv()
	lifecycleMgr := league.NewSeasonLifecycleManager(allStores, clock)

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	now := clock.Now()
	notificationsSent := 0

	for _, dbLeague := range leagues {
		log.Info().Str("league", dbLeague.Name).Msg("Checking league...")

		// Parse league settings
		leagueSettings, err := parseLeagueSettings(dbLeague.Settings)
		if err != nil {
			log.Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("failed to parse league settings")
			continue
		}

		// Get current active season
		currentSeason, err := allStores.LeagueStore.GetCurrentSeason(ctx, dbLeague.Uuid)
		hasCurrentSeason := err == nil

		if hasCurrentSeason {
			shouldRun, reason := league.ShouldRunTask(&currentSeason, "season-starting-soon", leagueSettings.SeasonLengthDays, now)
			log.Info().
				Bool("shouldRun", shouldRun).
				Str("reason", reason).
				Str("seasonID", currentSeason.Uuid.String()).
				Msg("Season starting soon check")

			if !shouldRun {
				continue
			}

			// Find the next season (should be in SCHEDULED status)
			seasons, err := allStores.LeagueStore.GetSeasonsByLeague(ctx, dbLeague.Uuid)
			if err != nil {
				log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to get seasons for reminder")
				continue
			}

			// Find the next season (current season number + 1)
			nextSeasonNumber := currentSeason.SeasonNumber + 1

			// Debug: Log all seasons found
			log.Info().Str("leagueID", dbLeague.Uuid.String()).Int32("currentSeasonNumber", currentSeason.SeasonNumber).Int32("lookingForSeasonNumber", nextSeasonNumber).Msg("Searching for next season")
			for _, s := range seasons {
				log.Info().
					Str("seasonID", s.Uuid.String()).
					Int32("seasonNumber", s.SeasonNumber).
					Int32("status", s.Status).
					Str("statusName", pb.SeasonStatus(s.Status).String()).
					Msg("Found season")
			}

			foundNextSeason := false
			for _, season := range seasons {
				// Accept both REGISTRATION_OPEN and SCHEDULED status since this runs before season starts
				if season.SeasonNumber == nextSeasonNumber &&
					(season.Status == int32(pb.SeasonStatus_SEASON_REGISTRATION_OPEN) ||
						season.Status == int32(pb.SeasonStatus_SEASON_SCHEDULED)) {
					log.Info().Str("leagueID", dbLeague.Uuid.String()).Int32("nextSeason", season.SeasonNumber).Str("status", pb.SeasonStatus(season.Status).String()).Msg("Sending season starting soon notifications...")

					err := lifecycleMgr.SendSeasonStartingSoonNotification(ctx, cfg, dbLeague.Uuid, season.Uuid)
					if err != nil {
						log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to send season starting soon notifications")
					} else {
						log.Info().Str("leagueID", dbLeague.Uuid.String()).Int32("season", season.SeasonNumber).Msg("✓ Season starting soon notifications sent successfully")
						notificationsSent++
					}
					foundNextSeason = true
					break
				}
			}

			if !foundNextSeason {
				log.Warn().Str("leagueID", dbLeague.Uuid.String()).Int32("expectedSeasonNumber", nextSeasonNumber).Msg("No scheduled next season found for notifications")
			}
		} else {
			// No current season - check if we should send notification for Season 1
			log.Info().Str("leagueID", dbLeague.Uuid.String()).Msg("No current season, checking for Season 1 notification")

			seasons, err := allStores.LeagueStore.GetSeasonsByLeague(ctx, dbLeague.Uuid)
			if err != nil {
				log.Warn().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to get seasons for Season 1 reminder check")
				continue
			}

			// Look for Season 1 in SCHEDULED or REGISTRATION_OPEN status
			foundSeason1 := false
			for _, season := range seasons {
				if season.SeasonNumber == 1 && (season.Status == int32(pb.SeasonStatus_SEASON_SCHEDULED) || season.Status == int32(pb.SeasonStatus_SEASON_REGISTRATION_OPEN)) {
					// For Season 1, we need to check if we're 1 day before it starts
					// (not the last day OF Season 1, which is what season-starting-soon normally checks)
					startDate := season.StartDate.Time
					oneDayBeforeStart := startDate.Add(-24 * time.Hour)

					// Check if we're at or after 1 day before start (and before the actual start)
					shouldRun := now.After(oneDayBeforeStart) && now.Before(startDate)
					reason := ""
					if now.Before(oneDayBeforeStart) {
						reason = fmt.Sprintf("Current time %s is before 1 day before start %s",
							now.Format("2006-01-02 15:04:05 MST"),
							oneDayBeforeStart.Format("2006-01-02 15:04:05 MST"))
					} else if now.After(startDate) {
						reason = fmt.Sprintf("Current time %s is after season start %s (too late for reminder)",
							now.Format("2006-01-02 15:04:05 MST"),
							startDate.Format("2006-01-02 15:04:05 MST"))
					} else {
						reason = fmt.Sprintf("Current time is 1 day before Season 1 start (%s)",
							startDate.Format("2006-01-02 15:04:05 MST"))
					}

					log.Info().
						Bool("shouldRun", shouldRun).
						Str("reason", reason).
						Str("seasonID", season.Uuid.String()).
						Msg("Season 1 starting soon check")

					if shouldRun {
						log.Info().Str("leagueID", dbLeague.Uuid.String()).Msg("Sending Season 1 starting soon notifications...")

						err := lifecycleMgr.SendSeasonStartingSoonNotification(ctx, cfg, dbLeague.Uuid, season.Uuid)
						if err != nil {
							log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to send Season 1 starting soon notifications")
						} else {
							log.Info().Str("leagueID", dbLeague.Uuid.String()).Msg("✓ Season 1 starting soon notifications sent successfully")
							notificationsSent++
						}
					}
					foundSeason1 = true
					break
				}
			}

			if !foundSeason1 {
				log.Warn().Str("leagueID", dbLeague.Uuid.String()).Msg("No Season 1 found for notifications")
			}
		}
	}

	log.Info().Int("notificationsSent", notificationsSent).Msg("completed league season starting soon")
	return nil
}

// LeagueMidnightRunner runs at midnight daily and checks if it should:
// 1. Close the current season (Day 21)
// 2. Prepare divisions for the next season
func LeagueMidnightRunner() error {
	log.Info().Msg("starting league midnight runner maintenance task")

	ctx := context.Background()
	cfg := &config.Config{}
	cfg.Load(nil)

	allStores, err := initLeagueStores(ctx, cfg)
	if err != nil {
		return err
	}

	clock := league.NewClockFromEnv()
	lifecycleMgr := league.NewSeasonLifecycleManager(allStores, clock)

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	now := clock.Now()
	tasksRun := 0

	for _, dbLeague := range leagues {
		log.Info().Str("league", dbLeague.Name).Msg("Checking league...")

		// Parse league settings
		leagueSettings, err := parseLeagueSettings(dbLeague.Settings)
		if err != nil {
			log.Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("failed to parse league settings")
			continue
		}

		// Get current active season (if any)
		currentSeason, err := allStores.LeagueStore.GetCurrentSeason(ctx, dbLeague.Uuid)
		hasCurrentSeason := (err == nil)

		if !hasCurrentSeason {
			log.Info().Str("leagueID", dbLeague.Uuid.String()).Msg("No current season found - checking for REGISTRATION_OPEN seasons to prepare")
		}

		// PHASE 1: Close current season (only if we have one)
		if hasCurrentSeason {
			// Check if we should close the season
			shouldRun, reason := league.ShouldRunTask(&currentSeason, "close-season", leagueSettings.SeasonLengthDays, now)
			log.Info().
				Bool("shouldRun", shouldRun).
				Str("reason", reason).
				Str("seasonID", currentSeason.Uuid.String()).
				Msg("Close season check")

			if !shouldRun {
				continue
			}

			log.Info().Str("leagueID", dbLeague.Uuid.String()).Msg("Closing current season...")
			closeResult, err := lifecycleMgr.CloseCurrentSeason(ctx, dbLeague.Uuid)
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
		}

		// PHASE 2: Prepare divisions for next season (or first season if no current season)
		// This should only run if:
		// - We just closed the current season (hasCurrentSeason && ran PHASE 1), OR
		// - It's time to prepare Season 1 (no current season, midnight on the day Season 1 starts)

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

			// Check if it's time to prepare this season
			// For Season 1 (no current season), check if it's midnight on the day the season starts
			// For Season 2+ (has current season), we already checked timing in PHASE 1
			if !hasCurrentSeason {
				shouldRun, reason := league.ShouldRunTask(&season, "prepare-divisions", leagueSettings.SeasonLengthDays, now)
				log.Info().
					Bool("shouldRun", shouldRun).
					Str("reason", reason).
					Str("seasonID", season.Uuid.String()).
					Msg("Prepare divisions check (no current season)")

				if !shouldRun {
					continue
				}
			}

			log.Info().Str("leagueID", dbLeague.Uuid.String()).Msg("Preparing divisions for season...")
			foundRegOpenSeason = true
			prepareResult, err := lifecycleMgr.PrepareAndScheduleSeason(ctx, dbLeague.Uuid, season.Uuid)
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
func LeagueMorningRunner() error {
	log.Info().Msg("starting league morning runner maintenance task")

	ctx := context.Background()
	cfg := &config.Config{}
	cfg.Load(nil)

	allStores, err := initLeagueStores(ctx, cfg)
	if err != nil {
		return err
	}

	clock := league.NewClockFromEnv()
	lifecycleMgr := league.NewSeasonLifecycleManager(allStores, clock)

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	now := clock.Now()
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
			shouldRun, reason := league.ShouldRunTask(&currentSeason, "open-registration", leagueSettings.SeasonLengthDays, now)
			log.Info().
				Bool("shouldRun", shouldRun).
				Str("reason", reason).
				Str("seasonID", currentSeason.Uuid.String()).
				Msg("Open registration check")

			if shouldRun {
				log.Info().Str("leagueID", dbLeague.Uuid.String()).Msg("Opening registration for next season...")
				result, err := lifecycleMgr.OpenRegistrationForNextSeason(ctx, dbLeague.Uuid)
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

		// TASK 3: Check if we should send "season starting soon" reminder
		// This runs on the last day of current season (1 day before next season starts)
		currentSeason, err = allStores.LeagueStore.GetCurrentSeason(ctx, dbLeague.Uuid)
		hasCurrentSeason := err == nil

		if hasCurrentSeason {
			shouldRun, reason := league.ShouldRunTask(&currentSeason, "season-starting-soon", leagueSettings.SeasonLengthDays, now)
			log.Info().
				Bool("shouldRun", shouldRun).
				Str("reason", reason).
				Str("seasonID", currentSeason.Uuid.String()).
				Msg("Season starting soon check")

			if shouldRun {
				// Find the next season (should be in SCHEDULED status)
				seasonsForReminder, err := allStores.LeagueStore.GetSeasonsByLeague(ctx, dbLeague.Uuid)
				if err != nil {
					log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to get seasons for reminder")
				} else {
					// Find the next season (current season number + 1)
					nextSeasonNumber := currentSeason.SeasonNumber + 1
					for _, season := range seasonsForReminder {
						if season.SeasonNumber == nextSeasonNumber && (season.Status == int32(pb.SeasonStatus_SEASON_SCHEDULED) || season.Status == int32(pb.SeasonStatus_SEASON_REGISTRATION_OPEN)) {
							log.Info().Str("leagueID", dbLeague.Uuid.String()).Int32("nextSeason", season.SeasonNumber).Msg("Sending season starting soon notifications...")

							err := lifecycleMgr.SendSeasonStartingSoonNotification(ctx, cfg, dbLeague.Uuid, season.Uuid)
							if err != nil {
								log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to send season starting soon notifications")
							} else {
								log.Info().Str("leagueID", dbLeague.Uuid.String()).Int32("season", season.SeasonNumber).Msg("✓ Season starting soon notifications sent successfully")
								tasksRun++
							}
							break
						}
					}
				}
			}
		} else {
			// No current season - check if we should send notification for Season 1
			log.Info().Str("leagueID", dbLeague.Uuid.String()).Msg("No current season, checking for Season 1 notification")

			seasonsForReminder, err := allStores.LeagueStore.GetSeasonsByLeague(ctx, dbLeague.Uuid)
			if err != nil {
				log.Warn().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to get seasons for Season 1 reminder check")
			} else {
				// Look for Season 1 in SCHEDULED or REGISTRATION_OPEN status
				for _, season := range seasonsForReminder {
					if season.SeasonNumber == 1 && (season.Status == int32(pb.SeasonStatus_SEASON_SCHEDULED) || season.Status == int32(pb.SeasonStatus_SEASON_REGISTRATION_OPEN)) {
						// For Season 1, we need to check if we're 1 day before it starts
						// (not the last day OF Season 1, which is what season-starting-soon normally checks)
						startDate := season.StartDate.Time
						oneDayBeforeStart := startDate.Add(-24 * time.Hour)

						// Check if we're at or after 1 day before start (and before the actual start)
						shouldRun := now.After(oneDayBeforeStart) && now.Before(startDate)
						reason := ""
						if now.Before(oneDayBeforeStart) {
							reason = fmt.Sprintf("Current time %s is before 1 day before start %s",
								now.Format("2006-01-02 15:04:05 MST"),
								oneDayBeforeStart.Format("2006-01-02 15:04:05 MST"))
						} else if now.After(startDate) {
							reason = fmt.Sprintf("Current time %s is after season start %s (too late for reminder)",
								now.Format("2006-01-02 15:04:05 MST"),
								startDate.Format("2006-01-02 15:04:05 MST"))
						} else {
							reason = fmt.Sprintf("Current time is 1 day before Season 1 start (%s)",
								startDate.Format("2006-01-02 15:04:05 MST"))
						}

						log.Info().
							Bool("shouldRun", shouldRun).
							Str("reason", reason).
							Str("seasonID", season.Uuid.String()).
							Msg("Season 1 starting soon check")

						if shouldRun {
							log.Info().Str("leagueID", dbLeague.Uuid.String()).Msg("Sending Season 1 starting soon notifications...")

							err := lifecycleMgr.SendSeasonStartingSoonNotification(ctx, cfg, dbLeague.Uuid, season.Uuid)
							if err != nil {
								log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to send Season 1 starting soon notifications")
							} else {
								log.Info().Str("leagueID", dbLeague.Uuid.String()).Msg("✓ Season 1 starting soon notifications sent successfully")
								tasksRun++
							}
						}
						break
					}
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

			shouldRun, reason := league.ShouldRunTask(&season, "start-season", leagueSettings.SeasonLengthDays, now)
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
			result, err := lifecycleMgr.StartScheduledSeason(ctx, dbLeague.Uuid, season.Uuid)
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

			// Send season started notifications to all registered players
			err = lifecycleMgr.SendSeasonStartedNotification(ctx, cfg, dbLeague.Uuid, season.Uuid)
			if err != nil {
				log.Error().Err(err).Str("seasonID", season.Uuid.String()).Msg("Failed to send season started notifications")
			}

			tasksRun++
		}
	}

	log.Info().Int("tasksRun", tasksRun).Msg("completed league morning runner")
	return nil
}

// LeagueHourlyRunner runs every hour and checks all leagues for tasks that need to be run.
// It uses the actual season start/end dates and idempotency columns to determine what to run.
// This replaces the separate midnight and morning runners.
// Tasks are run in this order for each league:
// 1. Close current season (if end time has passed)
// 2. Prepare divisions for next season (after closing)
// 3. Open registration for next season (halfway through current season)
// 4. Send "season starting soon" notifications (24 hours before next season)
// 5. Start scheduled seasons (if start time has passed)
func LeagueHourlyRunner() error {
	log.Info().Msg("starting league hourly runner maintenance task")

	ctx := context.Background()
	cfg := &config.Config{}
	cfg.Load(nil)

	allStores, err := initLeagueStores(ctx, cfg)
	if err != nil {
		return err
	}

	clock := league.NewClockFromEnv()
	lifecycleMgr := league.NewSeasonLifecycleManager(allStores, clock)
	now := clock.Now()

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

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

	tasksRun := 0

	for _, dbLeague := range leagues {
		log.Info().Str("league", dbLeague.Name).Msg("Checking league...")

		// Parse league settings
		leagueSettings, err := parseLeagueSettings(dbLeague.Settings)
		if err != nil {
			log.Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("failed to parse league settings")
			continue
		}

		// Get all seasons for this league
		allSeasons, err := allStores.LeagueStore.GetSeasonsByLeague(ctx, dbLeague.Uuid)
		if err != nil {
			log.Warn().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("failed to get seasons")
			continue
		}

		// Get current active season (if any)
		currentSeason, err := allStores.LeagueStore.GetCurrentSeason(ctx, dbLeague.Uuid)
		hasCurrentSeason := (err == nil)

		// TASK 1: Close current season (if end time has passed and not already closed)
		if hasCurrentSeason && currentSeason.Status == int32(pb.SeasonStatus_SEASON_ACTIVE) {
			endTime := currentSeason.EndDate.Time

			// Check if end time has passed and season not already closed
			if now.After(endTime) || now.Equal(endTime) {
				// Check idempotency: if closed_at is set, skip
				if !currentSeason.ClosedAt.Valid {
					log.Info().
						Str("seasonID", currentSeason.Uuid.String()).
						Time("endTime", endTime).
						Msg("Closing current season (end time reached)...")

					closeResult, err := lifecycleMgr.CloseCurrentSeason(ctx, dbLeague.Uuid)
					if err != nil {
						log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to close season")
					} else if closeResult != nil {
						// Mark as closed for idempotency
						if markErr := allStores.LeagueStore.MarkSeasonClosed(ctx, currentSeason.Uuid); markErr != nil {
							log.Warn().Err(markErr).Str("seasonID", currentSeason.Uuid.String()).Msg("Failed to mark season as closed")
						}

						log.Info().
							Str("currentSeasonID", closeResult.CurrentSeasonID.String()).
							Int("forceFinished", closeResult.ForceFinishedGames).
							Msg("✓ Successfully closed season")
						tasksRun++

						// Refresh current season since it may have changed
						hasCurrentSeason = false
					}
				} else {
					log.Info().
						Str("seasonID", currentSeason.Uuid.String()).
						Msg("Season already closed (idempotency check)")
				}
			}
		}

		// TASK 2: Prepare divisions for REGISTRATION_OPEN season (before start time)
		for _, season := range allSeasons {
			if season.Status != int32(pb.SeasonStatus_SEASON_REGISTRATION_OPEN) {
				continue
			}

			startTime := season.StartDate.Time
			// Prepare divisions starting 8 hours before season start
			prepareTime := startTime.Add(-8 * time.Hour)

			if now.After(prepareTime) && now.Before(startTime) {
				// Check idempotency: if divisions_prepared_at is set, skip
				if !season.DivisionsPreparedAt.Valid {
					log.Info().
						Str("seasonID", season.Uuid.String()).
						Time("startTime", startTime).
						Msg("Preparing divisions for season...")

					prepareResult, err := lifecycleMgr.PrepareAndScheduleSeason(ctx, dbLeague.Uuid, season.Uuid)
					if err != nil {
						log.Error().Err(err).Str("seasonID", season.Uuid.String()).Msg("Failed to prepare season")
					} else if prepareResult != nil {
						// Mark as prepared for idempotency
						if markErr := allStores.LeagueStore.MarkDivisionsPrepared(ctx, season.Uuid); markErr != nil {
							log.Warn().Err(markErr).Str("seasonID", season.Uuid.String()).Msg("Failed to mark divisions as prepared")
						}

						log.Info().
							Str("seasonID", prepareResult.SeasonID.String()).
							Int32("seasonNumber", prepareResult.SeasonNumber).
							Int("totalRegistrations", prepareResult.DivisionPreparation.TotalRegistrations).
							Msg("✓ Successfully prepared season")
						tasksRun++
					}
				} else {
					log.Info().
						Str("seasonID", season.Uuid.String()).
						Msg("Divisions already prepared (idempotency check)")
				}
			}
		}

		// TASK 3: Open registration for next season (halfway through current season)
		// We need to refresh currentSeason in case it was modified
		if hasCurrentSeason {
			currentSeason, err = allStores.LeagueStore.GetCurrentSeason(ctx, dbLeague.Uuid)
			if err == nil && currentSeason.Status == int32(pb.SeasonStatus_SEASON_ACTIVE) {
				// Calculate registration open time: halfway through the season
				daysUntilOpen := (leagueSettings.SeasonLengthDays / 2) - 1
				registrationOpenTime := currentSeason.StartDate.Time.Add(time.Duration(daysUntilOpen) * 24 * time.Hour)

				if now.After(registrationOpenTime) {
					// Check idempotency: if registration_opened_at is set on current season, skip
					if !currentSeason.RegistrationOpenedAt.Valid {
						log.Info().
							Str("seasonID", currentSeason.Uuid.String()).
							Time("registrationOpenTime", registrationOpenTime).
							Msg("Opening registration for next season...")

						result, err := lifecycleMgr.OpenRegistrationForNextSeason(ctx, dbLeague.Uuid)
						if err != nil {
							log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to open registration")
						} else if result != nil {
							// Mark registration as opened on the CURRENT season (so we don't repeat)
							if markErr := allStores.LeagueStore.MarkRegistrationOpenedForNextSeason(ctx, currentSeason.Uuid); markErr != nil {
								log.Warn().Err(markErr).Str("seasonID", currentSeason.Uuid.String()).Msg("Failed to mark registration as opened")
							}

							log.Info().
								Str("nextSeasonID", result.NextSeasonID.String()).
								Int32("seasonNumber", result.NextSeasonNumber).
								Time("startDate", result.StartDate).
								Msg("✓ Registration opened successfully")
							tasksRun++
						}
					} else {
						log.Info().
							Str("seasonID", currentSeason.Uuid.String()).
							Msg("Registration already opened (idempotency check)")
					}
				}
			}
		}

		// TASK 4: Send "season starting soon" notifications (24 hours before start)
		for _, season := range allSeasons {
			if season.Status != int32(pb.SeasonStatus_SEASON_REGISTRATION_OPEN) &&
				season.Status != int32(pb.SeasonStatus_SEASON_SCHEDULED) {
				continue
			}

			startTime := season.StartDate.Time
			notifyTime := startTime.Add(-24 * time.Hour)

			if now.After(notifyTime) && now.Before(startTime) {
				// Check idempotency: if starting_soon_notification_sent_at is set, skip
				if !season.StartingSoonNotificationSentAt.Valid {
					log.Info().
						Str("seasonID", season.Uuid.String()).
						Time("startTime", startTime).
						Msg("Sending season starting soon notifications...")

					err := lifecycleMgr.SendSeasonStartingSoonNotification(ctx, cfg, dbLeague.Uuid, season.Uuid)
					if err != nil {
						log.Error().Err(err).Str("seasonID", season.Uuid.String()).Msg("Failed to send notifications")
					} else {
						// Mark as sent for idempotency
						if markErr := allStores.LeagueStore.MarkStartingSoonNotificationSent(ctx, season.Uuid); markErr != nil {
							log.Warn().Err(markErr).Str("seasonID", season.Uuid.String()).Msg("Failed to mark notification as sent")
						}

						log.Info().
							Str("seasonID", season.Uuid.String()).
							Msg("✓ Season starting soon notifications sent")
						tasksRun++
					}
				} else {
					log.Info().
						Str("seasonID", season.Uuid.String()).
						Msg("Starting soon notification already sent (idempotency check)")
				}
			}
		}

		// TASK 5: Start scheduled seasons (if start time has passed)
		for _, season := range allSeasons {
			if season.Status != int32(pb.SeasonStatus_SEASON_SCHEDULED) {
				continue
			}

			startTime := season.StartDate.Time

			if now.After(startTime) || now.Equal(startTime) {
				// Check idempotency: if started_at is set, skip
				if !season.StartedAt.Valid {
					log.Info().
						Str("seasonID", season.Uuid.String()).
						Time("startTime", startTime).
						Msg("Starting season (start time reached)...")

					// Step 1: Start the season (changes status to ACTIVE)
					result, err := lifecycleMgr.StartScheduledSeason(ctx, dbLeague.Uuid, season.Uuid)
					if err != nil {
						log.Error().Err(err).Str("seasonID", season.Uuid.String()).Msg("Failed to start season")
						continue
					}

					// Step 2: Create ALL games for the season with batching
					startMgr := league.NewSeasonStartManager(allStores.LeagueStore, allStores, cfg, gameCreator)
					gameResult, err := startMgr.CreateGamesForSeasonWithDelay(ctx, dbLeague.Uuid, season.Uuid, leagueSettings, 150*time.Millisecond, 10)
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
							Msg("failed to create games for season - rolled back to SCHEDULED")
						continue
					}

					// Mark as started for idempotency
					if markErr := allStores.LeagueStore.MarkSeasonStarted(ctx, season.Uuid); markErr != nil {
						log.Warn().Err(markErr).Str("seasonID", season.Uuid.String()).Msg("Failed to mark season as started")
					}

					log.Info().
						Str("seasonID", result.SeasonID.String()).
						Int("totalGames", gameResult.TotalGamesCreated).
						Msg("✓ Successfully started season and created games")

					// Send season started notifications
					err = lifecycleMgr.SendSeasonStartedNotification(ctx, cfg, dbLeague.Uuid, season.Uuid)
					if err != nil {
						log.Error().Err(err).Str("seasonID", season.Uuid.String()).Msg("Failed to send season started notifications")
					}

					tasksRun++
				} else {
					log.Info().
						Str("seasonID", season.Uuid.String()).
						Msg("Season already started (idempotency check)")
				}
			}
		}
	}

	log.Info().Int("tasksRun", tasksRun).Msg("completed league hourly runner")
	return nil
}

// LeagueUnstartedGameReminder runs daily and sends reminders to players who haven't started their games
// This checks seasons that are 16+ hours old and sends reminders at two intervals:
// - Day 1 (16 hours after start): Gentle reminder
// - Day 2 (40 hours after start): Firmer warning about potential suspension
func LeagueUnstartedGameReminder() error {
	log.Info().Msg("starting league unstarted game reminder maintenance task")

	ctx := context.Background()
	cfg := &config.Config{}
	cfg.Load(nil)

	allStores, err := initLeagueStores(ctx, cfg)
	if err != nil {
		return err
	}

	clock := league.NewClockFromEnv()
	lifecycleMgr := league.NewSeasonLifecycleManager(allStores, clock)

	// Get all active leagues
	leagues, err := allStores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	now := clock.Now()
	remindersSent := 0

	for _, dbLeague := range leagues {
		log.Info().Str("league", dbLeague.Name).Msg("Checking for unstarted games...")

		// Get current active season
		currentSeason, err := allStores.LeagueStore.GetCurrentSeason(ctx, dbLeague.Uuid)
		if err != nil {
			continue // No active season, skip
		}

		// Only process ACTIVE seasons
		if currentSeason.Status != int32(pb.SeasonStatus_SEASON_ACTIVE) {
			continue
		}

		startTime := currentSeason.StartDate.Time
		hoursSinceStart := now.Sub(startTime).Hours()

		// Day 1 reminder: 16 hours after season start
		if hoursSinceStart >= 16 && hoursSinceStart < 24 {
			log.Info().
				Str("seasonID", currentSeason.Uuid.String()).
				Float64("hoursSinceStart", hoursSinceStart).
				Msg("Sending Day 1 gentle reminder for unstarted games")

			err := lifecycleMgr.SendUnstartedGameReminder(ctx, cfg, dbLeague.Uuid, currentSeason.Uuid, false)
			if err != nil {
				log.Error().Err(err).Str("seasonID", currentSeason.Uuid.String()).Msg("Failed to send Day 1 reminder")
			} else {
				log.Info().Str("seasonID", currentSeason.Uuid.String()).Msg("✓ Day 1 reminders sent successfully")
				remindersSent++
			}
		}

		// Day 2 reminder: 40 hours after season start
		if hoursSinceStart >= 40 && hoursSinceStart < 48 {
			log.Info().
				Str("seasonID", currentSeason.Uuid.String()).
				Float64("hoursSinceStart", hoursSinceStart).
				Msg("Sending Day 2 firm reminder for unstarted games")

			err := lifecycleMgr.SendUnstartedGameReminder(ctx, cfg, dbLeague.Uuid, currentSeason.Uuid, true)
			if err != nil {
				log.Error().Err(err).Str("seasonID", currentSeason.Uuid.String()).Msg("Failed to send Day 2 reminder")
			} else {
				log.Info().Str("seasonID", currentSeason.Uuid.String()).Msg("✓ Day 2 reminders sent successfully")
				remindersSent++
			}
		}
	}

	log.Info().Int("remindersSent", remindersSent).Msg("completed league unstarted game reminder")
	return nil
}
