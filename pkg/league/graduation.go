package league

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// GraduationManager handles graduating rookies from rookie divisions into regular divisions
type GraduationManager struct {
	store league.Store
}

// NewGraduationManager creates a new graduation manager
func NewGraduationManager(store league.Store) *GraduationManager {
	return &GraduationManager{
		store: store,
	}
}

// GraduationResult tracks the outcome of graduating rookies
type GraduationResult struct {
	// Rookies that were graduated and placed into regular divisions
	GraduatedRookies []PlacedPlayer
}

// GraduationGroup represents a group of rookies being placed into the same division
type GraduationGroup struct {
	StartRank      int                  // First rookie rank in this group (1-based)
	EndRank        int                  // Last rookie rank in this group (1-based)
	TargetDivision int32                // Division number to place this group
	Rookies        []models.LeagueStanding
}

// GraduateRookies places rookies from previous season's rookie divisions
// into the new season's regular divisions based on their final standings
func (gm *GraduationManager) GraduateRookies(
	ctx context.Context,
	previousSeasonID uuid.UUID,
	newSeasonID uuid.UUID,
) (*GraduationResult, error) {
	result := &GraduationResult{
		GraduatedRookies: []PlacedPlayer{},
	}

	// Get all rookie divisions from previous season
	prevDivisions, err := gm.store.GetDivisionsBySeason(ctx, previousSeasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous season divisions: %w", err)
	}

	// Filter to rookie divisions (100+)
	rookieDivisions := []models.LeagueDivision{}
	for _, div := range prevDivisions {
		if div.DivisionNumber >= RookieDivisionNumberBase {
			rookieDivisions = append(rookieDivisions, div)
		}
	}

	if len(rookieDivisions) == 0 {
		// No rookie divisions in previous season, nothing to graduate
		return result, nil
	}

	// Get standings from all rookie divisions
	allRookieStandings := []models.LeagueStanding{}
	for _, div := range rookieDivisions {
		standings, err := gm.store.GetStandings(ctx, div.Uuid)
		if err != nil {
			return nil, fmt.Errorf("failed to get standings for division %d: %w", div.DivisionNumber, err)
		}
		allRookieStandings = append(allRookieStandings, standings...)
	}

	if len(allRookieStandings) == 0 {
		// No rookies to graduate
		return result, nil
	}

	// Sort by rank (best to worst)
	sort.Slice(allRookieStandings, func(i, j int) bool {
		// Lower rank number = better performance
		if allRookieStandings[i].Rank.Valid && allRookieStandings[j].Rank.Valid {
			return allRookieStandings[i].Rank.Int32 < allRookieStandings[j].Rank.Int32
		}
		// Handle cases where rank might not be set (shouldn't happen, but be safe)
		return false
	})

	// Get new season divisions to determine where to place rookies
	newDivisions, err := gm.store.GetDivisionsBySeason(ctx, newSeasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get new season divisions: %w", err)
	}

	// Filter to regular divisions only
	regularDivisions := []models.LeagueDivision{}
	for _, div := range newDivisions {
		if div.DivisionNumber < RookieDivisionNumberBase {
			regularDivisions = append(regularDivisions, div)
		}
	}

	if len(regularDivisions) == 0 {
		return nil, fmt.Errorf("no regular divisions available in new season for rookie graduation")
	}

	// Sort divisions by number
	sort.Slice(regularDivisions, func(i, j int) bool {
		return regularDivisions[i].DivisionNumber < regularDivisions[j].DivisionNumber
	})

	highestDivision := regularDivisions[len(regularDivisions)-1].DivisionNumber

	// Calculate graduation groups
	groups := gm.calculateGraduationGroups(allRookieStandings, highestDivision)

	// Create a map for quick division lookup
	divisionMap := make(map[int32]models.LeagueDivision)
	for _, div := range regularDivisions {
		divisionMap[div.DivisionNumber] = div
	}

	// Place rookies into their target divisions
	for _, group := range groups {
		targetDiv, exists := divisionMap[group.TargetDivision]
		if !exists {
			// Division doesn't exist yet - this can happen in overflow cases
			// Skip for now, rebalancing will handle creating new divisions
			continue
		}

		divName := fmt.Sprintf("Division %d", targetDiv.DivisionNumber)
		if targetDiv.DivisionName.Valid {
			divName = targetDiv.DivisionName.String
		}

		for _, standing := range group.Rookies {
			// Update their division assignment
			err := gm.store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
				UserID:      standing.UserID,
				SeasonID:    newSeasonID,
				DivisionID:  pgtype.UUID{Bytes: targetDiv.Uuid, Valid: true},
				FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
			})
			if err != nil {
				// Player might not be registered yet, skip
				continue
			}

			result.GraduatedRookies = append(result.GraduatedRookies, PlacedPlayer{
				CategorizedPlayer: CategorizedPlayer{
					Registration: models.LeagueRegistration{
						UserID:   standing.UserID,
						SeasonID: newSeasonID,
					},
					Category: PlayerCategoryReturning, // They're returning from previous season
					Rating:   0,                       // Don't have current rating here
				},
				DivisionID:   targetDiv.Uuid,
				DivisionName: divName,
			})
		}
	}

	return result, nil
}

