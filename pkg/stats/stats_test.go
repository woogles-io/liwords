package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/domino14/liwords/pkg/entity"
	statstore "github.com/domino14/liwords/pkg/stores/stats"
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/domino14/macondo/alphabet"
	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/macondo/gcgio"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/jinzhu/gorm"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var TestDBHost = os.Getenv("TEST_DB_HOST")
var TestingDBConnStr = "host=" + TestDBHost + " port=5432 user=postgres password=pass sslmode=disable"

var DefaultConfig = macondoconfig.Config{
	LexiconPath:               os.Getenv("LEXICON_PATH"),
	LetterDistributionPath:    os.Getenv("LETTER_DISTRIBUTION_PATH"),
	DefaultLexicon:            "CSW21",
	DefaultLetterDistribution: "English",
}

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	os.Exit(m.Run())
}

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
}

func InstantiateNewStatsWithHistory(filename string, listStatStore ListStatStore) (*entity.Stats, error) {

	history, err := gcgio.ParseGCG(&DefaultConfig, filename)
	if err != nil {
		panic(err)
	}
	// For these tests, ensure "jvc" and "Josh" always have a player id of 1
	playerOneId := "1"
	playerTwoId := "2"
	if history.Players[1].Nickname == "jvc" || history.Players[1].Nickname == "Josh" {
		playerOneId = "2"
		playerTwoId = "1"
	}
	stats := InstantiateNewStats(playerOneId, playerTwoId)
	if err != nil {
		return nil, err
	}

	req := &realtime.GameRequest{Lexicon: "CSW21",
		Rules: &realtime.GameRules{BoardLayoutName: entity.CrosswordGame,
			LetterDistributionName: "letterdist",
			VariantName:            "classic"},

		InitialTimeSeconds: 25 * 60,
		IncrementSeconds:   0,
		ChallengeRule:      pb.ChallengeRule_FIVE_POINT,
		GameMode:           realtime.GameMode_REAL_TIME,
		RatingMode:         realtime.RatingMode_RATED,
		RequestId:          "yeet",
		MaxOvertimeMinutes: 10}

	// Just dummy info to test that rating stats work
	gameEndedEvent := &realtime.GameEndedEvent{
		Scores: map[string]int32{history.Players[0].Nickname: history.FinalScores[0],
			history.Players[1].Nickname: history.FinalScores[1]},
		NewRatings: map[string]int32{history.Players[0].Nickname: int32(1500),
			history.Players[1].Nickname: int32(1400)},
		EndReason: realtime.GameEndReason_STANDARD,
		Winner:    history.Players[0].Nickname,
		Loser:     history.Players[1].Nickname,
		Tie:       history.FinalScores[0] != history.FinalScores[1],
	}

	AddGame(stats, listStatStore, history, req, &DefaultConfig, gameEndedEvent, filename)

	return stats, nil
}

func JoshNationalsFromGames(useJSON bool, listStatStore ListStatStore) (*entity.Stats, []string, error) {
	annotatedGamePrefix := "josh_nationals_round_"
	stats := InstantiateNewStats("1", "2")
	files := []string{}
	for i := 1; i <= 31; i++ {
		annotatedGame := fmt.Sprintf("./testdata/%s%d.gcg", annotatedGamePrefix, i)
		otherStats, err := InstantiateNewStatsWithHistory(annotatedGame, listStatStore)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, annotatedGame)
		if useJSON {
			bytes, err := json.Marshal(otherStats)
			if err != nil {
				fmt.Println(err)
				return otherStats, files, err
			}
			// It is assumed that in production the json is stored in a database
			// and remains untouched until it is retrieved later for stats
			// aggregation
			var otherStatsFromJSON entity.Stats
			err = json.Unmarshal(bytes, &otherStatsFromJSON)
			if err != nil {
				return otherStats, files, err
			}
			otherStats = &otherStatsFromJSON
		}
		AddStats(stats, otherStats)
	}
	return stats, files, nil
}

func TestStatsFromJson(t *testing.T) {
	is := is.New(t)

	recreateDB()
	listStatStore, err := statstore.NewListStatStore(TestingDBConnStr + " dbname=liwords_test")
	is.NoErr(err)
	stats, gameIds, err := JoshNationalsFromGames(false, listStatStore)
	is.NoErr(err)
	statsJSON, gameIds, err := JoshNationalsFromGames(true, listStatStore)
	is.NoErr(err)

	err = Finalize(stats, listStatStore, gameIds, "", "")
	is.NoErr(err)
	err = Finalize(statsJSON, listStatStore, gameIds, "", "")
	is.NoErr(err)
	is.True(isEqual(stats, statsJSON))
	listStatStore.Disconnect()
}

