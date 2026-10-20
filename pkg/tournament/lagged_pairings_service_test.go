package tournament_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/tournament"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

// AutoStart announces every round it opens, with and without lag.
func TestAutoStartIsAnnounced(t *testing.T) {
	for _, tc := range []struct {
		name          string
		lagThirdRound bool
		currentRound  int
		startedRounds []int32
	}{
		// Completing round 1 opens round 2.
		{name: "no lag", currentRound: 1, startedRounds: []int32{0, 1}},
		// Completing round 1 opens round 2 and the lagged round 3.
		{name: "lagged", lagThirdRound: true, currentRound: 2, startedRounds: []int32{0, 1, 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			ctx := context.Background()
			stores, cfg := recreateDB()
			defer stores.Disconnect()
			tstore, us := stores.TournamentStore, stores.UserStore

			eventChan := make(chan *entity.EventWrapper, 1000)
			tstore.SetTournamentEventChan(eventChan)

			directors := makeTournamentPersons(map[string]int32{"Kieran:Kieran": 0})
			ty, err := makeTournament(ctx, tstore, cfg, directors, "autostart")
			is.NoErr(err)
			is.NoErr(tournament.AddDivision(ctx, tstore, ty.UUID, divOneName))
			is.NoErr(tournament.AddPlayers(ctx, tstore, us, ty.UUID, divOneName,
				makeTournamentPersons(map[string]int32{"Will": 4000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100})))

			controls := makeControls()
			controls.AutoStart = true
			is.NoErr(tournament.SetDivisionControls(ctx, tstore, ty.UUID, divOneName, controls))

			roundControls := makeRoundControls()[:3]
			for _, rc := range roundControls {
				rc.PairingMethod = ipc.PairingMethod_KING_OF_THE_HILL
			}
			if tc.lagThirdRound {
				roundControls[2].PairingLag = 1
			}
			is.NoErr(tournament.SetRoundControls(ctx, tstore, ty.UUID, divOneName, roundControls))
			is.NoErr(tournament.StartRoundCountdown(ctx, tstore, ty.UUID, divOneName, 0))

			for _, game := range [][2]string{{"Will", "Josh"}, {"Conrad", "Jesse"}} {
				is.NoErr(tournament.SetResult(ctx, tstore, us, ty.UUID, divOneName,
					game[0], game[1], 500, 400,
					ipc.TournamentGameResult_WIN, ipc.TournamentGameResult_LOSS,
					ipc.GameEndReason_STANDARD, 0, 0, false, nil, stores.Queries))
			}

			is.Equal(ty.Divisions[divOneName].DivisionManager.GetCurrentRound(), tc.currentRound)
			startedRounds := []int32{}
			for len(eventChan) > 0 {
				evt := <-eventChan
				if evt.Type == ipc.MessageType_TOURNAMENT_ROUND_STARTED {
					startedRounds = append(startedRounds, evt.Event.(*ipc.TournamentRoundStarted).Round)
				}
			}
			is.Equal(startedRounds, tc.startedRounds)
		})
	}
}

func setUpLaggedServiceDivision(t *testing.T, autoStart bool, methods []ipc.PairingMethod, laggedRound int) (
	context.Context, *stores.Stores, *entity.Tournament, chan *entity.EventWrapper) {
	roundControls := makeRoundControls()[:len(methods)]
	for i, rc := range roundControls {
		rc.PairingMethod = methods[i]
	}
	roundControls[laggedRound].PairingLag = 1
	return setUpServiceDivision(t, map[string]int32{"Will": 4000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100},
		autoStart, roundControls)
}

// setUpServiceDivision creates a tournament with one division of the given
// players and round controls, and opens round 1.
func setUpServiceDivision(t *testing.T, players map[string]int32, autoStart bool, roundControls []*ipc.RoundControl) (
	context.Context, *stores.Stores, *entity.Tournament, chan *entity.EventWrapper) {
	is := is.New(t)
	ctx := context.Background()
	stores, cfg := recreateDB()
	t.Cleanup(stores.Disconnect)
	tstore, us := stores.TournamentStore, stores.UserStore

	eventChan := make(chan *entity.EventWrapper, 1000)
	tstore.SetTournamentEventChan(eventChan)

	directors := makeTournamentPersons(map[string]int32{"Kieran:Kieran": 0})
	ty, err := makeTournament(ctx, tstore, cfg, directors, "lagged")
	is.NoErr(err)
	is.NoErr(tournament.AddDivision(ctx, tstore, ty.UUID, divOneName))
	is.NoErr(tournament.AddPlayers(ctx, tstore, us, ty.UUID, divOneName, makeTournamentPersons(players)))
	controls := makeControls()
	controls.AutoStart = autoStart
	is.NoErr(tournament.SetDivisionControls(ctx, tstore, ty.UUID, divOneName, controls))
	is.NoErr(tournament.SetRoundControls(ctx, tstore, ty.UUID, divOneName, roundControls))
	is.NoErr(tournament.StartRoundCountdown(ctx, tstore, ty.UUID, divOneName, 0))
	return ctx, stores, ty, eventChan
}

