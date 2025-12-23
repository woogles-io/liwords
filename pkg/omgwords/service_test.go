package omgwords

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gomodule/redigo/redis"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/omgwords/stores"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"github.com/woogles-io/liwords/rpc/api/proto/omgwords_service"

	"github.com/woogles-io/liwords/pkg/stores/user"
	pkguser "github.com/woogles-io/liwords/pkg/user"
)

var DefaultConfig = config.DefaultConfig()
var RedisURL = os.Getenv("REDIS_URL")
var pkg = "omgwords"

func newPool(addr string) *redis.Pool {
	log.Info().Str("addr", addr).Msg("new-redis-pool")
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
		Dial: func() (redis.Conn, error) { return redis.DialURL(addr) },
	}
}

/*
	func TestEditMoveAfterMaking(t *testing.T) {
		is := is.New(t)
		gdoc := loadGDoc("document-another-earlygame.json")
		ctx := ctxForTests()

		err := Send


		`{"event":{"gameId":"w3JMuMNw5kWM5vhuPZmzCD","positionCoords":"6H","tiles":"FICHE"},"userId":"internal-c","eventNumber":2,"amendment":true}`
	}
*/

func userStore() pkguser.Store {
	pool, err := common.OpenTestingDB(pkg)
	if err != nil {
		panic(err)
	}
	ustore, err := user.NewDBStore(pool)
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	return ustore
}

func metaStore() *stores.DBStore {
	pool, err := common.OpenTestingDB(pkg)
	if err != nil {
		panic(err)
	}
	mstore, err := stores.NewDBStore(pool)
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	return mstore
}

func gameStore() *stores.GameDocumentStore {
	redisPool := newPool(RedisURL)
	pool, err := common.OpenTestingDB(pkg)
	if err != nil {
		panic(err)
	}
	gds, err := stores.NewGameDocumentStore(DefaultConfig, redisPool, pool)
	if err != nil {
		panic(err)
	}
	return gds
}

func recreateDB() {
	// Create a database.
	err := common.RecreateTestDB(pkg)
	if err != nil {
		panic(err)
	}

	ustore := userStore()
	err = ustore.New(context.Background(), &entity.User{
		Username: "someuser", Email: "someemail@example.com", UUID: "someuser"})
	if err != nil {
		log.Fatal().Err(err).Msg("error creating user")
	}
	apikey, err := ustore.ResetAPIKey(context.Background(), "someuser")
	if err != nil {
		log.Fatal().Err(err).Msg("error resetting api key")
	}
	log.Info().Msgf("apikey for someuser is %s", apikey)
	ustore.(*user.DBStore).Disconnect()
}

func newService() *OMGWordsService {
	recreateDB()

	return NewOMGWordsService(userStore(), DefaultConfig, gameStore(), metaStore())
}

func ctxForTests() context.Context {
	ctx := context.Background()
	ctx = log.Logger.WithContext(ctx)
	ctx = DefaultConfig.WithContext(ctx)
	return ctx
}

func cleanupConns(svc *OMGWordsService) {
	svc.gameStore.DisconnectRDB()
	svc.userStore.(*user.DBStore).Disconnect()
	svc.metadataStore.Disconnect()
}

