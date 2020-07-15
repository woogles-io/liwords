package entity

import (
	"errors"
	"fmt"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
)

type ListItem struct {
	Word        string
	Probability int
	Score       int
	GameId      string
}

type MistakeType string

const (
	KnowledgeMistakeType = "knowledge"
	FindingMistakeType   = "finding"
	VisionMistakeType    = "vision"
	TacticsMistakeType   = "tactics"
	StrategyMistakeType  = "strategy"
	TimeMistakeType      = "time"
	EndgameMistakeType   = "endgame"
)

type MistakeMagnitude string

const (
	LargeMistakeMagnitude  = "large"
	MediumMistakeMagnitude = "medium"
	SmallMistakeMagnitude  = "small"

	SaddestMistakeMagnitude = "saddest"
	SadderMistakeMagnitude  = "sadder"
	SadMistakeMagnitude     = "sad"

	UnspecifiedMistakeMagnitude = "unspecified"
)

var MistakeTypeMapping = map[string]int{KnowledgeMistakeType: 0,
	FindingMistakeType:  1,
	VisionMistakeType:   2,
	TacticsMistakeType:  3,
	StrategyMistakeType: 4,
	TimeMistakeType:     5,
	EndgameMistakeType:  6}

var MistakeMagnitudeMapping = map[string]int{LargeMistakeMagnitude: 1,
	MediumMistakeMagnitude:      2,
	SmallMistakeMagnitude:       3,
	UnspecifiedMistakeMagnitude: 0,
}

var MistakeMagnitudeAliases = map[string]string{LargeMistakeMagnitude: LargeMistakeMagnitude,
	MediumMistakeMagnitude:      MediumMistakeMagnitude,
	SmallMistakeMagnitude:       SmallMistakeMagnitude,
	UnspecifiedMistakeMagnitude: UnspecifiedMistakeMagnitude,
	"saddest":                   LargeMistakeMagnitude,
	"sadder":                    MediumMistakeMagnitude,
	"sad":                       SmallMistakeMagnitude,
}

type StatItemType int

const (
	SingleType StatItemType = iota
	ListType
	MinimumType
	MaximumType
)

type IncrementType int

const (
	EventType IncrementType = iota
	GameType
	FinalType
)

const MaxNotableInt = 1000000000

type StatItem struct {
	Name               string
	Description        string
	Minimum            int
	Maximum            int
	Total              int
	DataType           StatItemType
	IncrementType      IncrementType
	Denominator        *StatItem
	IsProfileStat      bool
	List               []*ListItem
	Subitems           map[string]int
	HasMeaningfulTotal bool
	AddFunction        func(*StatItem, *StatItem, *pb.GameHistory, int, string, bool)
}

type Stats struct {
	PlayerOneId   int
	PlayerTwoId   int
	PlayerOneData []*StatItem
	PlayerTwoData []*StatItem
	NotableData   []*StatItem
}

func InstantiateNewStats(playerOneId int, playerTwoId int) *Stats {
	return &Stats{
		PlayerOneId:   playerOneId,
		PlayerTwoId:   playerTwoId,
		PlayerOneData: instantiatePlayerData(),
		PlayerTwoData: instantiatePlayerData(),
		NotableData:   instantiateNotableData()}
}

func (stats *Stats) AddGameToStats(history *pb.GameHistory, id string) error {
	events := history.GetEvents()
	for i := 0; i < len(events); i++ {
		event := events[i]
		if history.Players[0].Nickname == event.Nickname ||
			(history.Players[1].Nickname == event.Nickname && history.SecondWentFirst) {
			incrementStatItems(stats.PlayerOneData, stats.PlayerTwoData, history, i, id, false)
		} else {
			incrementStatItems(stats.PlayerTwoData, stats.PlayerOneData, history, i, id, false)
		}
		incrementStatItems(stats.NotableData, nil, history, i, id, false)
	}

	incrementStatItems(stats.PlayerOneData, stats.PlayerTwoData, history, -1, id, !history.SecondWentFirst)
	incrementStatItems(stats.PlayerTwoData, stats.PlayerOneData, history, -1, id, history.SecondWentFirst)
	incrementStatItems(stats.NotableData, nil, history, -1, id, false)

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

func (stats *Stats) FinalizeStats() {
	finalize(stats.PlayerOneData)
	finalize(stats.PlayerTwoData)
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

func listItemToString(listStat []*ListItem) string {
	s := ""
	for _, wordItem := range listStat {
		s += fmt.Sprintf("      %s, %d, %d, %s\n", wordItem.Word, wordItem.Score, wordItem.Probability, wordItem.GameId)
	}
	return s
}
