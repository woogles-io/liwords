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

		// TASK 2: Prepare divisions for REGISTRATION_OPEN or SCHEDULED season
		// Triggers:
		// - Season 1: 4 hours before start time (no previous season to wait for)
		// - Season 2+: when previous season is COMPLETED (and before start time)
		for _, season := range allSeasons {
			if season.Status != int32(pb.SeasonStatus_SEASON_REGISTRATION_OPEN) &&
				season.Status != int32(pb.SeasonStatus_SEASON_SCHEDULED) {
				continue
			}

			startTime := season.StartDate.Time

			// Don't prepare if season has already started
			if now.After(startTime) || now.Equal(startTime) {
				continue
			}

			// Determine if we should prepare based on season number
			shouldPrepare := false
			if season.SeasonNumber == 1 {
				// Season 1: prepare 4 hours before start
				prepareTime := startTime.Add(-4 * time.Hour)
				shouldPrepare = now.After(prepareTime) || now.Equal(prepareTime)
			} else {
				// Season 2+: prepare when previous season is COMPLETED
				prevSeason, err := allStores.LeagueStore.GetSeasonByLeagueAndNumber(ctx, dbLeague.Uuid, season.SeasonNumber-1)
				if err != nil {
					log.Warn().Err(err).
						Str("seasonID", season.Uuid.String()).
						Int32("prevSeasonNumber", season.SeasonNumber-1).
						Msg("Failed to get previous season for preparation check")
					continue
				}
				shouldPrepare = prevSeason.Status == int32(pb.SeasonStatus_SEASON_COMPLETED)
			}

			if !shouldPrepare {
				continue
			}

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

		// TASK 3: Open registration for next season (halfway through current season)
		// Idempotency is handled by OpenRegistrationForNextSeason: returns nil if next season already exists
		if hasCurrentSeason {
			currentSeason, err = allStores.LeagueStore.GetCurrentSeason(ctx, dbLeague.Uuid)
			if err == nil && currentSeason.Status == int32(pb.SeasonStatus_SEASON_ACTIVE) {
				// Calculate registration open time: halfway through the season
				daysUntilOpen := (leagueSettings.SeasonLengthDays / 2) - 1
				registrationOpenTime := currentSeason.StartDate.Time.Add(time.Duration(daysUntilOpen) * 24 * time.Hour)

				if now.After(registrationOpenTime) {
					result, err := lifecycleMgr.OpenRegistrationForNextSeason(ctx, dbLeague.Uuid)
					if err != nil {
						log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to open registration")
					} else if result != nil {
						// Record when registration was opened on the new season
						if markErr := allStores.LeagueStore.MarkRegistrationOpened(ctx, result.NextSeasonID); markErr != nil {
							log.Warn().Err(markErr).Str("seasonID", result.NextSeasonID.String()).Msg("Failed to mark registration opened timestamp")
						}

						log.Info().
							Str("nextSeasonID", result.NextSeasonID.String()).
							Int32("seasonNumber", result.NextSeasonNumber).
							Time("startDate", result.StartDate).
							Msg("✓ Registration opened successfully")
						tasksRun++
					}
					// result == nil means next season already exists (idempotent)
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
					gameResult, err := startMgr.CreateGamesForSeason(ctx, dbLeague.Uuid, season.Uuid, leagueSettings, 150*time.Millisecond, 10)
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
