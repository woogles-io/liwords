package pair

import (
	"errors"
	"math"

	"github.com/woogles-io/liwords/pkg/entity"
)

// Win Groups is a deterministic pairing method a director can check by hand.
// The field is ranked as in the Australian Draw (wins, a draw counting half,
// then spread, then seed) and paired group by group from the top:
//
//  1. A group is the top unpaired player, everyone below them on the same
//     wins, and, when that makes an odd number, the next player down.
//  2. The group's top half plays its bottom half: its first player meets the
//     first player of the bottom half, the second meets the second, and so
//     on. A player who may not meet that opponent (a repeat, or a director
//     block) takes the next allowed one down, backtracking when a choice
//     leaves the rest of the group unpairable. Two top-half players meet only
//     when the group has no pairing without that.
//  3. When the group cannot be paired, or the players below it then cannot be
//     paired among themselves, the next two players down join the group and
//     it is paired again.
//
// Repeats follow the reset round as in the Australian Draw: a meeting at or
// after reset_round is avoided, and when no pairing of the whole field avoids
// them all, the oldest round is forgiven first. In round 1 everyone is on the
// same wins, so the whole field is one group and its top half plays its
// bottom half.

// winGroupsMatch pairs the even field of n positions, in standings order,
// group by group from the top. score[p] is position p's wins (2*wins +
// draws); blocked is the symmetric matrix of disallowed pairings (a director
// block, or a repeat at the active reset round). It returns the pairings, in
// positions, and whether the whole field could be paired, or the error that
// stopped it.
func winGroupsMatch(n int, score []int, blocked [][]bool) ([]int, bool, error) {
	return winGroupsMatchMode(n, score, blocked, plainUntilStuck)
}

// winGroupsMatchMode is winGroupsMatch with each group's search run in the
// given mode. Only a plainOnly search skips the check below on [i, n).
func winGroupsMatchMode(n int, score []int, blocked [][]bool, mode searchMode) ([]int, bool, error) {
	pairings := make([]int, n)
	for i := range pairings {
		pairings[i] = -1
	}
	// failed[i] records that positions [i, n) cannot be paired among
	// themselves, so the search never asks that twice. It changes no result.
	failed := make([]bool, n)
	allowed := func(a, b int) bool { return !blocked[a][b] }
	var err error

	var pairFrom func(i int) bool
	pairFrom = func(i int) bool {
		if i == n {
			return true
		}
		if failed[i] {
			return false
		}
		// The group: position i, everyone below it on the same wins, and one
		// more player when that count is odd. Every group above was even and n
		// is even, so the extra player always exists.
		j := i + 1
		for j < n && score[j] == score[i] {
			j++
		}
		if (j-i)%2 == 1 {
			j++
		}
		checked := false
		for {
			ok, groupErr := winGroupPair(i, j, blocked, pairings, mode)
			if groupErr != nil {
				err = groupErr
				return false
			}
			if ok {
				if pairFrom(j) {
					return true
				}
				if err != nil {
					return false
				}
				// The players below the group pair only among themselves, so no
				// other pairing of this group can help them: take in two more.
				for p := i; p < j; p++ {
					pairings[p] = -1
				}
			}
			// The last try, the whole rest as one group, pairs it whenever it
			// can be paired at all, since that group's search may then make
			// any pair not blocked. So at the first failure ask whether it
			// can: when it cannot, no group from i will do.
			if !checked && mode != plainOnly {
				checked = true
				rest := make([]int, 0, n-i)
				for p := i; p < n; p++ {
					rest = append(rest, p)
				}
				pairable, checkErr := canPairAll(rest, allowed)
				if checkErr != nil {
					err = checkErr
					return false
				}
				if !pairable {
					failed[i] = true
					return false
				}
			}
			if j == n {
				failed[i] = true
				return false
			}
			j += 2
		}
	}

	if !pairFrom(0) {
		return nil, false, err
	}
	return pairings, true, nil
}

// winGroupPair pairs the group of positions [i, j) top half against bottom
// half, writing the pairings, in positions, into pairings. It reports whether
// the group could be paired, or the error that stopped its search.
func winGroupPair(i, j int, blocked [][]bool, pairings []int, mode searchMode) (bool, error) {
	g := j - i
	sub := make([][]bool, g)
	for a := range sub {
		sub[a] = blocked[i+a][i : i+g]
	}
	groupPairings, ok, err := pairHalvesMode(g, sub, mode)
	if err != nil || !ok {
		return false, err
	}
	for a, b := range groupPairings {
		pairings[i+a] = i + b
	}
	return true, nil
}

