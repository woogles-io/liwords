package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/macondo/alphabet"
	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/macondo/gcgio"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	os.Exit(m.Run())
}

var DefaultConfig = macondoconfig.Config{
	LexiconPath:               os.Getenv("LEXICON_PATH"),
	LetterDistributionPath:    os.Getenv("LETTER_DISTRIBUTION_PATH"),
	DefaultLexicon:            "CSW19",
	DefaultLetterDistribution: "English",
}

func InstantiateNewStatsWithHistory(filename string) (*entity.Stats, error) {

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
	stats := entity.InstantiateNewStats(playerOneId, playerTwoId)
	if err != nil {
		return nil, err
	}
	stats.AddGame(history, filename)
	return stats, nil
}

func JoshNationalsFromGames(useJSON bool) (*entity.Stats, error) {
	annotatedGamePrefix := "josh_nationals_round_"
	stats := entity.InstantiateNewStats("1", "2")

	for i := 1; i <= 31; i++ {
		annotatedGame := fmt.Sprintf("./testdata/%s%d.gcg", annotatedGamePrefix, i)
		//fmt.Println(annotatedGame)
		otherStats, _ := InstantiateNewStatsWithHistory(annotatedGame)
		if useJSON {
			bytes, err := json.Marshal(otherStats)
			if err != nil {
				fmt.Println(err)
				return otherStats, err
			}
			// It is assumed that in production the json is stored in a database
			// and remains untouched until it is retrieved later for stats
			// aggregation
			var otherStatsFromJSON entity.Stats
			err = json.Unmarshal(bytes, &otherStatsFromJSON)
			if err != nil {
				return otherStats, err
			}
			otherStats = &otherStatsFromJSON
		}
		stats.AddStats(otherStats)
	}
	return stats, nil
}

func TestStats(t *testing.T) {

	is := is.New(t)
	stats, error := JoshNationalsFromGames(false)
	is.True(error == nil)
	statsJSON, error := JoshNationalsFromGames(true)
	is.True(error == nil)
	stats.Finalize()
	statsJSON.Finalize()
	is.True(isEqual(stats, statsJSON))
	// fmt.Println(StatsToString(stats))
	is.True(stats.PlayerOneData[entity.BINGO_NINES_OR_ABOVE_STAT].Total == 1)
	is.True(stats.PlayerOneData[entity.NO_BINGOS_STAT].Total == 0)
	is.True(stats.PlayerOneData[entity.BINGOS_STAT].Total == 75)
	is.True(stats.PlayerOneData[entity.CHALLENGED_PHONIES_STAT].Total == 4)
	is.True(stats.PlayerOneData[entity.CHALLENGES_STAT].Total == 20)
	is.True(stats.PlayerOneData[entity.CHALLENGES_WON_STAT].Total == 9)
	is.True(stats.PlayerOneData[entity.CHALLENGES_LOST_STAT].Total == 11)
	is.True(stats.PlayerOneData[entity.COMMENTS_STAT].Total == 174)
	is.True(stats.PlayerOneData[entity.DRAWS_STAT].Total == 0)
	is.True(stats.PlayerOneData[entity.EXCHANGES_STAT].Total == 3)
	is.True(stats.PlayerOneData[entity.FIRSTS_STAT].Total == 16)
	is.True(stats.PlayerOneData[entity.GAMES_STAT].Total == 31)
	is.True(stats.PlayerOneData[entity.HIGH_GAME_STAT].Total == 619)
	is.True(stats.PlayerOneData[entity.HIGH_TURN_STAT].Total == 167)
	is.True(stats.PlayerOneData[entity.LOSSES_STAT].Total == 11)
	is.True(stats.PlayerOneData[entity.LOW_GAME_STAT].Total == 359)
	is.True(stats.PlayerOneData[entity.MISTAKES_STAT].Total == 134)
	is.True(stats.PlayerOneData[entity.PLAYS_THAT_WERE_CHALLENGED_STAT].Total == 18)
	is.True(stats.PlayerOneData[entity.SCORE_STAT].Total == 13977)
	is.True(stats.PlayerOneData[entity.TURNS_STAT].Total == 375)
	is.True(stats.PlayerOneData[entity.TURNS_WITH_BLANK_STAT].Total == 72)
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
}

func TestNotable(t *testing.T) {
	is := is.New(t)
	stats := entity.InstantiateNewStats("1", "2")
	everyPowerTileStats, _ := InstantiateNewStatsWithHistory("./testdata/jesse_vs_ayo.gcg")
	everyEStats, _ := InstantiateNewStatsWithHistory("./testdata/josh_vs_jesse.gcg")
	stats.AddStats(everyPowerTileStats)
	stats.AddStats(everyEStats)
	//fmt.Println(stats.ToString())

	is.True(len(stats.NotableData[entity.ONE_PLAYER_PLAYS_EVERY_POWER_TILE_STAT].List) == 1)
	is.True(len(stats.NotableData[entity.ONE_PLAYER_PLAYS_EVERY_E_STAT].List) == 1)
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
		isStatItemSubitemsEqual(statItemOne.Subitems, statItemTwo.Subitems) &&
		statItemOne.HasMeaningfulTotal == statItemTwo.HasMeaningfulTotal
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
	averagesString := ""
	if statItem.Averages != nil {
		averagesString = averagesToString(statItem.Averages)
	}
	return fmt.Sprintf("  %s:\n    Total: %d\n    Averages: %s\nSubitems:\n%s\n    List:\n%s", key, statItem.Total, averagesString, subitemString, listItemToString(statItem.List))
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
	for _, wordItem := range listStat {
		s += fmt.Sprintf("      %s, %d, %d, %s\n", wordItem.Word, wordItem.Score, wordItem.Probability, wordItem.GameId)
	}
	return s
}
