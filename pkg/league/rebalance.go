package league

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	// Priority score constants
	DivisionMultiplier = 1_000

	PriorityBonusRelegated       = 500
	PriorityBonusStayed          = 400
	PriorityBonusPromoted        = 300
	PriorityBonusHiatusReturning = 100
	PriorityBonusNew             = 50 // Lowest priority - new players placed naturally via rebalancing

	// Hiatus weight: 0.933^N (halves every ~10 seasons)
	HiatusWeightBase = 0.933

	// Division sizing
	MinimumFinalDivSize = 12
)

// RebalanceManager handles the rebalancing of divisions for a new season
type RebalanceManager struct {
	stores *stores.Stores
}

// NewRebalanceManager creates a new rebalance manager
func NewRebalanceManager(allStores *stores.Stores) *RebalanceManager {
	return &RebalanceManager{
		stores: allStores,
	}
}

// PlayerWithVirtualDiv represents a player with their assigned virtual division
type PlayerWithVirtualDiv struct {
	UserID               string // UUID string for external use
	UserDBID             int32  // Database ID for internal queries
	Username             string // Username for logging
	VirtualDivision      int32
	PlacementStatus      ipc.PlacementStatus
	PreviousDivisionSize int
	PreviousRank         int32
	HiatusSeasons        int32
	Rating               int
	RegistrationRow      models.LeagueRegistration
}

// PlayerWithPriority extends PlayerWithVirtualDiv with priority score
type PlayerWithPriority struct {
	PlayerWithVirtualDiv
	PriorityScore float64
}

// RebalanceResult tracks the outcome of rebalancing
type RebalanceResult struct {
	DivisionsCreated int
	PlayersAssigned  int
	FinalDivMerged   bool
	VirtualDivisions map[string]int32 // UserID -> virtual division
	FinalDivisions   map[string]int32 // UserID -> real division
}

// RebalanceDivisions orchestrates the complete rebalancing process
func (rm *RebalanceManager) RebalanceDivisions(
	ctx context.Context,
	leagueID uuid.UUID,
	previousSeasonID uuid.UUID,
	newSeasonID uuid.UUID,
	newSeasonNumber int32,
	categorizedPlayers []CategorizedPlayer,
	idealDivisionSize int32,
) (*RebalanceResult, error) {
	result := &RebalanceResult{
		VirtualDivisions: make(map[string]int32),
		FinalDivisions:   make(map[string]int32),
	}

	if len(categorizedPlayers) == 0 {
		return result, nil
	}

	// Step 1: Update placement statuses (NEW/GRADUATED/HIATUS/etc.)
	err := rm.UpdatePlacementStatuses(ctx, leagueID, previousSeasonID, newSeasonID, newSeasonNumber, categorizedPlayers)
	if err != nil {
		return nil, fmt.Errorf("failed to update placement statuses: %w", err)
	}

	// Step 2: Assign virtual divisions to all players
	playersWithVirtualDivs, err := rm.AssignVirtualDivisions(ctx, leagueID, previousSeasonID, newSeasonID, categorizedPlayers)
	if err != nil {
		return nil, fmt.Errorf("failed to assign virtual divisions: %w", err)
	}

	// Store virtual divisions for result
	for _, p := range playersWithVirtualDivs {
		result.VirtualDivisions[p.UserID] = p.VirtualDivision
	}

	// Calculate number of virtual divisions
	numVirtualDivs := int32(0)
	for _, p := range playersWithVirtualDivs {
		if p.VirtualDivision > numVirtualDivs {
			numVirtualDivs = p.VirtualDivision
		}
	}

	// Step 3: Calculate priority scores
	playersWithPriority := rm.CalculatePriorityScores(playersWithVirtualDivs, numVirtualDivs, newSeasonNumber)

	// Step 4: Determine number of divisions to create
	numDivisions := int(math.Round(float64(len(playersWithPriority)) / float64(idealDivisionSize)))
	if numDivisions < 1 {
		numDivisions = 1
	}

	// Step 5: Create divisions and assign players
	err = rm.CreateDivisionsAndAssign(ctx, newSeasonID, playersWithPriority, numDivisions, idealDivisionSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create divisions and assign players: %w", err)
	}

	result.DivisionsCreated = numDivisions
	result.PlayersAssigned = len(playersWithPriority)

	// Step 6: Merge undersized final division if needed
	merged, err := rm.MergeUndersizedFinalDivision(ctx, newSeasonID, numDivisions)
	if err != nil {
		return nil, fmt.Errorf("failed to merge undersized final division: %w", err)
	}
	result.FinalDivMerged = merged
	if merged {
		result.DivisionsCreated--
	}

	// Get final division assignments for result
	for _, p := range playersWithPriority {
		reg, err := rm.stores.LeagueStore.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
			SeasonID: newSeasonID,
			UserID:   p.UserDBID,
		})
		if err == nil && reg.DivisionID.Valid {
			// Get division number
			div, err := rm.stores.LeagueStore.GetDivision(ctx, reg.DivisionID.Bytes)
			if err == nil {
				result.FinalDivisions[p.UserID] = div.DivisionNumber
			}
		}
	}

	return result, nil
}

