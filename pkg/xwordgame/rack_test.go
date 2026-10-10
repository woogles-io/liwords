package xwordgame

import (
	"math/rand/v2"
	"testing"

	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
)

// rackFixture is a state with a full english bag and an empty 15x15 board.
func rackFixture(t *testing.T) (*State, *tilemapping.LetterDistribution) {
	t.Helper()
	ld := testLetterDistribution(t, "english")
	s, err := NewState(15)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FillBag(ld); err != nil {
		t.Fatal(err)
	}
	return s, ld
}

func TestRacksAreStoredSorted(t *testing.T) {
	is := is.New(t)
	s, ld := rackFixture(t)
	alph := ld.TileMapping()

	is.NoErr(s.SetRack(0, word(t, alph, "ZEBRA")))
	// A(1) B(2) E(5) R(18) Z(26)
	is.Equal(s.Rack(0), []tilemapping.MachineLetter{1, 2, 5, 18, 26})

	// The order tiles arrive in must not survive into the state, or two
	// identical positions would hash differently.
	other, err := NewState(15)
	is.NoErr(err)
	is.NoErr(other.FillBag(ld))
	is.NoErr(other.SetRack(0, word(t, alph, "BAZER")))
	is.Equal(other.Rack(0), s.Rack(0))
	is.Equal(other.Digest(), s.Digest())
}

func TestSetRackNormalizesBlanks(t *testing.T) {
	is := is.New(t)
	s, ld := rackFixture(t)
	alph := ld.TileMapping()

	// A blank designated as H is only a thing on the board. On a rack it is
	// just a blank.
	tiles := word(t, alph, "HAT")
	tiles[0] |= tilemapping.BlankMask
	is.NoErr(s.SetRack(0, tiles))
	is.Equal(s.Rack(0), []tilemapping.MachineLetter{0, 1, 20})
	is.NoErr(s.Validate())
}

func TestTakeFromRack(t *testing.T) {
	is := is.New(t)
	s, ld := rackFixture(t)
	alph := ld.TileMapping()
	is.NoErr(s.SetRack(0, word(t, alph, "AABCDE")))

	// Duplicates are counted, not just membership.
	is.True(s.RackHas(0, word(t, alph, "AA")))
	is.True(!s.RackHas(0, word(t, alph, "AAA")))

	is.NoErr(s.TakeFromRack(0, word(t, alph, "AB")))
	is.Equal(s.Rack(0), []tilemapping.MachineLetter(word(t, alph, "ACDE")))

	// A failed take leaves the rack alone.
	before := append([]tilemapping.MachineLetter(nil), s.Rack(0)...)
	err := s.TakeFromRack(0, word(t, alph, "AZ"))
	is.True(err != nil)
	is.Equal(s.Rack(0), before)
}

func TestTakeFromRackTakesPlayedBlanksAsBlanks(t *testing.T) {
	is := is.New(t)
	s, ld := rackFixture(t)
	alph := ld.TileMapping()
	is.NoErr(s.SetRack(0, []tilemapping.MachineLetter{0, 1, 20})) // ?AT

	// The play spells HAT using the blank as H. The rack gives up a blank.
	played := word(t, alph, "HAT")
	played[0] |= tilemapping.BlankMask
	is.NoErr(s.TakeFromRack(0, played))
	is.Equal(s.RackLen(0), 0)
}

func TestDrawToFull(t *testing.T) {
	is := is.New(t)
	s, _ := rackFixture(t)
	rng := rand.New(rand.NewPCG(1, 2))

	n, err := s.DrawToFull(rng, 0)
	is.NoErr(err)
	is.Equal(n, RackTileLimit)
	is.Equal(s.RackLen(0), RackTileLimit)
	is.Equal(s.TilesRemaining(), 100-RackTileLimit)

	// Already full: nothing happens.
	n, err = s.DrawToFull(rng, 0)
	is.NoErr(err)
	is.Equal(n, 0)

	// Near the end of the bag it takes what is left rather than failing.
	s2, _ := rackFixture(t)
	var scratch []tilemapping.MachineLetter
	scratch, err = s2.Draw(rng, 96, scratch)
	is.NoErr(err)
	is.Equal(len(scratch), 96)
	n, err = s2.DrawToFull(rng, 1)
	is.NoErr(err)
	is.Equal(n, 4)
	is.Equal(s2.TilesRemaining(), 0)
}

func TestAssignRackMovesTilesOutOfTheBag(t *testing.T) {
	is := is.New(t)
	s, ld := rackFixture(t)
	alph := ld.TileMapping()

	is.NoErr(s.AssignRack(0, word(t, alph, "QUIZTOP")))
	is.Equal(s.TilesRemaining(), 100-7)
	// English has exactly one Q and one Z, and the rack now holds both.
	is.Equal(s.BagCount(17), 0) // Q
	is.Equal(s.BagCount(26), 0) // Z
	is.NoErr(s.ValidateTileConservation(ld))

	// A rack the bag cannot supply is refused: there is no second Z.
	is.True(s.AssignRack(1, word(t, alph, "ZZ")) != nil)

	// Reassigning returns the old rack before taking the new one, so the total
	// stays put.
	is.NoErr(s.AssignRack(0, word(t, alph, "AEIOU")))
	is.Equal(s.TilesRemaining(), 100-5)
	is.NoErr(s.ValidateTileConservation(ld))
}

