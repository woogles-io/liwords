package entity

import (
	"errors"
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondoconfig "github.com/domino14/macondo/config"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
)

type ListItem struct {
	Word        string `json:"w"`
	Probability int    `json:"p"`
	Score       int    `json:"s"`
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

type IncrementInfo struct {
	cfg                 *macondoconfig.Config
	req                 *realtime.GameRequest
	statItem            *StatItem
	otherPlayerStatItem *StatItem
	history             *pb.GameHistory
	eventIndex          int
	id                  string
	isPlayerOne         bool
}

type StatItem struct {
	Minimum       int                        `json:"-"`
	Maximum       int                        `json:"-"`
	Total         int                        `json:"t"`
	DataType      StatItemType               `json:"-"`
	IncrementType IncrementType              `json:"-"`
	List          []*ListItem                `json:"l"`
	Subitems      map[string]int             `json:"s"`
	AddFunction   func(*IncrementInfo) error `json:"-"`
}

type Stats struct {
	PlayerOneId   string               `json:"i1"`
	PlayerTwoId   string               `json:"i2"`
	PlayerOneData map[string]*StatItem `json:"d1"`
	PlayerTwoData map[string]*StatItem `json:"d2"`
	NotableData   map[string]*StatItem `json:"n"`
}

type ProfileStats struct {
	Data map[VariantKey]*Stats
}

const (
	ALL_TRIPLE_LETTERS_COVERED_STAT        string = "All Triple Letter Squares Covered"
	ALL_TRIPLE_WORDS_COVERED_STAT          string = "All Triple Word Squares Covered"
	BINGOS_STAT                            string = "Bingos"
	CHALLENGED_PHONIES_STAT                string = "Challenged Phonies"
	CHALLENGES_LOST_STAT                   string = "Challenges Lost"
	CHALLENGES_WON_STAT                    string = "Challenges Won"
	COMMENTS_STAT                          string = "Comments"
	DRAWS_STAT                             string = "Draws"
	EXCHANGES_STAT                         string = "Exchanges"
	FIRSTS_STAT                            string = "Firsts"
	GAMES_STAT                             string = "Games"
	HIGH_GAME_STAT                         string = "High Game"
	HIGH_TURN_STAT                         string = "High Turn"
	LOSSES_STAT                            string = "Losses"
	LOW_GAME_STAT                          string = "Low Game"
	NO_BINGOS_STAT                         string = "Games with no Bingos"
	MANY_DOUBLE_LETTERS_COVERED_STAT       string = "Many Double Letter Squares Covered"
	MANY_DOUBLE_WORDS_COVERED_STAT         string = "Many Double Word Squares Covered"
	MISTAKES_STAT                          string = "Mistakes"
	SCORE_STAT                             string = "Score"
	TILES_PLAYED_STAT                      string = "Tiles Played"
	TIME_STAT                              string = "Time Taken"
	TRIPLE_TRIPLES_STAT                    string = "Triple Triples"
	TURNS_STAT                             string = "Turns"
	TURNS_WITH_BLANK_STAT                  string = "Turns With Blank"
	UNCHALLENGED_PHONIES_STAT              string = "Unchallenged Phonies"
	VALID_PLAYS_THAT_WERE_CHALLENGED_STAT  string = "Valid Plays That Were Challenged"
	VERTICAL_OPENINGS_STAT                 string = "Vertical Openings"
	WINS_STAT                              string = "Wins"
	NO_BLANKS_PLAYED_STAT                  string = "No Blanks Played"
	HIGH_SCORING_STAT                      string = "High Scoring"
	COMBINED_HIGH_SCORING_STAT             string = "Combined High Scoring"
	COMBINED_LOW_SCORING_STAT              string = "Combined Low Scoring"
	ONE_PLAYER_PLAYS_EVERY_POWER_TILE_STAT string = "One Player Plays Every Power Tile"
	ONE_PLAYER_PLAYS_EVERY_E_STAT          string = "One Player Plays Every E"
	MANY_CHALLENGES_STAT                   string = "Many Challenges"
	FOUR_OR_MORE_CONSECUTIVE_BINGOS_STAT   string = "Four or More Consecutive Bingos"
)

// InstantiateNewStats instantiates a new stats object. playerOneId MUST
// have gone first in the game.
func InstantiateNewStats(playerOneId string, playerTwoId string) *Stats {
	log.Debug().Str("p1id", playerOneId).Str("p2id", playerTwoId).Msg("instantiating new stats.")
	return &Stats{
		PlayerOneId:   playerOneId,
		PlayerTwoId:   playerTwoId,
		PlayerOneData: instantiatePlayerData(),
		PlayerTwoData: instantiatePlayerData(),
		NotableData:   instantiateNotableData()}
}

func (stats *Stats) AddGame(history *pb.GameHistory, req *realtime.GameRequest, cfg *macondoconfig.Config, id string) error {
	events := history.GetEvents()
	for i := 0; i < len(events); i++ {
		event := events[i]
		if history.Players[0].Nickname == event.Nickname {
			err := incrementStatItems(cfg, req, stats.PlayerOneData, stats.PlayerTwoData, history, i, id, false)
			if err != nil {
				return err
			}
		} else {
			err := incrementStatItems(cfg, req, stats.PlayerTwoData, stats.PlayerOneData, history, i, id, false)
			if err != nil {
				return err
			}
		}
		err := incrementStatItems(cfg, req, stats.NotableData, nil, history, i, id, false)
		if err != nil {
			return err
		}
	}

	err := incrementStatItems(cfg, req, stats.PlayerOneData, stats.PlayerTwoData, history, -1, id, true)
	if err != nil {
		return err
	}

	err = incrementStatItems(cfg, req, stats.PlayerTwoData, stats.PlayerOneData, history, -1, id, false)
	if err != nil {
		return err
	}

	err = incrementStatItems(cfg, req, stats.NotableData, nil, history, -1, id, false)
	if err != nil {
		return err
	}

	confirmNotableItems(stats.NotableData, id)
	return nil
}

func (stats *Stats) AddStats(otherStats *Stats) error {

	if stats.PlayerOneId != otherStats.PlayerOneId && stats.PlayerOneId != otherStats.PlayerTwoId {
		return errors.New("Stats do not share an identical PlayerOneId")
	}

	otherPlayerOneData := otherStats.PlayerOneData
	otherPlayerTwoData := otherStats.PlayerTwoData

	if stats.PlayerOneId == otherStats.PlayerTwoId {
		otherPlayerOneData = otherStats.PlayerTwoData
		otherPlayerTwoData = otherStats.PlayerOneData
	}

	err := combineStatItemMaps(stats.PlayerOneData, otherPlayerOneData)

	if err != nil {
		return err
	}

	err = combineStatItemMaps(stats.PlayerTwoData, otherPlayerTwoData)

	if err != nil {
		return err
	}

	err = combineStatItemMaps(stats.NotableData, otherStats.NotableData)

	if err != nil {
		return err
	}

	return nil
}

func (stats *Stats) Finalize() {
	finalize(stats.PlayerOneData)
	finalize(stats.PlayerTwoData)
}
