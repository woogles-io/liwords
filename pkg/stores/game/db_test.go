package game

import (
	"context"
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/domino14/macondo/board"
	macondoconfig "github.com/domino14/macondo/config"
	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/user"
	pkguser "github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
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
		log.Fatal().Err(err).Msg("error")
	}
	return ustore
}

func setup() {
	// Create a database.
	db, err := gorm.Open("postgres", TestingDBConnStr+" dbname=postgres")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	defer db.Close()
	db = db.Exec("DROP DATABASE IF EXISTS liwords_test")
	if db.Error != nil {
		log.Fatal().Err(db.Error).Msg("error")
	}
	db = db.Exec("CREATE DATABASE liwords_test")
	if db.Error != nil {
		log.Fatal().Err(db.Error).Msg("error")
	}
	// Create a user table. Initialize the user store.
	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")

	// Insert a couple of users into the table.

	for _, u := range []*entity.User{
		{Username: "cesar", Email: "cesar@woogles.io", UUID: "mozEwaVMvTfUA2oxZfYN8k"},
		{Username: "mina", Email: "mina@gmail.com", UUID: "iW7AaqNJDuaxgcYnrFfcJF"},
		{Username: "jesse", Email: "jesse@woogles.io", UUID: "3xpEkpRAy3AizbVmDg3kdi"},
	} {
		err = ustore.New(context.Background(), u)
		if err != nil {
			log.Fatal().Err(err).Msg("error")
		}
	}
	protocts, err := ioutil.ReadFile("./testdata/game1/history.pb")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}

	protocts, err = hex.DecodeString(string(protocts))
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}

	req, err := hex.DecodeString("12180a0d43726f7373776f726447616d651207656e676c697368183c2803")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}

	// Add some fake games to the table
	store, err := NewDBStore(&config.Config{
		DBConnString: TestingDBConnStr + " dbname=liwords_test"}, ustore)
	db = store.db.Exec("INSERT INTO games(created_at, updated_at, uuid, "+
		"player0_id, player1_id, timers, started, game_end_reason, winner_idx, loser_idx, "+
		"request, history) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		"2020-07-27 04:33:45.938304+00", "2020-07-27 04:33:45.938304+00",
		"wJxURccCgSAPivUvj4QdYL", 1, 3, `{"lu": 1595824425928, "mo": 0, "tr": [60000, 60000], "ts": 1595824425928}`,
		true, 0, 0, 0, req, protocts)

	if db.Error != nil {
		log.Fatal().Err(db.Error).Msg("error")
	}
	ustore.(*user.DBStore).Disconnect()
	store.Disconnect()
}

func teardown() {
	db, err := gorm.Open("postgres", TestingDBConnStr+" dbname=postgres")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	defer db.Close()
	db = db.Exec("DROP DATABASE liwords_test")
	if db.Error != nil {
		log.Fatal().Err(db.Error).Msg("error")
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
	store, err := NewDBStore(&config.Config{
		DBConnString: TestingDBConnStr + " dbname=liwords_test"}, ustore)
	is.NoErr(err)

	u1, err := ustore.Get(context.Background(), "cesar")
	is.NoErr(err)

	u2, err := ustore.Get(context.Background(), "mina")
	is.NoErr(err)

	mcg := newMacondoGame([2]*entity.User{u1, u2})
	mcg.StartGame()
	mcg.SetChallengeRule(macondopb.ChallengeRule_FIVE_POINT)
	mcg.SetBackupMode(macondogame.InteractiveGameplayMode)
	entGame := entity.NewGame(mcg, &pb.GameRequest{
		InitialTimeSeconds: 60,
		ChallengeRule:      macondopb.ChallengeRule_FIVE_POINT,
		Rules: &pb.GameRules{
			BoardLayoutName:        "CrosswordGame",
			LetterDistributionName: "english",
		},
	})
	entGame.ResetTimersAndStart()

	err = store.Create(context.Background(), entGame)
	is.NoErr(err)
	// Clean up connections
	ustore.(*user.DBStore).Disconnect()
	store.Disconnect()
}

func TestSet(t *testing.T) {
	is := is.New(t)
	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")
	store, err := NewDBStore(&config.Config{
		MacondoConfig: DefaultConfig,
		DBConnString:  TestingDBConnStr + " dbname=liwords_test"}, ustore)
	is.NoErr(err)

	// Fetch the game from the backend.
	entGame, err := store.Get(context.Background(), "wJxURccCgSAPivUvj4QdYL")
	// Make some modification.
	is.Equal(entGame.History().ChallengeRule, macondopb.ChallengeRule_FIVE_POINT)
	is.Equal(entGame.NickOnTurn(), "mina")
	is.Equal(entGame.Turn(), 0)

	_, err = entGame.PlayScoringMove("8E", "AGUE", true)
	is.NoErr(err)
	// Save it back
	err = store.Set(context.Background(), entGame)
	is.NoErr(err)

	// Now, fetch the game again and see if things have updated.
	g, err := store.Get(context.Background(), "wJxURccCgSAPivUvj4QdYL")
	is.NoErr(err)
	is.Equal(g.Turn(), 1)
	is.Equal(g.NickOnTurn(), "cesar")
	// cesar is player 0, mina is player 1, but mina went first because of
	// the second_went_first flag in the history.
	is.Equal(g.RackLettersFor(0), "AEIJVVW")
	// AGUE was worth 10. The spread for cesar is therefore -10.
	is.Equal(g.SpreadFor(0), -10)
}

func TestGet(t *testing.T) {
	is := is.New(t)

	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")
	store, err := NewDBStore(&config.Config{
		MacondoConfig: DefaultConfig,
		DBConnString:  TestingDBConnStr + " dbname=liwords_test",
	}, ustore)
	is.NoErr(err)

	entGame, err := store.Get(context.Background(), "wJxURccCgSAPivUvj4QdYL")
	is.NoErr(err)
	log.Debug().Interface("entGame history", entGame.History()).Msg("history")

	mina, err := ustore.Get(context.Background(), "mina")
	is.NoErr(err)
	log.Debug().Interface("mina", mina).Msg("playerinfo")

	is.Equal(entGame.GameID(), "wJxURccCgSAPivUvj4QdYL")
	is.Equal(entGame.NickOnTurn(), "mina")
	is.Equal(entGame.PlayerIDOnTurn(), mina.UUID)
	is.Equal(entGame.RackLettersFor(0), "AEIJVVW")
	is.Equal(entGame.RackLettersFor(1), "AEEGOUU")
	is.Equal(entGame.ChallengeRule(), macondopb.ChallengeRule_FIVE_POINT)
	is.Equal(entGame.History().ChallengeRule, macondopb.ChallengeRule_FIVE_POINT)

	// Clean up connections
	ustore.(*user.DBStore).Disconnect()
	store.Disconnect()
}
