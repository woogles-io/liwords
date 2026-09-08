package xwordgame

import (
	"testing"

	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
)

func stateWithBag(t *testing.T, counts map[tilemapping.MachineLetter]uint8) *State {
	t.Helper()
	s, err := NewState(15)
	if err != nil {
		t.Fatal(err)
	}
	c := make([]uint8, AlphabetSize)
	for ml, n := range counts {
		c[ml] = n
	}
	if err := s.SetBagCounts(c); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestDrawRemovesFromBag(t *testing.T) {
	is := is.New(t)
	s := stateWithBag(t, map[tilemapping.MachineLetter]uint8{1: 5, 2: 5})
	is.Equal(s.TilesRemaining(), 10)

	drawn, err := s.Draw(seededRand(), 7, nil)
	is.NoErr(err)
	is.Equal(len(drawn), 7)
	is.Equal(s.TilesRemaining(), 3)

	// Everything drawn came out of the bag; nothing was invented.
	for _, ml := range drawn {
		is.True(ml == 1 || ml == 2)
	}
	is.Equal(s.BagCount(1)+s.BagCount(2), 3)
}

func TestDrawMoreThanRemainingFails(t *testing.T) {
	is := is.New(t)
	s := stateWithBag(t, map[tilemapping.MachineLetter]uint8{1: 3})
	_, err := s.Draw(seededRand(), 4, nil)
	is.True(err != nil)
	is.Equal(s.TilesRemaining(), 3) // unchanged
}

func TestDrawAtMostShortDraw(t *testing.T) {
	is := is.New(t)
	s := stateWithBag(t, map[tilemapping.MachineLetter]uint8{1: 3})
	drawn, n := s.DrawAtMost(seededRand(), 7, nil)
	is.Equal(n, 3)
	is.Equal(len(drawn), 3)
	is.Equal(s.TilesRemaining(), 0)

	drawn, n = s.DrawAtMost(seededRand(), 7, drawn)
	is.Equal(n, 0)
	is.Equal(len(drawn), 3)
}

func TestDrawIsUniform(t *testing.T) {
	is := is.New(t)
	// Two tiles at a 3:1 ratio; over many single draws from a fresh bag the
	// observed split should track the ratio. This catches an off-by-one in the
	// counts walk, which would bias toward one end of the alphabet.
	const trials = 40000
	rng := seededRand()
	counts := map[tilemapping.MachineLetter]int{}
	for range trials {
		s := stateWithBag(t, map[tilemapping.MachineLetter]uint8{1: 3, 2: 1})
		drawn, err := s.Draw(rng, 1, nil)
		is.NoErr(err)
		counts[drawn[0]]++
	}
	ratio := float64(counts[1]) / float64(trials)
	t.Logf("tile 1 drawn %.4f of the time (expected 0.75)", ratio)
	is.True(ratio > 0.73 && ratio < 0.77)
}

func TestDrawCanReachEveryTile(t *testing.T) {
	is := is.New(t)
	// The last letter in the alphabet is the one an off-by-one in the walk
	// would make unreachable.
	last := tilemapping.MachineLetter(AlphabetSize - 1)
	s := stateWithBag(t, map[tilemapping.MachineLetter]uint8{0: 1, last: 1})
	drawn, err := s.Draw(seededRand(), 2, nil)
	is.NoErr(err)
	is.Equal(len(drawn), 2)
	is.Equal(s.TilesRemaining(), 0)
	seen := map[tilemapping.MachineLetter]bool{drawn[0]: true, drawn[1]: true}
	is.True(seen[0])
	is.True(seen[last])
}

// The central rule: a player must not be able to draw back the tiles they just
// exchanged. Enforced by drawing before returning.
// FillBag has to hold every distribution we ship, not just english's 26
// letters. Norwegian and polish reach tile 32, and dropping the tiles above 26
// would still leave a bag that looks plausible.
func TestFillBagAcrossDistributions(t *testing.T) {
	is := is.New(t)
	for _, tc := range []struct {
		name       string
		wantTiles  int
		wantMaxLtr tilemapping.MachineLetter
	}{
		{"english", 100, 26},
		{"english_super", 200, 26},
		{"french", 102, 26},
		{"german", 102, 29},
		{"norwegian", 100, 32},
		{"polish", 100, 32},
		{"spanish", 100, 28},
		{"catalan", 100, 26},
		{"slovene", 100, 29},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ld := testLetterDistribution(t, tc.name)
			s, err := NewState(15)
			is.NoErr(err)
			is.NoErr(s.FillBag(ld))

			is.Equal(s.TilesRemaining(), tc.wantTiles)
			// The highest tile in the distribution must actually be in the bag.
			is.True(s.BagCount(tc.wantMaxLtr) > 0)
			// And nothing above it.
			for ml := int(tc.wantMaxLtr) + 1; ml < AlphabetSize; ml++ {
				is.Equal(s.BagCount(tilemapping.MachineLetter(ml)), 0)
			}
			// Every count matches the distribution, tile for tile.
			for ml, ct := range ld.Distribution() {
				is.Equal(s.BagCount(tilemapping.MachineLetter(ml)), int(ct))
			}
			is.NoErr(s.ValidateTileConservation(ld))

			// Filling replaces rather than accumulates.
			is.NoErr(s.FillBag(ld))
			is.Equal(s.TilesRemaining(), tc.wantTiles)
		})
	}
}

