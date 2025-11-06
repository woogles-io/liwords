package league

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// SeasonLifecycleManager handles automated season lifecycle operations
type SeasonLifecycleManager struct {
	stores *stores.Stores
}

// NewSeasonLifecycleManager creates a new season lifecycle manager
func NewSeasonLifecycleManager(allStores *stores.Stores) *SeasonLifecycleManager {
	return &SeasonLifecycleManager{
		stores: allStores,
	}
}

// RegistrationOpenResult tracks the outcome of opening registration
type RegistrationOpenResult struct {
	LeagueID         uuid.UUID
	LeagueName       string
	NextSeasonID     uuid.UUID
	NextSeasonNumber int32
	StartDate        time.Time
}

// OpenRegistrationForNextSeason opens registration for the next season on Day 15
// Returns nil if conditions aren't met.
func (slm *SeasonLifecycleManager) OpenRegistrationForNextSeason(
	ctx context.Context,
	leagueID uuid.UUID,
	now time.Time,
) (*RegistrationOpenResult, error) {
	// Get current season
	currentSeason, err := slm.stores.LeagueStore.GetCurrentSeason(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("no current season found - use BootstrapSeason API to create first season: %w", err)
	}

	// Check if next season already exists
	nextSeasonNumber := currentSeason.SeasonNumber + 1
	_, err = slm.stores.LeagueStore.GetSeasonByLeagueAndNumber(ctx, leagueID, nextSeasonNumber)
	if err == nil {
		return nil, nil // Next season already exists, skip
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing season: %w", err)
	}

	// Get league info for result
	dbLeague, err := slm.stores.LeagueStore.GetLeagueByUUID(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get league: %w", err)
	}

	// Create next season with REGISTRATION_OPEN status
	nextStartDate := currentSeason.StartDate.Time.AddDate(0, 0, 21)
	nextEndDate := nextStartDate.AddDate(0, 0, 21)

	nextSeasonID := uuid.New()
	_, err = slm.stores.LeagueStore.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         nextSeasonID,
		LeagueID:     leagueID,
		SeasonNumber: nextSeasonNumber,
		StartDate:    pgtype.Timestamptz{Time: nextStartDate, Valid: true},
		EndDate:      pgtype.Timestamptz{Time: nextEndDate, Valid: true},
		Status:       int32(ipc.SeasonStatus_SEASON_REGISTRATION_OPEN),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create next season: %w", err)
	}

	return &RegistrationOpenResult{
		LeagueID:         leagueID,
		LeagueName:       dbLeague.Name,
		NextSeasonID:     nextSeasonID,
		NextSeasonNumber: nextSeasonNumber,
		StartDate:        nextStartDate,
	}, nil
}

