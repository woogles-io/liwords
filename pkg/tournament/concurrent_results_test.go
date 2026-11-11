package tournament_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
	tournamentstore "github.com/woogles-io/liwords/pkg/stores/tournament"
	"github.com/woogles-io/liwords/pkg/tournament"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// The worst case of results bunching up toward the end of a round: every game
// of the round ends at once, and the results come in through two servers. Each loads the tournament, records its result
// and saves; without the lock, a save made from a copy loaded before another
// result was saved would drop that result.
func TestConcurrentResultsAcrossServers(t *testing.T) {
	is := is.New(t)
	roundControls := makeRoundControls()[:2]
	for _, rc := range roundControls {
		rc.PairingMethod = ipc.PairingMethod_KING_OF_THE_HILL
	}
	ctx, stores, ty, _ := setUpServiceDivision(t, map[string]int32{
		"Will": 4000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100,
		"Kieran": 2000, "Vince": 1900, "Jennifer": 1800, "Guy": 1700,
	}, false, roundControls)

	// A second server: its own pool and store, so its own in-process lock.
	pool, err := common.OpenTestingDB(pkg)
	is.NoErr(err)
	defer pool.Close()
	other, err := tournamentstore.NewDBStore(pool, nil)
	is.NoErr(err)
	other.SetTournamentEventChan(make(chan *entity.EventWrapper, 1000))

	dm := ty.Divisions[divOneName].DivisionManager
	persons := dm.GetPlayers().Persons
	games := [][2]string{}
	seen := map[string]bool{}
	for _, p := range persons {
		if seen[p.Id] {
			continue
		}
		pairing, err := dm.GetPairing(p.Id, 0)
		is.NoErr(err)
		ids := [2]string{}
		for i, idx := range pairing.Players {
			ids[i] = persons[idx].Id
			seen[ids[i]] = true
		}
		games = append(games, ids)
	}
	is.Equal(len(games), 4)

	var wg sync.WaitGroup
	errs := make(chan error, len(games))
	for i, g := range games {
		var ts tournament.TournamentStore = stores.TournamentStore
		if i%2 == 1 {
			ts = other
		}
		// Results come in by user UUID; the test users' UUIDs are their names.
		p1, _, _ := strings.Cut(g[0], ":")
		p2, _, _ := strings.Cut(g[1], ":")
		wg.Go(func() {
			errs <- tournament.SetResult(ctx, ts, stores.UserStore, ty.UUID, divOneName,
				p1, p2, 500, 400, ipc.TournamentGameResult_WIN, ipc.TournamentGameResult_LOSS,
				ipc.GameEndReason_STANDARD, 0, 0, false, nil, stores.Queries)
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		is.NoErr(err)
	}

	complete, err := reload(ctx, is, stores.TournamentStore, ty.UUID).Divisions[divOneName].DivisionManager.IsRoundComplete(0)
	is.NoErr(err)
	is.True(complete) // every result was kept
}
