package mod

import (
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"

	"github.com/jinzhu/gorm"
)

type NotorietyStore struct {
	db *gorm.DB
}

type notoriousgame struct {
	GameID    string `gorm:"index"`
	PlayerID  string `gorm:"index"`
	Type      int    `gorm:"index"`
	Timestamp int64  `gorm:"index"`
}

func NewNotorietyStore(dbURL string) (*NotorietyStore, error) {
	db, err := gorm.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&notoriousgame{})
	return &NotorietyStore{db: db}, nil
}

func (ns *NotorietyStore) AddNotoriousGame(playerID string, gameID string, gameType int, time int64) error {
	dbi := &notoriousgame{
		GameID:    gameID,
		PlayerID:  playerID,
		Type:      gameType,
		Timestamp: time,
	}
	result := ns.db.Create(dbi)
	return result.Error
}

func (ns *NotorietyStore) GetNotoriousGames(playerID string) ([]*ms.NotoriousGame, error) {
	var games []notoriousgame

	result := ns.db.Table("notoriousgames").
		Select("game_id, type, timestamp").
		Where("player_id = ?", []interface{}{playerID}).
		Order("timestamp").Scan(&games)
	if result.Error != nil {
		return nil, result.Error
	}

	items := make([]*ms.NotoriousGame, len(games))
	for idx, dbgame := range games {

		items[idx] = &ms.NotoriousGame{
			Id:   dbgame.GameID,
			Type: ms.NotoriousGameType(dbgame.Type),
			// Converting from a Unix timestamp to
			// protobuf timestamp is quite expensive.
			// As far as I know, it involves two separate
			// conversions. We will omit this time for now.
			// Suggestions welcome.
			// CreatedAt: dbgame.Timestamp,
		}
	}

	return items, nil
}

func (ns *NotorietyStore) DeleteNotoriousGames(playerID string) error {
	result := ns.db.Table("notoriousgames").Delete(&notoriousgame{PlayerID: playerID})
	return result.Error
}

func (ns *NotorietyStore) Disconnect() {
	ns.db.Close()
}
