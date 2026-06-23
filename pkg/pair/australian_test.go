package pair

import (
	"fmt"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// auMembers builds an UnpairedPoolMembers in standings order from a list of
// ids. The members are given strictly decreasing win counts so the
// standings sort preserves the given order, letting the tests reason about
// standings positions directly. The repeats map gives the total number of
// prior meetings per pair; auMembers spreads each pair's meetings across the
// earliest rounds (round 0, 1, ...) when building the per-round RepeatRounds
// history the Australian Draw consults, which reproduces "all prior meetings
// count" under the default unset reset round. For tests that need meetings
// in specific rounds, use auMembersRounds.
func auMembers(round int32, maxRepeats int32, repeats map[string]int, ids ...string) *entity.UnpairedPoolMembers {
	repeatRounds := map[string][]int{}
	for key, count := range repeats {
		rounds := make([]int, count)
		for r := 0; r < count; r++ {
			rounds[r] = r
		}
		repeatRounds[key] = rounds
	}
	return auMembersRounds(round, maxRepeats, 0, repeats, repeatRounds, ids...)
}

// auMembersRounds is auMembers with explicit control over the reset round and
// the per-round meeting history. resetRound is the proto reset_round field,
// stored 1-based (the reset point as a round number; a meeting in 1-based
// round r is avoided iff r >= resetRound, and 0/unset means 1 = avoid all).
// repeatRounds maps each pair key to the ascending list of 0-indexed rounds
// in which the two players met (the history is 0-indexed, like round).
func auMembersRounds(round int32, maxRepeats int32, resetRound uint32, repeats map[string]int, repeatRounds map[string][]int, ids ...string) *entity.UnpairedPoolMembers {
	poolMembers := make([]*entity.PoolMember, len(ids))
	for i, id := range ids {
		poolMembers[i] = &entity.PoolMember{
			Id:   id,
			Wins: len(ids) - i, // strictly decreasing => standings == input order
		}
	}
	if repeats == nil {
		repeats = map[string]int{}
	}
	if repeatRounds == nil {
		repeatRounds = map[string][]int{}
	}
	return &entity.UnpairedPoolMembers{
		PoolMembers: poolMembers,
		RoundControls: &pb.RoundControl{
			PairingMethod: pb.PairingMethod_AUSTRALIAN_DRAW,
			Round:         round,
			MaxRepeats:    maxRepeats,
			ResetRound:    resetRound,
		},
		Repeats:      repeats,
		RepeatRounds: repeatRounds,
	}
}

// idPairings converts an index-pairing result into id->opponentId form so
// expectations are readable and independent of slice positions. A bye maps
// to the empty string.
func idPairings(members *entity.UnpairedPoolMembers, pairings []int) map[string]string {
	out := map[string]string{}
	for i, opp := range pairings {
		id := members.PoolMembers[i].Id
		if opp < 0 {
			out[id] = ""
		} else {
			out[id] = members.PoolMembers[opp].Id
		}
	}
	return out
}

// assertSymmetric checks that the pairing array is a valid involution: if a
// is paired to b then b is paired to a, and indices are in range.
func assertSymmetric(is *is.I, pairings []int) {
	for i, opp := range pairings {
		if opp == -1 {
			continue
		}
		is.True(opp >= 0 && opp < len(pairings)) // opponent index in range
		is.True(opp != i)                        // nobody plays themselves
		is.Equal(pairings[opp], i)               // pairing is symmetric
	}
}

func TestAustralianCasement(t *testing.T) {
	is := is.New(t)

	tests := []struct {
		name string
		n    int
		want []int
	}{
		{"two", 2, []int{1, 0}},
		{"four", 4, []int{2, 3, 0, 1}},
		{"six", 6, []int{3, 4, 5, 0, 1, 2}},
		{"three odd bye middle", 3, []int{2, -1, 0}},
		{"five odd bye middle", 5, []int{3, 4, -1, 0, 1}},
		{"one", 1, []int{-1}},
		{"zero", 0, []int{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := australianCasement(tc.n)
			is.NoErr(equalPairings(tc.want, got))
			assertSymmetric(is, got)
		})
	}
}

func TestAustralianRoundOneDispatch(t *testing.T) {
	is := is.New(t)

	// Round 0 (the first round) must use the casement draw regardless of
	// any repeat data.
	t.Run("even casement", func(t *testing.T) {
		m := auMembers(0, 0, nil, "a", "b", "c", "d")
		pairings, err := pairAustralianDraw(m)
		is.NoErr(err)
		assertSymmetric(is, pairings)
		got := idPairings(m, pairings)
		is.Equal(got, map[string]string{"a": "c", "b": "d", "c": "a", "d": "b"})
	})

	t.Run("odd casement gives middle the bye", func(t *testing.T) {
		m := auMembers(0, 0, nil, "a", "b", "c", "d", "e")
		pairings, err := pairAustralianDraw(m)
		is.NoErr(err)
		assertSymmetric(is, pairings)
		got := idPairings(m, pairings)
		// a..e in standings order; n/2 = 2 => "c" byes; a-d, b-e.
		is.Equal(got, map[string]string{"a": "d", "b": "e", "c": "", "d": "a", "e": "b"})
	})
}

// auMembersBlocking builds a round-0 even pool in standings order (strictly
// decreasing wins preserve the input order) and applies a director block list:
// blocking[id] is the set of opponent ids that id refuses to play. The block is
// honored symmetrically by pairable, so only one side needs to list it.
func auMembersBlocking(blocking map[string][]string, ids ...string) *entity.UnpairedPoolMembers {
	poolMembers := make([]*entity.PoolMember, len(ids))
	for i, id := range ids {
		poolMembers[i] = &entity.PoolMember{
			Id:       id,
			Wins:     len(ids) - i,
			Blocking: blocking[id],
		}
	}
	return &entity.UnpairedPoolMembers{
		PoolMembers: poolMembers,
		RoundControls: &pb.RoundControl{
			PairingMethod: pb.PairingMethod_AUSTRALIAN_DRAW,
			Round:         0,
		},
		Repeats: map[string]int{},
	}
}

func TestAustralianCasementAvoidsBlockedPair(t *testing.T) {
	is := is.New(t)

	// Round 0, standings a,b,c,d. The plain casement pairs a-c and b-d (top half
	// 0,1 vs bottom half 2,3), but a blocks c. The recursive repair re-casements
	// the unpaired players rather than drifting to a neighbor: a takes the next
	// bottom-half player d, and b takes the freed bottom-half player c, so the
	// draw stays top-vs-bottom (a-d, b-c) with no needless top-vs-top pair.
	m := auMembersBlocking(map[string][]string{"a": {"c"}}, "a", "b", "c", "d")
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{"a": "d", "b": "c", "c": "b", "d": "a"})

	// The repaired draw must never output the blocked pair.
	is.True(got["a"] != "c")
	is.True(got["c"] != "a")
}

