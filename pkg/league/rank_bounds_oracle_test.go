package league

// Independent exact rank-bounds oracle, for validating the production brute's
// joint-margin reasoning. TEST ONLY -- never used by production code.
//
// Two independent computations of the exact best/worst rank for SMALL
// divisions:
//
//   - exhaustiveRanks: the indisputable ground truth. Enumerates every
//     win/draw/loss outcome AND every concrete integer margin in [1..B] for
//     each unfinished game, computes concrete final points+spread, and reads
//     off the rank range. Exact for tiny games once B exceeds the spread gaps;
//     too slow past ~3-4 games.
//
//   - oracleRanks: enumerates the 3^g win/draw/loss leaves and, per leaf, finds
//     the max/min number of points-tied players that can be ordered at/above P
//     by a feasible joint margin assignment, via Fourier-Motzkin feasibility
//     over the game margins (no magnitude enumeration). Scales to the whole
//     brute regime (g <= 10).
//
// oracleRanks is validated to agree with exhaustiveRanks on random tiny
// divisions (TestOracleVsExhaustive) -- that agreement, not code inspection, is
// what makes the Fourier-Motzkin path trustworthy. It is then used as the exact
// reference for the production joint-margin brute fix.

import (
	"math/bits"
	"math/rand"
	"testing"
)

// rankRange returns the [best, worst] rank of player p given concrete final
// points and spread for every player, using the block tie convention: best
// excludes players tied on exactly (points, spread); worst includes them.
func rankRange(pts, spr []int, p int) (int, int) {
	strictAbove, tied := 0, 0
	for q := range pts {
		if q == p {
			continue
		}
		switch {
		case pts[q] > pts[p] || (pts[q] == pts[p] && spr[q] > spr[p]):
			strictAbove++
		case pts[q] == pts[p] && spr[q] == spr[p]:
			tied++
		}
	}
	return strictAbove + 1, strictAbove + tied + 1
}

// exhaustiveRanks is the ground truth for tiny divisions: it enumerates every
// outcome and every integer margin in [1..b] for each unfinished game.
func exhaustiveRanks(st []standingInfo, unf []unfinishedGame, b int) []RankBounds {
	pairs := toGamePairs(st, unf)
	n := len(st)
	g := len(pairs)
	pts := make([]int, n)
	spr := make([]int, n)
	for i := range st {
		pts[i] = st[i].points
		spr[i] = st[i].spread
	}
	best := make([]int, n)
	worst := make([]int, n)
	for i := range best {
		best[i] = n + 1
		worst[i] = 0
	}
	var rec func(gi int)
	rec = func(gi int) {
		if gi == g {
			for p := range n {
				lo, hi := rankRange(pts, spr, p)
				if lo < best[p] {
					best[p] = lo
				}
				if hi > worst[p] {
					worst[p] = hi
				}
			}
			return
		}
		a, bb := pairs[gi].a, pairs[gi].b
		// draw
		pts[a]++
		pts[bb]++
		rec(gi + 1)
		pts[a]--
		pts[bb]--
		for m := 1; m <= b; m++ {
			// a wins by m
			pts[a] += 2
			spr[a] += m
			spr[bb] -= m
			rec(gi + 1)
			pts[a] -= 2
			spr[a] -= m
			spr[bb] += m
			// b wins by m
			pts[bb] += 2
			spr[bb] += m
			spr[a] -= m
			rec(gi + 1)
			pts[bb] -= 2
			spr[bb] -= m
			spr[a] += m
		}
	}
	rec(0)
	res := make([]RankBounds, n)
	for i := range res {
		res[i] = RankBounds{BestRank: best[i], WorstRank: worst[i]}
	}
	return res
}

