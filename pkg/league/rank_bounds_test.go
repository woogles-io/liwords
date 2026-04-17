package league

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func si(userID int32, points, spread, gamesRemaining int) standingInfo {
	return standingInfo{
		userID:         userID,
		points:         points,
		spread:         spread,
		gamesRemaining: gamesRemaining,
	}
}

func uf(p0, p1 int32) unfinishedGame {
	return unfinishedGame{player0ID: p0, player1ID: p1}
}

func TestAllGamesComplete(t *testing.T) {
	standings := []standingInfo{
		si(1, 10, 100, 0),
		si(2, 8, 50, 0),
		si(3, 6, -50, 0),
	}
	bounds := CalculatePossibleRanks(standings, nil)
	for i, b := range bounds {
		if b.BestRank != i+1 || b.WorstRank != i+1 {
			t.Errorf("player %d: got %d-%d, want %d-%d", i, b.BestRank, b.WorstRank, i+1, i+1)
		}
	}
}

func TestTwoPlayersOneGame(t *testing.T) {
	// A(10pts) vs B(8pts), one game remaining between them.
	// If A wins: A=12, B=8 → ranks 1,2
	// If B wins: A=10, B=10 → tie on points, spread decides
	standings := []standingInfo{
		si(1, 10, 100, 1),
		si(2, 8, 50, 1),
	}
	games := []unfinishedGame{uf(1, 2)}
	bounds := CalculatePossibleRanks(standings, games)

	// A can finish 1st (wins) or 2nd (loses, tie on pts, B might have better spread)
	if bounds[0].BestRank != 1 {
		t.Errorf("A best: got %d, want 1", bounds[0].BestRank)
	}
	// B can finish 1st (wins and ties, beats on spread) or 2nd
	if bounds[1].BestRank != 1 {
		t.Errorf("B best: got %d, want 1", bounds[1].BestRank)
	}
	if bounds[1].WorstRank != 2 {
		t.Errorf("B worst: got %d, want 2", bounds[1].WorstRank)
	}
}

func TestMaxFlowTightensWorstRank(t *testing.T) {
	// 5 players. P (player 0) has 10 points, 0 games remaining.
	// Players 1,2,3 have 8 points each, and 2 games remaining.
	// Remaining games: 1v2, 1v3, 2v3 (round robin among 1,2,3).
	// Player 4 has 0 points, 0 games remaining.
	//
	// Without max-flow: each of 1,2,3 can individually reach 12 pts > 10,
	// so independent worst rank for P = 4.
	//
	// With max-flow: in the 3 games among {1,2,3}, there are 3 wins total.
	// For all 3 to reach 11+ pts, each needs at least 2 more points = 1 win.
	// 3 wins available, 3 needed → feasible! So worst rank = 4.
	//
	// But let's try a tighter case: P has 12 points.
	// Each of 1,2,3 needs 2 wins to reach 12. Total wins needed: 6.
	// Total wins available: 3. So at most 1 can reach 12 (needs 2 wins from 2 games).
	standings := []standingInfo{
		si(0, 12, 100, 0), // P
		si(1, 8, 0, 2),
		si(2, 8, 0, 2),
		si(3, 8, 0, 2),
		si(4, 0, -100, 0),
	}
	games := []unfinishedGame{uf(1, 2), uf(1, 3), uf(2, 3)}
	bounds := CalculatePossibleRanks(standings, games)

	// Without max-flow, worst rank for P would be 4 (all of 1,2,3 could reach 12).
	// With max-flow: each needs 4 more pts = 2 wins, but only 3 total wins available.
	// At most 1 can get 2 wins (e.g., player 1 beats 2 and 3).
	// So worst rank for P should be 2.
	if bounds[0].WorstRank != 2 {
		t.Errorf("P worst rank: got %d, want 2", bounds[0].WorstRank)
	}
}