func TestEditMoveAfterMaking(t *testing.T) {
	is := is.New(t)
	svc := newService()
	defer func() { cleanupConns(svc) }()
	c := make(chan *entity.EventWrapper)
	svc.SetEventChannel(c)
	apikey, err := svc.userStore.GetAPIKey(context.Background(), "someuser")
	is.NoErr(err)

	ctx := ctxForTests()
	ctx = apiserver.StoreAPIKeyInContext(ctx, apikey)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for m := range c {
			log.Info().Interface("m", m).Msg("received")
		}
		log.Info().Msg("leaving channel loop")
	}()

	r, err := svc.CreateBroadcastGame(ctx, connect.NewRequest(&omgwords_service.CreateBroadcastGameRequest{
		PlayersInfo: []*ipc.PlayerInfo{
			{Nickname: "cesar", FullName: "Cesar", First: true},
			{Nickname: "someone", FullName: "Someone"},
		},
		Lexicon:       "NWL20",
		Rules:         &ipc.GameRules{BoardLayoutName: "CrosswordGame", LetterDistributionName: "english", VariantName: "classic"},
		ChallengeRule: ipc.ChallengeRule_ChallengeRule_DOUBLE,
	}))
	is.NoErr(err)
	gid := r.Msg.GameId

	_, err = svc.SetRacks(ctx, connect.NewRequest(&omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks: [][]byte{
			{1, 5, 7, 15, 22, 25}, // AEGOVY
			{},
		},
	}))
	is.NoErr(err)
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         gid,
			PositionCoords: "8D",
			MachineLetters: []byte{22, 15, 25, 1, 7, 5}, // voyage
		},
		UserId: "internal-cesar",
	}))
	is.NoErr(err)

	_, err = svc.SetRacks(ctx, connect.NewRequest(&omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks: [][]byte{
			{},
			{26, 1},
		},
	}))
	is.NoErr(err)
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         gid,
			PositionCoords: "7G",
			MachineLetters: []byte{26, 1}, // za
		},
		UserId: "internal-someone",
	}))
	is.NoErr(err)
	// Edit ZA after making it.

	_, err = svc.SetRacks(ctx, connect.NewRequest(&omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks: [][]byte{
			{},
			{6, 1, 26}, // FA + Z, so we can still make ZA
		},
		EventNumber: 1,
		Amendment:   true,
	}))
	is.NoErr(err)

	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         gid,
			PositionCoords: "7G",
			MachineLetters: []byte{6, 1}, // FA
		},
		UserId:      "internal-someone",
		EventNumber: 1,
		Amendment:   true,
	}))
	is.NoErr(err)

	close(c)
	wg.Wait()
}

func TestEditMoveAfterChallenge(t *testing.T) {
	is := is.New(t)
	svc := newService()
	defer func() { cleanupConns(svc) }()
	c := make(chan *entity.EventWrapper)
	svc.SetEventChannel(c)
	apikey, err := svc.userStore.GetAPIKey(context.Background(), "someuser")
	is.NoErr(err)

	ctx := ctxForTests()
	ctx = apiserver.StoreAPIKeyInContext(ctx, apikey)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for m := range c {
			log.Info().Interface("m", m).Msg("received")
		}
		log.Info().Msg("leaving channel loop")
	}()

	r, err := svc.CreateBroadcastGame(ctx, connect.NewRequest(&omgwords_service.CreateBroadcastGameRequest{
		PlayersInfo: []*ipc.PlayerInfo{
			{Nickname: "cesar", FullName: "Cesar", First: true},
			{Nickname: "someone", FullName: "Someone"},
		},
		Lexicon:       "NWL20",
		Rules:         &ipc.GameRules{BoardLayoutName: "CrosswordGame", LetterDistributionName: "english", VariantName: "classic"},
		ChallengeRule: ipc.ChallengeRule_ChallengeRule_DOUBLE,
	}))
	is.NoErr(err)
	gid := r.Msg.GameId

	_, err = svc.SetRacks(ctx, connect.NewRequest(&omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks: [][]byte{
			{1, 5, 7, 15, 22, 25}, // AEGOVY
			{},
		},
	}))
	is.NoErr(err)
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         gid,
			PositionCoords: "8D",
			MachineLetters: []byte{22, 15, 25, 1, 7, 5}, // voyage
		},
		UserId: "internal-cesar",
	}))
	is.NoErr(err)

	_, err = svc.SetRacks(ctx, connect.NewRequest(&omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks: [][]byte{
			{},
			{26, 1},
		},
	}))
	is.NoErr(err)

	// Play ZA in the wrong spot, then challenge it off:
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         gid,
			PositionCoords: "7F",
			MachineLetters: []byte{26, 1}, // za
		},
		UserId: "internal-someone",
	}))
	is.NoErr(err)

	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
			GameId: gid,
		},
		UserId: "internal-cesar",
	}))
	is.NoErr(err)

	// It's still Cesar's turn since the play came off. Make another play.

	_, err = svc.SetRacks(ctx, connect.NewRequest(&omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks: [][]byte{
			{1, 2, 3},
			{},
		},
	}))
	is.NoErr(err)
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         gid,
			PositionCoords: "7I",
			MachineLetters: []byte{2, 1}, // ba
		},
		UserId: "internal-cesar",
	}))
	is.NoErr(err)

	// Go back a turn and make another play.

	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         gid,
			PositionCoords: "F6",
			MachineLetters: []byte{2, 1, 0}, // BA.
		},
		UserId:      "internal-cesar",
		EventNumber: 3,
		Amendment:   true,
	}))
	is.NoErr(err)

	close(c)
	wg.Wait()
}

