package stats

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
)

// A ListStatStore stores "list-based" statistics (for example, lists of player
// bingos).
type DBStore struct {
	dbPool *pgxpool.Pool
}

func NewDBStore(p *pgxpool.Pool) (*DBStore, error) {
	return &DBStore{dbPool: p}, nil
}

func (s *DBStore) Disconnect() {
	s.dbPool.Close()
}

// XXX: This should be a transaction that queues up many inserts.
// Fix before beta.
func (s *DBStore) AddListItem(ctx context.Context, gameID string, playerID string, statType int,
	time int64, item entity.ListDatum) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO liststats (game_id, player_id, timestamp, stat_type, item) VALUES ($1, $2, $3, $4, $5)`,
		gameID, playerID, time, statType, item)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// GetListItems gets list items for a stat type, a list of game IDs, and an optional
// player ID.
// XXX: This function will need to be modified a bit to work with a player ID of
// "opponent" -- that is when we want to get user list stats for arbitrary opponents.
func (s *DBStore) GetListItems(ctx context.Context, statType int, gameIds []string, playerID string) ([]*entity.ListItem, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	// playerID is optional
	// gameIds should have at least one item.
	if len(gameIds) == 0 {
		return nil, errors.New("need to provide a game id")
	}
	where := " stat_type = $1 "
	args := []interface{}{statType}
	nextPosArgCounter := 2
	if playerID != "" {
		nextPosArgCounter = 3
		where += " AND player_id = $2 "
		args = append(args, playerID)
	}

	inClause := common.BuildIn(len(gameIds), nextPosArgCounter)

	where += " AND game_id IN (" + inClause + ")"
	for _, gid := range gameIds {
		args = append(args, gid)
	}

	log.Debug().Str("where", where).Interface("args", args).Msg("query")

	query := fmt.Sprintf("SELECT game_id, player_id, timestamp, item FROM liststats WHERE %s ORDER BY timestamp", where)
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	listItems := []*entity.ListItem{}
	for rows.Next() {
		var gameID pgtype.Text
		var playerID pgtype.Text
		var timestamp pgtype.Int8
		var item entity.ListDatum
		if err := rows.Scan(&gameID, &playerID, &timestamp, &item); err != nil {
			return nil, err
		}
		listItems = append(listItems, &entity.ListItem{
			GameId:   gameID.String,
			PlayerId: playerID.String,
			Time:     timestamp.Int64,
			Item:     item,
		})
	}
	return listItems, nil
}
