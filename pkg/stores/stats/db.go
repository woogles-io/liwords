package stats

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/rs/zerolog/log"
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

// XXX: This should be a transaction that queues up many inserts.
// Fix before beta.
func (l *ListStatStore) AddListItem(gameID string, playerID string, statType int,
	time int64, item entity.ListDatum) error {

	jsonitem, err := json.Marshal(item)
	if err != nil {
		return err
	}
	dbi := &liststat{
		GameID:    gameID,
		PlayerID:  playerID,
		Timestamp: time,
		StatType:  statType,
		Item:      postgres.Jsonb{RawMessage: jsonitem},
	}
	result := l.db.Create(dbi)
	return result.Error
}

// GetListItems gets list items for a stat type, a list of game IDs, and an optional
// player ID.
// XXX: This function will need to be modified a bit to work with a player ID of
// "opponent" -- that is when we want to get user list stats for arbitrary opponents.
func (l *ListStatStore) GetListItems(statType int, gameIds []string, playerID string) ([]*entity.ListItem, error) {
	var stats []liststat

	// playerID is optional
	// gameIds should have at least one item.
	if len(gameIds) == 0 {
		return nil, errors.New("need to provide a game id")
	}
	where := "stat_type = ?"
	args := []interface{}{statType}
	if playerID != "" {
		where += " AND player_id = ?"
		args = append(args, playerID)
	}

	inClause := strings.Repeat("?,", len(gameIds))
	inClause = strings.TrimSuffix(inClause, ",")

	where += " AND game_id IN (" + inClause + ")"
	for _, gid := range gameIds {
		args = append(args, gid)
	}

	log.Info().Str("where", where).Interface("args", args).Msg("query")

	result := l.db.Table("liststats").
		Select("game_id, player_id, timestamp, item").
		Where(where, args...).
		Order("timestamp").Scan(&stats)
	if result.Error != nil {
		return nil, result.Error
	}

	items := make([]*entity.ListItem, len(stats))
	for idx, dbstat := range stats {
		datum := entity.ListDatum{}
		err := json.Unmarshal(dbstat.Item.RawMessage, &datum)
		if err != nil {
			return nil, err
		}

		items[idx] = &entity.ListItem{
			GameId:   dbstat.GameID,
			PlayerId: dbstat.PlayerID,
			Time:     dbstat.Timestamp,
			Item:     datum,
		}
	}

	return items, nil
}

func (l *ListStatStore) Disconnect() {
	l.db.Close()
}