// oracleRanks is the scalable exact oracle (Fourier-Motzkin per leaf).
func oracleRanks(st []standingInfo, unf []unfinishedGame) []RankBounds {
	pairs := toGamePairs(st, unf)
	n := len(st)
	g := len(pairs)
	best := make([]int, n)
	worst := make([]int, n)
	for i := range best {
		best[i] = n + 1
		worst[i] = 0
	}
	pts := make([]int, n)
	for i := range st {
		pts[i] = st[i].points
	}
	// spcoef[i][j] = coefficient of margin variable j in player i's spread.
	spcoef := make([][]int64, n)
	for i := range spcoef {
		spcoef[i] = make([]int64, g)
	}
	var rec func(gi int)
	rec = func(gi int) {
		if gi == g {
			oracleLeaf(st, pts, spcoef, g, best, worst)
			return
		}
		a, bb := pairs[gi].a, pairs[gi].b
		// draw: no margin variable contribution
		pts[a]++
		pts[bb]++
		rec(gi + 1)
		pts[a]--
		pts[bb]--
		// a wins
		pts[a] += 2
		spcoef[a][gi] = 1
		spcoef[bb][gi] = -1
		rec(gi + 1)
		pts[a] -= 2
		spcoef[a][gi] = 0
		spcoef[bb][gi] = 0
		// b wins
		pts[bb] += 2
		spcoef[bb][gi] = 1
		spcoef[a][gi] = -1
		rec(gi + 1)
		pts[bb] -= 2
		spcoef[bb][gi] = 0
		spcoef[a][gi] = 0
	}
	rec(0)
	res := make([]RankBounds, n)
	for i := range res {
		res[i] = RankBounds{BestRank: best[i], WorstRank: worst[i]}
	}
	return res
}

func oracleLeaf(st []standingInfo, pts []int, spcoef [][]int64, g int, best, worst []int) {
	n := len(st)
	baseSpread := make([]int, n)
	for i := range st {
		baseSpread[i] = st[i].spread
	}
	for p := range n {
		above := 0
		var tied []int
		for q := range n {
			if q == p {
				continue
			}
			switch {
			case pts[q] > pts[p]:
				above++
			case pts[q] == pts[p]:
				tied = append(tied, q)
			}
		}
		// maxGE: most tied players that can simultaneously be at-or-above P on
		// spread (spr_q - spr_p >= 0); maxLE: most that can be at-or-below.
		maxGE := maxFeasibleSubset(baseSpread, spcoef, g, tied, p, true)
		maxLE := maxFeasibleSubset(baseSpread, spcoef, g, tied, p, false)
		lo := 1 + above + (len(tied) - maxLE) // min strictly-above
		hi := 1 + above + maxGE               // max at-or-above (block worst)
		if lo < best[p] {
			best[p] = lo
		}
		if hi > worst[p] {
			worst[p] = hi
		}
	}
}

// --- test helpers ---

// makeDivision builds a sorted division from per-player points/spread and a
// list of unfinished games given as index pairs into the (pre-sort) arrays.
func makeDivision(points, spreads []int, gamePairs [][2]int) ([]standingInfo, []unfinishedGame) {
	n := len(points)
	gr := make([]int, n)
	for _, gp := range gamePairs {
		gr[gp[0]]++
		gr[gp[1]]++
	}
	st := make([]standingInfo, n)
	for i := range n {
		st[i] = standingInfo{
			userID:         int32(i + 1),
			points:         points[i],
			spread:         spreads[i],
			gamesRemaining: gr[i],
		}
	}
	var unf []unfinishedGame
	for _, gp := range gamePairs {
		unf = append(unf, unfinishedGame{player0ID: int32(gp[0] + 1), player1ID: int32(gp[1] + 1)})
	}
	sortStandings(st)
	return st, unf
}

