package xwordgame

import (
	"math/rand/v2"
	"testing"

	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
)

// fullState builds a state exercising every field, on a board of the given
// dimension.
func fullState(t *testing.T, dim int) *State {
	t.Helper()
	s, err := NewState(dim)
	if err != nil {
		t.Fatal(err)
	}
	s.PlayState = WaitingForFinalPass
	s.OnTurn = 1
	s.TurnNum = 37
	s.ScorelessTurns = 3
	s.Scores = [MaxPlayers]int32{412, 389}
	s.Bingos = [MaxPlayers]uint16{2, 1}
	s.PlayerTurns = [MaxPlayers]uint16{19, 18}

	// A handful of tiles, including a played blank (high bit set).
	s.SetTileAt(7, 7, 3)
	s.SetTileAt(7, 8, 1)
	s.SetTileAt(7, 9, 20|tilemapping.BlankMask)
	s.SetTileAt(dim-1, dim-1, 26)

	if err := s.SetRack(0, []tilemapping.MachineLetter{1, 5, 9, 15, 21, 0, 26}); err != nil {
		t.Fatal(err)
	}
	// A short rack, endgame style.
	if err := s.SetRack(1, []tilemapping.MachineLetter{17, 22}); err != nil {
		t.Fatal(err)
	}

	counts := make([]uint8, AlphabetSize)
	counts[0] = 2
	counts[1] = 9
	counts[5] = 12
	counts[26] = 1
	if err := s.SetBagCounts(counts); err != nil {
		t.Fatal(err)
	}

	s.LastWordsFormed = []tilemapping.MachineWord{
		{3, 1, 20},
		{17, 21, 9, 26},
	}
	return s
}

func TestCloneCopiesEverything(t *testing.T) {
	is := is.New(t)
	orig := fullState(t, 15)

	got := orig.Clone()
	is.Equal(got.Digest(), orig.Digest())
	is.True(got.Equal(orig))

	// Spot-check that the fields actually came across, not just that two
	// identically-broken digests match.
	is.Equal(got.Dim(), 15)
	is.Equal(got.PlayState, WaitingForFinalPass)
	is.Equal(got.OnTurn, uint8(1))
	is.Equal(got.TurnNum, uint16(37))
	is.Equal(got.ScorelessTurns, uint8(3))
	is.Equal(got.Scores, [MaxPlayers]int32{412, 389})
	is.Equal(got.Bingos, [MaxPlayers]uint16{2, 1})
	is.Equal(got.PlayerTurns, [MaxPlayers]uint16{19, 18})
	is.Equal(got.TileAt(7, 9), tilemapping.MachineLetter(20|tilemapping.BlankMask))
	is.Equal(got.TileAt(14, 14), tilemapping.MachineLetter(26))
	// Racks are stored canonically sorted, so the blank leads regardless of the
	// order it was set in. See the note in rack.go.
	is.Equal(got.Rack(0), []tilemapping.MachineLetter{0, 1, 5, 9, 15, 21, 26})
	is.Equal(got.Rack(1), []tilemapping.MachineLetter{17, 22})
	is.Equal(got.TilesRemaining(), 24)
	is.Equal(got.LastWordsFormed, orig.LastWordsFormed)
	// That the clone is a copy rather than an alias is TestCloneIsDeep's job.
}

func TestClone21x21SuperBoard(t *testing.T) {
	is := is.New(t)
	orig := fullState(t, MaxBoardDim)

	// Fill the board densely to hit the worst case.
	for i := range orig.Board() {
		orig.Board()[i] = tilemapping.MachineLetter(i%26 + 1)
	}
	// A 200-tile super-english bag.
	counts := make([]uint8, AlphabetSize)
	for i := 1; i <= 25; i++ {
		counts[i] = 8
	}
	is.NoErr(orig.SetBagCounts(counts))
	is.Equal(orig.TilesRemaining(), 200)

	got := orig.Clone()
	is.Equal(got.Dim(), MaxBoardDim)
	is.Equal(got.Digest(), orig.Digest())
	is.Equal(got.TilesRemaining(), 200)
	is.True(got.Equal(orig))
}

// Digest has to discriminate, not merely be stable. Equal is built on it, and a
// digest that ignored a field would make two different positions compare equal
// -- which is the kind of quiet wrongness this package exists to avoid. So
// change one thing at a time and require the digest to move.
func TestDigestDiscriminates(t *testing.T) {
	base := fullState(t, 15)

	for _, tc := range []struct {
		name  string
		apply func(*State)
	}{
		{"play state", func(s *State) { s.PlayState = Playing }},
		{"on turn", func(s *State) { s.OnTurn = 0 }},
		{"turn number", func(s *State) { s.TurnNum++ }},
		{"scoreless turns", func(s *State) { s.ScorelessTurns++ }},
		{"score", func(s *State) { s.Scores[0]++ }},
		{"other score", func(s *State) { s.Scores[1]++ }},
		{"bingos", func(s *State) { s.Bingos[1]++ }},
		{"player turns", func(s *State) { s.PlayerTurns[0]++ }},
		{"a board square", func(s *State) { s.Board()[0] = 7 }},
		{"a rack", func(s *State) {
			if err := s.SetRack(0, []tilemapping.MachineLetter{1, 2, 3}); err != nil {
				panic(err)
			}
		}},
		{"the bag", func(s *State) {
			counts := make([]uint8, AlphabetSize)
			counts[3] = 5
			if err := s.SetBagCounts(counts); err != nil {
				panic(err)
			}
		}},
		{"last words formed", func(s *State) {
			s.LastWordsFormed = []tilemapping.MachineWord{{1, 2}}
		}},
		{"no last words at all", func(s *State) { s.LastWordsFormed = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			mutated := base.Clone()
			tc.apply(mutated)
			is.True(mutated.Digest() != base.Digest())
			is.True(!mutated.Equal(base))
		})
	}
}