func TestAustralianCasementWithPrepair(t *testing.T) {
	is := is.New(t)

	// Director prepairs the top two players (a-b) in round 0. The remaining
	// pool is pulled out and run through the australian draw, exactly as the
	// caller does for byes and gibsonized players, then recombined. This
	// confirms the prepair remove/recombine path applies in round 1, not just
	// rounds 2+. The remaining pool also carries a block (c refuses e, its
	// natural casement partner) so the round-1 draw must honor prepair removal
	// and blocking together.
	full := auMembersBlocking(map[string][]string{"c": {"e"}},
		"a", "b", "c", "d", "e", "f")

	prepaired := map[string]string{"a": "b", "b": "a"}
	remaining := []*entity.PoolMember{}
	for _, pm := range full.PoolMembers {
		if _, ok := prepaired[pm.Id]; ok {
			continue
		}
		remaining = append(remaining, pm)
	}
	is.Equal(len(remaining), 4)

	sub := &entity.UnpairedPoolMembers{
		PoolMembers:   remaining,
		RoundControls: full.RoundControls,
		Repeats:       full.Repeats,
	}
	subPairings, err := pairAustralianDraw(sub)
	is.NoErr(err)
	assertSymmetric(is, subPairings)

	// Casement on c,d,e,f would pair c-e and d-f (top half c,d vs bottom half
	// e,f), but c blocks e. The recursive repair re-casements rather than
	// drifting: c takes the next bottom-half player f and d takes the freed
	// bottom-half player e, keeping the draw top-vs-bottom (c-f, d-e) with no
	// needless top-vs-top pair. The prepaired a-b are untouched (not in the
	// pool).
	subGot := idPairings(sub, subPairings)
	is.Equal(subGot, map[string]string{"c": "f", "d": "e", "e": "d", "f": "c"})
	is.True(subGot["c"] != "e")

	// Recombine the prepairs with the sub-pool result and confirm everyone is
	// accounted for exactly once and the whole thing is a valid involution.
	idToIndex := map[string]int{}
	for i, pm := range full.PoolMembers {
		idToIndex[pm.Id] = i
	}
	combined := make([]int, len(full.PoolMembers))
	for i := range combined {
		combined[i] = -2 // sentinel: must be overwritten
	}
	for id, opp := range prepaired {
		combined[idToIndex[id]] = idToIndex[opp]
	}
	for i, opp := range subPairings {
		id := sub.PoolMembers[i].Id
		if opp < 0 {
			combined[idToIndex[id]] = -1
		} else {
			combined[idToIndex[id]] = idToIndex[sub.PoolMembers[opp].Id]
		}
	}
	for _, v := range combined {
		is.True(v != -2) // every player was paired
	}
	assertSymmetric(is, combined)
	combinedGot := idPairings(full, combined)
	is.Equal(combinedGot, map[string]string{
		"a": "b", "b": "a",
		"c": "f", "d": "e", "e": "d", "f": "c",
	})
}

func TestAustralianCasementImpossibleUnderBlockingErrors(t *testing.T) {
	is := is.New(t)

	// Round 0 with four players where a refuses every possible opponent. No
	// complete first-round draw exists (the pool is even, so a cannot take a
	// bye), and the draw must fail with a clear error rather than emit a
	// blocked pair or a partial result.
	m := auMembersBlocking(map[string][]string{"a": {"b", "c", "d"}},
		"a", "b", "c", "d")
	_, err := pairAustralianDraw(m)
	is.True(err != nil) // no legal first-round pairing
}

// countTopVsTop returns how many produced pairs join two top-half positions.
// The pool is even in these tests and standings position equals input index
// (strictly decreasing wins), so the top half is positions [0, n/2). Each
// unordered pair is counted once.
func countTopVsTop(pairings []int) int {
	n := len(pairings)
	half := n / 2
	count := 0
	for p, opp := range pairings {
		if opp > p && p < half && opp < half {
			count++
		}
	}
	return count
}

// canByeAll permits every position to take the bye, the dispatcher's default.
func canByeAll(pos int) bool { return true }

