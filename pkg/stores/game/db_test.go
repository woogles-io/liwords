package game

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lithammer/shortuuid/v4"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/macondo/board"
	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/user"
	pkguser "github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

var pkg = "game"
var DefaultConfig = config.DefaultConfig()

func init() {
	DefaultConfig.DBConnDSN = common.TestingPostgresConnDSN(pkg)
}

func newMacondoGame(users [2]*entity.User) *macondogame.Game {
	rules, err := macondogame.NewBasicGameRules(
		DefaultConfig.MacondoConfig(), "NWL20",
		board.CrosswordGameLayout, "english",
		macondogame.CrossScoreOnly, "")
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

func userStore(pool *pgxpool.Pool) pkguser.Store {
	ustore, err := user.NewDBStore(pool)
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	return ustore
}

func recreateDB() (pkguser.Store, *DBStore) {
	err := common.RecreateTestDB(pkg)
	if err != nil {
		panic(err)
	}

	pool, err := common.OpenTestingDB(pkg)
	if err != nil {
		panic(err)
	}

	// Crete a user table. Initialize the user store.
	ustore := userStore(pool)
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

	cfg := DefaultConfig
	cfg.DBConnDSN = common.TestingPostgresConnDSN(pkg)

	gstore, err := NewDBStore(cfg, ustore, pool)
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}

	addfakeGames(gstore)
	return ustore, gstore
}

func addfakeGames(gstore *DBStore) {
	protocts, err := os.ReadFile("./testdata/game1/history.json")
	if err != nil {
		panic(err)
	}
	gh := &macondopb.GameHistory{}
	err = protojson.Unmarshal(protocts, gh)
	if err != nil {
		panic(err)
	}
	histbts, err := proto.Marshal(gh)
	if err != nil {
		panic(err)
	}

	reqbts := []byte(`{
	"rules": {
		"boardLayoutName": "CrosswordGame",
		"letterDistributionName": "english"
	},
	"initialTimeSeconds": 60,
	"challengeRule": "FIVE_POINT"
	}`)
	ctx := context.Background()
	_, err = gstore.dbPool.Exec(ctx, "INSERT INTO games(created_at, updated_at, uuid, "+
		"player0_id, player1_id, timers, started, game_end_reason, winner_idx, loser_idx, "+
		"game_request, history, quickdata) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)",
		"2020-07-27 04:33:45.938304+00", "2020-07-27 04:33:45.938304+00",
		"wJxURccCgSAPivUvj4QdYL", 2, 1,
		`{"lu": 1595824425928, "mo": 0, "tr": [60000, 60000], "ts": 1595824425928}`,
		true, 0, 0, 0, reqbts, histbts,
		`{"pi":[{"nickname":"mina","rating":"1600?"},{"nickname":"cesar","rating":"500?"}]}`)

	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
}

func createGame(ustore pkguser.Store, gstore *DBStore, p0, p1 string, initTime int32, is *is.I) *entity.Game {
	u1, err := ustore.Get(context.Background(), p0)
	is.NoErr(err)

	u2, err := ustore.Get(context.Background(), p1)
	is.NoErr(err)

	mcg := newMacondoGame([2]*entity.User{u1, u2})
	mcg.StartGame()
	// Overwrite Uid to make it liwords-compatible (short). This is probably
	// a code smell to do it here.
	mcg.History().Uid = shortuuid.New()[2:10]
	mcg.SetChallengeRule(macondopb.ChallengeRule_FIVE_POINT)
	mcg.SetBackupMode(macondogame.InteractiveGameplayMode)
	mcg.SetStateStackLength(1)
	entGame := entity.NewGame(mcg, &pb.GameRequest{
		InitialTimeSeconds: initTime,
		ChallengeRule:      macondopb.ChallengeRule_FIVE_POINT,
		Rules: &pb.GameRules{
			BoardLayoutName:        "CrosswordGame",
			LetterDistributionName: "english",
		},
	})
	entGame.PlayerDBIDs = [2]uint{u1.ID, u2.ID}
	entGame.Quickdata = &entity.Quickdata{
		PlayerInfo: []*pb.PlayerInfo{
			{Nickname: u1.Username, Rating: "1500?"},
			{Nickname: u2.Username, Rating: "1500?"},
		},
	}
	entGame.ResetTimersAndStart()

	err = gstore.Create(context.Background(), entGame)
	is.NoErr(err)
	return entGame
}

