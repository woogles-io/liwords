package league

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// A season start pauses for startPauseEvery after each startPauseGames games
// it creates. Each game's queries take about 20 ms, so the pause keeps the
// start from taking the database and NATS for the minute or two it runs:
// every game started sends events to both its players.
const (
	startPauseEvery = 50 * time.Millisecond
	startPauseGames = 10
)

// parseLeagueSettings parses the JSONB settings from the database
func parseLeagueSettings(settingsJSON []byte) (*pb.LeagueSettings, error) {
	var settings pb.LeagueSettings
	err := json.Unmarshal(settingsJSON, &settings)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// LifecycleRunnerResult contains information about tasks that were executed
type LifecycleRunnerResult struct {
	TasksRun                 int
	RegistrationClosed       bool
	DivisionsPrepared        bool
	RegistrationOpened       bool
	StartingSoonNotification bool
	SeasonStarted            bool
	NextSeasonCreated        bool
	GamesCreated             int
}

// RunLeagueLifecycleTasks executes all automated league lifecycle tasks for a single league
// This is called by:
// - the API's league_hourly job, pkg/maintenance (production)
// - cmd/league-tester TUI (testing)
func RunLeagueLifecycleTasks(
	ctx context.Context,
	cfg *config.Config,
	allStores *stores.Stores,
	gameCreator GameCreator,
	leagueUUID uuid.UUID,
	clock Clock,
) (*LifecycleRunnerResult, error) {

	result := &LifecycleRunnerResult{}
	now := clock.Now()

	// Get league
	dbLeague, err := allStores.LeagueStore.GetLeagueByUUID(ctx, leagueUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get league: %w", err)
	}

	// Parse league settings
	leagueSettings, err := parseLeagueSettings(dbLeague.Settings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse league settings: %w", err)
	}

	// Get all seasons for this league
	allSeasons, err := allStores.LeagueStore.GetSeasonsByLeague(ctx, dbLeague.Uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to get seasons: %w", err)
	}

	// Get current season
	hasCurrentSeason := dbLeague.CurrentSeasonID.Valid
	var currentSeason models.LeagueSeason
	if hasCurrentSeason {
		currentSeason, err = allStores.LeagueStore.GetCurrentSeason(ctx, dbLeague.Uuid)
		if err != nil {
			log.Warn().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to get current season")
			hasCurrentSeason = false
		}
	}

	lifecycleMgr := NewSeasonLifecycleManager(allStores, clock)

	// Seasons whose registration-open messages are still to be written.
	var pendingNotices []uuid.UUID
	for _, season := range allSeasons {
		if season.RegistrationNoticePending {
			pendingNotices = append(pendingNotices, season.Uuid)
		}
	}

	// TASK 1: Close current season (if end time has passed and not already closed)
	if hasCurrentSeason && currentSeason.Status == int32(pb.SeasonStatus_SEASON_ACTIVE) {
		endTime := currentSeason.EndDate.Time

		if now.After(endTime) || now.Equal(endTime) {
			// Check idempotency
			if !currentSeason.ClosedAt.Valid {
				log.Info().
					Str("seasonID", currentSeason.Uuid.String()).
					Time("endTime", endTime).
					Msg("Closing current season (end time reached)...")

				closeResult, err := lifecycleMgr.CloseCurrentSeason(ctx, dbLeague.Uuid)
				if err != nil {
					log.Error().Err(err).Str("seasonID", currentSeason.Uuid.String()).Msg("Failed to close season")
				} else if closeResult != nil {
					if markErr := allStores.LeagueStore.MarkSeasonClosed(ctx, closeResult.CurrentSeasonID); markErr != nil {
						log.Warn().Err(markErr).Str("seasonID", closeResult.CurrentSeasonID.String()).Msg("Failed to mark season as closed")
					}

					log.Info().
						Str("seasonID", closeResult.CurrentSeasonID.String()).
						Int("forceFinished", closeResult.ForceFinishedGames).
						Msg("✓ Successfully closed season")
					result.TasksRun++
					result.RegistrationClosed = true

					// Refresh current season state since it may have changed
					hasCurrentSeason = false
				}
			} else {
				log.Info().
					Str("seasonID", currentSeason.Uuid.String()).
					Msg("Season already closed (idempotency check)")
			}
		}
	}

	// TASK 2: Prepare divisions for REGISTRATION_OPEN season
	for _, season := range allSeasons {
		// Only prepare seasons that are still in REGISTRATION_OPEN status
		// SCHEDULED seasons have already been prepared (idempotency check below will catch this anyway)
		if season.Status != int32(pb.SeasonStatus_SEASON_REGISTRATION_OPEN) {
			continue
		}

		startTime := season.StartDate.Time

		// Determine if we should prepare based on season number
		shouldPrepare := false
		if season.SeasonNumber == 1 {
			// Season 1: prepare 4 hours before start (or anytime after that if we missed the window)
			// This handles the case where the hourly job didn't run in the ideal window
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

		// Check idempotency
		if !season.DivisionsPreparedAt.Valid {
			log.Info().
				Str("seasonID", season.Uuid.String()).
				Time("startTime", startTime).
				Msg("Preparing divisions for season...")

			prepareResult, err := lifecycleMgr.PrepareAndScheduleSeason(ctx, dbLeague.Uuid, season.Uuid)
			if err != nil {
				log.Error().Err(err).Str("seasonID", season.Uuid.String()).Msg("Failed to prepare season")
			} else if prepareResult != nil {
				if markErr := allStores.LeagueStore.MarkDivisionsPrepared(ctx, season.Uuid); markErr != nil {
					log.Warn().Err(markErr).Str("seasonID", season.Uuid.String()).Msg("Failed to mark divisions as prepared")
				}

				log.Info().
					Str("seasonID", prepareResult.SeasonID.String()).
					Int32("seasonNumber", prepareResult.SeasonNumber).
					Int("totalRegistrations", prepareResult.DivisionPreparation.TotalRegistrations).
					Msg("✓ Successfully prepared season")
				result.TasksRun++
				result.DivisionsPrepared = true
			}
		} else {
			log.Info().
				Str("seasonID", season.Uuid.String()).
				Msg("Divisions already prepared (idempotency check)")
		}
	}

	// TASK 3: Open registration for next season (halfway through current season)
	if hasCurrentSeason {
		// Re-fetch current season in case TASK 1 changed it
		currentSeason, err = allStores.LeagueStore.GetCurrentSeason(ctx, dbLeague.Uuid)
		if err == nil && currentSeason.Status == int32(pb.SeasonStatus_SEASON_ACTIVE) {
			// Calculate registration open time: halfway through the season
			daysUntilOpen := (leagueSettings.SeasonLengthDays / 2) - 1
			registrationOpenTime := currentSeason.StartDate.Time.Add(time.Duration(daysUntilOpen) * 24 * time.Hour)

			if now.After(registrationOpenTime) {
				openResult, err := lifecycleMgr.OpenRegistrationForNextSeason(ctx, dbLeague.Uuid)
				if err != nil {
					log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to open registration")
				} else if openResult != nil {
					log.Info().
						Str("nextSeasonID", openResult.NextSeasonID.String()).
						Int32("seasonNumber", openResult.NextSeasonNumber).
						Time("startDate", openResult.StartDate).
						Msg("✓ Registration opened successfully")
					pendingNotices = append(pendingNotices, openResult.NextSeasonID)
					result.TasksRun++
					result.RegistrationOpened = true
					result.NextSeasonCreated = true
				}
			}
		}
	}

	// Registration-open emails and Discord post, for the season just opened
	// and any whose run stopped before writing them.
	for _, seasonID := range pendingNotices {
		if _, err := lifecycleMgr.QueueRegistrationOpenNotices(ctx, dbLeague.Uuid, seasonID); err != nil {
			log.Error().Err(err).Str("seasonID", seasonID.String()).Msg("Failed to queue registration-open notices")
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
			// Check idempotency
			if !season.StartingSoonNotificationSentAt.Valid {
				log.Info().
					Str("seasonID", season.Uuid.String()).
					Time("startTime", startTime).
					Msg("Sending season starting soon notifications...")

				written, err := lifecycleMgr.QueueSeasonStartingSoonNotices(ctx, dbLeague.Uuid, season.Uuid)
				if err != nil {
					log.Error().Err(err).Str("seasonID", season.Uuid.String()).Msg("Failed to queue notifications")
				} else if written {
					log.Info().
						Str("seasonID", season.Uuid.String()).
						Msg("✓ Season starting soon notifications queued")
					result.TasksRun++
					result.StartingSoonNotification = true
				}
			} else {
				log.Info().
					Str("seasonID", season.Uuid.String()).
					Msg("Starting soon notification already sent (idempotency check)")
			}
		}
	}

	// TASK 5: Start scheduled seasons (if start time has passed), and create
	// their games. A season is set ACTIVE first; its games are then created
	// one pair at a time, and games_created_at is set once all exist, with
	// the season-started emails. A start cut off partway leaves the season
	// ACTIVE without games_created_at, and this carries on with the games
	// still missing.
	var toCreate []models.LeagueSeason
	for _, season := range allSeasons {
		switch {
		case season.Status == int32(pb.SeasonStatus_SEASON_ACTIVE) && !season.GamesCreatedAt.Valid:
			log.Info().Str("seasonID", season.Uuid.String()).Msg("Season start was cut off; creating the games still missing")
			toCreate = append(toCreate, season)
		case season.Status == int32(pb.SeasonStatus_SEASON_SCHEDULED) && !now.Before(season.StartDate.Time):
			log.Info().
				Str("seasonID", season.Uuid.String()).
				Time("startTime", season.StartDate.Time).
				Msg("Starting season (start time reached)...")
			// Changes the status to ACTIVE.
			if _, err := lifecycleMgr.StartScheduledSeason(ctx, dbLeague.Uuid, season.Uuid); err != nil {
				log.Error().Err(err).Str("seasonID", season.Uuid.String()).Msg("Failed to start season")
				continue
			}
			toCreate = append(toCreate, season)
		}
	}
	for _, season := range toCreate {
		startMgr := NewSeasonStartManager(allStores.LeagueStore, allStores, cfg, gameCreator)
		gameResult, err := startMgr.CreateGamesForSeason(ctx, dbLeague.Uuid, season.Uuid, leagueSettings, startPauseEvery, startPauseGames)
		if err != nil {
			// The season stays ACTIVE without games_created_at: the next run
			// creates the games still missing.
			log.Err(err).Str("seasonID", season.Uuid.String()).Msg("failed to create games for season; the next run carries on")
			continue
		}
		log.Info().
			Str("seasonID", season.Uuid.String()).
			Int("totalGames", gameResult.TotalGamesCreated).
			Msg("✓ Successfully started season and created games")

		// Marks every game created, with the season's emails and Discord post.
		if _, err := lifecycleMgr.QueueSeasonStartedNotices(ctx, dbLeague.Uuid, season.Uuid); err != nil {
			log.Error().Err(err).Str("seasonID", season.Uuid.String()).Msg("Failed to queue season started notifications")
		}

		result.TasksRun++
		result.SeasonStarted = true
		result.GamesCreated += gameResult.TotalGamesCreated
	}

	return result, nil
}