func TestImportGCG(t *testing.T) {
	is := is.New(t)
	svc := newService()
	defer func() { cleanupConns(svc) }()
	c := make(chan *entity.EventWrapper)
	svc.SetEventChannel(c)
	apikey, err := svc.userStore.GetAPIKey(context.Background(), "someuser")
	is.NoErr(err)

	ctx := ctxForTests()
	ctx = apiserver.StoreAPIKeyInContext(ctx, apikey)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for m := range c {
			log.Info().Interface("m", m).Msg("received")
		}
		log.Info().Msg("leaving channel loop")
	}()

	bts, err := os.ReadFile("../puzzles/testdata/r8_puneet.gcg")
	is.NoErr(err)

	gcg := string(bts)

	r, err := svc.ImportGCG(ctx, connect.NewRequest(&omgwords_service.ImportGCGRequest{
		Gcg:           gcg,
		Lexicon:       "CSW21",
		Rules:         &ipc.GameRules{BoardLayoutName: "CrosswordGame", LetterDistributionName: "english", VariantName: "classic"},
		ChallengeRule: ipc.ChallengeRule_ChallengeRule_FIVE_POINT,
	}))
	is.NoErr(err)
	gid := r.Msg.GameId

	gdoc, err := svc.GetGameDocument(ctx, connect.NewRequest(&omgwords_service.GetGameDocumentRequest{
		GameId: gid,
	}))
	is.NoErr(err)

	is.Equal(gdoc.Msg.EndReason, ipc.GameEndReason_STANDARD)
	is.Equal(gdoc.Msg.PlayState, ipc.PlayState_GAME_OVER)
	is.Equal(len(gdoc.Msg.Racks[0]), 4)
	is.Equal(len(gdoc.Msg.Racks[1]), 0)
}

func TestImportAnotherGCG(t *testing.T) {
	is := is.New(t)
	svc := newService()
	defer func() { cleanupConns(svc) }()
	c := make(chan *entity.EventWrapper)
	svc.SetEventChannel(c)
	apikey, err := svc.userStore.GetAPIKey(context.Background(), "someuser")
	is.NoErr(err)

	ctx := ctxForTests()
	ctx = apiserver.StoreAPIKeyInContext(ctx, apikey)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for m := range c {
			log.Info().Interface("m", m).Msg("received")
		}
		log.Info().Msg("leaving channel loop")
	}()

	bts, err := os.ReadFile("./testdata/vs_josh.gcg")
	is.NoErr(err)

	gcg := string(bts)

	r, err := svc.ImportGCG(ctx, connect.NewRequest(&omgwords_service.ImportGCGRequest{
		Gcg:           gcg,
		Lexicon:       "NWL20",
		Rules:         &ipc.GameRules{BoardLayoutName: "CrosswordGame", LetterDistributionName: "english", VariantName: "classic"},
		ChallengeRule: ipc.ChallengeRule_ChallengeRule_DOUBLE,
	}))
	is.NoErr(err)
	gid := r.Msg.GameId

	gdoc, err := svc.GetGameDocument(ctx, connect.NewRequest(&omgwords_service.GetGameDocumentRequest{
		GameId: gid,
	}))
	is.NoErr(err)

	is.Equal(gdoc.Msg.EndReason, ipc.GameEndReason_STANDARD)
	is.Equal(gdoc.Msg.PlayState, ipc.PlayState_GAME_OVER)
	is.Equal(len(gdoc.Msg.Racks[0]), 0)
	is.Equal(len(gdoc.Msg.Racks[1]), 6)
}

