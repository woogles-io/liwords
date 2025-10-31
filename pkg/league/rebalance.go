package league

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

const (
	// Priority score constants
	DivisionMultiplier = 1_000_000

	PriorityBonusStayed           = 500_000
	PriorityBonusPromoted         = 400_000
	PriorityBonusRelegated        = 300_000
	PriorityBonusGraduated        = 50_000
	PriorityBonusHiatusReturning  = 5_000
	PriorityBonusNew              = 0

	// Hiatus weight: 0.933^N (halves every ~10 seasons)
	HiatusWeightBase = 0.933

	// Division sizing
	IdealDivisionSize    = 15
	MinimumFinalDivSize  = 12
)

// RebalanceManager handles the rebalancing of divisions for a new season
type RebalanceManager struct {
	store league.Store
}

// NewRebalanceManager creates a new rebalance manager
func NewRebalanceManager(store league.Store) *RebalanceManager {
	return &RebalanceManager{
		store: store,
	}
}

// PlayerWithVirtualDiv represents a player with their assigned virtual division
type PlayerWithVirtualDiv struct {
	UserID              string
	VirtualDivision     int32
	PlacementStatus     string
	PreviousDivisionSize int
	PreviousRank        int32
	HiatusSeasons       int32
	Rating              int
	RegistrationRow     models.LeagueRegistration
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
	VirtualDivisions map[string]int32  // UserID -> virtual division
	FinalDivisions   map[string]int32  // UserID -> real division
}

