package game

import (
	"context"
	"os/user"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/jinzhu/gorm"
)

// DBStore is a postgres-backed store for games.
type DBStore struct {
	db *gorm.DB

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

	TimeOfLastUpdate uint
	TimeStarted      uint
	TimeRem0         uint
	TimeRem1         uint

	// Minutes
	MaxOvertime uint

	Started       bool
	GameEndReason uint
	WinnerIdx     uint
	LoserIdx      uint

	// Protobuf representations of the game request and history.
	Request []byte
	History []byte
}

// NewDBStore creates a new DB store for games.
func NewDBStore(dbURL string) (*DBStore, error) {
	db, err := gorm.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&game{})
	db.Model(&game{}).AddForeignKey("player0_id", "users(id)", "RESTRICT", "RESTRICT")
	db.Model(&game{}).AddForeignKey("player1_id", "users(id)", "RESTRICT", "RESTRICT")
	return &DBStore{db: db}, nil
}

// SetGameEventChan sets the game event channel to the passed in channel.
func (s *DBStore) SetGameEventChan(c chan<- *entity.EventWrapper) {
	s.gameEventChan = c
}

// Get creates an instantiated entity.Game from the database.
// This function should almost never be called. The db store should be wrapped
// with a cache. Only API nodes that have this game in its cache should respond
// to requests.
func (s *DBStore) Get(ctx context.Context, id string) (*entity.Game, error) {
	g := &game{}

	if result := s.db.Where("uuid = ?", id).First(g); result.Error != nil {
		return nil, result.Error
	}

	return nil, nil
}
