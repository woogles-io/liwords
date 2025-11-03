package league

import (
	"github.com/google/uuid"

	"github.com/woogles-io/liwords/pkg/stores/league"
)

// PlacementManager handles initial division placement for a season
type PlacementManager struct {
	store league.Store
}

// NewPlacementManager creates a new placement manager
func NewPlacementManager(store league.Store) *PlacementManager {
	return &PlacementManager{
		store: store,
	}
}

// PlacementResult tracks the outcome of placing players into divisions
type PlacementResult struct {
	// Successfully placed returning players (assigned to their previous divisions)
	PlacedReturning []PlacedPlayer

	// New players (rookies) that need division assignment
	NeedingRookiePlacement []CategorizedPlayer

	// Returning players placed in the lowest division because their previous division doesn't exist
	PlacedInLowestDivision []PlacedPlayer
}

// PlacedPlayer represents a player that has been assigned to a division
type PlacedPlayer struct {
	CategorizedPlayer
	DivisionID   uuid.UUID
	DivisionName string
}