// RebalanceDivisions orchestrates the complete rebalancing process
func (rm *RebalanceManager) RebalanceDivisions(
	ctx context.Context,
	leagueID uuid.UUID,
	previousSeasonID uuid.UUID,
	newSeasonID uuid.UUID,
	newSeasonNumber int32,
	categorizedPlayers []CategorizedPlayer,
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
	playersWithPriority := rm.CalculatePriorityScores(playersWithVirtualDivs, numVirtualDivs)

	// Step 4: Determine number of divisions to create
	numDivisions := int(math.Round(float64(len(playersWithPriority)) / float64(IdealDivisionSize)))
	if numDivisions < 1 {
		numDivisions = 1
	}

	// Step 5: Create divisions and assign players
	err = rm.CreateDivisionsAndAssign(ctx, newSeasonID, playersWithPriority, numDivisions)
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
		reg, err := rm.store.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
			SeasonID: newSeasonID,
			UserID:   p.UserID,
		})
		if err == nil && reg.DivisionID.Valid {
			// Get division number
			div, err := rm.store.GetDivision(ctx, reg.DivisionID.Bytes)
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
			err := rm.store.UpdatePlacementStatusWithSeasonsAway(ctx, models.UpdatePlacementStatusWithSeasonsAwayParams{
				UserID:               player.Registration.UserID,
				PlacementStatus:      pgtype.Text{String: "NEW", Valid: true},
				PreviousDivisionRank: pgtype.Int4{Int32: 0, Valid: false},
				SeasonsAway:          pgtype.Int4{Int32: 0, Valid: true},
				SeasonID:             newSeasonID,
			})
			if err != nil {
				return fmt.Errorf("failed to set NEW status for %s: %w", player.Registration.UserID, err)
			}
			continue
		}

		// RETURNING player - get their history
		history, err := rm.store.GetPlayerSeasonHistory(ctx, models.GetPlayerSeasonHistoryParams{
			UserID:   player.Registration.UserID,
			LeagueID: leagueID,
		})
		if err != nil {
			return fmt.Errorf("failed to get history for %s: %w", player.Registration.UserID, err)
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
			err := rm.store.UpdatePlacementStatusWithSeasonsAway(ctx, models.UpdatePlacementStatusWithSeasonsAwayParams{
				UserID:               player.Registration.UserID,
				PlacementStatus:      pgtype.Text{String: "NEW", Valid: true},
				PreviousDivisionRank: pgtype.Int4{Int32: 0, Valid: false},
				SeasonsAway:          pgtype.Int4{Int32: 0, Valid: true},
				SeasonID:             newSeasonID,
			})
			if err != nil {
				return fmt.Errorf("failed to set NEW status for %s: %w", player.Registration.UserID, err)
			}
			continue
		}

		// Check if from rookie division
		isRookieGraduate := false
		if lastSeason.DivisionID.Valid {
			// Look up the division to get its number
			div, err := rm.store.GetDivision(ctx, lastSeason.DivisionID.Bytes)
			if err == nil {
				isRookieGraduate = div.DivisionNumber >= RookieDivisionNumberBase
			}
		}

		if isRookieGraduate {
			// Rookie graduating to regular divisions
			err := rm.store.UpdatePlacementStatusWithSeasonsAway(ctx, models.UpdatePlacementStatusWithSeasonsAwayParams{
				UserID:               player.Registration.UserID,
				PlacementStatus:      pgtype.Text{String: "GRADUATED", Valid: true},
				PreviousDivisionRank: lastSeason.PreviousDivisionRank,
				SeasonsAway:          pgtype.Int4{Int32: 0, Valid: true},
				SeasonID:             newSeasonID,
			})
			if err != nil {
				return fmt.Errorf("failed to set GRADUATED status for %s: %w", player.Registration.UserID, err)
			}
			continue
		}

		// Calculate hiatus: newSeasonNumber - lastPlayedSeasonNumber - 1
		seasonsAway := newSeasonNumber - lastSeason.SeasonNumber - 1

		if seasonsAway > 0 {
			// Returning from hiatus
			var status string
			if seasonsAway >= 1 && seasonsAway <= 3 {
				status = "SHORT_HIATUS_RETURNING"
			} else {
				status = "LONG_HIATUS_RETURNING"
			}

			err := rm.store.UpdatePlacementStatusWithSeasonsAway(ctx, models.UpdatePlacementStatusWithSeasonsAwayParams{
				UserID:               player.Registration.UserID,
				PlacementStatus:      pgtype.Text{String: status, Valid: true},
				PreviousDivisionRank: lastSeason.PreviousDivisionRank,
				SeasonsAway:          pgtype.Int4{Int32: seasonsAway, Valid: true},
				SeasonID:             newSeasonID,
			})
			if err != nil {
				return fmt.Errorf("failed to set %s status for %s: %w", status, player.Registration.UserID, err)
			}
			continue
		}

		// Consecutive play - copy status from last season
		// (PROMOTED/RELEGATED/STAYED already set by MarkSeasonOutcomes)
		err = rm.store.UpdatePlacementStatusWithSeasonsAway(ctx, models.UpdatePlacementStatusWithSeasonsAwayParams{
			UserID:               player.Registration.UserID,
			PlacementStatus:      lastSeason.PlacementStatus,
			PreviousDivisionRank: lastSeason.PreviousDivisionRank,
			SeasonsAway:          pgtype.Int4{Int32: 0, Valid: true},
			SeasonID:             newSeasonID,
		})
		if err != nil {
			return fmt.Errorf("failed to copy status for %s: %w", player.Registration.UserID, err)
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

	// Get registrations to access updated placement_status
	for _, catPlayer := range categorizedPlayers {
		reg, err := rm.store.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
			SeasonID: newSeasonID,
			UserID:   catPlayer.Registration.UserID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get registration for %s: %w", catPlayer.Registration.UserID, err)
		}

		status := ""
		if reg.PlacementStatus.Valid {
			status = reg.PlacementStatus.String
		}

		player := PlayerWithVirtualDiv{
			UserID:              catPlayer.Registration.UserID,
			PlacementStatus:     status,
			PreviousRank:        0,
			HiatusSeasons:       0,
			PreviousDivisionSize: 0,
			Rating:              int(catPlayer.Rating),
			RegistrationRow:     reg,
		}

		if reg.PreviousDivisionRank.Valid {
			player.PreviousRank = reg.PreviousDivisionRank.Int32
		}
		if reg.SeasonsAway.Valid {
			player.HiatusSeasons = reg.SeasonsAway.Int32
		}

		// Get previous division size (if applicable)
		if status != "NEW" {
			history, err := rm.store.GetPlayerSeasonHistory(ctx, models.GetPlayerSeasonHistoryParams{
				UserID:   catPlayer.Registration.UserID,
				LeagueID: leagueID,
			})
			if err == nil && len(history) > 0 {
				// Find most recent season
				for _, h := range history {
					if h.SeasonID != newSeasonID && h.DivisionID.Valid {
						// Count players in that division
						standings, err := rm.store.GetStandings(ctx, h.DivisionID.Bytes)
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
	prevDivisions, err := rm.store.GetDivisionsBySeason(ctx, previousSeasonID)
	if err != nil && previousSeasonID != uuid.Nil {
		return nil, fmt.Errorf("failed to get previous season divisions: %w", err)
	}

	// Filter to regular divisions
	prevRegularDivs := []models.LeagueDivision{}
	for _, div := range prevDivisions {
		if div.DivisionNumber < RookieDivisionNumberBase {
			prevRegularDivs = append(prevRegularDivs, div)
		}
	}

	// Sort by division number
	sort.Slice(prevRegularDivs, func(i, j int) bool {
		return prevRegularDivs[i].DivisionNumber < prevRegularDivs[j].DivisionNumber
	})

	highestPrevDivNumber := int32(0)
	if len(prevRegularDivs) > 0 {
		highestPrevDivNumber = prevRegularDivs[len(prevRegularDivs)-1].DivisionNumber
	}

	// Separate players by status
	regularPlayers := []PlayerWithVirtualDiv{}
	graduates := []PlayerWithVirtualDiv{}
	newPlayers := []PlayerWithVirtualDiv{}

	for _, p := range players {
		switch p.PlacementStatus {
		case "GRADUATED":
			graduates = append(graduates, p)
		case "NEW":
			newPlayers = append(newPlayers, p)
		default:
			regularPlayers = append(regularPlayers, p)
		}
	}

	// Assign virtual divisions for regular players (PROMOTED/RELEGATED/STAYED/HIATUS)
	for i, p := range regularPlayers {
		// Get their previous division number
		history, err := rm.store.GetPlayerSeasonHistory(ctx, models.GetPlayerSeasonHistoryParams{
			UserID:   p.UserID,
			LeagueID: leagueID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get history for %s: %w", p.UserID, err)
		}

		prevDivNumber := int32(1) // Default
		for _, h := range history {
			if h.DivisionID.Valid {
				// Look up the division to get its number
				div, err := rm.store.GetDivision(ctx, h.DivisionID.Bytes)
				if err == nil {
					prevDivNumber = div.DivisionNumber
				}
				break
			}
		}

		// Apply outcome
		switch p.PlacementStatus {
		case "PROMOTED":
			regularPlayers[i].VirtualDivision = prevDivNumber - 1
			if regularPlayers[i].VirtualDivision < 1 {
				regularPlayers[i].VirtualDivision = 1
			}
		case "RELEGATED":
			regularPlayers[i].VirtualDivision = prevDivNumber + 1
		case "STAYED", "SHORT_HIATUS_RETURNING", "LONG_HIATUS_RETURNING":
			regularPlayers[i].VirtualDivision = prevDivNumber
		default:
			regularPlayers[i].VirtualDivision = prevDivNumber
		}
	}

	// Assign virtual divisions for graduates using graduation formula
	if len(graduates) > 0 {
		// Get rookie standings from previous season
		rookieStandings := []models.LeagueStanding{}
		for _, div := range prevDivisions {
			if div.DivisionNumber >= RookieDivisionNumberBase {
				standings, err := rm.store.GetStandings(ctx, div.Uuid)
				if err != nil {
					return nil, fmt.Errorf("failed to get rookie standings: %w", err)
				}
				rookieStandings = append(rookieStandings, standings...)
			}
		}

		// Sort by rank
		sort.Slice(rookieStandings, func(i, j int) bool {
			if rookieStandings[i].Rank.Valid && rookieStandings[j].Rank.Valid {
				return rookieStandings[i].Rank.Int32 < rookieStandings[j].Rank.Int32
			}
			return false
		})

		// Use graduation formula
		graduationMgr := NewGraduationManager(rm.store)
		groups := graduationMgr.calculateGraduationGroups(rookieStandings, highestPrevDivNumber)

		// Create map of userID -> virtual division
		graduateVirtualDivs := make(map[string]int32)
		for _, group := range groups {
			for _, standing := range group.Rookies {
				graduateVirtualDivs[standing.UserID] = group.TargetDivision
			}
		}

		// Assign virtual divisions
		for i, p := range graduates {
			if vDiv, exists := graduateVirtualDivs[p.UserID]; exists {
				graduates[i].VirtualDivision = vDiv
			} else {
				// Default to highest division if not found
				graduates[i].VirtualDivision = highestPrevDivNumber
				if graduates[i].VirtualDivision < 1 {
					graduates[i].VirtualDivision = 1
				}
			}
		}
	}

	// Assign virtual divisions for new players (<10 new rookies placed directly)
	if len(newPlayers) > 0 {
		numVirtualDivs := highestPrevDivNumber
		if numVirtualDivs < 1 {
			numVirtualDivs = 1
		}

		// Sort by rating (highest first)
		sort.Slice(newPlayers, func(i, j int) bool {
			return newPlayers[i].Rating > newPlayers[j].Rating
		})

		if numVirtualDivs == 1 {
			// All go to Division 1
			for i := range newPlayers {
				newPlayers[i].VirtualDivision = 1
			}
		} else {
			// Split: top half -> second-to-bottom, bottom half -> bottom
			midpoint := len(newPlayers) / 2
			secondBottom := numVirtualDivs - 1
			bottom := numVirtualDivs

			for i := 0; i < midpoint; i++ {
				newPlayers[i].VirtualDivision = secondBottom
			}
			for i := midpoint; i < len(newPlayers); i++ {
				newPlayers[i].VirtualDivision = bottom
			}
		}
	}

	// Combine all players
	result := append(regularPlayers, graduates...)
	result = append(result, newPlayers...)

	return result, nil
}

// CalculatePriorityScores calculates priority scores for all players
func (rm *RebalanceManager) CalculatePriorityScores(
	players []PlayerWithVirtualDiv,
	numVirtualDivs int32,
) []PlayerWithPriority {
	result := make([]PlayerWithPriority, len(players))

	for i, p := range players {
		// Calculate priority bonus based on status
		var priorityBonus int64
		switch p.PlacementStatus {
		case "STAYED":
			priorityBonus = PriorityBonusStayed
		case "PROMOTED":
			priorityBonus = PriorityBonusPromoted
		case "RELEGATED":
			priorityBonus = PriorityBonusRelegated
		case "GRADUATED":
			priorityBonus = PriorityBonusGraduated
		case "SHORT_HIATUS_RETURNING", "LONG_HIATUS_RETURNING":
			priorityBonus = PriorityBonusHiatusReturning
		case "NEW":
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
			(int64(DivisionMultiplier) * int64(numVirtualDivs - p.VirtualDivision)) +
			priorityBonus +
			rankComponent,
		) * weight

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
) error {
	// Create divisions 1, 2, 3, ..., numDivisions
	createdDivisions := []models.LeagueDivision{}
	manualMgr := NewManualDivisionManager(rm.store)

	for divNum := 1; divNum <= numDivisions; divNum++ {
		div, err := manualMgr.CreateDivision(ctx, seasonID, int32(divNum), "")
		if err != nil {
			return fmt.Errorf("failed to create division %d: %w", divNum, err)
		}
		createdDivisions = append(createdDivisions, div)
	}

	// Assign players sequentially: 15 per division
	for i, player := range playersWithPriority {
		// Determine which division this player goes to
		divIndex := i / IdealDivisionSize
		if divIndex >= len(createdDivisions) {
			divIndex = len(createdDivisions) - 1 // Put overflow in last division
		}

		targetDiv := createdDivisions[divIndex]

		// Assign player to this division
		err := rm.store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      player.UserID,
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
	divisions, err := rm.store.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return false, fmt.Errorf("failed to get divisions: %w", err)
	}

	// Filter to regular divisions and sort by number
	regularDivs := []models.LeagueDivision{}
	for _, div := range divisions {
		if div.DivisionNumber < RookieDivisionNumberBase {
			regularDivs = append(regularDivs, div)
		}
	}

	sort.Slice(regularDivs, func(i, j int) bool {
		return regularDivs[i].DivisionNumber < regularDivs[j].DivisionNumber
	})

	if len(regularDivs) <= 1 {
		return false, nil
	}

	// Get last division
	lastDiv := regularDivs[len(regularDivs)-1]
	secondToLast := regularDivs[len(regularDivs)-2]

	// Count players in last division
	lastDivPlayers, err := rm.store.GetDivisionRegistrations(ctx, lastDiv.Uuid)
	if err != nil {
		return false, fmt.Errorf("failed to get last division players: %w", err)
	}

	if len(lastDivPlayers) < MinimumFinalDivSize {
		// Merge into second-to-last division
		manualMgr := NewManualDivisionManager(rm.store)
		_, err := manualMgr.MergeDivisions(ctx, seasonID, secondToLast.Uuid, lastDiv.Uuid)
		if err != nil {
			return false, fmt.Errorf("failed to merge divisions: %w", err)
		}
		return true, nil
	}

	return false, nil
}
