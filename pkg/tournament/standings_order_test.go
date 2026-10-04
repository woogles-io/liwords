package tournament

import (
	"testing"

	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Issue #974: 3 wins and 2 draws must not rank above 4 wins and 1 loss
// when the 4-win player has the higher spread.
func TestClassicDivisionStandingsByPointsThenSpread(t *testing.T) {
	is := is.New(t)

	players := makeTournamentPersons(map[string]int32{
		"A": 6000, "B": 5000, "C": 4000, "D": 3000, "E": 2000, "F": 1000})
	roundControls := defaultRoundControls(5)
	for _, rc := range roundControls {
		rc.PairingMethod = pb.PairingMethod_MANUAL
	}
	tc, err := compactNewClassicDivision(players, roundControls, false)
	is.NoErr(err)

	type game struct {
		p1, p2 string
		s1, s2 int
	}
	rounds := [][]game{
		{{"A", "C", 400, 400}, {"B", "D", 500, 300}, {"E", "F", 300, 200}},
		{{"A", "D", 400, 400}, {"B", "C", 500, 300}, {"E", "F", 300, 200}},
		{{"A", "E", 401, 400}, {"B", "F", 500, 300}, {"C", "D", 300, 200}},
		{{"A", "F", 401, 400}, {"B", "E", 500, 300}, {"C", "D", 300, 200}},
		{{"A", "B", 401, 400}, {"C", "E", 300, 200}, {"D", "F", 300, 200}},
	}
	result := func(mine, theirs int) pb.TournamentGameResult {
		switch {
		case mine > theirs:
			return pb.TournamentGameResult_WIN
		case mine < theirs:
			return pb.TournamentGameResult_LOSS
		}
		return pb.TournamentGameResult_DRAW
	}
	for r, games := range rounds {
		for _, g := range games {
			_, err = tc.SetPairing(g.p1, g.p2, r, pb.TournamentGameResult_NO_RESULT)
			is.NoErr(err)
		}
		is.NoErr(tc.StartRound(true))
		for _, g := range games {
			_, err = tc.SubmitResult(r, g.p1, g.p2, g.s1, g.s2, result(g.s1, g.s2),
				result(g.s2, g.s1), pb.GameEndReason_STANDARD, false, 0, "")
			is.NoErr(err)
		}
	}

	standings, _, err := tc.GetStandings(4)
	is.NoErr(err)
	// B is 4-1 (+799) and A is 3-0-2 (+3): both have 8 points.
	is.NoErr(equalStandingsRecord(standings.Standings[0],
		&pb.PlayerStanding{PlayerId: "B", Wins: 4, Losses: 1, Spread: 799}))
	is.NoErr(equalStandingsRecord(standings.Standings[1],
		&pb.PlayerStanding{PlayerId: "A", Wins: 3, Draws: 2, Spread: 3}))
}
