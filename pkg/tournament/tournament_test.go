package tournament_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/stores/game"
	ts "github.com/domino14/liwords/pkg/stores/tournament"
	"github.com/domino14/liwords/pkg/stores/user"
	"github.com/domino14/liwords/pkg/tournament"
	pkguser "github.com/domino14/liwords/pkg/user"
	ipc "github.com/domino14/liwords/rpc/api/proto/ipc"
	pb "github.com/domino14/liwords/rpc/api/proto/tournament_service"
	macondoconfig "github.com/domino14/macondo/config"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

var tournamentName = "testTournament"
var TestDBHost = os.Getenv("TEST_DB_HOST")
var TestingDBConnStr = "host=" + TestDBHost + " port=5432 user=postgres password=pass sslmode=disable"
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

var DefaultConfig = macondoconfig.Config{
	LexiconPath:               os.Getenv("LEXICON_PATH"),
	LetterDistributionPath:    os.Getenv("LETTER_DISTRIBUTION_PATH"),
	DefaultLexicon:            "CSW21",
	DefaultLetterDistribution: "English",
}

var divOneName = "Division 1"
var divTwoName = "Division 2"

func recreateDB() {
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

	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")

	for _, u := range []*entity.User{
		{Username: "Will", Email: "cesar@woogles.io", UUID: "Will"},
		{Username: "Josh", Email: "mina@gmail.com", UUID: "Josh"},
		{Username: "Conrad", Email: "crad@woogles.io", UUID: "Conrad"},
		{Username: "Jesse", Email: "jesse@woogles.io", UUID: "Jesse"},
		{Username: "Kieran", Email: "kieran@woogles.io", UUID: "Kieran"},
		{Username: "Vince", Email: "vince@woogles.io", UUID: "Vince"},
		{Username: "Jennifer", Email: "jenn@woogles.io", UUID: "Jennifer"},
		{Username: "Guy", Email: "guy@woogles.io", UUID: "Guy"},
		{Username: "Evans", Email: "evans@woogles.io", UUID: "Evans"},
		{Username: "Bob", Email: "bob@woogles.io", UUID: "Bob"},
		{Username: "Noah", Email: "noah@woogles.io", UUID: "Noah"},
		{Username: "Zoof", Email: "zoof@woogles.io", UUID: "Zoof"},
		{Username: "Harry", Email: "harry@woogles.io", UUID: "Harry"},
		{Username: "Oof", Email: "oof@woogles.io", UUID: "Oof"},
		{Username: "Dude", Email: "dude@woogles.io", UUID: "Dude"},
		{Username: "Comrade", Email: "comrade@woogles.io", UUID: "Comrade"},
		{Username: "ValuedCustomer", Email: "valued@woogles.io", UUID: "ValuedCustomer"},
	} {
		err = ustore.New(context.Background(), u)
		if err != nil {
			log.Fatal().Err(err).Msg("error")
		}
	}
	ustore.(*user.DBStore).Disconnect()
}

