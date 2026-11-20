package league

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/user"
	"github.com/woogles-io/liwords/pkg/utilities"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

const seasonStartGamesPkg = "league_start_games_test"

// realGameCreator creates and starts games the way the API's adapter does
// (cmd/liwords-api/league_adapter.go).
type realGameCreator struct {
	stores    *stores.Stores
	cfg       *config.Config
	eventChan chan<- *entity.EventWrapper
	// startedWith is the users store each StartGame was given.
	startedWith []user.Store
}

func (r *realGameCreator) InstantiateNewGame(ctx context.Context, users [2]*entity.User, req *pb.GameRequest,
	tdata *entity.TournamentData, opts ...gameplay.GameOption) (*entity.Game, error) {
	return gameplay.InstantiateNewGame(ctx, r.stores.GameStore, r.cfg, users, req, tdata, opts...)
}

func (r *realGameCreator) StartGame(ctx context.Context, game *entity.Game, users user.Store) error {
	r.startedWith = append(r.startedWith, users)
	return gameplay.StartGameWithUsers(ctx, r.stores, users, r.eventChan, game)
}

func (r *realGameCreator) GetGame(ctx context.Context, id string) (*entity.Game, error) {
	return r.stores.GameStore.Get(ctx, id)
}

// events collects the events games send.
type events struct {
	mu   sync.Mutex
	list []*entity.EventWrapper
}

func (e *events) all() []*entity.EventWrapper {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]*entity.EventWrapper(nil), e.list...)
}

