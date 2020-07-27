package game

import (
	"context"
	"encoding/json"
	"errors"

	"google.golang.org/protobuf/proto"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/user"
	pkguser "github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/domino14/macondo/board"
	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

// DBStore is a postgres-backed store for games.
type DBStore struct {
	cfg *config.Config
	db  *gorm.DB

	userStore pkguser.Store

	// This reference is here so we can copy it to every game we pull
	// from the database.
	// All game events go down the same channel.
	gameEventChan chan<- *entity.EventWrapper
}

type game struct {
	gorm.Model
	UUID string `gorm:"type:varchar(24);index"`

	Player0ID uint
	Player0   user.User

	Player1ID uint
	Player1   user.User

	Timers postgres.Jsonb // A JSON blob containing the game timers.

	Started       bool
	GameEndReason int
	WinnerIdx     int
	LoserIdx      int

	// Protobuf representations of the game request and history.
	Request []byte
	History []byte
}

// NewDBStore creates a new DB store for games.
func NewDBStore(dbURL string, config *config.Config, userStore pkguser.Store) (*DBStore, error) {
	db, err := gorm.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&game{})
	db.Model(&game{}).AddForeignKey("player0_id", "users(id)", "RESTRICT", "RESTRICT")
	db.Model(&game{}).AddForeignKey("player1_id", "users(id)", "RESTRICT", "RESTRICT")
	return &DBStore{db: db, cfg: config, userStore: userStore}, nil
}

// SetGameEventChan sets the game event channel to the passed in channel.
func (s *DBStore) SetGameEventChan(c chan<- *entity.EventWrapper) {
	s.gameEventChan = c
}

// Get creates an instantiated entity.Game from the database.
// This function should almost never be called during a live game.
// The db store should be wrapped with a cache.
// Only API nodes that have this game in its cache should respond to requests.
func (s *DBStore) Get(ctx context.Context, id string) (*entity.Game, error) {
	g := &game{}

	if result := s.db.Where("uuid = ?", id).First(g); result.Error != nil {
		return nil, result.Error
	}

	var tdata entity.Timers
	err := json.Unmarshal(g.Timers.RawMessage, &tdata)
	if err != nil {
		return nil, err
	}

	return FromState(g.Player0, g.Player1, tdata, g.Started, g.GameEndReason,
		g.WinnerIdx, g.LoserIdx, g.Request, g.History, s.gameEventChan, s.cfg)
}

// FromState returns an entity.Game from a DB State.
func FromState(p0, p1 user.User, timers entity.Timers, Started bool,
	GameEndReason, WinnerIdx, LoserIdx int, reqBytes, histBytes []byte,
	gameEventChan chan<- *entity.EventWrapper, cfg *config.Config) (*entity.Game, error) {

	g := &entity.Game{
		Started:       Started,
		Timers:        timers,
		GameEndReason: pb.GameEndReason(GameEndReason),
		WinnerIdx:     WinnerIdx,
		LoserIdx:      LoserIdx,
		ChangeHook:    gameEventChan,
	}
	// Now copy the request
	req := &pb.GameRequest{}
	err := proto.Unmarshal(reqBytes, req)
	if err != nil {
		return nil, err
	}
	g.GameReq = req
	// Then unmarshal the history and start a game from it.
	hist := &macondopb.GameHistory{}
	err = proto.Unmarshal(histBytes, hist)
	if err != nil {
		return nil, err
	}

	var bd []string
	switch req.Rules.BoardLayoutName {
	case entity.CrosswordGame:
		bd = board.CrosswordGameBoard
	default:
		return nil, errors.New("unsupported board layout")
	}

	rules, err := macondogame.NewGameRules(&cfg.MacondoConfig, bd,
		req.Lexicon, req.Rules.LetterDistributionName)

	if err != nil {
		return nil, err
	}

	mcg, err := macondogame.NewFromHistory(hist, rules, len(hist.Events))
	if err != nil {
		return nil, err
	}

	g.Game = *mcg
	return g, nil
}

// Set takes in a game entity that _already exists_ in the DB, and writes it to
// the database.
func (s *DBStore) Set(ctx context.Context, g *entity.Game) error {

	dbg, err := s.toDBObj(ctx, g)
	if err != nil {
		return err
	}
	result := s.db.Where("uuid = ?", g.GameID()).Save(dbg)

	return result.Error
}

// Create saves a brand new entity to the database
func (s *DBStore) Create(ctx context.Context, g *entity.Game) error {
	dbg, err := s.toDBObj(ctx, g)
	if err != nil {
		return err
	}
	result := s.db.Create(dbg)
	return result.Error
}

func (s *DBStore) toDBObj(ctx context.Context, g *entity.Game) (*game, error) {
	timers, err := json.Marshal(g.Timers)
	if err != nil {
		return nil, err
	}
	req, err := proto.Marshal(g.GameReq)
	if err != nil {
		return nil, err
	}
	hist, err := proto.Marshal(g.History())
	if err != nil {
		return nil, err
	}
	players := g.History().Players

	// XXX: Maybe can cache later.
	p0, err := s.userStore.GetByUUID(ctx, players[0].UserId)
	if err != nil {
		return nil, err
	}

	p1, err := s.userStore.GetByUUID(ctx, players[1].UserId)
	if err != nil {
		return nil, err
	}

	dbg := &game{
		UUID:      g.GameID(),
		Player0ID: p0.ID,
		Player1ID: p1.ID,
		Timers:    postgres.Jsonb{RawMessage: timers},

		Started:       g.Started,
		GameEndReason: int(g.GameEndReason),
		WinnerIdx:     g.WinnerIdx,
		LoserIdx:      g.LoserIdx,
		Request:       req,
		History:       hist,
	}
	return dbg, nil
}

func (s *DBStore) Disconnect() {
	s.db.Close()
}