func TestEmptyBagIsRepresentable(t *testing.T) {
	is := is.New(t)
	s, err := NewState(15)
	is.NoErr(err)
	is.Equal(s.TilesRemaining(), 0)

	got := s.Clone()
	is.Equal(got.TilesRemaining(), 0)
	is.Equal(got.Digest(), s.Digest())
}

// A blank on a rack is the zero tile; a blank on the board carries a designated
// letter in its low bits. Both must survive, and they must stay distinct.
func TestBlanksAreDistinguishable(t *testing.T) {
	is := is.New(t)
	s, err := NewState(15)
	is.NoErr(err)
	is.NoErr(s.SetRack(0, []tilemapping.MachineLetter{0, 0}))
	s.SetTileAt(0, 0, 5|tilemapping.BlankMask)
	s.SetTileAt(0, 1, 5)

	got := s.Clone()
	is.Equal(got.Rack(0), []tilemapping.MachineLetter{0, 0})
	is.True(got.TileAt(0, 0).IsBlanked())
	is.True(!got.TileAt(0, 1).IsBlanked())
	is.Equal(got.TileAt(0, 0).Unblank(), got.TileAt(0, 1))
}

// Polish has 31 distinct tiles, well above English's 27; Catalan has multi-rune
// tiles. Both are just larger machine-letter values as far as the codec cares.
func TestLargeAlphabetIsRepresentable(t *testing.T) {
	is := is.New(t)
	s, err := NewState(15)
	is.NoErr(err)
	counts := make([]uint8, AlphabetSize)
	for i := range AlphabetSize {
		counts[i] = 1
	}
	is.NoErr(s.SetBagCounts(counts))
	s.SetTileAt(0, 0, AlphabetSize-1)
	is.NoErr(s.SetRack(0, []tilemapping.MachineLetter{AlphabetSize - 1, AlphabetSize - 2}))

	got := s.Clone()
	is.Equal(got.TilesRemaining(), AlphabetSize)
	is.Equal(got.TileAt(0, 0), tilemapping.MachineLetter(AlphabetSize-1))
}

func TestCloneIsDeep(t *testing.T) {
	is := is.New(t)
	orig := fullState(t, 15)
	c := orig.Clone()
	is.Equal(c.Digest(), orig.Digest())

	c.SetTileAt(0, 0, 12)
	c.LastWordsFormed[0][0] = 9
	is.NoErr(c.PutBack([]tilemapping.MachineLetter{1}))
	is.True(c.Digest() != orig.Digest())
	is.Equal(orig.TileAt(0, 0), tilemapping.MachineLetter(0))
	is.Equal(orig.LastWordsFormed[0][0], tilemapping.MachineLetter(3))
}

func TestNewStateRejectsBadDimension(t *testing.T) {
	is := is.New(t)
	for _, dim := range []int{0, -1, MaxBoardDim + 1, 100} {
		_, err := NewState(dim)
		is.True(err != nil)
	}
}

func FuzzDigestSurvivesAClone(f *testing.F) {
	f.Add(15, uint8(0), uint8(0), uint16(0), uint8(0), int32(0), int32(0))
	f.Add(21, uint8(2), uint8(1), uint16(65535), uint8(6), int32(-5), int32(999))

	f.Fuzz(func(t *testing.T, dim int, ps, onTurn uint8, turnNum uint16, scoreless uint8, s0, s1 int32) {
		dim = 1 + (dim%MaxBoardDim+MaxBoardDim)%MaxBoardDim
		st, err := NewState(dim)
		if err != nil {
			t.Fatal(err)
		}
		st.PlayState = PlayState(ps % (uint8(maxPlayState) + 1))
		st.OnTurn = onTurn % MaxPlayers
		st.TurnNum = turnNum
		st.ScorelessTurns = scoreless
		st.Scores = [MaxPlayers]int32{s0, s1}

		got := st.Clone()
		if got.Digest() != st.Digest() {
			t.Fatalf("clone digest differs for dim=%d", dim)
		}
		if got.Scores != st.Scores {
			t.Fatalf("scores %v != %v", got.Scores, st.Scores)
		}
	})
}

// seededRand is a deterministic Rand for tests.
func seededRand() Rand { return rand.New(rand.NewPCG(1, 2)) }