func TestMaxFlowWithPGames(t *testing.T) {
	// P (player 0) has 6 pts, 2 games remaining (vs players 1 and 2).
	// Player 1 has 4 pts, 1 game remaining (vs P).
	// Player 2 has 4 pts, 1 game remaining (vs P).
	//
	// Worst case for P: P loses both → P stays at 6 pts.
	// Player 1 beats P → 6 pts. Player 2 beats P → 6 pts.
	// All three at 6 pts, spread decides.
	// P has -∞ spread (has remaining games) → both 1 and 2 are above P.
	// Worst rank for P = 3.
	standings := []standingInfo{
		si(0, 6, 50, 2),
		si(1, 4, 0, 1),
		si(2, 4, -10, 1),
	}
	games := []unfinishedGame{uf(0, 1), uf(0, 2)}
	bounds := CalculatePossibleRanks(standings, games)

	if bounds[0].WorstRank != 3 {
		t.Errorf("P worst rank: got %d, want 3", bounds[0].WorstRank)
	}
	if bounds[0].BestRank != 1 {
		t.Errorf("P best rank: got %d, want 1", bounds[0].BestRank)
	}
}

func TestUnfinishedGameFromRow(t *testing.T) {
	g := UnfinishedGameFromRow(
		pgtype.Int4{Int32: 5, Valid: true},
		pgtype.Int4{Int32: 7, Valid: true},
	)
	if g.player0ID != 5 || g.player1ID != 7 {
		t.Errorf("got %v, want {5, 7}", g)
	}
}

func TestBestRankTightening(t *testing.T) {
	// P (player 0) has 0 pts, 0 games remaining.
	// Players 1,2,3,4 have 0 pts, 3 games remaining each.
	// Games: 1v2, 1v3, 1v4, 2v3, 2v4, 3v4 (round robin).
	// Total 6 games, 12 points to distribute.
	//
	// P's best = 0. One player can stay at 0 (losing all 3 games),
	// but that player has negative spread so P (spread 0) beats them.
	// At least 3 players must have > 0 pts. Best rank for P = 4.
	standings := []standingInfo{
		si(0, 0, 0, 0),
		si(1, 0, 0, 3),
		si(2, 0, 0, 3),
		si(3, 0, 0, 3),
		si(4, 0, 0, 3),
	}
	games := []unfinishedGame{uf(1, 2), uf(1, 3), uf(1, 4), uf(2, 3), uf(2, 4), uf(3, 4)}
	bounds := CalculatePossibleRanks(standings, games)

	if bounds[0].BestRank != 4 {
		t.Errorf("P best rank: got %d, want 4", bounds[0].BestRank)
	}
}

func TestSpreadTiebreakInBestRank(t *testing.T) {
	// P1: 15 pts, +80 spread, 0 games remaining
	// P2: 14 pts, +200 spread, 1 game remaining (vs P3)
	// P3: 14 pts, +100 spread, 1 game remaining (vs P2)
	//
	// If P2 wins: P2=16 > P1=15 > P3=14. P1 is 2nd.
	// If P3 wins: P3=16 > P1=15 > P2=14. P1 is 2nd.
	// If draw: P2=15/+200, P3=15/+100, P1=15/+80. P1 is 3rd.
	//
	// Best rank: 2. Worst rank: 3.
	standings := []standingInfo{
		si(1, 15, 80, 0),
		si(2, 14, 200, 1),
		si(3, 14, 100, 1),
	}
	games := []unfinishedGame{uf(2, 3)}
	bounds := CalculatePossibleRanks(standings, games)

	if bounds[0].BestRank != 2 {
		t.Errorf("P1 best rank: got %d, want 2", bounds[0].BestRank)
	}
	if bounds[0].WorstRank != 3 {
		t.Errorf("P1 worst rank: got %d, want 3", bounds[0].WorstRank)
	}
}

