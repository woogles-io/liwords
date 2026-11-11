package tournament_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/pair/cop"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/tournament"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Eight players; rounds 1 and 2 king of the hill, rounds 3 and 4 COP. The
// last result of round 2 pairs round 3 with COP.
func setUpCOPDivision(t *testing.T) (context.Context, *stores.Stores, string) {
	rcs := makeRoundControls()[:4]
	for i, rc := range rcs {
		rc.PairingMethod = ipc.PairingMethod_KING_OF_THE_HILL
		if i >= 2 {
			rc.PairingMethod = ipc.PairingMethod_PAIRING_METHOD_COP
			rc.DivisionSims = 1000
			rc.ControlLossSims = 1000
			rc.PlacePrizes = 1
			rc.ControlLossThreshold = 0.25
			rc.GibsonSpreads = []int32{250}
			rc.HopefulnessThresholds = []float64{0.1}
		}
	}
	ctx, st, ty, _ := setUpServiceDivision(t, map[string]int32{
		"Will": 4000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100,
		"Kieran": 2000, "Vince": 1900, "Jennifer": 1800, "Guy": 1700,
	}, false, rcs)
	return ctx, st, ty.UUID
}

// roundGames returns the round's games as pairs of player IDs.
func roundGames(is *is.I, dm entity.DivisionManager, round int) [][2]string {
	games := [][2]string{}
	seen := map[string]bool{}
	persons := dm.GetPlayers().Persons
	for _, p := range persons {
		if seen[p.Id] {
			continue
		}
		pairing, err := dm.GetPairing(p.Id, round)
		is.NoErr(err)
		var ids [2]string
		for i, idx := range pairing.Players {
			ids[i] = persons[idx].Id
			seen[ids[i]] = true
		}
		games = append(games, ids)
	}
	return games
}

func uuidOf(playerID string) string {
	u, _, _ := strings.Cut(playerID, ":")
	return u
}

func submit(ctx context.Context, is *is.I, st *stores.Stores, tid string, round int, g [2]string,
	s1, s2 int, amend bool) {
	r1, r2 := ipc.TournamentGameResult_WIN, ipc.TournamentGameResult_LOSS
	if s2 > s1 {
		r1, r2 = r2, r1
	}
	is.NoErr(tournament.SetResult(ctx, st.TournamentStore, st.UserStore, tid, divOneName,
		uuidOf(g[0]), uuidOf(g[1]), s1, s2, r1, r2, ipc.GameEndReason_STANDARD, round, 0, amend, nil, st.Queries))
}

// playUpToLastGame plays rounds 1 and 2, all but round 2's last game, and
// returns that game and round 1's first one.
func playUpToLastGame(ctx context.Context, is *is.I, st *stores.Stores, tid string) (last, first [2]string) {
	dm := reload(ctx, is, st.TournamentStore, tid).Divisions[divOneName].DivisionManager
	r1 := roundGames(is, dm, 0)
	for i, g := range r1 {
		submit(ctx, is, st, tid, 0, g, 400+10*i, 300, false)
	}
	is.NoErr(tournament.StartRoundCountdown(ctx, st.TournamentStore, tid, divOneName, 1))
	dm = reload(ctx, is, st.TournamentStore, tid).Divisions[divOneName].DivisionManager
	r2 := roundGames(is, dm, 1)
	for i, g := range r2[:len(r2)-1] {
		submit(ctx, is, st, tid, 1, g, 380+10*i, 350, false)
	}
	return r2[len(r2)-1], r1[0]
}

// opponents maps each player to their opponent in the round.
func opponents(is *is.I, dm entity.DivisionManager, round int) map[string]string {
	m := map[string]string{}
	for _, g := range roundGames(is, dm, round) {
		m[g[0]], m[g[1]] = g[1], g[0]
	}
	return m
}

// opponentsFrom maps each player to their opponent in COP's answer.
func opponentsFrom(dm entity.DivisionManager, resp *ipc.PairResponse) map[string]string {
	persons := dm.GetPlayers().Persons
	m := map[string]string{}
	for i, o := range resp.Pairings {
		m[persons[i].Id] = persons[o].Id
	}
	return m
}

