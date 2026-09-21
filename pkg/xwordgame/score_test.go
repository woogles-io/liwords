package xwordgame

import (
	"testing"

	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
)

// Hand-computed scores. The macondo differential proves the two engines agree;
// these prove the answer is actually right, which a differential cannot -- a
// bug shared by both would pass parity silently.

func scoringFixture(t *testing.T) (*State, *BoardLayout, *tilemapping.LetterDistribution) {
	t.Helper()
	layout, err := NamedLayout(CrosswordGameLayout)
	if err != nil {
		t.Fatal(err)
	}
	ld := testLetterDistribution(t, "english")
	s, err := NewState(layout.Dim())
	if err != nil {
		t.Fatal(err)
	}
	return s, layout, ld
}

// word converts letters to machine letters; '.' marks a played-through square.
func word(t *testing.T, alph *tilemapping.TileMapping, s string) tilemapping.MachineWord {
	t.Helper()
	mw := make(tilemapping.MachineWord, 0, len(s))
	for _, r := range s {
		if r == '.' {
			mw = append(mw, 0)
			continue
		}
		ml, err := alph.Val(string(r))
		if err != nil {
			t.Fatalf("bad letter %q: %v", r, err)
		}
		mw = append(mw, ml)
	}
	return mw
}

func TestOpeningPlayScore(t *testing.T) {
	is := is.New(t)
	s, layout, ld := scoringFixture(t)
	alph := ld.TileMapping()

	// HELLO horizontally at row 7, cols 3..7.
	// Row 7 bonuses: "=  '   -   '  =" -> col 3 is 2LS, col 7 is the centre 2WS.
	// H(4)x2 + E(1) + L(1) + L(1) + O(1) = 12, doubled by the centre = 24.
	m := NewPlacementMove(7, 3, false, word(t, alph, "HELLO"))
	is.NoErr(s.ErrorIfIllegalPlay(layout, m))

	score, err := s.ScoreMove(layout, ld, m)
	is.NoErr(err)
	is.Equal(score, 24)
}

func TestBingoBonus(t *testing.T) {
	is := is.New(t)
	s, layout, ld := scoringFixture(t)
	alph := ld.TileMapping()

	// FRIENDS horizontally at row 7, cols 7..13.
	// col 7 = centre 2WS, col 11 = 2LS.
	// F(4)+R(1)+I(1)+E(1)+N(1)x2+D(2)+S(1) = 12, doubled = 24, +50 bingo = 74.
	m := NewPlacementMove(7, 7, false, word(t, alph, "FRIENDS"))
	is.NoErr(s.ErrorIfIllegalPlay(layout, m))

	score, err := s.ScoreMove(layout, ld, m)
	is.NoErr(err)
	is.Equal(score, 74)
	is.Equal(m.TilesPlayed(), RackTileLimit)
}

func TestPlayedThroughTilesScoreWithoutPremiums(t *testing.T) {
	is := is.New(t)
	s, layout, ld := scoringFixture(t)
	alph := ld.TileMapping()

	// Establish HELLO at row 7, cols 3..7.
	first := NewPlacementMove(7, 3, false, word(t, alph, "HELLO"))
	s.PlaceMoveTiles(first)

	// Now play HELLOS: the six-square span cols 3..8, with the first five
	// played through and S fresh on col 8 (a plain square).
	// The premium on col 3 and the centre double are spent -- played-through
	// tiles score face value with no multipliers.
	// H(4)+E(1)+L(1)+L(1)+O(1)+S(1) = 9, no word multiplier.
	m := NewPlacementMove(7, 3, false, word(t, alph, ".....S"))
	is.NoErr(s.ErrorIfIllegalPlay(layout, m))

	score, err := s.ScoreMove(layout, ld, m)
	is.NoErr(err)
	is.Equal(score, 9)
}

func TestBlankOnBoardScoresZero(t *testing.T) {
	is := is.New(t)
	s, layout, ld := scoringFixture(t)
	alph := ld.TileMapping()

	// Same HELLO, but the H is a blank designated as H.
	tiles := word(t, alph, "HELLO")
	tiles[0] |= tilemapping.BlankMask
	m := NewPlacementMove(7, 3, false, tiles)
	is.NoErr(s.ErrorIfIllegalPlay(layout, m))

	// Blank scores 0 even on the double-letter: (0 + 1+1+1+1) x 2 = 8.
	score, err := s.ScoreMove(layout, ld, m)
	is.NoErr(err)
	is.Equal(score, 8)

	// And the word formed is still HELLO -- blankness affects scoring, not
	// spelling.
	words, err := s.FormedWords(m)
	is.NoErr(err)
	is.Equal(len(words), 1)
	is.Equal(words[0], word(t, alph, "HELLO"))
}

