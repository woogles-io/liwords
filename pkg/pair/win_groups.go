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
// positions, and whether the whole field could be paired.
func winGroupsMatch(n int, score []int, blocked [][]bool) ([]int, bool) {
	pairings := make([]int, n)
	for i := range pairings {
		pairings[i] = -1
	}
	// failed[i] records that positions [i, n) cannot be paired among
	// themselves, so the search never asks that twice. It changes no result.
	failed := make([]bool, n)

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
		for {
			if winGroupPair(i, j, blocked, pairings) {
				if pairFrom(j) {
					return true
				}
				// The players below the group pair only among themselves, so no
				// other pairing of this group can help them: take in two more.
				for p := i; p < j; p++ {
					pairings[p] = -1
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
		return nil, false
	}
	return pairings, true
}

// winGroupPair pairs the group of positions [i, j) top half against bottom
// half, writing the pairings, in positions, into pairings. It reports whether
// the group could be paired.
func winGroupPair(i, j int, blocked [][]bool, pairings []int) bool {
	g := j - i
	sub := make([][]bool, g)
	for a := range sub {
		sub[a] = blocked[i+a][i : i+g]
	}
	groupPairings, err := australianCasementBlocked(g, sub)
	if err != nil {
		return false
	}
	for a, b := range groupPairings {
		pairings[i+a] = i + b
	}
	return true
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
		func(n int, blocked [][]bool) ([]int, bool) {
			return winGroupsMatch(n, score, blocked)
		})
	if err != nil {
		return nil, err
	}
	return f.result(positionPairings), nil
}
