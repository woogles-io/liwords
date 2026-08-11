// Rack management: the operations that keep the players' racks consistent with
// the bag, and the end-of-game rack adjustments.
//
// Racks are held canonically sorted. Rack order carries no meaning in the rules
// -- macondo does not model it at all, holding a rack as a count vector -- but
// State stores racks as a slot array, which does imply an order. Sorting on
// every mutation keeps that implied order out of Equal and Digest, so two
// identical positions always compare equal and hash the same. Display order is
// a client concern and is not the server's to remember.

package xwordgame

import (
	"fmt"

	"github.com/domino14/word-golib/tilemapping"
)

// RackLen returns how many tiles player p is holding.
func (s *State) RackLen(p int) int {
	if p < 0 || p >= MaxPlayers {
		return 0
	}
	return int(s.rackLens[p])
}

// normalizeRackTile maps a tile to the form racks store it in. A blank only
// carries a designated letter while it is on the board; in the bag and on a
// rack it is the zero tile.
func normalizeRackTile(ml tilemapping.MachineLetter) (tilemapping.MachineLetter, error) {
	if ml.IsBlanked() {
		ml = 0
	}
	if int(ml) >= AlphabetSize {
		return 0, fmt.Errorf("xwordgame: tile %d is outside the alphabet", ml)
	}
	return ml, nil
}

// rackCounts returns player p's rack as a count vector.
func (s *State) rackCounts(p int) [AlphabetSize]uint8 {
	var counts [AlphabetSize]uint8
	for _, ml := range s.Rack(p) {
		counts[ml]++
	}
	return counts
}

// setRackFromCounts rewrites a rack from a count vector. Iterating the vector in
// order is what keeps racks sorted.
func (s *State) setRackFromCounts(p int, counts [AlphabetSize]uint8) {
	n := 0
	for ml, ct := range counts {
		for range ct {
			s.racks[p][n] = tilemapping.MachineLetter(ml)
			n++
		}
	}
	s.rackLens[p] = uint8(n)
	for i := n; i < RackTileLimit; i++ {
		s.racks[p][i] = 0
	}
}

// RackHas reports whether player p holds every one of the given tiles, counting
// duplicates. This is the check behind "you don't have those tiles".
func (s *State) RackHas(p int, tiles []tilemapping.MachineLetter) bool {
	if p < 0 || p >= MaxPlayers {
		return false
	}
	var want [AlphabetSize]uint8
	for _, ml := range tiles {
		n, err := normalizeRackTile(ml)
		if err != nil {
			return false
		}
		want[n]++
	}
	have := s.rackCounts(p)
	for ml, ct := range want {
		if have[ml] < ct {
			return false
		}
	}
	return true
}

// TakeFromRack removes the given tiles from player p's rack. Blanks may be
// passed with a designated letter -- as they appear in a play -- and are removed
// as undesignated blanks.
//
// The whole set is verified before anything is removed, so a failure leaves the
// rack untouched.
func (s *State) TakeFromRack(p int, tiles []tilemapping.MachineLetter) error {
	if p < 0 || p >= MaxPlayers {
		return fmt.Errorf("xwordgame: player index %d out of range", p)
	}
	var want [AlphabetSize]uint8
	for _, ml := range tiles {
		n, err := normalizeRackTile(ml)
		if err != nil {
			return err
		}
		want[n]++
	}
	have := s.rackCounts(p)
	for ml, ct := range want {
		if have[ml] < ct {
			return fmt.Errorf("xwordgame: player %d holds %d of tile %d, cannot play %d",
				p, have[ml], ml, ct)
		}
	}
	for ml, ct := range want {
		have[ml] -= ct
	}
	s.setRackFromCounts(p, have)
	return nil
}

// AddToRack puts tiles onto player p's rack, keeping it sorted.
func (s *State) AddToRack(p int, tiles []tilemapping.MachineLetter) error {
	if p < 0 || p >= MaxPlayers {
		return fmt.Errorf("xwordgame: player index %d out of range", p)
	}
	if total := int(s.rackLens[p]) + len(tiles); total > RackTileLimit {
		return fmt.Errorf("xwordgame: adding %d tiles to player %d's rack of %d exceeds the limit of %d",
			len(tiles), p, s.rackLens[p], RackTileLimit)
	}
	counts := s.rackCounts(p)
	for _, ml := range tiles {
		n, err := normalizeRackTile(ml)
		if err != nil {
			return err
		}
		counts[n]++
	}
	s.setRackFromCounts(p, counts)
	return nil
}

// DrawToFull tops player p's rack up to a full rack, or empties the bag trying.
// It returns the number of tiles drawn.
func (s *State) DrawToFull(rng Rand, p int) (int, error) {
	if p < 0 || p >= MaxPlayers {
		return 0, fmt.Errorf("xwordgame: player index %d out of range", p)
	}
	need := RackTileLimit - int(s.rackLens[p])
	if need <= 0 {
		return 0, nil
	}
	var buf [RackTileLimit]tilemapping.MachineLetter
	drawn, n := s.DrawAtMost(rng, need, buf[:0])
	if n == 0 {
		return 0, nil
	}
	// Cannot fail: n <= need, so the rack still fits.
	if err := s.AddToRack(p, drawn); err != nil {
		return 0, err
	}
	return n, nil
}

