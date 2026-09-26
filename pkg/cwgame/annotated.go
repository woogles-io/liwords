package cwgame

import (
	wglconfig "github.com/domino14/word-golib/config"
	"github.com/domino14/word-golib/tilemapping"

	"github.com/woogles-io/liwords/pkg/cwgame/tiles"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// In an annotated game a rack holds only the tiles known to be on it: the
// ones the annotator entered or a move revealed. Its unknown tiles are still
// in the bag, so the bag there is the unseen pool. A move is played on racks
// topped up from the pool, and the top-up goes back before the move is kept.

// withFilledRacks runs fn, which plays one move, on racks topped up to their
// real size, then puts the top-up back.
func withFilledRacks(cfg *wglconfig.Config, gdoc *ipc.GameDocument, fn func() error) error {
	if gdoc.Type != ipc.GameType_ANNOTATED {
		return fn()
	}
	known := make([][]tilemapping.MachineLetter, len(gdoc.Racks))
	for i, r := range gdoc.Racks {
		known[i] = tilemapping.FromByteArr(r)
	}
	firstEvt := len(gdoc.Events)

	inv := NewTileInventory(gdoc, cfg)
	for i, size := range rackSizes(gdoc) {
		if _, err := inv.drawTilesFromBagToRack(i, size-len(gdoc.Racks[i])); err != nil {
			return err
		}
	}
	if err := fn(); err != nil {
		return err
	}

	for _, evt := range gdoc.Events[firstEvt:] {
		p := evt.PlayerIndex
		switch evt.Type {
		case ipc.GameEvent_PASS, ipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS, ipc.GameEvent_CHALLENGE_BONUS:
			// playMove recorded the filled rack.
			evt.Rack = tilemapping.MachineWord(known[p]).ToByteArr()
		case ipc.GameEvent_TILE_PLACEMENT_MOVE:
			known[p] = removeTiles(known[p], rackTilesOf(evt.PlayedTiles))
		case ipc.GameEvent_EXCHANGE:
			known[p] = removeTiles(known[p], tilemapping.FromByteArr(evt.Exchanged))
		case ipc.GameEvent_PHONY_TILES_RETURNED:
			known[p] = tilemapping.FromByteArr(evt.Rack)
		}
	}
	for i, rack := range gdoc.Racks {
		unknown := removeTiles(tilemapping.FromByteArr(rack), known[i])
		kept := removeTiles(tilemapping.FromByteArr(rack), unknown)
		tiles.PutBack(gdoc.Bag, unknown)
		gdoc.Racks[i] = tilemapping.MachineWord(kept).ToByteArr()
	}
	return resolveKnownRacks(cfg, gdoc)
}

// resolveKnownRacks completes the one rack whose unknown tiles are all that is
// left in the pool, which happens once the bag is empty.
func resolveKnownRacks(cfg *wglconfig.Config, gdoc *ipc.GameDocument) error {
	unknownRack, unknownCount := -1, 0
	for i, size := range rackSizes(gdoc) {
		if size > len(gdoc.Racks[i]) {
			if unknownRack >= 0 {
				return nil
			}
			unknownRack, unknownCount = i, size-len(gdoc.Racks[i])
		}
	}
	if unknownRack < 0 || len(gdoc.Bag.Tiles) > unknownCount {
		return nil
	}
	_, err := NewTileInventory(gdoc, cfg).drawTilesFromBagToRack(unknownRack, unknownCount)
	return err
}

// rackSizes returns how many tiles each player actually holds. The racks
// themselves can't say, since they only hold the known tiles.
func rackSizes(gdoc *ipc.GameDocument) []int {
	sizes := make([]int, len(gdoc.Players))
	for i := range sizes {
		sizes[i] = RackTileLimit
	}
	offBoard := len(gdoc.Bag.Tiles)
	for _, r := range gdoc.Racks {
		offBoard += len(r)
	}
	full := RackTileLimit * len(sizes)
	if offBoard >= full {
		return sizes
	}
	// The bag is empty. Walk back to the play that emptied it: the other
	// players still held full racks then, and its player held the rest.
	placedSince := make([]int, len(sizes))
	placedTotal := 0
	for i := len(gdoc.Events) - 1; i >= 0; i-- {
		evt := gdoc.Events[i]
		if evt.Type != ipc.GameEvent_TILE_PLACEMENT_MOVE ||
			(i+1 < len(gdoc.Events) && gdoc.Events[i+1].Type == ipc.GameEvent_PHONY_TILES_RETURNED) {
			continue
		}
		placed := len(rackTilesOf(evt.PlayedTiles))
		if offBoard+placedTotal+placed > full {
			heldAfter := offBoard + placedTotal - RackTileLimit*(len(sizes)-1)
			for p := range sizes {
				held := RackTileLimit
				if p == int(evt.PlayerIndex) {
					held = heldAfter
				}
				sizes[p] = max(held-placedSince[p], 0)
			}
			return sizes
		}
		placedSince[evt.PlayerIndex] += placed
		placedTotal += placed
	}
	return sizes
}

// revealTiles adds to player p's known rack whichever of these tiles it lacks,
// taking them from the pool. If that overfills the rack, the entered rack was
// wrong, and the known rack becomes just these tiles.
func revealTiles(cfg *wglconfig.Config, gdoc *ipc.GameDocument, p int, needed []tilemapping.MachineLetter) ([]byte, error) {
	known := tilemapping.FromByteArr(gdoc.Racks[p])
	missing := removeTiles(needed, known)
	if len(missing) == 0 {
		return gdoc.Racks[p], nil
	}
	rack := append(known, missing...)
	if len(rack) > RackTileLimit {
		rack = needed
	}
	if err := NewTileInventory(gdoc, cfg).SetRack(p, tilemapping.MachineWord(rack).ToByteArr()); err != nil {
		return nil, err
	}
	return gdoc.Racks[p], nil
}

// rackTilesOf returns the rack tiles a play used: no play-through markers,
// and designated blanks as blanks.
func rackTilesOf(playedTiles []byte) []tilemapping.MachineLetter {
	out := []tilemapping.MachineLetter{}
	for _, t := range tilemapping.FromByteArr(playedTiles) {
		if t == 0 {
			continue
		}
		out = append(out, t.IntrinsicTileIdx())
	}
	return out
}

// removeTiles returns from minus one copy of each tile in sub, skipping tiles
// that from lacks.
func removeTiles(from, sub []tilemapping.MachineLetter) []tilemapping.MachineLetter {
	counts := map[tilemapping.MachineLetter]int{}
	for _, t := range sub {
		counts[t]++
	}
	out := []tilemapping.MachineLetter{}
	for _, t := range from {
		if counts[t] > 0 {
			counts[t]--
			continue
		}
		out = append(out, t)
	}
	return out
}