func TestStats(t *testing.T) {
	is := is.New(t)

	recreateDB()
	listStatStore, err := statstore.NewListStatStore(TestingDBConnStr + " dbname=liwords_test")
	is.NoErr(err)
	stats, gameIds, err := JoshNationalsFromGames(false, listStatStore)
	is.NoErr(err)

	err = Finalize(stats, listStatStore, gameIds, "", "")
	is.NoErr(err)

	// fmt.Println(StatsToString(stats))
	is.True(stats.PlayerOneData[entity.NO_BINGOS_STAT].Total == 0)
	is.True(stats.PlayerOneData[entity.BINGOS_STAT].Total == 75)
	is.True(stats.PlayerOneData[entity.CHALLENGED_PHONIES_STAT].Total == 4)
	is.True(stats.PlayerOneData[entity.CHALLENGES_WON_STAT].Total == 9)
	is.True(stats.PlayerOneData[entity.CHALLENGES_LOST_STAT].Total == 11)
	is.True(stats.PlayerOneData[entity.COMMENTS_STAT].Total == 174)
	is.True(stats.PlayerOneData[entity.DRAWS_STAT].Total == 0)
	is.True(stats.PlayerOneData[entity.EXCHANGES_STAT].Total == 3)
	is.True(stats.PlayerOneData[entity.FIRSTS_STAT].Total == 16)
	is.True(stats.PlayerOneData[entity.GAMES_STAT].Total == 31)
	is.True(stats.PlayerOneData[entity.GAMES_STAT].Subitems["VOID"] == 0)
	is.True(stats.PlayerOneData[entity.GAMES_STAT].Subitems["SINGLE"] == 0)
	is.True(stats.PlayerOneData[entity.GAMES_STAT].Subitems["DOUBLE"] == 0)
	is.True(stats.PlayerOneData[entity.GAMES_STAT].Subitems["FIVE_POINT"] == 31)
	is.True(stats.PlayerOneData[entity.GAMES_STAT].Subitems["TEN_POINT"] == 0)
	is.True(stats.PlayerOneData[entity.GAMES_STAT].Subitems["RATED"] == 31)
	is.True(stats.PlayerOneData[entity.GAMES_STAT].Subitems["CASUAL"] == 0)
	is.True(stats.PlayerOneData[entity.HIGH_GAME_STAT].Total == 619)
	is.True(stats.PlayerOneData[entity.HIGH_TURN_STAT].Total == 167)
	is.True(stats.PlayerOneData[entity.LOSSES_STAT].Total == 11)
	is.True(stats.PlayerOneData[entity.LOW_GAME_STAT].Total == 359)
	is.True(stats.PlayerOneData[entity.MISTAKES_STAT].Total == 134)
	is.True(stats.PlayerOneData[entity.VALID_PLAYS_THAT_WERE_CHALLENGED_STAT].Total == 14)
	is.True(stats.PlayerOneData[entity.SCORE_STAT].Total == 13977)
	is.True(stats.PlayerOneData[entity.TRIPLE_TRIPLES_STAT].Total == 1)
	is.True(stats.PlayerOneData[entity.TURNS_STAT].Total == 375)
	is.True(stats.PlayerOneData[entity.TURNS_WITH_BLANK_STAT].Total == 72)
	is.True(stats.PlayerOneData[entity.UNCHALLENGED_PHONIES_STAT].Total == 1)
	is.True(stats.PlayerOneData[entity.VERTICAL_OPENINGS_STAT].Total == 0)
	is.True(stats.PlayerOneData[entity.WINS_STAT].Total == 20)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Total == 1556)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["A"] == 134)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["B"] == 25)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["C"] == 26)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["D"] == 71)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["E"] == 186)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["F"] == 24)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["G"] == 50)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["H"] == 34)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["I"] == 144)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["J"] == 13)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["K"] == 14)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["L"] == 72)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["M"] == 32)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["N"] == 102)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["O"] == 125)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["P"] == 21)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["Q"] == 16)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["R"] == 94)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["S"] == 44)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["T"] == 94)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["U"] == 67)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["V"] == 30)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["W"] == 31)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["X"] == 16)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["Y"] == 33)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems["Z"] == 19)
	is.True(stats.PlayerOneData[entity.TILES_PLAYED_STAT].Subitems[string(alphabet.BlankToken)] == 39)

	is.True(len(stats.NotableData[entity.MANY_DOUBLE_WORDS_COVERED_STAT].List) == 0)
	is.True(len(stats.NotableData[entity.ALL_TRIPLE_LETTERS_COVERED_STAT].List) == 0)
	is.True(len(stats.NotableData[entity.ALL_TRIPLE_WORDS_COVERED_STAT].List) == 0)
	is.True(len(stats.NotableData[entity.COMBINED_HIGH_SCORING_STAT].List) == 0)
	is.True(len(stats.NotableData[entity.COMBINED_LOW_SCORING_STAT].List) == 0)
	is.True(len(stats.NotableData[entity.MANY_CHALLENGES_STAT].List) == 1)
	is.True(len(stats.NotableData[entity.ONE_PLAYER_PLAYS_EVERY_POWER_TILE_STAT].List) == 0)
	is.True(len(stats.NotableData[entity.ONE_PLAYER_PLAYS_EVERY_E_STAT].List) == 0)
	is.True(len(stats.NotableData[entity.FOUR_OR_MORE_CONSECUTIVE_BINGOS_STAT].List) == 0)
	listStatStore.Disconnect()
}