// UpdatePlacementStatuses sets placement_status for all players before rebalancing
// This includes NEW, GRADUATED, SHORT_HIATUS_RETURNING, LONG_HIATUS_RETURNING
// (PROMOTED/RELEGATED/STAYED are already set by MarkSeasonOutcomes at end of previous season)
func (rm *RebalanceManager) UpdatePlacementStatuses(
	ctx context.Context,
	leagueID uuid.UUID,
	previousSeasonID uuid.UUID,
	newSeasonID uuid.UUID,
	newSeasonNumber int32,
	categorizedPlayers []CategorizedPlayer,
) error {
	for _, player := range categorizedPlayers {
		if player.Category == PlayerCategoryNew {
			// Brand new player
			err := rm.stores.LeagueStore.UpdatePlacementStatusWithSeasonsAway(ctx, models.UpdatePlacementStatusWithSeasonsAwayParams{
				UserID:               player.Registration.UserID,
				PlacementStatus:      pgtype.Int4{Int32: int32(ipc.PlacementStatus_PLACEMENT_NEW), Valid: true},
				PreviousDivisionRank: pgtype.Int4{Int32: 0, Valid: false},
				SeasonsAway:          pgtype.Int4{Int32: 0, Valid: true},
				SeasonID:             newSeasonID,
			})
			if err != nil {
				return fmt.Errorf("failed to set NEW status for %d: %w", player.Registration.UserID, err)
			}
			continue
		}

		// RETURNING player - get their history
		history, err := rm.stores.LeagueStore.GetPlayerSeasonHistory(ctx, models.GetPlayerSeasonHistoryParams{
			UserID:   player.Registration.UserID,
			LeagueID: leagueID,
		})
		if err != nil {
			return fmt.Errorf("failed to get history for %d: %w", player.Registration.UserID, err)
		}

		// Find most recent season (not the new one)
		var lastSeason models.GetPlayerSeasonHistoryRow
		found := false
		for _, h := range history {
			if h.SeasonID != newSeasonID {
				lastSeason = h
				found = true
				break
			}
		}

		if !found {
			// No history found, treat as new
			err := rm.stores.LeagueStore.UpdatePlacementStatusWithSeasonsAway(ctx, models.UpdatePlacementStatusWithSeasonsAwayParams{
				UserID:               player.Registration.UserID,
				PlacementStatus:      pgtype.Int4{Int32: int32(ipc.PlacementStatus_PLACEMENT_NEW), Valid: true},
				PreviousDivisionRank: pgtype.Int4{Int32: 0, Valid: false},
				SeasonsAway:          pgtype.Int4{Int32: 0, Valid: true},
				SeasonID:             newSeasonID,
			})
			if err != nil {
				return fmt.Errorf("failed to set NEW status for %d: %w", player.Registration.UserID, err)
			}
			continue
		}

		// Calculate hiatus: newSeasonNumber - lastPlayedSeasonNumber - 1
		seasonsAway := newSeasonNumber - lastSeason.SeasonNumber - 1

		if seasonsAway > 0 {
			// Returning from hiatus
			var status ipc.PlacementStatus
			if seasonsAway >= 1 && seasonsAway <= 3 {
				status = ipc.PlacementStatus_PLACEMENT_SHORT_HIATUS_RETURNING
			} else {
				status = ipc.PlacementStatus_PLACEMENT_LONG_HIATUS_RETURNING
			}

			err := rm.stores.LeagueStore.UpdatePlacementStatusWithSeasonsAway(ctx, models.UpdatePlacementStatusWithSeasonsAwayParams{
				UserID:               player.Registration.UserID,
				PlacementStatus:      pgtype.Int4{Int32: int32(status), Valid: true},
				PreviousDivisionRank: lastSeason.PreviousDivisionRank,
				SeasonsAway:          pgtype.Int4{Int32: seasonsAway, Valid: true},
				SeasonID:             newSeasonID,
			})
			if err != nil {
				return fmt.Errorf("failed to set %s status for %d: %w", status, player.Registration.UserID, err)
			}
			continue
		}

		// Consecutive play - copy status from last season
		// (PROMOTED/RELEGATED/STAYED already set by MarkSeasonOutcomes)
		err = rm.stores.LeagueStore.UpdatePlacementStatusWithSeasonsAway(ctx, models.UpdatePlacementStatusWithSeasonsAwayParams{
			UserID:               player.Registration.UserID,
			PlacementStatus:      lastSeason.PlacementStatus,
			PreviousDivisionRank: lastSeason.PreviousDivisionRank,
			SeasonsAway:          pgtype.Int4{Int32: 0, Valid: true},
			SeasonID:             newSeasonID,
		})
		if err != nil {
			return fmt.Errorf("failed to copy status for %d: %w", player.Registration.UserID, err)
		}
	}

	return nil
}