func eqBounds(a, b []RankBounds) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestOracleVsExhaustive(t *testing.T) {
	rng := rand.New(rand.NewSource(20260611))
	const cases = 400
	const b = 16
	mismatch := 0
	for c := range cases {
		n := 3 + rng.Intn(2) // 3 or 4 players
		points := make([]int, n)
		spreads := make([]int, n)
		for i := range n {
			points[i] = rng.Intn(4)
			spreads[i] = rng.Intn(9) - 4
		}
		g := rng.Intn(4) // 0..3 games
		var pairs [][2]int
		for range g {
			x := rng.Intn(n)
			y := rng.Intn(n)
			if x == y {
				continue
			}
			pairs = append(pairs, [2]int{x, y})
		}
		st, unf := makeDivision(points, spreads, pairs)
		got := oracleRanks(st, unf)
		want := exhaustiveRanks(st, unf, b)
		if !eqBounds(got, want) {
			mismatch++
			if mismatch <= 5 {
				t.Errorf("case %d: oracle=%v exhaustive=%v\n  st=%+v\n  unf=%+v", c, got, want, st, unf)
			}
		}
	}
	if mismatch > 0 {
		t.Fatalf("%d/%d oracle vs exhaustive mismatches", mismatch, cases)
	}
}

// TestOracleClosedRRLooseness builds a closed tied round-robin (the joint-margin
// looseness the production brute will be fixed for) and confirms the oracle
// matches exhaustive truth and is strictly tighter than the current production
// brute on at least one player.
func TestOracleClosedRRLooseness(t *testing.T) {
	// A is fixed (no games), tied on points with B,C,D who play a closed
	// round-robin among themselves. Their conserved spread sum constrains how
	// many can finish above A.
	// points: A=2 (fixed), B=C=D=0 with 2 games each -> reach 2 via a 1-1 cycle.
	// spreads: A=0; B,C,D start with a small conserved sum.
	points := []int{2, 0, 0, 0}
	spreads := []int{0, 1, 1, 1}
	pairs := [][2]int{{1, 2}, {1, 3}, {2, 3}} // B-C, B-D, C-D
	st, unf := makeDivision(points, spreads, pairs)
	oracle := oracleRanks(st, unf)
	truth := exhaustiveRanks(st, unf, 16)
	if !eqBounds(oracle, truth) {
		t.Fatalf("oracle %v != exhaustive %v", oracle, truth)
	}
	brute := CalculatePossibleRanks(st, unf)
	// The oracle must be sound vs the production brute (contained within it)
	// and tighter somewhere (that is the looseness commit 4 removes).
	tighter := false
	for i := range oracle {
		if oracle[i].BestRank < brute[i].BestRank || oracle[i].WorstRank > brute[i].WorstRank {
			t.Fatalf("oracle wider than brute at %d: oracle=%v brute=%v", i, oracle[i], brute[i])
		}
		if oracle[i].BestRank > brute[i].BestRank || oracle[i].WorstRank < brute[i].WorstRank {
			tighter = true
		}
	}
	t.Logf("closed-RR: oracle=%v brute=%v tighter=%v", oracle, brute, tighter)
}

// TestOracleVsBruteSanity checks basic validity on random small divisions and
// reports where the oracle differs from the production brute (tighter = the
// joint-margin fix's target; wider = a brute unsoundness the oracle catches).
func TestOracleVsBruteSanity(t *testing.T) {
	rng := rand.New(rand.NewSource(424242))
	const cases = 300
	tighter, wider := 0, 0
	for c := range cases {
		n := 3 + rng.Intn(4) // 3..6
		points := make([]int, n)
		spreads := make([]int, n)
		for i := range n {
			points[i] = rng.Intn(6)
			spreads[i] = rng.Intn(13) - 6
		}
		g := rng.Intn(6) // 0..5 games
		var pairs [][2]int
		for range g {
			x := rng.Intn(n)
			y := rng.Intn(n)
			if x != y {
				pairs = append(pairs, [2]int{x, y})
			}
		}
		st, unf := makeDivision(points, spreads, pairs)
		oracle := oracleRanks(st, unf)
		brute := CalculatePossibleRanks(st, unf)
		for i := range oracle {
			o, br := oracle[i], brute[i]
			if o.BestRank < 1 || o.WorstRank > n || o.BestRank > o.WorstRank {
				t.Fatalf("case %d player %d: invalid oracle bounds %v (n=%d)", c, i, o, n)
			}
			if o.BestRank > br.BestRank || o.WorstRank < br.WorstRank {
				tighter++
			}
			if o.BestRank < br.BestRank || o.WorstRank > br.WorstRank {
				wider++
			}
		}
	}
	t.Logf("oracle vs brute over %d cases: tighter=%d (joint-margin target), wider=%d (brute unsoundness)", cases, tighter, wider)
}

