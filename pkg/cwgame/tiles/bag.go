package tiles

import (
	"fmt"
	"sort"

	"github.com/rs/zerolog/log"
	"lukechampine.com/frand"

	"github.com/domino14/liwords/pkg/cwgame/runemapping"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
)

// TileBag returns a list of bytes rather than MachineLetters to keep
// it as compatible with the protobuf object as possible. A byte is just
// an unsigned MachineLetter, but bit-identical.
func TileBag(d *LetterDistribution) *ipc.Bag {
	tiles := make([]byte, d.numLetters)

	idx := 0
	for rn, ct := range d.Distribution {
		val, err := d.runemapping.Val(rn)
		if err != nil {
			log.Fatal().Msgf("attempt to initialize bag failed: %v", err)
		}
		for j := uint8(0); j < ct; j++ {
			tiles[idx] = byte(val)
			idx++
		}
	}
	sort.Slice(tiles, func(i, j int) bool { return tiles[i] < tiles[j] })
	return &ipc.Bag{Tiles: tiles}
}

// Draw draws n tiles from the bag into the passed-in ml MachineLetter array.
// The receiving array must be properly sized!
// The bag will be modified.
func Draw(bag *ipc.Bag, n int, ml []runemapping.MachineLetter) error {
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
		ml[i-k] = runemapping.MachineLetter(bag.Tiles[i])
	}
	bag.Tiles = bag.Tiles[:k]
	return nil
}

// DrawAtMost draws at most n tiles from the bag. It can draw fewer if there
// are fewer tiles than n, and even draw no tiles at all :o
// This is a zero-alloc draw into the passed-in slice.
func DrawAtMost(bag *ipc.Bag, n int, ml []runemapping.MachineLetter) (int, error) {
	if n > len(bag.Tiles) {
		n = len(bag.Tiles)
	}
	err := Draw(bag, n, ml)
	return n, err
}

// Exchange exchanges the junk in your rack with new tiles.
func Exchange(bag *ipc.Bag, letters []runemapping.MachineLetter, ml []runemapping.MachineLetter) error {
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
func PutBack(bag *ipc.Bag, letters []runemapping.MachineLetter) {
	if len(letters) == 0 {
		return
	}
	for _, l := range letters {
		bag.Tiles = append(bag.Tiles, byte(l))
	}
}
