package cwgame

import (
	"fmt"

	wglconfig "github.com/domino14/word-golib/config"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/cwgame/tiles"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// ValidatedBagOps provides validated bag operations that enforce tile count invariants
// and prevent corruption. All bag operations should go through these functions.

// CountTotalTiles returns the total number of tiles across bag, racks, and board
func CountTotalTiles(gdoc *ipc.GameDocument) int {
	total := len(gdoc.Bag.Tiles)

	// Add tiles from all racks
	for _, rack := range gdoc.Racks {
		total += len(rack)
	}

	// Add tiles from board (excluding through-tile markers)
	total += countBoardTiles(gdoc)

	return total
}

// ValidatedDraw draws n tiles from the bag with validation
func ValidatedDraw(gdoc *ipc.GameDocument, n int, ml []tilemapping.MachineLetter) error {
	if n > len(gdoc.Bag.Tiles) {
		return fmt.Errorf("tried to draw %v tiles, bag has %v", n, len(gdoc.Bag.Tiles))
	}

	return tiles.Draw(gdoc.Bag, n, ml)
}

// ValidatedDrawAtMost draws at most n tiles from the bag
func ValidatedDrawAtMost(gdoc *ipc.GameDocument, n int, ml []tilemapping.MachineLetter) (int, error) {
	return tiles.DrawAtMost(gdoc.Bag, n, ml)
}

// ValidatedPutBack puts tiles back in the bag with duplicate checking
// This function requires a Config to load the LetterDistribution for validation
func ValidatedPutBack(cfg *wglconfig.Config, gdoc *ipc.GameDocument, letters []tilemapping.MachineLetter) error {
	if len(letters) == 0 {
		return nil
	}

	// Get the letter distribution for this game
	ld, err := tilemapping.NamedLetterDistribution(cfg, gdoc.LetterDistribution)
	if err != nil {
		return fmt.Errorf("failed to load letter distribution: %w", err)
	}

	return ValidatedPutBackWithDist(gdoc, letters, ld)
}

// ValidatedPutBackWithDist puts tiles back in the bag with duplicate checking
// This variant accepts a pre-loaded LetterDistribution instead of cfg
func ValidatedPutBackWithDist(gdoc *ipc.GameDocument, letters []tilemapping.MachineLetter, ld *tilemapping.LetterDistribution) error {
	if len(letters) == 0 {
		return nil
	}

	// Count current tiles in bag and on board (NOT in racks, since we're putting back TO the bag)
	bagAndBoardCounts := make(map[byte]int)

	// Count bag tiles
	for _, t := range gdoc.Bag.Tiles {
		bagAndBoardCounts[t]++
	}

	// Count board tiles
	if gdoc.Board != nil {
		for _, tile := range gdoc.Board.Tiles {
			if tile == 0 {
				continue // Empty square
			}
			if tile&0x80 != 0 {
				// Designated blank - count as tile 0
				bagAndBoardCounts[0]++
			} else {
				// Regular tile
				bagAndBoardCounts[tile]++
			}
		}
	}

	// Create a map of tiles being put back
	putBackCounts := make(map[byte]int)
	for _, l := range letters {
		putBackCounts[byte(l)]++
	}

	// Check that putting back these tiles won't exceed the letter distribution
	// We check bag+board+putBack (NOT including other racks, since we're moving tiles from racks to bag)
	for tile, putBackCount := range putBackCounts {
		currentBagAndBoardCount := bagAndBoardCounts[tile]
		newCount := currentBagAndBoardCount + putBackCount
		maxCount := int(ld.Distribution()[tilemapping.MachineLetter(tile)])

		if newCount > maxCount {
			return fmt.Errorf("putting back %d of tile %d would give %d tiles in bag+board (max %d for this letter) - bag:%d board:computed",
				putBackCount, tile, newCount, maxCount, len(gdoc.Bag.Tiles))
		}
	}

	// Validation passed, put tiles back
	tiles.PutBack(gdoc.Bag, letters)
	return nil
}

// ValidatedRemoveTiles removes tiles from the bag with validation
func ValidatedRemoveTiles(gdoc *ipc.GameDocument, letters []tilemapping.MachineLetter) error {
	return tiles.RemoveTiles(gdoc.Bag, letters)
}

// CanRemoveTiles checks if the given tiles can be removed from the bag without actually removing them
func CanRemoveTiles(gdoc *ipc.GameDocument, letters []tilemapping.MachineLetter) bool {
	// Create a temporary map for tile counts
	tm := make(map[byte]int)
	for _, t := range gdoc.Bag.Tiles {
		tm[t]++
	}
	for _, t := range letters {
		b := byte(t)
		tm[b]--
		if tm[b] < 0 {
			return false
		}
	}
	return true
}

// ValidatedExchange exchanges tiles with validation
func ValidatedExchange(cfg *wglconfig.Config, gdoc *ipc.GameDocument, letters []tilemapping.MachineLetter, ml []tilemapping.MachineLetter) error {
	// Get the letter distribution for this game
	ld, err := tilemapping.NamedLetterDistribution(cfg, gdoc.LetterDistribution)
	if err != nil {
		return fmt.Errorf("failed to load letter distribution: %w", err)
	}

	return ValidatedExchangeWithDist(gdoc, letters, ml, ld)
}

// ValidatedExchangeWithDist exchanges tiles with validation
// This variant accepts a pre-loaded LetterDistribution instead of cfg
func ValidatedExchangeWithDist(gdoc *ipc.GameDocument, letters []tilemapping.MachineLetter, ml []tilemapping.MachineLetter, ld *tilemapping.LetterDistribution) error {
	// Draw first
	err := ValidatedDraw(gdoc, len(letters), ml)
	if err != nil {
		return err
	}

	// Put back with validation
	return ValidatedPutBackWithDist(gdoc, letters, ld)
}

// GetLetterDistribution returns the letter distribution for a game
func GetLetterDistribution(cfg *wglconfig.Config, distName string) (*tilemapping.LetterDistribution, error) {
	return tilemapping.NamedLetterDistribution(cfg, distName)
}

// ValidateTotalTiles checks that the total number of tiles equals the expected total
// and validates the distribution of each letter
func ValidateTotalTiles(gdoc *ipc.GameDocument, expectedTotal int) error {
	actual := CountTotalTiles(gdoc)
	if actual != expectedTotal {
		return fmt.Errorf("tile count mismatch: expected %d total tiles, found %d (bag: %d, racks: %d+%d, board: computed from events)",
			expectedTotal, actual, len(gdoc.Bag.Tiles), len(gdoc.Racks[0]), len(gdoc.Racks[1]))
	}
	return nil
}

// ValidateTileDistribution checks that each letter's count matches the expected distribution
func ValidateTileDistribution(gdoc *ipc.GameDocument, ld *tilemapping.LetterDistribution) error {
	// Count tiles by letter across all locations
	tileCounts := make(map[byte]int)

	// Count bag tiles
	for _, t := range gdoc.Bag.Tiles {
		tileCounts[t]++
	}

	// Count rack tiles
	for _, rack := range gdoc.Racks {
		for _, t := range rack {
			tileCounts[t]++
		}
	}

	// Count board tiles from actual board
	if gdoc.Board != nil {
		for _, tile := range gdoc.Board.Tiles {
			if tile == 0 {
				// Empty square, skip
				continue
			}
			if tile&0x80 != 0 {
				// Designated blank (e.g., 0x81 for blank-A)
				// Count as a blank (tile 0)
				tileCounts[0]++
			} else {
				// Regular tile
				tileCounts[tile]++
			}
		}
	}

	// Check against expected distribution
	for tile, count := range tileCounts {
		expected := int(ld.Distribution()[tilemapping.MachineLetter(tile)])
		if count != expected {
			return fmt.Errorf("tile %d count mismatch: expected %d, found %d", tile, expected, count)
		}
	}

	// Also check for missing tiles (tiles in distribution but not counted)
	for ml, expectedCount := range ld.Distribution() {
		if expectedCount > 0 {
			actualCount := tileCounts[byte(ml)]
			if actualCount != int(expectedCount) {
				return fmt.Errorf("tile %d count mismatch: expected %d, found %d", ml, expectedCount, actualCount)
			}
		}
	}

	return nil
}

// countBoardTilesFromEvents counts tiles by summing up tiles from all events
func countBoardTilesFromEvents(gdoc *ipc.GameDocument) int {
	count := 0
	for _, evt := range gdoc.Events {
		if evt.Type == ipc.GameEvent_TILE_PLACEMENT_MOVE {
			// Only count non-zero tiles (exclude through-tile markers)
			for _, tile := range evt.PlayedTiles {
				if tile != 0 {
					count++
				}
			}
		}
	}
	return count
}

// countBoardTilesFromBoard counts non-zero tiles from the actual board representation
func countBoardTilesFromBoard(gdoc *ipc.GameDocument) int {
	if gdoc.Board == nil {
		return 0
	}
	count := 0
	for _, tile := range gdoc.Board.Tiles {
		if tile != 0 {
			// Count both regular tiles and designated blanks (high bit set)
			count++
		}
	}
	return count
}

// countBoardTiles counts tiles on the board
func countBoardTiles(gdoc *ipc.GameDocument) int {
	// Use actual board as source of truth
	return countBoardTilesFromBoard(gdoc)
}

// LogTileState logs the current tile counts across bag, racks, and board for debugging
func LogTileState(gdoc *ipc.GameDocument, label string) {
	bagCount := len(gdoc.Bag.Tiles)
	rack0Count := len(gdoc.Racks[0])
	rack1Count := len(gdoc.Racks[1])
	boardCount := countBoardTiles(gdoc)
	total := bagCount + rack0Count + rack1Count + boardCount

	log.Debug().
		Str("label", label).
		Int("bag", bagCount).
		Int("rack0", rack0Count).
		Int("rack1", rack1Count).
		Int("board", boardCount).
		Int("total", total).
		Msg("tile-state")
}