// pairWinGroups is the dispatcher entry point. Like pairAustralianDraw it
// takes the members in standings order and returns an index-into-PoolMembers
// array (opponent index per player, -1 for a bye). An odd field is padded
// with a phantom bye at the bottom of the standings.
func pairWinGroups(members *entity.UnpairedPoolMembers) ([]int, error) {
	if members.RoundControls == nil {
		return nil, errors.New("win groups requires round controls")
	}

	f := newStandingsField(members)
	if f.n == 0 {
		return []int{}, nil
	}

	// round is 0-indexed in the round controls; the first round is round 0.
	currentRound := int(members.RoundControls.Round)

	// A player's group is set by their wins, a draw counting half, as in the
	// standings sort. The phantom bye sits below everyone in a group of its
	// own, so an odd field's lowest group takes it in to make up its number.
	score := make([]int, f.m)
	for p := 0; p < f.n; p++ {
		score[p] = 2*f.standings[p].wins + f.standings[p].draws
	}
	if f.bye >= 0 {
		score[f.bye] = math.MinInt
	}

	positionPairings, err := matchWithReset("win groups", f.m,
		f.earliestReset(currentRound), currentRound, f.hasPlayedSince,
		f.blockedPair,
		func(n int, blocked [][]bool) ([]int, bool, error) {
			return winGroupsMatch(n, score, blocked)
		})
	if err != nil {
		return nil, err
	}
	return f.result(positionPairings), nil
}

// pairHalves pairs an even group of n positions, in standings order, top half
// against bottom half while honoring blocked pairs, and guarantees no two
// top-half players meet whenever a pairing without that exists. The caller
// pads an odd field with a phantom bye before calling, so n is always even
// here. blocked[a][b] is true when positions a and b may not be paired (a
// director block, a repeat, or, for the phantom column, a position barred
// from the bye). When nothing is blocked it pairs 0 vs n/2, 1 vs n/2+1, and so
// on. When a pair is blocked it repairs the pairing by sliding the affected
// positions to the nearest legal partner across the halves, keeping every
// other position on its usual opponent. It reports false when no complete
// legal pairing exists.
//
// "Top-vs-top" is defined by the top half: positions [0, n/2) are the top
// players, and a pair is top-vs-top iff both endpoints lie there. On a padded
// odd field the phantom bye is the last (lowest) position, so it is always a
// bottom player -- pairing a top player with it (the real bye) is never
// top-vs-top.
//
// The repair is a two-phase backtracker over the remaining positions in
// standings order (see halves below). It is dispatched in two passes:
//
//  1. halves(all, false) -- flips (top-vs-top pairings) disallowed. The only
//     pairings tried join two players that are not both top-half, so a complete
//     draw it finds has no top-vs-top pair. p (the top of the remaining set)
//     may pair any non-top-vs-top partner; backtracking explores every such
//     choice. This pass therefore succeeds iff a draw with zero top-vs-top
//     exists, and when it does the result is guaranteed to have none.
//  2. halves(all, true) -- flips permitted, tried only when pass 1 fails (no
//     zero-top-vs-top draw exists). It admits the forced top-vs-top pairings,
//     nearest first, so the draw is perturbed as little as the halves allow
//     while never wrongly rejecting a legal draw.
//
// An unobstructed field is resolved entirely by pass 1 and pairs 0 vs n/2,
// 1 vs n/2+1, and so on (with any phantom bye landing on the lone middle): the
// proper opponent across the halves is always nearest-first and unblocked, so
// the boundary fallback is never taken until forced.
//
// A pass whose search meets a dead end goes on guided (see searchRun), which
// finds the same draw. It returns an error only when the matcher behind the
// guided search fails.
func pairHalves(n int, blocked [][]bool) ([]int, bool, error) {
	return pairHalvesMode(n, blocked, plainUntilStuck)
}

