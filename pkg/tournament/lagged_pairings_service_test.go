package tournament_test

import (
	"context"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/tournament"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
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