// AssignVirtualDivisions assigns virtual divisions to all players based on their outcomes
func (rm *RebalanceManager) AssignVirtualDivisions(
	ctx context.Context,
	leagueID uuid.UUID,
	previousSeasonID uuid.UUID,
	newSeasonID uuid.UUID,
	categorizedPlayers []CategorizedPlayer,
) ([]PlayerWithVirtualDiv, error) {
	players := []PlayerWithVirtualDiv{}

	// Fetch all registrations for the season at once to avoid N+1 queries
	allRegistrations, err := rm.stores.LeagueStore.GetSeasonRegistrations(ctx, newSeasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get season registrations: %w", err)
	}

	// Create a map for O(1) lookup by user_id
	registrationMap := make(map[int32]models.GetSeasonRegistrationsRow)
	for _, reg := range allRegistrations {
		registrationMap[reg.UserID] = reg
	}

	// Get registrations to access updated placement_status
	for _, catPlayer := range categorizedPlayers {
		regRow, exists := registrationMap[catPlayer.Registration.UserID]
		if !exists {
			return nil, fmt.Errorf("registration not found for user %d in season", catPlayer.Registration.UserID)
		}

		// Convert GetSeasonRegistrationsRow to LeagueRegistration
		reg := models.LeagueRegistration{
			ID:                   regRow.ID,
			UserID:               regRow.UserID,
			SeasonID:             regRow.SeasonID,
			DivisionID:           regRow.DivisionID,
			RegistrationDate:     regRow.RegistrationDate,
			FirstsCount:          regRow.FirstsCount,
			Status:               regRow.Status,
			PlacementStatus:      regRow.PlacementStatus,
			PreviousDivisionRank: regRow.PreviousDivisionRank,
			SeasonsAway:          regRow.SeasonsAway,
			CreatedAt:            regRow.CreatedAt,
			UpdatedAt:            regRow.UpdatedAt,
		}

		status := ipc.PlacementStatus_PLACEMENT_NONE
		if reg.PlacementStatus.Valid {
			status = ipc.PlacementStatus(reg.PlacementStatus.Int32)
		}

		player := PlayerWithVirtualDiv{
			UserID:               catPlayer.Registration.UserUuid.String, // UUID string from JOIN
			UserDBID:             catPlayer.Registration.UserID,          // Database ID
			Username:             catPlayer.Registration.Username.String, // Username from JOIN
			PlacementStatus:      status,
			PreviousRank:         0,
			HiatusSeasons:        0,
			PreviousDivisionSize: 0,
			Rating:               int(catPlayer.Rating),
			RegistrationRow:      reg,
		}

		if reg.PreviousDivisionRank.Valid {
			player.PreviousRank = reg.PreviousDivisionRank.Int32
		}
		if reg.SeasonsAway.Valid {
			player.HiatusSeasons = reg.SeasonsAway.Int32
		}

		// Get previous division size (if applicable)
		if status != ipc.PlacementStatus_PLACEMENT_NEW {
			history, err := rm.stores.LeagueStore.GetPlayerSeasonHistory(ctx, models.GetPlayerSeasonHistoryParams{
				UserID:   catPlayer.Registration.UserID,
				LeagueID: leagueID,
			})
			if err == nil && len(history) > 0 {
				// Find most recent season
				for _, h := range history {
					if h.SeasonID != newSeasonID && h.DivisionID.Valid {
						// Count players in that division
						standings, err := rm.stores.LeagueStore.GetStandings(ctx, h.DivisionID.Bytes)
						if err == nil {
							player.PreviousDivisionSize = len(standings)
						}
						break
					}
				}
			}
		}

		players = append(players, player)
	}

	// Now assign virtual divisions based on status
	return rm.calculateVirtualDivisions(ctx, leagueID, previousSeasonID, players)
}