// casementBlockedField runs the round-1 casement on a field of n real positions
// under blocked, padding an odd field with a phantom bye exactly as the
// dispatcher does and mapping the phantom's partner back to a -1 bye. canBye
// reports whether a real position may take the bye (the phantom is blocked
// against positions that may not); pass canByeAll for the dispatcher's default.
func casementBlockedField(n int, blocked [][]bool, canBye func(pos int) bool) ([]int, error) {
	m := n
	bye := -1
	if n%2 == 1 {
		m = n + 1
		bye = n
	}
	bl := make([][]bool, m)
	for i := range bl {
		bl[i] = make([]bool, m)
	}
	for a := 0; a < n; a++ {
		for b := a + 1; b < n; b++ {
			bl[a][b] = blocked[a][b]
			bl[b][a] = blocked[b][a]
		}
	}
	if bye >= 0 {
		for p := 0; p < n; p++ {
			if !canBye(p) {
				bl[p][bye] = true
				bl[bye][p] = true
			}
		}
	}
	got, err := australianCasementBlocked(m, bl)
	if err != nil {
		return nil, err
	}
	out := make([]int, n)
	for p := 0; p < n; p++ {
		if got[p] == bye {
			out[p] = -1
		} else {
			out[p] = got[p]
		}
	}
	return out, nil
}

func TestAustralianBlockedMatchesPlainCasement(t *testing.T) {
	is := is.New(t)

	// With no blocks the backtracking repair must reproduce australianCasement
	// byte for byte, including the odd middle bye, so the common round-1 draw
	// is unchanged. Trying the exact casement opponent first guarantees this;
	// the lone middle of an odd field is reached only once every other position
	// is paired (the phantom bye lands on it), so it takes the bye.
	for _, n := range []int{2, 3, 4, 5, 6, 7, 8, 9} {
		blocked := make([][]bool, n)
		for i := range blocked {
			blocked[i] = make([]bool, n)
		}
		got, err := casementBlockedField(n, blocked, canByeAll)
		is.NoErr(err) // an unblocked field is always pairable
		is.NoErr(equalPairings(australianCasement(n), got))
		assertSymmetric(is, got)
	}
}

func TestAustralianBlockedAvoidsTopVsTopWhenAlternativeExists(t *testing.T) {
	is := is.New(t)

	// Round 0, standings a..f. Plain casement pairs the top half a,b,c against
	// the bottom half d,e,f (a-d, b-e, c-f). a blocks its ideal partner d. The
	// backtracking repair must keep the draw top-vs-bottom: a slides to the next
	// bottom-half player e, and the freed bottom-half player d is taken by the
	// next top b, so no top-vs-top pair is created even though a-b was an option.
	m := auMembersBlocking(map[string][]string{"a": {"d"}},
		"a", "b", "c", "d", "e", "f")
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{
		"a": "e", "e": "a", "b": "d", "d": "b", "c": "f", "f": "c",
	})
	is.Equal(countTopVsTop(pairings), 0) // the whole point: no needless top-vs-top
}

func TestAustralianBlockedForcesMinimalTopVsTop(t *testing.T) {
	is := is.New(t)

	// Round 0, standings a..f. a and b each block every bottom-half player
	// (d,e,f), so neither top can pair a bottom and no zero-top-vs-top draw
	// exists. Phase 1 (flips disallowed) therefore fails, and phase 2 admits the
	// single unavoidable top-vs-top: a pairs its nearest top b. The remaining
	// field is then re-casemented over the players still unpaired -- c,d,e,f --
	// whose casement is c-e (the top of that set's bottom half) and d-f. The one
	// top-vs-top is forced; everything else stays top-vs-bottom.
	m := auMembersBlocking(map[string][]string{
		"a": {"d", "e", "f"},
		"b": {"d", "e", "f"},
	}, "a", "b", "c", "d", "e", "f")
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{
		"a": "b", "b": "a", "c": "e", "e": "c", "d": "f", "f": "d",
	})
	is.Equal(countTopVsTop(pairings), 1) // exactly one, and only because forced
}

func TestAustralianBlockedResolvesCorneredCaseCasementFaithfully(t *testing.T) {
	is := is.New(t)

	// Round 0, standings a..f. a alone blocks every bottom-half player (d,e,f),
	// so a cannot pair any bottom and no zero-top-vs-top draw exists. A naive
	// first-fit repair could paint into a corner here -- pairing b and c off against
	// bottoms first strands a among the bottoms it blocks. The two-phase repair
	// resolves it cleanly: phase 1 (flips disallowed) fails, phase 2 admits the
	// single unavoidable top-vs-top, a with its nearest top b, then re-casements
	// the players still unpaired -- c,d,e,f -- as c-e and d-f. Exactly one
	// top-vs-top, and only because a forced it.
	m := auMembersBlocking(map[string][]string{"a": {"d", "e", "f"}},
		"a", "b", "c", "d", "e", "f")
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{
		"a": "b", "b": "a", "c": "e", "e": "c", "d": "f", "f": "d",
	})

	// a must avoid all of d, e, f, and only one top-vs-top is forced (a's).
	is.True(got["a"] != "d")
	is.True(got["a"] != "e")
	is.True(got["a"] != "f")
	is.Equal(countTopVsTop(pairings), 1) // exactly the one forced by a's blocks
}

