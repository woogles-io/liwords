package standings

import (
	"testing"

	"github.com/matryer/is"
)

// recordsInOrder returns standings records for the given players, best
// first.
func recordsInOrder(players ...int) []uint64 {
	records := make([]uint64, len(players))
	for rankIdx, playerIdx := range players {
		records[rankIdx] = getRecordFromWinsAndSpread(len(players)-rankIdx, initialSpreadValue) + uint64(playerIdx)
	}
	return records
}

// The round being paired is paired from the standings before the pending
// round, so the simulations must play the pairing those standings give,
// even when the pending round reorders the leaders.
func TestResolvePairedRoundRow(t *testing.T) {
	is := is.New(t)

	// Before the pending round: A(0), C(2), E(4), F(5), D(3), B(1), then
	// the evener. The round being paired is 1st v 4th, 2nd v 5th, 3rd v 6th,
	// which is A-F, C-D and E-B, and the 7th's bye.
	basisOrder := []int{0, 2, 4, 5, 3, 1, ByePlayerIndex}
	row := []int{0, 3, 1, 4, 2, 5, 6, 6}
	// After the pending round: C, A, F, E, B, D, then the evener.
	records := recordsInOrder(2, 0, 5, 4, 1, 3, ByePlayerIndex)
	resolved := make([]int, len(row))
	resolvePairedRoundRow(basisOrder, records, row, make([]int, 6), resolved)
	// A is now 2nd, F 3rd, C 1st, D 6th, E 4th, B 5th, the evener 7th.
	is.Equal(resolved, []int{1, 2, 0, 5, 3, 4, 6, 6})
}

// In the round being paired, playing 1st means playing whoever led before
// the pending round, so the swap is made in those ranks.
func TestForcedWinnerSwap(t *testing.T) {
	is := is.New(t)

	row := []int{0, 3, 1, 4, 2, 5}
	// D was 5th: swapping positions 1 and 3 has D play A, at position 0.
	a, b := forcedWinnerSwap(row, 4, true, true)
	is.Equal([2]int{a, b}, [2]int{1, 3})
	// F, 4th, already plays 1st, so without vsFirst F is moved to 2nd's
	// opponent instead.
	a, b = forcedWinnerSwap(row, 3, false, true)
	is.Equal([2]int{a, b}, [2]int{1, 2})
	// Nobody else is swapped, and nothing is in the final round.
	a, b = forcedWinnerSwap(row, 2, false, true)
	is.Equal([2]int{a, b}, [2]int{-1, -1})
	a, b = forcedWinnerSwap(row, 4, true, false)
	is.Equal([2]int{a, b}, [2]int{-1, -1})
}
