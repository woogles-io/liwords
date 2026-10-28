package pair

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// wgMembers builds a Win Groups pool in standings order. wins gives each
// player's wins and must not increase down the list; spreads are all 0, so the
// standings sort keeps the given order. repeatRounds maps a pair key to the
// 0-indexed rounds in which the two players met.
func wgMembers(round int32, resetRound uint32, wins []int, repeatRounds map[string][]int, ids ...string) *entity.UnpairedPoolMembers {
	poolMembers := make([]*entity.PoolMember, len(ids))
	for i, id := range ids {
		poolMembers[i] = &entity.PoolMember{Id: id, Wins: wins[i]}
	}
	if repeatRounds == nil {
		repeatRounds = map[string][]int{}
	}
	return &entity.UnpairedPoolMembers{
		PoolMembers: poolMembers,
		RoundControls: &pb.RoundControl{
			PairingMethod: pb.PairingMethod_WIN_GROUPS,
			Round:         round,
			ResetRound:    resetRound,
		},
		Repeats:      map[string]int{},
		RepeatRounds: repeatRounds,
	}
}

func TestWinGroupsRoundOnePairsTopHalfAgainstBottomHalf(t *testing.T) {
	is := is.New(t)

	// Everyone is on 0 wins, so the field is one group.
	m := wgMembers(0, 0, make([]int, 8), nil, "a", "b", "c", "d", "e", "f", "g", "h")
	pairings, err := pairWinGroups(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	is.Equal(idPairings(m, pairings), map[string]string{
		"a": "e", "b": "f", "c": "g", "d": "h",
		"e": "a", "f": "b", "g": "c", "h": "d",
	})
}

func TestWinGroupsRoundOneOddFieldGivesTheMiddleTheBye(t *testing.T) {
	is := is.New(t)

	// The phantom bye joins the bottom of the one group, so the last player
	// of the top half takes it.
	m := wgMembers(0, 0, make([]int, 7), nil, "a", "b", "c", "d", "e", "f", "g")
	pairings, err := pairWinGroups(m)
	is.NoErr(err)
	is.Equal(idPairings(m, pairings), map[string]string{
		"a": "e", "b": "f", "c": "g", "d": "",
		"e": "a", "f": "b", "g": "c",
	})
}

func TestWinGroupsPairsWithinEachWinGroup(t *testing.T) {
	is := is.New(t)

	// After round 1 (a-e, b-f, c-g, d-h), a to d are on 1 win and e to h on 0.
	repeatRounds := map[string][]int{
		GetRepeatKey("a", "e"): {0},
		GetRepeatKey("b", "f"): {0},
		GetRepeatKey("c", "g"): {0},
		GetRepeatKey("d", "h"): {0},
	}
	m := wgMembers(1, 0, []int{1, 1, 1, 1, 0, 0, 0, 0}, repeatRounds,
		"a", "b", "c", "d", "e", "f", "g", "h")
	pairings, err := pairWinGroups(m)
	is.NoErr(err)
	is.Equal(idPairings(m, pairings), map[string]string{
		"a": "c", "b": "d", "c": "a", "d": "b",
		"e": "g", "f": "h", "g": "e", "h": "f",
	})
}

func TestWinGroupsEvensAnOddGroup(t *testing.T) {
	is := is.New(t)

	// a, b, and c are on 2 wins; d, the top of the next group, joins them.
	m := wgMembers(2, 0, []int{2, 2, 2, 1, 1, 1}, nil, "a", "b", "c", "d", "e", "f")
	pairings, err := pairWinGroups(m)
	is.NoErr(err)
	is.Equal(idPairings(m, pairings), map[string]string{
		"a": "c", "b": "d", "c": "a", "d": "b", "e": "f", "f": "e",
	})
}

func TestWinGroupsSlidesPastARepeat(t *testing.T) {
	is := is.New(t)

	// a's opponent would be c, the top of the bottom half, but they have met,
	// so a takes d and b takes c.
	repeatRounds := map[string][]int{GetRepeatKey("a", "c"): {0}}
	m := wgMembers(1, 0, []int{1, 1, 1, 1}, repeatRounds, "a", "b", "c", "d")
	pairings, err := pairWinGroups(m)
	is.NoErr(err)
	is.Equal(idPairings(m, pairings), map[string]string{
		"a": "d", "b": "c", "c": "b", "d": "a",
	})
}

func TestWinGroupsTakesInTwoWhenAGroupCannotPair(t *testing.T) {
	is := is.New(t)

	// a and b are alone on 1 win and have met, so c and d join their group.
	repeatRounds := map[string][]int{
		GetRepeatKey("a", "b"): {0},
		GetRepeatKey("c", "d"): {0},
	}
	m := wgMembers(1, 0, []int{1, 1, 0, 0}, repeatRounds, "a", "b", "c", "d")
	pairings, err := pairWinGroups(m)
	is.NoErr(err)
	is.Equal(idPairings(m, pairings), map[string]string{
		"a": "c", "b": "d", "c": "a", "d": "b",
	})
}

func TestWinGroupsTakesInTwoWhenTheRestCannotPair(t *testing.T) {
	is := is.New(t)

	// a to d pair among themselves, but that would leave e and f, who have
	// met, so e and f join the top group, which is paired again.
	repeatRounds := map[string][]int{GetRepeatKey("e", "f"): {1}}
	m := wgMembers(2, 0, []int{1, 1, 1, 1, 0, 0}, repeatRounds, "a", "b", "c", "d", "e", "f")
	pairings, err := pairWinGroups(m)
	is.NoErr(err)
	is.Equal(idPairings(m, pairings), map[string]string{
		"a": "d", "b": "e", "c": "f", "d": "a", "e": "b", "f": "c",
	})
}

func TestWinGroupsForgivesTheOldestRoundWhenNothingElsePairs(t *testing.T) {
	is := is.New(t)

	// Only two players, who met in round 1: the reset is raised until the
	// meeting is forgiven.
	repeatRounds := map[string][]int{GetRepeatKey("a", "b"): {0}}
	m := wgMembers(1, 0, []int{1, 0}, repeatRounds, "a", "b")
	pairings, err := pairWinGroups(m)
	is.NoErr(err)
	is.Equal(idPairings(m, pairings), map[string]string{"a": "b", "b": "a"})
}

func TestWinGroupsHonorsTheResetRound(t *testing.T) {
	is := is.New(t)

	// a and c met in round 2; with the reset at round 3 that meeting may
	// recur, so the group pairs top half against bottom half as usual.
	repeatRounds := map[string][]int{GetRepeatKey("a", "c"): {1}}
	m := wgMembers(3, 2, []int{2, 2, 2, 2}, repeatRounds, "a", "b", "c", "d")
	pairings, err := pairWinGroups(m)
	is.NoErr(err)
	is.Equal(idPairings(m, pairings), map[string]string{
		"a": "c", "b": "d", "c": "a", "d": "b",
	})
}

func TestWinGroupsDispatchesThroughPair(t *testing.T) {
	is := is.New(t)

	m := wgMembers(0, 0, make([]int, 4), nil, "a", "b", "c", "d")
	pairings, err := Pair(m)
	is.NoErr(err)
	is.Equal(idPairings(m, pairings), map[string]string{
		"a": "c", "b": "d", "c": "a", "d": "b",
	})
}

// withoutPair returns r (ascending) without a and b, as a new slice.
func withoutPair(r []int, a, b int) []int {
	out := make([]int, 0, len(r))
	for _, x := range r {
		if x != a && x != b {
			out = append(out, x)
		}
	}
	return out
}

// casementEach calls done with every pairing of the even group of n positions
// that australianCasementBlocked's search visits, in the same order: first
// every pairing with no two top-half players together (its first pass), then
// every pairing at all (its second). It returns true as soon as done does.
func casementEach(n int, blocked [][]bool, done func([]int) bool) bool {
	pairings := make([]int, n)
	for i := range pairings {
		pairings[i] = -1
	}
	topCount := n / 2
	topVsTop := func(a, b int) bool { return a < topCount && b < topCount }

	var walk func(r []int, allowFlip bool) bool
	walk = func(r []int, allowFlip bool) bool {
		if len(r) == 0 {
			return done(slices.Clone(pairings))
		}
		p := r[0]
		half := len(r) / 2
		try := func(b int) bool {
			pairings[p], pairings[b] = b, p
			ok := walk(withoutPair(r, p, b), allowFlip)
			pairings[p], pairings[b] = -1, -1
			return ok
		}
		for _, b := range r[half:] {
			if !blocked[p][b] && try(b) {
				return true
			}
		}
		for _, b := range r[1:half] {
			if !topVsTop(p, b) && !blocked[p][b] && try(b) {
				return true
			}
		}
		if allowFlip {
			for _, b := range r[1:half] {
				if topVsTop(p, b) && !blocked[p][b] && try(b) {
					return true
				}
			}
		}
		return false
	}

	all := make([]int, n)
	for i := range all {
		all[i] = i
	}
	return walk(all, false) || walk(all, true)
}

// winGroupsAsWritten is the Win Groups rule with no shortcut: every pairing of
// a group, in search order, is tried with the rest of the field before the
// group takes in two more players.
func winGroupsAsWritten(n int, score []int, blocked [][]bool) ([]int, bool) {
	pairings := make([]int, n)
	for i := range pairings {
		pairings[i] = -1
	}
	var from func(i int) bool
	from = func(i int) bool {
		if i == n {
			return true
		}
		j := i + 1
		for j < n && score[j] == score[i] {
			j++
		}
		if (j-i)%2 == 1 {
			j++
		}
		for ; ; j += 2 {
			g := j - i
			sub := make([][]bool, g)
			for a := range sub {
				sub[a] = blocked[i+a][i:j]
			}
			if casementEach(g, sub, func(gp []int) bool {
				for a, b := range gp {
					pairings[i+a] = i + b
				}
				if from(j) {
					return true
				}
				for a := range gp {
					pairings[i+a] = -1
				}
				return false
			}) {
				return true
			}
			if j == n {
				return false
			}
		}
	}
	if !from(0) {
		return nil, false
	}
	return pairings, true
}

// randomField returns an even field of up to 10 positions with
// non-increasing scores and a random symmetric blocked matrix.
func randomField(rng *rand.Rand) (int, []int, [][]bool) {
	n := 2 * (1 + rng.IntN(5))
	score := make([]int, n)
	for i := range score {
		score[i] = rng.IntN(4)
	}
	slices.SortFunc(score, func(a, b int) int { return b - a })
	density := []float64{0.15, 0.35, 0.6}[rng.IntN(3)]
	blocked := make([][]bool, n)
	for i := range blocked {
		blocked[i] = make([]bool, n)
	}
	for a := 0; a < n; a++ {
		for b := a + 1; b < n; b++ {
			if rng.Float64() < density {
				blocked[a][b] = true
				blocked[b][a] = true
			}
		}
	}
	return n, score, blocked
}

func TestCasementEachStartsWhereTheCasementSearchStops(t *testing.T) {
	is := is.New(t)

	// casementEach's first pairing must be australianCasementBlocked's answer,
	// or winGroupsAsWritten below would not be the same rule.
	rng := rand.New(rand.NewPCG(7, 13))
	for trial := 0; trial < 3000; trial++ {
		n, _, blocked := randomField(rng)
		var first []int
		casementEach(n, blocked, func(p []int) bool {
			first = p
			return true
		})
		got, err := australianCasementBlocked(n, blocked)
		if first == nil {
			is.True(err != nil)
			continue
		}
		is.NoErr(err)
		is.Equal(got, first)
	}
}

func TestWinGroupsMatchesTheRuleAsWritten(t *testing.T) {
	is := is.New(t)

	rng := rand.New(rand.NewPCG(1, 2))
	for trial := 0; trial < 3000; trial++ {
		n, score, blocked := randomField(rng)
		got, gotOK := winGroupsMatch(n, score, blocked)
		want, wantOK := winGroupsAsWritten(n, score, blocked)
		is.Equal(gotOK, wantOK)
		is.Equal(got, want)
		if gotOK {
			assertSymmetric(is, got)
			for a, b := range got {
				is.True(!blocked[a][b]) // never a blocked pair
			}
		}
	}
}
