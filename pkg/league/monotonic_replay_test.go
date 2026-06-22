package league

// Rank-bounds monotonicity + final-result reproducer.
//
// Replays real league games against the production rank-bounds code and checks
// that the displayed possible-rank range only ever tightens.
//
// Export the full schedule (completed games with results + still-unplayed
// pairings) in one snapshot via LEAGUE_REPLAY_CSV:
//
//	\copy (
//	  SELECT gp.league_season_id, reg.division_id, gp.game_uuid,
//	         gp.player_id AS p0_id, gp.opponent_id AS p1_id,
//	         gp.score AS p0_score, gp.opponent_score AS p1_score,
//	         gp.won AS p0_won, gp.updated_at AS completed_at, true AS played
//	    FROM game_players gp
//	    JOIN league_registrations reg
//	      ON reg.user_id = gp.player_id AND reg.season_id = gp.league_season_id
//	   WHERE gp.league_season_id IS NOT NULL AND gp.player_index = 0
//	     AND gp.game_end_reason NOT IN (0,5,7)
//	  UNION ALL
//	  SELECT g.season_id, g.league_division_id, g.uuid,
//	         g.player0_id, g.player1_id,
//	         NULL::int, NULL::int, NULL::boolean, NULL::timestamptz, false
//	    FROM games g
//	   WHERE g.league_id IS NOT NULL AND g.game_end_reason = 0
//	   ORDER BY league_season_id, division_id, played DESC, completed_at, game_uuid
//	) TO 'league-all.csv' CSV HEADER;
//
// One transaction snapshot, so played and unplayed are disjoint and consistent.
// League games are all created at season start, so played ∪ unplayed is the
// fixed schedule. (A 9-column played-only dump without the `played` column also
// works; its in-progress divisions are then skipped.)
//
// For each (season, division) we walk the played games in completion order. At
// each snapshot the standings come from the played prefix and BOTH
// gamesRemaining and the unfinished list come from (unplayed ∪ played-suffix),
// so the inputs are internally consistent regardless of expectedGames /
// force-finish quirks, and correct for >15-player divisions where the schedule
// is a capped subset (not full round robin).
//
// Two invariants are checked:
//
//  1. Monotonicity: the displayed range only ever tightens. bestRank
//     non-decreasing, worstRank non-increasing. A decrease in best (e.g. "7
//     then 6") or increase in worst breaks the promise the feature makes.
//  2. Final result: a fully-completed division's bounds must equal the real
//     final standings (tie-aware: best = 1 + strictly-above, worst = best +
//     equally-placed).
//
// Skipped unless LEAGUE_REPLAY_CSV is set, so normal CI does not need (and the
// repo never commits) the prod data file.

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type divKey struct{ season, div string }

// worst single CalculatePossibleRanks call seen during a replay (perf guard:
// the production budget is the worst per-division-snapshot, ~29ms).
var (
	maxCalcDur           time.Duration
	maxCalcN, maxCalcRem int
)

type replayGame struct {
	p0, p1 int32
	s0, s1 int
	result int // +1 = p0 won, -1 = p1 won, 0 = draw
}

type replayReport struct {
	divisions, skipped, snapshots int
	monoViol, finalMiss           int
	// Violations bucketed by the regime transition that exposed them:
	// MM = maxflow->maxflow, MB = maxflow->brute boundary, BB = brute->brute.
	// brute is exact, so a nonzero BB would mean the bug is not maxflow-only.
	viaMM, viaMB, viaBB                    int
	cleanDivs, anomalyDivs                 int
	cleanViol, anomalyViol                 int
	monoSamples, finalSamples, jointWinner []string
}

func TestRankBoundsMonotonicReplay(t *testing.T) {
	path := os.Getenv("LEAGUE_REPLAY_CSV")
	if path == "" {
		t.Skip("set LEAGUE_REPLAY_CSV to the exported league-all.csv to run")
	}

	played, unplayed, order := loadSchedule(t, path)

	var rep replayReport
	for _, k := range order {
		replayDivision(&rep, k.season, k.div, played[k], unplayed[k])
	}

	t.Logf("replayed %d division-seasons (%d skipped: incomplete schedule), %d snapshots",
		rep.divisions, rep.skipped, rep.snapshots)
	t.Logf("worst single CalculatePossibleRanks: %v (n=%d, remaining games=%d)",
		maxCalcDur, maxCalcN, maxCalcRem)
	t.Logf("monotonicity violations: %d total (MM=%d maxflow->maxflow, MB=%d maxflow->brute boundary, BB=%d brute->brute)",
		rep.monoViol, rep.viaMM, rep.viaMB, rep.viaBB)
	t.Logf("final-result mismatches: %d", rep.finalMiss)
	t.Logf("violations by division kind: clean (no timeout-sign game)=%d in %d divs; anomaly=%d in %d divs",
		rep.cleanViol, rep.cleanDivs, rep.anomalyViol, rep.anomalyDivs)
	if len(rep.jointWinner) > 0 {
		t.Logf("joint winners (tie for 1st): %d division(s):\n%s",
			len(rep.jointWinner), strings.Join(rep.jointWinner, "\n"))
	}

	if rep.monoViol > 0 {
		t.Errorf("rank-bounds widened %d time(s); first %d:\n%s",
			rep.monoViol, len(rep.monoSamples), strings.Join(rep.monoSamples, "\n"))
	}
	if rep.finalMiss > 0 {
		t.Errorf("final bounds disagreed with actual standings %d time(s); first %d:\n%s",
			rep.finalMiss, len(rep.finalSamples), strings.Join(rep.finalSamples, "\n"))
	}
}