func tournamentStore(dbURL string, gs gameplay.GameStore) (*config.Config, tournament.TournamentStore) {
	cfg := &config.Config{}
	cfg.MacondoConfig = DefaultConfig
	cfg.DBConnString = dbURL

	tmp, err := ts.NewDBStore(cfg, gs)
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	tournamentStore := ts.NewCache(tmp)
	return cfg, tournamentStore
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

func userStore(dbURL string) pkguser.Store {
	ustore, err := user.NewDBStore(TestingDBConnStr + " dbname=liwords_test")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	return ustore
}

func gameStore(dbURL string, userStore pkguser.Store) (*config.Config, gameplay.GameStore) {
	cfg := &config.Config{}
	cfg.MacondoConfig = DefaultConfig
	cfg.DBConnString = dbURL

	tmp, err := game.NewDBStore(cfg, userStore)
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	gameStore := game.NewCache(tmp)
	return cfg, gameStore
}

func TestTournamentSingleDivision(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	cstr := TestingDBConnStr + " dbname=liwords_test"
	recreateDB()
	us := userStore(cstr)
	_, gs := gameStore(cstr, us)
	cfg, tstore := tournamentStore(cstr, gs)

	players := makeTournamentPersons(map[string]int32{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100})
	directors := makeTournamentPersons(map[string]int32{"Kieran:Kieran": 0, "Vince:Vince": 2, "Jennifer:Jennifer": 2})
	directorsTwoExecutives := makeTournamentPersons(map[string]int32{"Kieran:Kieran": 0, "Vince:Vince": 0, "Jennifer:Jennifer": 2})
	directorsNoExecutives := makeTournamentPersons(map[string]int32{"Kieran:Kieran": 1, "Vince:Vince": 3, "Jennifer:Jennifer": 2})

	_, err := makeTournament(ctx, tstore, cfg, directorsTwoExecutives)
	is.True(err != nil)

	_, err = makeTournament(ctx, tstore, cfg, directorsNoExecutives)
	is.True(err != nil)

	ty, err := makeTournament(ctx, tstore, cfg, directors)
	is.NoErr(err)

	meta := &pb.TournamentMetadata{
		Id:          ty.UUID,
		Name:        tournamentName,
		Description: "New Description",
		Slug:        "/tournament/foo",
		Type:        pb.TType_STANDARD,
	}

	err = tournament.SetTournamentMetadata(ctx, tstore, meta)
	is.NoErr(err)

	// Check that directors are set correctly
	is.NoErr(equalTournamentPersons(directors, ty.Directors))

	// Attempt to remove a division that doesn't exist in the empty tournament
	err = tournament.RemoveDivision(ctx, tstore, ty.UUID, "The Big Boys")
	is.True(err != nil)

	// Add a division
	err = tournament.AddDivision(ctx, tstore, ty.UUID, divOneName)
	is.NoErr(err)

	// Attempt to remove a division that doesn't exist when other
	// divisions are present
	err = tournament.RemoveDivision(ctx, tstore, ty.UUID, "Nope")
	is.True(err != nil)

	// Attempt to add a division that already exists
	err = tournament.AddDivision(ctx, tstore, ty.UUID, divOneName)
	is.True(err != nil)

	// Attempt to add directors that already exist
	err = tournament.AddDirectors(ctx, tstore, us, ty.UUID, makeTournamentPersons(map[string]int32{"Guy": 1, "Vince": 2}))
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(directors, ty.Directors))

	// Attempt to add another executive director
	err = tournament.AddDirectors(ctx, tstore, us, ty.UUID, makeTournamentPersons(map[string]int32{"Guy": 1, "Harry": 0}))
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(directors, ty.Directors))

	// Remove Vince and Jennifer
	err = tournament.RemoveDirectors(ctx, tstore, us, ty.UUID, makeTournamentPersons(map[string]int32{"Vince": -1, "Jennifer": -1}))
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{"Kieran:Kieran": 0}), ty.Directors))

	// Add directors
	err = tournament.AddDirectors(ctx, tstore, us, ty.UUID, makeTournamentPersons(map[string]int32{"Evans": 4, "Oof": 2, "Vince": 2, "Guy": 10, "Harry": 11, "Jennifer": 2}))
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{"Kieran:Kieran": 0, "Vince:Vince": 2, "Jennifer:Jennifer": 2, "Evans:Evans": 4, "Oof:Oof": 2, "Guy:Guy": 10, "Harry:Harry": 11}), ty.Directors))

	// Attempt to remove directors that don't exist
	err = tournament.RemoveDirectors(ctx, tstore, us, ty.UUID, makeTournamentPersons(map[string]int32{"Evans": -1, "Zoof": 2}))
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_PLAYER, tournamentName, "", "0", "Zoof:Zoof", "removeTournamentPersons").Error())
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{"Kieran:Kieran": 0, "Vince:Vince": 2, "Jennifer:Jennifer": 2, "Evans:Evans": 4, "Oof:Oof": 2, "Guy:Guy": 10, "Harry:Harry": 11}), ty.Directors))

	// Attempt to remove the executive director
	err = tournament.RemoveDirectors(ctx, tstore, us, ty.UUID, makeTournamentPersons(map[string]int32{"Evans": -1, "Kieran": 0}))
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_EXECUTIVE_DIRECTOR_REMOVAL, tournamentName, "", "Kieran:Kieran").Error())
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{"Kieran:Kieran": 0, "Vince:Vince": 2, "Jennifer:Jennifer": 2, "Evans:Evans": 4, "Oof:Oof": 2, "Guy:Guy": 10, "Harry:Harry": 11}), ty.Directors))

	// Remove directors
	err = tournament.RemoveDirectors(ctx, tstore, us, ty.UUID, makeTournamentPersons(map[string]int32{"Evans": -1, "Oof": 2, "Guy": -5, "Harry": 300}))
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{"Kieran:Kieran": 0, "Vince:Vince": 2, "Jennifer:Jennifer": 2}), ty.Directors))

	// Attempt to remove the executive director
	err = tournament.RemoveDirectors(ctx, tstore, us, ty.UUID, makeTournamentPersons(map[string]int32{"Vince": -1, "Kieran": 0}))
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_EXECUTIVE_DIRECTOR_REMOVAL, tournamentName, "", "Kieran:Kieran").Error())
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{"Kieran:Kieran": 0, "Vince:Vince": 2, "Jennifer:Jennifer": 2}), ty.Directors))

	// Same thing for players.
	div1 := ty.Divisions[divOneName]

	// Add players
	err = tournament.AddPlayers(ctx, tstore, us, ty.UUID, divOneName, players)
	is.NoErr(err)
	XHRResponse, err := div1.DivisionManager.GetXHRResponse()
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{"Will:Will": 1000, "Josh:Josh": 3000, "Conrad:Conrad": 2200, "Jesse:Jesse": 2100}), XHRResponse.Players))

	// Add players to a division that doesn't exist
	err = tournament.AddPlayers(ctx, tstore, us, ty.UUID, divOneName+"not quite", makeTournamentPersons(map[string]int32{"Noah": 4, "Bob": 2}))
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, tournamentName, divOneName+"not quite").Error())
	XHRResponse, err = div1.DivisionManager.GetXHRResponse()
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{"Will:Will": 1000, "Josh:Josh": 3000, "Conrad:Conrad": 2200, "Jesse:Jesse": 2100}), XHRResponse.Players))

	// Add players
	err = tournament.AddPlayers(ctx, tstore, us, ty.UUID, divOneName, makeTournamentPersons(map[string]int32{"Noah": 4, "Bob": 2}))
	is.NoErr(err)
	XHRResponse, err = div1.DivisionManager.GetXHRResponse()
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{"Will:Will": 1000, "Josh:Josh": 3000, "Conrad:Conrad": 2200, "Jesse:Jesse": 2100, "Noah:Noah": 4, "Bob:Bob": 2}), XHRResponse.Players))

	// Remove players that don't exist
	err = tournament.RemovePlayers(ctx, tstore, us, ty.UUID, divOneName, makeTournamentPersons(map[string]int32{"Evans": -1}))
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_PLAYER, tournamentName, divOneName, "0", "Evans:Evans", "removePlayers").Error())
	XHRResponse, err = div1.DivisionManager.GetXHRResponse()
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{"Will:Will": 1000, "Josh:Josh": 3000, "Conrad:Conrad": 2200, "Jesse:Jesse": 2100, "Noah:Noah": 4, "Bob:Bob": 2}), XHRResponse.Players))

	// Remove players from a division that doesn't exist
	err = tournament.RemovePlayers(ctx, tstore, us, ty.UUID, divOneName+"hmm", makeTournamentPersons(map[string]int32{"Josh": -1, "Conrad": 2}))
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, tournamentName, divOneName+"hmm").Error())
	XHRResponse, err = div1.DivisionManager.GetXHRResponse()
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{"Will:Will": 1000, "Josh:Josh": 3000, "Conrad:Conrad": 2200, "Jesse:Jesse": 2100, "Noah:Noah": 4, "Bob:Bob": 2}), XHRResponse.Players))

	// Remove players
	err = tournament.RemovePlayers(ctx, tstore, us, ty.UUID, divOneName, makeTournamentPersons(map[string]int32{"Josh": -1, "Conrad": 2}))
	is.NoErr(err)
	XHRResponse, err = div1.DivisionManager.GetXHRResponse()
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{"Will:Will": 1000, "Jesse:Jesse": 2100, "Noah:Noah": 4, "Bob:Bob": 2}), XHRResponse.Players))

	// Set tournament controls
	err = tournament.SetDivisionControls(ctx, tstore, ty.UUID, divOneName, makeControls())
	is.NoErr(err)

	// Set tournament controls for a division that does not exist
	err = tournament.SetDivisionControls(ctx, tstore, ty.UUID, divOneName+" another one", makeControls())
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, tournamentName, divOneName+" another one").Error())

	// Set division round controls
	err = tournament.SetRoundControls(ctx, tstore, ty.UUID, divOneName, makeRoundControls())
	is.NoErr(err)
	// Tournament should not be started

	isStarted, err := tournament.IsStarted(ctx, tstore, ty.UUID)
	is.NoErr(err)
	is.True(!isStarted)

	_, err = div1.DivisionManager.GetXHRResponse()
	is.NoErr(err)
	// Set pairing should work before the tournament starts
	pairings := []*pb.TournamentPairingRequest{{PlayerOneId: "Will:Will", PlayerTwoId: "Jesse:Jesse", Round: 0}}
	err = tournament.SetPairings(ctx, tstore, ty.UUID, divOneName, pairings)
	is.NoErr(err)

	// Remove players and attempt to set pairings
	err = tournament.RemovePlayers(ctx, tstore, us, ty.UUID, divOneName, makeTournamentPersons(map[string]int32{"Will": 1000, "Jesse": 2100, "Noah": 4, "Bob": 2}))
	is.NoErr(err)
	XHRResponse, err = div1.DivisionManager.GetXHRResponse()
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{}), XHRResponse.Players))

	err = tournament.SetPairings(ctx, tstore, ty.UUID, divOneName, pairings)
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_PLAYER, tournamentName, divOneName, "1", "Will:Will", "playerOne").Error())

	err = tournament.SetResult(ctx,
		tstore,
		us,
		ty.UUID,
		divOneName,
		"Will:Will",
		"Jesse:Jesse",
		500,
		400,
		ipc.TournamentGameResult_WIN,
		ipc.TournamentGameResult_LOSS,
		ipc.GameEndReason_STANDARD,
		0,
		0,
		false,
		nil,
	)
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NOT_STARTED, tournamentName, divOneName).Error())

	// Add players back in
	players = makeTournamentPersons(map[string]int32{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100})
	err = tournament.AddPlayers(ctx, tstore, us, ty.UUID, divOneName, players)
	is.NoErr(err)
	XHRResponse, err = div1.DivisionManager.GetXHRResponse()
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{"Will:Will": 1000, "Josh:Josh": 3000, "Conrad:Conrad": 2200, "Jesse:Jesse": 2100}), XHRResponse.Players))

	// Start the tournament

	err = tournament.StartAllRoundCountdowns(ctx, tstore, ty.UUID, 0)
	is.NoErr(err)

	// Attempt to add a division after the tournament has started
	err = tournament.AddDivision(ctx, tstore, ty.UUID, divOneName+" this time it's different")
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_ADD_DIVISION_AFTER_START, tournamentName, divOneName+" this time it's different").Error())

	// Attempt to remove a division after the tournament has started
	err = tournament.RemoveDivision(ctx, tstore, ty.UUID, divOneName)
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_DIVISION_REMOVAL_AFTER_START, tournamentName, divOneName).Error())

	// Tournament pairings and results are tested in the
	// entity package
	pairings = []*pb.TournamentPairingRequest{{PlayerOneId: "Will:Will", PlayerTwoId: "Jesse:Jesse", Round: 0},
		{PlayerOneId: "Josh:Josh", PlayerTwoId: "Conrad:Conrad", Round: 0}}
	err = tournament.SetPairings(ctx, tstore, ty.UUID, divOneName, pairings)
	is.NoErr(err)

	// Set pairings for division that does not exist
	err = tournament.SetPairings(ctx, tstore, ty.UUID, divOneName+"yeet", pairings)
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, tournamentName, divOneName+"yeet").Error())

	err = tournament.SetResult(ctx,
		tstore,
		us,
		ty.UUID,
		divOneName,
		"Will",
		"Jesse",
		500,
		400,
		ipc.TournamentGameResult_WIN,
		ipc.TournamentGameResult_LOSS,
		ipc.GameEndReason_STANDARD,
		0,
		0,
		false,
		nil)
	is.NoErr(err)

	// Set results for a division that does not exist
	err = tournament.SetResult(ctx,
		tstore,
		us,
		ty.UUID,
		divOneName+"big boi",
		"Will",
		"Jesse",
		500,
		400,
		ipc.TournamentGameResult_WIN,
		ipc.TournamentGameResult_LOSS,
		ipc.GameEndReason_STANDARD,
		0,
		0,
		false,
		nil)
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, tournamentName, divOneName+"big boi").Error())

	isStarted, err = tournament.IsStarted(ctx, tstore, ty.UUID)
	is.NoErr(err)
	is.True(isStarted)

	isRoundComplete, err := tournament.IsRoundComplete(ctx, tstore, ty.UUID, divOneName, 0)
	is.NoErr(err)
	is.True(!isRoundComplete)

	// Complete the round
	err = tournament.SetResult(ctx,
		tstore,
		us,
		ty.UUID,
		divOneName,
		"Josh",
		"Conrad",
		500,
		400,
		ipc.TournamentGameResult_WIN,
		ipc.TournamentGameResult_LOSS,
		ipc.GameEndReason_STANDARD,
		0,
		0,
		false,
		nil)
	is.NoErr(err)

	isRoundComplete, err = tournament.IsRoundComplete(ctx, tstore, ty.UUID, divOneName, 0)
	is.NoErr(err)
	is.True(isRoundComplete)

	// Complete another round to test PairRound

	// Complete the round
	err = tournament.SetResult(ctx,
		tstore,
		us,
		ty.UUID,
		divOneName,
		"Josh",
		"Jesse",
		500,
		400,
		ipc.TournamentGameResult_WIN,
		ipc.TournamentGameResult_LOSS,
		ipc.GameEndReason_STANDARD,
		1,
		0,
		false,
		nil)
	is.NoErr(err)

	err = tournament.SetResult(ctx,
		tstore,
		us,
		ty.UUID,
		divOneName,
		"Will",
		"Conrad",
		500,
		400,
		ipc.TournamentGameResult_WIN,
		ipc.TournamentGameResult_LOSS,
		ipc.GameEndReason_STANDARD,
		1,
		0,
		false,
		nil)
	is.NoErr(err)

	isRoundComplete, err = tournament.IsRoundComplete(ctx, tstore, ty.UUID, divOneName, 1)
	is.NoErr(err)
	is.True(isRoundComplete)

	err = tournament.PairRound(ctx, tstore, ty.UUID, divOneName, -1, true)
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_PAIR_NON_FUTURE_ROUND, tournamentName, divOneName, "0", "3").Error())

	err = tournament.PairRound(ctx, tstore, ty.UUID, divOneName, 5, true)
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_ROUND_NUMBER_OUT_OF_RANGE, tournamentName, divOneName, "6", "PairRound").Error())

	err = tournament.PairRound(ctx, tstore, ty.UUID, divOneName, 0, true)
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_PAIR_NON_FUTURE_ROUND, tournamentName, divOneName, "1", "3").Error())

	err = tournament.PairRound(ctx, tstore, ty.UUID, divOneName, 1, true)
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_PAIR_NON_FUTURE_ROUND, tournamentName, divOneName, "2", "3").Error())

	err = tournament.PairRound(ctx, tstore, ty.UUID, divOneName, 2, true)
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_PAIR_NON_FUTURE_ROUND, tournamentName, divOneName, "3", "3").Error())

	err = tournament.PairRound(ctx, tstore, ty.UUID, divOneName, 3, true)
	is.NoErr(err)

	// See if round is complete for division that does not exist
	_, err = tournament.IsRoundComplete(ctx, tstore, ty.UUID, divOneName+"yah", 0)
	is.True(err.Error() == entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, tournamentName, divOneName+"yah").Error())

	isFinished, err := tournament.IsFinished(ctx, tstore, ty.UUID)
	is.NoErr(err)
	is.True(!isFinished)

	us.(*user.DBStore).Disconnect()
	tstore.(*ts.Cache).Disconnect()
	gs.(*game.Cache).Disconnect()
}

