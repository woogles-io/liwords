package pair

import (
	"errors"
	"fmt"
	"sort"

	"github.com/woogles-io/liwords/pkg/entity"
)

// The Australian Draw is a transparent, hand-playable, deterministic
// pairing method. A director can reproduce it by hand: at each step the top
// remaining player is paired with the highest-standing opponent that still
// leaves the rest of the field pairable, backtracking when a choice paints
// the draw into a corner.
//
// Round 1 uses a casement draw (top half vs bottom half). Every later round
// pairs the top remaining player with the highest-standing opponent they are
// allowed to face, recursing on the rest and backtracking if no completion
// exists. A rematch is avoided only when the two players met in a round at or
// after the reset point (reset_round, a 0-based round number, matching the
// 0-based rounds used everywhere else), so a configured day boundary forgives
// earlier meetings; meetings before the reset may recur freely. reset_round of
// 0 avoids every prior meeting (the default); a
// reset_round at or past the current round forgives them all. If no draw
// exists under the current reset, the reset point is raised one round at a
// time -- forgiving the next-oldest round of repeats -- and the search is
// retried until a pairing is found or no repeat remains to forgive.
//
// An odd field is paired by padding it with a phantom "bye" participant at the
// bottom of the standings, pairing the resulting even field, and handing the
// real bye to whoever the phantom is paired with. The phantom is pairable with
// every player and has no meeting history, so every odd case reduces to the
// even algorithm with no special-casing.

// australianStanding is the projection of a PoolMember used for the
// deterministic standings sort. The index field records the position of
// the member in the original PoolMembers slice so results can be mapped
// back to the indices the caller expects.
type australianStanding struct {
	index  int
	id     string
	wins   int
	draws  int
	spread int
}

// sortAustralianStandings orders members best-to-worst. The sort is a
// transparent restatement of the division standings order using only the
// fields available on a PoolMember:
//
//  1. wins, counted as 2*wins + draws (a win outranks a draw, matching the
//     win metric used everywhere else in this package and in getRecords),
//  2. spread, descending,
//  3. player id, ascending (a deterministic final tiebreaker).
//
// "Total points since round 1" is part of the canonical standings order
// the caller already applied, but it is not carried on PoolMember, so it
// cannot participate in this local re-sort; id keeps the order total and
// deterministic in its absence.
func sortAustralianStandings(members []*entity.PoolMember) []australianStanding {
	standings := make([]australianStanding, len(members))
	for i, m := range members {
		standings[i] = australianStanding{
			index:  i,
			id:     m.Id,
			wins:   m.Wins,
			draws:  m.Draws,
			spread: m.Spread,
		}
	}
	sort.SliceStable(standings, func(i, j int) bool {
		wi := standings[i].wins*2 + standings[i].draws
		wj := standings[j].wins*2 + standings[j].draws
		if wi != wj {
			return wi > wj
		}
		if standings[i].spread != standings[j].spread {
			return standings[i].spread > standings[j].spread
		}
		return standings[i].id < standings[j].id
	})
	return standings
}

// australianCasement returns the round-1 casement pairings for n players in
// standings order. Pairings are expressed in standings positions: position
// p is paired against the value at index p. The top half plays the bottom
// half (0 vs n/2, 1 vs n/2+1, ...). With an odd count the lone middle
// player (position n/2) gets a bye, represented by -1.
func australianCasement(n int) []int {
	pairings := make([]int, n)
	for i := range pairings {
		pairings[i] = -1
	}
	half := n / 2
	for i := 0; i < half; i++ {
		top := i
		bottom := i + half + (n % 2)
		pairings[top] = bottom
		pairings[bottom] = top
	}
	// When n is odd the middle player (position half) keeps its -1 bye.
	return pairings
}

