//go:build integ

package integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/lithammer/shortuuid/v4"
	"github.com/matryer/is"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	macondoconfig "github.com/domino14/macondo/config"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/woogles-io/liwords/pkg/bus"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores/common"
	cfgstore "github.com/woogles-io/liwords/pkg/stores/config"
	"github.com/woogles-io/liwords/pkg/stores/game"
	pkgredis "github.com/woogles-io/liwords/pkg/stores/redis"
	"github.com/woogles-io/liwords/pkg/stores/soughtgame"
	ts "github.com/woogles-io/liwords/pkg/stores/tournament"
	"github.com/woogles-io/liwords/pkg/stores/user"
	"github.com/woogles-io/liwords/pkg/tournament"
	pkguser "github.com/woogles-io/liwords/pkg/user"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

var pkg = "integration_test"

const (
	tournamentName = "my tournament"
)

var DefaultConfig = macondoconfig.Config{
	LexiconPath:               os.Getenv("LEXICON_PATH"),
	LetterDistributionPath:    os.Getenv("LETTER_DISTRIBUTION_PATH"),
	DataPath:                  os.Getenv("DATA_PATH"),
	DefaultLexicon:            "CSW21",
	DefaultLetterDistribution: "English",
}

var gameReq = &ipc.GameRequest{Lexicon: "CSW21",
	Rules: &ipc.GameRules{BoardLayoutName: entity.CrosswordGame,
		LetterDistributionName: "English",
		VariantName:            "classic"},

	InitialTimeSeconds: 25 * 60,
	IncrementSeconds:   0,
	ChallengeRule:      macondopb.ChallengeRule_FIVE_POINT,
	GameMode:           ipc.GameMode_REAL_TIME,
	RatingMode:         ipc.RatingMode_RATED,
	RequestId:          "yeet",
	OriginalRequestId:  "originalyeet",
	MaxOvertimeMinutes: 10}

func userStore() pkguser.Store {
	ustore, err := user.NewDBStore(common.TestingPostgresConnDSN(pkg))
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	return ustore
}

func gameStore(userStore pkguser.Store) (*config.Config, gameplay.GameStore) {
	cfg := &config.Config{}
	cfg.MacondoConfig = DefaultConfig
	cfg.DBConnDSN = common.TestingPostgresConnDSN(pkg)

	tmp, err := game.NewDBStore(cfg, userStore)
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	gameStore := game.NewCache(tmp)
	return cfg, gameStore
}

func tournamentStore(gs gameplay.GameStore) (*config.Config, tournament.TournamentStore) {
	cfg := &config.Config{}
	cfg.MacondoConfig = DefaultConfig
	cfg.DBConnDSN = common.TestingPostgresConnDSN(pkg)

	tmp, err := ts.NewDBStore(cfg, gs)
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	tournamentStore := ts.NewCache(tmp)
	return cfg, tournamentStore
}

const NumUsers = 200

func recreateDBManyUsers() {
	err := common.RecreateTestDB(pkg)
	if err != nil {
		panic(err)
	}

	ustore := userStore()

	for i := 0; i < NumUsers; i++ {
		u := &entity.User{
			Username: fmt.Sprintf("Player%d", i+1),
			Email:    fmt.Sprintf("player%d@example.com", i+1),
			UUID:     fmt.Sprintf("uuid-%d", i+1), // i know it's not a UUID
		}
		err = ustore.New(context.Background(), u)
		if err != nil {
			log.Fatal().Err(err).Msg("error")
		}
	}
	err = ustore.New(context.Background(), &entity.User{
		Username: "Kieran",
		Email:    "kieran@example.com",
		UUID:     "uuid-kieran",
	})
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	ustore.(*user.DBStore).Disconnect()
}

func makeTournament(ctx context.Context, ts tournament.TournamentStore, cfg *config.Config, directors *ipc.TournamentPersons) (*entity.Tournament, error) {
	return tournament.NewTournament(ctx,
		ts,
		tournamentName,
		"This is a test Tournament",
		directors,
		entity.TypeStandard,
		"",
		"/tournament/slug-tourney",
	)
}

func makeTournamentPersons(persons map[string]int32) *ipc.TournamentPersons {
	tp := &ipc.TournamentPersons{}
	for key, value := range persons {
		tp.Persons = append(tp.Persons, &ipc.TournamentPerson{Id: key, Rating: value})
	}
	return tp
}

func makeRoundControls() []*ipc.RoundControl {
	return []*ipc.RoundControl{{FirstMethod: ipc.FirstMethod_AUTOMATIC_FIRST,
		PairingMethod:               ipc.PairingMethod_ROUND_ROBIN,
		GamesPerRound:               1,
		Factor:                      1,
		MaxRepeats:                  1,
		AllowOverMaxRepeats:         true,
		RepeatRelativeWeight:        1,
		WinDifferenceRelativeWeight: 1},
		{FirstMethod: ipc.FirstMethod_AUTOMATIC_FIRST,
			PairingMethod:               ipc.PairingMethod_ROUND_ROBIN,
			GamesPerRound:               1,
			Factor:                      1,
			MaxRepeats:                  1,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1},
		{FirstMethod: ipc.FirstMethod_AUTOMATIC_FIRST,
			PairingMethod:               ipc.PairingMethod_ROUND_ROBIN,
			GamesPerRound:               1,
			Factor:                      1,
			MaxRepeats:                  1,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1},
		{FirstMethod: ipc.FirstMethod_AUTOMATIC_FIRST,
			PairingMethod:               ipc.PairingMethod_KING_OF_THE_HILL,
			GamesPerRound:               1,
			Factor:                      1,
			MaxRepeats:                  1,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1}}
}

