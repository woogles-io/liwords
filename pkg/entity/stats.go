package entity

import (
	"fmt"
	"unicode"
	"strings"
	"errors"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
)

type ListItem struct {
	Word        string
	Probability int
	Score       int
	GameId      string
}

type StatItemDataType int

const (
	SingleType StatItemDataType = iota
	ListType
	DualType
)

type StatItemProcessingType int

const (
	TurnType StatItemProcessingType = iota
	GameType
)

type NotableItem struct {
	Name               string
	Description        string
	Total              int
	Minimum            int
	Maximum            int
	GameList           []string
	AddFunction        func(*NotableItem, *pb.GameEvent)
}

type StatItem struct {
	Name               string
	Description        string
	Total              int
	DataType           StatItemDataType
	ProcessingType     StatItemProcessingType
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
	NotableData   []*NotableItem
}

func InstantiateNewStats() (*Stats) {
	return &Stats{
		PlayerOneId:   1,
		PlayerTwoId:   2,
		GamesPlayed:   1,
		PlayerOneData: instantiatePlayerData(),
		PlayerTwoData: instantiatePlayerData(),
		NotableData:   instantiateNotableData()}
}

func (stats *Stats) AddGameToStats(history *pb.GameHistory, id string) (error) {

	// Loop through all the turns in the game history
	turns := history.GetTurns()
	for _, turn := range turns {
		for _, event := range turn.Events {
			if history.Players[0].Nickname == event.Nickname || 
			   (history.Players[1].Nickname == event.Nickname && history.SecondWentFirst){
				addTurnToStatItems(stats.PlayerOneData, event, id)
			} else {
				addTurnToStatItems(stats.PlayerTwoData, event, id)
			}
			addTurnToNotableItems(stats.NotableData, event)
		}
	}
	confirmNotableItems(stats.NotableData, id)
	
	stats.GamesPlayed++

	return nil
}

func (stats *Stats) AddStatsToStats(otherStats *Stats) (error) {

	if stats.PlayerOneId != otherStats.PlayerOneId && stats.PlayerOneId != otherStats.PlayerTwoId {
		return errors.New("Stats do not share an identical PlayerOneId")
	}

	otherPlayerOneData := otherStats.PlayerOneData
	otherPlayerTwoData := otherStats.PlayerTwoData

	if stats.PlayerOneId == otherStats.PlayerTwoId {
		otherPlayerOneData = otherStats.PlayerTwoData
		otherPlayerTwoData = otherStats.PlayerOneData
	}

	err := combinePlayerData(stats.PlayerOneData, otherPlayerOneData)
	
	if err != nil {
		return err
	}

	err = combinePlayerData(stats.PlayerTwoData, otherPlayerTwoData)
	
	if err != nil {
		return err
	}

	err = combineNotableData(stats.NotableData, otherStats.NotableData)
	
	if err != nil {
		return err
	}

	return nil
}

func (stats *Stats) ToString() (string) {
	s := "Player One Data:\n" + playerDataToString(stats.PlayerOneData)
	s += "\nPlayer Two Data:\n" + playerDataToString(stats.PlayerTwoData)
	s += "\nNotable Data:\n" + notableDataToString(stats.NotableData)
	return s
}

func playerDataToString(playerData []*StatItem) (string) {
	s := ""
	for _, statItem := range playerData {
		s += statItemToString(statItem)
	}
	return s
}

func notableDataToString(notableData []*NotableItem) (string) {
	s := ""
	for _, notableItem := range notableData {
		s += notableItemToString(notableItem)
	}
	return s
}

func statItemToString(statItem *StatItem) (string) {
	return fmt.Sprintf("  %s:\n    Total: %d\n    List:\n%s", statItem.Name, statItem.Total, listItemToString(statItem.List))
}

func notableItemToString(notableItem *NotableItem) (string) {
	return fmt.Sprintf("     %s: %d, %s\n", notableItem.Name, notableItem.Total, strings.Join(notableItem.GameList[:], ","))
}

func listItemToString(listStat []*ListItem) (string) {
	s := ""
	for _, wordItem := range listStat {
		s += fmt.Sprintf("      %s, %d, %d, %s\n", wordItem.Word, wordItem.Score, wordItem.Probability, wordItem.GameId)
	}
	return s
}