// australianCasementBlocked returns the round-1 casement pairings for an even
// field of n positions while honoring director blocks, guaranteeing zero
// top-vs-top pairings whenever a draw with none exists. The caller pads an odd
// field with a phantom bye before calling, so n is always even here.
// blocked[a][b] is true when positions a and b may not be paired (a director
// block, or, for the phantom column, a position barred from the bye). When no
// casement pair is blocked it reproduces the plain casement byte for byte (top
// half vs bottom half), so the common round-1 draw is unchanged. When a
// casement pair is blocked it repairs the draw by sliding the affected
// positions to the nearest legal partner within the casement structure,
// keeping every other position on its casement opponent. If no complete legal
// pairing exists it returns a clear error.
//
// "Top-vs-top" is defined by the top half: positions [0, n/2) are the top
// players, and a pair is top-vs-top iff both endpoints lie there. On a padded
// odd field the phantom bye is the last (lowest) position, so it is always a
// bottom player -- pairing a top player with it (the real bye) is never
// top-vs-top.
//
// The repair is a two-phase backtracker over the remaining positions in
// standings order (see casement below). It is dispatched in two passes:
//
//  1. casement(all, false) -- flips (top-vs-top pairings) disallowed. The only
//     pairings tried join two players that are not both top-half, so a complete
//     draw it finds has no top-vs-top pair. p (the top of the remaining set)
//     may pair any non-top-vs-top partner; backtracking explores every such
//     choice. This pass therefore succeeds iff a draw with zero top-vs-top
//     exists, and when it does the result is guaranteed to have none.
//  2. casement(all, true) -- flips permitted, tried only when pass 1 fails (no
//     zero-top-vs-top draw exists). It admits the forced top-vs-top pairings,
//     nearest first, so the draw is perturbed as little as the casement
//     structure allows while never wrongly rejecting a legal draw.
//
// An unobstructed field is resolved entirely by pass 1 and reproduces
// australianCasement (with any phantom bye landing on the lone middle): the
// proper casement opponent is always nearest-first and unblocked, so the
// boundary fallback is never taken until forced.
func australianCasementBlocked(n int, blocked [][]bool) ([]int, error) {
	pairings := make([]int, n)

	// topCount is the size of the top half. A pair (a, b) is top-vs-top (a
	// "flip") iff both endpoints are top players.
	topCount := n / 2
	topVsTop := func(a, b int) bool { return a < topCount && b < topCount }

	// casement attempts to pair the remaining positions r (a subset of [0, n)
	// in ascending standings order) into pairings, recursing on the rest. When
	// allowFlip is false only non-top-vs-top pairings are tried, so a complete
	// draw it finds has no top-vs-top pair; when true it also tries the forced
	// top-vs-top pairings as a fallback. It mutates pairings in place and undoes
	// its writes on a failed branch, so on return false pairings is unchanged.
	// The field is even and each step removes a pair, so every recursive r is
	// even too.
	var casement func(r []int, allowFlip bool) bool
	casement = func(r []int, allowFlip bool) bool {
		if len(r) == 0 {
			return true
		}

		// p is the top remaining position; half = len(r)/2 marks p's ideal
		// casement opponent r[half] (the top of the bottom half). r[half:] are
		// the proper casement bottoms; r[1:half] are the nearer positions.
		p := r[0]
		half := len(r) / 2

		// Phase 1a: proper casement bottoms r[half:], nearest to p's ideal
		// opponent first (r is ascending, so ascending over r[half:] is
		// nearest-first). None of these are top-vs-top.
		for _, b := range r[half:] {
			if blocked[p][b] {
				continue
			}
			pairings[p] = b
			pairings[b] = p
			if casement(without(r, p, b), allowFlip) {
				return true
			}
			pairings[p] = -1
			pairings[b] = -1
		}

		// Phase 1b: the non-top-vs-top positions among r[1:half] -- bottoms that
		// fell into the near half after earlier removals. A fallback after the
		// proper bottoms, but still part of the no-flip pass because pairing them
		// is not top-vs-top.
		for _, b := range r[1:half] {
			if topVsTop(p, b) || blocked[p][b] {
				continue
			}
			pairings[p] = b
			pairings[b] = p
			if casement(without(r, p, b), allowFlip) {
				return true
			}
			pairings[p] = -1
			pairings[b] = -1
		}

		// Phase 2: the top-vs-top positions among r[1:half], nearest to p first
		// -- a forced flip. Reached only when flips are permitted, i.e. only
		// after the no-flip pass has shown no zero-top-vs-top draw exists.
		if allowFlip {
			for _, b := range r[1:half] {
				if !topVsTop(p, b) || blocked[p][b] {
					continue
				}
				pairings[p] = b
				pairings[b] = p
				if casement(without(r, p, b), allowFlip) {
					return true
				}
				pairings[p] = -1
				pairings[b] = -1
			}
		}

		return false
	}

	all := make([]int, n)
	for i := range all {
		all[i] = i
	}

	// Pass 1 first: a complete draw with flips disallowed is guaranteed to have
	// zero top-vs-top, and it exists whenever any zero-top-vs-top draw does.
	for i := range pairings {
		pairings[i] = -1
	}
	if casement(all, false) {
		return pairings, nil
	}

	// Pass 2: no zero-top-vs-top draw exists, so permit the forced top-vs-top
	// pairings.
	for i := range pairings {
		pairings[i] = -1
	}
	if casement(all, true) {
		return pairings, nil
	}

	return nil, fmt.Errorf(
		"australian draw could not pair %d players in the first round under the"+
			" configured blocks", n)
}

