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

	allStores, err := stores.NewInitializedStores(dbPool, redisPool, cfg)
	if err != nil {
		return nil, err
	}

	// Wire up the league standings updater to avoid circular dependencies
	allStores.SetLeagueStandingsUpdater(league.NewStandingsUpdaterImpl(allStores.LeagueStore))

	return allStores, nil
}

func parseLeagueSettings(settingsJSON []byte) (*pb.LeagueSettings, error) {
	var settings pb.LeagueSettings
	err := json.Unmarshal(settingsJSON, &settings)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// LeagueHourlyRunner runs every hour and checks all leagues for tasks that need to be run.
// It uses the actual season start/end dates and idempotency columns to determine what to run.
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

	totalTasksRun := 0

	for _, dbLeague := range leagues {
		log.Info().Str("league", dbLeague.Name).Msg("Checking league...")

		// Run all lifecycle tasks for this league using the shared function
		result, err := league.RunLeagueLifecycleTasks(ctx, cfg, allStores, gameCreator, dbLeague.Uuid, clock)
		if err != nil {
			log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to run lifecycle tasks")
			continue
		}

		totalTasksRun += result.TasksRun

		log.Info().
			Str("leagueID", dbLeague.Uuid.String()).
			Int("tasksRun", result.TasksRun).
			Bool("registrationClosed", result.RegistrationClosed).
			Bool("divisionsPrepared", result.DivisionsPrepared).
			Bool("registrationOpened", result.RegistrationOpened).
			Bool("startingSoonNotification", result.StartingSoonNotification).
			Bool("seasonStarted", result.SeasonStarted).
			Int("gamesCreated", result.GamesCreated).
			Msg("✓ League lifecycle tasks completed")
	}

	log.Info().Int("totalTasksRun", totalTasksRun).Msg("completed league hourly runner")
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

// ResendSeasonStartedEmails is a one-time fix to resend "season started" emails for seasons
// that started recently but failed to send emails (e.g., due to SES permission issues).
// This finds seasons that were started in the last N hours and resends their notifications.
func ResendSeasonStartedEmails() error {
	log.Info().Msg("starting resend season started emails maintenance task")

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

	emailsSent := 0
	cutoffTime := now.Add(-72 * time.Hour) // Look for seasons started in last 72 hours

	for _, dbLeague := range leagues {
		log.Info().Str("league", dbLeague.Name).Msg("Checking for recently started seasons...")

		// Get all seasons for this league
		allSeasons, err := allStores.LeagueStore.GetSeasonsByLeague(ctx, dbLeague.Uuid)
		if err != nil {
			log.Warn().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("failed to get seasons")
			continue
		}

		for _, season := range allSeasons {
			// Only process ACTIVE seasons that were started recently
			if season.Status != int32(pb.SeasonStatus_SEASON_ACTIVE) {
				continue
			}

			// Check if season was started in the last 72 hours
			if !season.StartedAt.Valid {
				continue
			}

			startedAt := season.StartedAt.Time
			if startedAt.Before(cutoffTime) {
				continue
			}

			log.Info().
				Str("seasonID", season.Uuid.String()).
				Time("startedAt", startedAt).
				Msg("Resending season started notification for recently started season...")

			// Send season started notifications
			err = lifecycleMgr.SendSeasonStartedNotification(ctx, cfg, dbLeague.Uuid, season.Uuid)
			if err != nil {
				log.Error().Err(err).Str("seasonID", season.Uuid.String()).Msg("Failed to send season started notifications")
			} else {
				log.Info().Str("seasonID", season.Uuid.String()).Msg("✓ Successfully resent season started notifications")
				emailsSent++
			}
		}
	}

	log.Info().Int("emailsSent", emailsSent).Msg("completed resending season started emails")
	return nil
}