func TestAustralianBlockedGuaranteesZeroTopVsTopWhenAchievable(t *testing.T) {
	is := is.New(t)

	// Round 0, standings a..f (top half a,b,c; bottom half d,e,f). The blocks
	// are a-d, b-d, b-f. A zero-top-vs-top draw exists -- a-f, b-e, c-d -- and
	// the two-phase repair must produce one with no top-vs-top pair at all.
	//
	// This is the case a single first-complete-solution backtracker (preferring
	// bottom partners per position but committing to the first full draw it
	// completes) gets wrong: it can settle on a-e, b-c, d-f -- pairing b with c,
	// an avoidable top-vs-top -- because b's only free bottom at that point is f,
	// which b blocks. Running the whole search with flips forbidden first is what
	// turns "prefer bottom" into a guarantee: phase 1 either returns a draw with
	// zero top-vs-top or proves none exists.
	m := auMembersBlocking(map[string][]string{
		"a": {"d"},
		"b": {"d", "f"},
	}, "a", "b", "c", "d", "e", "f")
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{
		"a": "f", "f": "a", "b": "e", "e": "b", "c": "d", "d": "c",
	})
	is.Equal(countTopVsTop(pairings), 0) // the guarantee: none, because none is forced

	// None of the blocked pairs may appear.
	is.True(got["a"] != "d")
	is.True(got["b"] != "d")
	is.True(got["b"] != "f")
}

func TestAustralianBlockedByesATopPlayerToAvoidTopVsTop(t *testing.T) {
	is := is.New(t)

	// Odd field of five, standings a..e. Padded with a phantom bye, the casement
	// splits top half a,b,c vs bottom half d,e,BYE. The top player a blocks both
	// real bottoms d and e, so its only flip-free option is the phantom bye --
	// pairing fellow top b or c would be top-vs-top. The single zero-top-vs-top
	// draw therefore byes a -- a TOP player -- and pairs the rest top-vs-bottom
	// (b-d, c-e).
	m := auMembersBlocking(map[string][]string{"a": {"c", "d", "e"}},
		"a", "b", "c", "d", "e")
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{
		"a": "", "b": "d", "c": "e", "d": "b", "e": "c",
	})
	is.Equal(countTopVsTop(pairings), 0) // the guarantee, now on odd pools: a byes

	// a takes the bye and none of its blocked pairs appear.
	is.Equal(got["a"], "")
	is.True(got["a"] != "c")
	is.True(got["a"] != "d")
	is.True(got["a"] != "e")
}

func TestAustralianBlockedShiftsByeOffTheMiddle(t *testing.T) {
	is := is.New(t)

	// Odd field of five positions 0..4. Unblocked, the lone middle (position 2)
	// byes. Here the middle is barred from the bye (canBye(2) == false), so the
	// phantom bye must land on a different position. The casement pairs the
	// middle off (0-3, 2-4) and the bye shifts to position 1, leaving a
	// flip-free draw. The bye is just the phantom's partner, so it lands wherever
	// the even casement leaves a legal draw -- here on position 1, not the barred
	// middle.
	canBye := func(pos int) bool { return pos != 2 }
	n := 5
	blocked := make([][]bool, n)
	for i := range blocked {
		blocked[i] = make([]bool, n)
	}
	got, err := casementBlockedField(n, blocked, canBye)
	is.NoErr(err)
	assertSymmetric(is, got)
	is.NoErr(equalPairings([]int{3, -1, 4, 0, 2}, got))

	// The middle (2) is paired, and a different position byes.
	is.True(got[2] != -1) // the middle no longer byes
	is.Equal(got[1], -1)  // the bye shifted off the middle, onto position 1

	// Contrast: with every position bye-eligible the bye stays on the middle,
	// reproducing the plain casement.
	plain, err := casementBlockedField(n, blocked, canByeAll)
	is.NoErr(err)
	is.NoErr(equalPairings(australianCasement(n), plain)) // middle (2) byes
}

// existsMatching brute-forces whether the even field of n positions has a
// complete legal pairing (no bye). When zeroTopVsTopOnly is set it forbids any
// pair of top-half positions, so it reports whether a zero-top-vs-top draw
// exists. It is an independent oracle for australianCasementBlocked's two-phase
// guarantee, sharing none of its candidate-ordering logic.
func existsMatching(n int, blocked [][]bool, zeroTopVsTopOnly bool) bool {
	half := n / 2
	used := make([]bool, n)
	var rec func() bool
	rec = func() bool {
		p := -1
		for i := 0; i < n; i++ {
			if !used[i] {
				p = i
				break
			}
		}
		if p == -1 {
			return true
		}
		for q := p + 1; q < n; q++ {
			if used[q] || blocked[p][q] {
				continue
			}
			if zeroTopVsTopOnly && p < half && q < half {
				continue
			}
			used[p], used[q] = true, true
			if rec() {
				return true
			}
			used[p], used[q] = false, false
		}
		return false
	}
	return rec()
}

// checkBlockedAgainstOracle runs australianCasementBlocked on an even field
// under the given blocks and asserts the two-phase contract against the brute
// oracle: it succeeds exactly when a legal draw exists, never emits a blocked
// pair, and emits zero top-vs-top exactly when a zero-top-vs-top draw exists.
func checkBlockedAgainstOracle(is *is.I, n int, blocked [][]bool) {
	got, err := australianCasementBlocked(n, blocked)
	is.Equal(err == nil, existsMatching(n, blocked, false)) // succeeds iff legal draw exists
	if err != nil {
		return
	}
	assertSymmetric(is, got)
	for p, opp := range got {
		if opp >= 0 {
			is.True(!blocked[p][opp]) // never emits a blocked pair
		}
	}
	// The guarantee: zero top-vs-top exactly when one is achievable.
	is.Equal(countTopVsTop(got) == 0, existsMatching(n, blocked, true))
}