// pairingsSentForRound reports whether a pairings message with pairings for
// the round was sent.
func pairingsSentForRound(evts []*entity.EventWrapper, round int32) bool {
	for _, evt := range evts {
		if evt.Type != ipc.MessageType_TOURNAMENT_DIVISION_PAIRINGS_MESSAGE {
			continue
		}
		for _, p := range evt.Event.(*ipc.DivisionPairingsResponse).DivisionPairings {
			if p.Round == round {
				return true
			}
		}
	}
	return false
}

// laggedRepeatLimitedSwissControls returns round controls for two players
// where the lagged third round can't be paired: it may not repeat a
// pairing, and the only possible pairing has been played already.
func laggedRepeatLimitedSwissControls(second ipc.PairingMethod) []*ipc.RoundControl {
	roundControls := makeRoundControls()[:3]
	roundControls[0].PairingMethod = ipc.PairingMethod_KING_OF_THE_HILL
	roundControls[1].PairingMethod = second
	roundControls[2].PairingMethod = ipc.PairingMethod_SWISS
	roundControls[2].PairingLag = 1
	roundControls[2].AllowOverMaxRepeats = false
	roundControls[2].MaxRepeats = 1
	return roundControls
}

// Pairing a lagged round after a director's pairings is follow-on work: if
// it fails, the director's pairings are still saved and sent out.
func TestLaggedPairingFailureKeepsDirectorPairings(t *testing.T) {
	players := map[string]int32{"Will": 2000, "Josh": 1000}

	t.Run("SetPairings", func(t *testing.T) {
		is := is.New(t)
		ctx, stores, ty, eventChan := setUpServiceDivision(t, players, false,
			laggedRepeatLimitedSwissControls(ipc.PairingMethod_MANUAL))
		dm := ty.Divisions[divOneName].DivisionManager
		is.NoErr(setServiceResult(ctx, stores, ty, 0, "Will", "Josh"))
		drainEvents(eventChan)

		warnings, err := tournament.SetPairings(ctx, stores.TournamentStore, ty.UUID, divOneName, []*pb.TournamentPairingRequest{
			{PlayerOneId: "Will:Will", PlayerTwoId: "Josh:Josh", Round: 1},
		})
		is.NoErr(err)
		is.Equal(warnings, []string{autoPairFailed(3)})
		is.True(dm.IsRoundPaired(1))
		is.True(!dm.IsRoundPaired(2))
		is.True(pairingsSentForRound(drainEvents(eventChan), 1))
	})

	t.Run("PairRound", func(t *testing.T) {
		is := is.New(t)
		ctx, stores, ty, eventChan := setUpServiceDivision(t, players, false,
			laggedRepeatLimitedSwissControls(ipc.PairingMethod_KING_OF_THE_HILL))
		dm := ty.Divisions[divOneName].DivisionManager
		// Completing round 1 pairs round 2, but not the lagged round 3,
		// and nobody is waiting on the result, so the directors are told.
		is.NoErr(setServiceResult(ctx, stores, ty, 0, "Will", "Josh"))
		is.True(dm.IsRoundPaired(1))
		is.True(!dm.IsRoundPaired(2))
		is.Equal(directorNotices(drainEvents(eventChan)), []string{"user.Kieran: " + autoPairFailed(3)})

		is.NoErr(tournament.DeletePairings(ctx, stores.TournamentStore, ty.UUID, divOneName, 1))
		drainEvents(eventChan)
		warnings, err := tournament.PairRound(ctx, stores.TournamentStore, ty.UUID, divOneName, 1, false)
		is.NoErr(err)
		is.Equal(warnings, []string{autoPairFailed(3)})
		is.True(dm.IsRoundPaired(1))
		is.True(!dm.IsRoundPaired(2))
		is.True(pairingsSentForRound(drainEvents(eventChan), 1))
	})
}

// autoPairFailed is the warning for follow-on pairing that failed to pair
// the 1-based round.
func autoPairFailed(round int) string {
	return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_AUTO_PAIR_FAILED, tournamentName, divOneName, strconv.Itoa(round)).Error()
}

// directorNotices returns the error messages sent to users, as
// "audience: message".
func directorNotices(evts []*entity.EventWrapper) []string {
	notices := []string{}
	for _, evt := range evts {
		if evt.Type == ipc.MessageType_ERROR_MESSAGE {
			notices = append(notices, strings.Join(evt.Audience(), ",")+": "+evt.Event.(*ipc.ErrorMessage).Message)
		}
	}
	return notices
}

