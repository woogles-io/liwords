package stores

import (
	"context"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/common"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
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

func (s *DBStore) CreateAnnotatedGame(ctx context.Context, creatorUUID string, gameUUID string,
	private bool, quickdata *entity.Quickdata) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
	INSERT INTO annotated_game_metadata (game_uuid, creator_uuid, private_broadcast, done)
	VALUES ($1, $2, $3, $4)`, gameUUID, creatorUUID, private, false)

	if err != nil {
		return err
	}
	// Also insert it into the old games table. We will need to migrate this.
	_, err = tx.Exec(ctx, `
	INSERT INTO games (created_at, updated_at, uuid, type, quickdata)
	VALUES (NOW(), NOW(), $1, $2, $3)
	`, gameUUID, ipc.GameType_ANNOTATED, quickdata)
	if err != nil {
		return err
	}
	// All other fields are blank. We will update the quickdata field with
	// necessary metadata
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) DeleteAnnotatedGame(ctx context.Context, uuid string) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `DELETE from annotated_game_metadata WHERE game_uuid = $1`, uuid)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM games WHERE uuid = $1`, uuid)
	if err != nil {
		return err
	}
	// All other fields are blank. We will update the quickdata field with
	// necessary metadata
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// OutstandingGames returns a list of game IDs for games that are not yet done being
// annotated. The system will only allow a certain number of games to remain
// undone for an annotator.
func (s *DBStore) OutstandingGames(ctx context.Context, creatorUUID string) ([]string, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `SELECT game_uuid FROM annotated_game_metadata 
	WHERE creator_uuid = $1 AND done = 'f'`

	rows, err := tx.Query(ctx, query, creatorUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	uuids := []string{}
	for rows.Next() {
		var uuid string
		if err := rows.Scan(&uuid); rows != nil {
			return nil, err
		}
		uuids = append(uuids, uuid)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return uuids, nil
}

func (s *DBStore) GameOwnedBy(ctx context.Context, gid, uid string) (bool, error) {
	var ct int
	err := s.dbPool.QueryRow(ctx, `SELECT 1 FROM annotated_game_metadata
		WHERE creator_uuid = $1 AND game_uuid = $2 LIMIT 1`, uid, gid).Scan(&ct)
	if err != nil {
		return false, err
	}
	if ct == 1 {
		return true, nil
	}
	return false, nil
}
