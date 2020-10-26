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

var directors = &entity.TournamentPersons{Persons: map[string]int{"Kieran": 1, "Vince": 2, "Jennifer": 2}}
var players = &entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100}}

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
		NumberOfRounds: 4,
		GamesPerRound:  1,
		Type:           entity.ClassicTournamentType,
		StartTime:      time.Now()}
}

func makeTournament(ctx context.Context, ts TournamentStore, cfg *config.Config) (*entity.Tournament, error) {
	return InstantiateNewTournament(ctx,
		ts,
		cfg,
		"Tournament",
		"This is a test Tournament",
		players,
		directors,
		makeControls())
}

func TestTournament(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	cstr := TestingDBConnStr + " dbname=liwords_test"
	cfg, tstore := tournamentStore(cstr)

	te, err := makeTournament(ctx, tstore, cfg)
	is.NoErr(err)

	// Check that players and directors are set correctly
	is.NoErr(equalTournamentPersons(players, te.Players))
	is.NoErr(equalTournamentPersons(directors, te.Directors))

	// Add directors that already exist
	err = tstore.AddDirectors(ctx, te.UUID, &entity.TournamentPersons{Persons: map[string]int{"Guy": 1, "Vince": 2}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(directors, te.Directors))

	// Add directors
	err = tstore.AddDirectors(ctx, te.UUID, &entity.TournamentPersons{Persons: map[string]int{"Evans": 4, "Oof": 2}})
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Kieran": 1, "Vince": 2, "Jennifer": 2, "Evans": 4, "Oof": 2}}, te.Directors))

	// Remove directors that don't exist
	err = tstore.RemoveDirectors(ctx, te.UUID, &entity.TournamentPersons{Persons: map[string]int{"Evans": -1, "Zoof": 2}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Kieran": 1, "Vince": 2, "Jennifer": 2, "Evans": 4, "Oof": 2}}, te.Directors))

	// Remove directors
	err = tstore.RemoveDirectors(ctx, te.UUID, &entity.TournamentPersons{Persons: map[string]int{"Evans": -1, "Oof": 2}})
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Kieran": 1, "Vince": 2, "Jennifer": 2}}, te.Directors))

	// Same thing for players.
	// Perhaps slightly redundant but at least it's thorough.

	// Add players that already exist
	err = tstore.AddPlayers(ctx, te.UUID, &entity.TournamentPersons{Persons: map[string]int{"Guy": 1, "Will": 2}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(players, te.Players))

	// Add players
	err = tstore.AddPlayers(ctx, te.UUID, &entity.TournamentPersons{Persons: map[string]int{"Noah": 4, "Bob": 2}})
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100, "Noah": 4, "Bob": 2}}, te.Players))

	// Remove players that don't exist
	err = tstore.RemovePlayers(ctx, te.UUID, &entity.TournamentPersons{Persons: map[string]int{"Evans": -1, "Zoof": 2}})
	is.True(err != nil)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100, "Noah": 4, "Bob": 2}}, te.Players))

	// Remove players
	err = tstore.RemovePlayers(ctx, te.UUID, &entity.TournamentPersons{Persons: map[string]int{"Josh": -1, "Conrad": 2}})
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(&entity.TournamentPersons{Persons: map[string]int{"Will": 1000, "Jesse": 2100, "Noah": 4, "Bob": 2}}, te.Players))

	// Set tournament controls
	err = tstore.SetTournamentControls(ctx,
		te.UUID,
		"The Changed Tournament Name",
		"It has been changed",
		makeControls())

	// Tournament should not be started
	isStarted, err := tstore.IsStarted(ctx, te.UUID)
	is.NoErr(err)
	is.True(!isStarted)

	// These should all fail because the tournament
	// has not started
	err = tstore.SetPairing(ctx, te.UUID, "Will", "Jesse", 0)
	is.True(err != nil)
	err = tstore.SetResult(ctx,
		te.UUID,
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

	isRoundComplete, err := tstore.IsRoundComplete(ctx, te.UUID, 0)
	is.True(err != nil)

	isFinished, err := tstore.IsFinished(ctx, te.UUID)
	is.True(err != nil)

	// Start the tournament

	err = tstore.StartRound(ctx, te.UUID, 0)
	is.NoErr(err)

	// Trying setting the controls again, this should fail
	err = tstore.SetTournamentControls(ctx,
		te.UUID,
		"The Changed Tournament Name",
		"It has been changed",
		makeControls())
	is.True(err != nil)

	// Tournament pairings and results are tested in the
	// entity package
	err = tstore.SetPairing(ctx, te.UUID, "Will", "Jesse", 0)
	is.NoErr(err)
	err = tstore.SetResult(ctx,
		te.UUID,
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

	isStarted, err = tstore.IsStarted(ctx, te.UUID)
	is.NoErr(err)
	is.True(isStarted)

	isRoundComplete, err = tstore.IsRoundComplete(ctx, te.UUID, 0)
	is.NoErr(err)
	is.True(!isRoundComplete)

	isFinished, err = tstore.IsFinished(ctx, te.UUID)
	is.NoErr(err)
	is.True(!isFinished)

	tstore.(*ts.Cache).Disconnect()
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
