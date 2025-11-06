package league

import (
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
	Rookies        []models.GetStandingsRow
}

// calculateGraduationGroups determines how to split rookies into groups
// and which divisions each group should be placed into
func (gm *GraduationManager) calculateGraduationGroups(
	rookieStandings []models.GetStandingsRow,
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