// pairHalvesMode is pairHalves with each pass searched in the given mode.
func pairHalvesMode(n int, blocked [][]bool, mode searchMode) ([]int, bool, error) {
	pairings := make([]int, n)

	// topCount is the size of the top half. A pair (a, b) is top-vs-top (a
	// "flip") iff both endpoints are top players.
	topCount := n / 2
	topVsTop := func(a, b int) bool { return a < topCount && b < topCount }

	// halves attempts to pair the remaining positions r (a subset of [0, n)
	// in ascending standings order) into pairings, recursing on the rest. When
	// allowFlip is false only non-top-vs-top pairings are tried, so a complete
	// draw it finds has no top-vs-top pair; when true it also tries the forced
	// top-vs-top pairings as a fallback. It mutates pairings in place and undoes
	// its writes on a failed branch, so on return false pairings is unchanged.
	// The field is even and each step removes a pair, so every recursive r is
	// even too.
	// Per-depth scratch buffers for the remaining-positions list, so the
	// backtracking allocates nothing per node. Each level removes a pair, so
	// the recursion depth is bounded by n/2.
	scratch := make([][]int, n/2+1)
	for i := range scratch {
		scratch[i] = make([]int, n)
	}
	all := make([]int, n)
	for i := range all {
		all[i] = i
	}
	// level(k) is the players the choice at depth k is made among, while the
	// search is that deep.
	level := func(k int) []int {
		if k == 0 {
			return all
		}
		return scratch[k-1][:n-2*k]
	}

	// allowed holds the pairs the running pass may make, and search tracks
	// how it runs.
	var allowed func(a, b int) bool
	var search *searchRun

	var halves func(r []int, allowFlip bool, depth int) bool
	halves = func(r []int, allowFlip bool, depth int) bool {
		if len(r) == 0 {
			return true
		}

		// p is the top remaining position; half = len(r)/2 marks p's ideal
		// opponent r[half] (the top of the bottom half). r[half:] are the
		// proper bottoms; r[1:half] are the nearer positions.
		p := r[0]
		half := len(r) / 2

		// pairWith pairs p with b and recurses on the rest, undoing the pair
		// if the rest cannot be paired. A guided search skips a b that leaves
		// the rest unpairable. It reports whether the rest was paired, and
		// otherwise whether p may try its next opponent (see backUp).
		pairWith := func(b int) (bool, bool) {
			rest := withoutInto(scratch[depth], r, p, b)
			checked := search.guided
			if checked && !search.pairable(rest, allowed) {
				return false, !search.stop()
			}
			pairings[p] = b
			pairings[b] = p
			if halves(rest, allowFlip, depth+1) {
				return true, false
			}
			pairings[p] = -1
			pairings[b] = -1
			return false, search.backUp(checked, depth, level, allowed)
		}

		// Phase 1a: proper bottoms r[half:], nearest to p's ideal
		// opponent first (r is ascending, so ascending over r[half:] is
		// nearest-first). None of these are top-vs-top.
		for _, b := range r[half:] {
			if blocked[p][b] {
				continue
			}
			if paired, more := pairWith(b); paired || !more {
				return paired
			}
		}

		// Phase 1b: the non-top-vs-top positions among r[1:half] -- bottoms that
		// fell into the near half after earlier removals. A fallback after the
		// proper bottoms, but still part of the no-flip pass because pairing them
		// is not top-vs-top.
		for _, b := range r[1:half] {
			if topVsTop(p, b) || blocked[p][b] {
				continue
			}
			if paired, more := pairWith(b); paired || !more {
				return paired
			}
		}

		// Phase 2: the top-vs-top positions among r[1:half], nearest to p first
		// -- a forced flip. Reached only when flips are permitted, i.e. only
		// after the no-flip pass has shown no zero-top-vs-top draw exists.
		if allowFlip {
			for _, b := range r[1:half] {
				if !topVsTop(p, b) || blocked[p][b] {
					continue
				}
				if paired, more := pairWith(b); paired || !more {
					return paired
				}
			}
		}

		return false
	}

	// pass runs halves over the whole field. The pairs it may make are those
	// the search tries: none blocked, and none top-vs-top unless flips are
	// permitted.
	pass := func(allowFlip bool) (bool, error) {
		allowed = func(a, b int) bool {
			return !blocked[a][b] && (allowFlip || !topVsTop(a, b))
		}
		search = newSearchRun(mode)
		return search.run(pairings, func() bool {
			return (!search.guided || search.pairable(all, allowed)) &&
				halves(all, allowFlip, 0)
		})
	}

	// Pass 1 first: a complete draw with flips disallowed is guaranteed to have
	// zero top-vs-top, and it exists whenever any zero-top-vs-top draw does.
	ok, err := pass(false)
	if err != nil {
		return nil, false, err
	}
	if ok {
		return pairings, true, nil
	}

	// Pass 2: no zero-top-vs-top draw exists, so permit the forced top-vs-top
	// pairings.
	ok, err = pass(true)
	if err != nil {
		return nil, false, err
	}
	if ok {
		return pairings, true, nil
	}

	return nil, false, nil
}