func instantiatePlayerData() []*StatItem {
	return []*StatItem{

&StatItem{Name: "Bingos",
		Description:        "The list of bingos played",
		Total:              0,
		DataType:            DualType,
		ProcessingType: TurnType,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addBingos},

&StatItem{Name: "Triple Triples",
		Description:        "Number of triple triples played",
		Total:              0,
		DataType:           DualType,
		ProcessingType: TurnType,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addTripleTriples},

&StatItem{Name: "Bingo Nines or Above",
		Description:        "The list of bingos that were nines tiles or above",
		Total:              0,
		DataType:           DualType,
		ProcessingType: TurnType,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addBingoNinesOrAbove}}
}

func instantiateNotableData() []*NotableItem {
	return []*NotableItem{&NotableItem{Name: "No Blanks Played",
		Description:        "Games in which no blanks are played",
		Minimum: 0,
		Maximum: 0,
		Total:              0,
		GameList:           []string{},
		AddFunction:        addNoBlanksPlayed}}
}

func addTurnToStatItems(statItems []*StatItem, event *pb.GameEvent, id string) {
	for _, statItem := range statItems {
		statItem.AddFunction(statItem, event, id)
	}
}

func addTurnToNotableItems(notableItems []*NotableItem, event *pb.GameEvent) {
	for _, notableItem := range notableItems {
		notableItem.AddFunction(notableItem, event)
	}
}

func combinePlayerData(statItems []*StatItem, otherStatItems []*StatItem) error {
	if len(statItems) != len(otherStatItems) {
		return errors.New("StatItem lists do not match in length")
	}
	for i := 0; i < len(statItems); i++ {
		item := statItems[i]
		itemOther := otherStatItems[i]
		if item.Description != itemOther.Description {
			return errors.New("StatItem descriptions do not match")
		}
		combineItems(item, itemOther)
	}
	return nil
}

func combineNotableData(notableItems []*NotableItem, otherNotableItems []*NotableItem) error {
	if len(notableItems) != len(otherNotableItems) {
		return errors.New("NotableItem lists do not match in length")
	}
	for i := 0; i < len(notableItems); i++ {
		item := notableItems[i]
		itemOther := otherNotableItems[i]
		if item.Description != itemOther.Description {
			return errors.New("NotableItem descriptions do not match")
		}
		combineNotableItems(item, itemOther)
	}
	return nil
}

func confirmNotableItems(notableItems []*NotableItem, id string) {
	for i := 0; i < len(notableItems); i++ {
		item := notableItems[i]
		if item.Total >= item.Minimum && item.Total <= item.Maximum {
			item.GameList = append(item.GameList, id)
		}
		item.Total = 0
	}
}

func combineItems(statItem *StatItem, otherStatItem *StatItem) {
	if statItem.DataType == SingleType {
		statItem.Total += otherStatItem.Total
	} else {
		statItem.List = append(statItem.List, otherStatItem.List...)
	}
}

func combineNotableItems(notableItem *NotableItem, otherNotableItem *NotableItem) {
	notableItem.GameList = append(notableItem.GameList, otherNotableItem.GameList...)
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
	if statItem.DataType == SingleType {
		statItem.Total++
	} else if statItem.DataType == ListType {
		statItem.List = append(statItem.List, &ListItem{Word:"YEET", Probability: 1, Score: 1, GameId: id})
	} else if statItem.DataType == DualType {
		statItem.Total++
		statItem.List = append(statItem.List, &ListItem{Word:"YEET", Probability: 1, Score: 1, GameId: id})
	}
}

func addNoBlanksPlayed(notableItem *NotableItem, event *pb.GameEvent) {
	tiles := event.PlayedTiles
    for _, char := range tiles {
    	if unicode.IsLower(char) {
			notableItem.Total++
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

func getNumberOfTilesPlayed(play string) (int) {
	sum := 0
    for _, char := range play {
    	// Unicode nonsense, Cesar plz help
    	if !unicode.IsPunct(char) {
			sum++
    	}
    }
    return sum
}