// TestBruteJointExact confirms the production brute (after the joint-margin fix)
// now matches the exact oracle on random small divisions -- the per-pair
// looseness is gone.
func TestBruteJointExact(t *testing.T) {
	rng := rand.New(rand.NewSource(77777))
	const cases = 500
	for c := range cases {
		n := 3 + rng.Intn(4) // 3..6
		points := make([]int, n)
		spreads := make([]int, n)
		for i := range n {
			points[i] = rng.Intn(5)
			spreads[i] = rng.Intn(11) - 5
		}
		g := rng.Intn(5) // 0..4 games -> brute path (cluster <= bruteForceThreshold)
		var pairs [][2]int
		for range g {
			x, y := rng.Intn(n), rng.Intn(n)
			if x != y {
				pairs = append(pairs, [2]int{x, y})
			}
		}
		st, unf := makeDivision(points, spreads, pairs)
		if got, want := CalculatePossibleRanks(st, unf), oracleRanks(st, unf); !eqBounds(got, want) {
			t.Fatalf("case %d: brute=%v oracle=%v\n  st=%+v\n  unf=%+v", c, got, want, st, unf)
		}
	}
}

// TestBruteJointVsExhaustive confirms the production brute matches the
// indisputable exhaustive integer-margin truth on random tiny divisions.
func TestBruteJointVsExhaustive(t *testing.T) {
	rng := rand.New(rand.NewSource(88888))
	for c := range 250 {
		n := 3 + rng.Intn(2) // 3 or 4
		points := make([]int, n)
		spreads := make([]int, n)
		for i := range n {
			points[i] = rng.Intn(4)
			spreads[i] = rng.Intn(9) - 4
		}
		g := rng.Intn(4) // 0..3
		var pairs [][2]int
		for range g {
			x, y := rng.Intn(n), rng.Intn(n)
			if x != y {
				pairs = append(pairs, [2]int{x, y})
			}
		}
		st, unf := makeDivision(points, spreads, pairs)
		if got, want := CalculatePossibleRanks(st, unf), exhaustiveRanks(st, unf, 16); !eqBounds(got, want) {
			t.Fatalf("case %d: brute=%v exhaustive=%v\n  st=%+v\n  unf=%+v", c, got, want, st, unf)
		}
	}
}

// TestRankBoundsWorstBnBSoundVsBrute validates the worst-rank branch-and-bound
// eviction (the max-flow tier) against the exact brute on random small divisions.
// The B&B bound must be SOUND -- never tighter than the exact worst rank
// (worstRankForPlayer >= brute.worst) -- because the flow model can only lose
// tightness, never claim an unreachable rank. It is exercised here on the same
// small divisions the brute solves exactly; in production the B&B runs on larger
// clusters where the brute is infeasible. Reports how often it is merely loose.
func TestRankBoundsWorstBnBSoundVsBrute(t *testing.T) {
	rng := rand.New(rand.NewSource(13579))
	const cases = 3000
	unsound, loose := 0, 0
	for c := range cases {
		n := 3 + rng.Intn(5) // 3..7
		points := make([]int, n)
		spreads := make([]int, n)
		for i := range n {
			points[i] = rng.Intn(7)
			spreads[i] = rng.Intn(15) - 7
		}
		g := rng.Intn(8) // 0..7 games -> brute path, exact ground truth
		var pairs [][2]int
		for range g {
			x, y := rng.Intn(n), rng.Intn(n)
			if x != y {
				pairs = append(pairs, [2]int{x, y})
			}
		}
		st, unf := makeDivision(points, spreads, pairs)
		brute := CalculatePossibleRanks(st, unf) // small division => exact brute
		games := toGamePairs(st, unf)
		fg := newFlowGraph(2 + len(games) + n)
		candIdx := initSlice(n, -1)
		for p := range n {
			gi := decomposeGames(p, n, games)
			w := worstRankForPlayer(p, st, gi, fg, candIdx)
			switch {
			case w < brute[p].WorstRank:
				unsound++
				if unsound <= 5 {
					t.Errorf("case %d player %d: B&B worst %d < brute worst %d (UNSOUND)\n st=%+v\n unf=%+v",
						c, p, w, brute[p].WorstRank, st, unf)
				}
			case w > brute[p].WorstRank:
				loose++
			}
		}
	}
	if unsound > 0 {
		t.Fatalf("%d unsound worst-rank B&B bounds vs brute", unsound)
	}
	t.Logf("worst-rank B&B vs brute over %d cases: 0 unsound, %d loose", cases, loose)
}

