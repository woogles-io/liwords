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

	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// SeasonLifecycleManager handles automated season lifecycle operations
type SeasonLifecycleManager struct {
	store     league.Store
	gameStore GameStore
}

// NewSeasonLifecycleManager creates a new season lifecycle manager
func NewSeasonLifecycleManager(store league.Store, gameStore GameStore) *SeasonLifecycleManager {
	return &SeasonLifecycleManager{
		store:     store,
		gameStore: gameStore,
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
	currentSeason, err := slm.store.GetCurrentSeason(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("no current season found - use BootstrapSeason API to create first season: %w", err)
	}

	// Check if next season already exists
	nextSeasonNumber := currentSeason.SeasonNumber + 1
	_, err = slm.store.GetSeasonByLeagueAndNumber(ctx, leagueID, nextSeasonNumber)
	if err == nil {
		return nil, nil // Next season already exists, skip
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing season: %w", err)
	}

	// Get league info for result
	dbLeague, err := slm.store.GetLeagueByUUID(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get league: %w", err)
	}

	// Create next season with REGISTRATION_OPEN status
	nextStartDate := currentSeason.StartDate.Time.AddDate(0, 0, 21)
	nextEndDate := nextStartDate.AddDate(0, 0, 21)

	nextSeasonID := uuid.New()
	_, err = slm.store.CreateSeason(ctx, models.CreateSeasonParams{
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

// SeasonCloseResult tracks the outcome of closing a season
type SeasonCloseResult struct {
	LeagueID            uuid.UUID
	LeagueName          string
	CurrentSeasonID     uuid.UUID
	NextSeasonID        uuid.UUID
	ForceFinishedGames  int
	DivisionPreparation *DivisionPreparationResult
}

// CloseCurrentSeason closes the current season on Day 20 at midnight
// Returns nil if conditions aren't met (not Day 20, no current season, etc.)
func (slm *SeasonLifecycleManager) CloseCurrentSeason(
	ctx context.Context,
	leagueID uuid.UUID,
	now time.Time,
) (*SeasonCloseResult, error) {
	// Get current season
	currentSeason, err := slm.store.GetCurrentSeason(ctx, leagueID)
	if err != nil {
		return nil, nil // No current season, skip
	}

	// Check if current season is active
	if currentSeason.Status != int32(ipc.SeasonStatus_SEASON_ACTIVE) {
		return nil, nil // Not active, skip
	}

	// Get league info for result
	dbLeague, err := slm.store.GetLeagueByUUID(ctx, currentSeason.LeagueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get league: %w", err)
	}

	result := &SeasonCloseResult{
		LeagueID:        leagueID,
		LeagueName:      dbLeague.Name,
		CurrentSeasonID: currentSeason.Uuid,
	}

	// Step 1: Force-finish unfinished games
	forceFinishMgr := NewForceFinishManager(slm.store, slm.gameStore)
	ffResult, err := forceFinishMgr.ForceFinishUnfinishedGames(ctx, currentSeason.Uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to force-finish games: %w", err)
	}
	result.ForceFinishedGames = ffResult.ForceForfeitGames

	// Step 2: Mark season outcomes (PROMOTED/RELEGATED/STAYED)
	endOfSeasonMgr := NewEndOfSeasonManager(slm.store)
	err = endOfSeasonMgr.MarkSeasonOutcomes(ctx, currentSeason.Uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to mark season outcomes: %w", err)
	}

	// Step 3: Get next season (should exist from Day 15)
	nextSeasonNumber := currentSeason.SeasonNumber + 1
	nextSeason, err := slm.store.GetSeasonByLeagueAndNumber(ctx, leagueID, nextSeasonNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("next season not found - was registration opened on Day 15?")
		}
		return nil, fmt.Errorf("failed to get next season: %w", err)
	}
	result.NextSeasonID = nextSeason.Uuid

	// Parse league settings to get ideal division size
	var leagueSettings ipc.LeagueSettings
	if err := json.Unmarshal(dbLeague.Settings, &leagueSettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal league settings: %w", err)
	}

	// Step 4: Prepare next season divisions
	orchestrator := NewSeasonOrchestrator(slm.store)
	divPrep, err := orchestrator.PrepareNextSeasonDivisions(
		ctx,
		leagueID,
		currentSeason.Uuid,
		nextSeason.Uuid,
		nextSeasonNumber,
		leagueSettings.IdealDivisionSize,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare next season divisions: %w", err)
	}
	result.DivisionPreparation = divPrep

	// Step 5: Update next season status to SCHEDULED
	err = slm.store.UpdateSeasonStatus(ctx, models.UpdateSeasonStatusParams{
		Uuid:   nextSeason.Uuid,
		Status: int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update next season status: %w", err)
	}

	// Step 6: Mark current season as COMPLETED
	err = slm.store.UpdateSeasonStatus(ctx, models.UpdateSeasonStatusParams{
		Uuid:   currentSeason.Uuid,
		Status: int32(ipc.SeasonStatus_SEASON_COMPLETED),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to mark current season as completed: %w", err)
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
	season, err := slm.store.GetSeason(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get season: %w", err)
	}

	// Check if season is SCHEDULED
	if season.Status != int32(ipc.SeasonStatus_SEASON_SCHEDULED) {
		return nil, nil // Not scheduled, skip
	}

	// Check if start time has passed
	if now.Before(season.StartDate.Time) {
		return nil, nil // Start time not reached, skip
	}

	// Get league info for result
	dbLeague, err := slm.store.GetLeagueByUUID(ctx, leagueID)
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
	err = slm.store.UpdateSeasonStatus(ctx, models.UpdateSeasonStatusParams{
		Uuid:   seasonID,
		Status: int32(ipc.SeasonStatus_SEASON_ACTIVE),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update season status: %w", err)
	}

	// Set this season as the current season
	err = slm.store.SetCurrentSeason(ctx, models.SetCurrentSeasonParams{
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
