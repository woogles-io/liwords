package tournament_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/jobs"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/tournament"
	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// heldPairer runs COP in process, calling before (if set) each time first:
// a change made there is made while COP runs, with no lock held.
type heldPairer struct {
	mu     sync.Mutex
	calls  int
	before func(call int)
}

func (p *heldPairer) PairCOP(ctx context.Context, req *ipc.PairRequest) (*ipc.PairResponse, error) {
	p.mu.Lock()
	p.calls++
	call := p.calls
	p.mu.Unlock()
	if p.before != nil {
		p.before(call)
	}
	return tournament.LocalCOPPairer{}.PairCOP(ctx, req)
}

// copControls are four rounds: King of the Hill, then COP.
func copControls() []*ipc.RoundControl {
	rcs := makeRoundControls()[:4]
	rcs[0].PairingMethod = ipc.PairingMethod_KING_OF_THE_HILL
	for _, rc := range rcs[1:] {
		rc.PairingMethod = ipc.PairingMethod_PAIRING_METHOD_COP
		rc.DivisionSims = 1000
		rc.ControlLossSims = 1000
		rc.PlacePrizes = 1
		rc.ControlLossThreshold = 0.25
		rc.GibsonSpreads = []int32{250}
		rc.HopefulnessThresholds = []float64{0.1}
		// Control-loss sims run up to 6s on a division this small.
		rc.ControlLossActivationRound = 100
	}
	return rcs
}

// setUpCOPDivision starts a division whose round 2 is COP, with deferral on,
// and plays round 1, which leaves round 2 for a pair_round job.
func setUpCOPDivision(t *testing.T) (context.Context, *stores.Stores, *entity.Tournament, chan *entity.EventWrapper) {
	is := is.New(t)
	tournament.DeferCOPPairing(true)
	t.Cleanup(func() { tournament.DeferCOPPairing(false) })
	ctx, stores, ty, eventChan := setUpServiceDivision(t, map[string]int32{
		"Will": 4000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100,
	}, false, copControls())
	for _, game := range [][2]string{{"Will", "Josh"}, {"Conrad", "Jesse"}} {
		is.NoErr(setServiceResult(ctx, stores, ty, 0, game[0], game[1]))
	}
	return ctx, stores, ty, eventChan
}

func pairRoundJob(ty *entity.Tournament, round string) jobs.Job {
	return jobs.Job{Kind: tournament.PairRoundJobKind, Key: ty.UUID + "/" + divOneName + "/" + round, Attempt: 1}
}

// The result that makes a COP round pairable leaves it unpaired, with a
// pair_round job written with it; the job pairs it and sends the pairings,
// and running it again does nothing.
func TestCOPRoundPairedByJob(t *testing.T) {
	is := is.New(t)
	ctx, stores, ty, eventChan := setUpCOPDivision(t)
	tstore := stores.TournamentStore

	dm := reload(ctx, is, tstore, ty.UUID).Divisions[divOneName].DivisionManager
	is.True(!dm.IsRoundPaired(1))
	job := pairRoundJob(ty, "1")
	row, err := stores.Queries.GetJob(ctx, models.GetJobParams{Kind: job.Kind, Key: job.Key})
	is.NoErr(err)
	is.True(!row.DoneAt.Valid)
	drainEvents(eventChan)

	pairer := &heldPairer{}
	run := tournament.PairRoundJob(tstore, pairer)
	is.NoErr(run(ctx, job))
	is.Equal(pairer.calls, 1)
	is.True(reload(ctx, is, tstore, ty.UUID).Divisions[divOneName].DivisionManager.IsRoundPaired(1))
	is.True(pairingsSentForRound(drainEvents(eventChan), 1))

	is.NoErr(run(ctx, job))
	is.Equal(pairer.calls, 1) // the round is paired: nothing to do
}

// A result amended while COP runs changes the request: the first answer is
// thrown away and COP runs again on the new one. The amendment commits
// while COP runs, so no lock is held then.
func TestCOPRequestChangedWhileRunning(t *testing.T) {
	is := is.New(t)
	ctx, stores, ty, _ := setUpCOPDivision(t)
	tstore := stores.TournamentStore

	pairer := &heldPairer{}
	pairer.before = func(call int) {
		if call != 1 {
			return
		}
		amendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		is.NoErr(tournament.SetResult(amendCtx, tstore, stores.UserStore, ty.UUID, divOneName,
			"Will", "Josh", 500, 350, ipc.TournamentGameResult_WIN, ipc.TournamentGameResult_LOSS,
			ipc.GameEndReason_STANDARD, 0, 0, true, nil, stores.Queries))
	}
	is.NoErr(tournament.PairRoundJob(tstore, pairer)(ctx, pairRoundJob(ty, "1")))
	is.Equal(pairer.calls, 2)
	is.True(reload(ctx, is, tstore, ty.UUID).Divisions[divOneName].DivisionManager.IsRoundPaired(1))
}

// A round a director pairs by hand while COP runs keeps the director's
// pairings.
func TestCOPRoundPairedByHandWhileRunning(t *testing.T) {
	is := is.New(t)
	ctx, stores, ty, _ := setUpCOPDivision(t)
	tstore := stores.TournamentStore

	pairer := &heldPairer{}
	pairer.before = func(int) {
		_, err := tournament.SetPairings(ctx, tstore, ty.UUID, divOneName, []*pb.TournamentPairingRequest{
			{PlayerOneId: "Will:Will", PlayerTwoId: "Jesse:Jesse", Round: 1},
			{PlayerOneId: "Josh:Josh", PlayerTwoId: "Conrad:Conrad", Round: 1},
		})
		is.NoErr(err)
	}
	is.NoErr(tournament.PairRoundJob(tstore, pairer)(ctx, pairRoundJob(ty, "1")))
	is.Equal(pairer.calls, 1)

	dm := reload(ctx, is, tstore, ty.UUID).Divisions[divOneName].DivisionManager
	will, err := dm.GetPairing(fullID(dm, "Will"), 1)
	is.NoErr(err)
	persons := dm.GetPlayers().Persons
	ids := []string{persons[will.Players[0]].Id, persons[will.Players[1]].Id}
	is.True(ids[0] == fullID(dm, "Jesse") || ids[1] == fullID(dm, "Jesse"))
}

// When the job gives up, the directors are told the round wasn't paired.
func TestCOPGaveUpTellsDirectors(t *testing.T) {
	is := is.New(t)
	ctx, stores, ty, eventChan := setUpCOPDivision(t)
	drainEvents(eventChan)
	tournament.PairRoundGaveUp(stores.TournamentStore)(ctx, pairRoundJob(ty, "1"), errors.New("no"))
	evts := drainEvents(eventChan)
	is.True(len(evts) > 0)
	is.Equal(evts[0].Type, ipc.MessageType_ERROR_MESSAGE)
}

// fullID is a player's ID in the division, from their username.
func fullID(dm entity.DivisionManager, username string) string {
	for _, p := range dm.GetPlayers().Persons {
		if p.Id == username || len(p.Id) > len(username) && p.Id[len(p.Id)-len(username)-1:] == ":"+username {
			return p.Id
		}
	}
	return username
}