// Filling clears whatever was in the bag first. Re-filling with the same
// distribution would not notice -- the counts are assigned, not added -- but
// switching to a distribution with a smaller alphabet would leave the high
// tiles of the previous one behind.
func TestFillBagClearsTheOldDistribution(t *testing.T) {
	is := is.New(t)
	polish := testLetterDistribution(t, "polish")   // reaches tile 32
	english := testLetterDistribution(t, "english") // stops at 26

	s, err := NewState(15)
	is.NoErr(err)
	is.NoErr(s.FillBag(polish))
	is.True(s.BagCount(32) > 0)

	is.NoErr(s.FillBag(english))
	is.Equal(s.BagCount(32), 0)
	is.Equal(s.TilesRemaining(), 100)
	is.NoErr(s.ValidateTileConservation(english))
}

func TestExchangeCannotReturnTheSameTiles(t *testing.T) {
	is := is.New(t)
	// The bag holds only tile 2. The player exchanges three of tile 1, so if
	// the returned tiles went in before the draw, some 1s could come back.
	s := stateWithBag(t, map[tilemapping.MachineLetter]uint8{2: 8})

	discard := []tilemapping.MachineLetter{1, 1, 1}
	drawn, err := s.Exchange(seededRand(), discard, nil)
	is.NoErr(err)
	is.Equal(len(drawn), 3)
	for _, ml := range drawn {
		is.Equal(ml, tilemapping.MachineLetter(2))
	}

	// Bag conserved: 8 - 3 drawn + 3 returned.
	is.Equal(s.TilesRemaining(), 8)
	is.Equal(s.BagCount(1), 3)
	is.Equal(s.BagCount(2), 5)
}

func TestExchangeConservesTiles(t *testing.T) {
	is := is.New(t)
	s := stateWithBag(t, map[tilemapping.MachineLetter]uint8{1: 4, 2: 4, 3: 4})
	before := s.TilesRemaining()

	drawn, err := s.Exchange(seededRand(), []tilemapping.MachineLetter{9, 9, 9, 9, 9}, nil)
	is.NoErr(err)
	is.Equal(len(drawn), 5)
	is.Equal(s.TilesRemaining(), before)
	is.Equal(s.BagCount(9), 5)
}