// AssignRack gives player p exactly these tiles, taking them out of the bag. Any
// tiles the player is already holding go back into the bag first, so the total
// tile count is conserved either way.
//
// This is the reconstruction entry point: replaying a GameHistory, importing a
// CGP, or setting up a test position, where the rack is already known and the
// bag simply has to agree.
func (s *State) AssignRack(p int, tiles []tilemapping.MachineLetter) error {
	if p < 0 || p >= MaxPlayers {
		return fmt.Errorf("xwordgame: player index %d out of range", p)
	}
	if len(tiles) > RackTileLimit {
		return fmt.Errorf("xwordgame: rack of %d tiles exceeds the limit of %d", len(tiles), RackTileLimit)
	}
	// The bag is a fixed array, so snapshotting it to roll back on failure costs
	// one copy and makes this all-or-nothing.
	saved := s.bag
	if err := s.PutBack(s.Rack(p)); err != nil {
		s.bag = saved
		return err
	}
	if err := s.RemoveFromBag(tiles); err != nil {
		s.bag = saved
		return err
	}
	if err := s.SetRack(p, tiles); err != nil {
		s.bag = saved
		return err
	}
	return nil
}

// RackScoreFor is the face value of player p's remaining tiles.
func (s *State) RackScoreFor(ld *tilemapping.LetterDistribution, p int) int32 {
	if p < 0 || p >= MaxPlayers {
		return 0
	}
	return int32(RackScore(ld, s.Rack(p)))
}

// ValidateTileConservation checks the invariant that matters most: every tile in
// the distribution is accounted for exactly once, as a tile in the bag, a tile
// on a rack, or a tile on the board.
//
// State.Validate cannot check this because it does not know the distribution.
// This is the assertion that catches a corrupt position at the moment it is
// created rather than hundreds of games later, so it is worth calling after
// every transition in tests and behind a flag in production.
func (s *State) ValidateTileConservation(ld *tilemapping.LetterDistribution) error {
	if ld == nil {
		return fmt.Errorf("xwordgame: a letter distribution is required to check tile conservation")
	}
	var got [AlphabetSize]int
	for ml, ct := range s.bag {
		got[ml] += int(ct)
	}
	for p := range MaxPlayers {
		for _, ml := range s.Rack(p) {
			if int(ml) >= AlphabetSize {
				return fmt.Errorf("xwordgame: player %d rack holds out-of-range tile %d", p, ml)
			}
			got[ml]++
		}
	}
	for i, ml := range s.Board() {
		if ml == 0 {
			continue
		}
		// A designated blank on the board is still a blank tile.
		if ml.IsBlanked() {
			got[0]++
			continue
		}
		if int(ml) >= AlphabetSize {
			return fmt.Errorf("xwordgame: board square %d holds out-of-range tile %d", i, ml)
		}
		got[ml]++
	}

	want := ld.Distribution()
	for ml := range got {
		w := 0
		if ml < len(want) {
			w = int(want[ml])
		}
		if got[ml] != w {
			return fmt.Errorf("xwordgame: tile %d is not conserved: distribution has %d, position accounts for %d",
				ml, w, got[ml])
		}
	}
	return nil
}

// ApplyOutBonus credits the player who went out with twice the value of the
// opponent's remaining rack and ends the game. It returns the points awarded.
func (s *State) ApplyOutBonus(ld *tilemapping.LetterDistribution, out int) (int32, error) {
	if out < 0 || out >= MaxPlayers {
		return 0, fmt.Errorf("xwordgame: player index %d out of range", out)
	}
	if ld == nil {
		return 0, fmt.Errorf("xwordgame: a letter distribution is required to score the end of a game")
	}
	pts := s.RackScoreFor(ld, 1-out) * 2
	s.Scores[out] += pts
	s.PlayState = GameOver
	return pts, nil
}

// ApplyScorelessPenalties ends a game stalled on consecutive scoreless turns:
// each player loses the value of their own remaining rack. It returns the
// penalty deducted from each.
//
// The player on turn flips, exactly once, as it does in macondo. Whose turn it
// is has no meaning once the game is over, but the flip is what macondo's event
// attribution implies, and matching it keeps a byte-level state comparison
// between the two engines quiet.
func (s *State) ApplyScorelessPenalties(ld *tilemapping.LetterDistribution) [MaxPlayers]int32 {
	var penalties [MaxPlayers]int32
	s.PlayState = GameOver

	p := int(s.OnTurn)
	penalties[p] = s.RackScoreFor(ld, p)
	s.Scores[p] -= penalties[p]

	s.OnTurn = otherPlayer(s.OnTurn)

	p = int(s.OnTurn)
	penalties[p] = s.RackScoreFor(ld, p)
	s.Scores[p] -= penalties[p]

	return penalties
}

// otherPlayer returns the opponent's index.
func otherPlayer(p uint8) uint8 { return 1 - p }
