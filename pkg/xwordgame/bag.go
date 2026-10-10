// Bag operations.
//
// word-golib already has a perfectly good tilemapping.Bag, and its rules
// semantics are what we mirror here -- in particular Exchange draws before it
// returns tiles, which is the actual rule. We do not use it directly for two
// reasons:
//
//  1. It has no serialisation story. It keeps an ordered tiles slice and a
//     counts slice in parallel, with the reconciliation between them private,
//     so there is no supported way to restore one from persisted counts. Its
//     counts slice is also sized to the distribution's alphabet rather than a
//     fixed width, which would force the snapshot decoder to know the letter
//     distribution before it could parse a bag.
//  2. It is four heap-allocated slices (including a full copy of the initial
//     distribution) plus Monte Carlo machinery -- fixedOrder, SwapTile,
//     Copy/CopyFrom -- that a live game does not need. State keeps its bag as a
//     fixed inline array so that decoding a snapshot allocates nothing.
//
// The tradeoff is that this is code we own rather than code we share. It is
// about a hundred lines of counting, it is covered by the parity harness
// against macondo, and it buys a snapshot that decodes without allocating.
package xwordgame

import (
	"fmt"
	"math/rand/v2"

	"github.com/domino14/word-golib/tilemapping"
)

// Rand is the source of randomness for tile draws. *math/rand/v2.Rand
// satisfies it, as does *math/rand/v2.ChaCha8 wrapped in a Rand. Tests inject a
// deterministic source; production uses DefaultRand.
type Rand interface {
	// IntN returns a uniformly distributed value in [0, n). It panics if n <= 0.
	IntN(n int) int
}

// DefaultRand is the source used when a nil Rand is passed. The math/rand/v2
// global source is seeded randomly at process start and is ChaCha8-based, which
// is plenty for dealing tiles.
var DefaultRand Rand = globalRand{}

type globalRand struct{}

func (globalRand) IntN(n int) int { return rand.IntN(n) }

// TilesRemaining returns the number of tiles left in the bag.
func (s *State) TilesRemaining() int {
	n := 0
	for _, c := range s.bag {
		n += int(c)
	}
	return n
}

// BagCount returns how many of the given tile remain in the bag.
func (s *State) BagCount(ml tilemapping.MachineLetter) int {
	if int(ml) >= AlphabetSize {
		return 0
	}
	return int(s.bag[ml])
}

// BagCounts returns a copy of the bag's tile counts, indexed by MachineLetter.
func (s *State) BagCounts() []uint8 {
	out := make([]uint8, AlphabetSize)
	copy(out, s.bag[:])
	return out
}

// FillBag replaces the bag with a full set of tiles from the given letter
// distribution. Blanks are counted as tile 0, consistent with the rest of the
// tilemapping package.
func (s *State) FillBag(ld *tilemapping.LetterDistribution) error {
	s.bag = [AlphabetSize]uint8{}
	for ml, ct := range ld.Distribution() {
		if ml >= AlphabetSize {
			return fmt.Errorf("xwordgame: letter distribution has tile %d beyond alphabet size %d", ml, AlphabetSize)
		}
		s.bag[ml] = ct
	}
	return nil
}

// SetBagCounts replaces the bag's contents wholesale. Mainly for reconstructing
// a state from an external representation (a GameHistory replay, a CGP string,
// a test fixture).
func (s *State) SetBagCounts(counts []uint8) error {
	if len(counts) > AlphabetSize {
		return fmt.Errorf("xwordgame: %d tile counts exceeds alphabet size %d", len(counts), AlphabetSize)
	}
	s.bag = [AlphabetSize]uint8{}
	copy(s.bag[:], counts)
	return nil
}

// Draw removes n tiles from the bag uniformly at random and appends them to
// dst, returning the extended slice. It is an error to draw more tiles than
// remain; use DrawAtMost when a short draw is acceptable.
//
// Passing a nil Rand uses DefaultRand.
func (s *State) Draw(rng Rand, n int, dst []tilemapping.MachineLetter) ([]tilemapping.MachineLetter, error) {
	if n < 0 {
		return dst, fmt.Errorf("xwordgame: cannot draw %d tiles", n)
	}
	if remaining := s.TilesRemaining(); n > remaining {
		return dst, fmt.Errorf("xwordgame: tried to draw %d tiles, bag has %d", n, remaining)
	}
	return s.drawUnchecked(rng, n, dst), nil
}