// calculateGraduationGroups determines how to split rookies into groups
// and which divisions each group should be placed into
func (gm *GraduationManager) calculateGraduationGroups(
	rookieStandings []models.LeagueStanding,
	highestRegularDivision int32,
) []GraduationGroup {
	numRookies := len(rookieStandings)
	if numRookies == 0 {
		return []GraduationGroup{}
	}

	// Calculate group sizes using graduation formula: ceil(N/6)
	groupSizes := calculateGraduationGroupSizes(numRookies)
	numGroups := len(groupSizes)

	// Determine starting division
	var startingDivision int32
	if highestRegularDivision == 1 {
		// Special case: only one division exists, must place all rookies there
		startingDivision = 1
		// Override group sizes to have just one group
		groupSizes = []int{numRookies}
		numGroups = 1
	} else {
		// Normal case: skip Division 1
		startingDivision = highestRegularDivision - int32(numGroups) + 1
		if startingDivision < 2 {
			startingDivision = 2
		}
	}

	// Create groups
	groups := make([]GraduationGroup, 0, numGroups)
	currentRank := 1 // 1-based rank

	for i, groupSize := range groupSizes {
		targetDivision := startingDivision + int32(i)

		// Handle case where we need more divisions than exist
		// Cap at highest division, multiple groups can go to same division
		if targetDivision > highestRegularDivision {
			targetDivision = highestRegularDivision
		}

		endRank := currentRank + groupSize - 1
		if endRank > numRookies {
			endRank = numRookies
		}

		// Extract rookies for this group
		groupRookies := rookieStandings[currentRank-1 : endRank]

		groups = append(groups, GraduationGroup{
			StartRank:      currentRank,
			EndRank:        endRank,
			TargetDivision: targetDivision,
			Rookies:        groupRookies,
		})

		currentRank = endRank + 1
	}

	return groups
}

// calculateGraduationGroupSizes determines how many rookies go into each group
// Uses the formula: groupSize = ceil(N/6), then distributes rookies evenly
func calculateGraduationGroupSizes(numRookies int) []int {
	if numRookies == 0 {
		return []int{}
	}

	// Group size is ceil(numRookies / 6)
	groupSize := (numRookies + 5) / 6 // ceiling division

	// Number of groups is ceil(numRookies / groupSize)
	numGroups := (numRookies + groupSize - 1) / groupSize

	// Distribute rookies across groups
	baseSize := numRookies / numGroups
	remainder := numRookies % numGroups

	sizes := make([]int, numGroups)
	for i := 0; i < numGroups; i++ {
		sizes[i] = baseSize
		if i < remainder {
			sizes[i]++
		}
	}

	return sizes
}