func TestSpreadTiebreakBelowP(t *testing.T) {
	// P1: 15 pts, +300 spread, 0 games left
	// P2: 14 pts, +200 spread, 1 game left (vs P3)
	// P3: 14 pts, +100 spread, 1 game left (vs P2)
	//
	// If P2 and P3 draw: both reach 15 pts with preserved spread
	// (+200 and +100), both below P1's +300. P1 is 1st.
	// If either wins: winner reaches 16 (above P1). P1 is 2nd.
	//
	// P2 and P3 have worse spread than P1 and can reach 15 via draws
	// (1 point each, spread preserved). The flow restricts per-game
	// capacity to 1 for these players, correctly finding bestRank=1.
	standings := []standingInfo{
		si(1, 15, 300, 0),
		si(2, 14, 200, 1),
		si(3, 14, 100, 1),
	}
	games := []unfinishedGame{uf(2, 3)}
	bounds := CalculatePossibleRanks(standings, games)

	if bounds[0].BestRank != 1 {
		t.Errorf("P1 best rank: got %d, want 1", bounds[0].BestRank)
	}
	if bounds[0].WorstRank != 2 {
		t.Errorf("P1 worst rank: got %d, want 2", bounds[0].WorstRank)
	}
}

func TestDrawsOnlyBestRank(t *testing.T) {
	// P1: 20 pts, +300 spread, 0 games
	// P2: 17 pts, +100 spread, 3 games (vs P3, P4, P5)
	// P3: 17 pts, +50 spread, 3 games (vs P2, P4, P5)
	// P4: 17 pts, +50 spread, 3 games (vs P2, P3, P5)
	// P5: 17 pts, +50 spread, 3 games (vs P2, P3, P4)
	//
	// P2-P5 can all draw all games: each reaches 20 with preserved spread,
	// all below P1's +300. P1 bestRank = 1.
	// If any wins: winner reaches 21+ (above P1). P1 worstRank = 2+.
	standings := []standingInfo{
		si(1, 20, 300, 0),
		si(2, 17, 100, 3),
		si(3, 17, 50, 3),
		si(4, 17, 50, 3),
		si(5, 17, 50, 3),
	}
	games := []unfinishedGame{uf(2, 3), uf(2, 4), uf(2, 5), uf(3, 4), uf(3, 5), uf(4, 5)}
	bounds := CalculatePossibleRanks(standings, games)

	if bounds[0].BestRank != 1 {
		t.Errorf("P1 best rank: got %d, want 1", bounds[0].BestRank)
	}
}

func TestDrawsOnlyBestRankInfeasible(t *testing.T) {
	// P1: 20 pts, +300 spread, 0 games
	// P2: 17 pts, +400 spread, 3 games (vs P3, P4, P5)
	// P3: 17 pts, +50 spread, 3 games (vs P2, P4, P5)
	// P4: 17 pts, +50 spread, 3 games (vs P2, P3, P5)
	// P5: 17 pts, +50 spread, 3 games (vs P2, P3, P4)
	//
	// P2 has spread +400 > P1's +300. P2 at 20 beats P1 on spread.
	// P2 must stay strictly below 20 (absorb ≤ 2 points from 3 games).
	// P3-P5 can draw all games (maxPerGame=1, spread preserved).
	// P2 needs maxBelow=2 (decremented from 3) with maxPerGame=2.
	standings := []standingInfo{
		si(1, 20, 300, 0),
		si(2, 17, 400, 3),
		si(3, 17, 50, 3),
		si(4, 17, 50, 3),
		si(5, 17, 50, 3),
	}
	games := []unfinishedGame{uf(2, 3), uf(2, 4), uf(2, 5), uf(3, 4), uf(3, 5), uf(4, 5)}
	bounds := CalculatePossibleRanks(standings, games)

	// Total absorb capacity (2+3+3+3=11) < total within-set game points
	// (6 games × 2 = 12). Not all can stay below P1. bestRank = 2.
	if bounds[0].BestRank != 2 {
		t.Errorf("P1 best rank: got %d, want 2", bounds[0].BestRank)
	}
}

func TestAllGamesCompleteSamePointsDifferentSpread(t *testing.T) {
	// All games finished. A and B have the same points but A has better spread.
	// A is definitively 1st, B is definitively 2nd, C is definitively 3rd.
	standings := []standingInfo{
		si(1, 10, 100, 0),
		si(2, 10, 50, 0),
		si(3, 6, -50, 0),
	}
	bounds := CalculatePossibleRanks(standings, nil)

	if bounds[0].BestRank != 1 || bounds[0].WorstRank != 1 {
		t.Errorf("A: got %d-%d, want 1-1", bounds[0].BestRank, bounds[0].WorstRank)
	}
	if bounds[1].BestRank != 2 || bounds[1].WorstRank != 2 {
		t.Errorf("B: got %d-%d, want 2-2", bounds[1].BestRank, bounds[1].WorstRank)
	}
	if bounds[2].BestRank != 3 || bounds[2].WorstRank != 3 {
		t.Errorf("C: got %d-%d, want 3-3", bounds[2].BestRank, bounds[2].WorstRank)
	}
}

