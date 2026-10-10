package xwordgame

// Differential tests against macondo.
//
// Scoring in this package is a reimplementation, not a transcription: macondo
// reads a cached per-square cross-score that we compute on the fly. These tests
// are the evidence that the two agree. They are the in-package counterpart to
// the corpus replay harness described in docs/mikado/referee_gap_audit.md.
//
// macondo is a test-only dependency here.

import (
	"math/rand/v2"
	"testing"

	macondoboard "github.com/domino14/macondo/board"
	"github.com/domino14/macondo/cross_set"
	macondomove "github.com/domino14/macondo/move"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
)

type parityFixture struct {
	layout *BoardLayout
	ld     *tilemapping.LetterDistribution
	alph   *tilemapping.TileMapping
	state  *State
	mboard *macondoboard.GameBoard
	maxLtr tilemapping.MachineLetter
}

func newParityFixture(t *testing.T, layoutName, distName string, desc []string) *parityFixture {
	t.Helper()
	layout, err := NamedLayout(layoutName)
	if err != nil {
		t.Fatal(err)
	}
	ld := testLetterDistribution(t, distName)
	st, err := NewState(layout.Dim())
	if err != nil {
		t.Fatal(err)
	}
	// Highest real tile in this distribution.
	var maxLtr tilemapping.MachineLetter
	for ml, ct := range ld.Distribution() {
		if ct > 0 && ml > 0 {
			maxLtr = tilemapping.MachineLetter(ml)
		}
	}
	return &parityFixture{
		layout: layout, ld: ld, alph: ld.TileMapping(), state: st,
		mboard: macondoboard.MakeBoard(desc), maxLtr: maxLtr,
	}
}

// setTile writes the same tile to both boards.
func (f *parityFixture) setTile(row, col int, ml tilemapping.MachineLetter) {
	f.state.SetTileAt(row, col, ml)
	f.mboard.SetLetter(row, col, ml)
}

// syncCrossScores refreshes macondo's cached cross-scores. Ours needs no
// equivalent, which is the whole point.
func (f *parityFixture) syncCrossScores() {
	cross_set.GenAllCrossScores(f.mboard, f.ld)
}

// macondoScore replicates macondo's scoring path exactly as
// game.CreateAndScorePlacementMove does it (game.go:874): transpose for
// vertical plays, swap the coordinates, score with the opposite cross
// direction, transpose back.
func (f *parityFixture) macondoScore(m *Move) int {
	row, col := m.Row, m.Col
	crossDir := macondoboard.VerticalDirection
	if m.Vertical {
		crossDir = macondoboard.HorizontalDirection
		row, col = col, row
		f.mboard.Transpose()
	}
	score := f.mboard.ScoreWord(m.Tiles, row, col, m.TilesPlayed(), crossDir, f.ld)
	if m.Vertical {
		f.mboard.Transpose()
	}
	return score
}

func (f *parityFixture) macondoMove(m *Move) *macondomove.Move {
	return macondomove.NewScoringMove(0, m.Tiles, nil, m.Vertical,
		m.TilesPlayed(), f.alph, m.Row, m.Col)
}

// randomBoard scatters tiles over both boards identically.
func (f *parityFixture) randomBoard(rng *rand.Rand, nTiles int) {
	dim := f.layout.Dim()
	for range nTiles {
		row, col := rng.IntN(dim), rng.IntN(dim)
		ml := tilemapping.MachineLetter(1 + rng.IntN(int(f.maxLtr)))
		if rng.IntN(10) == 0 {
			ml |= tilemapping.BlankMask // a played blank scores zero
		}
		f.setTile(row, col, ml)
	}
	// macondo's IsEmpty() reads a cached tilesPlayed counter rather than the
	// squares, and SetLetter does not maintain it. Ours scans the board. Sync
	// the counter so both engines agree the board is non-empty.
	f.mboard.TestSetTilesPlayed(f.state.TilesOnBoard())
	f.syncCrossScores()
}

// randomPlay builds a candidate placement. It may well be illegal; both engines
// get asked, and they are expected to agree.
func (f *parityFixture) randomPlay(rng *rand.Rand) *Move {
	dim := f.layout.Dim()
	vertical := rng.IntN(2) == 0
	// Deliberately includes length 1, which is always illegal ("no one-letter
	// words"), so the legality differential covers that rule too.
	length := 1 + rng.IntN(RackTileLimit)
	dRow, dCol := 0, 1
	if vertical {
		dRow, dCol = 1, 0
	}
	maxStart := max(dim-length, 0)
	row, col := rng.IntN(dim), rng.IntN(dim)
	if vertical {
		row = rng.IntN(maxStart + 1)
	} else {
		col = rng.IntN(maxStart + 1)
	}

	tiles := make(tilemapping.MachineWord, 0, length)
	for i := range length {
		r, c := row+dRow*i, col+dCol*i
		if !f.state.PosExists(r, c) {
			break
		}
		if f.state.HasLetter(r, c) {
			tiles = append(tiles, 0) // played through
			continue
		}
		ml := tilemapping.MachineLetter(1 + rng.IntN(int(f.maxLtr)))
		if rng.IntN(12) == 0 {
			ml |= tilemapping.BlankMask
		}
		tiles = append(tiles, ml)
	}
	return NewPlacementMove(row, col, vertical, tiles)
}