func TestTournamentMultipleDivisions(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	cstr := TestingDBConnStr + " dbname=liwords_test"

	recreateDB()
	us := userStore(cstr)
	_, gs := gameStore(cstr, us)
	cfg, tstore := tournamentStore(cstr, gs)

	divOnePlayers := makeTournamentPersons(map[string]int32{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100})
	divTwoPlayers := makeTournamentPersons(map[string]int32{"Guy": 1000, "Dude": 3000, "Comrade": 2200, "ValuedCustomer": 2100})
	directors := makeTournamentPersons(map[string]int32{"Kieran": 0, "Vince": 2, "Jennifer": 2})

	divOnePlayersCompare := makeTournamentPersons(map[string]int32{"Will:Will": 1000, "Josh:Josh": 3000, "Conrad:Conrad": 2200, "Jesse:Jesse": 2100})
	divTwoPlayersCompare := makeTournamentPersons(map[string]int32{"Guy:Guy": 1000, "Dude:Dude": 3000, "Comrade:Comrade": 2200, "ValuedCustomer:ValuedCustomer": 2100})

	ty, err := makeTournament(ctx, tstore, cfg, directors)
	is.NoErr(err)

	// Add divisions
	err = tournament.AddDivision(ctx, tstore, ty.UUID, divOneName)
	is.NoErr(err)

	err = tournament.AddDivision(ctx, tstore, ty.UUID, divTwoName)
	is.NoErr(err)

	// Set tournament controls
	err = tournament.SetDivisionControls(ctx,
		tstore,
		ty.UUID,
		divOneName,
		makeControls())
	is.NoErr(err)

	err = tournament.SetDivisionControls(ctx,
		tstore,
		ty.UUID,
		divTwoName,
		makeControls())
	is.NoErr(err)

	err = tournament.SetRoundControls(ctx, tstore, ty.UUID, divOneName, makeRoundControls())
	is.NoErr(err)

	err = tournament.SetRoundControls(ctx, tstore, ty.UUID, divTwoName, makeRoundControls())
	is.NoErr(err)

	div1 := ty.Divisions[divOneName]
	div2 := ty.Divisions[divTwoName]

	// Add players
	err = tournament.AddPlayers(ctx, tstore, us, ty.UUID, divOneName, divOnePlayers)
	is.NoErr(err)
	XHRResponse, err := div1.DivisionManager.GetXHRResponse()
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(divOnePlayersCompare, XHRResponse.Players))

	err = tournament.AddPlayers(ctx, tstore, us, ty.UUID, divTwoName, divTwoPlayers)
	is.NoErr(err)
	XHRResponse, err = div2.DivisionManager.GetXHRResponse()
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(divTwoPlayersCompare, XHRResponse.Players))

	pairings := []*pb.TournamentPairingRequest{{PlayerOneId: "Will:Will", PlayerTwoId: "Jesse:Jesse", Round: 0}}
	err = tournament.SetPairings(ctx, tstore, ty.UUID, divOneName, pairings)
	is.NoErr(err)

	pairings = []*pb.TournamentPairingRequest{{PlayerOneId: "Guy:Guy", PlayerTwoId: "Comrade:Comrade", Round: 0}}
	err = tournament.SetPairings(ctx, tstore, ty.UUID, divTwoName, pairings)
	is.NoErr(err)

	pairings = []*pb.TournamentPairingRequest{{PlayerOneId: "Conrad:Conrad", PlayerTwoId: "Josh:Josh", Round: 0}}
	err = tournament.SetPairings(ctx, tstore, ty.UUID, divOneName, pairings)
	is.NoErr(err)

	pairings = []*pb.TournamentPairingRequest{{PlayerOneId: "Dude:Dude", PlayerTwoId: "ValuedCustomer:ValuedCustomer", Round: 0}}
	err = tournament.SetPairings(ctx, tstore, ty.UUID, divTwoName, pairings)
	is.NoErr(err)

	// Start the tournament

	err = tournament.StartAllRoundCountdowns(ctx, tstore, ty.UUID, 0)
	is.NoErr(err)

	err = tournament.SetResult(ctx,
		tstore,
		us,
		ty.UUID,
		divOneName,
		"Will",
		"Jesse",
		500,
		400,
		ipc.TournamentGameResult_WIN,
		ipc.TournamentGameResult_LOSS,
		ipc.GameEndReason_STANDARD,
		0,
		0,
		false,
		nil)
	is.NoErr(err)

	err = tournament.SetResult(ctx,
		tstore,
		us,
		ty.UUID,
		divTwoName,
		"Comrade",
		"Guy",
		500,
		400,
		ipc.TournamentGameResult_WIN,
		ipc.TournamentGameResult_LOSS,
		ipc.GameEndReason_STANDARD,
		0,
		0,
		false,
		nil)
	is.NoErr(err)

	err = tournament.SetResult(ctx,
		tstore,
		us,
		ty.UUID,
		divOneName,
		"Conrad",
		"Josh",
		500,
		400,
		ipc.TournamentGameResult_WIN,
		ipc.TournamentGameResult_LOSS,
		ipc.GameEndReason_STANDARD,
		0,
		0,
		false,
		nil)
	is.NoErr(err)

	err = tournament.SetResult(ctx,
		tstore,
		us,
		ty.UUID,
		divTwoName,
		"ValuedCustomer",
		"Dude",
		500,
		400,
		ipc.TournamentGameResult_WIN,
		ipc.TournamentGameResult_LOSS,
		ipc.GameEndReason_STANDARD,
		0,
		0,
		false,
		nil)
	is.NoErr(err)

	divOneComplete, err := tournament.IsRoundComplete(ctx, tstore, ty.UUID, divOneName, 0)
	is.NoErr(err)
	is.True(divOneComplete)

	divTwoComplete, err := tournament.IsRoundComplete(ctx, tstore, ty.UUID, divTwoName, 0)
	is.NoErr(err)
	is.True(divTwoComplete)

	us.(*user.DBStore).Disconnect()
	tstore.(*ts.Cache).Disconnect()
	gs.(*game.Cache).Disconnect()
}

func equalTournamentPersons(tp1 *ipc.TournamentPersons, tp2 *ipc.TournamentPersons) error {
	tp1String := tournamentPersonsToString(tp1)
	tp2String := tournamentPersonsToString(tp2)

	if len(tp1.Persons) != len(tp2.Persons) {
		return fmt.Errorf("tournamentPersons structs are not equal:\n%s\n%s", tp1String, tp2String)
	}

	for _, v1 := range tp1.Persons {
		personPresent := false
		for _, v2 := range tp2.Persons {
			if v1.Id == v2.Id && v1.Rating == v2.Rating {
				personPresent = true
				break
			}
		}
		if !personPresent {
			return fmt.Errorf("tournamentPersons structs are not equal:\n%s\n%s", tp1String, tp2String)
		}
	}
	return nil
}

func tournamentPersonsToString(tp *ipc.TournamentPersons) string {
	s := "{"
	for i := 0; i < len(tp.Persons); i++ {
		s += fmt.Sprintf("%s: %d", tp.Persons[i].Id, tp.Persons[i].Rating)
		if i != len(tp.Persons)-1 {
			s += ", "
		}
	}
	return s + "}"
}
