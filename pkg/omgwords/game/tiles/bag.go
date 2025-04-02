package tiles

import (
	"fmt"
	rv2 "math/rand/v2"
	"sort"
	"sync"

	"github.com/domino14/word-golib/tilemapping"
	flatbuffers "github.com/google/flatbuffers/go"

	"github.com/woogles-io/liwords/pkg/omgwords/game/gamestate"
)

// The bag functions do a lot of byte-twiddling and manipulation inside the flatbuffer.
// It's not even clear whether this is acceptable from the documentation, but looking
// at the generated code, it should all work. Note that we are not modifying the vector sizes,
// only the contents of the vectors.
// Still, this seems a bit fragile and we should test this thoroughly to make sure
// updates to flatbuffers/Go/etc don't break our code.

func marshalRandomizer(r *rv2.ChaCha8) []byte {
	bts, err := r.MarshalBinary()
	if err != nil {
		panic(err)
	}
	// This requires knowing some internals. The maximum length bts can be is 64,
	// but we don't wish to be resizing this internal array inside the flatbuffer
	// all the time. So let's always pad it to 64 bytes.
	// The reader code seems to handle this ok. This is still not necessarily
	// great for the future, so we need to have some tests for this behavior.
	actualLength := len(bts)
	if actualLength < 64 {
		bts = append(bts, make([]byte, 64-len(bts))...)
	}
	// prefix the _actual_ length of the randomizer bytes to the byte array.
	bts = append([]byte{byte(actualLength)}, bts...)

	return bts
}

// global pool for ChaCha8 randomizers.
var chaCha8Pool = sync.Pool{
	New: func() interface{} {
		// Allocate a ChaCha8 using a default (zero) seed.
		return rv2.NewChaCha8([32]byte{})
	},
}

// GetChaCha8 retrieves a ChaCha8 from the pool
func GetChaCha8() *rv2.ChaCha8 {
	rng := chaCha8Pool.Get().(*rv2.ChaCha8)
	return rng
}

// PutChaCha8 returns the ChaCha8 back into the pool.
func PutChaCha8(rng *rv2.ChaCha8) {
	chaCha8Pool.Put(rng)
}

func unmarshalRandomizer(bts []byte) (*rv2.ChaCha8, error) {
	if len(bts) < 1 {
		return nil, fmt.Errorf("randomizer bytes are empty")
	}
	actualLength := bts[0]
	if int(actualLength) > len(bts)-1 {
		return nil, fmt.Errorf("randomizer bytes are too short")
	}
	bts = bts[1 : actualLength+1]
	ccrd := GetChaCha8()
	err := ccrd.UnmarshalBinary(bts)
	if err != nil {
		return nil, err
	}
	return ccrd, nil
}

// BuildTileBag creates a flatbuffer tilebag.
func BuildTileBag(builder *flatbuffers.Builder, d *tilemapping.LetterDistribution, seed [32]byte) flatbuffers.UOffsetT {
	tiles := make([]byte, d.NumTotalLetters())

	idx := 0
	for ml, ct := range d.Distribution() {
		for j := uint8(0); j < ct; j++ {
			tiles[idx] = byte(ml)
			idx++
		}
	}
	sort.Slice(tiles, func(i, j int) bool { return tiles[i] < tiles[j] })
	bagVector := builder.CreateByteVector(tiles)

	rdmzr := rv2.NewChaCha8(seed) // 256 bits of entropy; 2^256 = roughly 10^77, close to the number of atoms in the universe.
	bts := marshalRandomizer(rdmzr)
	randStateVector := builder.CreateByteVector(bts)

	gamestate.TileBagStart(builder)
	gamestate.TileBagAddBag(builder, bagVector)
	gamestate.TileBagAddNumTilesInBag(builder, int8(idx))
	gamestate.TileBagAddRandState(builder, randStateVector)

	return gamestate.TileBagEnd(builder)
}