func TestImportOneMoreGCG(t *testing.T) {
	is := is.New(t)
	svc := newService()
	defer func() { cleanupConns(svc) }()
	c := make(chan *entity.EventWrapper)
	svc.SetEventChannel(c)
	apikey, err := svc.userStore.GetAPIKey(context.Background(), "someuser")
	is.NoErr(err)

	ctx := ctxForTests()
	ctx = apiserver.StoreAPIKeyInContext(ctx, apikey)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for m := range c {
			log.Info().Interface("m", m).Msg("received")
		}
		log.Info().Msg("leaving channel loop")
	}()

	bts, err := os.ReadFile("../puzzles/testdata/r4_lipe.gcg")
	is.NoErr(err)

	gcg := string(bts)

	r, err := svc.ImportGCG(ctx, connect.NewRequest(&omgwords_service.ImportGCGRequest{
		Gcg:           gcg,
		Lexicon:       "CSW21",
		Rules:         &ipc.GameRules{BoardLayoutName: "CrosswordGame", LetterDistributionName: "english", VariantName: "classic"},
		ChallengeRule: ipc.ChallengeRule_ChallengeRule_FIVE_POINT,
	}))
	is.NoErr(err)
	gid := r.Msg.GameId

	gdoc, err := svc.GetGameDocument(ctx, connect.NewRequest(&omgwords_service.GetGameDocumentRequest{
		GameId: gid,
	}))
	is.NoErr(err)

	is.Equal(gdoc.Msg.EndReason, ipc.GameEndReason_STANDARD)
	is.Equal(gdoc.Msg.PlayState, ipc.PlayState_GAME_OVER)
	is.Equal(len(gdoc.Msg.Racks[0]), 0)
	is.Equal(len(gdoc.Msg.Racks[1]), 5)
}

