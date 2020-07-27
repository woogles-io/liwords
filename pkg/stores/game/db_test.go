package game

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/domino14/liwords/pkg/stores/user"
	pkguser "github.com/domino14/liwords/pkg/user"

	"github.com/domino14/liwords/pkg/entity"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/domino14/macondo/board"
	macondoconfig "github.com/domino14/macondo/config"
	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/jinzhu/gorm"
	"github.com/matryer/is"
)

var DefaultConfig = macondoconfig.Config{
	StrategyParamsPath:        os.Getenv("STRATEGY_PARAMS_PATH"),
	LexiconPath:               os.Getenv("LEXICON_PATH"),
	LetterDistributionPath:    os.Getenv("LETTER_DISTRIBUTION_PATH"),
	DefaultLexicon:            "NWL18",
	DefaultLetterDistribution: "English",
}

var TestDBHost = os.Getenv("TEST_DB_HOST")

var TestingDBConnStr = "host=" + TestDBHost + " port=5432 user=postgres password=pass sslmode=disable"

func newMacondoGame(users [2]*entity.User) *macondogame.Game {
	rules, err := macondogame.NewGameRules(&DefaultConfig, board.CrosswordGameBoard,
		DefaultConfig.DefaultLexicon, DefaultConfig.DefaultLetterDistribution)
	if err != nil {
		panic(err)
	}
	var players []*macondopb.PlayerInfo

	for _, u := range users {
		players = append(players, &macondopb.PlayerInfo{
			Nickname: u.Username,
			UserId:   u.UUID,
			RealName: u.RealName(),
		})
	}

	mcg, err := macondogame.NewGame(rules, players)
	if err != nil {
		panic(err)
	}
	return mcg
}

func userStore(dbURL string) pkguser.Store {
	ustore, err := user.NewDBStore(TestingDBConnStr + " dbname=liwords_test")
	if err != nil {
		log.Fatal(err)
	}
	return ustore
}

func setup() {
	// Create a database.
	db, err := gorm.Open("postgres", TestingDBConnStr+" dbname=postgres")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db = db.Exec("DROP DATABASE IF EXISTS liwords_test")
	if db.Error != nil {
		log.Fatal(db.Error)
	}
	db = db.Exec("CREATE DATABASE liwords_test")
	if db.Error != nil {
		log.Fatal(db.Error)
	}
	// Create a user table. Initialize the user store.
	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")

	// Insert a couple of users into the table.

	for _, u := range []*entity.User{
		&entity.User{Username: "cesar", Email: "cesar@woogles.io"},
		&entity.User{Username: "mina", Email: "mina@gmail.com"},
		&entity.User{Username: "jesse", Email: "jesse@woogles.io"},
	} {
		err = ustore.New(context.Background(), u)
		if err != nil {
			log.Fatal(err)
		}
	}
	protocts, err := ioutil.ReadFile("./testdata/game1/history.pb")
	if err != nil {
		log.Fatal(err)
	}

	protocts, err = hex.DecodeString(string(protocts))
	if err != nil {
		log.Fatal(err)
	}

	req, err := hex.DecodeString("183c2803")
	if err != nil {
		log.Fatal(err)
	}

	// Add some fake games to the table
	store, err := NewDBStore(TestingDBConnStr+" dbname=liwords_test", nil, ustore)
	db = store.db.Exec("INSERT INTO games(created_at, updated_at, uuid, "+
		"player0_id, player1_id, timers, started, game_end_reason, winner_idx, loser_idx, "+
		"request, history) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		"2020-07-27 04:33:45.938304+00", "2020-07-27 04:33:45.938304+00",
		"hvxXXKPBeFDQAUcoKwtreB", 1, 3, `{"lu": 1595824425928, "mo": 0, "tr": [60000, 60000], "ts": 1595824425928}`,
		true, 0, 0, 0, req, protocts)

	if db.Error != nil {
		log.Fatal(db.Error)
	}
	ustore.(*user.DBStore).Disconnect()
	store.Disconnect()
}

func teardown() {
	db, err := gorm.Open("postgres", TestingDBConnStr+" dbname=postgres")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db = db.Exec("DROP DATABASE liwords_test")
	if db.Error != nil {
		log.Fatal(db.Error)
	}
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	//teardown()
	os.Exit(code)
}

func TestCreate(t *testing.T) {
	is := is.New(t)
	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")
	store, err := NewDBStore(TestingDBConnStr+" dbname=liwords_test", nil, ustore)
	is.NoErr(err)

	u1, err := ustore.Get(context.Background(), "cesar")
	is.NoErr(err)

	u2, err := ustore.Get(context.Background(), "mina")
	is.NoErr(err)

	mcg := newMacondoGame([2]*entity.User{u1, u2})
	mcg.StartGame()
	mcg.SetBackupMode(macondogame.InteractiveGameplayMode)
	entGame := entity.NewGame(mcg, &pb.GameRequest{InitialTimeSeconds: 60, ChallengeRule: macondopb.ChallengeRule_FIVE_POINT})
	entGame.ResetTimersAndStart()

	fmt.Println("entGame history", entGame.History(), entGame.Game.Playing())
	err = store.Create(context.Background(), entGame)
	is.NoErr(err)
	// Clean up connections
	ustore.(*user.DBStore).Disconnect()
	store.Disconnect()
}

func TestGet(t *testing.T) {
	is := is.New(t)

	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")
	store, err := NewDBStore(TestingDBConnStr+" dbname=liwords_test", nil, ustore)
	is.NoErr(err)

	entGame, err := store.Get(context.Background(), "hvxXXKPBeFDQAUcoKwtreB")
	is.NoErr(err)
	is.Equal(entGame.GameID(), "hvxXXKPBeFDQAUcoKwtreB")
	// Clean up connections
	ustore.(*user.DBStore).Disconnect()
	store.Disconnect()
}
