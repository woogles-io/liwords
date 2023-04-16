package tiles

import (
	"fmt"
	"sort"

	"lukechampine.com/frand"

	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/domino14/macondo/tilemapping"
)

// TileBag returns a list of bytes rather than MachineLetters to keep
// it as compatible with the protobuf object as possible. A byte is just
// an unsigned MachineLetter, but bit-identical.
func TileBag(d *tilemapping.LetterDistribution) *ipc.Bag {
	tiles := make([]byte, d.NumTotalLetters())

	idx := 0
	for ml, ct := range d.Distribution() {
		for j := uint8(0); j < ct; j++ {
			tiles[idx] = byte(ml)
			idx++
		}
	}
	sort.Slice(tiles, func(i, j int) bool { return tiles[i] < tiles[j] })
	return &ipc.Bag{Tiles: tiles}
}

// Draw draws n tiles from the bag into the passed-in ml MachineLetter array.
// The receiving array must be properly sized!
// The bag will be modified.
func Draw(bag *ipc.Bag, n int, ml []tilemapping.MachineLetter) error {
	if n > len(bag.Tiles) {
		return fmt.Errorf("tried to draw %v tiles, tile bag has %v",
			n, len(bag.Tiles))
	}
	l := len(bag.Tiles)
	k := l - n
	for i := l; i > k; i-- {
		xi := frand.Intn(i)
		// move the selected tile to the end
		bag.Tiles[i-1], bag.Tiles[xi] = bag.Tiles[xi], bag.Tiles[i-1]
	}
	for i := k; i < l; i++ {
		ml[i-k] = tilemapping.MachineLetter(bag.Tiles[i])
	}
	bag.Tiles = bag.Tiles[:k]
	return nil
}

// DrawAtMost draws at most n tiles from the bag. It can draw fewer if there
// are fewer tiles than n, and even draw no tiles at all :o
// This is a zero-alloc draw into the passed-in slice.
func DrawAtMost(bag *ipc.Bag, n int, ml []tilemapping.MachineLetter) (int, error) {
	if n > len(bag.Tiles) {
		n = len(bag.Tiles)
	}
	err := Draw(bag, n, ml)
	return n, err
}

// Exchange exchanges the junk in your rack with new tiles.
func Exchange(bag *ipc.Bag, letters []tilemapping.MachineLetter, ml []tilemapping.MachineLetter) error {
	err := Draw(bag, len(letters), ml)
	if err != nil {
		return err
	}
	// put exchanged tiles back into the bag. no need to shuffle because
	// drawing always shuffles.
	PutBack(bag, letters)
	return nil
}

// PutBack puts the tiles back in the bag.
func PutBack(bag *ipc.Bag, letters []tilemapping.MachineLetter) {
	if len(letters) == 0 {
		return
	}
	for _, l := range letters {
		bag.Tiles = append(bag.Tiles, byte(l))
	}
}

func RemoveTiles(bag *ipc.Bag, letters []tilemapping.MachineLetter) error {

	// Create a temporary map for speed (well, maybe)
	ntiles := 0
	tm := make(map[byte]int)
	for _, t := range bag.Tiles {
		tm[t]++
		ntiles++
	}
	for _, t := range letters {
		b := byte(t)
		tm[b]--
		if tm[b] < 0 {
			return fmt.Errorf("tried to remove tile %d from bag that was not there", b)
		}
	}
	bag.Tiles = make([]byte, ntiles-len(letters))
	idx := 0
	// Replace tile array.
	for k, v := range tm {
		for i := 0; i < v; i++ {
			bag.Tiles[idx] = k
			idx++
		}
	}
	return nil
}

func Count(bag *ipc.Bag, letter tilemapping.MachineLetter) int {
	ct := 0
	for _, t := range bag.Tiles {
		if t == byte(letter) {
			ct++
		}
	}
	return ct
}

// Sort sorts the bag. Normally there is no need to do this, since we always
// draw randomly from the bag, but this can be used for determinism (for
// example in tests)
func Sort(bag *ipc.Bag) {
	sort.Slice(bag.Tiles, func(i, j int) bool {
		return bag.Tiles[i] < bag.Tiles[j]
	})
}

func InBag(bag *ipc.Bag) int {
	return len(bag.Tiles)
}