// TestTileConsistencyDuringAmendments replays operations from a real game that experienced
// tile corruption and validates tile counts after each operation.
// This test is based on game WEksJ4iTKCFZMJBcY6qJg2 which had 4 G's (should be 3).
func TestTileConsistencyDuringAmendments(t *testing.T) {
	is := is.New(t)
	svc := newService()
	defer func() { cleanupConns(svc) }()
	c := make(chan *entity.EventWrapper)
	svc.SetEventChannel(c)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for m := range c {
			log.Info().Interface("m", m).Msg("received")
		}
	}()

	apikey, err := svc.userStore.GetAPIKey(context.Background(), "someuser")
	is.NoErr(err)

	ctx := ctxForTests()
	ctx = apiserver.StoreAPIKeyInContext(ctx, apikey)

	resp, err := svc.CreateBroadcastGame(ctx, connect.NewRequest(&omgwords_service.CreateBroadcastGameRequest{
		PlayersInfo: []*ipc.PlayerInfo{
			{Nickname: "player1", FullName: "Player One", UserId: "player1", First: true},
			{Nickname: "player2", FullName: "Player Two", UserId: "player2", First: false},
		},
		Lexicon:       "CSW21",
		ChallengeRule: ipc.ChallengeRule_ChallengeRule_FIVE_POINT,
		Rules: &ipc.GameRules{
			BoardLayoutName:        "CrosswordGame",
			LetterDistributionName: "english",
			VariantName:            "classic",
		},
		Public: false,
	}))
	is.NoErr(err)
	gid := resp.Msg.GameId

	// Helper function to validate tile counts
	validateTileCount := func(label string) {
		gdoc, err := svc.GetGameDocument(ctx, connect.NewRequest(&omgwords_service.GetGameDocumentRequest{
			GameId: gid,
		}))
		is.NoErr(err)

		// Count total tiles
		bagCount := len(gdoc.Msg.Bag.Tiles)
		rack0Count := len(gdoc.Msg.Racks[0])
		rack1Count := len(gdoc.Msg.Racks[1])
		boardCount := 0
		// Count actual tiles on the board (excluding empty squares marked as 0)
		if gdoc.Msg.Board != nil {
			for _, tile := range gdoc.Msg.Board.Tiles {
				if tile != 0 {
					boardCount++
				}
			}
		}
		total := bagCount + rack0Count + rack1Count + boardCount

		log.Info().
			Str("label", label).
			Int("bag", bagCount).
			Int("rack0", rack0Count).
			Int("rack1", rack1Count).
			Int("board", boardCount).
			Int("total", total).
			Msg("tile-count-validation")

		// English has 100 tiles total
		is.Equal(total, 100) // Must have exactly 100 tiles at all times
	}

	// Initial validation
	validateTileCount("initial-state")

	// Operation 1: 8D - decoded from Fg8JAwU= -> [22, 15, 9, 3, 5]
	// V O I C E
	_, err = svc.SetRacks(ctx, connect.NewRequest(&omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks:  [][]byte{{22, 15, 9, 3, 5}, {}},
	}))
	is.NoErr(err)
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			GameId:         gid,
			PositionCoords: "8D",
			MachineLetters: []byte{22, 15, 9, 3, 5},
		},
		UserId:    "internal-player1",
		Amendment: false,
	}))
	is.NoErr(err)
	validateTileCount("after-op1-8D")

	// Operation 2: H5 - decoded from BgkSAAYBDgc= -> [6, 9, 18, 0, 6, 1, 14, 7]
	// Playing FIR(E)FANG - the 0 is through-tile marker, not part of rack
	_, err = svc.SetRacks(ctx, connect.NewRequest(&omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks:  [][]byte{{}, {6, 9, 18, 6, 1, 14, 7}}, // 7 tiles, excluding the 0
	}))
	is.NoErr(err)
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			GameId:         gid,
			PositionCoords: "H5",
			MachineLetters: []byte{6, 9, 18, 0, 6, 1, 14, 7},
		},
		UserId:    "internal-player2",
		Amendment: false,
	}))
	is.NoErr(err)
	validateTileCount("after-op2-H5")

	// Operation 3: D8 - decoded from AA8MFg8= -> [0, 15, 12, 22, 15]
	// The 0 is through-tile marker, not part of rack
	_, err = svc.SetRacks(ctx, connect.NewRequest(&omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks:  [][]byte{{15, 12, 22, 15}, {}}, // 4 tiles, excluding the 0
	}))
	is.NoErr(err)
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			GameId:         gid,
			PositionCoords: "D8",
			MachineLetters: []byte{0, 15, 12, 22, 15},
		},
		UserId:    "internal-player1",
		Amendment: false,
	}))
	is.NoErr(err)
	validateTileCount("after-op3-D8")

	// Operation 4: 11G - decoded from iQAKhQMU -> [137, 0, 10, 133, 3, 20]
	// The played tiles contain designated blanks (high bit set), but racks should have undesignated blanks
	_, err = svc.SetRacks(ctx, connect.NewRequest(&omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks:  [][]byte{{}, {0, 10, 0, 3, 20}}, // 5 tiles: blank, J, blank, D, T
	}))
	is.NoErr(err)
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			GameId:         gid,
			PositionCoords: "11G",
			MachineLetters: []byte{137, 0, 10, 133, 3, 20},
		},
		UserId:    "internal-player2",
		Amendment: false,
	}))
	is.NoErr(err)
	validateTileCount("after-op4-11G")

	// Operation 5: AMENDMENT - 11G again but with different tiles
	// decoded from CQAKBQMU -> [9, 0, 10, 5, 3, 20]
	// This edits event at index 3 (0-based, the 4th event)
	_, err = svc.SetRacks(ctx, connect.NewRequest(&omgwords_service.SetRacksEvent{
		GameId:      gid,
		Racks:       [][]byte{{}, {9, 10, 5, 3, 20}}, // 5 tiles, excluding the 0
		EventNumber: 3,
		Amendment:   true,
	}))
	is.NoErr(err)
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			GameId:         gid,
			PositionCoords: "11G",
			MachineLetters: []byte{9, 0, 10, 5, 3, 20},
		},
		UserId:      "internal-player2",
		Amendment:   true,
		EventNumber: 3, // Edit the 4th event (0-indexed)
	}))
	is.NoErr(err)
	validateTileCount("after-op5-11G-amended")

	// Operation 6: 10D - decoded from FgkPDAA= -> [22, 9, 15, 12, 0]
	// This should FAIL because tile 22 (V) is already used twice on the board (only 2 V's in distribution)
	// Op 1 used one V, Op 3 used the other V
	_, err = svc.SetRacks(ctx, connect.NewRequest(&omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks:  [][]byte{{22, 9, 15, 12}, {}}, // Trying to set rack with V, I, O, L
	}))
	is.True(err != nil) // Should fail - can't get third V from distribution
	// Don't validate tile count after failed operation

	close(c)
	wg.Wait()
}
func TestTileCorruption5Z(t *testing.T) {
	is := is.New(t)
	svc := newService()
	defer func() { cleanupConns(svc) }()
	c := make(chan *entity.EventWrapper)
	svc.SetEventChannel(c)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for m := range c {
			log.Info().Interface("m", m).Msg("received")
		}
	}()

	apikey, err := svc.userStore.GetAPIKey(context.Background(), "someuser")
	is.NoErr(err)

	ctx := ctxForTests()
	ctx = apiserver.StoreAPIKeyInContext(ctx, apikey)

	resp, err := svc.CreateBroadcastGame(ctx, connect.NewRequest(&omgwords_service.CreateBroadcastGameRequest{
		PlayersInfo: []*ipc.PlayerInfo{
			{Nickname: "amy", FullName: "Amy", UserId: "amy", First: true},
			{Nickname: "mikey", FullName: "Mikey", UserId: "mikey", First: false},
		},
		Lexicon:       "CSW21",
		ChallengeRule: ipc.ChallengeRule_ChallengeRule_FIVE_POINT,
		Rules: &ipc.GameRules{
			BoardLayoutName:        "CrosswordGame",
			LetterDistributionName: "english",
			VariantName:            "classic",
		},
		Public: false,
	}))
	is.NoErr(err)
	gid := resp.Msg.GameId

	// Helper function to validate tile counts
	validateTileCount := func(label string) {
		gdoc, err := svc.GetGameDocument(ctx, connect.NewRequest(&omgwords_service.GetGameDocumentRequest{
			GameId: gid,
		}))
		is.NoErr(err)

		// Count total tiles
		bagCount := len(gdoc.Msg.Bag.Tiles)
		rack0Count := len(gdoc.Msg.Racks[0])
		rack1Count := len(gdoc.Msg.Racks[1])
		boardCount := 0
		for _, evt := range gdoc.Msg.Events {
			if evt.Type == ipc.GameEvent_TILE_PLACEMENT_MOVE {
				for _, tile := range evt.PlayedTiles {
					if tile != 0 {
						boardCount++
					}
				}
			}
		}
		total := bagCount + rack0Count + rack1Count + boardCount

		log.Info().
			Str("label", label).
			Int("bag", bagCount).
			Int("rack0", rack0Count).
			Int("rack1", rack1Count).
			Int("board", boardCount).
			Int("total", total).
			Msg("tile-count-validation")

		// English has 100 tiles total
		is.Equal(total, 100) // Must have exactly 100 tiles at all times
	}

	// Initial validation
	validateTileCount("initial-state")

	// Op 0: amy, type=EXCHANGE, tiles=[9, 9, 15]
	// Set amy's rack to include these tiles first
	_, err = svc.SetRacks(ctx, connect.NewRequest(&omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks:  [][]byte{{9, 9, 15, 1, 2, 3, 4}, {}}, // Include the exchange tiles
	}))
	is.NoErr(err)
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_EXCHANGE,
			GameId:         gid,
			MachineLetters: []byte{9, 9, 15},
		},
		UserId:    "internal-amy",
		Amendment: false,
	}))
	is.NoErr(err)
	validateTileCount("after-op0-exchange")

	// Op 1: mikey, coords="8D", tiles=[12, 5, 21, 3, 9, 14, 5]
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			GameId:         gid,
			PositionCoords: "8D",
			MachineLetters: []byte{12, 5, 21, 3, 9, 14, 5},
		},
		UserId:    "internal-mikey",
		Amendment: false,
	}))
	is.NoErr(err)
	validateTileCount("after-op1-8D")

	// Op 2: amy, coords="I3", tiles=[13, 25, 5, 12, 9, 0, 5, 19]
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			GameId:         gid,
			PositionCoords: "I3",
			MachineLetters: []byte{13, 25, 5, 12, 9, 0, 5, 19},
		},
		UserId:    "internal-amy",
		Amendment: false,
	}))
	is.NoErr(err)
	validateTileCount("after-op2-I3")

	// Op 3: mikey, coords="8D", tiles=[14, 21, 3, 12, 5, 9] - AMENDMENT of event 1
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			GameId:         gid,
			PositionCoords: "8D",
			MachineLetters: []byte{14, 21, 3, 12, 5, 9},
		},
		UserId:      "internal-mikey",
		Amendment:   true,
		EventNumber: 1,
	}))
	is.NoErr(err)
	validateTileCount("after-op3-8D-amended")

	// Op 4: amy, coords="D3", tiles=[13, 25, 5, 12, 9, 0, 5, 19] - AMENDMENT of event 2
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			GameId:         gid,
			PositionCoords: "D3",
			MachineLetters: []byte{13, 25, 5, 12, 9, 0, 5, 19},
		},
		UserId:      "internal-amy",
		Amendment:   true,
		EventNumber: 2,
	}))
	is.NoErr(err)
	validateTileCount("after-op4-D3-amended")

	// Op 5: mikey, coords="C9", tiles=[2, 9, 1, 12, 9]
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			GameId:         gid,
			PositionCoords: "C9",
			MachineLetters: []byte{2, 9, 1, 12, 9},
		},
		UserId:    "internal-mikey",
		Amendment: false,
	}))
	is.NoErr(err)
	validateTileCount("after-op5-C9")

	// Op 6: amy, coords="3A", tiles=[26, 15, 15, 0]
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			GameId:         gid,
			PositionCoords: "3A",
			MachineLetters: []byte{26, 15, 15, 0},
		},
		UserId:    "internal-amy",
		Amendment: false,
	}))
	is.NoErr(err)
	validateTileCount("after-op6-3A")

	// Op 7: mikey, coords="B1", tiles=[2, 9, 0, 14, 20, 19]
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			GameId:         gid,
			PositionCoords: "B1",
			MachineLetters: []byte{2, 9, 0, 14, 20, 19},
		},
		UserId:    "internal-mikey",
		Amendment: false,
	}))
	is.NoErr(err)
	validateTileCount("after-op7-B1")

	// Op 8: amy, coords="1B", tiles=[0, 5, 8, 1, 22, 137, 14, 135]
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			GameId:         gid,
			PositionCoords: "1B",
			MachineLetters: []byte{0, 5, 8, 1, 22, 137, 14, 135},
		},
		UserId:    "internal-amy",
		Amendment: false,
	}))
	is.NoErr(err)
	validateTileCount("after-op8-1B")

	// Op 9: mikey, coords="12A", tiles=[2, 15, 0, 20, 5, 4]
	// This should FAIL because tile 2 (B) is already used twice on the board (only 2 B's in distribution)
	_, err = svc.SendGameEvent(ctx, connect.NewRequest(&omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			GameId:         gid,
			PositionCoords: "12A",
			MachineLetters: []byte{2, 15, 0, 20, 5, 4},
		},
		UserId:    "internal-mikey",
		Amendment: false,
	}))
	is.True(err != nil) // Should fail - can't get third B from distribution
	// Don't validate tile count after failed operation

	close(c)
	wg.Wait()
}