// OpenRegistrationForSeason opens registration for a specific existing season
// Changes the season status from SCHEDULED to REGISTRATION_OPEN
func (slm *SeasonLifecycleManager) OpenRegistrationForSeason(
	ctx context.Context,
	seasonID uuid.UUID,
) (*RegistrationOpenResult, error) {
	// Get the season
	season, err := slm.stores.LeagueStore.GetSeason(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get season: %w", err)
	}

	// Verify season is in SCHEDULED status
	if season.Status != int32(ipc.SeasonStatus_SEASON_SCHEDULED) {
		return nil, fmt.Errorf("season must be SCHEDULED to open registration, current status: %d", season.Status)
	}

	// Get league info
	dbLeague, err := slm.stores.LeagueStore.GetLeagueByUUID(ctx, season.LeagueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get league: %w", err)
	}

	// Update season status to REGISTRATION_OPEN
	err = slm.stores.LeagueStore.UpdateSeasonStatus(ctx, models.UpdateSeasonStatusParams{
		Uuid:   seasonID,
		Status: int32(ipc.SeasonStatus_SEASON_REGISTRATION_OPEN),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update season status: %w", err)
	}

	return &RegistrationOpenResult{
		LeagueID:         season.LeagueID,
		LeagueName:       dbLeague.Name,
		NextSeasonID:     seasonID,
		NextSeasonNumber: season.SeasonNumber,
		StartDate:        season.StartDate.Time,
	}, nil
}

// SeasonCloseResult tracks the outcome of closing a season
type SeasonCloseResult struct {
	LeagueID           uuid.UUID
	LeagueName         string
	CurrentSeasonID    uuid.UUID
	ForceFinishedGames int
}

// CloseCurrentSeason closes the current season on Day 20 at midnight
// This force-finishes unfinished games, marks season outcomes, and sets the season to COMPLETED
// Returns nil if conditions aren't met (no current season, season not ACTIVE, etc.)
func (slm *SeasonLifecycleManager) CloseCurrentSeason(
	ctx context.Context,
	leagueID uuid.UUID,
	now time.Time,
) (*SeasonCloseResult, error) {
	// Get current season
	currentSeason, err := slm.stores.LeagueStore.GetCurrentSeason(ctx, leagueID)
	if err != nil {
		return nil, nil // No current season, skip
	}

	// Check if current season is active
	if currentSeason.Status != int32(ipc.SeasonStatus_SEASON_ACTIVE) {
		return nil, nil // Not active, skip
	}

	// Get league info for result
	dbLeague, err := slm.stores.LeagueStore.GetLeagueByUUID(ctx, currentSeason.LeagueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get league: %w", err)
	}

	result := &SeasonCloseResult{
		LeagueID:        leagueID,
		LeagueName:      dbLeague.Name,
		CurrentSeasonID: currentSeason.Uuid,
	}

	// Step 1: Force-finish unfinished games
	forceFinishMgr := NewForceFinishManager(slm.stores)
	ffResult, err := forceFinishMgr.ForceFinishUnfinishedGames(ctx, currentSeason.Uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to force-finish games: %w", err)
	}
	result.ForceFinishedGames = ffResult.ForceForfeitGames

	// Step 2: Mark season outcomes (PROMOTED/RELEGATED/STAYED)
	endOfSeasonMgr := NewEndOfSeasonManager(slm.stores.LeagueStore)
	err = endOfSeasonMgr.MarkSeasonOutcomes(ctx, currentSeason.Uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to mark season outcomes: %w", err)
	}

	// Step 3: Mark current season as COMPLETED
	err = slm.stores.LeagueStore.UpdateSeasonStatus(ctx, models.UpdateSeasonStatusParams{
		Uuid:   currentSeason.Uuid,
		Status: int32(ipc.SeasonStatus_SEASON_COMPLETED),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to mark current season as completed: %w", err)
	}

	return result, nil
}

// PrepareAndScheduleSeasonResult tracks the outcome of preparing divisions
type PrepareAndScheduleSeasonResult struct {
	LeagueID            uuid.UUID
	LeagueName          string
	SeasonID            uuid.UUID
	SeasonNumber        int32
	DivisionPreparation *DivisionPreparationResult
}

// PrepareAndScheduleSeason closes registration and prepares divisions for a SCHEDULED season
// This should be called on Day 21 at 7:45 AM (15 minutes before season start at 8:00 AM)
// For Season 1 (bootstrapped), this is the first division creation step
// Returns nil if conditions aren't met (season not SCHEDULED, etc.)
func (slm *SeasonLifecycleManager) PrepareAndScheduleSeason(
	ctx context.Context,
	leagueID uuid.UUID,
	seasonID uuid.UUID,
	now time.Time,
) (*PrepareAndScheduleSeasonResult, error) {
	// Get season
	season, err := slm.stores.LeagueStore.GetSeason(ctx, seasonID)
	if err != nil {
		return nil, errors.New("season not found")
	}

	// Check if season is SCHEDULED
	if season.Status != int32(ipc.SeasonStatus_SEASON_SCHEDULED) {
		return nil, errors.New("season is not in SCHEDULED status")
	}

	// Get league info for result
	dbLeague, err := slm.stores.LeagueStore.GetLeagueByUUID(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get league: %w", err)
	}

	result := &PrepareAndScheduleSeasonResult{
		LeagueID:     leagueID,
		LeagueName:   dbLeague.Name,
		SeasonID:     seasonID,
		SeasonNumber: season.SeasonNumber,
	}

	// Parse league settings to get ideal division size
	var leagueSettings ipc.LeagueSettings
	if err := json.Unmarshal(dbLeague.Settings, &leagueSettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal league settings: %w", err)
	}

	// Determine previous season ID (uuid.Nil for Season 1)
	previousSeasonID := uuid.Nil
	if season.SeasonNumber > 1 {
		prevSeason, err := slm.stores.LeagueStore.GetSeasonByLeagueAndNumber(ctx, leagueID, season.SeasonNumber-1)
		if err != nil {
			return nil, fmt.Errorf("failed to get previous season: %w", err)
		}
		previousSeasonID = prevSeason.Uuid
	}

	// Check if divisions already exist for this season (e.g., if registration was reopened)
	// If they do, delete them so we can recreate with updated registrations
	existingDivisions, err := slm.stores.LeagueStore.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing divisions: %w", err)
	}

	if len(existingDivisions) > 0 {
		log.Info().
			Str("seasonID", seasonID.String()).
			Int("existingDivisions", len(existingDivisions)).
			Msg("found existing divisions, deleting them before re-preparing season")

		// Delete each division (standings will CASCADE delete, registrations will have division_id SET NULL)
		for _, div := range existingDivisions {
			// Delete standings first (explicit for clarity, though CASCADE handles this)
			if err := slm.stores.LeagueStore.DeleteDivisionStandings(ctx, div.Uuid); err != nil {
				return nil, fmt.Errorf("failed to delete standings for division %d: %w", div.DivisionNumber, err)
			}

			// Delete division
			if err := slm.stores.LeagueStore.DeleteDivision(ctx, div.Uuid); err != nil {
				return nil, fmt.Errorf("failed to delete division %d: %w", div.DivisionNumber, err)
			}
		}

		log.Info().
			Str("seasonID", seasonID.String()).
			Int("divisionsDeleted", len(existingDivisions)).
			Msg("successfully deleted existing divisions")
	}

	// Prepare divisions for this season
	orchestrator := NewSeasonOrchestrator(slm.stores)
	divPrep, err := orchestrator.PrepareNextSeasonDivisions(
		ctx,
		leagueID,
		previousSeasonID,
		seasonID,
		season.SeasonNumber,
		leagueSettings.IdealDivisionSize,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare season divisions: %w", err)
	}
	result.DivisionPreparation = divPrep

	// Update season status to SCHEDULED
	err = slm.stores.LeagueStore.UpdateSeasonStatus(ctx, models.UpdateSeasonStatusParams{
		Uuid:   seasonID,
		Status: int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update season status: %w", err)
	}

	return result, nil
}

// SeasonStartResult tracks the outcome of starting a season
type SeasonStartResult struct {
	LeagueID   uuid.UUID
	LeagueName string
	SeasonID   uuid.UUID
}

// StartScheduledSeason starts a season that is SCHEDULED and past its start date
// Returns nil if conditions aren't met (not SCHEDULED, start time not reached, etc.)
func (slm *SeasonLifecycleManager) StartScheduledSeason(
	ctx context.Context,
	leagueID uuid.UUID,
	seasonID uuid.UUID,
	now time.Time,
) (*SeasonStartResult, error) {
	// Get season
	season, err := slm.stores.LeagueStore.GetSeason(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get season: %w", err)
	}

	// Check if season is SCHEDULED
	if season.Status != int32(ipc.SeasonStatus_SEASON_SCHEDULED) {
		statusNames := map[int32]string{
			0: "SCHEDULED",
			1: "ACTIVE",
			2: "COMPLETED",
			3: "CANCELLED",
			4: "REGISTRATION_OPEN",
		}
		statusName := statusNames[season.Status]
		if statusName == "" {
			statusName = fmt.Sprintf("UNKNOWN(%d)", season.Status)
		}
		return nil, fmt.Errorf("season must be SCHEDULED to start, current status: %s", statusName)
	}

	// Check if start time has passed
	if now.Before(season.StartDate.Time) {
		return nil, fmt.Errorf("season start time (%s) has not been reached yet (current time: %s)",
			season.StartDate.Time.Format(time.RFC3339),
			now.Format(time.RFC3339))
	}

	// Get league info for result
	dbLeague, err := slm.stores.LeagueStore.GetLeagueByUUID(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get league: %w", err)
	}

	// Note: Game creation should be done by the caller using SeasonStartManager
	// after this function returns successfully. This keeps SeasonLifecycleManager
	// lightweight and focused on status transitions.
	//
	// To create games:
	//   gameCreator := &GameplayAdapter{stores: stores, cfg: cfg, eventChan: eventChan}
	//   startMgr := NewSeasonStartManager(store, stores, cfg, eventChan, gameCreator)
	//   result, err := startMgr.CreateGamesForSeason(ctx, leagueID, seasonID, leagueSettings)

	// Update season status to ACTIVE
	err = slm.stores.LeagueStore.UpdateSeasonStatus(ctx, models.UpdateSeasonStatusParams{
		Uuid:   seasonID,
		Status: int32(ipc.SeasonStatus_SEASON_ACTIVE),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update season status: %w", err)
	}

	// Set this season as the current season
	err = slm.stores.LeagueStore.SetCurrentSeason(ctx, models.SetCurrentSeasonParams{
		Uuid:            leagueID,
		CurrentSeasonID: pgtype.UUID{Bytes: seasonID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set current season: %w", err)
	}

	return &SeasonStartResult{
		LeagueID:   leagueID,
		LeagueName: dbLeague.Name,
		SeasonID:   seasonID,
	}, nil
}

// RollbackSeasonToScheduled rolls back a season from ACTIVE to SCHEDULED
// This is used when game creation fails after the season was started
func (slm *SeasonLifecycleManager) RollbackSeasonToScheduled(ctx context.Context, seasonID uuid.UUID) error {
	// Get the season to verify it exists
	season, err := slm.stores.LeagueStore.GetSeason(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("failed to get season: %w", err)
	}

	// Update season status back to SCHEDULED
	err = slm.stores.LeagueStore.UpdateSeasonStatus(ctx, models.UpdateSeasonStatusParams{
		Uuid:   seasonID,
		Status: int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	if err != nil {
		return fmt.Errorf("failed to rollback season status: %w", err)
	}

	// Clear current_season_id if this was set as current
	err = slm.stores.LeagueStore.SetCurrentSeason(ctx, models.SetCurrentSeasonParams{
		Uuid:            season.LeagueID,
		CurrentSeasonID: pgtype.UUID{Valid: false}, // NULL
	})
	if err != nil {
		// Log but don't fail - this is best effort cleanup
		log.Warn().Err(err).
			Str("leagueID", season.LeagueID.String()).
			Str("seasonID", seasonID.String()).
			Msg("failed to clear current season during rollback")
	}

	return nil
}