func TestNotable(t *testing.T) {
	is := is.New(t)
	recreateDB()

	stats := InstantiateNewStats("1", "2")
	listStatStore, err := statstore.NewListStatStore(TestingDBConnStr + " dbname=liwords_test")
	is.NoErr(err)

	manyDoubleWordsCoveredStats, _ := InstantiateNewStatsWithHistory("./testdata/many_double_words_covered.gcg", listStatStore)
	allTripleLettersCoveredStats, _ := InstantiateNewStatsWithHistory("./testdata/all_triple_letters_covered.gcg", listStatStore)
	allTripleWordsCoveredStats, _ := InstantiateNewStatsWithHistory("./testdata/all_triple_words_covered.gcg", listStatStore)
	combinedHighScoringStats, _ := InstantiateNewStatsWithHistory("./testdata/combined_high_scoring.gcg", listStatStore)
	combinedLowScoringStats, _ := InstantiateNewStatsWithHistory("./testdata/combined_low_scoring.gcg", listStatStore)
	everyPowerTileStats, _ := InstantiateNewStatsWithHistory("./testdata/every_power_tile.gcg", listStatStore)
	everyEStats, _ := InstantiateNewStatsWithHistory("./testdata/every_e.gcg", listStatStore)
	manyChallengesStats, _ := InstantiateNewStatsWithHistory("./testdata/many_challenges.gcg", listStatStore)
	fourOrMoreConsecutiveBingosStats, _ := InstantiateNewStatsWithHistory("./testdata/four_or_more_consecutive_bingos.gcg", listStatStore)

	AddStats(stats, manyDoubleWordsCoveredStats)
	AddStats(stats, allTripleLettersCoveredStats)
	AddStats(stats, allTripleWordsCoveredStats)
	AddStats(stats, combinedHighScoringStats)
	AddStats(stats, combinedLowScoringStats)
	AddStats(stats, manyChallengesStats)
	AddStats(stats, fourOrMoreConsecutiveBingosStats)
	AddStats(stats, everyPowerTileStats)
	AddStats(stats, everyEStats)

	err = Finalize(stats, listStatStore, []string{
		"./testdata/many_double_words_covered.gcg",
		"./testdata/all_triple_letters_covered.gcg",
		"./testdata/all_triple_words_covered.gcg",
		"./testdata/combined_high_scoring.gcg",
		"./testdata/combined_low_scoring.gcg",
		"./testdata/every_power_tile.gcg",
		"./testdata/every_e.gcg",
		"./testdata/many_challenges.gcg",
		"./testdata/four_or_more_consecutive_bingos.gcg",
	}, "", "")
	is.NoErr(err)

	is.True(len(stats.NotableData[entity.MANY_DOUBLE_WORDS_COVERED_STAT].List) == 1)
	is.True(len(stats.NotableData[entity.ALL_TRIPLE_LETTERS_COVERED_STAT].List) == 1)
	is.True(len(stats.NotableData[entity.ALL_TRIPLE_WORDS_COVERED_STAT].List) == 1)
	is.True(len(stats.NotableData[entity.COMBINED_HIGH_SCORING_STAT].List) == 1)
	is.True(len(stats.NotableData[entity.COMBINED_LOW_SCORING_STAT].List) == 1)
	is.True(len(stats.NotableData[entity.MANY_CHALLENGES_STAT].List) == 2)
	is.True(len(stats.NotableData[entity.ONE_PLAYER_PLAYS_EVERY_POWER_TILE_STAT].List) == 1)
	is.True(len(stats.NotableData[entity.ONE_PLAYER_PLAYS_EVERY_E_STAT].List) == 1)
	is.True(len(stats.NotableData[entity.FOUR_OR_MORE_CONSECUTIVE_BINGOS_STAT].List) == 1)
	listStatStore.Disconnect()

}