func TestExchangeFailsWhenBagTooSmall(t *testing.T) {
	is := is.New(t)
	s := stateWithBag(t, map[tilemapping.MachineLetter]uint8{1: 2})
	_, err := s.Exchange(seededRand(), []tilemapping.MachineLetter{5, 5, 5}, nil)
	is.True(err != nil)
	// The failed draw must not have leaked the returned tiles into the bag.
	is.Equal(s.TilesRemaining(), 2)
	is.Equal(s.BagCount(5), 0)
}

func TestPutBackNormalisesBlanks(t *testing.T) {
	is := is.New(t)
	s := stateWithBag(t, nil)
	// A blank recalled from the board still carries its designated letter.
	is.NoErr(s.PutBack([]tilemapping.MachineLetter{5 | tilemapping.BlankMask}))
	is.Equal(s.BagCount(0), 1)
	is.Equal(s.BagCount(5), 0)
}

func TestRemoveFromBag(t *testing.T) {
	is := is.New(t)
	s := stateWithBag(t, map[tilemapping.MachineLetter]uint8{1: 2, 2: 1})

	is.NoErr(s.RemoveFromBag([]tilemapping.MachineLetter{1, 2}))
	is.Equal(s.BagCount(1), 1)
	is.Equal(s.BagCount(2), 0)

	// A partially-satisfiable removal must be all-or-nothing.
	err := s.RemoveFromBag([]tilemapping.MachineLetter{1, 2})
	is.True(err != nil)
	is.Equal(s.BagCount(1), 1)
}

func TestRemoveFromBagNormalisesBlanks(t *testing.T) {
	is := is.New(t)
	s := stateWithBag(t, map[tilemapping.MachineLetter]uint8{0: 1})
	is.NoErr(s.RemoveFromBag([]tilemapping.MachineLetter{5 | tilemapping.BlankMask}))
	is.Equal(s.BagCount(0), 0)
}

func TestFillBagMatchesDistribution(t *testing.T) {
	is := is.New(t)
	ld := testLetterDistribution(t, "english")

	s, err := NewState(15)
	is.NoErr(err)
	is.NoErr(s.FillBag(ld))
	is.Equal(s.TilesRemaining(), int(ld.NumTotalLetters()))
	is.Equal(s.BagCount(0), 2) // two blanks in English
}

func TestFillBagSuperDistribution(t *testing.T) {
	is := is.New(t)
	ld := testLetterDistribution(t, "english_super")

	s, err := NewState(MaxBoardDim)
	is.NoErr(err)
	is.NoErr(s.FillBag(ld))
	is.Equal(s.TilesRemaining(), int(ld.NumTotalLetters()))

	// A full super bag survives a clone with every tile intact. There used to
	// be a byte-size assertion here too, back when a position was stored in a
	// column; there is no stored form any more.
	got := s.Clone()
	is.Equal(got.TilesRemaining(), s.TilesRemaining())
	is.True(got.Equal(s))
}

// Drawing the entire bag one tile at a time must yield exactly the starting
// multiset -- no tile invented, none lost.
func TestDrainBagConservesEveryTile(t *testing.T) {
	is := is.New(t)
	ld := testLetterDistribution(t, "english")
	s, err := NewState(15)
	is.NoErr(err)
	is.NoErr(s.FillBag(ld))

	want := s.BagCounts()
	got := make([]uint8, AlphabetSize)
	rng := seededRand()
	for s.TilesRemaining() > 0 {
		drawn, err := s.Draw(rng, 1, nil)
		is.NoErr(err)
		got[drawn[0]]++
	}
	is.Equal(got, want)
}

func BenchmarkDrawSeven(b *testing.B) {
	ld, err := tilemapping.NamedLetterDistribution(&testConfig, "english")
	if err != nil {
		b.Fatal(err)
	}
	s, _ := NewState(15)
	rng := seededRand()
	buf := make([]tilemapping.MachineLetter, 0, RackTileLimit)
	b.ReportAllocs()
	for b.Loop() {
		_ = s.FillBag(ld)
		buf, _ = s.DrawAtMost(rng, RackTileLimit, buf[:0])
	}
}
