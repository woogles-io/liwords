package tournament

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	ts "github.com/domino14/liwords/pkg/stores/tournament"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondoconfig "github.com/domino14/macondo/config"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

var TestDBHost = os.Getenv("TEST_DB_HOST")
var TestingDBConnStr = "host=" + TestDBHost + " port=5432 user=postgres password=pass sslmode=disable"
var gameReq = &pb.GameRequest{Lexicon: "CSW19",
	Rules: &pb.GameRules{BoardLayoutName: entity.CrosswordGame,
		LetterDistributionName: "English",
		VariantName:            "classic"},

	InitialTimeSeconds: 25 * 60,
	IncrementSeconds:   0,
	ChallengeRule:      macondopb.ChallengeRule_FIVE_POINT,
	GameMode:           pb.GameMode_REAL_TIME,
	RatingMode:         pb.RatingMode_RATED,
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

func tournamentStore(dbURL string) (*config.Config, TournamentStore) {
	cfg := &config.Config{}
	cfg.MacondoConfig = DefaultConfig
	cfg.DBConnString = dbURL

	tmp, err := ts.NewDBStore(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	tournamentStore := ts.NewCache(tmp)
	return cfg, tournamentStore
}

func makeControls() *entity.TournamentControls {
	return &entity.TournamentControls{
		GameRequest:    gameReq,
		PairingMethods: []entity.PairingMethod{entity.RoundRobin, entity.RoundRobin, entity.RoundRobin, entity.KingOfTheHill},
		FirstMethods:   []entity.FirstMethod{entity.AutomaticFirst, entity.AutomaticFirst, entity.AutomaticFirst, entity.AutomaticFirst},
		NumberOfRounds: 4,
		GamesPerRound:  []int{1, 1, 1, 1},
		Type:           entity.ClassicTournamentType,
		StartTime:      time.Now()}
}

func makeTournament(ctx context.Context, ts TournamentStore, cfg *config.Config, directors *entity.TournamentPersons) (*entity.Tournament, error) {
	return NewTournament(ctx,
		ts,
		cfg,
		"Tournament",
		"This is a test Tournament",
		directors)
}

func TestTournamentSingleDivision(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	cstr := TestingDBConnStr + " dbname=liwords_test"
	cfg, tstore := tournamentStore(cstr)

	players := &entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100}}
	directors := &entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 2, "Jennifer": 2}}
	directorsTwoExecutives := &entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 0, "Jennifer": 2}}
	directorsNoExecutives := &entity.TournamentPersons{Persons: map[string]int{"Kieran": 1, "Vince": 3, "Jennifer": 2}}

	tournament, err := makeTournament(ctx, tstore, cfg, directorsTwoExecutives)
	is.True(err != nil)

	tournament, err = makeTournament(ctx, tstore, cfg, directorsNoExecutives)
	is.True(err != nil)

	tournament, err = makeTournament(ctx, tstore, cfg, directors)
	is.NoErr(err)

	err = SetTournamentMetadata(ctx, tstore, tournament.UUID, "New Name", "New Description")
	is.NoErr(err)

	// Check that directors are set correctly
	is.NoErr(equalTournamentPersons(directors, tournament.Directors))

	// Attempt to remove a division that doesn't exist in the empty tournament
	err = RemoveDivision(ctx, tstore, tournament.UUID, "The Big Boys")
	is.True(err != nil)

	// Add a division
	err = AddDivision(ctx, tstore, tournament.UUID, divOneName)
	is.NoErr(err)

	// Attempt to remove a division that doesn't exist when other
	// divisions are present
	err = RemoveDivision(ctx, tstore, tournament.UUID, "Nope")
	is.True(err != nil)

	// Attempt to add a division that already exists
	err = AddDivision(ctx, tstore, tournament.UUID, divOneName)
	is.True(err != nil)

	// Attempt to add directors that already exist
	err = AddDirectors(ctx, tstore, tournament.UUID, &entity.TournamentPersons{Persons: map[string]int{"Guy": 1, "Vince": 2}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(directors, tournament.Directors))

	// Attempt to add another executive director
	err = AddDirectors(ctx, tstore, tournament.UUID, &entity.TournamentPersons{Persons: map[string]int{"Guy": 1, "Harry": 0}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(directors, tournament.Directors))

	// Add directors
	err = AddDirectors(ctx, tstore, tournament.UUID, &entity.TournamentPersons{Persons: map[string]int{"Evans": 4, "Oof": 2}})
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 2, "Jennifer": 2, "Evans": 4, "Oof": 2}}, tournament.Directors))

	// Attempt to remove directors that don't exist
	err = RemoveDirectors(ctx, tstore, tournament.UUID, &entity.TournamentPersons{Persons: map[string]int{"Evans": -1, "Zoof": 2}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 2, "Jennifer": 2, "Evans": 4, "Oof": 2}}, tournament.Directors))

	// Attempt to remove the executive director
	err = RemoveDirectors(ctx, tstore, tournament.UUID, &entity.TournamentPersons{Persons: map[string]int{"Evans": -1, "Kieran": 0}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 2, "Jennifer": 2, "Evans": 4, "Oof": 2}}, tournament.Directors))

	// Remove directors
	err = RemoveDirectors(ctx, tstore, tournament.UUID, &entity.TournamentPersons{Persons: map[string]int{"Evans": -1, "Oof": 2}})
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 2, "Jennifer": 2}}, tournament.Directors))

	// Attempt to remove the executive director
	err = RemoveDirectors(ctx, tstore, tournament.UUID, &entity.TournamentPersons{Persons: map[string]int{"Vince": -1, "Kieran": 0}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 2, "Jennifer": 2}}, tournament.Directors))

	// Same thing for players.
	div1 := tournament.Divisions[divOneName]

	// Add players
	err = AddPlayers(ctx, tstore, tournament.UUID, divOneName, players)
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(players, div1.Players))

	// Add players to a division that doesn't exist
	err = AddPlayers(ctx, tstore, tournament.UUID, divOneName+"not quite", &entity.TournamentPersons{Persons: map[string]int{"Noah": 4, "Bob": 2}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(players, div1.Players))

	// Add players
	err = AddPlayers(ctx, tstore, tournament.UUID, divOneName, &entity.TournamentPersons{Persons: map[string]int{"Noah": 4, "Bob": 2}})
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100, "Noah": 4, "Bob": 2}}, div1.Players))

	// Remove players that don't exist
	err = RemovePlayers(ctx, tstore, tournament.UUID, divOneName, &entity.TournamentPersons{Persons: map[string]int{"Evans": -1, "Zoof": 2}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100, "Noah": 4, "Bob": 2}}, div1.Players))

	// Remove players from a division that doesn't exist
	err = RemovePlayers(ctx, tstore, tournament.UUID, divOneName+"hmm", &entity.TournamentPersons{Persons: map[string]int{"Josh": -1, "Conrad": 2}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100, "Noah": 4, "Bob": 2}}, div1.Players))

	// Remove players
	err = RemovePlayers(ctx, tstore, tournament.UUID, divOneName, &entity.TournamentPersons{Persons: map[string]int{"Josh": -1, "Conrad": 2}})
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Jesse": 2100, "Noah": 4, "Bob": 2}}, div1.Players))

	// Set tournament controls
	err = SetTournamentControls(ctx,
		tstore,
		tournament.UUID,
		divOneName,
		makeControls())
	is.NoErr(err)

	// Set tournament controls for a division that does not exist
	err = SetTournamentControls(ctx,
		tstore,
		tournament.UUID,
		divOneName+" another one",
		makeControls())
	is.True(err != nil)

	// Tournament should not be started
	isStarted, err := IsStarted(ctx, tstore, tournament.UUID)
	is.NoErr(err)
	is.True(!isStarted)

	// Set pairing should work before the tournament starts
	err = SetPairing(ctx, tstore, tournament.UUID, divOneName, "Will", "Jesse", 0)
	is.NoErr(err)

	// Remove players and attempt to set pairings
	err = RemovePlayers(ctx, tstore, tournament.UUID, divOneName, &entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Jesse": 2100, "Noah": 4, "Bob": 2}})
	is.NoErr(err)

	err = SetPairing(ctx, tstore, tournament.UUID, divOneName, "Will", "Jesse", 0)
	is.True(err != nil)

	err = SetResult(ctx,
		tstore,
		tournament.UUID,
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
		false)
	is.True(err != nil)

	isRoundComplete, err := IsRoundComplete(ctx, tstore, tournament.UUID, divOneName, 0)
	is.True(err != nil)

	isFinished, err := IsFinished(ctx, tstore, tournament.UUID, divOneName)
	is.True(err != nil)

	// Add players back in
	err = AddPlayers(ctx, tstore, tournament.UUID, divOneName, players)
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(players, div1.Players))

	// Start the tournament

	err = StartTournament(ctx, tstore, tournament.UUID)
	is.NoErr(err)

	// Attempt to add a division after the tournament has started
	err = AddDivision(ctx, tstore, tournament.UUID, divOneName+" this time it's different")
	is.True(err != nil)

	// Attempt to remove a division after the tournament has started
	err = RemoveDivision(ctx, tstore, tournament.UUID, divOneName)
	is.True(err != nil)

	// Trying setting the controls after the tournament has started, this should fail
	err = SetTournamentControls(ctx,
		tstore,
		tournament.UUID,
		divOneName,
		makeControls())
	is.True(err != nil)

	// Tournament pairings and results are tested in the
	// entity package
	err = SetPairing(ctx, tstore, tournament.UUID, divOneName, "Will", "Jesse", 0)
	is.NoErr(err)

	// Set pairings for division that does not exist
	err = SetPairing(ctx, tstore, tournament.UUID, divOneName+"yeet", "Will", "Jesse", 0)
	is.True(err != nil)

	err = SetResult(ctx,
		tstore,
		tournament.UUID,
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
		false)
	is.NoErr(err)

	// Set results for a division that does not exist
	err = SetResult(ctx,
		tstore,
		tournament.UUID,
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
		false)
	is.True(err != nil)

	isStarted, err = IsStarted(ctx, tstore, tournament.UUID)
	is.NoErr(err)
	is.True(isStarted)

	isRoundComplete, err = IsRoundComplete(ctx, tstore, tournament.UUID, divOneName, 0)
	is.NoErr(err)
	is.True(!isRoundComplete)

	// See if round is complete for division that does not exist
	isRoundComplete, err = IsRoundComplete(ctx, tstore, tournament.UUID, divOneName+"yah", 0)
	is.True(err != nil)

	isFinished, err = IsFinished(ctx, tstore, tournament.UUID, divOneName)
	is.NoErr(err)
	is.True(!isFinished)

	// See if division is finished (except it doesn't exist)
	isFinished, err = IsFinished(ctx, tstore, tournament.UUID, divOneName+"but wait there's more")
	is.True(err != nil)

	tstore.(*ts.Cache).Disconnect()
}

