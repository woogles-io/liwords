package pair

import (
	"math/rand/v2"
	"testing"

	"github.com/matryer/is"
)

// randomBlocks blocks each pair of n positions with probability density.
func randomBlocks(rng *rand.Rand, n int, density float64) [][]bool {
	blocked := make([][]bool, n)
	for i := range blocked {
		blocked[i] = make([]bool, n)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if rng.Float64() < density {
				blocked[i][j] = true
				blocked[j][i] = true
			}
		}
	}
	return blocked
}

// blockedClubs blocks every pair inside each club, as a director blocking
// clubmates would. clubOf gives each position's club.
func blockedClubs(clubOf []int) [][]bool {
	n := len(clubOf)
	blocked := make([][]bool, n)
	for i := range blocked {
		blocked[i] = make([]bool, n)
		for j := range blocked[i] {
			blocked[i][j] = i != j && clubOf[i] == clubOf[j]
		}
	}
	return blocked
}

// canPairAllByHand answers canPairAll's question by trying every pairing.
func canPairAllByHand(r []int, allowed func(a, b int) bool) bool {
	if len(r) == 0 {
		return true
	}
	for i := 1; i < len(r); i++ {
		if !allowed(r[0], r[i]) {
			continue
		}
		rest := append(append([]int{}, r[1:i]...), r[i+1:]...)
		if canPairAllByHand(rest, allowed) {
			return true
		}
	}
	return false
}

func TestCanPairAllMatchesTryingEveryPairing(t *testing.T) {
	is := is.New(t)
	rng := rand.New(rand.NewPCG(1, 2))
	for trial := 0; trial < 3000; trial++ {
		n := 1 + rng.IntN(14)
		blocked := randomBlocks(rng, n, rng.Float64())
		allowed := func(a, b int) bool { return !blocked[a][b] }
		// A random subset of the positions, odd sizes included.
		r := []int{}
		for p := 0; p < n; p++ {
			if rng.IntN(4) > 0 {
				r = append(r, p)
			}
		}
		is.Equal(canPairAll(r, allowed), canPairAllByHand(r, allowed))
	}
}

// TestGuidedSearchFindsTheSameDraw checks that guiding the search changes no
// draw: guided from the start, after a few plain steps, or never.
func TestGuidedSearchFindsTheSameDraw(t *testing.T) {
	is := is.New(t)
	rng := rand.New(rand.NewPCG(3, 4))
	for trial := 0; trial < 3000; trial++ {
		n := 2 * (1 + rng.IntN(8))
		blocked := randomBlocks(rng, n, 0.7*rng.Float64())
		wantHalves, okHalves := pairHalvesSteps(n, blocked, -1)
		wantMatch, okMatch := australianMatchSteps(n, blocked, -1)
		for _, steps := range []int{0, 1, 3} {
			got, ok := pairHalvesSteps(n, blocked, steps)
			is.Equal(ok, okHalves)
			is.Equal(got, wantHalves)
			got, ok = australianMatchSteps(n, blocked, steps)
			is.Equal(ok, okMatch)
			is.Equal(got, wantMatch)
		}
	}
}

// TestBlockedClubsFailAtOnce: with two clubs blocked among themselves and
// one bigger than half the field, no draw exists, and the search used to
// try every order before saying so.
func TestBlockedClubsFailAtOnce(t *testing.T) {
	is := is.New(t)
	clubOf := make([]int, 38)
	for p := 20; p < 38; p++ {
		clubOf[p] = 1
	}
	blocked := blockedClubs(clubOf)
	_, ok := pairHalves(38, blocked)
	is.True(!ok)
	_, ok = australianMatch(38, blocked)
	is.True(!ok)
	_, ok = winGroupsMatch(38, make([]int, 38), blocked)
	is.True(!ok)
}

// TestBlockedClubsWithOneWayOut: club 0 (20 players) can be paired only if
// the free player at the top takes one of them, and the search reaches that
// opponent only after every member of club 1 (19 players). Each of those
// leaves club 0 a player short of opponents, so a search that backs up
// explores each dead end in full before moving on.
func TestBlockedClubsWithOneWayOut(t *testing.T) {
	is := is.New(t)
	const n = 40

	// pairHalves tries the proper bottoms 20 to 39 first, so club 1 sits at
	// 20 to 38 and the way out at 39.
	clubOf := make([]int, n)
	clubOf[0] = 2
	for p := 20; p < 39; p++ {
		clubOf[p] = 1
	}
	got, ok := pairHalves(n, blockedClubs(clubOf))
	is.True(ok)
	want := make([]int, n)
	want[0], want[39] = 39, 0
	for p := 1; p < 20; p++ {
		want[p], want[p+19] = p+19, p
	}
	is.Equal(got, want)

	// australianMatch tries opponents from the top, so club 1 sits at 1 to
	// 19 and the way out at 20.
	clubOf = make([]int, n)
	clubOf[0] = 2
	for p := 1; p < 20; p++ {
		clubOf[p] = 1
	}
	got, ok = australianMatch(n, blockedClubs(clubOf))
	is.True(ok)
	want = make([]int, n)
	want[0], want[20] = 20, 0
	for p := 1; p < 20; p++ {
		want[p], want[p+20] = p+20, p
	}
	is.Equal(got, want)
}
