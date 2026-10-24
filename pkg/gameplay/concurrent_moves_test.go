package gameplay_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/common"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// lockedBuffer collects log lines written from many goroutines.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// drainEvents swallows game events until ctx ends; nothing in these tests
// listens to them.
func drainEvents(ctx context.Context) chan *entity.EventWrapper {
	ch := make(chan *entity.EventWrapper, 256)
	go func() {
		for {
			select {
			case <-ch:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

// Each contested move must be accepted exactly once, and at the end the saved
// history, the turn rows and the shadows must all agree. Many games run at
// once, so every variant also runs well past the per-process limit on held
// locks.

// Two app servers -- separate connection pools, separate caches, one database
// -- each receive the same move for the same game at the same moment, the way
// a double-click or two open tabs can deliver it. For correspondence games,
// which every server loads fresh from the database, the game lock is all that
// keeps the two from both playing it.
func TestConcurrentMovesFromTwoServers(t *testing.T) {
	contestedMoves(t, 2, pb.GameMode_CORRESPONDENCE)
}

// The same on one server. A correspondence game is still a fresh copy on every
// load, so two goroutines on the same server are as independent as two
// servers, and only the game lock keeps them apart.
func TestConcurrentMovesOnOneServerCorrespondence(t *testing.T) {
	contestedMoves(t, 1, pb.GameMode_CORRESPONDENCE)
}

// A real-time game on one server, which is production today: the cache hands
// both goroutines the same object, and that object's own mutex already keeps
// them apart -- this passes with the game lock disabled. It becomes a test of
// the game lock once the cache is gone and every load is a fresh copy.
func TestConcurrentMovesOnOneServerRealTime(t *testing.T) {
	contestedMoves(t, 1, pb.GameMode_REAL_TIME)
}

// contestedMoves plays rounds of contested moves on many games at once. Each
// move is handed to two callers at the same moment: on two store stacks
// (separate pools and caches, one database) when servers is 2, or twice on the
// same one when it is 1.
func contestedMoves(t *testing.T, servers int, mode pb.GameMode) {
	is := is.New(t)

	pool, storesA, cfg := recreateDB()
	defer storesA.Disconnect()
	storesB := storesA
	if servers == 2 {
		poolB, err := common.OpenTestingDB(pkg)
		is.NoErr(err)
		storesB, err = stores.NewInitializedStores(poolB, nil, cfg)
		is.NoErr(err)
		defer storesB.Disconnect()
	}

	// Production's flags: turn rows written with every save, and both shadows on.
	runCfg := *cfg
	runCfg.DualWriteTurns = true
	runCfg.ShadowTurns = true
	runCfg.ShadowTurnsLoad = true
	var errLog lockedBuffer
	logger := zerolog.New(&errLog).Level(zerolog.WarnLevel)
	ctx, cancel := context.WithCancel(runCfg.WithContext(logger.WithContext(context.Background())))
	defer cancel()

	chA := drainEvents(ctx)
	storesA.GameStore.SetGameEventChan(chA)
	if storesB != storesA {
		storesB.GameStore.SetGameEventChan(drainEvents(ctx))
	}

	jesse, err := storesA.UserStore.Get(ctx, "jesse")
	is.NoErr(err)
	cesar, err := storesA.UserStore.Get(ctx, "cesar4")
	is.NoErr(err)

	const games = 20
	// Passes, so any rack will do. Six scoreless turns would end the game.
	const rounds = 4

	gameIDs := make([]string, games)
	for i := range gameIDs {
		gr := proto.Clone(gameReq).(*pb.GameRequest)
		if mode == pb.GameMode_CORRESPONDENCE {
			withCorrespondence(gr)
			// Correspondence timing: the increment is the time allowed per turn.
			gr.InitialTimeSeconds = 3 * 24 * 3600
			gr.IncrementSeconds = 3 * 24 * 3600
			gr.MaxOvertimeMinutes = 0
		}
		gr.RequestId = fmt.Sprintf("concurrent-%d", i)
		g, err := gameplay.InstantiateNewGame(ctx, storesA.GameStore, &runCfg,
			[2]*entity.User{jesse, cesar}, gr, nil)
		is.NoErr(err)
		is.NoErr(gameplay.StartGame(ctx, storesA, chA, g))
		gameIDs[i] = g.GameID()
	}

	type outcome struct {
		game, round, accepted int
		errs                  []string
	}
	results := make(chan outcome, games)

	var wg sync.WaitGroup
	for gi, gid := range gameIDs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for r := range rounds {
				g, err := storesA.GameStore.Get(ctx, gid)
				if err != nil {
					results <- outcome{game: gi, round: r, errs: []string{err.Error()}}
					return
				}
				g.RLock()
				onTurn := g.Game.PlayerIDOnTurn()
				g.RUnlock()
				cge := &pb.ClientGameplayEvent{Type: pb.ClientGameplayEvent_PASS, GameId: gid}

				// Both callers are released together.
				start := make(chan struct{})
				var mu sync.Mutex
				var errs []string
				accepted := 0
				var both sync.WaitGroup
				for _, st := range []*stores.Stores{storesA, storesB} {
					both.Add(1)
					go func() {
						defer both.Done()
						<-start
						_, err := gameplay.HandleEvent(ctx, st, onTurn, cge)
						mu.Lock()
						defer mu.Unlock()
						if err == nil {
							accepted++
						} else {
							errs = append(errs, err.Error())
						}
					}()
				}
				close(start)
				both.Wait()
				if accepted != 1 {
					results <- outcome{game: gi, round: r, accepted: accepted, errs: errs}
					return
				}
			}
			results <- outcome{game: gi, round: rounds, accepted: 1}
		}()
	}
	wg.Wait()
	close(results)

	for o := range results {
		if o.round != rounds {
			t.Errorf("game %d round %d: %d copies accepted, want 1 (errors: %v)",
				o.game, o.round, o.accepted, o.errs)
		}
	}

	// Every game has exactly the moves that were accepted, in the history and
	// in the turn rows alike, whichever server loads it.
	for _, gid := range gameIDs {
		for _, st := range []*stores.Stores{storesA, storesB} {
			g, err := st.GameStore.Get(ctx, gid)
			is.NoErr(err)
			g.RLock()
			n := len(g.History().Events)
			g.RUnlock()
			is.Equal(n, rounds)
		}
		var rows int
		is.NoErr(pool.QueryRow(ctx,
			"SELECT count(*) FROM game_turns WHERE game_uuid = $1", gid).Scan(&rows))
		is.Equal(rows, rounds)
	}

	for _, line := range strings.Split(strings.TrimSpace(errLog.String()), "\n") {
		if strings.Contains(line, "shadow-") && !strings.Contains(line, "shadow-summary") {
			t.Errorf("shadow reported a problem: %s", line)
		}
		if strings.Contains(line, "game_turns_pkey") {
			t.Errorf("a duplicate turn row reached the database: %s", line)
		}
	}
}