// without returns the positions in r (ascending) with a and b removed,
// preserving order. r[0] is always a; b is some later element. The result is a
// fresh slice so recursion never aliases the caller's view of r.
func without(r []int, a, b int) []int {
	rest := make([]int, 0, len(r)-2)
	for _, x := range r {
		if x == a || x == b {
			continue
		}
		rest = append(rest, x)
	}
	return rest
}

// australianMatch pairs the even field of n positions in standings order by
// recursive backtracking, the later-round Australian rule. It pairs the top
// remaining player with the highest-standing opponent that leads to a complete
// draw: it tries that player's legal opponents nearest-first, recurses on the
// rest, and undoes the choice if the recursion cannot complete. blocked is the
// symmetric matrix of disallowed pairings (a director block, or a repeat at the
// active reset round; the phantom bye column is all-false). It returns the
// pairing array (in standings positions) and whether a complete draw exists.
// Unlike the round-1 casement it has no top-vs-top concept -- later rounds avoid
// only repeats, not strong-vs-strong pairings. The caller pads an odd field with
// a phantom bye, so n is even and each step removes a pair, keeping every
// recursive r even too.
func australianMatch(n int, blocked [][]bool) ([]int, bool) {
	pairings := make([]int, n)
	for i := range pairings {
		pairings[i] = -1
	}

	var match func(r []int) bool
	match = func(r []int) bool {
		if len(r) == 0 {
			return true
		}
		p := r[0]
		for _, q := range r[1:] {
			if blocked[p][q] {
				continue
			}
			pairings[p] = q
			pairings[q] = p
			if match(without(r, p, q)) {
				return true
			}
			pairings[p] = -1
			pairings[q] = -1
		}
		return false
	}

	all := make([]int, n)
	for i := range all {
		all[i] = i
	}
	if !match(all) {
		return nil, false
	}
	return pairings, true
}

// australianMatchWithReset runs the later-round backtracking matcher with the
// reset-relaxation retry loop. earliestReset is the earliest round at which the
// repeat rule applies; hasPlayedSince reports whether two positions are still
// considered a repeat at the active reset round, and blockedPair reports a hard
// director block. currentRound is the round being paired. The reset starts at
// earliestReset and is relaxed (incremented) one round at a time; raising it
// must be monotonically more permissive. The fully relaxed pass uses
// reset == currentRound: a prior meeting can only be in a round < currentRound
// (the round being paired has not happened yet), so reset == currentRound
// already forgives every repeat. If even that pass cannot pair the field the
// failure is final. n is the padded, even field size.
func australianMatchWithReset(
	n int,
	earliestReset int,
	currentRound int,
	hasPlayedSince func(a, b, reset int) bool,
	blockedPair func(a, b int) bool,
) ([]int, error) {
	// The director-block half of the block matrix is constant across reset
	// passes (blockedPair does not depend on reset), so compute it once. Only
	// the repeat half (hasPlayedSince) changes as the reset relaxes, so each
	// pass overwrites the reused blocked matrix in place rather than allocating
	// a fresh one. australianMatch reads blocked without mutating it, so reuse
	// is safe.
	dirBlocked := make([][]bool, n)
	blocked := make([][]bool, n)
	for i := range blocked {
		dirBlocked[i] = make([]bool, n)
		blocked[i] = make([]bool, n)
	}
	for a := 0; a < n; a++ {
		for b := a + 1; b < n; b++ {
			d := blockedPair(a, b)
			dirBlocked[a][b] = d
			dirBlocked[b][a] = d
		}
	}

	for reset := earliestReset; ; reset++ {
		for a := 0; a < n; a++ {
			for b := a + 1; b < n; b++ {
				block := dirBlocked[a][b] || hasPlayedSince(a, b, reset)
				blocked[a][b] = block
				blocked[b][a] = block
			}
		}

		pairings, ok := australianMatch(n, blocked)
		if ok {
			return pairings, nil
		}

		// The next pass relaxes the repeat rule by one round. The pass at
		// reset == currentRound already forbids no repeats at all -- every prior
		// meeting is in a round < currentRound -- so if it still fails the
		// obstruction is a hard block (director block or parity), not a repeat,
		// and the failure is final. (Relaxing further, to currentRound+1, would
		// rebuild an identical block matrix and fail identically.)
		if reset >= currentRound {
			return nil, fmt.Errorf(
				"australian draw could not pair %d players even after relaxing"+
					" the repeat rule to round %d", n, currentRound)
		}
	}
}

