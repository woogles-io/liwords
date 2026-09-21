package cwgame

import (
	"github.com/domino14/word-golib/tilemapping"

	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// In annotated games the server tops racks up with random tiles so that tile
// accounting stays valid. KnownRacks holds the subset of each rack that the
// annotator actually entered (or that a move implied); only those tiles may be
// recorded as an event's rack. Documents without KnownRacks keep the old
// behaviour of recording the whole rack.

func tracksKnownRacks(gdoc *ipc.GameDocument) bool {
	return gdoc.Type == ipc.GameType_ANNOTATED && len(gdoc.KnownRacks) == len(gdoc.Players)
}

// rackTileKey maps a tile to the rack tile it came from (blank designations
// become the blank).
func rackTileKey(t byte) byte {
	ml := tilemapping.MachineLetter(t)
	if ml.IsBlanked() {
		return 0
	}
	return t
}

func tileCounts(tiles []byte, skipPlaythrough bool) map[byte]int {
	counts := map[byte]int{}
	for _, t := range tiles {
		if skipPlaythrough && t == 0 {
			continue
		}
		counts[rackTileKey(t)]++
	}
	return counts
}

// intersectTiles returns the tiles of a that are also in b (multiset).
func intersectTiles(a, b []byte) []byte {
	avail := tileCounts(b, false)
	out := []byte{}
	for _, t := range a {
		k := rackTileKey(t)
		if avail[k] > 0 {
			avail[k]--
			out = append(out, k)
		}
	}
	return out
}

// knownRack returns the tiles of player p's rack that should be recorded.
func knownRack(gdoc *ipc.GameDocument, p int) []byte {
	if !tracksKnownRacks(gdoc) {
		return gdoc.Racks[p]
	}
	return gdoc.KnownRacks[p]
}

// knownRackWith returns player p's known tiles plus whichever of the given
// tiles (played or exchanged, all of which must be on the rack) they don't
// already cover.
func knownRackWith(gdoc *ipc.GameDocument, p int, tiles []byte) []byte {
	if !tracksKnownRacks(gdoc) {
		return gdoc.Racks[p]
	}
	out := append([]byte{}, gdoc.KnownRacks[p]...)
	have := tileCounts(out, false)
	for _, t := range tiles {
		k := rackTileKey(t)
		if have[k] > 0 {
			have[k]--
			continue
		}
		out = append(out, k)
	}
	return intersectTiles(out, gdoc.Racks[p])
}

// recordedRack returns the rack to record for a move made from the full rack
// in gevt, and whether known racks are tracked at all.
func recordedRack(gdoc *ipc.GameDocument, gevt *ipc.GameEvent) ([]byte, bool) {
	if !tracksKnownRacks(gdoc) {
		return nil, false
	}
	p := int(gdoc.PlayerOnTurn)
	switch gevt.Type {
	case ipc.GameEvent_TILE_PLACEMENT_MOVE:
		used := []byte{}
		for _, t := range gevt.PlayedTiles {
			if t != 0 {
				used = append(used, rackTileKey(t))
			}
		}
		return knownRackWith(gdoc, p, used), true
	case ipc.GameEvent_EXCHANGE:
		return knownRackWith(gdoc, p, gevt.Exchanged), true
	}
	return knownRack(gdoc, p), true
}

func setKnownRack(gdoc *ipc.GameDocument, p int, rack []byte) {
	if tracksKnownRacks(gdoc) {
		gdoc.KnownRacks[p] = intersectTiles(rack, gdoc.Racks[p])
	}
}

// removeKnownTiles drops tiles that left player p's rack (played or
// exchanged) from its known tiles.
func removeKnownTiles(gdoc *ipc.GameDocument, p int, tiles []byte, skipPlaythrough bool) {
	if !tracksKnownRacks(gdoc) {
		return
	}
	remove := tileCounts(tiles, skipPlaythrough)
	out := []byte{}
	for _, t := range gdoc.KnownRacks[p] {
		if remove[t] > 0 {
			remove[t]--
			continue
		}
		out = append(out, t)
	}
	gdoc.KnownRacks[p] = out
}

// clampKnownRacks keeps each known rack a subset of the actual rack, e.g.
// after tiles were borrowed from the opponent.
func clampKnownRacks(gdoc *ipc.GameDocument) {
	if !tracksKnownRacks(gdoc) {
		return
	}
	for p := range gdoc.KnownRacks {
		gdoc.KnownRacks[p] = intersectTiles(gdoc.KnownRacks[p], gdoc.Racks[p])
	}
}
