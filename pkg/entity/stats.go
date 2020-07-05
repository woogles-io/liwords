package entity

import (
	"errors"
	"fmt"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
	"unicode"
)

type ListItem struct {
	Word        string
	Probability int
	Score       int
	GameId      string
}

type StatItemType int

const (
	SingleType StatItemType = iota
	ListType
	DualType
)

type StatItem struct {
	Name               string
	Description        string
	Minimum            int
	Maximum            int
	Total              int
	Type               StatItemType
	Denominator        *StatItem
	IsProfileStat      bool
	List               []*ListItem
	HasMeaningfulTotal bool
	AddFunction        func(*StatItem, *pb.GameEvent, string)
}

type Stats struct {
	PlayerOneId   int
	PlayerTwoId   int
	GamesPlayed   int
	PlayerOneData []*StatItem
	PlayerTwoData []*StatItem
	NotableData   []*StatItem
}

func InstantiateNewStats() *Stats {
	return &Stats{
		PlayerOneId:   1,
		PlayerTwoId:   2,
		GamesPlayed:   0,
		PlayerOneData: instantiatePlayerData(),
		PlayerTwoData: instantiatePlayerData(),
		NotableData:   instantiateNotableData()}
}

func (stats *Stats) AddGameToStats(history *pb.GameHistory, id string) error {

	// Stats not computed on an event basis:
	//   Wins
	//   Score
	//   Firsts
	//   Vertical Openings per First
	//   Horizontal Openings per First

	// Loop through all the turns in the game history
	turns := history.GetTurns()
	for _, turn := range turns {
		for _, event := range turn.Events {
			fmt.Println(event)
			if history.Players[0].Nickname == event.Nickname ||
				(history.Players[1].Nickname == event.Nickname && history.SecondWentFirst) {
				addEventToStatItems(stats.PlayerOneData, event, id)
			} else {
				addEventToStatItems(stats.PlayerTwoData, event, id)
			}
			addEventToStatItems(stats.NotableData, event, id)
		}
	}
	confirmNotableItems(stats.NotableData, id)
	return nil
}

func (stats *Stats) AddStatsToStats(otherStats *Stats) error {

	if stats.PlayerOneId != otherStats.PlayerOneId && stats.PlayerOneId != otherStats.PlayerTwoId {
		return errors.New("Stats do not share an identical PlayerOneId")
	}

	otherPlayerOneData := otherStats.PlayerOneData
	otherPlayerTwoData := otherStats.PlayerTwoData

	if stats.PlayerOneId == otherStats.PlayerTwoId {
		otherPlayerOneData = otherStats.PlayerTwoData
		otherPlayerTwoData = otherStats.PlayerOneData
	}

	err := combineStatItemLists(stats.PlayerOneData, otherPlayerOneData)

	if err != nil {
		return err
	}

	err = combineStatItemLists(stats.PlayerTwoData, otherPlayerTwoData)

	if err != nil {
		return err
	}

	err = combineStatItemLists(stats.NotableData, otherStats.NotableData)

	if err != nil {
		return err
	}

	return nil
}

func (stats *Stats) ToString() string {
	s := "Player One Data:\n" + statItemListToString(stats.PlayerOneData)
	s += "\nPlayer Two Data:\n" + statItemListToString(stats.PlayerTwoData)
	s += "\nNotable Data:\n" + statItemListToString(stats.NotableData)
	return s
}

func statItemListToString(data []*StatItem) string {
	s := ""
	for _, statItem := range data {
		s += statItemToString(statItem)
	}
	return s
}

func statItemToString(statItem *StatItem) string {
	return fmt.Sprintf("  %s:\n    Total: %d\n    List:\n%s", statItem.Name, statItem.Total, listItemToString(statItem.List))
}

func listItemToString(listStat []*ListItem) string {
	s := ""
	for _, wordItem := range listStat {
		s += fmt.Sprintf("      %s, %d, %d, %s\n", wordItem.Word, wordItem.Score, wordItem.Probability, wordItem.GameId)
	}
	return s
}

