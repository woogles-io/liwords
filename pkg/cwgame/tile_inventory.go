package cwgame

import (
	"fmt"

	wglconfig "github.com/domino14/word-golib/config"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/cwgame/tiles"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// TileInventory manages all tile movements between bag, racks, and board.
// It enforces the core invariant: total tiles across all locations must
// match the letter distribution for the game.
//
// All operations are explicit about where tiles are moving from/to, making
// the tile flow clear and preventing accidental tile creation/destruction.
type TileInventory struct {
	gdoc *ipc.GameDocument
	cfg  *wglconfig.Config
}

// NewTileInventory creates a new TileInventory for the given game document.
func NewTileInventory(gdoc *ipc.GameDocument, cfg *wglconfig.Config) *TileInventory {
	return &TileInventory{
		gdoc: gdoc,
		cfg:  cfg,
	}
}

// ValidateInvariants checks that the total tile count and per-letter distribution
// are correct across bag, racks, and board.
func (inv *TileInventory) ValidateInvariants() error {
	// Get the letter distribution for this game
	dist, err := tilemapping.GetDistribution(inv.cfg, inv.gdoc.LetterDistribution)
	if err != nil {
		return fmt.Errorf("failed to load letter distribution: %w", err)
	}

	// Count tiles by letter across all locations
	tileCounts := make(map[byte]int)

	// Count bag tiles
	for _, t := range inv.gdoc.Bag.Tiles {
		tileCounts[t]++
	}

	// Count rack tiles
	for _, rack := range inv.gdoc.Racks {
		for _, t := range rack {
			tileCounts[t]++
		}
	}

	// Count board tiles (handling designated blanks)
	if inv.gdoc.Board != nil {
		for _, tile := range inv.gdoc.Board.Tiles {
			if tile == 0 {
				continue // Empty square
			}
			if tile&0x80 != 0 {
				// Designated blank (high bit set) - count as blank (tile 0)
				tileCounts[0]++
			} else {
				// Regular tile
				tileCounts[tile]++
			}
		}
	}

	// Check against expected distribution
	for tile, count := range tileCounts {
		expected := int(dist.Distribution()[tilemapping.MachineLetter(tile)])
		if count != expected {
			return fmt.Errorf("tile %d count mismatch: expected %d, found %d (bag=%d, board=%d, racks=%v)",
				tile, expected, count,
				len(inv.gdoc.Bag.Tiles),
				inv.GetBoardTileCount(),
				func() []int {
					counts := make([]int, len(inv.gdoc.Racks))
					for i := range inv.gdoc.Racks {
						counts[i] = len(inv.gdoc.Racks[i])
					}
					return counts
				}())
		}
	}

	// Also check for missing tiles (tiles in distribution but not counted)
	for ml, expectedCount := range dist.Distribution() {
		if expectedCount > 0 {
			actualCount := tileCounts[byte(ml)]
			if actualCount != int(expectedCount) {
				return fmt.Errorf("tile %d count mismatch: expected %d, found %d", ml, expectedCount, actualCount)
			}
		}
	}

	return nil
}

// =========================================================================
// Low-level primitives - explicit about tile movement
// These just do the movement and trust that the caller will validate.
// =========================================================================

// moveTilesFromRackToBag moves tiles from a player's rack into the bag.
func (inv *TileInventory) moveTilesFromRackToBag(playerIdx int, tilesToMove []tilemapping.MachineLetter) error {
	if len(tilesToMove) == 0 {
		return nil
	}

	// Verify tiles exist in the rack and remove them
	currentRack := tilemapping.FromByteArr(inv.gdoc.Racks[playerIdx])
	// zeroIsPlaythrough=false: treat tile 0 as a regular blank, not a play-through marker
	leave, err := tilemapping.Leave(currentRack, tilesToMove, false)
	if err != nil {
		return fmt.Errorf("rack doesn't contain tiles to move: %w", err)
	}

	// Update rack and bag
	inv.gdoc.Racks[playerIdx] = tilemapping.MachineWord(leave).ToByteArr()
	tiles.PutBack(inv.gdoc.Bag, tilesToMove)

	return nil
}

// moveTilesFromBagToRack moves specific tiles from the bag to a player's rack.
func (inv *TileInventory) moveTilesFromBagToRack(playerIdx int, tilesToMove []tilemapping.MachineLetter) error {
	if len(tilesToMove) == 0 {
		return nil
	}

	// Remove from bag (validates they exist)
	err := tiles.RemoveTiles(inv.gdoc.Bag, tilesToMove)
	if err != nil {
		return fmt.Errorf("bag doesn't contain tiles to move: %w", err)
	}

	// Add to rack
	currentRack := tilemapping.FromByteArr(inv.gdoc.Racks[playerIdx])
	newRack := append(currentRack, tilesToMove...)
	inv.gdoc.Racks[playerIdx] = tilemapping.MachineWord(newRack).ToByteArr()

	return nil
}

// drawTilesFromBagToRack draws up to n random tiles from the bag to a player's rack.
// Returns the number of tiles actually drawn.
func (inv *TileInventory) drawTilesFromBagToRack(playerIdx int, n int) (int, error) {
	if n <= 0 {
		return 0, nil
	}

	placeholder := make([]tilemapping.MachineLetter, n)
	drew, err := tiles.DrawAtMost(inv.gdoc.Bag, n, placeholder)
	if err != nil {
		return 0, err
	}

	if drew > 0 {
		drawnTiles := placeholder[:drew]
		currentRack := tilemapping.FromByteArr(inv.gdoc.Racks[playerIdx])
		newRack := append(currentRack, drawnTiles...)
		inv.gdoc.Racks[playerIdx] = tilemapping.MachineWord(newRack).ToByteArr()
	}

	return drew, nil
}

// removeFromRackWithoutReturning removes tiles from a rack without putting them anywhere.
// This is used when tiles are being played to the board.
func (inv *TileInventory) removeFromRackWithoutReturning(playerIdx int, tilesToRemove []tilemapping.MachineLetter) error {
	if len(tilesToRemove) == 0 {
		return nil
	}

	currentRack := tilemapping.FromByteArr(inv.gdoc.Racks[playerIdx])
	leave, err := tilemapping.Leave(currentRack, tilesToRemove, false)
	if err != nil {
		return fmt.Errorf("rack doesn't contain tiles to remove: %w", err)
	}

	inv.gdoc.Racks[playerIdx] = tilemapping.MachineWord(leave).ToByteArr()
	return nil
}

// =========================================================================
// High-level operations - declarative API
// These validate invariants after completing the operation.
// =========================================================================

// SetRack sets a player's rack to the specified tiles.
// Automatically puts back the current rack and draws the new rack from the bag.
// If tiles aren't available in the bag, borrows only the needed tiles from the opponent.
func (inv *TileInventory) SetRack(playerIdx int, desiredRack []byte) error {
	// Put current rack back in bag (if any)
	if len(inv.gdoc.Racks[playerIdx]) > 0 {
		currentRack := tilemapping.FromByteArr(inv.gdoc.Racks[playerIdx])
		if err := inv.moveTilesFromRackToBag(playerIdx, currentRack); err != nil {
			return fmt.Errorf("failed to put back current rack: %w", err)
		}
	}

	if len(desiredRack) == 0 {
		return inv.ValidateInvariants()
	}

	desiredTiles := tilemapping.FromByteArr(desiredRack)

	// Try to get tiles from bag
	err := inv.moveTilesFromBagToRack(playerIdx, desiredTiles)

	if err != nil {
		// Some tiles not available - figure out which ones and borrow from opponent
		opponentIdx := 1 - playerIdx

		if len(inv.gdoc.Racks[opponentIdx]) == 0 {
			return fmt.Errorf("tiles not available in bag and no opponent rack to borrow from: %w", err)
		}

		// Count what we need
		needed := make(map[byte]int)
		for _, t := range desiredTiles {
			needed[byte(t)]++
		}

		// Subtract what's available in the bag
		for _, t := range inv.gdoc.Bag.Tiles {
			if needed[t] > 0 {
				needed[t]--
			}
		}

		// Build list of tiles we need to borrow from opponent
		tilesToBorrow := []tilemapping.MachineLetter{}
		for tile, count := range needed {
			for i := 0; i < count; i++ {
				tilesToBorrow = append(tilesToBorrow, tilemapping.MachineLetter(tile))
			}
		}

		if len(tilesToBorrow) == 0 {
			// Shouldn't happen, but just in case
			return fmt.Errorf("failed to determine which tiles to borrow: %w", err)
		}

		// Borrow only the needed tiles from opponent
		if err := inv.moveTilesFromRackToBag(opponentIdx, tilesToBorrow); err != nil {
			return fmt.Errorf("opponent doesn't have needed tiles %v: %w", tilesToBorrow, err)
		}

		log.Debug().
			Interface("borrowed_tiles", tilesToBorrow).
			Int("opponent_rack_size", len(inv.gdoc.Racks[opponentIdx])).
			Int("bag_size", len(inv.gdoc.Bag.Tiles)).
			Msg("borrowed-tiles-from-opponent")

		// Now try again to get desired tiles from bag
		err = inv.moveTilesFromBagToRack(playerIdx, desiredTiles)
		if err != nil {
			return fmt.Errorf("failed to set rack even after borrowing: %w", err)
		}

		// Top off opponent's rack (they lost some tiles)
		tilesDrawn, errFill := inv.DrawToFillRack(opponentIdx)
		if errFill != nil {
			return fmt.Errorf("failed to fill opponent's rack after borrowing: %w", errFill)
		}

		log.Debug().
			Int("tiles_drawn", tilesDrawn).
			Interface("opponent_rack_after", inv.gdoc.Racks[opponentIdx]).
			Int("bag_size_after", len(inv.gdoc.Bag.Tiles)).
			Msg("filled-opponent-rack-after-borrowing")
	}

	// Validate invariants after the operation
	return inv.ValidateInvariants()
}

// SetAllRacks sets all players' racks at once.
// This is more efficient than calling SetRack multiple times when you're
// assigning racks for all players simultaneously.
//
// IMPORTANT: Empty or nil racks in the input array mean "don't touch this player's rack".
// This is intentional - the annotation editor frontend may call SetRacks with
// ["", "ABCDEFG"] to set player 1's rack without affecting player 0's rack.
// Only racks with len(r) > 0 will be put back and reassigned.
//
// If tiles aren't available in the bag and allowBorrowing is true, this will attempt
// to borrow from racks that are being preserved (empty/nil in input). If both racks are
// being set and overlap, the operation will fail. Borrowing should only be enabled for
// editor mode operations where rack inference is desired.
func (inv *TileInventory) SetAllRacks(racks [][]byte, allowBorrowing bool) error {
	if len(racks) != len(inv.gdoc.Racks) {
		return fmt.Errorf("racks length %d doesn't match player count %d", len(racks), len(inv.gdoc.Racks))
	}

	// Put back current racks based on mode:
	// - If allowBorrowing (editor mode): Only put back racks being reassigned (non-empty in input)
	//   This preserves racks where input is nil/empty - the "preserve rack" semantic
	// - If !allowBorrowing (replay mode): Put back ALL current racks (old behavior)
	//   This matches master where nil/empty input means "leave empty for now, don't preserve"
	for i, newRack := range racks {
		shouldPutBack := false
		if allowBorrowing {
			// Editor mode: only put back if we're assigning a new rack
			shouldPutBack = len(newRack) > 0 && len(inv.gdoc.Racks[i]) > 0
		} else {
			// Replay mode: always put back existing racks (master behavior)
			shouldPutBack = len(inv.gdoc.Racks[i]) > 0
		}

		if shouldPutBack {
			currentRack := tilemapping.FromByteArr(inv.gdoc.Racks[i])
			if err := inv.moveTilesFromRackToBag(i, currentRack); err != nil {
				return fmt.Errorf("failed to put back rack %d: %w", i, err)
			}
			inv.gdoc.Racks[i] = nil
		}
	}

	// Now assign all new racks from the bag
	// If tiles aren't available and borrowing is allowed, try to borrow from preserved racks
	for i, r := range racks {
		if len(r) > 0 {
			desiredTiles := tilemapping.FromByteArr(r)
			err := inv.moveTilesFromBagToRack(i, desiredTiles)

			if err != nil && allowBorrowing {
				// Try to borrow from preserved racks (those with empty/nil input)
				if err := inv.borrowFromPreservedRacks(i, desiredTiles, racks); err != nil {
					return fmt.Errorf("failed to assign rack %d: %w", i, err)
				}
			} else if err != nil {
				// Borrowing not allowed or not possible
				return fmt.Errorf("failed to assign rack %d: %w", i, err)
			}
		}
		// Note: nil or empty racks stay as-is (current rack is preserved)
	}

	// Validate invariants after the operation
	return inv.ValidateInvariants()
}

// borrowFromPreservedRacks attempts to borrow tiles from racks that are being preserved.
// A preserved rack is one where the input was nil or empty (meaning "don't touch this rack").
// This matches the borrowing behavior in SetRack but works across all preserved racks.
func (inv *TileInventory) borrowFromPreservedRacks(playerIdx int, desiredTiles []tilemapping.MachineLetter, racks [][]byte) error {
	// Count what we need
	needed := make(map[byte]int)
	for _, t := range desiredTiles {
		needed[byte(t)]++
	}

	// Subtract what's available in the bag
	for _, t := range inv.gdoc.Bag.Tiles {
		if needed[t] > 0 {
			needed[t]--
		}
	}

	// Build list of tiles we need to borrow
	tilesToBorrow := []tilemapping.MachineLetter{}
	for tile, count := range needed {
		for i := 0; i < count; i++ {
			tilesToBorrow = append(tilesToBorrow, tilemapping.MachineLetter(tile))
		}
	}

	if len(tilesToBorrow) == 0 {
		// Shouldn't happen, but just in case
		return fmt.Errorf("failed to determine which tiles to borrow")
	}

	// Try to borrow from preserved racks (those with empty/nil input)
	for opponentIdx, opponentRack := range racks {
		if opponentIdx == playerIdx {
			continue // Can't borrow from self
		}

		// Only borrow from preserved racks (empty/nil input)
		if len(opponentRack) > 0 {
			continue // This rack is being set, can't borrow from it
		}

		if len(inv.gdoc.Racks[opponentIdx]) == 0 {
			continue // No rack to borrow from
		}

		// Try to borrow from this opponent
		if err := inv.moveTilesFromRackToBag(opponentIdx, tilesToBorrow); err != nil {
			// This opponent doesn't have the tiles, try next
			continue
		}

		log.Debug().
			Interface("borrowed_tiles", tilesToBorrow).
			Int("from_player", opponentIdx).
			Int("to_player", playerIdx).
			Int("opponent_rack_size", len(inv.gdoc.Racks[opponentIdx])).
			Int("bag_size", len(inv.gdoc.Bag.Tiles)).
			Msg("borrowed-tiles-from-preserved-rack")

		// Now try again to get desired tiles from bag
		err := inv.moveTilesFromBagToRack(playerIdx, desiredTiles)
		if err != nil {
			return fmt.Errorf("failed to set rack even after borrowing: %w", err)
		}

		// Top off opponent's rack (they lost some tiles)
		tilesDrawn, errFill := inv.DrawToFillRack(opponentIdx)
		if errFill != nil {
			return fmt.Errorf("failed to fill opponent's rack after borrowing: %w", errFill)
		}

		log.Debug().
			Int("tiles_drawn", tilesDrawn).
			Interface("opponent_rack_after", inv.gdoc.Racks[opponentIdx]).
			Int("bag_size_after", len(inv.gdoc.Bag.Tiles)).
			Msg("filled-preserved-rack-after-borrowing")

		return nil // Success
	}

	return fmt.Errorf("tiles not available in bag and no preserved racks to borrow from")
}

// ExchangeTiles exchanges tiles from a player's rack with the bag.
// IMPORTANT: Draws new tiles BEFORE putting exchanged tiles back,
// to avoid redrawing the same tiles.
func (inv *TileInventory) ExchangeTiles(playerIdx int, tilesToExchange []tilemapping.MachineLetter) error {
	if len(tilesToExchange) == 0 {
		return nil
	}

	// Verify the tiles exist in the rack first
	currentRack := tilemapping.FromByteArr(inv.gdoc.Racks[playerIdx])
	_, err := tilemapping.Leave(currentRack, tilesToExchange, false)
	if err != nil {
		return fmt.Errorf("rack doesn't contain tiles to exchange: %w", err)
	}

	// Draw new tiles FIRST (before putting exchanged tiles back)
	drew, err := inv.drawTilesFromBagToRack(playerIdx, len(tilesToExchange))
	if err != nil {
		return fmt.Errorf("failed to draw tiles: %w", err)
	}

	if drew != len(tilesToExchange) {
		return fmt.Errorf("tried to exchange %d tiles but only drew %d", len(tilesToExchange), drew)
	}

	// Now remove the exchanged tiles from rack and put them in bag
	if err := inv.moveTilesFromRackToBag(playerIdx, tilesToExchange); err != nil {
		return fmt.Errorf("failed to put tiles in bag: %w", err)
	}

	// Validate invariants after the operation
	return inv.ValidateInvariants()
}

// DrawToFillRack draws tiles from the bag to fill a player's rack up to 7 tiles.
// Returns the number of tiles drawn.
func (inv *TileInventory) DrawToFillRack(playerIdx int) (int, error) {
	currentRackSize := len(inv.gdoc.Racks[playerIdx])
	tilesNeeded := RackTileLimit - currentRackSize

	if tilesNeeded <= 0 {
		return 0, nil
	}

	drew, err := inv.drawTilesFromBagToRack(playerIdx, tilesNeeded)
	if err != nil {
		return 0, err
	}

	// Validate invariants after the operation
	if err := inv.ValidateInvariants(); err != nil {
		return 0, err
	}

	return drew, nil
}

// ClearRack removes all tiles from a player's rack and returns them to the bag.
func (inv *TileInventory) ClearRack(playerIdx int) error {
	if len(inv.gdoc.Racks[playerIdx]) == 0 {
		return nil
	}

	currentRack := tilemapping.FromByteArr(inv.gdoc.Racks[playerIdx])
	if err := inv.moveTilesFromRackToBag(playerIdx, currentRack); err != nil {
		return err
	}

	// Validate invariants after the operation
	return inv.ValidateInvariants()
}

// GetBoardTileCount returns the number of non-zero tiles on the board
// (excluding through-tile markers).
func (inv *TileInventory) GetBoardTileCount() int {
	if inv.gdoc.Board == nil {
		return 0
	}
	count := 0
	for _, tile := range inv.gdoc.Board.Tiles {
		if tile != 0 {
			count++
		}
	}
	return count
}

// GetTotalTileCount returns the total number of tiles across bag, racks, and board.
func (inv *TileInventory) GetTotalTileCount() int {
	total := len(inv.gdoc.Bag.Tiles)

	// Add tiles from all racks
	for _, rack := range inv.gdoc.Racks {
		total += len(rack)
	}

	// Add tiles from board
	total += inv.GetBoardTileCount()

	return total
}