// checkBlockedAgainstOracleOdd is checkBlockedAgainstOracle for an odd field.
// The dispatcher pads an odd field with a phantom bye (pairable with everyone)
// into an even field, so the odd-pool contract is exactly the even contract on
// the padded field: pad and delegate. The phantom is the last position, always
// a bottom player, so it is never part of a top-vs-top pair -- the even
// guarantee on the padded field is the zero-top-vs-top guarantee for some valid
// bye choice on the original odd field.
func checkBlockedAgainstOracleOdd(is *is.I, n int, blocked [][]bool) {
	m := n + 1
	aug := make([][]bool, m)
	for i := range aug {
		aug[i] = make([]bool, m)
	}
	for a := 0; a < n; a++ {
		for b := a + 1; b < n; b++ {
			aug[a][b] = blocked[a][b]
			aug[b][a] = blocked[b][a]
		}
	}
	checkBlockedAgainstOracle(is, m, aug)
}

func TestAustralianBlockedGuaranteeMatchesOracle(t *testing.T) {
	is := is.New(t)

	// Exhaustively confirm the two-phase guarantee on every block configuration
	// of small even fields, judged against the independent brute oracle. The
	// pool is even here (canBye is never needed), matching the dispatcher path.
	for _, n := range []int{4, 6} {
		type pr struct{ a, b int }
		allPairs := []pr{}
		for a := 0; a < n; a++ {
			for b := a + 1; b < n; b++ {
				allPairs = append(allPairs, pr{a, b})
			}
		}
		for mask := 0; mask < (1 << len(allPairs)); mask++ {
			blocked := make([][]bool, n)
			for i := range blocked {
				blocked[i] = make([]bool, n)
			}
			for i, p := range allPairs {
				if mask&(1<<i) != 0 {
					blocked[p.a][p.b] = true
					blocked[p.b][p.a] = true
				}
			}
			checkBlockedAgainstOracle(is, n, blocked)
		}
	}

	// n=8 has too many block configurations to enumerate, so sample a fixed
	// pseudo-random spread of them. n=8 exercises the deeper recursion where a
	// per-position first-solution order is most prone to an avoidable top-vs-top.
	const n = 8
	state := uint64(0x9e3779b97f4a7c15) // fixed seed: deterministic test
	next := func() uint64 {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		return state
	}
	for trial := 0; trial < 20000; trial++ {
		blocked := make([][]bool, n)
		for i := range blocked {
			blocked[i] = make([]bool, n)
		}
		for a := 0; a < n; a++ {
			for b := a + 1; b < n; b++ {
				// Block each pair with probability ~1/4 to keep many fields legal.
				if next()%4 == 0 {
					blocked[a][b] = true
					blocked[b][a] = true
				}
			}
		}
		checkBlockedAgainstOracle(is, n, blocked)
	}
}

func TestAustralianBlockedGuaranteeMatchesOracleOdd(t *testing.T) {
	is := is.New(t)

	// The odd-pool counterpart of the even oracle test. An odd field is padded
	// with a phantom bye into an even field, so its guarantee is exactly the even
	// guarantee on the padded field: the phantom is the last position, always a
	// bottom, so it is never a top-vs-top partner, and whoever pairs it takes the
	// real bye. checkBlockedAgainstOracleOdd does the padding and delegates to the
	// even oracle. Confirm it exhaustively on small odd fields.
	for _, n := range []int{3, 5, 7} {
		type pr struct{ a, b int }
		allPairs := []pr{}
		for a := 0; a < n; a++ {
			for b := a + 1; b < n; b++ {
				allPairs = append(allPairs, pr{a, b})
			}
		}
		for mask := 0; mask < (1 << len(allPairs)); mask++ {
			blocked := make([][]bool, n)
			for i := range blocked {
				blocked[i] = make([]bool, n)
			}
			for i, p := range allPairs {
				if mask&(1<<i) != 0 {
					blocked[p.a][p.b] = true
					blocked[p.b][p.a] = true
				}
			}
			checkBlockedAgainstOracleOdd(is, n, blocked)
		}
	}

	// n=9 has too many block configurations to enumerate, so sample a fixed
	// pseudo-random spread, like the even oracle's n=8. This exercises the deeper
	// odd recursion where byeing a stuck top player rescues the no-flip pass.
	const n = 9
	state := uint64(0x9e3779b97f4a7c15) // fixed seed: deterministic test
	next := func() uint64 {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		return state
	}
	for trial := 0; trial < 20000; trial++ {
		blocked := make([][]bool, n)
		for i := range blocked {
			blocked[i] = make([]bool, n)
		}
		for a := 0; a < n; a++ {
			for b := a + 1; b < n; b++ {
				// Block each pair with probability ~1/4 to keep many fields legal.
				if next()%4 == 0 {
					blocked[a][b] = true
					blocked[b][a] = true
				}
			}
		}
		checkBlockedAgainstOracleOdd(is, n, blocked)
	}
}

func TestAustralianBlockedImpossibleStillErrors(t *testing.T) {
	is := is.New(t)

	// Round 0, even pool, where a refuses every opponent. The backtracking
	// repair exhausts every partner order and still cannot pair the field (no
	// bye on an even pool), so the draw must surface a clear error.
	m := auMembersBlocking(map[string][]string{"a": {"b", "c", "d", "e", "f"}},
		"a", "b", "c", "d", "e", "f")
	_, err := pairAustralianDraw(m)
	is.True(err != nil) // no legal first-round pairing exists
}

func TestAustralianHonorsBlockingRoundsTwoPlus(t *testing.T) {
	is := is.New(t)

	// A later round (round 1) must keep honoring director blocks through the
	// matcher, independent of any repeat history. Standings a,b,c,d with no
	// repeats: the natural pairing is a-b, but a blocks b, so a takes its next
	// legal opponent c, leaving b-d.
	poolMembers := []*entity.PoolMember{
		{Id: "a", Wins: 4, Blocking: []string{"b"}},
		{Id: "b", Wins: 3},
		{Id: "c", Wins: 2},
		{Id: "d", Wins: 1},
	}
	m := &entity.UnpairedPoolMembers{
		PoolMembers: poolMembers,
		RoundControls: &pb.RoundControl{
			PairingMethod: pb.PairingMethod_AUSTRALIAN_DRAW,
			Round:         1,
		},
		Repeats:      map[string]int{},
		RepeatRounds: map[string][]int{},
	}
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{"a": "c", "b": "d", "c": "a", "d": "b"})
	is.True(got["a"] != "b")
}