// pairAustralianDraw is the dispatcher entry point. It mirrors the other
// pairing-method functions: members arrive in standings order and the
// result is an index-into-PoolMembers array (opponent index per player, -1
// for a bye). An odd field is padded with a phantom bye so the casement and
// later-round matchers only ever see an even field.
func pairAustralianDraw(members *entity.UnpairedPoolMembers) ([]int, error) {
	if members.RoundControls == nil {
		return nil, errors.New("australian draw requires round controls")
	}

	n := len(members.PoolMembers)
	result := make([]int, n)
	for i := range result {
		result[i] = -1
	}
	if n == 0 {
		return result, nil
	}

	standings := sortAustralianStandings(members.PoolMembers)

	// round is 0-indexed in the round controls; the first round is round 0.
	currentRound := int(members.RoundControls.Round)

	// Pad an odd field with a phantom bye at the bottom of the standings. m is
	// the padded, even field size; bye is the phantom position (-1 when the
	// field is already even). The phantom is pairable with every real player and
	// has no meeting history, so whoever it is paired with simply takes the bye.
	m := n
	bye := -1
	if n%2 == 1 {
		m = n + 1
		bye = n
	}

	// isReal reports whether a padded position is a real player (not the phantom
	// bye). The phantom never blocks and never counts as a repeat.
	isReal := func(pos int) bool { return pos != bye }

	blockedPair := func(a, b int) bool {
		if !isReal(a) || !isReal(b) {
			// The phantom bye is pairable with every real player (any player may
			// take the bye).
			return false
		}
		return !pairable(members, standings[a].index, standings[b].index)
	}

	var positionPairings []int
	if currentRound == 0 {
		// The first round uses the casement draw (top half vs bottom half) on
		// the padded even field, but it must still honor director blocks: a
		// casement pair where one player blocks the other is repaired to the
		// nearest legal partners. Repeats never apply in round 1, so the blocked
		// matrix carries director blocks alone.
		blocked := make([][]bool, m)
		for i := range blocked {
			blocked[i] = make([]bool, m)
		}
		for a := 0; a < m; a++ {
			for b := a + 1; b < m; b++ {
				block := blockedPair(a, b)
				blocked[a][b] = block
				blocked[b][a] = block
			}
		}

		var err error
		positionPairings, err = australianCasementBlocked(m, blocked)
		if err != nil {
			return nil, err
		}
	} else {
		// reset_round is the 0-indexed round threshold the repeat rule uses
		// (members.RoundControls.Round and the RepeatRounds history are also
		// 0-indexed). A meeting in round r is avoided iff r >= reset_round, so
		// reset_round == 0 avoids every prior round (the default) and a large
		// reset_round forgives them all (King-of-the-Hill). The proto3 zero-
		// default (0) is therefore the correct default and needs no clamping.
		// The retry loop relaxes this threshold upward one round at a time when
		// a strict pairing is impossible.
		earliestReset := int(members.RoundControls.ResetRound)
		if earliestReset > currentRound {
			// A reset point past the current round would forgive everything;
			// clamp the threshold so the strictest pass still avoids the most
			// recent rounds.
			earliestReset = currentRound
		}

		// hasPlayedSince consults the per-round history: a pair is blocked
		// when the number of rounds >= reset in which they met exceeds
		// MaxRepeats. Meetings in rounds before reset are forgiven, so as the
		// retry loop raises reset the rule relaxes one round at a time. The
		// final relaxed pass (reset == currentRound) counts no prior round, so
		// the field can always be paired when no hard block stands in the way.
		// RepeatRounds may be nil (e.g. the dispatcher path on a pre-round-1
		// sub-pool); a nil map yields no meetings and thus no repeat blocks,
		// which is the correct "no history" behavior.
		hasPlayedSince := func(a, b, reset int) bool {
			if !isReal(a) || !isReal(b) {
				// The phantom bye has no meeting history.
				return false
			}
			rounds := members.RepeatRounds[GetRepeatKey(standings[a].id, standings[b].id)]
			met := 0
			for _, r := range rounds {
				if r >= reset {
					met++
				}
			}
			allowed := int(members.RoundControls.MaxRepeats)
			return met >= allowed+1
		}

		var err error
		positionPairings, err = australianMatchWithReset(
			m, earliestReset, currentRound, hasPlayedSince, blockedPair)
		if err != nil {
			return nil, err
		}
	}

	// Map padded-position pairings back to PoolMembers indices. The phantom
	// bye's own slot is dropped; its real partner takes the bye.
	for pos := 0; pos < n; pos++ {
		idx := standings[pos].index
		opp := positionPairings[pos]
		if opp < 0 || opp == bye {
			result[idx] = -1
		} else {
			result[idx] = standings[opp].index
		}
	}
	return result, nil
}