func TestTournamentMultipleDivisions(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	cstr := TestingDBConnStr + " dbname=liwords_test"
	cfg, tstore := tournamentStore(cstr)

	divOnePlayers := &entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100}}
	divTwoPlayers := &entity.TournamentPersons{Persons: map[string]int{"Guy": 1000, "Dude": 3000, "Comrade": 2200, "Valued Customer": 2100}}
	directors := &entity.TournamentPersons{Persons: map[string]int{"Kieran": 0, "Vince": 2, "Jennifer": 2}}

	tournament, err := makeTournament(ctx, tstore, cfg, directors)
	is.NoErr(err)

	// Add divisions
	err = AddDivision(ctx, tstore, tournament.UUID, divOneName)
	is.NoErr(err)

	err = AddDivision(ctx, tstore, tournament.UUID, divTwoName)
	is.NoErr(err)

	// Set tournament controls
	err = SetTournamentControls(ctx,
		tstore,
		tournament.UUID,
		divOneName,
		makeControls())
	is.NoErr(err)

	err = SetTournamentControls(ctx,
		tstore,
		tournament.UUID,
		divTwoName,
		makeControls())
	is.NoErr(err)

	div1 := tournament.Divisions[divOneName]
	div2 := tournament.Divisions[divTwoName]

	// Add players
	err = AddPlayers(ctx, tstore, tournament.UUID, divOneName, divOnePlayers)
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(divOnePlayers, div1.Players))

	err = AddPlayers(ctx, tstore, tournament.UUID, divTwoName, divTwoPlayers)
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(divTwoPlayers, div2.Players))

	err = SetPairing(ctx, tstore, tournament.UUID, divOneName, "Will", "Jesse", 0)
	is.NoErr(err)

	err = SetPairing(ctx, tstore, tournament.UUID, divTwoName, "Guy", "Comrade", 0)
	is.NoErr(err)

	err = SetPairing(ctx, tstore, tournament.UUID, divOneName, "Conrad", "Josh", 0)
	is.NoErr(err)

	err = SetPairing(ctx, tstore, tournament.UUID, divTwoName, "Dude", "Valued Customer", 0)
	is.NoErr(err)

	// Start the tournament

	err = StartTournament(ctx, tstore, tournament.UUID)
	is.NoErr(err)

	err = SetResult(ctx,
		tstore,
		tournament.UUID,
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
		false)
	is.NoErr(err)

	err = SetResult(ctx,
		tstore,
		tournament.UUID,
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
		false)
	is.NoErr(err)

	err = SetResult(ctx,
		tstore,
		tournament.UUID,
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
		false)
	is.NoErr(err)

	err = SetResult(ctx,
		tstore,
		tournament.UUID,
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
		false)
	is.NoErr(err)

	divOneComplete, err := IsRoundComplete(ctx, tstore, tournament.UUID, divOneName, 0)
	is.NoErr(err)
	is.True(divOneComplete)

	divTwoComplete, err := IsRoundComplete(ctx, tstore, tournament.UUID, divTwoName, 0)
	is.NoErr(err)
	is.True(divTwoComplete)
}

func equalTournamentPersons(tp1 *entity.TournamentPersons, tp2 *entity.TournamentPersons) error {
	tp1String := tournamentPersonsToString(tp1)
	tp2String := tournamentPersonsToString(tp2)
	for k, v1 := range tp1.Persons {
		v2, ok := tp2.Persons[k]
		if !ok || v1 != v2 {
			return errors.New(fmt.Sprintf("TournamentPersons structs are not equal:\n%s\n%s\n", tp1String, tp2String))
		}
	}
	for k, v2 := range tp2.Persons {
		v1, ok := tp1.Persons[k]
		if !ok || v1 != v2 {
			return errors.New(fmt.Sprintf("TournamentPersons structs are not equal:\n%s\n%s\n", tp1String, tp2String))
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