// setupSeasonStartGames builds the full stores, so games are really created
// and started, with an event channel whose events are collected.
func setupSeasonStartGames(t *testing.T) (context.Context, *stores.Stores, *config.Config, chan *entity.EventWrapper, *events) {
	if err := common.RecreateTestDB(seasonStartGamesPkg); err != nil {
		t.Fatal(err)
	}
	pool, err := common.OpenTestingDB(seasonStartGamesPkg)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.DBConnDSN = common.TestingPostgresConnDSN(seasonStartGamesPkg)
	allStores, err := stores.NewInitializedStores(pool, nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := createIntegrationTestUsers(pool); err != nil {
		t.Fatal(err)
	}

	ch := make(chan *entity.EventWrapper, 100)
	allStores.GameStore.SetGameEventChan(ch)
	ctx, cancel := context.WithCancel(cfg.WithContext(log.Logger.WithContext(context.Background())))
	evts := &events{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case e := <-ch:
				evts.mu.Lock()
				evts.list = append(evts.list, e)
				evts.mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()
	t.Cleanup(func() {
		cancel()
		<-done
		allStores.Disconnect()
	})
	return ctx, allStores, cfg, ch, evts
}

// Starting a season creates and starts a game for every pair, through the
// real gameplay code, while looking each player up once; a suspended
// player's name is still censored in every game they're in; no game is taken
// for a rematch; and the start pauses after each batch of games.
func TestSeasonStartCreatesGames(t *testing.T) {
	is := is.New(t)
	ctx, allStores, cfg, ch, evts := setupSeasonStartGames(t)

	leagueID, seasonID, divID := seasonToStart(t, ctx, allStores.LeagueStore)
	regs, err := allStores.LeagueStore.GetDivisionRegistrations(ctx, divID)
	is.NoErr(err)
	is.Equal(len(regs), startPlayers)

	// One player is suspended for good.
	suspended := regs[0].UserUuid
	is.NoErr(allStores.UserStore.ApplyActions(ctx, []*ms.ModAction{
		{UserId: suspended, Type: ms.ModActionType_SUSPEND_ACCOUNT}}))

	creator := &realGameCreator{stores: allStores, cfg: cfg, eventChan: ch}
	ssm := NewSeasonStartManager(allStores.LeagueStore, allStores, cfg, creator)
	counted := newCountingUsers(allStores.UserStore)
	ssm.users = counted
	var pauses []time.Duration
	ssm.sleep = func(_ context.Context, d time.Duration) error {
		pauses = append(pauses, d)
		return nil
	}

	settings := &pb.LeagueSettings{
		Lexicon:       "CSW21",
		Variant:       "classic",
		ChallengeRule: pb.ChallengeRule_ChallengeRule_FIVE_POINT,
		TimeControl:   &pb.TimeControl{IncrementSeconds: 28800, TimeBankMinutes: 4320},
	}
	result, err := ssm.CreateGamesForSeason(ctx, leagueID, seasonID, settings, startPauseEvery, startPauseGames)
	is.NoErr(err)
	const games = startPlayers * (startPlayers - 1) / 2
	is.Equal(result.TotalGamesCreated, games)

	// Every pair has one game, started, in this division.
	rows, err := allStores.LeagueStore.GetDivisionGamePlayers(ctx, divID)
	is.NoErr(err)
	is.Equal(len(rows), games)
	pairs := map[[2]int32]bool{}
	gameIDs := map[string]bool{}
	for _, r := range rows {
		a, b := r.Player0ID.Int32, r.Player1ID.Int32
		if a > b {
			a, b = b, a
		}
		is.True(!pairs[[2]int32{a, b}]) // a pair with two games
		pairs[[2]int32{a, b}] = true
		is.True(r.Started.Bool)
		gameIDs[r.Uuid.String] = true
	}

	// Each game was started with the same users store, and each player was
	// looked up once in the store under it, however many games they're in.
	is.Equal(len(creator.startedWith), games)
	for _, u := range creator.startedWith {
		is.True(u == creator.startedWith[0])
	}
	is.Equal(len(counted.getByUUID), startPlayers)
	for _, r := range regs {
		is.Equal(counted.getByUUID[r.UserUuid], 1)
		is.Equal(counted.getActions[r.UserUuid], 1)
		is.True(counted.isBot[r.UserUuid] <= 1)
	}

	// Pauses: one after every startPauseGames games, none after the last.
	is.Equal(len(pauses), (games-1)/startPauseGames)
	for _, d := range pauses {
		is.Equal(d, startPauseEvery)
	}

	// Wait for every game's start events: each game sends its history to its
	// players and, censored, to its viewers.
	deadline := time.Now().Add(5 * time.Second)
	var refreshers []*pb.GameHistoryRefresher
	for {
		refreshers = refreshers[:0]
		for _, e := range evts.all() {
			is.True(e.Type != pb.MessageType_REMATCH_STARTED) // no league game is a rematch
			if e.Type == pb.MessageType_GAME_HISTORY_REFRESHER {
				refreshers = append(refreshers, e.Event.(*pb.GameHistoryRefresher))
			}
		}
		if len(refreshers) >= 2*games || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	is.Equal(len(refreshers), 2*games)

	// The suspended player is censored in each of their games, and only they
	// are.
	censoredGames := map[string]bool{}
	for _, r := range refreshers {
		is.True(gameIDs[r.History.Uid])
		for _, p := range r.History.Players {
			is.True(p.UserId != suspended) // shown uncensored
			if p.UserId == utilities.CensoredUsername {
				is.Equal(p.Nickname, utilities.CensoredUsername)
				censoredGames[r.History.Uid] = true
			}
		}
	}
	is.Equal(len(censoredGames), startPlayers-1)
}

// A start stops when its context is done during a pause, with the games
// created so far counted.
func TestSeasonStartStopsDuringPause(t *testing.T) {
	is := is.New(t)
	ctx, allStores, cfg, ch, _ := setupSeasonStartGames(t)

	leagueID, seasonID, divID := seasonToStart(t, ctx, allStores.LeagueStore)
	creator := &realGameCreator{stores: allStores, cfg: cfg, eventChan: ch}
	ssm := NewSeasonStartManager(allStores.LeagueStore, allStores, cfg, creator)
	ssm.sleep = func(context.Context, time.Duration) error { return context.Canceled }

	settings := &pb.LeagueSettings{Lexicon: "CSW21", Variant: "classic"}
	result, err := ssm.CreateGamesForSeason(ctx, leagueID, seasonID, settings, startPauseEvery, 4)
	is.True(err != nil)
	is.Equal(result.TotalGamesCreated, 0) // the division failed
	rows, err := allStores.LeagueStore.GetDivisionGamePlayers(ctx, divID)
	is.NoErr(err)
	is.Equal(len(rows), 4) // created before the first pause
	for _, r := range rows {
		is.True(r.Started.Bool)
	}
}

// sleepCtx sleeps for the whole period, or stops when the context is done.
func TestSleepCtx(t *testing.T) {
	is := is.New(t)
	start := time.Now()
	is.NoErr(sleepCtx(context.Background(), 20*time.Millisecond))
	is.True(time.Since(start) >= 20*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start = time.Now()
	is.Equal(sleepCtx(ctx, time.Hour), context.Canceled)
	is.True(time.Since(start) < time.Second)
}