func TestAustralianRepeatAvoidance(t *testing.T) {
	is := is.New(t)

	// Round 1 (second round). Standings order a,b,c,d. In round 0 the
	// casement would have paired a-c and b-d, so those are the repeats to
	// avoid. With repeats forbidden, a should take the next-best legal
	// opponent (b), leaving c-d.
	repeats := map[string]int{
		GetRepeatKey("a", "c"): 1,
		GetRepeatKey("b", "d"): 1,
	}
	m := auMembers(1, 0, repeats, "a", "b", "c", "d")
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{"a": "b", "b": "a", "c": "d", "d": "c"})
}

func TestAustralianPicksHighestLegalOpponent(t *testing.T) {
	is := is.New(t)

	// a has only played b. The matcher should still prefer the highest-standing
	// legal opponent, which is c (b is forbidden).
	repeats := map[string]int{
		GetRepeatKey("a", "b"): 1,
	}
	m := auMembers(1, 0, repeats, "a", "b", "c", "d")
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	// a-c (skipping forbidden b), leaving b-d.
	is.Equal(got, map[string]string{"a": "c", "b": "d", "c": "a", "d": "b"})
}

func TestAustralianResetRelaxesToAllowRematch(t *testing.T) {
	is := is.New(t)

	// Only two players and they have already met (in round 0). With no
	// repeats allowed the strict pass fails; the reset-relaxation loop raises
	// the reset round until the round-0 meeting is forgiven, at which point the
	// repeat is permitted.
	repeats := map[string]int{
		GetRepeatKey("a", "b"): 1,
	}
	// current round is 3 here; the single meeting is in round 0, so relaxing
	// the reset to round 1 already forgives it.
	m := auMembers(3, 0, repeats, "a", "b")
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{"a": "b", "b": "a"})
}

func TestAustralianMaxRepeatsAllowsRepeat(t *testing.T) {
	is := is.New(t)

	// Standings order a,b,c,d. The top player a has already met its natural
	// partner b once. MaxRepeats controls whether that single prior
	// meeting blocks an a-b rematch.
	repeats := map[string]int{
		GetRepeatKey("a", "b"): 1,
	}

	// maxRepeats = 1: one prior meeting is still acceptable, so the matcher
	// pairs a with its highest-standing opponent b, leaving c-d.
	m := auMembers(1, 1, repeats, "a", "b", "c", "d")
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{"a": "b", "b": "a", "c": "d", "d": "c"})

	// maxRepeats = 0: the same single prior meeting is now over the limit, so
	// a must skip b and take the next legal opponent c, leaving b-d.
	m2 := auMembers(1, 0, repeats, "a", "b", "c", "d")
	pairings2, err := pairAustralianDraw(m2)
	is.NoErr(err)
	got2 := idPairings(m2, pairings2)
	is.Equal(got2, map[string]string{"a": "c", "b": "d", "c": "a", "d": "b"})
}

func TestAustralianResetRoundBehavior(t *testing.T) {
	is := is.New(t)

	// The reset-relaxation loop, exercised directly, demonstrates the reset
	// semantics: hasPlayedSince returns false once the relaxed reset reaches the
	// current round, so a single retry past the reset succeeds even when
	// every pair has met.
	n := 2
	hasPlayedSince := func(a, b, reset int) bool {
		// Always "have played" until the reset is relaxed to the current
		// round.
		return reset < 5
	}
	blockedPair := func(a, b int) bool { return false }

	// earliestReset below currentRound: must relax up to currentRound.
	pairings, err := australianMatchWithReset(n, 2, 5, hasPlayedSince, blockedPair)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	// earliestReset == currentRound and still blocked: no relaxation is
	// possible, so this fails.
	alwaysPlayed := func(a, b, reset int) bool { return true }
	_, err = australianMatchWithReset(n, 5, 5, alwaysPlayed, blockedPair)
	is.True(err != nil)
}

func TestAustralianResetRoundForgivesEarlierRepeats(t *testing.T) {
	is := is.New(t)

	// Standings order a,b,c,d, pairing round 10. The history: a met b in
	// round 2 and a met c in round 9. With MaxRepeats 0, only meetings at or
	// after the reset round are avoided.
	repeatRounds := map[string][]int{
		GetRepeatKey("a", "b"): {2},
		GetRepeatKey("a", "c"): {9},
	}

	// reset_round 9 (1-based) emulates a day boundary at round 9: the round-2
	// a-b meeting is before it (1-based round 3 < 9) and may recur, but the
	// round-9 a-c meeting is at or after it (1-based round 10 >= 9) and is
	// avoided. The matcher therefore pairs a with its highest opponent b,
	// leaving c-d.
	m := auMembersRounds(10, 0, 9, nil, repeatRounds, "a", "b", "c", "d")
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{"a": "b", "b": "a", "c": "d", "d": "c"})

	// With reset_round 1 (the strict default: avoid every prior meeting) on
	// the same history, both prior meetings are avoided: a cannot take b
	// (round 2) or c (round 9), so a falls to d, leaving b-c (who never met).
	m2 := auMembersRounds(10, 0, 1, nil, repeatRounds, "a", "b", "c", "d")
	pairings2, err := pairAustralianDraw(m2)
	is.NoErr(err)
	got2 := idPairings(m2, pairings2)
	is.Equal(got2, map[string]string{"a": "d", "b": "c", "c": "b", "d": "a"})
}

