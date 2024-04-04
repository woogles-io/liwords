package omgwords

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

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
	pool, err := common.OpenTestingDB()
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
	pool, err := common.OpenTestingDB()
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
	pool, err := common.OpenTestingDB()
	if err != nil {
		panic(err)
	}
	gds, err := stores.NewGameDocumentStore(&DefaultConfig, redisPool, pool)
	if err != nil {
		panic(err)
	}
	return gds
}

func recreateDB() {
	// Create a database.
	err := common.RecreateTestDB()
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

	return NewOMGWordsService(userStore(), &DefaultConfig, gameStore(), metaStore())
}

func ctxForTests() context.Context {
	ctx := context.Background()
	ctx = log.Logger.WithContext(ctx)
	ctx = context.WithValue(ctx, config.CtxKeyword, &DefaultConfig)
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

	r, err := svc.CreateBroadcastGame(ctx, &omgwords_service.CreateBroadcastGameRequest{
		PlayersInfo: []*ipc.PlayerInfo{
			{Nickname: "cesar", FullName: "Cesar", First: true},
			{Nickname: "someone", FullName: "Someone"},
		},
		Lexicon:       "NWL20",
		Rules:         &ipc.GameRules{BoardLayoutName: "CrosswordGame", LetterDistributionName: "english", VariantName: "classic"},
		ChallengeRule: ipc.ChallengeRule_ChallengeRule_DOUBLE,
	})
	is.NoErr(err)
	gid := r.GameId

	_, err = svc.SetRacks(ctx, &omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks: [][]byte{
			{1, 5, 7, 15, 22, 25}, // AEGOVY
			{},
		},
	})
	is.NoErr(err)
	_, err = svc.SendGameEvent(ctx, &omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         gid,
			PositionCoords: "8D",
			MachineLetters: []byte{22, 15, 25, 1, 7, 5}, // voyage
		},
		UserId: "internal-cesar",
	})
	is.NoErr(err)

	_, err = svc.SetRacks(ctx, &omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks: [][]byte{
			{},
			{26, 1},
		},
	})
	is.NoErr(err)
	_, err = svc.SendGameEvent(ctx, &omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         gid,
			PositionCoords: "7G",
			MachineLetters: []byte{26, 1}, // za
		},
		UserId: "internal-someone",
	})
	is.NoErr(err)
	// Edit ZA after making it.

	_, err = svc.SetRacks(ctx, &omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks: [][]byte{
			{},
			{6, 1},
		},
		EventNumber: 1,
		Amendment:   true,
	})
	is.NoErr(err)

	_, err = svc.SendGameEvent(ctx, &omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         gid,
			PositionCoords: "7G",
			MachineLetters: []byte{6, 1}, // FA
		},
		UserId:      "internal-someone",
		EventNumber: 1,
		Amendment:   true,
	})
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

	r, err := svc.CreateBroadcastGame(ctx, &omgwords_service.CreateBroadcastGameRequest{
		PlayersInfo: []*ipc.PlayerInfo{
			{Nickname: "cesar", FullName: "Cesar", First: true},
			{Nickname: "someone", FullName: "Someone"},
		},
		Lexicon:       "NWL20",
		Rules:         &ipc.GameRules{BoardLayoutName: "CrosswordGame", LetterDistributionName: "english", VariantName: "classic"},
		ChallengeRule: ipc.ChallengeRule_ChallengeRule_DOUBLE,
	})
	is.NoErr(err)
	gid := r.GameId

	_, err = svc.SetRacks(ctx, &omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks: [][]byte{
			{1, 5, 7, 15, 22, 25}, // AEGOVY
			{},
		},
	})
	is.NoErr(err)
	_, err = svc.SendGameEvent(ctx, &omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         gid,
			PositionCoords: "8D",
			MachineLetters: []byte{22, 15, 25, 1, 7, 5}, // voyage
		},
		UserId: "internal-cesar",
	})
	is.NoErr(err)

	_, err = svc.SetRacks(ctx, &omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks: [][]byte{
			{},
			{26, 1},
		},
	})
	is.NoErr(err)

	// Play ZA in the wrong spot, then challenge it off:
	_, err = svc.SendGameEvent(ctx, &omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         gid,
			PositionCoords: "7F",
			MachineLetters: []byte{26, 1}, // za
		},
		UserId: "internal-someone",
	})
	is.NoErr(err)

	_, err = svc.SendGameEvent(ctx, &omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
			GameId: gid,
		},
		UserId: "internal-cesar",
	})
	is.NoErr(err)

	// It's still Cesar's turn since the play came off. Make another play.

	_, err = svc.SetRacks(ctx, &omgwords_service.SetRacksEvent{
		GameId: gid,
		Racks: [][]byte{
			{1, 2, 3},
			{},
		},
	})
	is.NoErr(err)
	_, err = svc.SendGameEvent(ctx, &omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         gid,
			PositionCoords: "7I",
			MachineLetters: []byte{2, 1}, // ba
		},
		UserId: "internal-cesar",
	})
	is.NoErr(err)

	// Go back a turn and make another play.

	_, err = svc.SendGameEvent(ctx, &omgwords_service.AnnotatedGameEvent{
		Event: &ipc.ClientGameplayEvent{
			Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         gid,
			PositionCoords: "F6",
			MachineLetters: []byte{2, 1, 0}, // BA.
		},
		UserId:      "internal-cesar",
		EventNumber: 3,
		Amendment:   true,
	})
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

	r, err := svc.ImportGCG(ctx, &omgwords_service.ImportGCGRequest{
		Gcg:           gcg,
		Lexicon:       "CSW21",
		Rules:         &ipc.GameRules{BoardLayoutName: "CrosswordGame", LetterDistributionName: "english", VariantName: "classic"},
		ChallengeRule: ipc.ChallengeRule_ChallengeRule_FIVE_POINT,
	})
	is.NoErr(err)
	gid := r.GameId

	gdoc, err := svc.GetGameDocument(ctx, &omgwords_service.GetGameDocumentRequest{
		GameId: gid,
	})
	is.NoErr(err)

	fmt.Println(gdoc.Events)

	is.Equal(gdoc.EndReason, ipc.GameEndReason_STANDARD)
	is.Equal(gdoc.PlayState, ipc.PlayState_GAME_OVER)
	is.True(false)
}