// DrawAtMost draws up to n tiles, stopping early if the bag empties. It returns
// the extended slice and the number of tiles actually drawn.
func (s *State) DrawAtMost(rng Rand, n int, dst []tilemapping.MachineLetter) ([]tilemapping.MachineLetter, int) {
	if remaining := s.TilesRemaining(); n > remaining {
		n = remaining
	}
	if n <= 0 {
		return dst, 0
	}
	return s.drawUnchecked(rng, n, dst), n
}

// drawUnchecked draws n tiles, assuming the caller has verified the bag holds
// at least that many.
func (s *State) drawUnchecked(rng Rand, n int, dst []tilemapping.MachineLetter) []tilemapping.MachineLetter {
	if rng == nil {
		rng = DefaultRand
	}
	remaining := s.TilesRemaining()
	for range n {
		// Pick the k'th remaining tile and walk the counts to find which letter
		// that is. The alphabet is tiny and bounded, so the walk is cheaper
		// than maintaining any auxiliary index.
		k := rng.IntN(remaining)
		for ml, ct := range s.bag {
			if k < int(ct) {
				s.bag[ml]--
				dst = append(dst, tilemapping.MachineLetter(ml))
				break
			}
			k -= int(ct)
		}
		remaining--
	}
	return dst
}

// PutBack returns tiles to the bag. Blanked letters are normalised back to
// undesignated blanks, since a blank only carries a letter while it is on the
// board.
func (s *State) PutBack(tiles []tilemapping.MachineLetter) error {
	for _, ml := range tiles {
		if ml.IsBlanked() {
			ml = 0
		}
		if int(ml) >= AlphabetSize {
			return fmt.Errorf("xwordgame: cannot return invalid tile %d to the bag", ml)
		}
		s.bag[ml]++
	}
	return nil
}

// Exchange swaps the given tiles for the same number of fresh ones, returning
// the drawn tiles appended to dst.
//
// The order of operations is load-bearing and is the actual rule: the player
// draws FIRST, and only then are the exchanged tiles returned to the bag. Doing
// it the other way round would let a player draw back the very tiles they just
// discarded. Because the bag is a multiset rather than an ordered sequence,
// this is the only ordering question there is -- there is no "where in the bag
// do the returned tiles go", which is precisely why the bag is not stored as a
// sequence. See the note on State.bag.
//
// Exchange does not enforce the minimum-tiles-in-bag rule; that is a variant
// setting and belongs to the referee, which checks it before calling here.
func (s *State) Exchange(rng Rand, tiles []tilemapping.MachineLetter, dst []tilemapping.MachineLetter) ([]tilemapping.MachineLetter, error) {
	drawn, err := s.Draw(rng, len(tiles), dst)
	if err != nil {
		return dst, err
	}
	if err := s.PutBack(tiles); err != nil {
		return dst, err
	}
	return drawn, nil
}

// RemoveFromBag takes specific tiles out of the bag without randomness. It is
// used when reconstructing a state from a known position -- replaying a
// GameHistory, importing a CGP -- where the tiles a player holds are already
// known and must simply be accounted for.
func (s *State) RemoveFromBag(tiles []tilemapping.MachineLetter) error {
	// Verify the whole set is present before mutating, so a failure leaves the
	// bag untouched.
	var want [AlphabetSize]uint8
	for _, ml := range tiles {
		if ml.IsBlanked() {
			ml = 0
		}
		if int(ml) >= AlphabetSize {
			return fmt.Errorf("xwordgame: cannot remove invalid tile %d from the bag", ml)
		}
		want[ml]++
	}
	for ml, ct := range want {
		if s.bag[ml] < ct {
			return fmt.Errorf("xwordgame: bag has %d of tile %d, cannot remove %d", s.bag[ml], ml, ct)
		}
	}
	for ml, ct := range want {
		s.bag[ml] -= ct
	}
	return nil
}
