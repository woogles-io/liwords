package tournament

import (
	"testing"

	"github.com/matryer/is"
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
