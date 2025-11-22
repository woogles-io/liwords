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

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// SeasonLifecycleManager handles automated season lifecycle operations
type SeasonLifecycleManager struct {
	stores *stores.Stores
	clock  Clock
}

// NewSeasonLifecycleManager creates a new season lifecycle manager
func NewSeasonLifecycleManager(allStores *stores.Stores, clock Clock) *SeasonLifecycleManager {
	return &SeasonLifecycleManager{
		stores: allStores,
		clock:  clock,
	}
}

// GetRegisteredPlayerIDs returns the UUIDs of all players registered for a season
func (slm *SeasonLifecycleManager) GetRegisteredPlayerIDs(ctx context.Context, seasonID uuid.UUID) ([]string, error) {
	registrations, err := slm.stores.LeagueStore.GetSeasonRegistrations(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get season registrations: %w", err)
	}

	userIDs := make([]string, 0, len(registrations))
	for _, reg := range registrations {
		if reg.UserUuid.Valid {
			userIDs = append(userIDs, reg.UserUuid.String)
		}
	}

	return userIDs, nil
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
) (*RegistrationOpenResult, error) {
	// Get current season (try current_season_id first, fall back to latest season)
	currentSeason, err := slm.stores.LeagueStore.GetCurrentSeason(ctx, leagueID)
	if err != nil {
		// If no current season set, try to get the latest season by number
		allSeasons, err := slm.stores.LeagueStore.GetSeasonsByLeague(ctx, leagueID)
		if err != nil || len(allSeasons) == 0 {
			return nil, fmt.Errorf("no seasons found - use BootstrapSeason API to create first season")
		}

		// Find the latest season (highest season number)
		latestSeason := allSeasons[0]
		for _, season := range allSeasons {
			if season.SeasonNumber > latestSeason.SeasonNumber {
				latestSeason = season
			}
		}
		currentSeason = latestSeason
	}

	// Get league info early to check if league is active
	dbLeague, err := slm.stores.LeagueStore.GetLeagueByUUID(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get league: %w", err)
	}

	// Parse league settings to get season length
	var leagueSettings ipc.LeagueSettings
	if err := json.Unmarshal(dbLeague.Settings, &leagueSettings); err != nil {
		return nil, fmt.Errorf("failed to parse league settings: %w", err)
	}

	// Safety check: League must be active
	if !dbLeague.IsActive.Valid || !dbLeague.IsActive.Bool {
		return nil, fmt.Errorf("cannot open registration: league is not active")
	}

	// Safety check: Verify no orphaned REGISTRATION_OPEN seasons exist
	allSeasons, err := slm.stores.LeagueStore.GetSeasonsByLeague(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing seasons: %w", err)
	}
	for _, season := range allSeasons {
		if season.Status == int32(ipc.SeasonStatus_SEASON_REGISTRATION_OPEN) {
			return nil, fmt.Errorf("cannot open registration: another season (%d) is already in REGISTRATION_OPEN status", season.SeasonNumber)
		}
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

	// Create next season with REGISTRATION_OPEN status
	// Preserve the time-of-day from the current season
	// Use the season length from league settings
	seasonLengthDays := int(leagueSettings.SeasonLengthDays)
	nextStartDate := currentSeason.StartDate.Time.AddDate(0, 0, seasonLengthDays)
	nextEndDate := currentSeason.EndDate.Time.AddDate(0, 0, seasonLengthDays)

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

// SendSeasonStartingSoonNotification sends reminder emails 1 day before season starts
func (slm *SeasonLifecycleManager) SendSeasonStartingSoonNotification(
	ctx context.Context,
	cfg *config.Config,
	leagueID uuid.UUID,
	nextSeasonID uuid.UUID,
) error {
	// Get league info
	dbLeague, err := slm.stores.LeagueStore.GetLeagueByUUID(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("failed to get league: %w", err)
	}

	// Get next season info
	nextSeason, err := slm.stores.LeagueStore.GetSeason(ctx, nextSeasonID)
	if err != nil {
		return fmt.Errorf("failed to get next season: %w", err)
	}

	// Get registered players for the next season
	registeredUserIDs, err := slm.GetRegisteredPlayerIDs(ctx, nextSeasonID)
	if err != nil {
		return fmt.Errorf("failed to get registered players: %w", err)
	}

	if len(registeredUserIDs) == 0 {
		log.Info().Str("leagueID", leagueID.String()).Int32("season", nextSeason.SeasonNumber).Msg("no-registered-players-for-season-starting-soon-notification")
		return nil
	}

	// Send emails
	SendSeasonStartingSoonEmail(
		ctx,
		cfg,
		slm.stores.UserStore,
		dbLeague.Name,
		dbLeague.Slug,
		int(nextSeason.SeasonNumber),
		nextSeason.StartDate.Time,
		registeredUserIDs,
	)

	log.Info().Str("leagueID", leagueID.String()).Int32("season", nextSeason.SeasonNumber).Int("count", len(registeredUserIDs)).Msg("sent-season-starting-soon-notifications")
	return nil
}

// SendSeasonStartedNotification sends emails to players when season starts and games are created
func (slm *SeasonLifecycleManager) SendSeasonStartedNotification(
	ctx context.Context,
	cfg *config.Config,
	leagueID uuid.UUID,
	seasonID uuid.UUID,
) error {
	// Get league info
	dbLeague, err := slm.stores.LeagueStore.GetLeagueByUUID(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("failed to get league: %w", err)
	}

	// Get season info
	season, err := slm.stores.LeagueStore.GetSeason(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("failed to get season: %w", err)
	}

	// Build player assignment map
	playerAssignments, err := slm.buildPlayerAssignments(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("failed to build player assignments: %w", err)
	}

	if len(playerAssignments) == 0 {
		log.Info().Str("leagueID", leagueID.String()).Int32("season", season.SeasonNumber).Msg("no-players-for-season-started-notification")
		return nil
	}

	// Send emails
	SendSeasonStartedEmail(
		ctx,
		cfg,
		slm.stores.UserStore,
		dbLeague.Name,
		dbLeague.Slug,
		int(season.SeasonNumber),
		playerAssignments,
	)

	log.Info().Str("leagueID", leagueID.String()).Int32("season", season.SeasonNumber).Int("count", len(playerAssignments)).Msg("sent-season-started-notifications")
	return nil
}

// buildPlayerAssignments creates a map of player assignments with division and opponent info
func (slm *SeasonLifecycleManager) buildPlayerAssignments(ctx context.Context, seasonID uuid.UUID) (map[string]*PlayerSeasonInfo, error) {
	// Get all registrations for the season
	registrations, err := slm.stores.LeagueStore.GetSeasonRegistrations(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get season registrations: %w", err)
	}

	// Map players to their divisions
	playerDivisions := make(map[string]uuid.UUID) // userUUID -> divisionID
	divisionNames := make(map[uuid.UUID]string)   // divisionID -> divisionName

	for _, reg := range registrations {
		if !reg.UserUuid.Valid || !reg.DivisionID.Valid {
			continue
		}

		userUUID := reg.UserUuid.String
		divisionID := uuid.UUID(reg.DivisionID.Bytes)

		playerDivisions[userUUID] = divisionID

		// Fetch division name if we haven't yet
		if _, exists := divisionNames[divisionID]; !exists {
			division, err := slm.stores.LeagueStore.GetDivision(ctx, divisionID)
			if err == nil {
				if division.DivisionName.Valid {
					divisionNames[divisionID] = division.DivisionName.String
				} else {
					divisionNames[divisionID] = fmt.Sprintf("Division %d", division.DivisionNumber)
				}
			} else {
				divisionNames[divisionID] = fmt.Sprintf("Division %d", len(divisionNames)+1)
			}
		}
	}

	// Build the player assignment map
	assignments := make(map[string]*PlayerSeasonInfo)
	for userUUID, divisionID := range playerDivisions {
		// Get actual opponents from games (not all division members)
		opponents, err := slm.stores.LeagueStore.GetPlayerSeasonOpponents(ctx, seasonID, userUUID)
		if err != nil {
			log.Warn().Err(err).Str("userUUID", userUUID).Msg("failed-to-get-player-opponents")
			opponents = []string{}
		}

		assignments[userUUID] = &PlayerSeasonInfo{
			DivisionName:  divisionNames[divisionID],
			OpponentNames: opponents,
		}
	}

	return assignments, nil
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

	// Safety check: Season must have at least one division
	divisions, err := slm.stores.LeagueStore.GetDivisionsBySeason(ctx, currentSeason.Uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to get divisions: %w", err)
	}
	if len(divisions) == 0 {
		return nil, fmt.Errorf("cannot close season: no divisions exist")
	}

	// Safety check: All divisions must have standings calculated
	for _, division := range divisions {
		divisionUUID, err := uuid.FromBytes(division.Uuid[:])
		if err != nil {
			continue
		}
		standings, err := slm.stores.LeagueStore.GetStandings(ctx, divisionUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get standings for division %d: %w", division.DivisionNumber, err)
		}
		if len(standings) == 0 {
			return nil, fmt.Errorf("cannot close season: division %d has no standings", division.DivisionNumber)
		}
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

	// Post-operation check: Verify all games are finished
	for _, division := range divisions {
		divisionUUID, err := uuid.FromBytes(division.Uuid[:])
		if err != nil {
			continue
		}
		totalGames, err := slm.stores.LeagueStore.CountDivisionGamesTotal(ctx, divisionUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to count total games: %w", err)
		}
		completedGames, err := slm.stores.LeagueStore.CountDivisionGamesComplete(ctx, divisionUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to count completed games: %w", err)
		}
		unfinishedCount := totalGames - completedGames
		if unfinishedCount > 0 {
			return nil, fmt.Errorf("force-finish failed: division %d still has %d unfinished games", division.DivisionNumber, unfinishedCount)
		}
	}

	// Step 2: Mark season outcomes (PROMOTED/RELEGATED/STAYED)
	endOfSeasonMgr := NewEndOfSeasonManager(slm.stores.LeagueStore)
	err = endOfSeasonMgr.MarkSeasonOutcomes(ctx, currentSeason.Uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to mark season outcomes: %w", err)
	}

	// Post-operation check: Verify all registrations have placement_status set
	registrations, err := slm.stores.LeagueStore.GetSeasonRegistrations(ctx, currentSeason.Uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to get season registrations: %w", err)
	}
	for _, reg := range registrations {
		if !reg.PlacementStatus.Valid {
			return nil, fmt.Errorf("registration for user ID %d missing placement status after marking outcomes", reg.UserID)
		}
	}

	// Step 3: Mark current season as COMPLETED
	err = slm.stores.LeagueStore.UpdateSeasonStatus(ctx, models.UpdateSeasonStatusParams{
		Uuid:   currentSeason.Uuid,
		Status: int32(ipc.SeasonStatus_SEASON_COMPLETED),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to mark current season as completed: %w", err)
	}

	// Step 4: Clear current_season_id since this season is no longer active
	err = slm.stores.LeagueStore.SetCurrentSeason(ctx, models.SetCurrentSeasonParams{
		Uuid:            leagueID,
		CurrentSeasonID: pgtype.UUID{Valid: false}, // Set to NULL
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clear current season: %w", err)
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

// PrepareAndScheduleSeason closes registration and prepares divisions for a season
// Accepts seasons in REGISTRATION_OPEN or SCHEDULED status:
// - REGISTRATION_OPEN: closes registration, creates divisions, sets to SCHEDULED
// - SCHEDULED: recreates divisions (allows re-running if registrations changed)
// This should be called on Day 21 at midnight (8 hours before season start at 8:00 AM)
// Returns nil if season is not in an appropriate status (silently skips)
func (slm *SeasonLifecycleManager) PrepareAndScheduleSeason(
	ctx context.Context,
	leagueID uuid.UUID,
	seasonID uuid.UUID,
) (*PrepareAndScheduleSeasonResult, error) {
	// Get season
	season, err := slm.stores.LeagueStore.GetSeason(ctx, seasonID)
	if err != nil {
		return nil, errors.New("season not found")
	}

	// Check if season is REGISTRATION_OPEN or SCHEDULED
	if season.Status != int32(ipc.SeasonStatus_SEASON_SCHEDULED) &&
		season.Status != int32(ipc.SeasonStatus_SEASON_REGISTRATION_OPEN) {
		return nil, nil // Skip seasons that aren't ready (return nil to avoid error logs)
	}

	// Safety check: Season must have valid start/end dates
	if !season.StartDate.Valid || !season.EndDate.Valid {
		return nil, fmt.Errorf("cannot prepare season: invalid start/end dates")
	}

	// Safety check: Season must have registrations
	registrations, err := slm.stores.LeagueStore.GetSeasonRegistrations(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get season registrations: %w", err)
	}
	if len(registrations) == 0 {
		return nil, fmt.Errorf("cannot prepare season: no registrations found")
	}

	// Safety check: Minimum player threshold
	const MinimumPlayersForSeason = 11
	if len(registrations) < MinimumPlayersForSeason {
		return nil, fmt.Errorf("cannot prepare season: insufficient registrations (%d, minimum %d)", len(registrations), MinimumPlayersForSeason)
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

		// Safety check: Previous season must be COMPLETED
		if prevSeason.Status != int32(ipc.SeasonStatus_SEASON_COMPLETED) {
			return nil, fmt.Errorf("cannot prepare season %d: previous season (season %d) must be COMPLETED (current status: %s)",
				season.SeasonNumber, prevSeason.SeasonNumber, ipc.SeasonStatus(prevSeason.Status).String())
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

	// Post-operation check: Verify divisions were created
	createdDivisions, err := slm.stores.LeagueStore.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify division creation: %w", err)
	}
	if len(createdDivisions) == 0 {
		return nil, fmt.Errorf("division preparation failed: no divisions created")
	}

	// Post-operation check: Verify all registrations are assigned to divisions
	updatedRegistrations, err := slm.stores.LeagueStore.GetSeasonRegistrations(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify registration assignments: %w", err)
	}
	unassignedCount := 0
	for _, reg := range updatedRegistrations {
		if !reg.DivisionID.Valid {
			unassignedCount++
		}
	}
	if unassignedCount > 0 {
		return nil, fmt.Errorf("division preparation failed: %d players not assigned to divisions", unassignedCount)
	}

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

// StartScheduledSeason starts a season that is SCHEDULED
// Returns nil if conditions aren't met (season not SCHEDULED)
// Note: Timing is controlled by external task scheduling, not checked here
func (slm *SeasonLifecycleManager) StartScheduledSeason(
	ctx context.Context,
	leagueID uuid.UUID,
	seasonID uuid.UUID,
) (*SeasonStartResult, error) {
	// Get season
	season, err := slm.stores.LeagueStore.GetSeason(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get season: %w", err)
	}

	// Check if season is SCHEDULED
	if season.Status != int32(ipc.SeasonStatus_SEASON_SCHEDULED) {
		return nil, nil // Skip seasons that aren't SCHEDULED (no error logged)
	}

	// Get league info for result
	dbLeague, err := slm.stores.LeagueStore.GetLeagueByUUID(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get league: %w", err)
	}

	// Safety check: Ensure divisions have been created and have players
	divisions, err := slm.stores.LeagueStore.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get divisions: %w", err)
	}
	if len(divisions) == 0 {
		return nil, fmt.Errorf("cannot start season: no divisions created yet (run PrepareAndScheduleSeason first)")
	}
	log.Info().Str("leagueID", leagueID.String()).Int32("season-number", season.SeasonNumber).Msg("season divisions verified")

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

// ShouldRunTask checks if a maintenance task should run based on season dates and time
// Returns (shouldRun bool, reason string)
func ShouldRunTask(season *models.LeagueSeason, taskType string, seasonLengthDays int32, now time.Time) (bool, string) {
	// Ensure season has valid dates
	if !season.StartDate.Valid || !season.EndDate.Valid {
		return false, "Season has invalid start/end dates"
	}

	startDate := season.StartDate.Time
	endDate := season.EndDate.Time

	switch taskType {
	case "close-season":
		// Should run at midnight (or later) on the last day of season
		// The EndDate is the last day of the season
		seasonEndDate := endDate.Truncate(24 * time.Hour)
		todayMidnight := now.Truncate(24 * time.Hour)

		if !todayMidnight.Equal(seasonEndDate) {
			return false, fmt.Sprintf("Today=%s, SeasonEnd=%s",
				todayMidnight.Format("2006-01-02"),
				seasonEndDate.Format("2006-01-02"))
		}

		// Date matches - midnight tasks can run any time on the correct day
		return true, "Today is last day of season"

	case "open-registration":
		// Should run at the same time as season start, but halfway through the season
		// Calculate registration open time: start time + (seasonLength / 2) days
		// This preserves the time-of-day and timezone from the start date
		daysUntilOpen := (seasonLengthDays / 2) - 1 // -1 because Day 1 is start date
		registrationOpenTime := startDate.Add(time.Duration(daysUntilOpen) * 24 * time.Hour)

		if now.Before(registrationOpenTime) {
			return false, fmt.Sprintf("Current time %s is before registration open time %s",
				now.Format("2006-01-02 15:04:05 MST"),
				registrationOpenTime.Format("2006-01-02 15:04:05 MST"))
		}

		return true, fmt.Sprintf("Current time has reached registration open time (%s, halfway through %d-day season)",
			registrationOpenTime.Format("2006-01-02 15:04:05 MST"), seasonLengthDays)

	case "start-season":
		// Should run at or after the season's start date/time
		// The start date includes the time (e.g., 2025-11-19 08:00:00-05:00 for 8 AM Eastern)
		if now.Before(startDate) {
			return false, fmt.Sprintf("Current time %s is before season start %s",
				now.Format("2006-01-02 15:04:05 MST"),
				startDate.Format("2006-01-02 15:04:05 MST"))
		}

		return true, fmt.Sprintf("Current time has reached season start time (%s)",
			startDate.Format("2006-01-02 15:04:05 MST"))

	case "season-starting-soon":
		// Should run at the same time as season start, but on the last day of the season (1 day before next season starts)
		// Calculate: start time + (seasonLength - 1) days
		// This preserves the time-of-day and timezone from the start date
		lastDayTime := startDate.Add(time.Duration(seasonLengthDays-1) * 24 * time.Hour)

		if now.Before(lastDayTime) {
			return false, fmt.Sprintf("Current time %s is before last day time %s",
				now.Format("2006-01-02 15:04:05 MST"),
				lastDayTime.Format("2006-01-02 15:04:05 MST"))
		}

		return true, fmt.Sprintf("Current time has reached last day of season (%s, 1 day before next season starts)",
			lastDayTime.Format("2006-01-02 15:04:05 MST"))

	case "prepare-divisions":
		// Should run at midnight (or later) on the day the season starts (before the actual start time)
		// Calculate midnight on the start date by truncating to the day
		// This runs before the season actually starts (e.g., midnight before 8 AM start)
		midnightBeforeStart := startDate.Truncate(24 * time.Hour)

		if now.Before(midnightBeforeStart) {
			return false, fmt.Sprintf("Current time %s is before midnight on start day %s",
				now.Format("2006-01-02 15:04:05 MST"),
				midnightBeforeStart.Format("2006-01-02 15:04:05 MST"))
		}

		// Make sure we haven't passed the actual start time yet
		// (divisions should be prepared BEFORE the season starts)
		if now.After(startDate) {
			return false, fmt.Sprintf("Current time %s is after season start %s (too late to prepare divisions)",
				now.Format("2006-01-02 15:04:05 MST"),
				startDate.Format("2006-01-02 15:04:05 MST"))
		}

		return true, fmt.Sprintf("Current time is between midnight and season start (prepare divisions now)")
	}

	return false, "Unknown task type"
}

// SendUnstartedGameReminder sends reminder emails to players who haven't started their games
// isFirm determines whether to send a gentle reminder (Day 1) or firm warning (Day 2)
func (slm *SeasonLifecycleManager) SendUnstartedGameReminder(
	ctx context.Context,
	cfg *config.Config,
	leagueID uuid.UUID,
	seasonID uuid.UUID,
	isFirm bool,
) error {
	// Get league info
	dbLeague, err := slm.stores.LeagueStore.GetLeagueByUUID(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("failed to get league: %w", err)
	}

	// Get season info
	season, err := slm.stores.LeagueStore.GetSeason(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("failed to get season: %w", err)
	}

	// Get players with unstarted games using the new query
	playersWithUnstartedGames, err := slm.stores.LeagueStore.GetSeasonPlayersWithUnstartedGames(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("failed to get players with unstarted games: %w", err)
	}

	if len(playersWithUnstartedGames) == 0 {
		log.Info().Str("leagueID", leagueID.String()).Int32("season", season.SeasonNumber).Msg("no-players-with-unstarted-games")
		return nil
	}

	// Send emails to each player
	SendUnstartedGameReminderEmail(
		ctx,
		cfg,
		slm.stores.UserStore,
		dbLeague.Name,
		dbLeague.Slug,
		int(season.SeasonNumber),
		playersWithUnstartedGames,
		isFirm,
	)

	reminderType := "gentle"
	if isFirm {
		reminderType = "firm"
	}
	log.Info().
		Str("leagueID", leagueID.String()).
		Int32("season", season.SeasonNumber).
		Int("count", len(playersWithUnstartedGames)).
		Str("type", reminderType).
		Msg("sent-unstarted-game-reminders")
	return nil
}