// The core differential: same board, same play, same score.
func TestParityScoringAgainstMacondo(t *testing.T) {
	is := is.New(t)
	for _, tc := range []struct {
		name, layout, dist string
		desc               []string
	}{
		{"classic", CrosswordGameLayout, "english", macondoboard.CrosswordGameBoard},
		{"super", SuperCrosswordGameLayout, "english_super", macondoboard.SuperCrosswordGameBoard},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rng := rand.New(rand.NewPCG(7, 11))
			compared, legalAgreed := 0, 0

			for iter := range 4000 {
				f := newParityFixture(t, tc.layout, tc.dist, tc.desc)
				f.randomBoard(rng, 1+rng.IntN(60))
				m := f.randomPlay(rng)

				ourErr := f.state.ErrorIfIllegalPlay(f.layout, m)
				theirErr := f.mboard.ErrorIfIllegalPlay(m.Row, m.Col, m.Vertical, m.Tiles)

				// Legality verdicts must match. The messages may differ; the
				// accept/reject decision may not.
				if (ourErr == nil) != (theirErr == nil) {
					t.Fatalf("iter %d: legality disagreement: ours=%v theirs=%v (row %d col %d vert %v tiles %v)",
						iter, ourErr, theirErr, m.Row, m.Col, m.Vertical, m.Tiles)
				}
				legalAgreed++
				if ourErr != nil {
					continue
				}

				ourScore, err := f.state.ScoreMove(f.layout, f.ld, m)
				is.NoErr(err)
				theirScore := f.macondoScore(m)
				if ourScore != theirScore {
					t.Fatalf("iter %d: score %d != macondo %d (row %d col %d vert %v tiles %v)",
						iter, ourScore, theirScore, m.Row, m.Col, m.Vertical, m.Tiles)
				}

				// And the words formed must match, since a challenge depends on
				// them.
				ourWords, err := f.state.FormedWords(m)
				is.NoErr(err)
				theirWords, err := f.mboard.FormedWords(f.macondoMove(m))
				is.NoErr(err)
				if len(ourWords) != len(theirWords) {
					t.Fatalf("iter %d: formed %d words, macondo formed %d: %v vs %v",
						iter, len(ourWords), len(theirWords), ourWords, theirWords)
				}
				for i := range ourWords {
					if string(ourWords[i]) != string(theirWords[i]) {
						t.Fatalf("iter %d: word %d = %v, macondo says %v", iter, i, ourWords[i], theirWords[i])
					}
				}
				compared++
			}
			t.Logf("%s: %d legality verdicts agreed, %d scored plays compared", tc.name, legalAgreed, compared)
			is.True(compared > 200) // the generator must actually be producing legal plays
		})
	}
}

// Opening plays exercise the centre-square rule and the empty-board branch,
// which the random-board test rarely reaches.
func TestParityOpeningPlays(t *testing.T) {
	is := is.New(t)
	for _, tc := range []struct {
		name, layout, dist string
		desc               []string
	}{
		{"classic", CrosswordGameLayout, "english", macondoboard.CrosswordGameBoard},
		{"super", SuperCrosswordGameLayout, "english_super", macondoboard.SuperCrosswordGameBoard},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rng := rand.New(rand.NewPCG(3, 5))
			checked := 0
			for range 2000 {
				f := newParityFixture(t, tc.layout, tc.dist, tc.desc)
				f.syncCrossScores()
				m := f.randomPlay(rng)
				ourErr := f.state.ErrorIfIllegalPlay(f.layout, m)
				theirErr := f.mboard.ErrorIfIllegalPlay(m.Row, m.Col, m.Vertical, m.Tiles)
				if (ourErr == nil) != (theirErr == nil) {
					t.Fatalf("opening legality disagreement: ours=%v theirs=%v", ourErr, theirErr)
				}
				if ourErr != nil {
					continue
				}
				ourScore, err := f.state.ScoreMove(f.layout, f.ld, m)
				is.NoErr(err)
				is.Equal(ourScore, f.macondoScore(m))
				checked++
			}
			t.Logf("%s: %d opening plays compared", tc.name, checked)
			is.True(checked > 20)
		})
	}
}

// Play sequences: score, place, repeat. This is closer to how a real game
// evolves than scattered tiles, and it catches errors that only appear once
// plays start hooking onto previous ones.
func TestParitySequentialPlays(t *testing.T) {
	is := is.New(t)
	rng := rand.New(rand.NewPCG(13, 17))
	totalPlays := 0

	for range 400 {
		f := newParityFixture(t, CrosswordGameLayout, "english", macondoboard.CrosswordGameBoard)
		f.syncCrossScores()

		for range 25 {
			m := f.randomPlay(rng)
			if f.state.ErrorIfIllegalPlay(f.layout, m) != nil {
				continue
			}
			is.Equal(f.mboard.ErrorIfIllegalPlay(m.Row, m.Col, m.Vertical, m.Tiles), nil)

			ourScore, err := f.state.ScoreMove(f.layout, f.ld, m)
			is.NoErr(err)
			if theirScore := f.macondoScore(m); ourScore != theirScore {
				t.Fatalf("sequential play score %d != macondo %d (row %d col %d vert %v tiles %v)",
					ourScore, theirScore, m.Row, m.Col, m.Vertical, m.Tiles)
			}

			f.state.PlaceMoveTiles(m)
			f.mboard.PlayMove(f.macondoMove(m))
			f.syncCrossScores()

			// The two boards must stay identical square for square.
			for r := range f.layout.Dim() {
				for c := range f.layout.Dim() {
					if f.state.TileAt(r, c) != f.mboard.GetLetter(r, c) {
						t.Fatalf("board divergence at row %d col %d: %d vs %d",
							r, c, f.state.TileAt(r, c), f.mboard.GetLetter(r, c))
					}
				}
			}
			is.Equal(f.state.TilesOnBoard(), f.mboard.TilesPlayed())
			totalPlays++
		}
	}
	t.Logf("%d sequential plays compared", totalPlays)
	is.True(totalPlays > 500)
}
