package league

import (
	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

const (
	// RookieDivisionNumberBase is the starting division number for rookie divisions
	// Rookie divisions use numbers 100, 101, 102, etc. to distinguish from regular divisions
	RookieDivisionNumberBase = 100

	// MinPlayersForRookieDivision is the minimum number of rookies needed to create separate rookie divisions
	MinPlayersForRookieDivision = 10

	// Regular division size constraints
	// Regular divisions have higher minimums to maintain competitive balance

	// MinRegularDivisionSize is the minimum number of players in a regular division
	// (unless the league has collapsed and there aren't enough players)
	MinRegularDivisionSize = 13

	// MaxRegularDivisionSize is the maximum number of players in a regular division
	MaxRegularDivisionSize = 20

	// Rookie division size constraints
	// Rookie divisions are more flexible to accommodate varying numbers of new players

	// MinRookieDivisionSize is the minimum number of players in a rookie division
	MinRookieDivisionSize = 10

	// MaxRookieDivisionSize is the maximum number of players in a rookie division
	MaxRookieDivisionSize = 20
)

// RookieManager handles rookie division creation and placement
type RookieManager struct {
	store league.Store
}

// NewRookieManager creates a new rookie manager
func NewRookieManager(store league.Store) *RookieManager {
	return &RookieManager{
		store: store,
	}
}

// RookiePlacementResult tracks the outcome of placing rookies
type RookiePlacementResult struct {
	// Rookie divisions that were created
	CreatedDivisions []models.LeagueDivision

	// Rookies placed in newly created rookie divisions
	PlacedInRookieDivisions []PlacedPlayer

	// Rookies placed in existing regular divisions (when < 10 rookies)
	PlacedInRegularDivisions []PlacedPlayer
}
