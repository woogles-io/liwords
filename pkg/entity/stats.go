package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// A ListDatum is the individual datum that is stored in a list. It is
// a sort of "union" of various struct types. Depending on the type of stat,
// only some of thees fields will be filled in.
type ListDatum struct {
	// Used for words
	Word        string `json:"w,omitempty"`
	Probability int    `json:"p,omitempty"`
	// Used for words or games:
	Score int `json:"s,omitempty"`

	// Used for comments:
	Comment string `json:"c,omitempty"`

	// Used for mistakes:
	MistakeType int `json:"t,omitempty"`
	MistakeSize int `json:"z,omitempty"`

	// Used for ratings:
	Rating  int    `json:"r,omitempty"`
	Variant string `json:"v,omitempty"`
}

type ListItem struct {
	GameId   string
	PlayerId string
	Time     int64
	Item     ListDatum
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
	Name          string         `json:"-"`
	Minimum       int            `json:"-"`
	Maximum       int            `json:"-"`
	Total         int            `json:"t"`
	IncrementType IncrementType  `json:"-"`
	List          []*ListItem    `json:"l"`
	Subitems      map[string]int `json:"s"`
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
	RATINGS_STAT                           string = "Ratings"
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
	LOW_WIN_STAT                           string = "Low Win"
	HIGH_LOSS_STAT                         string = "High Loss"
	UPSET_WIN_STAT                         string = "Upset Win"
)

var StatName_value = map[string]int{
	ALL_TRIPLE_LETTERS_COVERED_STAT:        0,
	ALL_TRIPLE_WORDS_COVERED_STAT:          1,
	BINGOS_STAT:                            2,
	CHALLENGED_PHONIES_STAT:                3,
	CHALLENGES_LOST_STAT:                   4,
	CHALLENGES_WON_STAT:                    5,
	COMMENTS_STAT:                          6,
	DRAWS_STAT:                             7,
	EXCHANGES_STAT:                         8,
	FIRSTS_STAT:                            9,
	GAMES_STAT:                             10,
	HIGH_GAME_STAT:                         11,
	HIGH_TURN_STAT:                         12,
	LOSSES_STAT:                            13,
	LOW_GAME_STAT:                          14,
	NO_BINGOS_STAT:                         15,
	MANY_DOUBLE_LETTERS_COVERED_STAT:       16,
	MANY_DOUBLE_WORDS_COVERED_STAT:         17,
	MISTAKES_STAT:                          18,
	SCORE_STAT:                             19,
	RATINGS_STAT:                           20,
	TILES_PLAYED_STAT:                      21,
	TIME_STAT:                              22,
	TRIPLE_TRIPLES_STAT:                    23,
	TURNS_STAT:                             24,
	TURNS_WITH_BLANK_STAT:                  25,
	UNCHALLENGED_PHONIES_STAT:              26,
	VALID_PLAYS_THAT_WERE_CHALLENGED_STAT:  27,
	VERTICAL_OPENINGS_STAT:                 28,
	WINS_STAT:                              29,
	NO_BLANKS_PLAYED_STAT:                  30,
	HIGH_SCORING_STAT:                      31,
	COMBINED_HIGH_SCORING_STAT:             32,
	COMBINED_LOW_SCORING_STAT:              33,
	ONE_PLAYER_PLAYS_EVERY_POWER_TILE_STAT: 34,
	ONE_PLAYER_PLAYS_EVERY_E_STAT:          35,
	MANY_CHALLENGES_STAT:                   36,
	FOUR_OR_MORE_CONSECUTIVE_BINGOS_STAT:   37,
	LOW_WIN_STAT:                           38,
	HIGH_LOSS_STAT:                         39,
	UPSET_WIN_STAT:                         40,
}

func (ld *ListDatum) Value() (driver.Value, error) {
	return json.Marshal(ld)
}

// Remove these transformation functions once we get rid of Gorm everywhere.
func (ld *ListDatum) Scan(value interface{}) error {
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	case nil:
		return nil
	default:
		return fmt.Errorf("unexpected type %T for listdatum", value)
	}

	return json.Unmarshal(b, &ld)
}
