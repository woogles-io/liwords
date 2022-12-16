package stores

import (
	"context"

	"github.com/domino14/liwords/pkg/stores/common"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lithammer/shortuuid"
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

func (s *DBStore) CreateAnnotatedGame(ctx context.Context, creatorUUID string, private bool) (string, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	guuid := shortuuid.New()
	_, err = tx.Exec(ctx, `
	INSERT INTO annotated_game_metadata (game_uuid, creator_uuid, private_broadcast, done)
	VALUES ($1, $2, $3, $4)`, guuid, creatorUUID, private, false)

	if err != nil {
		return "", err
	}
	// Also insert it into the old games table. We will need to migrate this.
	_, err = tx.Exec(ctx, `
	INSERT INTO games (created_at, updated_at, uuid, type)
	VALUES (NOW(), NOW(), $1, $2)
	`, guuid, ipc.GameType_ANNOTATED)

	// All other fields are blank. We will update the quickdata field with
	// necessary metadata
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return guuid, nil

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

	query := `SELECT game_uuid WHERE creator_uuid = $1 AND done = 'f'`

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