func TestEqualPointsAndSpread(t *testing.T) {
	// Two players with identical points and spread, both finished.
	// Username tiebreak is arbitrary, so both could be in either position.
	// Each should show a range of 1-2, not a fixed rank.
	standings := []standingInfo{
		si(1, 10, 50, 0),
		si(2, 10, 50, 0),
	}
	bounds := CalculatePossibleRanks(standings, nil)

	// Player 0: could be 1st or 2nd
	if bounds[0].BestRank != 1 {
		t.Errorf("P0 best rank: got %d, want 1", bounds[0].BestRank)
	}
	if bounds[0].WorstRank != 2 {
		t.Errorf("P0 worst rank: got %d, want 2", bounds[0].WorstRank)
	}
	// Player 1: same range
	if bounds[1].BestRank != 1 {
		t.Errorf("P1 best rank: got %d, want 1", bounds[1].BestRank)
	}
	if bounds[1].WorstRank != 2 {
		t.Errorf("P1 worst rank: got %d, want 2", bounds[1].WorstRank)
	}
}

func TestBestRankIgnoresGuaranteedBelow(t *testing.T) {
	// Reproduces the liwords Collins S11 Div1 bug: a finished player with
	// absorb capacity 0 and no remaining games was being removed during
	// feasibility iteration and counted as "forced above P", inflating P's
	// bestRank.
	//
	// P (player 0): 4 pts, 0 spread, 0 games remaining.
	// Q1 (1): 2 pts, +10 spread, 2 games remaining (vs Q2, vs Q3).
	// Q2 (2): 2 pts, +10 spread, 1 game remaining (vs Q1).
	// Q3 (3): 2 pts, +10 spread, 1 game remaining (vs Q1).
	// Fin (4): 0 pts, 0 spread, 0 games remaining — guaranteed below P.
	//
	// B = P.points = 4. Q1/Q2/Q3 each have spread > P's, so maxBelow=1 after
	// decrement, absorb capped at 1. Sum absorb = 3. Within-set games = 2
	// (Q1-Q2, Q1-Q3) × 2 = 4. 4 > 3 → infeasible by 1 pt, so exactly 1 of
	// Q1/Q2/Q3 must exceed cap. Truth: bestRank = 2.
	//
	// Before fix: Fin entered belowCandidates with absorb=0, got removed as
	// smallest-absorb without improving feasibility, then a Q was removed.
	// Final len=2, both Fin and one Q counted as forced above → bestRank=3.
	standings := []standingInfo{
		si(0, 4, 0, 0),
		si(1, 2, 10, 2),
		si(2, 2, 10, 1),
		si(3, 2, 10, 1),
		si(4, 0, 0, 0),
	}
	games := []unfinishedGame{uf(1, 2), uf(1, 3)}
	bounds := CalculatePossibleRanks(standings, games)

	if bounds[0].BestRank != 2 {
		t.Errorf("P best rank: got %d, want 2", bounds[0].BestRank)
	}
}

func TestEqualPointsAndSpreadThreePlayers(t *testing.T) {
	// Three players, all finished with same points and spread.
	// Each should show range 1-3.
	standings := []standingInfo{
		si(1, 10, 50, 0),
		si(2, 10, 50, 0),
		si(3, 10, 50, 0),
	}
	bounds := CalculatePossibleRanks(standings, nil)

	for i := 0; i < 3; i++ {
		if bounds[i].BestRank != 1 {
			t.Errorf("P%d best rank: got %d, want 1", i, bounds[i].BestRank)
		}
		if bounds[i].WorstRank != 3 {
			t.Errorf("P%d worst rank: got %d, want 3", i, bounds[i].WorstRank)
		}
	}
}