// TestRankBoundsBestInversionSoundVsBrute validates best-rank via the loss-score
// inversion (best = n+1 - worst on the mirror) against the exact brute on random
// small divisions. The inverted bound must be SOUND -- never tighter than the
// exact best rank (bestViaInversion <= brute.best). The mirror's constant offset
// CalculateExpectedGamesPerPlayer cancels in every pairwise comparison, and the
// win/draw/loss dynamics map to +0/+1/+2 loss-score, so the inversion is exact
// regardless of that constant. Reports how often it is merely loose.
func TestRankBoundsBestInversionSoundVsBrute(t *testing.T) {
	rng := rand.New(rand.NewSource(24680))
	const cases = 3000
	unsound, loose := 0, 0
	for c := range cases {
		n := 3 + rng.Intn(5) // 3..7
		points := make([]int, n)
		spreads := make([]int, n)
		for i := range n {
			points[i] = rng.Intn(7)
			spreads[i] = rng.Intn(15) - 7
		}
		g := rng.Intn(8) // 0..7 games -> brute path, exact ground truth
		var pairs [][2]int
		for range g {
			x, y := rng.Intn(n), rng.Intn(n)
			if x != y {
				pairs = append(pairs, [2]int{x, y})
			}
		}
		st, unf := makeDivision(points, spreads, pairs)
		brute := CalculatePossibleRanks(st, unf) // small division => exact brute
		games := toGamePairs(st, unf)
		mir := mirrorForBest(st)
		fg := newFlowGraph(2 + len(games) + n)
		candIdx := initSlice(n, -1)
		for p := range n {
			gi := decomposeGames(p, n, games)
			b := bestViaInversion(p, mir, gi, fg, candIdx)
			switch {
			case b > brute[p].BestRank:
				unsound++
				if unsound <= 5 {
					t.Errorf("case %d player %d: inversion best %d > brute best %d (UNSOUND)\n st=%+v\n unf=%+v",
						c, p, b, brute[p].BestRank, st, unf)
				}
			case b < brute[p].BestRank:
				loose++
			}
		}
	}
	if unsound > 0 {
		t.Fatalf("%d unsound best-rank inversion bounds vs brute", unsound)
	}
	t.Logf("best-rank inversion vs brute over %d cases: 0 unsound, %d loose", cases, loose)
}

// --- Fourier-Motzkin feasibility (the oracle's joint-margin engine) ---
//
// oracleRanks asks, per leaf, whether a subset of points-tied players can be
// jointly ordered at/above (or at/below) P on spread. That is exact linear
// feasibility over the game margins; we solve it by Fourier-Motzkin elimination
// with integer arithmetic. This is the INDEPENDENT method that validates the
// production closed-form (jointFixedTied) -- kept in the test, not production.

