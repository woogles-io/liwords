package entity

import (
	"errors"
	"fmt"
	"github.com/domino14/macondo/alphabet"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
	"strings"
	"unicode"
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
	MediumMistakeMagnitude: 2,
	SmallMistakeMagnitude:  3,

	SaddestMistakeMagnitude:     1,
	SadderMistakeMagnitude:      2,
	SadMistakeMagnitude:         3,
	UnspecifiedMistakeMagnitude: 0,
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
	AddFunction        func(*StatItem, *pb.GameHistory, int, string, bool)
}

type Stats struct {
	PlayerOneId   int
	PlayerTwoId   int
	PlayerOneData []*StatItem
	PlayerTwoData []*StatItem
	NotableData   []*StatItem
}

func InstantiateNewStats() *Stats {
	return &Stats{
		PlayerOneId:   1,
		PlayerTwoId:   2,
		PlayerOneData: instantiatePlayerData(),
		PlayerTwoData: instantiatePlayerData(),
		NotableData:   instantiateNotableData()}
}

func (stats *Stats) AddGameToStats(history *pb.GameHistory, id string) error {
	events := history.GetEvents()
	for i := 0; i < len(events); i++ {
		event := events[i]
		fmt.Println(event)
		if history.Players[0].Nickname == event.Nickname ||
			(history.Players[1].Nickname == event.Nickname && history.SecondWentFirst) {
			incrementStatItems(stats.PlayerOneData, history, i, id, false)
		} else {
			incrementStatItems(stats.PlayerTwoData, history, i, id, false)
		}
		incrementStatItems(stats.NotableData, history, i, id, false)
	}

	incrementStatItems(stats.PlayerOneData, history, -1, id, !history.SecondWentFirst)
	incrementStatItems(stats.PlayerTwoData, history, -1, id, history.SecondWentFirst)
	incrementStatItems(stats.NotableData, history, -1, id, false)

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

func makeAlphabetSubitems() map[string]int {
	alphabetSubitems := make(map[string]int)
	for i := 0; i < 26; i++ {
		alphabetSubitems[string('A'+rune(i))] = 0
	}
	alphabetSubitems[string(alphabet.BlankToken)] = 0
	return alphabetSubitems
}

func instantiatePlayerData() []*StatItem {
	gamesStat := &StatItem{Name: "Games",
		Description:        "The number of games played",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      GameType,
		Denominator:        nil,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addGames}
	scoreStat := &StatItem{Name: "Score",
		Description:        "The average score of the player",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      GameType,
		Denominator:        gamesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addScore}

	firstsStat := &StatItem{Name: "Firsts",
		Description:        "The number of times the player went first",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      GameType,
		Denominator:        gamesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addFirsts}
	verticalOpeningsStat := &StatItem{Name: "Vertical Openings",
		Description:        "The number of times the player opened vertically",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      EventType,
		Denominator:        gamesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addVerticalOpenings}
	turnsStat := &StatItem{Name: "Turns",
		Description:        "The number of turns the player had",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      EventType,
		Denominator:        gamesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addTurns}

	exchangesStat := &StatItem{Name: "Exchanges",
		Description:        "The number of times the player exchanged",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      EventType,
		Denominator:        gamesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addExchanges}

	phoniesStat := &StatItem{Name: "Phonies",
		Description:        "The number of phonies plays made",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      EventType,
		Denominator:        gamesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addPhonies}

	challengedPhoniesStat := &StatItem{Name: "Challenged Phonies",
		Description:        "The number of phonies plays made that were challenged off the board",
		Total:              0,
		DataType:           ListType,
		IncrementType:      EventType,
		Denominator:        phoniesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addChallengedPhonies}

	unchallengedPhoniesStat := &StatItem{Name: "Unchallenged Phonies",
		Description:        "The number of phonies plays made that were not challenged off the board",
		Total:              0,
		DataType:           ListType,
		IncrementType:      EventType,
		Denominator:        phoniesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addUnchallengedPhonies}

	challengesStat := &StatItem{Name: "Challenges",
		Description:        "The number of challenges made by the player",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      EventType,
		Denominator:        gamesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addChallenges}

	challengesWonStat := &StatItem{Name: "Challenges Won",
		Description:        "The number of challenges won by the player",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      EventType,
		Denominator:        challengesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addChallengesWon}

	challengesLostStat := &StatItem{Name: "Unchallenged Phonies",
		Description:        "The number challenges lost by the player",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      EventType,
		Denominator:        challengesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addChallengesLost}

	playsThatWereChallengedStat := &StatItem{Name: "Plays That Were Challenged",
		Description:        "The number of plays that were challenged by the opponent",
		Total:              0,
		DataType:           ListType,
		IncrementType:      EventType,
		Denominator:        turnsStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addPlaysThatWereChallenged}

	winsStat := &StatItem{Name: "Wins",
		Description:        "The number of wins",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      GameType,
		Denominator:        gamesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addWins}
	lossesStat := &StatItem{Name: "Losses",
		Description:        "The number of losses",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      GameType,
		Denominator:        gamesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addLosses}
	drawsStat := &StatItem{Name: "Draws",
		Description:        "The number of draws",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      GameType,
		Denominator:        gamesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addDraws}
	bingosStat := &StatItem{Name: "Bingos",
		Description:        "The list of bingos played",
		Total:              0,
		DataType:           ListType,
		IncrementType:      EventType,
		Denominator:        gamesStat,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addBingos}

	noBingosStat := &StatItem{Name: "Bingoless Games",
		Description:        "The list of bingos played",
		Total:              0,
		DataType:           ListType,
		IncrementType:      GameType,
		Denominator:        gamesStat,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addNoBingos}

	tripleTriplesStat := &StatItem{Name: "Triple Triples",
		Description:        "Number of triple triples played",
		Total:              0,
		DataType:           ListType,
		IncrementType:      EventType,
		Denominator:        gamesStat,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addTripleTriples}

	bingoNinesOrAboveStat := &StatItem{Name: "Bingo Nines or Above",
		Description:        "The list of bingos that were nines tiles or above",
		Total:              0,
		DataType:           ListType,
		IncrementType:      EventType,
		Denominator:        gamesStat,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addBingoNinesOrAbove}

	highGameStat := &StatItem{Name: "Highest Scoring Game",
		Description:        "The game with the highest score",
		Total:              0,
		DataType:           MaximumType,
		IncrementType:      GameType,
		Denominator:        nil,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        setHighGame}

	lowGameStat := &StatItem{Name: "Lowest Scoring Game",
		Description:        "The game with the lowest score",
		Total:              0,
		DataType:           MinimumType,
		IncrementType:      GameType,
		Denominator:        nil,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        setLowGame}

	highTurnStat := &StatItem{Name: "Highest Scoring Turn",
		Description:        "The turn with the highest score",
		Total:              0,
		DataType:           MaximumType,
		IncrementType:      EventType,
		Denominator:        nil,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        setHighTurn}

	tilesPlayedStat := &StatItem{Name: "Tiles Played",
		Description:        "The number of tiles played",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      EventType,
		Denominator:        gamesStat,
		IsProfileStat:      false,
		List:               []*ListItem{},
		Subitems:           makeAlphabetSubitems(),
		HasMeaningfulTotal: true,
		AddFunction:        addTilesPlayed}

	turnsWithBlankStat := &StatItem{Name: "Turns With at Least One Blank",
		Description:        "The number of turns where the player had at least one blank on their rack",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      EventType,
		Denominator:        turnsStat,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addTurnsWithBlank}

	commentsStat := &StatItem{Name: "Comments",
		Description:        "The number of annotated comments",
		Total:              0,
		DataType:           SingleType,
		IncrementType:      EventType,
		Denominator:        gamesStat,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addComments}

	confidenceIntervalsStat := &StatItem{Name: "Confidence Intervals",
		Description:   "The confidence intervals for each tile drawn",
		Total:         0,
		DataType:      SingleType,
		IncrementType: FinalType,
		// Not actually a denominator, just a needed ref
		// because this stat is special
		Denominator:        tilesPlayedStat,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addConfidenceIntervals}

	return []*StatItem{gamesStat,
		scoreStat,
		firstsStat,
		winsStat,
		drawsStat,
		lossesStat,
		turnsStat,
		verticalOpeningsStat,
		bingosStat,
		noBingosStat,
		exchangesStat,
		tripleTriplesStat,
		bingoNinesOrAboveStat,
		phoniesStat,
		challengedPhoniesStat,
		unchallengedPhoniesStat,
		challengesStat,
		challengesWonStat,
		challengesLostStat,
		playsThatWereChallengedStat,
		highGameStat,
		lowGameStat,
		highTurnStat,
		tilesPlayedStat,
		turnsWithBlankStat,
		commentsStat,
		confidenceIntervalsStat,
	}
	/*
		Missing stats:
			Full rack per turn
			Bonus square coverage
			Comments word length
			Dynamic Mistakes
			Confidence Intervals
		Stats that are missing but can be derive from the current stats data:
			Bingo probabilities
			Power tiles played
			power tiles stuck with
	*/
}

func instantiateNotableData() []*StatItem {
	return []*StatItem{&StatItem{Name: "No Blanks Played",
		Description:        "Games in which no blanks are played",
		Minimum:            0,
		Maximum:            0,
		Total:              0,
		DataType:           ListType,
		IncrementType:      EventType,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: false,
		AddFunction:        addBlanksPlayed},

		&StatItem{Name: "High Scoring",
			Description:        "Games in which one player scores at least 700 points",
			Minimum:            700,
			Maximum:            MaxNotableInt,
			Total:              0,
			DataType:           ListType,
			IncrementType:      GameType,
			IsProfileStat:      false,
			List:               []*ListItem{},
			HasMeaningfulTotal: false,
			AddFunction:        addHighScoring},

		&StatItem{Name: "Combined High Scoring",
			Description:        "Games in which the combined score is at least 1100 points",
			Minimum:            1100,
			Maximum:            MaxNotableInt,
			Total:              0,
			DataType:           ListType,
			IncrementType:      GameType,
			IsProfileStat:      false,
			List:               []*ListItem{},
			HasMeaningfulTotal: false,
			AddFunction:        addCombinedScoring},

		&StatItem{Name: "Combined Low Scoring",
			Description:        "Games in which the combined score no greater than 200 points",
			Minimum:            0,
			Maximum:            200,
			Total:              0,
			DataType:           ListType,
			IncrementType:      GameType,
			IsProfileStat:      false,
			List:               []*ListItem{},
			HasMeaningfulTotal: false,
			AddFunction:        addCombinedScoring},

		&StatItem{Name: "One Player Plays Every Power Tile",
			Description:        "Games in which one player plays the Z, X, Q, J, every blank, and every S",
			Minimum:            10,
			Maximum:            10,
			Total:              0,
			DataType:           ListType,
			IncrementType:      EventType,
			IsProfileStat:      false,
			List:               []*ListItem{},
			HasMeaningfulTotal: false,
			AddFunction:        addEveryPowerTile},

		&StatItem{Name: "One Player Plays Every E",
			Description:        "Games in which one player plays every E",
			Minimum:            12,
			Maximum:            12,
			Total:              0,
			DataType:           ListType,
			IncrementType:      EventType,
			IsProfileStat:      false,
			List:               []*ListItem{},
			HasMeaningfulTotal: false,
			AddFunction:        addEveryE},

		&StatItem{Name: "Many Challenges",
			Description:        "Games in which there are at least 5 challenges made",
			Minimum:            5,
			Maximum:            MaxNotableInt,
			Total:              0,
			DataType:           ListType,
			IncrementType:      EventType,
			IsProfileStat:      false,
			List:               []*ListItem{},
			HasMeaningfulTotal: false,
			AddFunction:        addManyChallenges},

		&StatItem{Name: "Four or More Consecutive Bingos",
			Description:        "Games in which there are at least four consecutive bingos",
			Minimum:            4,
			Maximum:            MaxNotableInt,
			Total:              0,
			DataType:           ListType,
			IncrementType:      EventType,
			IsProfileStat:      false,
			List:               []*ListItem{},
			Subitems:           map[string]int{"player_one_streak": 0, "player_two_streak": 0},
			HasMeaningfulTotal: false,
			AddFunction:        addConsecutiveBingos},
	}
}

func incrementStatItems(statItems []*StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	for _, statItem := range statItems {
		if (statItem.IncrementType == EventType && i >= 0) ||
			(statItem.IncrementType == GameType && i < 0) {
			statItem.AddFunction(statItem, history, i, id, isPlayerOne)
		}
	}
}

func incrementStatItem(statItem *StatItem, event *pb.GameEvent, id string) {
	if statItem.DataType == SingleType {
		statItem.Total++
	} else if statItem.DataType == ListType {
		statItem.List = append(statItem.List, &ListItem{Word: "YEET", Probability: 1, Score: 1, GameId: id})
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
		// For one player plays every _ stats
		// Player one adds to the total, player two subtracts
		// So we need the absolute value to account for both possibilities
		if item.Total < 0 {
			item.Total = item.Total * (-1)
		}
		if item.Total >= item.Minimum && item.Total <= item.Maximum {
			item.List = append(item.List, &ListItem{Word: "", Probability: 0, Score: 0, GameId: id})
		}
		item.Total = 0
	}
}

func combineItems(statItem *StatItem, otherStatItem *StatItem) {
	if statItem.DataType == SingleType {
		statItem.Total += otherStatItem.Total
	} else if statItem.DataType == ListType {
		statItem.List = append(statItem.List, otherStatItem.List...)
	} else if (statItem.DataType == MaximumType && otherStatItem.Total > statItem.Total) ||
		(statItem.DataType == MinimumType && otherStatItem.Total < statItem.Total) {
		statItem.Total = otherStatItem.Total
		statItem.List = otherStatItem.List
	}

	if statItem.Subitems != nil {
		for key, _ := range statItem.Subitems {
			statItem.Subitems[key] += otherStatItem.Subitems[key]
		}
	}
}

func finalize(statItems []*StatItem) {
	for _, statItem := range statItems {
		if (statItem.IncrementType == FinalType){
			statItem.AddFunction(nil, nil, -1, "", false)
		}
	}
}
func addBingoNinesOrAbove(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if isTripleTriple(event) {
		incrementStatItem(statItem, event, id)
	}
}

func addBingos(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.IsBingo {
		incrementStatItem(statItem, event, id)
	}
}

func addBlanksPlayed(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	tiles := event.PlayedTiles
	for _, char := range tiles {
		if unicode.IsLower(char) {
			statItem.Total++
		}
	}
}

func addCombinedScoring(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	playerOneScore := 400 // Get actual history from score when Cesar fixes history
	playerTwoScore := 500 // Get actual history from score when Cesar fixes history
	statItem.Total = playerOneScore + playerTwoScore
}

func addConfidenceIntervals(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	// We'll need to decide if we even want this
	// We pull a sneaky here and overload some struct fields
	/*	tilesPlayedStat := statItem.Denominator
		gamesStat := tilesPlayedStat.Denominator
		totalGames := gamesStat.Total
		for tile, timesPlayed := range tilesPlayedStat.Subitems {
			bagSizes := 100 // Get real bag frequency somehow
			tileFrequency := 8 // Get real frequency somehow
	          my $tile_frequencies = Constants::TILE_FREQUENCIES;
	          my $f           = $tile_frequencies->{$subtitle};
	          my $P           = $average / 100;
	          my $n           = $f * $numgames;
	          my ($lower, $upper) = Utils::get_confidence_interval($P, $n);
	          my $prob        = sprintf "%.4f", $subaverage / $f;
		}*/
}

func addHighScoring(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	playerOneScore := 400 // Get actual history from score when Cesar fixes history
	playerTwoScore := 500 // Get actual history from score when Cesar fixes history
	if playerOneScore > playerTwoScore {
		statItem.Total = playerOneScore
	} else {
		statItem.Total = playerTwoScore
	}
}

func addExchanges(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Type == pb.GameEvent_EXCHANGE {
		incrementStatItem(statItem, event, id)
	}
}

func addPhonies(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	isPhony := false // Add isPhony to GameEvent probably
	if isPhony {
		incrementStatItem(statItem, event, id)
	}
}

func addChallengedPhonies(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED {
		incrementStatItem(statItem, event, id)
		// Need to increment opp's challengesWon stat
	}
}

func addUnchallengedPhonies(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	isPhony := false // Add isPhony to GameEvent probably
	if isPhony {
		incrementStatItem(statItem, event, id)
	}
}

func addPlaysThatWereChallenged(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	var succEvent *pb.GameEvent
	if i+1 < len(events) {
		succEvent = events[i+1]
	}
	if succEvent != nil &&
		(succEvent.Type == pb.GameEvent_CHALLENGE_BONUS ||
			succEvent.Type == pb.GameEvent_PHONY_TILES_RETURNED) {
		incrementStatItem(statItem, event, id)
		// Need to increment opp's challengesLost stat
	}
}

func addChallenges(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	var succEvent *pb.GameEvent
	if i+1 < len(events) {
		succEvent = events[i+1]
	}
	var succSuccEvent *pb.GameEvent
	if i+2 < len(events) {
		succSuccEvent = events[i+2]
	}
	if succSuccEvent != nil &&
		(succSuccEvent.Type == pb.GameEvent_CHALLENGE_BONUS || // Opp's bonus
			succSuccEvent.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS || // Player's turn loss
			succSuccEvent.Type == pb.GameEvent_PHONY_TILES_RETURNED) { // Opp's phony tiles returned
		incrementStatItem(statItem, succEvent, id)
	}
}

func addChallengesWon(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	var succEvent *pb.GameEvent
	if i+1 < len(events) {
		succEvent = events[i+1]
	}
	var succSuccEvent *pb.GameEvent
	if i+2 < len(events) {
		succSuccEvent = events[i+2]
	}
	if succSuccEvent != nil &&
		(succSuccEvent.Type == pb.GameEvent_PHONY_TILES_RETURNED) { // Opp's phony tiles returned
		incrementStatItem(statItem, succEvent, id)
	}
}

func addChallengesLost(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	var succEvent *pb.GameEvent
	if i+1 < len(events) {
		succEvent = events[i+1]
	}
	var succSuccEvent *pb.GameEvent
	if i+2 < len(events) {
		succSuccEvent = events[i+2]
	}
	if succSuccEvent != nil &&
		(succSuccEvent.Type == pb.GameEvent_CHALLENGE_BONUS || // Opp's bonus
			succSuccEvent.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS) { // Player's turn loss
		incrementStatItem(statItem, succEvent, id)
	}
}

func addComments(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Note != "" {
		incrementStatItem(statItem, event, id)
	}
}

func addConsecutiveBingos(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	player := "player_one_streak"
	if !isPlayerOne {
		player = "player_two_streak"
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
		event.Type == pb.GameEvent_PASS ||
		event.Type == pb.GameEvent_EXCHANGE {
		if event.IsBingo {
			statItem.Subitems[player]++
			if statItem.Subitems[player] > statItem.Total {
				statItem.Total = statItem.Subitems[player]
			}
		} else {
			statItem.Subitems[player] = 0
		}
	}
}

func addDraws(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	playerOneScore := 400 // Get actual history from score when Cesar fixes history
	playerTwoScore := 500 // Get actual history from score when Cesar fixes history
	if playerOneScore == playerTwoScore && isPlayerOne {
		incrementStatItem(statItem, nil, id)
	}
}

func addEveryE(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	var succEvent *pb.GameEvent
	if i+1 < len(events) {
		succEvent = events[i+1]
	}
	multiplier := 1
	if !isPlayerOne {
		multiplier = -1
	}
	if succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED {
		for _, char := range event.PlayedTiles {
			if char == 'E' {
				statItem.Total += 1 * multiplier
			}
		}
	}
}

func addEveryPowerTile(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	var succEvent *pb.GameEvent
	if i+1 < len(events) {
		succEvent = events[i+1]
	}
	multiplier := 1
	if !isPlayerOne {
		multiplier = -1
	}
	if succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED {
		for _, char := range event.PlayedTiles {
			if char == 'J' || char == 'Q' || char == 'X' || char == 'Z' || char == alphabet.BlankToken || char == 'S' {
				statItem.Total += 1 * multiplier
			}
		}
	}
}

func addFirsts(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	if isPlayerOne {
		incrementStatItem(statItem, nil, id)
	}
}

func addGames(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	incrementStatItem(statItem, nil, id)
}

func addLosses(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	playerOneScore := 400 // Get actual history from score when Cesar fixes history
	playerTwoScore := 500 // Get actual history from score when Cesar fixes history
	if (playerOneScore < playerTwoScore && isPlayerOne) ||
		(playerOneScore > playerTwoScore && !isPlayerOne) {
		incrementStatItem(statItem, nil, id)
	}
}

func addManyChallenges(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED ||
		event.Type == pb.GameEvent_CHALLENGE_BONUS ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		statItem.Total++
	}
}

func addMistakes(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	mistakeTypes := []string{KnowledgeMistakeType, FindingMistakeType, VisionMistakeType, TacticsMistakeType, StrategyMistakeType, TimeMistakeType, EndgameMistakeType}
	mistakeMagnitudes := []string{LargeMistakeMagnitude, MediumMistakeMagnitude, SmallMistakeMagnitude, SaddestMistakeMagnitude, SadderMistakeMagnitude, SadMistakeMagnitude}
	if event.Note != "" {
		for _, mistakeT := range mistakeTypes {
			totalOccurences := 0
			for _, mistakeM := range mistakeMagnitudes {
				occurences := strings.Count(strings.ToLower(event.Note), "#"+mistakeT+mistakeM)
				statItem.Total += occurences
				statItem.Subitems[mistakeT] += occurences
				statItem.Subitems[mistakeM] += occurences
				totalOccurences += occurences
				for i := 0; i < occurences; i++ {
					statItem.List = append(statItem.List, &ListItem{Word: event.Note, Probability: MistakeTypeMapping[mistakeT], Score: MistakeMagnitudeMapping[mistakeM], GameId: id})
				}
			}
			unspecifiedOccurences := strings.Count(strings.ToLower(event.Note), "#"+mistakeT)
			statItem.Total += unspecifiedOccurences - totalOccurences
			statItem.Subitems[mistakeT] += unspecifiedOccurences - totalOccurences
			statItem.Subitems[UnspecifiedMistakeMagnitude] += unspecifiedOccurences - totalOccurences
			for i := 0; i < unspecifiedOccurences-totalOccurences; i++ {
				statItem.List = append(statItem.List, &ListItem{Word: event.Note, Probability: MistakeTypeMapping[mistakeT], Score: -1, GameId: id})
			}
		}
	}
}

func addNoBingos(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	atLeastOneBingo := false
	// SAD! (have to loop through events again, should not do this, is the big not good)
	for i := 0; i < len(events); i++ {
		event := events[i]
		fmt.Println(event)
		if (history.Players[0].Nickname == event.Nickname && isPlayerOne) ||
			(history.Players[1].Nickname == event.Nickname && !isPlayerOne) {
			atLeastOneBingo = true
			break
		}
	}
	if !atLeastOneBingo {
		incrementStatItem(statItem, nil, id)
	}
}

func addScore(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	playerOneScore := 400 // Get these from history later
	playerTwoScore := 500
	if isPlayerOne {
		statItem.Total += playerOneScore
	} else {
		statItem.Total += playerTwoScore
	}
}

func addTilesPlayed(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	var succEvent *pb.GameEvent
	if i+1 < len(events) {
		succEvent = events[i+1]
	}
	if succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED {
		for _, char := range event.PlayedTiles {
			if char != alphabet.ASCIIPlayedThrough {
				statItem.Total++
				if unicode.IsLower(char) {
					statItem.Subitems[string(alphabet.BlankToken)]++
				} else {
					statItem.Subitems[string(char)]++
				}
			}
		}
	}
}

func addTilesStuckWith(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[len(events)-1]
	if event.Type == pb.GameEvent_END_RACK_PTS &&
		((isPlayerOne && event.Nickname == history.Players[0].Nickname) ||
			(!isPlayerOne && event.Nickname == history.Players[1].Nickname)) {
		tilesStuckWith := event.Rack
		for _, char := range tilesStuckWith {
			statItem.Total++
			statItem.Subitems[string(char)]++
		}
	}
}

func addTripleTriples(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if isTripleTriple(event) {
		incrementStatItem(statItem, event, id)
	}
}

func addTurns(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
		event.Type == pb.GameEvent_PASS ||
		event.Type == pb.GameEvent_EXCHANGE ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		incrementStatItem(statItem, event, id)
	}
}

