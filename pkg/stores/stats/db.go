package stats

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// A ListStatStore stores "list-based" statistics (for example, lists of player
// bingos).
type DBStore struct {
	dbPool  *pgxpool.Pool
	queries *models.Queries
}

func NewDBStore(p *pgxpool.Pool) (*DBStore, error) {
	return &DBStore{dbPool: p, queries: models.New(p)}, nil
}

func (s *DBStore) Disconnect() {
	s.dbPool.Close()
}

// XXX: This should be a transaction that queues up many inserts.
// Fix before beta.
func (s *DBStore) AddListItem(ctx context.Context, gameID string, playerID string, statType int,
	time int64, item entity.ListDatum) error {
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return s.queries.AddListItem(ctx, models.AddListItemParams{
		GameID:    pgtype.Text{String: gameID, Valid: true},
		PlayerID:  pgtype.Text{String: playerID, Valid: true},
		Timestamp: pgtype.Int8{Int64: time, Valid: true},
		StatType:  pgtype.Int4{Int32: int32(statType), Valid: true},
		Item:      data,
	})
}

// GetListItems gets list items for a stat type, a list of game IDs, and an optional
// player ID.
// XXX: This function will need to be modified a bit to work with a player ID of
// "opponent" -- that is when we want to get user list stats for arbitrary opponents.
func (s *DBStore) GetListItems(ctx context.Context, statType int, gameIds []string, playerID string) ([]*entity.ListItem, error) {
	// gameIds should have at least one item.
	if len(gameIds) == 0 {
		return nil, errors.New("need to provide a game id")
	}

	rows, err := s.queries.GetListItems(ctx, models.GetListItemsParams{
		StatType: pgtype.Int4{Int32: int32(statType), Valid: true},
		PlayerID: playerID,
		GameIds:  gameIds,
	})
	if err != nil {
		return nil, err
	}

	listItems := []*entity.ListItem{}
	for _, row := range rows {
		var item entity.ListDatum
		if len(row.Item) > 0 {
			if err := json.Unmarshal(row.Item, &item); err != nil {
				return nil, err
			}
		}
		listItems = append(listItems, &entity.ListItem{
			GameId:   row.GameID.String,
			PlayerId: row.PlayerID.String,
			Time:     row.Timestamp.Int64,
			Item:     item,
		})
	}
	return listItems, nil
}
