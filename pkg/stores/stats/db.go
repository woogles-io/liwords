package stats

import (
	"encoding/json"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"google.golang.org/protobuf/proto"
)

// A ListStatStore stores "list-based" statistics (for example, lists of player
// bingos).
type ListStatStore struct {
	db *gorm.DB
}

type liststat struct {
	GameID    string `gorm:"index"`
	PlayerID  string `gorm:"index"`
	Timestamp int64  `gorm:"index"` // unix timestamp in milliseconds

	StatType int
	Item     postgres.Jsonb
}

func NewListStatStore(dbURL string) (*ListStatStore, error) {
	db, err := gorm.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&liststat{})
	return &ListStatStore{db: db}, nil
}

func (l *ListStatStore) AddListItem(gameId string, playerId string, statType int, time int64, item interface{}) error {

	jsonitem, err := json.Marshal(item)
	if err != nil {
		return err
	}
	dbi := &liststat{
		GameID:    gameId,
		PlayerID:  playerId,
		Timestamp: time,
		StatType:  statType,
		Item:      postgres.Jsonb{RawMessage: jsonitem},
	}
	result := l.db.Create(dbi)
	return result.Error
}

func (l *ListStatStore) GetListItems(statType int, gameIds []string, playerId string) ([]*entity.ListItem, error) {
	var stats []liststat

	result := l.db.Table("liststats").
		Select("game_id, player_id, timestamp, item").
		Where("stat_type = ?", statType).
		Order("timestamp").Scan(&stats)
	if result.Error != nil {
		return nil, result.Error
	}

	items := make([]*entity.ListItem, len(stats))
	for idx, dbstat := range stats {
		req := &pb.GameRequest{}
		err := proto.Unmarshal(a.Request, req)
		if err != nil {
			return nil, err
		}

		items[idx] = &entity.ListItem{
			GameId:   dbstat.GameID,
			PlayerId: dbstat.PlayerID,
			Time:     dbstat.Timestamp,
		}
	}

	return nil, nil
}
