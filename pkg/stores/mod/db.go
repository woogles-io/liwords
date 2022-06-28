package mod

import (
	"context"

	"github.com/domino14/liwords/pkg/stores/common"
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	"github.com/jackc/pgx/v4/pgxpool"
)

type DBStore struct {
	dbPool *pgxpool.Pool
}

func NewDBStore(p *pgxpool.Pool) (*DBStore, error) {
	return &DBStore{dbPool: p}, nil
}

func (s *DBStore) Disconnect() {
	s.dbPool.Close()
}

func (s *DBStore) AddNotoriousGame(ctx context.Context, playerID string, gameID string, gameType int, time int64) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO notoriousgames (game_id, player_id, type, timestamp) VALUES ($1, $2, $3, $4)`, gameID, playerID, gameType, time)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *DBStore) GetNotoriousGames(ctx context.Context, playerID string, limit int) ([]*ms.NotoriousGame, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT game_id, type FROM notoriousgames WHERE player_id = $1 ORDER BY timestamp ASC LIMIT $2`, playerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	games := []*ms.NotoriousGame{}

	for rows.Next() {
		var gameID string
		var gameType int
		if err := rows.Scan(&gameID, &gameType); err != nil {
			return nil, err
		}
		games = append(games, &ms.NotoriousGame{Id: gameID, Type: ms.NotoriousGameType(gameType)})
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return games, nil
}

func (s *DBStore) DeleteNotoriousGames(ctx context.Context, playerID string) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM notoriousgames WHERE player_id = $1`, playerID)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}