func TestPhonyHooks(t *testing.T) {
	is := is.New(t)

	recreateDB()
	listStatStore, err := statstore.NewListStatStore(TestingDBConnStr + " dbname=liwords_test")
	is.NoErr(err)
	stats := InstantiateNewStats("1", "2")

	phonyHooks, err := InstantiateNewStatsWithHistory("./testdata/phonies_formed.gcg", listStatStore)
	is.NoErr(err)

	AddStats(stats, phonyHooks)

	err = Finalize(stats, listStatStore, []string{
		"./testdata/phonies_formed.gcg",
	}, "", "")
	is.NoErr(err)

	// fmt.Println(StatsToString(stats))
	is.True(stats.PlayerOneData[entity.UNCHALLENGED_PHONIES_STAT].Total == 2)
	is.True(stats.PlayerTwoData[entity.UNCHALLENGED_PHONIES_STAT].Total == 2)

	listStatStore.Disconnect()
}

func isEqual(statsOne *entity.Stats, statsTwo *entity.Stats) bool {
	return statsOne.PlayerOneId == statsTwo.PlayerOneId &&
		statsOne.PlayerTwoId == statsTwo.PlayerTwoId &&
		isStatItemListEqual(statsOne.PlayerOneData, statsTwo.PlayerOneData) &&
		isStatItemListEqual(statsOne.PlayerTwoData, statsTwo.PlayerTwoData) &&
		isStatItemListEqual(statsOne.NotableData, statsTwo.NotableData)
}

func isStatItemListEqual(statItemListOne map[string]*entity.StatItem, statItemListTwo map[string]*entity.StatItem) bool {
	for key, value := range statItemListOne {
		statsItemOne := value
		statsItemTwo := statItemListTwo[key]
		if !isStatItemEqual(statsItemOne, statsItemTwo) {
			return false
		}
	}
	return true
}

func isStatItemEqual(statItemOne *entity.StatItem, statItemTwo *entity.StatItem) bool {
	return statItemOne.Total == statItemTwo.Total &&
		// Floating points nonsense
		//isStatItemAveragesEqual(statItemOne.Averages, statItemTwo.Averages) &&
		isStatItemSubitemsEqual(statItemOne.Subitems, statItemTwo.Subitems)
}

func isStatItemAveragesEqual(arrOne []float64, arrTwo []float64) bool {
	if len(arrOne) != len(arrTwo) {
		return false
	}
	for i := 0; i < len(arrOne); i++ {
		if arrOne[i] != arrTwo[i] {
			return false
		}
	}
	return true
}

func isStatItemSubitemsEqual(mapOne map[string]int, mapTwo map[string]int) bool {
	for key, value := range mapOne {
		if value != mapTwo[key] {
			return false
		}
	}
	for key, value := range mapTwo {
		if value != mapOne[key] {
			return false
		}
	}
	return true
}

func StatsToString(stats *entity.Stats) string {
	s := "Player One Data:\n" + statItemListToString(stats.PlayerOneData)
	s += "\nPlayer Two Data:\n" + statItemListToString(stats.PlayerTwoData)
	s += "\nNotable Data:\n" + statItemListToString(stats.NotableData)
	return s
}

func statItemListToString(data map[string]*entity.StatItem) string {
	s := ""
	for key, statItem := range data {
		s += statItemToString(key, statItem)
	}
	return s
}

func statItemToString(key string, statItem *entity.StatItem) string {
	subitemString := ""
	if statItem.Subitems != nil {
		subitemString = subitemsToString(statItem.Subitems)
	}
	return fmt.Sprintf("  %s:\n    Total: %d\n    Subitems:\n%s\n    List:\n%s", key, statItem.Total, subitemString, listItemToString(statItem.List))
}

func averagesToString(averages []float64) string {
	s := ""
	for i := 0; i < len(averages); i++ {
		s += fmt.Sprintf("%.2f", averages[i])
	}
	return s
}

func subitemsToString(subitems map[string]int) string {
	s := ""
	for key, value := range subitems {
		s += fmt.Sprintf("      %s -> %d\n", key, value)
	}
	return s
}

func listItemToString(listStat []*entity.ListItem) string {
	s := ""
	for _, item := range listStat {
		s += fmt.Sprintf("      %s, %s, %d, %v\n", item.GameId, item.PlayerId, item.Time, item.Item)
	}
	return s
}