// noWarnings checks that a director request succeeded without warnings.
func noWarnings(is *is.I) func([]string, error) {
	return func(warnings []string, err error) {
		is.Helper()
		is.NoErr(err)
		is.Equal(len(warnings), 0)
	}
}

func setServiceResult(ctx context.Context, stores *stores.Stores, ty *entity.Tournament, round int, p1, p2 string) error {
	return tournament.SetResult(ctx, stores.TournamentStore, stores.UserStore, ty.UUID, divOneName,
		p1, p2, 500, 400, ipc.TournamentGameResult_WIN, ipc.TournamentGameResult_LOSS,
		ipc.GameEndReason_STANDARD, round, 0, false, nil, stores.Queries)
}

func drainEvents(eventChan chan *entity.EventWrapper) []*entity.EventWrapper {
	evts := []*entity.EventWrapper{}
	for len(eventChan) > 0 {
		evts = append(evts, <-eventChan)
	}
	return evts
}

// A lagged round waiting for a MANUAL round before it to be paired is
// paired when the director pairs that round, and AutoStart that can't open
// the MANUAL round doesn't reject the result that completed the round
// before it: the result is still saved and sent out.
func TestLaggedRoundWaitingOnManualRound(t *testing.T) {
	is := is.New(t)
	ctx, stores, ty, eventChan := setUpLaggedServiceDivision(t, true, []ipc.PairingMethod{
		ipc.PairingMethod_KING_OF_THE_HILL, ipc.PairingMethod_MANUAL,
		ipc.PairingMethod_KING_OF_THE_HILL, ipc.PairingMethod_KING_OF_THE_HILL}, 2)
	dm := ty.Divisions[divOneName].DivisionManager

	is.NoErr(setServiceResult(ctx, stores, ty, 0, "Will", "Josh"))
	drainEvents(eventChan)
	is.NoErr(setServiceResult(ctx, stores, ty, 0, "Conrad", "Jesse"))
	sentPairings := false
	for _, evt := range drainEvents(eventChan) {
		if evt.Type == ipc.MessageType_TOURNAMENT_DIVISION_PAIRINGS_MESSAGE {
			sentPairings = true
		}
	}
	is.True(sentPairings)
	complete, err := dm.IsRoundComplete(0)
	is.NoErr(err)
	is.True(complete)
	is.Equal(dm.GetCurrentRound(), 0)
	is.True(!dm.IsRoundPaired(2))

	// Pairing the MANUAL round by hand pairs the lagged round after it.
	noWarnings(is)(tournament.SetPairings(ctx, stores.TournamentStore, ty.UUID, divOneName, []*pb.TournamentPairingRequest{
		{PlayerOneId: "Will:Will", PlayerTwoId: "Conrad:Conrad", Round: 1},
		{PlayerOneId: "Josh:Josh", PlayerTwoId: "Jesse:Jesse", Round: 1},
	}))
	is.True(dm.IsRoundPaired(2))

	// Once the director opens round 2, AutoStart opens round 3 alongside it.
	is.NoErr(tournament.StartRoundCountdown(ctx, stores.TournamentStore, ty.UUID, divOneName, 1))
	is.NoErr(setServiceResult(ctx, stores, ty, 1, "Will", "Conrad"))
	is.Equal(dm.GetCurrentRound(), 2)
}

// The service only pairs a waiting lagged round when the round before it
// becomes fully paired, so pairings a director deleted from the lagged
// round itself stay deleted when the round before it is paired again.
func TestLaggedRoundDeletedPairingsStayDeleted(t *testing.T) {
	is := is.New(t)
	ctx, stores, ty, _ := setUpLaggedServiceDivision(t, false, []ipc.PairingMethod{
		ipc.PairingMethod_KING_OF_THE_HILL, ipc.PairingMethod_KING_OF_THE_HILL,
		ipc.PairingMethod_KING_OF_THE_HILL}, 2)
	dm := ty.Divisions[divOneName].DivisionManager

	is.NoErr(setServiceResult(ctx, stores, ty, 0, "Will", "Josh"))
	is.NoErr(setServiceResult(ctx, stores, ty, 0, "Conrad", "Jesse"))
	is.True(dm.IsRoundPaired(2))

	is.NoErr(tournament.DeletePairings(ctx, stores.TournamentStore, ty.UUID, divOneName, 2))
	noWarnings(is)(tournament.SetPairings(ctx, stores.TournamentStore, ty.UUID, divOneName, []*pb.TournamentPairingRequest{
		{PlayerOneId: "Will:Will", PlayerTwoId: "Conrad:Conrad", Round: 1},
		{PlayerOneId: "Josh:Josh", PlayerTwoId: "Jesse:Jesse", Round: 1},
	}))
	noWarnings(is)(tournament.PairRound(ctx, stores.TournamentStore, ty.UUID, divOneName, 1, false))
	is.True(!dm.IsRoundPaired(2))
}
