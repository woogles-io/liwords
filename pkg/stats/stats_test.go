package stats

import (
	"os"
	"fmt"
	"encoding/json"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/macondo/gcgio"
	"testing"
	"github.com/domino14/macondo/alphabet"
	"github.com/matryer/is"
	macondoconfig "github.com/domino14/macondo/config"
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

func convertStatItemListToMap(statItems []*entity.StatItem) (map[string]*entity.StatItem) {
	statItemMap := make(map[string]*entity.StatItem)
	for _, statItem := range statItems {
		statItemMap[statItem.Name] = statItem
	}
	return statItemMap
}

func InstantiateNewStatsWithHistory(filename string) (*entity.Stats, error) {

	history, err := gcgio.ParseGCG(&DefaultConfig, filename)

	// For these tests, ensure "jvc" and "Josh" always have a player id of 1
	playerOneId := 1
	playerTwoId := 2
	if history.Players[1].Nickname == "jvc" || history.Players[1].Nickname == "Josh" {
		playerOneId = 2
		playerTwoId = 1
	}
	stats := entity.InstantiateNewStats(playerOneId, playerTwoId)
	if err != nil {
		return nil, err
	}
	stats.AddGameToStats(history, filename)
	return stats, nil
}

func JoshNationalsFromGames(useJSON bool) (*entity.Stats, error) {
	annotatedGamePrefix := "josh_nationals_round_"
	stats :=  entity.InstantiateNewStats(1, 2)

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
		stats.AddStatsToStats(otherStats)
	}
	return stats, nil
}

func TestStats(t *testing.T) {

	is := is.New(t)
	stats, error := JoshNationalsFromGames(false)
	is.True(error == nil)
	statsJSON, error := JoshNationalsFromGames(true)
	is.True(error == nil)
	is.True(isEqual(stats, statsJSON))
	playerOneStatsMap := convertStatItemListToMap(stats.PlayerOneData)
	fmt.Println(StatsToString(stats))
	is.True(playerOneStatsMap["Bingo Nines or Above"].Total == 1)
	is.True(playerOneStatsMap["Bingoless Games"].Total == 0)
	is.True(playerOneStatsMap["Bingos"].Total == 75)
	is.True(playerOneStatsMap["Challenged Phonies"].Total == 4)
	is.True(playerOneStatsMap["Challenges"].Total == 20)
	is.True(playerOneStatsMap["Challenges Won"].Total == 9)
	is.True(playerOneStatsMap["Challenges Lost"].Total == 11)
	is.True(playerOneStatsMap["Comments"].Total == 174)
	is.True(playerOneStatsMap["Draws"].Total == 0)
	is.True(playerOneStatsMap["Exchanges"].Total == 3)
	is.True(playerOneStatsMap["Firsts"].Total == 16)
	is.True(playerOneStatsMap["Games"].Total == 31)
	is.True(playerOneStatsMap["Highest Scoring Game"].Total == 619)
	is.True(playerOneStatsMap["Highest Scoring Turn"].Total == 167)
	is.True(playerOneStatsMap["Losses"].Total == 11)
	is.True(playerOneStatsMap["Lowest Scoring Game"].Total == 359)
	is.True(playerOneStatsMap["Mistakes"].Total == 134)
	is.True(playerOneStatsMap["Plays That Were Challenged"].Total == 18)
	is.True(playerOneStatsMap["Score"].Total == 13977)
	is.True(playerOneStatsMap["Turns"].Total == 375)
	is.True(playerOneStatsMap["Turns With at Least One Blank"].Total == 72)
	is.True(playerOneStatsMap["Vertical Openings"].Total == 0)
	is.True(playerOneStatsMap["Wins"].Total == 20)
	is.True(playerOneStatsMap["Tiles Played"].Total == 1556)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["A"] == 134)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["B"] == 25)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["C"] == 26)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["D"] == 71)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["E"] == 186)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["F"] == 24)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["G"] == 50)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["H"] == 34)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["I"] == 144)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["J"] == 13)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["K"] == 14)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["L"] == 72)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["M"] == 32)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["N"] == 102)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["O"] == 125)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["P"] == 21)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["Q"] == 16)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["R"] == 94)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["S"] == 44)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["T"] == 94)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["U"] == 67)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["V"] == 30)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["W"] == 31)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["X"] == 16)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["Y"] == 33)
	is.True(playerOneStatsMap["Tiles Played"].Subitems["Z"] == 19)
	is.True(playerOneStatsMap["Tiles Played"].Subitems[string(alphabet.BlankToken)] == 39)
}

func TestNotable(t *testing.T) {
	is := is.New(t)
	stats :=  entity.InstantiateNewStats(1, 2)
	everyPowerTileStats, _ := InstantiateNewStatsWithHistory("./testdata/jesse_vs_ayo.gcg")
	everyEStats, _ := InstantiateNewStatsWithHistory("./testdata/josh_vs_jesse.gcg")
	stats.AddStatsToStats(everyPowerTileStats)
	stats.AddStatsToStats(everyEStats)
	notableStatsMap := convertStatItemListToMap(stats.NotableData)
	//fmt.Println(stats.ToString())

	is.True(len(notableStatsMap["One Player Plays Every Power Tile"].List) == 1)
	is.True(len(notableStatsMap["One Player Plays Every E"].List) == 1)
}

func isEqual(statsOne *entity.Stats, statsTwo *entity.Stats) bool {
	return statsOne.PlayerOneId == statsTwo.PlayerOneId &&
	       statsOne.PlayerTwoId == statsTwo.PlayerTwoId &&
	       isStatItemListEqual(statsOne.PlayerOneData, statsTwo.PlayerOneData) &&
	       isStatItemListEqual(statsOne.PlayerTwoData, statsTwo.PlayerTwoData) &&
	       isStatItemListEqual(statsOne.NotableData, statsTwo.NotableData)
}

func isStatItemListEqual(statItemListOne []*entity.StatItem, statItemListTwo []*entity.StatItem) bool {
	if len(statItemListOne) != len(statItemListTwo) {
		return false
	}
	for i := 0; i < len(statItemListOne); i++ {
		statsItemOne := statItemListOne[i]
		statsItemTwo := statItemListTwo[i]
		if !isStatItemEqual(statsItemOne, statsItemTwo) {
			return false
		}
	}
	return true
}

func isStatItemEqual(statItemOne *entity.StatItem, statItemTwo *entity.StatItem) bool {
	return statItemOne.Name == statItemTwo.Name &&
	       statItemOne.Description == statItemTwo.Description &&
	       statItemOne.Total == statItemTwo.Total &&
	       isStatItemAveragesEqual(statItemOne.Averages, statItemTwo.Averages) &&
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

func statItemListToString(data []*entity.StatItem) string {
	s := ""
	for _, statItem := range data {
		s += statItemToString(statItem)
	}
	return s
}

func statItemToString(statItem *entity.StatItem) string {
	subitemString := ""
	if statItem.Subitems != nil {
		subitemString = subitemsToString(statItem.Subitems)
	}
	return fmt.Sprintf("  %s:\n    Total: %d\n    Subitems:\n%s\n    List:\n%s", statItem.Name, statItem.Total, subitemString, listItemToString(statItem.List))
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
