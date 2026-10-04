package cwgame

import (
	"bytes"

	wglconfig "github.com/domino14/word-golib/config"
	"github.com/domino14/word-golib/tilemapping"

	"github.com/woogles-io/liwords/pkg/cwgame/tiles"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// In an annotated game a rack holds only the tiles known to be on it; the
// unknown ones stay in the bag, which is therefore the unseen pool. A move is
// played on racks topped up from the unseen pool, and the top-up goes back
// after.
func withFilledRacks(cfg *wglconfig.Config, gdoc *ipc.GameDocument, fn func() error) error {
	if gdoc.Type != ipc.GameType_ANNOTATED {
		return fn()
	}
	known := make([]tilemapping.MachineWord, len(gdoc.Racks))
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

	var err error
	for _, evt := range gdoc.Events[firstEvt:] {
		p := evt.PlayerIndex
		switch evt.Type {
		case ipc.GameEvent_PASS, ipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS, ipc.GameEvent_CHALLENGE_BONUS:
			evt.Rack = known[p].ToByteArr() // playMove recorded the filled rack
		case ipc.GameEvent_TILE_PLACEMENT_MOVE:
			known[p], err = tilemapping.Leave(known[p], tilemapping.FromByteArr(evt.PlayedTiles), true)
		case ipc.GameEvent_EXCHANGE:
			known[p], err = tilemapping.Leave(known[p], tilemapping.FromByteArr(evt.Exchanged), false)
		case ipc.GameEvent_PHONY_TILES_RETURNED:
			known[p] = tilemapping.FromByteArr(evt.Rack)
		}
		if err != nil {
			return err
		}
	}
	for i, rack := range gdoc.Racks {
		unknown, err := tilemapping.Leave(tilemapping.FromByteArr(rack), known[i], false)
		if err != nil {
			return err
		}
		tiles.PutBack(gdoc.Bag, unknown)
		gdoc.Racks[i] = known[i].ToByteArr()
	}
	return resolveKnownRacks(cfg, gdoc)
}

// resolveKnownRacks completes the only rack with unknown tiles once the bag is
// empty, since those tiles must be all that is left.
func resolveKnownRacks(cfg *wglconfig.Config, gdoc *ipc.GameDocument) error {
	sizes := rackSizes(gdoc)
	unknown := []int{}
	for i, size := range sizes {
		if size > len(gdoc.Racks[i]) {
			unknown = append(unknown, i)
		}
	}
	if len(unknown) != 1 || len(gdoc.Bag.Tiles) > sizes[unknown[0]]-len(gdoc.Racks[unknown[0]]) {
		return nil
	}
	_, err := NewTileInventory(gdoc, cfg).drawTilesFromBagToRack(unknown[0], len(gdoc.Bag.Tiles))
	return err
}

// rackSizes returns how many tiles each player really holds.
func rackSizes(gdoc *ipc.GameDocument) []int {
	offBoard := len(gdoc.Bag.Tiles) + len(gdoc.Racks[0]) + len(gdoc.Racks[1])
	sizes := []int{RackTileLimit, RackTileLimit}
	if offBoard >= 2*RackTileLimit {
		return sizes
	}
	// The bag is empty. Walk back to the play that emptied it; the other
	// player held a full rack then.
	placed := []int{0, 0}
	for i := len(gdoc.Events) - 1; i >= 0; i-- {
		evt := gdoc.Events[i]
		if evt.Type != ipc.GameEvent_TILE_PLACEMENT_MOVE ||
			(i+1 < len(gdoc.Events) && gdoc.Events[i+1].Type == ipc.GameEvent_PHONY_TILES_RETURNED) {
			continue
		}
		n := len(evt.PlayedTiles) - bytes.Count(evt.PlayedTiles, []byte{0})
		if offBoard+placed[0]+placed[1]+n > 2*RackTileLimit {
			sizes[evt.PlayerIndex] = offBoard + placed[0] + placed[1] - RackTileLimit
			break
		}
		placed[evt.PlayerIndex] += n
	}
	return []int{max(sizes[0]-placed[0], 0), max(sizes[1]-placed[1], 0)}
}

// revealTiles adds the tiles a move uses to player p's known rack, or makes
// them the whole known rack if that would hold more tiles than p really has.
func revealTiles(cfg *wglconfig.Config, gdoc *ipc.GameDocument, p int, used []tilemapping.MachineLetter) ([]byte, error) {
	counts := map[tilemapping.MachineLetter]int{}
	for _, t := range gdoc.Racks[p] {
		counts[tilemapping.MachineLetter(t)]++
	}
	rack := tilemapping.FromByteArr(gdoc.Racks[p])
	for _, t := range used {
		if counts[t] > 0 {
			counts[t]--
		} else {
			rack = append(rack, t)
		}
	}
	if len(rack) == len(gdoc.Racks[p]) {
		return gdoc.Racks[p], nil
	}
	if len(rack) > rackSizes(gdoc)[p] {
		rack = used
	}
	err := NewTileInventory(gdoc, cfg).SetRack(p, rack.ToByteArr())
	return gdoc.Racks[p], err
}

// RestoreRackForAmendment sets up the rack for re-playing an amended event:
// the player's known rack before it, plus any tiles its old rack held that its
// old move didn't use. The old move's tiles are dropped, since they may only
// have been known through the move being replaced.
func RestoreRackForAmendment(cfg *wglconfig.Config, gdoc *ipc.GameDocument, evt *ipc.GameEvent) error {
	p := int(evt.PlayerIndex)
	unused := tilemapping.FromByteArr(evt.Rack)
	var err error
	switch evt.Type {
	case ipc.GameEvent_TILE_PLACEMENT_MOVE:
		unused, err = tilemapping.Leave(unused, tilemapping.FromByteArr(evt.PlayedTiles), true)
	case ipc.GameEvent_EXCHANGE:
		unused, err = tilemapping.Leave(unused, tilemapping.FromByteArr(evt.Exchanged), false)
	}
	if err != nil {
		unused = nil
	}
	rack := tilemapping.FromByteArr(gdoc.Racks[p])
	counts := map[tilemapping.MachineLetter]int{}
	for _, t := range rack {
		counts[t]++
	}
	for _, t := range unused {
		if counts[t] > 0 {
			counts[t]--
		} else {
			rack = append(rack, t)
		}
	}
	if len(rack) > rackSizes(gdoc)[p] {
		return nil // keep the known rack from before the event
	}
	if err := NewTileInventory(gdoc, cfg).SetRack(p, rack.ToByteArr()); err != nil {
		return err
	}
	return resolveKnownRacks(cfg, gdoc)
}
