package entity

import (
	"encoding/json"
	"sync"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

type DivisionManager interface {
	SubmitResult(int, string, string, int, int, realtime.TournamentGameResult,
		realtime.TournamentGameResult, realtime.GameEndReason, bool, int, string) (*realtime.DivisionPairingsResponse, error)
	PairRound(int, bool) (*realtime.DivisionPairingsResponse, error)
	DeletePairings(int) error
	GetStandings(int, bool) (*realtime.RoundStandings, error)
	GetCurrentRound() int
	GetPlayers() *realtime.TournamentPersons
	SetPairing(string, string, int) (*realtime.DivisionPairingsResponse, error)
	SetSingleRoundControls(int, *realtime.RoundControl) (*realtime.RoundControl, error)
	SetRoundControls([]*realtime.RoundControl) (*realtime.DivisionPairingsResponse, []*realtime.RoundControl, error)
	SetDivisionControls(*realtime.DivisionControls) (*realtime.DivisionControls, error)
	GetDivisionControls() *realtime.DivisionControls
	AddPlayers(*realtime.TournamentPersons) (*realtime.DivisionPairingsResponse, error)
	RemovePlayers(*realtime.TournamentPersons) (*realtime.DivisionPairingsResponse, error)
	IsRoundReady(int) (bool, error)
	IsRoundComplete(int) (bool, error)
	IsStarted() bool
	IsFinished() (bool, error)
	StartRound() error
	GetXHRResponse() (*realtime.TournamentDivisionDataResponse, error)
	SetReadyForGame(userID, connID string, round, gameIndex int, unready bool) ([]string, bool, error)
	ClearReadyStates(userID string, round, gameIndex int) ([]*realtime.Pairing, error)
	ResetToBeginning() error
}

/**	SetCheckedIn(userID string) error
ClearCheckedIn()*/

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

type TournamentDivision struct {
	ManagerType        TournamentType  `json:"mgrType"`
	DivisionRawMessage json.RawMessage `json:"json"`
	DivisionManager    DivisionManager `json:"-"`
}

type TournamentMeta struct {
	Disclaimer                string                `json:"disclaimer"`
	TileStyle                 string                `json:"tileStyle"`
	BoardStyle                string                `json:"boardStyle"`
	DefaultClubSettings       *realtime.GameRequest `json:"defaultClubSettings"`
	FreeformClubSettingFields []string              `json:"freeformClubSettingFields"`
	Password                  string                `json:"password"`
	Logo                      string                `json:"logo"`
	Color                     string                `json:"color"`
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
	IsFinished        bool                           `json:"finished"`
	Divisions         map[string]*TournamentDivision `json:"divs"`
	Type              CompetitionType                `json:"type"`
	ParentID          string                         `json:"parent"`
	Slug              string                         `json:"slug"`
	ExtraMeta         *TournamentMeta                `json:"extraMeta"`
}
