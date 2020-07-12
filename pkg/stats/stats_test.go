package stats

import (
	"fmt"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/macondo/gcgio"
	"testing"
	"github.com/domino14/macondo/alphabet"
	"github.com/matryer/is"
)

func convertStatItemListToMap(statItems []*entity.StatItem) (map[string]*entity.StatItem) {
	statItemMap := make(map[string]*entity.StatItem)
	for _, statItem := range statItems {
		statItemMap[statItem.Name] = statItem
	}
	return statItemMap
}

func InstantiateNewStatsWithHistory(filename string) (*entity.Stats, error) {
	history, err := gcgio.ParseGCG(filename)

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

func TestStats(t *testing.T) {
	is := is.New(t)
	annotatedGamePrefix := "josh_nationals_round_"
	stats :=  entity.InstantiateNewStats(1, 2)

	for i := 1; i <= 31; i++ {
		annotatedGame := fmt.Sprintf("./testdata/%s%d.gcg", annotatedGamePrefix, i)
		//fmt.Println(annotatedGame)
		otherStats, _ := InstantiateNewStatsWithHistory(annotatedGame)
		stats.AddStatsToStats(otherStats)
	}
	playerOneStatsMap := convertStatItemListToMap(stats.PlayerOneData)
	fmt.Println(stats.ToString())
	is.True(playerOneStatsMap["Games"].Total == 31)
	is.True(playerOneStatsMap["Firsts"].Total == 16)
	is.True(playerOneStatsMap["Turns"].Total == 375)
	is.True(playerOneStatsMap["Vertical Openings"].Total == 0)
	is.True(playerOneStatsMap["Bingos"].Total == 75)
	is.True(playerOneStatsMap["Bingoless Games"].Total == 0)
	is.True(playerOneStatsMap["Exchanges"].Total == 3)
	is.True(playerOneStatsMap["Bingo Nines or Above"].Total == 1)
	is.True(playerOneStatsMap["Challenged Phonies"].Total == 4)
	is.True(playerOneStatsMap["Challenges"].Total == 20)
	is.True(playerOneStatsMap["Challenges Won"].Total == 9)
	is.True(playerOneStatsMap["Challenges Lost"].Total == 11)
	is.True(playerOneStatsMap["Plays That Were Challenged"].Total == 18)
	is.True(playerOneStatsMap["Highest Scoring Turn"].Total == 167)
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
	is.True(playerOneStatsMap["Turns With at Least One Blank"].Total == 72)
	is.True(playerOneStatsMap["Comments"].Total == 174)
}

func TestNotable(t *testing.T) {
	is := is.New(t)
	stats :=  entity.InstantiateNewStats(1, 2)
	everyPowerTileStats, _ := InstantiateNewStatsWithHistory("./testdata/jesse_vs_ayo.gcg")
	everyEStats, _ := InstantiateNewStatsWithHistory("./testdata/josh_vs_jesse.gcg")
	stats.AddStatsToStats(everyPowerTileStats)
	stats.AddStatsToStats(everyEStats)
	notableStatsMap := convertStatItemListToMap(stats.NotableData)
	is.True(len(notableStatsMap["One Player Plays Every Power Tile"].List) == 1)
	is.True(len(notableStatsMap["One Player Plays Every E"].List) == 1)
}