func TestAustralianResetRoundRelaxesStepwise(t *testing.T) {
	is := is.New(t)

	// Two players a and b who met in (0-indexed) round 7, pairing round 10
	// with MaxRepeats 0. reset_round starts at 7 (1-based), giving a 0-indexed
	// threshold of 6, so the round-7 meeting is at or after the reset and is
	// avoided; with only two players the strict pass has no legal pairing. The
	// relaxation must raise the threshold one step at a time: thresholds 6 and
	// 7 both still avoid the round-7 meeting (the boundary is inclusive), but
	// threshold 8 forgives it, so the pair is finally allowed. This exercises
	// the per-round relaxation -- it is not the all-or-nothing jump the totals
	// map could only express.
	repeatRounds := map[string][]int{
		GetRepeatKey("a", "b"): {7},
	}
	m := auMembersRounds(10, 0, 7, nil, repeatRounds, "a", "b")
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{"a": "b", "b": "a"})
}

func TestAustralianRoundTwoPlusOddBye(t *testing.T) {
	is := is.New(t)

	// Round 1, three players a,b,c. a has already met both b and c, and b and c
	// have not met. With repeats forbidden a cannot be paired, so the phantom
	// bye falls to a and b-c play. This is the later-round odd path: the bye is
	// just the phantom's partner, found by the same backtracking matcher.
	repeats := map[string]int{
		GetRepeatKey("a", "b"): 1,
		GetRepeatKey("a", "c"): 1,
	}
	m := auMembers(1, 0, repeats, "a", "b", "c")
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{"a": "", "b": "c", "c": "b"})
}

func TestAustralianPrepairRemoveAndRecombine(t *testing.T) {
	is := is.New(t)

	// Director prepairs the top two players (a-b). The remaining pool
	// (c,d,e,f) is run through the australian draw and the results are
	// recombined. This mirrors the splitMembers/combinePairings pattern the
	// factor method already uses.
	full := auMembers(0, 0, nil, "a", "b", "c", "d", "e", "f")

	// Prepair a-b (positions 0 and 1) and pull them out of the pool.
	prepaired := map[string]string{"a": "b", "b": "a"}
	remaining := []*entity.PoolMember{}
	removedIdx := []int{}
	for i, pm := range full.PoolMembers {
		if _, ok := prepaired[pm.Id]; ok {
			removedIdx = append(removedIdx, i)
			continue
		}
		remaining = append(remaining, pm)
	}
	is.Equal(len(remaining), 4)

	sub := &entity.UnpairedPoolMembers{
		PoolMembers:   remaining,
		RoundControls: full.RoundControls,
		Repeats:       full.Repeats,
	}
	subPairings, err := pairAustralianDraw(sub)
	is.NoErr(err)
	assertSymmetric(is, subPairings)

	// Casement on c,d,e,f => c-e, d-f.
	subGot := idPairings(sub, subPairings)
	is.Equal(subGot, map[string]string{"c": "e", "d": "f", "e": "c", "f": "d"})

	// Recombine: reconstruct full-pool pairings from the prepairs plus the
	// sub-pool result, and verify everyone is accounted for exactly once.
	idToIndex := map[string]int{}
	for i, pm := range full.PoolMembers {
		idToIndex[pm.Id] = i
	}
	combined := make([]int, len(full.PoolMembers))
	for i := range combined {
		combined[i] = -2 // sentinel: must be overwritten
	}
	for id, opp := range prepaired {
		combined[idToIndex[id]] = idToIndex[opp]
	}
	for i, opp := range subPairings {
		id := sub.PoolMembers[i].Id
		if opp < 0 {
			combined[idToIndex[id]] = -1
		} else {
			combined[idToIndex[id]] = idToIndex[sub.PoolMembers[opp].Id]
		}
	}
	for _, v := range combined {
		is.True(v != -2) // every player was paired
	}
	assertSymmetric(is, combined)
	combinedGot := idPairings(full, combined)
	is.Equal(combinedGot, map[string]string{
		"a": "b", "b": "a",
		"c": "e", "d": "f", "e": "c", "f": "d",
	})

	// Sanity: removedIdx captured the prepaired positions.
	is.Equal(removedIdx, []int{0, 1})
}

func TestAustralianDispatchThroughPair(t *testing.T) {
	is := is.New(t)

	// The public Pair() dispatcher must route AUSTRALIAN_DRAW to the new
	// method.
	m := auMembers(0, 0, nil, "a", "b", "c", "d")
	pairings, err := Pair(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{"a": "c", "b": "d", "c": "a", "d": "b"})
}

func TestAustralianEmptyPool(t *testing.T) {
	is := is.New(t)

	m := auMembers(1, 0, nil)
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	is.Equal(len(pairings), 0)
}

func TestAustralianStandingsResort(t *testing.T) {
	is := is.New(t)

	// Members handed in out of standings order must be re-sorted before
	// pairing. Here the input order is reversed relative to wins, so the
	// casement must operate on the corrected order.
	poolMembers := []*entity.PoolMember{
		{Id: "low", Wins: 0},
		{Id: "mid", Wins: 1},
		{Id: "high", Wins: 2},
		{Id: "top", Wins: 3},
	}
	m := &entity.UnpairedPoolMembers{
		PoolMembers: poolMembers,
		RoundControls: &pb.RoundControl{
			PairingMethod: pb.PairingMethod_AUSTRALIAN_DRAW,
			Round:         0,
		},
		Repeats: map[string]int{},
	}
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	// Standings order: top, high, mid, low => casement top-mid, high-low.
	is.Equal(got, map[string]string{
		"top": "mid", "mid": "top", "high": "low", "low": "high",
	})
}