// Draw draws n tiles from the bag into the passed-in ml MachineLetter array.
// The receiving array must be properly sized!
// The bag will be modified.
func Draw(bag *gamestate.TileBag, n int, ml []tilemapping.MachineLetter) error {

	rdstbts := bag.RandStateBytes()
	ccrd, err := unmarshalRandomizer(rdstbts)
	if err != nil {
		return err
	}
	rnd := rv2.New(ccrd)
	// Put the ChaCha8 back into the pool when we're done with it.
	defer func() {
		PutChaCha8(ccrd)
	}()

	l := bag.NumTilesInBag()

	if n > int(l) {
		return fmt.Errorf("tried to draw %v tiles, tile bag has %v",
			n, l)
	}
	k := int(l) - n
	// Directly mutate bag without accessor functions. This should only work
	// if we're swapping bytes around and not changing the length of the slice.
	bbts := bag.BagBytes()

	for i := int(l); i > k; i-- {
		xi := rnd.IntN(i)
		// move the selected tile to the end
		bbts[i-1], bbts[xi] = bbts[xi], bbts[i-1]
	}
	for i := k; i < int(l); i++ {
		ml[i-k] = tilemapping.MachineLetter(bbts[i])
	}
	bag.MutateNumTilesInBag(l - int8(n))

	// re-marshal randomizer bytes and copy directly into memory location of
	// old bagstate bytes. This better all work.
	newbts := marshalRandomizer(ccrd)
	if len(newbts) != len(rdstbts) {
		return fmt.Errorf("new randomizer bytes are not the same length as old ones")
	}
	copy(rdstbts, newbts)

	return nil
}

// DrawAtMost draws at most n tiles from the bag. It can draw fewer if there
// are fewer tiles than n, and even draw no tiles at all :o
// This is a zero-alloc draw into the passed-in slice.
func DrawAtMost(bag *gamestate.TileBag, n int, ml []tilemapping.MachineLetter) (int, error) {
	nt := bag.NumTilesInBag()
	if n > int(nt) {
		n = int(nt)
	}
	err := Draw(bag, n, ml)
	return n, err
}

// Exchange exchanges the junk in your rack with new tiles.
func Exchange(bag *gamestate.TileBag, letters []tilemapping.MachineLetter, ml []tilemapping.MachineLetter) error {
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
func PutBack(bag *gamestate.TileBag, letters []tilemapping.MachineLetter) {
	if len(letters) == 0 {
		return
	}

	bbts := bag.BagBytes()
	nt := bag.NumTilesInBag()
	for i := 0; i < len(letters); i++ {
		bbts[i+int(nt)] = byte(letters[i])
	}
	bag.MutateNumTilesInBag(nt + int8(len(letters)))
}

func RemoveTiles(bag *gamestate.TileBag, letters []tilemapping.MachineLetter) error {

	// Create a temporary map for speed (well, maybe)
	ntiles := 0
	tm := make(map[byte]int)
	bbts := bag.BagBytes()
	for _, t := range bbts[:bag.NumTilesInBag()] {
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

	idx := 0
	// Replace tile array "in-place"
	for k, v := range tm {
		for i := 0; i < v; i++ {
			bbts[idx] = k
			idx++
		}
	}
	// Set the number of tiles in the bag to the new size.
	bag.MutateNumTilesInBag(int8(ntiles - len(letters)))

	return nil
}

func Count(bag *gamestate.TileBag, letter tilemapping.MachineLetter) int {
	ct := 0
	for _, t := range bag.BagBytes()[:bag.NumTilesInBag()] {
		if t == byte(letter) {
			ct++
		}
	}
	return ct
}

// Sort sorts the bag. Normally there is no need to do this, since we always
// draw randomly from the bag, but this can be used for determinism (for
// example in tests)
func Sort(bag *gamestate.TileBag) {
	sort.Slice(bag.BagBytes()[:bag.NumTilesInBag()], func(i, j int) bool {
		return bag.BagBytes()[i] < bag.BagBytes()[j]
	})
}

func InBag(bag *gamestate.TileBag) int {
	return int(bag.NumTilesInBag())
}