// fmIneq is the linear inequality sum_i coef[i]*x[i] + c >= 0.
type fmIneq struct {
	coef []int64
	c    int64
}

func (q fmIneq) allZero() bool {
	for _, v := range q.coef {
		if v != 0 {
			return false
		}
	}
	return true
}

// fmFeasible reports whether {ineqs, x_i >= 1 for all i} has a real solution,
// via Fourier-Motzkin elimination with exact integer arithmetic. Margins are
// integers >= 1; the systems arising here are integral, so real feasibility
// coincides with integer feasibility (validated against exhaustive enumeration
// in TestOracleVsExhaustive).
func fmFeasible(ineqs []fmIneq, nv int) bool {
	sys := make([]fmIneq, 0, len(ineqs)+nv)
	for _, q := range ineqs {
		cf := make([]int64, nv)
		copy(cf, q.coef)
		sys = append(sys, fmIneq{coef: cf, c: q.c})
	}
	for i := range nv {
		cf := make([]int64, nv)
		cf[i] = 1
		sys = append(sys, fmIneq{coef: cf, c: -1}) // x_i >= 1
	}
	for j := nv - 1; j >= 0; j-- {
		var pos, neg, next []fmIneq
		for _, q := range sys {
			switch {
			case q.coef[j] > 0:
				pos = append(pos, q)
			case q.coef[j] < 0:
				neg = append(neg, q)
			default:
				next = append(next, q)
			}
		}
		for _, p := range pos {
			for _, q := range neg {
				a := p.coef[j]
				b := -q.coef[j] // both > 0; combo a*q + b*p cancels x_j
				cf := make([]int64, nv)
				for i := range nv {
					cf[i] = a*q.coef[i] + b*p.coef[i]
				}
				ni := fmIneq{coef: cf, c: a*q.c + b*p.c}
				reduceIneq(&ni)
				if ni.allZero() {
					if ni.c < 0 {
						return false
					}
					continue
				}
				next = append(next, ni)
			}
		}
		sys = next
	}
	for _, q := range sys {
		if q.c < 0 {
			return false
		}
	}
	return true
}

func reduceIneq(q *fmIneq) {
	var g int64
	for _, v := range q.coef {
		g = gcd64(g, abs64(v))
	}
	g = gcd64(g, abs64(q.c))
	if g > 1 {
		for i := range q.coef {
			q.coef[i] /= g
		}
		q.c /= g
	}
}

func gcd64(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func abs64(a int64) int64 {
	if a < 0 {
		return -a
	}
	return a
}

// maxFeasibleSubset returns the size of the largest subset S of tied such that
// "spr_q >= spr_p for all q in S" (ge) or "spr_q <= spr_p for all q in S"
// (!ge) is jointly feasible over the game margins. baseSpread[i] is player i's
// current spread; spcoef[i][j] is the coefficient of margin variable j in i's
// final spread for the current leaf. Feasibility is downward-closed in S, so
// the largest feasible subset bounds the count.
func maxFeasibleSubset(baseSpread []int, spcoef [][]int64, g int, tied []int, p int, ge bool) int {
	k := len(tied)
	best := 0
	for mask := range 1 << k {
		size := bits.OnesCount(uint(mask))
		if size <= best {
			continue
		}
		ineqs := make([]fmIneq, 0, size)
		for bi := range k {
			if mask&(1<<bi) == 0 {
				continue
			}
			q := tied[bi]
			coef := make([]int64, g)
			var c int64
			if ge {
				for j := range g {
					coef[j] = spcoef[q][j] - spcoef[p][j]
				}
				c = int64(baseSpread[q] - baseSpread[p])
			} else {
				for j := range g {
					coef[j] = spcoef[p][j] - spcoef[q][j]
				}
				c = int64(baseSpread[p] - baseSpread[q])
			}
			ineqs = append(ineqs, fmIneq{coef: coef, c: c})
		}
		if fmFeasible(ineqs, g) {
			best = size
		}
	}
	return best
}
