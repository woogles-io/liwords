package entity

import (
	"encoding/json"
	"sync"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

type DivisionManager interface {
	SubmitResult(int, string, string, int, int, realtime.TournamentGameResult,
		realtime.TournamentGameResult, realtime.GameEndReason, bool, int, string) (map[int][]*realtime.Pairing, error)
	PairRound(int) (map[int][]*realtime.Pairing, error)
	GetStandings(int) ([]*realtime.PlayerStanding, error)
	GetCurrentRound() int
	SetPairing(string, string, int) (map[int][]*realtime.Pairing, error)
	SetSingleRoundControls(int, *realtime.RoundControl) (*realtime.RoundControl, error)
	SetRoundControls([]*realtime.RoundControl) (map[int][]*realtime.Pairing, []*realtime.RoundControl, error)
	SetDivisionControls(*realtime.DivisionControls) (*realtime.DivisionControls, error)
	AddPlayers(*realtime.TournamentPersons) (map[int][]*realtime.Pairing, error)
	RemovePlayers(*realtime.TournamentPersons) (map[int][]*realtime.Pairing, error)
	IsRoundReady(int) (bool, error)
	IsRoundComplete(int) (bool, error)
	IsStarted() bool
	IsFinished() (bool, error)
	StartRound() error
	GetXHRResponse() (*realtime.TournamentDivisionDataResponse, error)
	SetReadyForGame(userID, connID string, round, gameIndex int, unready bool) ([]string, bool, error)
	ClearReadyStates(userID string, round, gameIndex int) (map[int][]*realtime.Pairing, error)
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

type TournamentDivision struct {
	ManagerType        TournamentType  `json:"mgrType"`
	DivisionRawMessage json.RawMessage `json:"json"`
	DivisionManager    DivisionManager `json:"-"`
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
	DefaultSettings   *realtime.GameRequest          `json:"settings"`
	Type              CompetitionType                `json:"type"`
	ParentID          string                         `json:"parent"`
	Slug              string                         `json:"slug"`
}
