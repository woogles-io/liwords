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
	TasksRun                  int
	RegistrationClosed        bool
	DivisionsPrepared         bool
	RegistrationOpened        bool
	StartingSoonNotification  bool
	SeasonStarted             bool
	NextSeasonCreated         bool
	GamesCreated              int
}

// RunLeagueLifecycleTasks executes all automated league lifecycle tasks for a single league
// This is called by:
// - cmd/maintenance hourly cron job (production)
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

	// TASK 1: Close registration when end date is reached
	if hasCurrentSeason && currentSeason.Status == int32(pb.SeasonStatus_SEASON_REGISTRATION_OPEN) {
		endTime := currentSeason.EndDate.Time

		if now.After(endTime) || now.Equal(endTime) {
			// Check idempotency
			if !currentSeason.ClosedAt.Valid {
				log.Info().
					Str("seasonID", currentSeason.Uuid.String()).
					Time("endTime", endTime).
					Msg("Closing registration (end time reached)...")

				closeResult, err := lifecycleMgr.CloseCurrentSeason(ctx, dbLeague.Uuid)
				if err != nil {
					log.Error().Err(err).Str("seasonID", currentSeason.Uuid.String()).Msg("Failed to close registration")
				} else if closeResult != nil {
					if markErr := allStores.LeagueStore.MarkSeasonClosed(ctx, closeResult.CurrentSeasonID); markErr != nil {
						log.Warn().Err(markErr).Str("seasonID", closeResult.CurrentSeasonID.String()).Msg("Failed to mark season as closed")
					}

					log.Info().
						Str("seasonID", closeResult.CurrentSeasonID.String()).
						Int("totalRegistrations", closeResult.ForceFinishedGames).
						Msg("✓ Registration closed successfully")
					result.TasksRun++
					result.RegistrationClosed = true
				}
			} else {
				log.Info().
					Str("seasonID", currentSeason.Uuid.String()).
					Msg("Registration already closed (idempotency check)")
			}
		}
	}

	// TASK 2: Prepare divisions for REGISTRATION_OPEN or SCHEDULED season
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
	if hasCurrentSeason && currentSeason.Status == int32(pb.SeasonStatus_SEASON_ACTIVE) {
		// Calculate registration open time: halfway through the season
		daysUntilOpen := (leagueSettings.SeasonLengthDays / 2) - 1
		registrationOpenTime := currentSeason.StartDate.Time.Add(time.Duration(daysUntilOpen) * 24 * time.Hour)

		if now.After(registrationOpenTime) {
			openResult, err := lifecycleMgr.OpenRegistrationForNextSeason(ctx, dbLeague.Uuid)
			if err != nil {
				log.Error().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("Failed to open registration")
			} else if openResult != nil {
				if markErr := allStores.LeagueStore.MarkRegistrationOpened(ctx, openResult.NextSeasonID); markErr != nil {
					log.Warn().Err(markErr).Str("seasonID", openResult.NextSeasonID.String()).Msg("Failed to mark registration opened timestamp")
				}

				log.Info().
					Str("nextSeasonID", openResult.NextSeasonID.String()).
					Int32("seasonNumber", openResult.NextSeasonNumber).
					Time("startDate", openResult.StartDate).
					Msg("✓ Registration opened successfully")

				// Send registration open notifications (email + Discord)
				go func(seasonID uuid.UUID, seasonNumber int32) {
					// Get current season registrants
					currentRegistrants, err := allStores.LeagueStore.GetSeasonRegistrations(ctx, seasonID)
					if err != nil {
						log.Error().Err(err).Str("seasonID", seasonID.String()).Msg("Failed to get current registrants")
						return
					}

					// Get previous season registrants (not in current)
					previousRegistrants, err := allStores.LeagueStore.GetPreviousSeasonRegistrantsNotInCurrent(ctx, models.GetPreviousSeasonRegistrantsNotInCurrentParams{
						LeagueID:     dbLeague.Uuid,
						SeasonNumber: seasonNumber - 1,
					})
					if err != nil {
						log.Warn().Err(err).Msg("Failed to get previous season registrants (continuing anyway)")
					}

					// Send bulk email
					SendRegistrationOpenEmail(ctx, cfg, allStores.UserStore, dbLeague.Name, dbLeague.Slug, int(seasonNumber), currentRegistrants, previousRegistrants)

					// Send Discord notification
					SendRegistrationOpenDiscord(cfg, dbLeague.Name, dbLeague.Slug, int(seasonNumber))
				}(openResult.NextSeasonID, openResult.NextSeasonNumber)

				result.TasksRun++
				result.RegistrationOpened = true
				result.NextSeasonCreated = true
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
			// Check idempotency
			if !season.StartingSoonNotificationSentAt.Valid {
				log.Info().
					Str("seasonID", season.Uuid.String()).
					Time("startTime", startTime).
					Msg("Sending season starting soon notifications...")

				err := lifecycleMgr.SendSeasonStartingSoonNotification(ctx, cfg, dbLeague.Uuid, season.Uuid)
				if err != nil {
					log.Error().Err(err).Str("seasonID", season.Uuid.String()).Msg("Failed to send notifications")
				} else {
					if markErr := allStores.LeagueStore.MarkStartingSoonNotificationSent(ctx, season.Uuid); markErr != nil {
						log.Warn().Err(markErr).Str("seasonID", season.Uuid.String()).Msg("Failed to mark notification as sent")
					}

					log.Info().
						Str("seasonID", season.Uuid.String()).
						Msg("✓ Season starting soon notifications sent")
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

	// TASK 5: Start scheduled seasons (if start time has passed)
	for _, season := range allSeasons {
		if season.Status != int32(pb.SeasonStatus_SEASON_SCHEDULED) {
			continue
		}

		startTime := season.StartDate.Time

		if now.After(startTime) || now.Equal(startTime) {
			// Check idempotency
			if !season.StartedAt.Valid {
				log.Info().
					Str("seasonID", season.Uuid.String()).
					Time("startTime", startTime).
					Msg("Starting season (start time reached)...")

				// Step 1: Start the season (changes status to ACTIVE)
				startResult, err := lifecycleMgr.StartScheduledSeason(ctx, dbLeague.Uuid, season.Uuid)
				if err != nil {
					log.Error().Err(err).Str("seasonID", season.Uuid.String()).Msg("Failed to start season")
					continue
				}

				// Step 2: Create ALL games for the season with batching
				startMgr := NewSeasonStartManager(allStores.LeagueStore, allStores, cfg, gameCreator)
				gameResult, err := startMgr.CreateGamesForSeason(ctx, dbLeague.Uuid, season.Uuid, leagueSettings, 100*time.Millisecond, 2)
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
					Str("seasonID", startResult.SeasonID.String()).
					Int("totalGames", gameResult.TotalGamesCreated).
					Msg("✓ Successfully started season and created games")

				// Send season started notifications
				err = lifecycleMgr.SendSeasonStartedNotification(ctx, cfg, dbLeague.Uuid, season.Uuid)
				if err != nil {
					log.Error().Err(err).Str("seasonID", season.Uuid.String()).Msg("Failed to send season started notifications")
				}

				// Send Discord notification for season start
				SendSeasonStartedDiscord(cfg, dbLeague.Name, dbLeague.Slug, int(season.SeasonNumber))

				result.TasksRun++
				result.SeasonStarted = true
				result.GamesCreated = gameResult.TotalGamesCreated
			} else {
				log.Info().
					Str("seasonID", season.Uuid.String()).
					Msg("Season already started (idempotency check)")
			}
		}
	}

	return result, nil
}