// bounded is ctx with a deadline, for a change made while COP runs: if COP
// ran under the lock, that change would wait for it forever.
func bounded(t *testing.T, ctx context.Context) context.Context {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// lockFree reports whether the tournament can be locked right now.
func lockFree(ts tournament.TournamentStore, tid string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, lk, err := ts.LockForChange(ctx, tid)
	if err != nil {
		return false
	}
	lk.Release()
	return true
}

// The result that completes a round pairs the next with COP while the
// tournament is unlocked, and pairs it exactly as COP inside the change
// would have.
func TestCOPRunsOutsideTheLock(t *testing.T) {
	is := is.New(t)
	ctx, st, tid := setUpCOPDivision(t)
	last, _ := playUpToLastGame(ctx, is, st, tid)

	// What pairing inside the change gives: the same result, applied to a
	// copy, with COP run directly.
	copy := reload(ctx, is, st.TournamentStore, tid)
	cd := copy.Divisions[divOneName].DivisionManager
	_, err := cd.SubmitResult(1, last[0], last[1], 420, 350, ipc.TournamentGameResult_WIN,
		ipc.TournamentGameResult_LOSS, ipc.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)
	want := opponents(is, cd, 2)

	calls, unlocked := 0, 0
	defer tournament.SetCOPRunForTest(func(req *ipc.PairRequest) *ipc.PairResponse {
		calls++
		if lockFree(st.TournamentStore, tid) {
			unlocked++
		}
		return cop.COPPair(req)
	})()
	submit(ctx, is, st, tid, 1, last, 420, 350, false)

	is.Equal(calls, 1)    // COP ran once
	is.Equal(unlocked, 1) // with the tournament unlocked
	dm := reload(ctx, is, st.TournamentStore, tid).Divisions[divOneName].DivisionManager
	is.Equal(opponents(is, dm, 2), want)
	complete, err := dm.IsRoundComplete(1)
	is.NoErr(err)
	is.True(complete) // and the result was kept
}

// A change that lands while COP runs makes its answer stale: the result is
// applied again, COP runs for the new state, and the round is paired from
// that answer, with the other change kept.
func TestCOPRunsAgainAfterAChange(t *testing.T) {
	is := is.New(t)
	ctx, st, tid := setUpCOPDivision(t)
	last, first := playUpToLastGame(ctx, is, st, tid)

	var answers []*ipc.PairResponse
	defer tournament.SetCOPRunForTest(func(req *ipc.PairRequest) *ipc.PairResponse {
		if len(answers) == 0 {
			// A director corrects a round 1 score meanwhile.
			submit(bounded(t, ctx), is, st, tid, 0, first, 455, 300, true)
		}
		resp := cop.COPPair(req)
		answers = append(answers, resp)
		return resp
	})()
	submit(ctx, is, st, tid, 1, last, 420, 350, false)

	is.Equal(len(answers), 2) // the first answer was for the old scores
	ty := reload(ctx, is, st.TournamentStore, tid)
	dm := ty.Divisions[divOneName].DivisionManager
	is.Equal(opponents(is, dm, 2), opponentsFrom(dm, answers[1]))
	pairing, err := dm.GetPairing(first[0], 0)
	is.NoErr(err)
	is.Equal(pairing.Games[0].Scores[0], int32(455)) // the correction was kept
	complete, err := dm.IsRoundComplete(1)
	is.NoErr(err)
	is.True(complete)
}

// A tournament that keeps changing while COP runs is paired anyway: the
// last attempt runs COP inside the change, as before.
func TestCOPFallsBackToRunningUnderTheLock(t *testing.T) {
	is := is.New(t)
	ctx, st, tid := setUpCOPDivision(t)
	last, first := playUpToLastGame(ctx, is, st, tid)

	var answers []*ipc.PairResponse
	var unlocked []bool
	defer tournament.SetCOPRunForTest(func(req *ipc.PairRequest) *ipc.PairResponse {
		free := lockFree(st.TournamentStore, tid)
		unlocked = append(unlocked, free)
		if free && len(answers) < 2 {
			submit(bounded(t, ctx), is, st, tid, 0, first, 460+len(answers), 300, true)
		}
		resp := cop.COPPair(req)
		answers = append(answers, resp)
		return resp
	})()
	submit(ctx, is, st, tid, 1, last, 420, 350, false)

	is.Equal(unlocked, []bool{true, true, false}) // the third run held the lock
	dm := reload(ctx, is, st.TournamentStore, tid).Divisions[divOneName].DivisionManager
	is.Equal(opponents(is, dm, 2), opponentsFrom(dm, answers[2]))
	pairing, err := dm.GetPairing(first[0], 0)
	is.NoErr(err)
	is.Equal(pairing.Games[0].Scores[0], int32(461))
}

// A director pairing a COP round doesn't hold the tournament while COP runs
// either.
func TestDirectorCOPPairingOutsideTheLock(t *testing.T) {
	is := is.New(t)
	ctx, st, tid := setUpCOPDivision(t)
	last, _ := playUpToLastGame(ctx, is, st, tid)
	submit(ctx, is, st, tid, 1, last, 420, 350, false) // pairs round 3

	calls, unlocked := 0, 0
	defer tournament.SetCOPRunForTest(func(req *ipc.PairRequest) *ipc.PairResponse {
		calls++
		if lockFree(st.TournamentStore, tid) {
			unlocked++
		}
		return cop.COPPair(req)
	})()
	_, err := tournament.PairRound(ctx, st.TournamentStore, tid, divOneName, 2, false)
	is.NoErr(err)
	is.Equal(calls, 1)
	is.Equal(unlocked, 1)
	dm := reload(ctx, is, st.TournamentStore, tid).Divisions[divOneName].DivisionManager
	is.True(dm.IsRoundPaired(2))
}

// A change that panics while it's applied doesn't leave the tournament
// locked: the bus recovers panics in its handlers and carries on.
func TestPanicInChangeReleasesTheLock(t *testing.T) {
	is := is.New(t)
	ctx, st, tid := setUpCOPDivision(t)

	func() {
		defer func() { is.True(recover() != nil) }()
		_, lk, _ := tournament.LockForChangeWithCOPForTest(ctx, st.TournamentStore, tid,
			func(*entity.Tournament) error { panic("a bug") })
		if lk != nil {
			lk.Release()
		}
	}()
	is.True(lockFree(st.TournamentStore, tid))
}
