package tournament_test

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

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
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondoconfig "github.com/domino14/macondo/config"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

var TestDBHost = os.Getenv("TEST_DB_HOST")
var TestingDBConnStr = "host=" + TestDBHost + " port=5432 user=postgres password=pass sslmode=disable"
var gameReq = &realtime.GameRequest{Lexicon: "CSW19",
	Rules: &realtime.GameRules{BoardLayoutName: entity.CrosswordGame,
		LetterDistributionName: "English",
		VariantName:            "classic"},

	InitialTimeSeconds: 25 * 60,
	IncrementSeconds:   0,
	ChallengeRule:      macondopb.ChallengeRule_FIVE_POINT,
	GameMode:           realtime.GameMode_REAL_TIME,
	RatingMode:         realtime.RatingMode_RATED,
	RequestId:          "yeet",
	OriginalRequestId:  "originalyeet",
	MaxOvertimeMinutes: 10}

var DefaultConfig = macondoconfig.Config{
	LexiconPath:               os.Getenv("LEXICON_PATH"),
	LetterDistributionPath:    os.Getenv("LETTER_DISTRIBUTION_PATH"),
	DefaultLexicon:            "CSW19",
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
		{Username: "Dude", Email: "dude@woogles.io", UUID: "Dude"},
		{Username: "Comrade", Email: "comrade@woogles.io", UUID: "Comrade"},
		{Username: "Valued Customer", Email: "valued@woogles.io", UUID: "Valued Customer"},
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

func makeRoundControls() []*entity.RoundControls {
	return []*entity.RoundControls{&entity.RoundControls{FirstMethod: entity.AutomaticFirst,
		PairingMethod:               entity.RoundRobin,
		GamesPerRound:               1,
		Factor:                      1,
		MaxRepeats:                  1,
		AllowOverMaxRepeats:         true,
		RepeatRelativeWeight:        1,
		WinDifferenceRelativeWeight: 1},
		&entity.RoundControls{FirstMethod: entity.AutomaticFirst,
			PairingMethod:               entity.RoundRobin,
			GamesPerRound:               1,
			Factor:                      1,
			MaxRepeats:                  1,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1},
		&entity.RoundControls{FirstMethod: entity.AutomaticFirst,
			PairingMethod:               entity.RoundRobin,
			GamesPerRound:               1,
			Factor:                      1,
			MaxRepeats:                  1,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1},
		&entity.RoundControls{FirstMethod: entity.AutomaticFirst,
			PairingMethod:               entity.KingOfTheHill,
			GamesPerRound:               1,
			Factor:                      1,
			MaxRepeats:                  1,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1}}
}

func makeControls() *entity.TournamentControls {
	return &entity.TournamentControls{
		GameRequest:    gameReq,
		RoundControls:  makeRoundControls(),
		NumberOfRounds: 4,
		Type:           entity.ClassicTournamentType,
		AutoStart:      false,
		StartTime:      time.Now()}
}

func makeTournament(ctx context.Context, ts tournament.TournamentStore, cfg *config.Config, directors *entity.TournamentPersons) (*entity.Tournament, error) {
	return tournament.NewTournament(ctx,
		ts,
		"Tournament",
		"This is a test Tournament",
		directors,
		entity.TypeStandard,
		"",
		"/tournament/slug-tourney",
	)
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

	players := &entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100}}
	directors := &entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 2, "Jennifer": 2}}
	directorsTwoExecutives := &entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 0, "Jennifer": 2}}
	directorsNoExecutives := &entity.TournamentPersons{Persons: map[string]int{"Kieran": 1, "Vince": 3, "Jennifer": 2}}

	ty, err := makeTournament(ctx, tstore, cfg, directorsTwoExecutives)
	is.True(err != nil)

	ty, err = makeTournament(ctx, tstore, cfg, directorsNoExecutives)
	is.True(err != nil)

	ty, err = makeTournament(ctx, tstore, cfg, directors)
	is.NoErr(err)

	err = tournament.SetTournamentMetadata(ctx, tstore, ty.UUID, "New Name", "New Description")
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
	err = tournament.AddDirectors(ctx, tstore, ty.UUID, &entity.TournamentPersons{Persons: map[string]int{"Guy": 1, "Vince": 2}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(directors, ty.Directors))

	// Attempt to add another executive director
	err = tournament.AddDirectors(ctx, tstore, ty.UUID, &entity.TournamentPersons{Persons: map[string]int{"Guy": 1, "Harry": 0}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(directors, ty.Directors))

	// Add directors
	err = tournament.AddDirectors(ctx, tstore, ty.UUID, &entity.TournamentPersons{Persons: map[string]int{"Evans": 4, "Oof": 2}})
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 2, "Jennifer": 2, "Evans": 4, "Oof": 2}}, ty.Directors))

	// Attempt to remove directors that don't exist
	err = tournament.RemoveDirectors(ctx, tstore, ty.UUID, &entity.TournamentPersons{Persons: map[string]int{"Evans": -1, "Zoof": 2}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 2, "Jennifer": 2, "Evans": 4, "Oof": 2}}, ty.Directors))

	// Attempt to remove the executive director
	err = tournament.RemoveDirectors(ctx, tstore, ty.UUID, &entity.TournamentPersons{Persons: map[string]int{"Evans": -1, "Kieran": 0}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 2, "Jennifer": 2, "Evans": 4, "Oof": 2}}, ty.Directors))

	// Remove directors
	err = tournament.RemoveDirectors(ctx, tstore, ty.UUID, &entity.TournamentPersons{Persons: map[string]int{"Evans": -1, "Oof": 2}})
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 2, "Jennifer": 2}}, ty.Directors))

	// Attempt to remove the executive director
	err = tournament.RemoveDirectors(ctx, tstore, ty.UUID, &entity.TournamentPersons{Persons: map[string]int{"Vince": -1, "Kieran": 0}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 2, "Jennifer": 2}}, ty.Directors))

	// Same thing for players.
	div1 := ty.Divisions[divOneName]

	// Add players
	err = tournament.AddPlayers(ctx, tstore, ty.UUID, divOneName, players)
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(players, div1.Players))

	// Add players to a division that doesn't exist
	err = tournament.AddPlayers(ctx, tstore, ty.UUID, divOneName+"not quite", &entity.TournamentPersons{Persons: map[string]int{"Noah": 4, "Bob": 2}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(players, div1.Players))

	// Add players
	err = tournament.AddPlayers(ctx, tstore, ty.UUID, divOneName, &entity.TournamentPersons{Persons: map[string]int{"Noah": 4, "Bob": 2}})
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100, "Noah": 4, "Bob": 2}}, div1.Players))

	// Remove players that don't exist
	err = tournament.RemovePlayers(ctx, tstore, ty.UUID, divOneName, &entity.TournamentPersons{Persons: map[string]int{"Evans": -1, "Zoof": 2}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100, "Noah": 4, "Bob": 2}}, div1.Players))

	// Remove players from a division that doesn't exist
	err = tournament.RemovePlayers(ctx, tstore, ty.UUID, divOneName+"hmm", &entity.TournamentPersons{Persons: map[string]int{"Josh": -1, "Conrad": 2}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100, "Noah": 4, "Bob": 2}}, div1.Players))

	// Remove players
	err = tournament.RemovePlayers(ctx, tstore, ty.UUID, divOneName, &entity.TournamentPersons{Persons: map[string]int{"Josh": -1, "Conrad": 2}})
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Jesse": 2100, "Noah": 4, "Bob": 2}}, div1.Players))

	// Set tournament controls
	err = tournament.SetTournamentControls(ctx,
		tstore,
		ty.UUID,
		divOneName,
		makeControls())
	is.NoErr(err)

	// Set tournament controls for a division that does not exist
	err = tournament.SetTournamentControls(ctx,
		tstore,
		ty.UUID,
		divOneName+" another one",
		makeControls())
	is.True(err != nil)

	// Tournament should not be started
	isStarted, err := tournament.IsStarted(ctx, tstore, ty.UUID)
	is.NoErr(err)
	is.True(!isStarted)

	// Set pairing should work before the tournament starts
	err = tournament.SetPairing(ctx, tstore, ty.UUID, divOneName, "Will", "Jesse", 0)
	is.NoErr(err)

	// Remove players and attempt to set pairings
	err = tournament.RemovePlayers(ctx, tstore, ty.UUID, divOneName, &entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Jesse": 2100, "Noah": 4, "Bob": 2}})
	is.NoErr(err)

	err = tournament.SetPairing(ctx, tstore, ty.UUID, divOneName, "Will", "Jesse", 0)
	is.True(err != nil)

	err = tournament.SetResult(ctx,
		tstore,
		us,
		ty.UUID,
		divOneName,
		"Will",
		"Jesse",
		500,
		400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD,
		0,
		0,
		false,
		nil,
	)
	is.True(err != nil)

	isRoundComplete, err := tournament.IsRoundComplete(ctx, tstore, ty.UUID, divOneName, 0)
	is.True(err != nil)

	isFinished, err := tournament.IsFinished(ctx, tstore, ty.UUID, divOneName)
	is.True(err != nil)

	// Add players back in
	err = tournament.AddPlayers(ctx, tstore, ty.UUID, divOneName, players)
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(players, div1.Players))

	// Start the tournament

	err = tournament.StartTournament(ctx, tstore, ty.UUID, true)
	is.NoErr(err)

	// Attempt to add a division after the tournament has started
	err = tournament.AddDivision(ctx, tstore, ty.UUID, divOneName+" this time it's different")
	is.True(err != nil)

	// Attempt to remove a division after the tournament has started
	err = tournament.RemoveDivision(ctx, tstore, ty.UUID, divOneName)
	is.True(err != nil)

	// Trying setting the controls after the tournament has started, this should fail
	err = tournament.SetTournamentControls(ctx,
		tstore,
		ty.UUID,
		divOneName,
		makeControls())
	is.True(err != nil)

	// Tournament pairings and results are tested in the
	// entity package
	err = tournament.SetPairing(ctx, tstore, ty.UUID, divOneName, "Will", "Jesse", 0)
	is.NoErr(err)

	// Set pairings for division that does not exist
	err = tournament.SetPairing(ctx, tstore, ty.UUID, divOneName+"yeet", "Will", "Jesse", 0)
	is.True(err != nil)

	err = tournament.SetResult(ctx,
		tstore,
		us,
		ty.UUID,
		divOneName,
		"Will",
		"Jesse",
		500,
		400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD,
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
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD,
		0,
		0,
		false,
		nil)
	is.True(err != nil)

	isStarted, err = tournament.IsStarted(ctx, tstore, ty.UUID)
	is.NoErr(err)
	is.True(isStarted)

	isRoundComplete, err = tournament.IsRoundComplete(ctx, tstore, ty.UUID, divOneName, 0)
	is.NoErr(err)
	is.True(!isRoundComplete)

	// See if round is complete for division that does not exist
	isRoundComplete, err = tournament.IsRoundComplete(ctx, tstore, ty.UUID, divOneName+"yah", 0)
	is.True(err != nil)

	isFinished, err = tournament.IsFinished(ctx, tstore, ty.UUID, divOneName)
	is.NoErr(err)
	is.True(!isFinished)

	// See if division is finished (except it doesn't exist)
	isFinished, err = tournament.IsFinished(ctx, tstore, ty.UUID, divOneName+"but wait there's more")
	is.True(err != nil)

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

	divOnePlayers := &entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100}}
	divTwoPlayers := &entity.TournamentPersons{Persons: map[string]int{"Guy": 1000, "Dude": 3000, "Comrade": 2200, "Valued Customer": 2100}}
	directors := &entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 2, "Jennifer": 2}}

	ty, err := makeTournament(ctx, tstore, cfg, directors)
	is.NoErr(err)

	// Add divisions
	err = tournament.AddDivision(ctx, tstore, ty.UUID, divOneName)
	is.NoErr(err)

	err = tournament.AddDivision(ctx, tstore, ty.UUID, divTwoName)
	is.NoErr(err)

	// Set tournament controls
	err = tournament.SetTournamentControls(ctx,
		tstore,
		ty.UUID,
		divOneName,
		makeControls())
	is.NoErr(err)

	err = tournament.SetTournamentControls(ctx,
		tstore,
		ty.UUID,
		divTwoName,
		makeControls())
	is.NoErr(err)

	div1 := ty.Divisions[divOneName]
	div2 := ty.Divisions[divTwoName]

	// Add players
	err = tournament.AddPlayers(ctx, tstore, ty.UUID, divOneName, divOnePlayers)
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(divOnePlayers, div1.Players))

	err = tournament.AddPlayers(ctx, tstore, ty.UUID, divTwoName, divTwoPlayers)
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(divTwoPlayers, div2.Players))

	err = tournament.SetPairing(ctx, tstore, ty.UUID, divOneName, "Will", "Jesse", 0)
	is.NoErr(err)

	err = tournament.SetPairing(ctx, tstore, ty.UUID, divTwoName, "Guy", "Comrade", 0)
	is.NoErr(err)

	err = tournament.SetPairing(ctx, tstore, ty.UUID, divOneName, "Conrad", "Josh", 0)
	is.NoErr(err)

	err = tournament.SetPairing(ctx, tstore, ty.UUID, divTwoName, "Dude", "Valued Customer", 0)
	is.NoErr(err)

	// Start the tournament

	err = tournament.StartTournament(ctx, tstore, ty.UUID, true)
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
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD,
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
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD,
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
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD,
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
		"Valued Customer",
		"Dude",
		500,
		400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD,
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

func equalTournamentPersons(tp1 *entity.TournamentPersons, tp2 *entity.TournamentPersons) error {
	tp1String := tournamentPersonsToString(tp1)
	tp2String := tournamentPersonsToString(tp2)
	for k, v1 := range tp1.Persons {
		v2, ok := tp2.Persons[k]
		if !ok || v1 != v2 {
			return fmt.Errorf("tournamentPersons structs are not equal: %s, %s", tp1String, tp2String)
		}
	}
	for k, v2 := range tp2.Persons {
		v1, ok := tp1.Persons[k]
		if !ok || v1 != v2 {
			return fmt.Errorf("tournamentPersons structs are not equal: %s, %s", tp1String, tp2String)
		}
	}

	return nil
}

func tournamentPersonsToString(tp *entity.TournamentPersons) string {
	s := "{"
	keys := []string{}
	for k, _ := range tp.Persons {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i := 0; i < len(keys); i++ {
		s += fmt.Sprintf("%s: %d", keys[i], tp.Persons[keys[i]])
		if i != len(keys)-1 {
			s += ", "
		}
	}
	return s + "}"
}