// calculateVirtualDivisions determines virtual division for each player
func (rm *RebalanceManager) calculateVirtualDivisions(
	ctx context.Context,
	leagueID uuid.UUID,
	previousSeasonID uuid.UUID,
	players []PlayerWithVirtualDiv,
) ([]PlayerWithVirtualDiv, error) {
	// Get previous season divisions to determine virtual division structure
	prevDivisions, err := rm.stores.LeagueStore.GetDivisionsBySeason(ctx, previousSeasonID)
	if err != nil && previousSeasonID != uuid.Nil {
		return nil, fmt.Errorf("failed to get previous season divisions: %w", err)
	}

	// Sort divisions by division number
	sort.Slice(prevDivisions, func(i, j int) bool {
		return prevDivisions[i].DivisionNumber < prevDivisions[j].DivisionNumber
	})

	highestPrevDivNumber := int32(0)
	if len(prevDivisions) > 0 {
		highestPrevDivNumber = prevDivisions[len(prevDivisions)-1].DivisionNumber
	}

	// Assign virtual divisions for all players based on their status
	result := make([]PlayerWithVirtualDiv, len(players))

	for i, p := range players {
		// Get their previous division number
		history, err := rm.stores.LeagueStore.GetPlayerSeasonHistory(ctx, models.GetPlayerSeasonHistoryParams{
			UserID:   p.UserDBID,
			LeagueID: leagueID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get history for %s: %w", p.UserID, err)
		}

		prevDivNumber := int32(1) // Default
		for _, h := range history {
			if h.DivisionID.Valid {
				// Look up the division to get its number
				div, err := rm.stores.LeagueStore.GetDivision(ctx, h.DivisionID.Bytes)
				if err == nil {
					prevDivNumber = div.DivisionNumber
				}
				break
			}
		}

		// Copy player data to result
		result[i] = p

		// Apply outcome to determine virtual division
		switch p.PlacementStatus {
		case ipc.PlacementStatus_PLACEMENT_PROMOTED:
			result[i].VirtualDivision = prevDivNumber - 1
			if result[i].VirtualDivision < 1 {
				result[i].VirtualDivision = 1
			}
		case ipc.PlacementStatus_PLACEMENT_RELEGATED:
			result[i].VirtualDivision = prevDivNumber + 1
		case ipc.PlacementStatus_PLACEMENT_NEW:
			// New players start at the highest division and low priority pushes them down
			if highestPrevDivNumber > 0 {
				result[i].VirtualDivision = highestPrevDivNumber
			} else {
				result[i].VirtualDivision = 1
			}
		case ipc.PlacementStatus_PLACEMENT_STAYED, ipc.PlacementStatus_PLACEMENT_SHORT_HIATUS_RETURNING, ipc.PlacementStatus_PLACEMENT_LONG_HIATUS_RETURNING:
			result[i].VirtualDivision = prevDivNumber
		default:
			result[i].VirtualDivision = prevDivNumber
		}
	}

	return result, nil
}

// CalculatePriorityScores calculates priority scores for all players
func (rm *RebalanceManager) CalculatePriorityScores(
	players []PlayerWithVirtualDiv,
	numVirtualDivs int32,
	seasonNumber int32,
) []PlayerWithPriority {
	result := make([]PlayerWithPriority, len(players))

	for i, p := range players {
		// Calculate priority bonus based on status
		var priorityBonus int64
		switch p.PlacementStatus {
		case ipc.PlacementStatus_PLACEMENT_STAYED:
			priorityBonus = PriorityBonusStayed
		case ipc.PlacementStatus_PLACEMENT_PROMOTED:
			priorityBonus = PriorityBonusPromoted
		case ipc.PlacementStatus_PLACEMENT_RELEGATED:
			priorityBonus = PriorityBonusRelegated
		case ipc.PlacementStatus_PLACEMENT_SHORT_HIATUS_RETURNING, ipc.PlacementStatus_PLACEMENT_LONG_HIATUS_RETURNING:
			priorityBonus = PriorityBonusHiatusReturning
		case ipc.PlacementStatus_PLACEMENT_NEW:
			priorityBonus = PriorityBonusNew
		default:
			priorityBonus = 0
		}

		// Calculate weight based on hiatus
		weight := 1.0
		if p.HiatusSeasons > 0 {
			weight = math.Pow(HiatusWeightBase, float64(p.HiatusSeasons))
		}

		// Calculate rank component (num_players_in_previous_division - our_rank_in_previous_division)
		rankComponent := int64(0)
		if p.PreviousDivisionSize > 0 && p.PreviousRank > 0 {
			rankComponent = int64(p.PreviousDivisionSize) - int64(p.PreviousRank)
		}

		// Calculate priority score
		// score = ((DivMultiplier * (num_virtual_divs - our_virtual_div)) + PriorityBonus + rankComponent) * Weight
		score := float64(
			(int64(DivisionMultiplier)*int64(numVirtualDivs-p.VirtualDivision))+
				priorityBonus+
				rankComponent,
		) * weight

		// For NEW players in Season 1 ONLY, add their rating to prioritize by skill
		// This ensures that in the initial season, higher-rated players are placed in higher divisions
		if p.PlacementStatus == ipc.PlacementStatus_PLACEMENT_NEW && seasonNumber == 1 {
			score += float64(p.Rating)
		}

		result[i] = PlayerWithPriority{
			PlayerWithVirtualDiv: p,
			PriorityScore:        score,
		}
	}

	// Sort by priority score (descending - highest priority first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].PriorityScore > result[j].PriorityScore
	})

	return result
}