func addTurnsWithBlank(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
		event.Type == pb.GameEvent_PASS ||
		event.Type == pb.GameEvent_EXCHANGE ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		for _, char := range event.Rack {
			if char == alphabet.BlankToken {
				incrementStatItem(statItem, event, id)
				break
			}
		}
	}
}

func addVerticalOpenings(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[0]
	if isPlayerOne && events[0].Direction == pb.GameEvent_VERTICAL {
		incrementStatItem(statItem, event, id)
	}
}

func addWins(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	playerOneScore := 400 // Get actual history from score when Cesar fixes history
	playerTwoScore := 500 // Get actual history from score when Cesar fixes history
	if (playerOneScore > playerTwoScore && isPlayerOne) ||
		(playerOneScore < playerTwoScore && !isPlayerOne) {
		incrementStatItem(statItem, nil, id)
	}
}

func setHighGame(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	playerScore := 400 // Replace with actual scores later
	if !isPlayerOne {
		playerScore = 500 //Replace with actual scores later
	}
	if playerScore > statItem.Total {
		statItem.Total = playerScore
		statItem.List = []*ListItem{&ListItem{Word: "", Probability: 0, Score: 0, GameId: id}}
	}
}

func setLowGame(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	playerScore := 400 // Replace with actual scores later
	if !isPlayerOne {
		playerScore = 500 //Replace with actual scores later
	}
	if playerScore < statItem.Total {
		statItem.Total = playerScore
		statItem.List = []*ListItem{&ListItem{Word: "", Probability: 0, Score: 0, GameId: id}}
	}
}

func setHighTurn(statItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	score := int(event.Score)
	if score > statItem.Total {
		statItem.Total = score
		statItem.List = []*ListItem{&ListItem{Word: "", Probability: 0, Score: 0, GameId: id}}
	}
}

func isTripleTriple(event *pb.GameEvent) bool {
	// Need to implement
	return false
}

func isBingoNineOrAbove(event *pb.GameEvent) bool {
	return event.IsBingo && len(event.PlayedTiles) >= 9
}