// loadSchedule reads the unified dump, grouping each division's played games
// (in completion order) and still-unplayed pairings. A 9-column played-only
// dump (no `played` column) is treated as all played.
func loadSchedule(t *testing.T, path string) (map[divKey][]replayGame, map[divKey][]unfinishedGame, []divKey) {
	rows := readCSV(t, path)
	played := map[divKey][]replayGame{}
	unplayed := map[divKey][]unfinishedGame{}
	var order []divKey
	seen := map[divKey]bool{}
	for i, row := range rows {
		if i == 0 || len(row) < 9 {
			continue // header / short row
		}
		div := row[1]
		if div == "" {
			continue // registration without a division assignment
		}
		k := divKey{row[0], div}
		if !seen[k] {
			seen[k] = true
			order = append(order, k)
		}

		isPlayed := true
		if len(row) >= 10 {
			switch strings.ToLower(strings.TrimSpace(row[9])) {
			case "f", "false":
				isPlayed = false
			}
		}
		if isPlayed {
			res := 0
			switch strings.ToLower(strings.TrimSpace(row[7])) {
			case "t", "true":
				res = 1
			case "f", "false":
				res = -1
			}
			played[k] = append(played[k], replayGame{
				p0:     atoi32(row[3]),
				p1:     atoi32(row[4]),
				s0:     atoi(row[5]),
				s1:     atoi(row[6]),
				result: res,
			})
		} else {
			unplayed[k] = append(unplayed[k], unfinishedGame{
				player0ID: atoi32(row[3]),
				player1ID: atoi32(row[4]),
			})
		}
	}
	return played, unplayed, order
}

