package entity

import (
	"gorm.io/datatypes"
	"sync"
	"time"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

type DivisionManager interface {
	GetPlayerRoundInfo(string, int) (*PlayerRoundInfo, error)
	SubmitResult(int, string, string, int, int, realtime.TournamentGameResult,
		realtime.TournamentGameResult, realtime.GameEndReason, bool, int) error
	GetStandings(int) ([]*Standing, error)
	SetPairing(string, string, int) error
	StartRound(int) error
	IsRoundComplete(int) (bool, error)
	IsFinished() (bool, error)
	Serialize() (datatypes.JSON, error)
}

type TournamentGame struct {
	Scores        []int                           `json:"s"`
	Results       []realtime.TournamentGameResult `json:"r"`
	GameEndReason realtime.GameEndReason          `json:"g"`
}

const Unpaired = -1

type Pairing struct {
	Players  []string                        `json:"p"`
	Games    []*TournamentGame               `json:"g"`
	Outcomes []realtime.TournamentGameResult `json:"o"`
}

type FirstMethod int

const (
	// Firsts and seconds are set by the director
	// when submitting pairings. The player listed
	// first goes first.
	ManualFirst FirstMethod = iota

	// Random pairings do not use any previous first/second
	// data from the tournament and random assigns first and second
	// for the round
	RandomFirst

	// Automatic uses previous first/second records to decide
	// which player goes first.
	AutomaticFirst
)

type PlayerRoundInfo struct {
	Pairing          *Pairing `json:"p"`
	Record           []int    `json:"r"`
	Spread           int      `json:"s"`
	FirstsAndSeconds []int    `json:"f"`
}

type Standing struct {
	Player string
	Wins   int
	Losses int
	Draws  int
	Spread int
}

type PairingMethod int

const (
	Random PairingMethod = iota
	RoundRobin
	KingOfTheHill
	Elimination
	// Need to implement eventually
	// Swiss
	// Performance

	// Manual simply does not make any
	// pairings at all. The director
	// has to make all the pairings themselves.
	Manual
)

type TournamentType int

const (
	ClassicTournamentType TournamentType = iota
	// It's gonna be lit:
	ArenaTournamentType
)

type TournamentPersons struct {
	Persons map[string]int `json:"p"`
}

type TournamentControls struct {
	GameRequest    *realtime.GameRequest `json:"g"`
	PairingMethods []PairingMethod       `json:"p"`
	FirstMethods   []FirstMethod         `json:"f"`
	NumberOfRounds int                   `json:"n"`
	GamesPerRound  int                   `json:"g"`
	Type           TournamentType        `json:"t"`
	StartTime      time.Time             `json:"s"`
}

type TournamentDivision struct {
	Players         *TournamentPersons  `json:"p"`
	Controls        *TournamentControls `json:"c"`
	DivisionManager DivisionManager     `json:"m"`
}

type Tournament struct {
	sync.RWMutex
	UUID        string                         `json:"u"`
	Name        string                         `json:"n"`
	Description string                         `json:"e"`
	Directors   *TournamentPersons             `json:"d"`
	IsStarted   bool                           `json:"i"`
	Divisions   map[string]*TournamentDivision `json:"m"`
}
