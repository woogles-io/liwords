package tournament

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// irlUserStore finds no users, as for an IRL tournament.
type irlUserStore struct{ user.Store }

func (irlUserStore) GetByUUID(context.Context, string) (*entity.User, error) {
	return nil, errors.New("not found")
}

func TestExportTSHForfeits(t *testing.T) {
	is := is.New(t)

	alice, bob, carol, dave := irlID("Alice"), irlID("Bob"), irlID("Carol"), irlID("Dave")
	players := makeTournamentPersons(map[string]int32{alice: 4000, bob: 3000, carol: 2000, dave: 1000})
	roundControls := defaultRoundControls(2)
	for _, rc := range roundControls {
		rc.PairingMethod = pb.PairingMethod_MANUAL
	}
	tc, err := compactNewClassicDivision(players, roundControls, false)
	is.NoErr(err)

	submit := func(round int, p1, p2 string, s1, s2 int, r1, r2 pb.TournamentGameResult) {
		_, err := tc.SubmitResult(round, p1, p2, s1, s2, r1, r2, pb.GameEndReason_STANDARD, false, 0, "")
		is.NoErr(err)
	}

	// Round 1: a normal game, and a forfeit between two real players that
	// the director entered as 0-0.
	_, err = tc.SetPairing(alice, bob, 0, pb.TournamentGameResult_NO_RESULT)
	is.NoErr(err)
	_, err = tc.SetPairing(carol, dave, 0, pb.TournamentGameResult_NO_RESULT)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	submit(0, alice, bob, 400, 300, pb.TournamentGameResult_WIN, pb.TournamentGameResult_LOSS)
	submit(0, carol, dave, 0, 0, pb.TournamentGameResult_FORFEIT_WIN, pb.TournamentGameResult_FORFEIT_LOSS)

	// Round 2: a voided game, a bye, and a self-paired forfeit loss.
	_, err = tc.SetPairing(alice, carol, 1, pb.TournamentGameResult_NO_RESULT)
	is.NoErr(err)
	_, err = tc.SetPairing(bob, bob, 1, pb.TournamentGameResult_BYE)
	is.NoErr(err)
	_, err = tc.SetPairing(dave, dave, 1, pb.TournamentGameResult_FORFEIT_LOSS)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	submit(1, alice, carol, 0, 0, pb.TournamentGameResult_VOID, pb.TournamentGameResult_VOID)

	tourney := &entity.Tournament{
		Divisions: map[string]*entity.TournamentDivision{
			"A": {ManagerType: entity.ClassicTournamentType, DivisionManager: tc},
		},
	}
	out, err := exportToTSH(context.Background(), tourney, irlUserStore{})
	is.NoErr(err)

	body := out[strings.Index(out, "#begin_file name=A.t\n")+len("#begin_file name=A.t\n") : strings.Index(out, "#end_file")]
	is.Equal(body, strings.Join([]string{
		"Alice 4000 2 0; 400 0",
		"Bob 3000 1 0; 300 50",
		"Carol 2000 0 0; 50 0",
		"Dave 1000 0 0; -50 -50",
		"",
	}, "\n"))
}