func replayDivision(rep *replayReport, season, div string, played []replayGame, unplayed []unfinishedGame) {
	// Distinct players, across played and unplayed games.
	var players []int32
	seen := map[int32]bool{}
	totalGP := map[int32]int{}
	add := func(p int32) {
		if !seen[p] {
			seen[p] = true
			players = append(players, p)
		}
		totalGP[p]++
	}
	for _, g := range played {
		add(g.p0)
		add(g.p1)
	}
	for _, u := range unplayed {
		add(u.player0ID)
		add(u.player1ID)
	}
	n := len(players)

	// Skip malformed / partially-dumped divisions: a valid full schedule has
	// every player at exactly the expected game count (played + unplayed).
	expected := CalculateExpectedGamesPerPlayer(n)
	for _, p := range players {
		if totalGP[p] != expected {
			rep.skipped++
			return
		}
	}
	rep.divisions++

	// A division has a "timeout-sign" anomaly if any completed game's winner
	// scored less than the loser (e.g. a leading player who timed out).
	hasAnomaly := false
	for _, g := range played {
		if (g.result == 1 && g.s0 < g.s1) || (g.result == -1 && g.s1 < g.s0) {
			hasAnomaly = true
			break
		}
	}
	if hasAnomaly {
		rep.anomalyDivs++
	} else {
		rep.cleanDivs++
	}

	type acc struct{ wins, draws, spread int }
	st := make(map[int32]*acc, n)
	playedCnt := make(map[int32]int, n)
	for _, p := range players {
		st[p] = &acc{}
	}

	prevBest := map[int32]int{}
	prevWorst := map[int32]int{}

	for k := 0; k <= len(played); k++ {
		// Standings reflect played[0:k]; remaining = unplayed ∪ played[k:].
		standings := make([]standingInfo, 0, n)
		for _, p := range players {
			a := st[p]
			standings = append(standings, standingInfo{
				userID:         p,
				points:         a.wins*2 + a.draws,
				spread:         a.spread,
				gamesRemaining: totalGP[p] - playedCnt[p],
			})
		}
		sort.SliceStable(standings, func(i, j int) bool {
			if standings[i].points != standings[j].points {
				return standings[i].points > standings[j].points
			}
			if standings[i].spread != standings[j].spread {
				return standings[i].spread > standings[j].spread
			}
			return standings[i].userID < standings[j].userID
		})

		unf := make([]unfinishedGame, 0, len(unplayed)+len(played)-k)
		unf = append(unf, unplayed...)
		for _, g := range played[k:] {
			unf = append(unf, unfinishedGame{player0ID: g.p0, player1ID: g.p1})
		}
		remCount := len(unf)

		t0 := time.Now()
		bounds := CalculatePossibleRanks(standings, unf)
		if d := time.Since(t0); d > maxCalcDur {
			maxCalcDur, maxCalcN, maxCalcRem = d, len(standings), remCount
		}
		rep.snapshots++
		inMaxflow := remCount > bruteForceThreshold

		for i := range standings {
			id := standings[i].userID
			b := bounds[i]
			if k > 0 && (b.BestRank < prevBest[id] || b.WorstRank > prevWorst[id]) {
				rep.monoViol++
				switch {
				case remCount > bruteForceThreshold:
					rep.viaMM++
				case remCount == bruteForceThreshold:
					rep.viaMB++
				default:
					rep.viaBB++
				}
				if hasAnomaly {
					rep.anomalyViol++
				} else {
					rep.cleanViol++
				}
				if len(rep.monoSamples) < 50 {
					clusters := buildBruteForceClusters(standings, toGamePairs(standings, unf))
					mc := maxClusterGames(clusters)
					pc := playerClusterGames(clusters, i) // wobbling player's OWN cluster
					rep.monoSamples = append(rep.monoSamples, fmt.Sprintf(
						"season=%s div=%s snap=%d/%d player=%d best %d->%d worst %d->%d (pcluster=%d maxcluster=%d rem=%d n=%d gr=%d %s)",
						season, div, k, len(played), id,
						prevBest[id], b.BestRank, prevWorst[id], b.WorstRank,
						pc, mc, remCount, n, standings[i].gamesRemaining,
						regimeLabel(inMaxflow)))
				}
			}
			prevBest[id] = b.BestRank
			prevWorst[id] = b.WorstRank
		}

		// Terminal snapshot of a fully-completed division: bounds must equal the
		// real final standings. (In-progress divisions still have unplayed games
		// here, so skip.)
		if k == len(played) && len(unplayed) == 0 {
			checkFinalResult(rep, season, div, standings, bounds)
		}

		// Apply played game k to advance to k+1.
		if k < len(played) {
			g := played[k]
			playedCnt[g.p0]++
			playedCnt[g.p1]++
			st[g.p0].spread += g.s0 - g.s1
			st[g.p1].spread += g.s1 - g.s0
			switch g.result {
			case 1:
				st[g.p0].wins++
			case -1:
				st[g.p1].wins++
			default:
				st[g.p0].draws++
				st[g.p1].draws++
			}
		}
	}
}

// checkFinalResult verifies that, with no games remaining, each player's bounds
// equal their actual competition rank range: best = 1 + (players strictly
// above), worst = best + (players tied on exactly points and spread).
func checkFinalResult(rep *replayReport, season, div string, standings []standingInfo, bounds []RankBounds) {
	firstPlace := 0
	for i := range standings {
		si := standings[i]
		strictlyAbove, tied := 0, 0
		for j := range standings {
			if i == j {
				continue
			}
			sj := standings[j]
			switch {
			case sj.points > si.points, sj.points == si.points && sj.spread > si.spread:
				strictlyAbove++
			case sj.points == si.points && sj.spread == si.spread:
				tied++
			}
		}
		if strictlyAbove == 0 {
			firstPlace++
		}
		expBest := 1 + strictlyAbove
		expWorst := expBest + tied
		if bounds[i].BestRank != expBest || bounds[i].WorstRank != expWorst {
			rep.finalMiss++
			if len(rep.finalSamples) < 50 {
				rep.finalSamples = append(rep.finalSamples, fmt.Sprintf(
					"season=%s div=%s player=%d got %d-%d want %d-%d (pts=%d spread=%d)",
					season, div, si.userID, bounds[i].BestRank, bounds[i].WorstRank,
					expBest, expWorst, si.points, si.spread))
			}
		}
	}
	if firstPlace > 1 && len(rep.jointWinner) < 50 {
		rep.jointWinner = append(rep.jointWinner,
			fmt.Sprintf("season=%s div=%s: %d-way tie for 1st", season, div, firstPlace))
	}
}

func readCSV(t *testing.T, path string) [][]string {
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// playerClusterGames returns the number of unfinished games in the cluster
// containing the given standings index, or -1 if the player is in no cluster
// (rank already fixed).
func playerClusterGames(clusters []bfCluster, playerIdx int) int {
	for _, c := range clusters {
		for _, m := range c.members {
			if m == playerIdx {
				return len(c.games)
			}
		}
	}
	return -1
}

func regimeLabel(maxflow bool) string {
	if maxflow {
		return "maxflow"
	}
	return "brute"
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func atoi32(s string) int32 {
	return int32(atoi(s))
}