func TestAssignRackRollsBackOnFailure(t *testing.T) {
	is := is.New(t)
	s, ld := rackFixture(t)
	alph := ld.TileMapping()

	is.NoErr(s.AssignRack(0, word(t, alph, "AAAAAAA")))
	before := s.BagCounts()

	// English has nine As; the rack already holds seven, so eleven more is
	// impossible -- and the attempt must not disturb anything.
	err := s.AssignRack(1, word(t, alph, "AAAAAAA"))
	is.True(err != nil)
	is.Equal(s.BagCounts(), before)
	is.Equal(s.RackLen(1), 0)
	is.NoErr(s.ValidateTileConservation(ld))
}

func TestTileConservation(t *testing.T) {
	is := is.New(t)
	s, ld := rackFixture(t)
	alph := ld.TileMapping()
	is.NoErr(s.ValidateTileConservation(ld))

	// Deal both racks, then play tiles onto the board.
	is.NoErr(s.AssignRack(0, word(t, alph, "HELLO")))
	is.NoErr(s.AssignRack(1, word(t, alph, "WORLD")))
	is.NoErr(s.ValidateTileConservation(ld))

	m := NewPlacementMove(7, 3, false, word(t, alph, "HELLO"))
	s.PlaceMoveTiles(m)
	is.NoErr(s.TakeFromRack(0, m.Tiles))
	is.NoErr(s.ValidateTileConservation(ld))

	// A designated blank on the board still counts as a blank tile.
	is.NoErr(s.AssignRack(0, []tilemapping.MachineLetter{0}))
	is.NoErr(s.ValidateTileConservation(ld))
	blank := word(t, alph, "S")
	blank[0] |= tilemapping.BlankMask
	s.PlaceMoveTiles(NewPlacementMove(7, 8, false, blank))
	is.NoErr(s.TakeFromRack(0, blank))
	is.NoErr(s.ValidateTileConservation(ld))
	is.Equal(s.BagCount(0), 1) // one blank left in the bag

	// Now break it on purpose and make sure we notice.
	s.SetTileAt(0, 0, 26)
	err := s.ValidateTileConservation(ld)
	is.True(err != nil)
	t.Log(err)
}

// A long random sequence of rack and bag operations must never lose or invent a
// tile. This is the invariant that would have caught the May 2026 corruption at
// the moment it happened.
func TestTileConservationSurvivesRandomOperations(t *testing.T) {
	is := is.New(t)
	s, ld := rackFixture(t)
	rng := rand.New(rand.NewPCG(42, 43))

	for i := range 20000 {
		p := rng.IntN(MaxPlayers)
		switch rng.IntN(4) {
		case 0:
			_, err := s.DrawToFull(rng, p)
			is.NoErr(err)
		case 1:
			// Play some tiles from the rack onto empty squares.
			n := rng.IntN(s.RackLen(p) + 1)
			if n == 0 {
				continue
			}
			tiles := append([]tilemapping.MachineLetter(nil), s.Rack(p)[:n]...)
			placed := 0
			for _, ml := range tiles {
				row, col := rng.IntN(s.Dim()), rng.IntN(s.Dim())
				if s.HasLetter(row, col) {
					continue
				}
				if ml == 0 {
					// Designate the blank, as a real play would.
					s.SetTileAt(row, col, tilemapping.MachineLetter(1+rng.IntN(26))|tilemapping.BlankMask)
				} else {
					s.SetTileAt(row, col, ml)
				}
				is.NoErr(s.TakeFromRack(p, []tilemapping.MachineLetter{ml}))
				placed++
			}
		case 2:
			n := rng.IntN(s.RackLen(p) + 1)
			if n == 0 || s.TilesRemaining() < n {
				continue
			}
			tiles := append([]tilemapping.MachineLetter(nil), s.Rack(p)[:n]...)
			is.NoErr(s.TakeFromRack(p, tiles))
			var drawn []tilemapping.MachineLetter
			drawn, err := s.Exchange(rng, tiles, drawn)
			is.NoErr(err)
			is.NoErr(s.AddToRack(p, drawn))
		case 3:
			// Put the whole rack back.
			tiles := append([]tilemapping.MachineLetter(nil), s.Rack(p)...)
			if len(tiles) == 0 {
				continue
			}
			is.NoErr(s.TakeFromRack(p, tiles))
			is.NoErr(s.PutBack(tiles))
		}
		if err := s.ValidateTileConservation(ld); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
	}
}

func TestApplyOutBonus(t *testing.T) {
	is := is.New(t)
	s, ld := rackFixture(t)
	alph := ld.TileMapping()

	is.NoErr(s.AssignRack(1, word(t, alph, "QUIZ"))) // 10+1+1+10 = 22
	s.Scores = [MaxPlayers]int32{300, 280}

	pts, err := s.ApplyOutBonus(ld, 0)
	is.NoErr(err)
	is.Equal(pts, int32(44))
	is.Equal(s.Scores, [MaxPlayers]int32{344, 280})
	is.Equal(s.PlayState, GameOver)
}

func TestApplyScorelessPenalties(t *testing.T) {
	is := is.New(t)
	s, ld := rackFixture(t)
	alph := ld.TileMapping()

	is.NoErr(s.AssignRack(0, word(t, alph, "QI")))  // 11
	is.NoErr(s.AssignRack(1, word(t, alph, "JOT"))) // 8+1+1 = 10
	s.Scores = [MaxPlayers]int32{300, 280}
	s.OnTurn = 0

	pen := s.ApplyScorelessPenalties(ld)
	is.Equal(pen, [MaxPlayers]int32{11, 10})
	is.Equal(s.Scores, [MaxPlayers]int32{289, 270})
	is.Equal(s.PlayState, GameOver)
	// macondo flips exactly once here; we match it so state comparisons stay
	// quiet. See the note on ApplyScorelessPenalties.
	is.Equal(s.OnTurn, uint8(1))
}