func instantiatePlayerData() []*StatItem {
	gamesStat := &StatItem{Name: "Games",
		Description:        "The number of games played",
		Total:              0,
		Type:               SingleType,
		Denominator:        nil,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addGames}
	bingosStat := &StatItem{Name: "Bingos",
		Description:        "The list of bingos played",
		Total:              0,
		Type:               DualType,
		Denominator:        gamesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addBingos}
	tripleTriplesStat := &StatItem{Name: "Triple Triples",
		Description:        "Number of triple triples played",
		Total:              0,
		Type:               DualType,
		Denominator:        gamesStat,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addTripleTriples}
	bingoNinesOrAboveStat := &StatItem{Name: "Bingo Nines or Above",
		Description:        "The list of bingos that were nines tiles or above",
		Total:              0,
		Type:               DualType,
		Denominator:        gamesStat,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addBingoNinesOrAbove}

	return []*StatItem{gamesStat,
		bingosStat,
		tripleTriplesStat,
		bingoNinesOrAboveStat}
}

func instantiateNotableData() []*StatItem {
	return []*StatItem{&StatItem{Name: "No Blanks Played",
		Description:        "Games in which no blanks are played",
		Minimum:            0,
		Maximum:            0,
		Total:              0,
		Type:               ListType,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: false,
		AddFunction:        addBlanksPlayed}}
}

func addEventToStatItems(statItems []*StatItem, event *pb.GameEvent, id string) {
	for _, statItem := range statItems {
		statItem.AddFunction(statItem, event, id)
	}
}

func combineStatItemLists(statItems []*StatItem, otherStatItems []*StatItem) error {
	if len(statItems) != len(otherStatItems) {
		return errors.New("StatItem lists do not match in length")
	}
	for i := 0; i < len(statItems); i++ {
		item := statItems[i]
		itemOther := otherStatItems[i]
		if item.Name != itemOther.Name {
			return errors.New("StatItem names do not match")
		}
		combineItems(item, itemOther)
	}
	return nil
}

func confirmNotableItems(statItems []*StatItem, id string) {
	for i := 0; i < len(statItems); i++ {
		item := statItems[i]
		if item.Total >= item.Minimum && item.Total <= item.Maximum {
			item.List = append(item.List, &ListItem{Word: "", Probability: 0, Score: 0, GameId: id})
		}
		item.Total = 0
	}
}

func combineItems(statItem *StatItem, otherStatItem *StatItem) {
	if statItem.Type == SingleType {
		statItem.Total += otherStatItem.Total
	} else {
		statItem.List = append(statItem.List, otherStatItem.List...)
	}
}

func addGames(statItem *StatItem, event *pb.GameEvent, id string) {
	incrementStatItem(statItem, event, id)
}

func addBingos(statItem *StatItem, event *pb.GameEvent, id string) {
	if isBingo(event) {
		incrementStatItem(statItem, event, id)
	}
}

func addTripleTriples(statItem *StatItem, event *pb.GameEvent, id string) {
	if isTripleTriple(event) {
		incrementStatItem(statItem, event, id)
	}
}

func addBingoNinesOrAbove(statItem *StatItem, event *pb.GameEvent, id string) {
	if isTripleTriple(event) {
		incrementStatItem(statItem, event, id)
	}
}

func incrementStatItem(statItem *StatItem, event *pb.GameEvent, id string) {
	if statItem.Type == SingleType || statItem.Type == DualType {
		statItem.Total++
	}
	if statItem.Type == ListType || statItem.Type == DualType {
		statItem.List = append(statItem.List, &ListItem{Word: "YEET", Probability: 1, Score: 1, GameId: id})
	}
}

func addBlanksPlayed(statItem *StatItem, event *pb.GameEvent, id string) {
	tiles := event.PlayedTiles
	for _, char := range tiles {
		if unicode.IsLower(char) {
			statItem.Total++
		}
	}
}

func isTripleTriple(event *pb.GameEvent) bool {
	// Need to implement
	return false
}

func isBingo(event *pb.GameEvent) bool {
	return getNumberOfTilesPlayed(event.PlayedTiles) == 7
}

func isBingoNineOrAbove(event *pb.GameEvent) bool {
	// Need to implement
	return isBingo(event) && len(event.PlayedTiles) >= 9
}

func getNumberOfTilesPlayed(play string) int {
	sum := 0
	for _, char := range play {
		// Unicode nonsense, Cesar plz help
		if !unicode.IsPunct(char) {
			sum++
		}
	}
	return sum
}
