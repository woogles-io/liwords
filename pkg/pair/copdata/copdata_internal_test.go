package copdata

import "testing"

// TestHopefulToCashExtensionRankIdx_ScatteredGapDoesntBreakWindowParity
// reproduces the real-tournament bug this function fixes: a "gap" player
// (not individually hopeful for cash) sitting between two hopeful stretches,
// where a further player beyond the gap is independently hopeful and
// already extends the window past it. The scattered headcount of
// individually-hopeful players (28 contiguous + 1 further = 29) is odd, but
// the actual window (ranks 1-30, driven by the further player) is already
// even, so no extension should fire.
func TestHopefulToCashExtensionRankIdx_ScatteredGapDoesntBreakWindowParity(t *testing.T) {
	const numPlayers = 40
	const placePrizes = 10
	const lowestCashPlace = placePrizes - 1 // 0-indexed "10th"
	const nonHopeful = numPlayers - 1       // the "never reaches within range" sentinel GetPrecompData uses

	// Ranks 0-27 (28 players): all hopeful for a cash place.
	// Rank 28 (the "gap", like Daniel Blake): not hopeful.
	// Rank 29 (like Jason Ubeika): hopeful again, independently extending
	// the window past the gap.
	// Ranks 30+: filler, not hopeful, irrelevant to the window.
	highestRankHopefully := make([]int, numPlayers)
	for i := 0; i < 28; i++ {
		highestRankHopefully[i] = lowestCashPlace
	}
	highestRankHopefully[28] = nonHopeful
	highestRankHopefully[29] = lowestCashPlace
	for i := 30; i < numPlayers; i++ {
		highestRankHopefully[i] = nonHopeful
	}
	gibsonizedPlayers := make([]bool, numPlayers)

	// The natural (pre-extension) window already reaches rank 30 (index 29)
	// because of the independently-hopeful player beyond the gap.
	naturalBoundary := computeLowestPossibleHopeNth(highestRankHopefully)[lowestCashPlace]
	if naturalBoundary != 29 {
		t.Fatalf("natural boundary = %d, want 29 (should already reach the independently-hopeful player at rank 30)", naturalBoundary)
	}

	// No extension should fire: the window (30 players) is already even.
	if extendedRankIdx := hopefulToCashExtensionRankIdx(highestRankHopefully, gibsonizedPlayers, placePrizes); extendedRankIdx != -1 {
		t.Fatalf("got extension to rank index %d, want -1: an already-even window (driven by a player past the gap) must not be extended", extendedRankIdx)
	}

	// Sanity check the fixed baseline against a genuinely odd, unbroken
	// window (27 contiguous hopeful players, no independently-hopeful
	// player beyond it): extension should fire, targeting the very next
	// rank (27).
	oddWindow := make([]int, numPlayers)
	for i := 0; i < 27; i++ {
		oddWindow[i] = lowestCashPlace
	}
	for i := 27; i < numPlayers; i++ {
		oddWindow[i] = nonHopeful
	}
	if extendedRankIdx := hopefulToCashExtensionRankIdx(oddWindow, gibsonizedPlayers, placePrizes); extendedRankIdx != 27 {
		t.Fatalf("got extension to rank index %d, want 27: a genuinely odd, unbroken window should extend to the very next rank", extendedRankIdx)
	}
}

// TestHopefulToCashExtensionRankIdx_GibsonizedPlayersExcluded verifies
// Gibsonized players inside the window don't count toward the parity check.
func TestHopefulToCashExtensionRankIdx_GibsonizedPlayersExcluded(t *testing.T) {
	const numPlayers = 20
	const placePrizes = 10
	const lowestCashPlace = placePrizes - 1
	const nonHopeful = numPlayers - 1

	// 6 hopeful players, one of them (rank 2) Gibsonized - so only 5 count
	// toward parity: odd, so the window should extend by one rank (to 6).
	highestRankHopefully := make([]int, numPlayers)
	for i := 0; i < 6; i++ {
		highestRankHopefully[i] = lowestCashPlace
	}
	for i := 6; i < numPlayers; i++ {
		highestRankHopefully[i] = nonHopeful
	}
	gibsonizedPlayers := make([]bool, numPlayers)
	gibsonizedPlayers[2] = true

	extendedRankIdx := hopefulToCashExtensionRankIdx(highestRankHopefully, gibsonizedPlayers, placePrizes)
	if extendedRankIdx != 6 {
		t.Fatalf("got %d, want 6 (a Gibsonized player inside the window should make an otherwise-even window odd)", extendedRankIdx)
	}
}