func makeControls() *ipc.DivisionControls {
	return &ipc.DivisionControls{
		SuspendedResult: ipc.TournamentGameResult_BYE,
		GameRequest:     gameReq,
		AutoStart:       true}
}

func createBus() (context.Context, *bus.Bus, context.CancelFunc, bus.Stores, *config.Config) {

	recreateDBManyUsers()
	redisPool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
		Dial: func() (redis.Conn, error) { return redis.DialURL(os.Getenv("REDIS_URL")) },
	}

	stores := bus.Stores{}
	stores.UserStore = userStore()
	_, gs := gameStore(stores.UserStore)
	stores.GameStore = gs
	cfg, tstore := tournamentStore(gs)
	stores.TournamentStore = tstore
	var err error
	stores.SoughtGameStore, err = soughtgame.NewDBStore(cfg)

	stores.PresenceStore = pkgredis.NewRedisPresenceStore(redisPool)
	stores.ChatStore = pkgredis.NewRedisChatStore(redisPool, stores.PresenceStore, stores.TournamentStore, nil)
	stores.ConfigStore = cfgstore.NewRedisConfigStore(redisPool)

	if err != nil {
		panic(err)
	}
	natsconn, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		panic(err)
	}
	pubsubBus, err := bus.NewBus(cfg, natsconn, stores, redisPool)
	if err != nil {
		panic(err)
	}
	tournamentService := tournament.NewTournamentService(stores.TournamentStore, stores.UserStore)
	tournamentService.SetEventChannel(pubsubBus.TournamentEventChannel())

	ctx, pubsubCancel := context.WithCancel(context.Background())
	return ctx, pubsubBus, pubsubCancel, stores, cfg
}

func TestLargeTournamentProfile(t *testing.T) {
	// go test -tags integ -run  TestLargeTournamentProfile -memprofile mem.out -cpuprofile cpu.out
	is := is.New(t)

	ctx, bus, cancel, stores, cfg := createBus()
	go bus.ProcessMessages(ctx)

	// Sleep while the bus starts.
	time.Sleep(2 * time.Second)

	directors := makeTournamentPersons(map[string]int32{"Kieran:Kieran": 0})

	// Create player map
	pmap := map[string]int32{}
	for i := 0; i < NumUsers; i++ {
		pmap[fmt.Sprintf("Player%d", i+1)] = int32((i + 1) * 10)
	}
	players := makeTournamentPersons(pmap)

	ty, err := makeTournament(ctx, stores.TournamentStore, cfg, directors)
	is.NoErr(err)

	meta := &pb.TournamentMetadata{
		Id:          ty.UUID,
		Name:        tournamentName,
		Description: "New Description",
		Slug:        "/tournament/foo",
		Type:        pb.TType_STANDARD,
	}
	tstore := stores.TournamentStore
	ustore := stores.UserStore
	err = tournament.SetTournamentMetadata(ctx, tstore, meta)
	is.NoErr(err)

	const divOneName = "Division 1"

	// Add a division
	err = tournament.AddDivision(ctx, tstore, ty.UUID, divOneName)
	is.NoErr(err)

	err = tournament.AddPlayers(ctx, tstore, ustore, ty.UUID, divOneName, players)
	is.NoErr(err)
	div1 := ty.Divisions[divOneName]
	_, err = div1.DivisionManager.GetXHRResponse()
	is.NoErr(err)

	err = tournament.SetDivisionControls(ctx, tstore, ty.UUID, divOneName, makeControls())
	is.NoErr(err)

	err = tournament.SetRoundControls(ctx, tstore, ty.UUID, divOneName, makeRoundControls())
	is.NoErr(err)

	// Start the tournament.

	err = tournament.StartAllRoundCountdowns(ctx, tstore, ty.UUID, 0)
	is.NoErr(err)

	natsconn, err := nats.Connect(cfg.NatsURL)
	is.NoErr(err)

	go func() {
		// Simulate everyone pressing Ready within a few seconds.
		for i := 0; i < NumUsers; i++ {
			uid := fmt.Sprintf("uuid-%d", i+1)
			msg := &ipc.ReadyForTournamentGame{
				TournamentId: ty.UUID,
				Division:     divOneName,
				Round:        0,
				PlayerId:     uid,
			}
			bts, err := proto.Marshal(msg)
			is.NoErr(err)
			topic := fmt.Sprintf("ipc.pb.%d.%s.%s.%s",
				ipc.MessageType_READY_FOR_TOURNAMENT_GAME.Number(),
				"auth",
				uid,
				shortuuid.New()[2:10], // a connection id
			)
			natsconn.Publish(topic, bts)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// exit bus cleanly
	idleConnsClosed := make(chan struct{})
	go func() {
		time.Sleep(10*time.Millisecond*NumUsers + (2 * time.Second))
		cancel()
		close(idleConnsClosed)
	}()
	<-idleConnsClosed
	ct, err := stores.GameStore.Count(ctx)
	is.NoErr(err)
	is.Equal(int(ct), NumUsers>>1)
}
