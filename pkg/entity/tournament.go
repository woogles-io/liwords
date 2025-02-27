package entity

import (
	"encoding/json"
	"sync"
	"time"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type DivisionManager interface {
	SubmitResult(int, string, string, int, int, pb.TournamentGameResult,
		pb.TournamentGameResult, pb.GameEndReason, bool, int, string) (*pb.DivisionPairingsResponse, error)
	PairRound(int, bool) (*pb.DivisionPairingsResponse, error)
	DeletePairings(int) error
	GetStandings(int) (*pb.RoundStandings, int, error)
	GetCurrentRound() int
	GetPlayers() *pb.TournamentPersons
	SetPairing(string, string, int, pb.TournamentGameResult) (*pb.DivisionPairingsResponse, error)
	SetSingleRoundControls(int, *pb.RoundControl) (*pb.RoundControl, error)
	SetRoundControls([]*pb.RoundControl) (*pb.DivisionPairingsResponse, []*pb.RoundControl, error)
	SetDivisionControls(*pb.DivisionControls) (*pb.DivisionControls, map[int32]*pb.RoundStandings, error)
	GetDivisionControls() *pb.DivisionControls
	AddPlayers(*pb.TournamentPersons) (*pb.DivisionPairingsResponse, error)
	RemovePlayers(*pb.TournamentPersons) (*pb.DivisionPairingsResponse, error)
	IsRoundReady(int) error
	IsRoundComplete(int) (bool, error)
	IsStarted() bool
	IsFinished() (bool, error)
	StartRound(bool) error
	IsRoundStartable() error
	GetXHRResponse() (*pb.TournamentDivisionDataResponse, error)
	SetReadyForGame(userID, connID string, round, gameIndex int, unready bool) ([]string, bool, error)
	ClearReadyStates(userID string, round, gameIndex int) ([]*pb.Pairing, error)
	ResetToBeginning() error
	ChangeName(string)
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
	Disclaimer                string          `json:"disclaimer"`
	TileStyle                 string          `json:"tileStyle"`
	BoardStyle                string          `json:"boardStyle"`
	DefaultClubSettings       *pb.GameRequest `json:"defaultClubSettings"`
	FreeformClubSettingFields []string        `json:"freeformClubSettingFields"`
	Password                  string          `json:"password"`
	Logo                      string          `json:"logo"`
	Color                     string          `json:"color"`
	PrivateAnalysis           bool            `json:"privateAnalysis"`
	IRLMode                   bool            `json:"irlMode"`
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
	ExecutiveDirector  string                         `json:"execDirector"`
	Directors          *pb.TournamentPersons          `json:"directors"`
	IsStarted          bool                           `json:"started"`
	IsFinished         bool                           `json:"finished"`
	Divisions          map[string]*TournamentDivision `json:"divs"`
	Type               CompetitionType                `json:"type"`
	ParentID           string                         `json:"parent"`
	Slug               string                         `json:"slug"`
	ExtraMeta          *TournamentMeta                `json:"extraMeta"`
	ScheduledStartTime *time.Time                     `json:"scheduledStartTime"`
	ScheduledEndTime   *time.Time                     `json:"scheduledEndTime"`
	CreatedBy          uint
}