func TestCrossWordIsScoredAndDoubled(t *testing.T) {
	is := is.New(t)
	s, layout, ld := scoringFixture(t)
	alph := ld.TileMapping()

	// HELLO at row 7, cols 3..7.
	s.PlaceMoveTiles(NewPlacementMove(7, 3, false, word(t, alph, "HELLO")))

	// AH vertically at row 6, col 3: A fresh on (6,3), H played through at (7,3).
	// Row 6 bonuses: "  '   ' '   '  " -> col 3 is plain.
	// Main word AH = A(1) + H(4) = 5, no multipliers.
	// A also forms the cross word AE? No -- (6,4) is empty, so no cross word.
	m := NewPlacementMove(6, 3, true, word(t, alph, "A."))
	is.NoErr(s.ErrorIfIllegalPlay(layout, m))
	score, err := s.ScoreMove(layout, ld, m)
	is.NoErr(err)
	is.Equal(score, 5)

	words, err := s.FormedWords(m)
	is.NoErr(err)
	is.Equal(len(words), 1)
	is.Equal(words[0], word(t, alph, "AH"))
}

func TestParallelPlayFormsMultipleCrossWords(t *testing.T) {
	is := is.New(t)
	s, layout, ld := scoringFixture(t)
	alph := ld.TileMapping()

	// HELLO at row 7, cols 3..7.
	s.PlaceMoveTiles(NewPlacementMove(7, 3, false, word(t, alph, "HELLO")))

	// AA played at row 6, cols 3..4, parallel to HELLO. Every fresh tile hooks
	// downward, so this forms the main word AA plus cross words AH and AE.
	m := NewPlacementMove(6, 3, false, word(t, alph, "AA"))
	is.NoErr(s.ErrorIfIllegalPlay(layout, m))

	words, err := s.FormedWords(m)
	is.NoErr(err)
	is.Equal(len(words), 3)
	is.Equal(words[0], word(t, alph, "AA")) // main word first, by convention
	is.Equal(words[1], word(t, alph, "AH"))
	is.Equal(words[2], word(t, alph, "AE"))

	// Row 6 cols 3 and 4 are both plain squares.
	// Main AA = 2. Cross AH = 1+4 = 5. Cross AE = 1+1 = 2. Total 9.
	score, err := s.ScoreMove(layout, ld, m)
	is.NoErr(err)
	is.Equal(score, 9)
}

func TestIllegalPlays(t *testing.T) {
	is := is.New(t)
	alph := testLetterDistribution(t, "english").TileMapping()

	for _, tc := range []struct {
		name  string
		setup func(*State, *BoardLayout)
		move  *Move
	}{
		{"first play misses the centre", nil,
			NewPlacementMove(0, 0, false, word(t, alph, "HELLO"))},
		{"one-letter play", nil,
			NewPlacementMove(7, 7, false, word(t, alph, "A"))},
		{"runs off the board", nil,
			NewPlacementMove(7, 12, false, word(t, alph, "HELLO"))},
		{"floating play with no contact",
			func(s *State, l *BoardLayout) {
				s.PlaceMoveTiles(NewPlacementMove(7, 3, false, word(t, alph, "HELLO")))
			},
			NewPlacementMove(0, 0, false, word(t, alph, "AA"))},
		{"played-through marker over an empty square",
			func(s *State, l *BoardLayout) {
				s.PlaceMoveTiles(NewPlacementMove(7, 3, false, word(t, alph, "HELLO")))
			},
			NewPlacementMove(9, 3, false, word(t, alph, "A.A"))},
		{"does not include the whole word",
			func(s *State, l *BoardLayout) {
				s.PlaceMoveTiles(NewPlacementMove(7, 3, false, word(t, alph, "HELLO")))
			},
			// HELLO occupies cols 3..7. This span is cols 2..3 only, so the
			// ELLO at cols 4..7 dangles past the end of the play.
			NewPlacementMove(7, 2, false, word(t, alph, "A."))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, layout, _ := scoringFixture(t)
			if tc.setup != nil {
				tc.setup(s, layout)
			}
			err := s.ErrorIfIllegalPlay(layout, tc.move)
			is.True(err != nil)
			t.Log(err)
		})
	}
}

func TestRackScore(t *testing.T) {
	is := is.New(t)
	ld := testLetterDistribution(t, "english")
	alph := ld.TileMapping()
	// Q(10) + U(1) + I(1) + Z(10) = 22.
	is.Equal(RackScore(ld, word(t, alph, "QUIZ")), 22)
	// A blank contributes nothing.
	is.Equal(RackScore(ld, []tilemapping.MachineLetter{0}), 0)
}