func TestAustralianSpreadTiebreak(t *testing.T) {
	is := is.New(t)

	// Equal wins: spread (desc) then id (asc) decide standings order.
	poolMembers := []*entity.PoolMember{
		{Id: "zed", Wins: 1, Spread: 10},
		{Id: "ann", Wins: 1, Spread: 50},
		{Id: "bob", Wins: 1, Spread: 50},
		{Id: "cat", Wins: 1, Spread: -5},
	}
	standings := sortAustralianStandings(poolMembers)
	order := make([]string, len(standings))
	for i, s := range standings {
		order[i] = s.id
	}
	// ann & bob lead on spread 50 (ann before bob by id), then zed (10),
	// then cat (-5).
	is.Equal(order, []string{"ann", "bob", "zed", "cat"})
	is.Equal(fmt.Sprintf("%d", standings[0].wins*2+standings[0].draws), "2")
}

func TestAustralianRoundTwoPlusBacktracksToMatch(t *testing.T) {
	is := is.New(t)

	// Round 1 (a later round). Standings a,b,c,d. Director blocks a-b and b-d.
	// The matcher must backtrack: pairing a with its nearest legal opponent c
	// (a-b is blocked) strands b, whose only remaining players are the
	// already-paired c and the blocked d, and the even pool has no bye. So the
	// matcher undoes a-c, pairs a with d instead, and completes with b-c. A
	// naive first-fit walk would instead fail outright here.
	poolMembers := []*entity.PoolMember{
		{Id: "a", Wins: 4, Blocking: []string{"b"}},
		{Id: "b", Wins: 3, Blocking: []string{"d"}},
		{Id: "c", Wins: 2},
		{Id: "d", Wins: 1},
	}
	m := &entity.UnpairedPoolMembers{
		PoolMembers: poolMembers,
		RoundControls: &pb.RoundControl{
			PairingMethod: pb.PairingMethod_AUSTRALIAN_DRAW,
			Round:         1,
		},
		Repeats:      map[string]int{},
		RepeatRounds: map[string][]int{},
	}
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{"a": "d", "b": "c", "c": "b", "d": "a"})
}

func TestAustralianCasementOddByesBlockedTopNotFlip(t *testing.T) {
	is := is.New(t)

	// Round 0, seven players a..g. The casement adds a phantom bye and splits
	// the field into top half a,b,c,d and bottom half e,f,g,BYE. a blocks every
	// real bottom (e,f,g), so a's only flip-free option is the phantom bye; a
	// takes it and the rest casement cleanly: b-e, c-f, d-g. a must NOT be
	// paired with the fourth seed d -- that is a top-vs-top flip and belongs to
	// the fallback pass, which is never reached because byeing a is flip-free.
	m := auMembersBlocking(map[string][]string{"a": {"e", "f", "g"}},
		"a", "b", "c", "d", "e", "f", "g")
	pairings, err := pairAustralianDraw(m)
	is.NoErr(err)
	assertSymmetric(is, pairings)
	got := idPairings(m, pairings)
	is.Equal(got, map[string]string{
		"a": "", "b": "e", "c": "f", "d": "g",
		"e": "b", "f": "c", "g": "d",
	})
}

func TestAustralianMatchMatchesOracle(t *testing.T) {
	is := is.New(t)

	// The later-round backtracking matcher must find a complete draw exactly
	// when one exists, and never emit a blocked pair. Confirm exhaustively on
	// small even fields against the independent brute oracle, then on a sampled
	// spread of the larger field where a naive first-fit walk would be most
	// prone to stranding a player.
	check := func(n int, blocked [][]bool) {
		got, ok := australianMatch(n, blocked)
		is.Equal(ok, existsMatching(n, blocked, false)) // succeeds iff a draw exists
		if !ok {
			return
		}
		assertSymmetric(is, got)
		for p, opp := range got {
			if opp >= 0 {
				is.True(!blocked[p][opp]) // never emits a blocked pair
			}
		}
	}

	for _, n := range []int{2, 4, 6} {
		type pr struct{ a, b int }
		allPairs := []pr{}
		for a := 0; a < n; a++ {
			for b := a + 1; b < n; b++ {
				allPairs = append(allPairs, pr{a, b})
			}
		}
		for mask := 0; mask < (1 << len(allPairs)); mask++ {
			blocked := make([][]bool, n)
			for i := range blocked {
				blocked[i] = make([]bool, n)
			}
			for i, p := range allPairs {
				if mask&(1<<i) != 0 {
					blocked[p.a][p.b] = true
					blocked[p.b][p.a] = true
				}
			}
			check(n, blocked)
		}
	}

	// n=8 has too many block configurations to enumerate, so sample a fixed
	// pseudo-random spread, like the casement oracle's n=8.
	const n = 8
	state := uint64(0x9e3779b97f4a7c15) // fixed seed: deterministic test
	next := func() uint64 {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		return state
	}
	for trial := 0; trial < 20000; trial++ {
		blocked := make([][]bool, n)
		for i := range blocked {
			blocked[i] = make([]bool, n)
		}
		for a := 0; a < n; a++ {
			for b := a + 1; b < n; b++ {
				// Block each pair with probability ~1/4 to keep many fields legal.
				if next()%4 == 0 {
					blocked[a][b] = true
					blocked[b][a] = true
				}
			}
		}
		check(n, blocked)
	}
}
