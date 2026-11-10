package tournament_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/tournament"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Ready states are stored outside the divisions: a click doesn't save the
// tournament, a page load shows who is waiting, and a cancelled game lets the
// players ready again.
func TestReadyForGame(t *testing.T) {
	is := is.New(t)
	roundControls := makeRoundControls()[:2]
	for _, rc := range roundControls {
		rc.PairingMethod = ipc.PairingMethod_KING_OF_THE_HILL
	}
	ctx, stores, ty, _ := setUpServiceDivision(t,
		map[string]int32{"Will": 4000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100}, false, roundControls)
	tstore := stores.TournamentStore
	dm := ty.Divisions[divOneName].DivisionManager

	readyStates := func(player string) []string {
		resp, err := tournament.GetXHRResponse(ctx, tstore, ty.UUID)
		is.NoErr(err)
		key, err := dm.PairingKey(player, 0)
		is.NoErr(err)
		return resp.Divisions[divOneName].PairingMap[key].ReadyStates
	}
	playerIdx := func(player string) int {
		p, err := dm.GetPairing(player, 0)
		is.NoErr(err)
		if dm.GetPlayers().Persons[p.Players[0]].Id == player {
			return 0
		}
		return 1
	}
	updatedAt := func() time.Time {
		row, err := stores.Queries.GetTournamentByUUID(ctx, pgtype.Text{String: ty.UUID, Valid: true})
		is.NoErr(err)
		return row.UpdatedAt.Time
	}
	before := updatedAt()

	// KOTH pairs Will with Josh.
	involved, bothReady, err := tournament.SetReadyForGame(ctx, tstore, ty, "Will:Will", "willconn", divOneName, 0, 0, false)
	is.NoErr(err)
	is.True(!bothReady)
	is.Equal(len(involved), 2)
	is.Equal(involved[playerIdx("Will:Will")], "Will:Will:willconn")
	is.Equal(involved[playerIdx("Josh:Josh")], "Josh:Josh:")

	want := []string{"", ""}
	want[playerIdx("Will:Will")] = "ready"
	is.Equal(readyStates("Will:Will"), want)
	// The stored tournament holds no ready states.
	stored, err := tstore.Get(ctx, ty.UUID)
	is.NoErr(err)
	p, err := stored.Divisions[divOneName].DivisionManager.GetPairing("Will:Will", 0)
	is.NoErr(err)
	is.Equal(p.ReadyStates, []string{"", ""})

	_, _, err = tournament.SetReadyForGame(ctx, tstore, ty, "Will:Will", "willconn", divOneName, 0, 0, false)
	is.True(hasWooglesCode(err, ipc.WooglesError_TOURNAMENT_ALREADY_READY))

	involved, bothReady, err = tournament.SetReadyForGame(ctx, tstore, ty, "Josh:Josh", "joshconn", divOneName, 0, 0, false)
	is.NoErr(err)
	is.True(bothReady)
	is.Equal(involved[playerIdx("Will:Will")], "Will:Will:willconn")
	is.Equal(involved[playerIdx("Josh:Josh")], "Josh:Josh:joshconn")
	// Once the game has started, nobody is shown as waiting.
	is.Equal(readyStates("Will:Will"), []string{"", ""})

	// Nothing above saved the tournament.
	is.Equal(updatedAt(), before)

	// The game was cancelled: both can ready again.
	is.NoErr(tournament.ClearReadyStates(ctx, tstore, ty, divOneName, "Josh:Josh", 0, 0))
	_, bothReady, err = tournament.SetReadyForGame(ctx, tstore, ty, "Will:Will", "willconn", divOneName, 0, 0, false)
	is.NoErr(err)
	is.True(!bothReady)

	// Unreadying is shown, and unreadying again changes nothing.
	involved, _, err = tournament.SetReadyForGame(ctx, tstore, ty, "Will:Will", "willconn", divOneName, 0, 0, true)
	is.NoErr(err)
	is.Equal(len(involved), 2)
	is.Equal(readyStates("Will:Will"), []string{"", ""})
	involved, _, err = tournament.SetReadyForGame(ctx, tstore, ty, "Will:Will", "willconn", divOneName, 0, 0, true)
	is.NoErr(err)
	is.Equal(len(involved), 0)
}

func hasWooglesCode(err error, code ipc.WooglesError) bool {
	return err != nil && strings.HasPrefix(err.Error(), entity.NewWooglesError(code).Error()+entity.WooglesErrorDelimiter)
}