func TestCreate(t *testing.T) {
	log.Info().Msg("TestCreate")
	is := is.New(t)

	ustore, gstore := recreateDB()

	entGame := createGame(ustore, gstore, "cesar", "mina", int32(60), is)

	is.True(entGame.Quickdata != nil)

	// Make sure we can fetch the game from the DB.
	log.Debug().Str("entGameID", entGame.GameID()).Msg("trying-to-fetch")
	cpy, err := gstore.Get(context.Background(), entGame.GameID())
	is.NoErr(err)
	is.True(cpy.Quickdata != nil)

	// Clean up connections
	ustore.(*user.DBStore).Disconnect()
	gstore.Disconnect()

}

func TestSet(t *testing.T) {
	log.Info().Msg("TestSet")
	ustore, gstore := recreateDB()

	is := is.New(t)

	// Fetch the game from the backend.
	entGame, err := gstore.Get(context.Background(), "wJxURccCgSAPivUvj4QdYL")
	is.NoErr(err)
	// Make some modification.

	log.Debug().Interface("history", entGame.History()).Msg("test-set")
	is.Equal(entGame.History().ChallengeRule, macondopb.ChallengeRule_FIVE_POINT)
	is.Equal(entGame.NickOnTurn(), "mina")
	is.Equal(entGame.Turn(), 0)

	_, err = entGame.PlayScoringMove("8E", "AGUE", true)
	is.NoErr(err)
	// Save it back
	err = gstore.Set(context.Background(), entGame)
	is.NoErr(err)

	// Now, fetch the game again and see if things have updated.
	g, err := gstore.Get(context.Background(), "wJxURccCgSAPivUvj4QdYL")
	is.NoErr(err)
	is.Equal(g.Turn(), 1)
	is.Equal(g.NickOnTurn(), "cesar")
	is.Equal(g.RackLettersFor(1), "AEIJVVW")
	// AGUE was worth 10. The spread for cesar is therefore -10.
	is.Equal(g.SpreadFor(1), -10)

	// Clean up connections
	ustore.(*user.DBStore).Disconnect()
	gstore.Disconnect()
}

func TestGet(t *testing.T) {
	log.Info().Msg("TestGet")
	ustore, gstore := recreateDB()
	is := is.New(t)

	entGame, err := gstore.Get(context.Background(), "wJxURccCgSAPivUvj4QdYL")
	is.NoErr(err)
	log.Info().Interface("entGame history", entGame.History()).Msg("history")

	mina, err := ustore.Get(context.Background(), "mina")
	is.NoErr(err)
	log.Debug().Interface("mina", mina).Msg("playerinfo")

	is.Equal(entGame.GameID(), "wJxURccCgSAPivUvj4QdYL")
	is.Equal(entGame.NickOnTurn(), "mina")
	is.Equal(entGame.PlayerIDOnTurn(), mina.UUID)
	is.Equal(entGame.RackLettersFor(0), "AEEGOUU")
	is.Equal(entGame.RackLettersFor(1), "AEIJVVW")
	is.Equal(entGame.ChallengeRule(), macondopb.ChallengeRule_FIVE_POINT)
	is.Equal(entGame.History().ChallengeRule, macondopb.ChallengeRule_FIVE_POINT)
	// Clean up connections
	ustore.(*user.DBStore).Disconnect()
	gstore.Disconnect()
}

func TestListActive(t *testing.T) {
	log.Info().Msg("TestListActive")
	ustore, gstore := recreateDB()
	is := is.New(t)

	createGame(ustore, gstore, "cesar", "jesse", int32(120), is)
	createGame(ustore, gstore, "jesse", "mina", int32(240), is)

	// There should be an additional game, so 3 total, from recreateDB()
	// The first game is cesar vs mina. (see TestGet)

	games, err := gstore.ListActive(context.Background(), "", true)
	is.NoErr(err)
	is.Equal(len(games.GameInfo), 3)
	is.Equal(games.GameInfo[0].Players, []*pb.PlayerInfo{
		{Rating: "1600?", Nickname: "mina"},
		{Rating: "500?", Nickname: "cesar"},
	})
	is.Equal(games.GameInfo[1].Players, []*pb.PlayerInfo{
		{Rating: "1500?", Nickname: "cesar"},
		{Rating: "1500?", Nickname: "jesse"},
	})
	is.Equal(games.GameInfo[2].Players, []*pb.PlayerInfo{
		{Rating: "1500?", Nickname: "jesse"},
		{Rating: "1500?", Nickname: "mina"},
	})

	is.Equal(games.GameInfo[1].GameRequest.InitialTimeSeconds, int32(120))
	is.Equal(games.GameInfo[1].GameRequest.ChallengeRule, macondopb.ChallengeRule_FIVE_POINT)
	is.True(proto.Equal(games.GameInfo[1].GameRequest.Rules, &pb.GameRules{
		BoardLayoutName:        "CrosswordGame",
		LetterDistributionName: "english",
	}))
	ustore.(*user.DBStore).Disconnect()
	gstore.Disconnect()
}