// CreateDivisionsAndAssign creates real divisions and assigns players sequentially
func (rm *RebalanceManager) CreateDivisionsAndAssign(
	ctx context.Context,
	seasonID uuid.UUID,
	playersWithPriority []PlayerWithPriority,
	numDivisions int,
	idealDivisionSize int32,
) error {
	// Create divisions 1, 2, 3, ..., numDivisions
	createdDivisions := []models.LeagueDivision{}
	manualMgr := NewManualDivisionManager(rm.stores)

	for divNum := 1; divNum <= numDivisions; divNum++ {
		div, err := manualMgr.CreateDivision(ctx, seasonID, int32(divNum), "")
		if err != nil {
			return fmt.Errorf("failed to create division %d: %w", divNum, err)
		}
		createdDivisions = append(createdDivisions, div)
	}

	// Assign players sequentially based on ideal division size
	for i, player := range playersWithPriority {
		// Determine which division this player goes to
		divIndex := i / int(idealDivisionSize)
		if divIndex >= len(createdDivisions) {
			divIndex = len(createdDivisions) - 1 // Put overflow in last division
		}

		targetDiv := createdDivisions[divIndex]

		// Log player placement with priority score details
		log.Info().
			Str("username", player.Username).
			Int("assignmentOrder", i+1).
			Int32("divisionNumber", targetDiv.DivisionNumber).
			Float64("priorityScore", player.PriorityScore).
			Int32("virtualDivision", player.VirtualDivision).
			Str("placementStatus", player.PlacementStatus.String()).
			Int32("previousRank", player.PreviousRank).
			Int("previousDivisionSize", player.PreviousDivisionSize).
			Int32("hiatusSeasons", player.HiatusSeasons).
			Int("rating", player.Rating).
			Msg("Player division assignment")

		// Assign player to this division
		err := rm.stores.LeagueStore.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      player.UserDBID,
			SeasonID:    seasonID,
			DivisionID:  pgtype.UUID{Bytes: targetDiv.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to assign player %s to division %d: %w", player.UserID, targetDiv.DivisionNumber, err)
		}
	}

	return nil
}

// MergeUndersizedFinalDivision merges the final division into the second-to-last if it's too small
func (rm *RebalanceManager) MergeUndersizedFinalDivision(
	ctx context.Context,
	seasonID uuid.UUID,
	numDivisions int,
) (bool, error) {
	if numDivisions <= 1 {
		return false, nil // Can't merge if only one division
	}

	// Get all divisions
	divisions, err := rm.stores.LeagueStore.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return false, fmt.Errorf("failed to get divisions: %w", err)
	}

	// Sort divisions by number
	sort.Slice(divisions, func(i, j int) bool {
		return divisions[i].DivisionNumber < divisions[j].DivisionNumber
	})

	if len(divisions) <= 1 {
		return false, nil
	}

	// Get last division
	lastDiv := divisions[len(divisions)-1]
	secondToLast := divisions[len(divisions)-2]

	// Count players in last division
	lastDivPlayers, err := rm.stores.LeagueStore.GetDivisionRegistrations(ctx, lastDiv.Uuid)
	if err != nil {
		return false, fmt.Errorf("failed to get last division players: %w", err)
	}

	if len(lastDivPlayers) < MinimumFinalDivSize {
		// Merge into second-to-last division
		manualMgr := NewManualDivisionManager(rm.stores)
		_, err := manualMgr.MergeDivisions(ctx, seasonID, secondToLast.Uuid, lastDiv.Uuid)
		if err != nil {
			return false, fmt.Errorf("failed to merge divisions: %w", err)
		}
		return true, nil
	}

	return false, nil
}
