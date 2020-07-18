package entity

import (
	"errors"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
)

type ListItem struct {
	Word        string `json:"w"`
	Probability int `json:"p"`
	Score       int `json:"s"`
	GameId      string `json:"g"`
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
	Name               string `json:"n"`
	Description        string `json:"d"`
	Minimum            int `json:"-"`
	Maximum            int `json:"-"`
	Total              int `json:"t"`
	DataType           StatItemType `json:"-"`
	IncrementType      IncrementType `json:"-"`
	DenominatorList    []*StatItem `json:"-"`
	Averages           []float64 `json:"a"`
	IsProfileStat      bool `json:"-"`
	List               []*ListItem `json:"l"`
	Subitems           map[string]int `json:"s"`
	HasMeaningfulTotal bool `json:"h"`
	AddFunction        func(*StatItem, *StatItem, *pb.GameHistory, int, string, bool) `json:"-"`
}

type Stats struct {
	PlayerOneId   int `json:"i1"`
	PlayerTwoId   int `json:"i2"`
	PlayerOneData []*StatItem `json:"d1"`
	PlayerTwoData []*StatItem `json:"d2"`
	NotableData   []*StatItem `json:"n"`
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
		//fmt.Println(event)
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

func (stats *Stats) Finalize() {
	finalize(stats.PlayerOneData)
	finalize(stats.PlayerTwoData)
}