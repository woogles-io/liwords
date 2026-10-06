package tournament

import (
	"strings"
	"testing"

	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestClassicDivisionSetPlayerRatingBeforeStart(t *testing.T) {
	is := is.New(t)

	players := makeTournamentPersons(map[string]int32{"Will": 10000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100})
	tc, err := compactNewClassicDivision(players, defaultRoundControls(defaultRounds), false)
	is.NoErr(err)
	is.Equal(tc.PlayerIndexMap["Jesse"], int32(3))

	is.NoErr(tc.SetPlayerRating("Jesse", 5000))

	// Jesse is now the second seed.
	is.Equal(tc.Players.Persons[1].Id, "Jesse")
	is.Equal(tc.Players.Persons[1].Rating, int32(5000))
	is.Equal(tc.PlayerIndexMap["Jesse"], int32(1))
	is.Equal(tc.PlayerIndexMap["Josh"], int32(2))
	is.Equal(len(tc.Matrix[0]), 4)
}

func TestClassicDivisionSetPlayerRatingAfterStart(t *testing.T) {
	is := is.New(t)

	players := makeTournamentPersons(map[string]int32{"Will": 10000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100})
	tc, err := compactNewClassicDivision(players, defaultRoundControls(defaultRounds), true)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))

	pairingsBefore := tc.getPlayerPairings(0)

	is.NoErr(tc.SetPlayerRating("Jesse", 5000))

	// The rating changes but the seeds and pairings do not.
	is.Equal(tc.Players.Persons[3].Id, "Jesse")
	is.Equal(tc.Players.Persons[3].Rating, int32(5000))
	is.Equal(tc.PlayerIndexMap["Jesse"], int32(3))
	is.Equal(tc.getPlayerPairings(0), pairingsBefore)
}

func TestClassicDivisionSetPlayerRatingErrors(t *testing.T) {
	is := is.New(t)

	players := makeTournamentPersons(map[string]int32{"Will": 10000, "Josh": 3000})
	tc, err := compactNewClassicDivision(players, defaultRoundControls(defaultRounds), false)
	is.NoErr(err)

	is.True(tc.SetPlayerRating("Nobody", 1000) != nil)
	is.True(tc.SetPlayerRating("Will", -1) != nil)
	is.Equal(tc.Players.Persons[0].Rating, int32(10000))
}

func irlID(name string) string {
	return md5hash(name) + ":" + name
}

func TestClassicDivisionRenamePlayer(t *testing.T) {
	is := is.New(t)

	players := makeTournamentPersons(map[string]int32{
		irlID("Will"): 10000, irlID("Josh"): 3000, irlID("Conrad"): 2200, irlID("Jesse"): 2100})
	tc, err := compactNewClassicDivision(players, defaultRoundControls(defaultRounds), true)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))

	// Play round 1 so there are standings to carry over.
	for _, pairing := range tc.getPlayerPairings(0) {
		_, err = tc.SubmitResult(0, pairing[0], pairing[1], 400, 300,
			pb.TournamentGameResult_WIN, pb.TournamentGameResult_LOSS,
			pb.GameEndReason_STANDARD, false, 0, "")
		is.NoErr(err)
	}
	// EditPlayer keeps the token when renaming.
	oldID, newID := irlID("Jesse"), md5hash("Jesse")+":Jessica"
	idx := tc.PlayerIndexMap[oldID]
	standingsBefore, _, err := tc.GetStandings(0)
	is.NoErr(err)
	var recordBefore *pb.PlayerStanding
	for _, ps := range standingsBefore.Standings {
		if ps.PlayerId == oldID {
			recordBefore = ps
		}
	}
	is.True(recordBefore != nil)

	is.NoErr(tc.RenamePlayer(oldID, newID))

	is.Equal(tc.Players.Persons[idx].Id, newID)
	is.Equal(tc.PlayerIndexMap[newID], idx)
	_, stillThere := tc.PlayerIndexMap[oldID]
	is.True(!stillThere)
	for _, ps := range tc.Standings[0].Standings {
		is.True(ps.PlayerId != oldID)
		if ps.PlayerId == newID {
			is.Equal(ps.Wins, recordBefore.Wins)
			is.Equal(ps.Spread, recordBefore.Spread)
		}
	}

	// Renaming to a name already in the division is rejected.
	is.True(tc.RenamePlayer(newID, irlID("Will")) != nil)
	is.True(tc.RenamePlayer(oldID, irlID("Someone")) != nil)

	// IRL score entry identifies players by the token part of their ID
	// only, so the token from the printed scorecard still works, and the
	// md5 of the new name does not. (Round 2 has already started: this
	// division auto-starts rounds.)
	is.Equal(tc.CurrentRound, int32(1))
	opp, err := tc.opponentOf(newID, 1)
	is.NoErr(err)
	oppMD5 := strings.Split(opp, ":")[0]
	_, err = tc.SubmitResult(1, md5hash("Jessica"), oppMD5, 400, 300,
		pb.TournamentGameResult_WIN, pb.TournamentGameResult_LOSS,
		pb.GameEndReason_STANDARD, false, 0, "")
	is.True(err != nil)
	_, err = tc.SubmitResult(1, md5hash("Jesse"), oppMD5, 400, 300,
		pb.TournamentGameResult_WIN, pb.TournamentGameResult_LOSS,
		pb.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)
}
