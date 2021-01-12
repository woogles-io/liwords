package entity

import (
	"encoding/json"
	"sync"

	"gorm.io/datatypes"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

type DivisionManager interface {
	SubmitResult(int, string, string, int, int, realtime.TournamentGameResult,
		realtime.TournamentGameResult, realtime.GameEndReason, bool, int) error
	PairRound(int) error
	GetStandings(int) ([]*realtime.PlayerStanding, error)
	SetPairing(string, string, int, bool) error
	AddPlayers(*realtime.TournamentPersons) error
	RemovePlayers(*realtime.TournamentPersons) error
	IsRoundReady(int) (bool, error)
	IsRoundComplete(int) (bool, error)
	IsFinished() (bool, error)
	StartRound() error
	ToResponse() (*realtime.TournamentDivisionDataResponse, error)
	SetReadyForGame(userID, connID string, round, gameIndex int, unready bool) ([]string, bool, error)
	SetLastStarted(*realtime.TournamentRoundStarted) error
	Serialize() (datatypes.JSON, error)
}

type CompetitionType string

const (
	// TypeStandard is a standard tournament
	TypeStandard CompetitionType = "tournament"
	// TypeClub is a club/clubhouse
	TypeClub = "club"
	// TypeChild is spawned from a club or tournament
	TypeChild = "child"
	// TypeLegacy is a tournament, but in club/clubhouse mode. The only different
	// from a clubhouse is that it can have a /tournament URL.
	TypeLegacy = "legacy"
)

const (
	ByeScore     int = 50
	ForfeitScore int = -50
)

type TournamentType int

const (
	ClassicTournamentType TournamentType = iota
	// It's gonna be lit:
	ArenaTournamentType
)

// This type is a struct in anticipation of
// future additional properties
type PlayerProperties struct {
	Removed bool
}

type TournamentDivision struct {
	Players            *realtime.TournamentPersons  `json:"players"`
	Controls           *realtime.TournamentControls `json:"controls"`
	ManagerType        TournamentType               `json:"mgrType"`
	DivisionRawMessage json.RawMessage              `json:"json"`
	DivisionManager    DivisionManager              `json:"-"`
}

type Tournament struct {
	sync.RWMutex
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"desc"`
	// XXX: We will likely remove the following two fields
	AliasOf string `json:"aliasOf"`
	URL     string `json:"url"`
	// XXX: Investigate above.
	ExecutiveDirector string                         `json:"execDirector"`
	Directors         *realtime.TournamentPersons    `json:"directors"`
	IsStarted         bool                           `json:"started"`
	Divisions         map[string]*TournamentDivision `json:"divs"`
	DefaultSettings   *realtime.GameRequest          `json:"settings"`
	Type              CompetitionType                `json:"type"`
	ParentID          string                         `json:"parent"`
	Slug              string                         `json:"slug"`
}
