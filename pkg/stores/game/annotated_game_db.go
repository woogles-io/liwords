package game

import (
	"context"

	"github.com/domino14/liwords/pkg/stores/common"
	"github.com/jackc/pgx/v4/pgxpool"
)

type AnnotatedDBStore struct {
	dbPool *pgxpool.Pool
}

func NewAnnotatedDBStore(p *pgxpool.Pool) (*AnnotatedDBStore, error) {
	return &AnnotatedDBStore{dbPool: p}, nil
}

func (s *AnnotatedDBStore) Disconnect() {
	s.dbPool.Close()
}

func (s *AnnotatedDBStore) AddAnnotatedGame(ctx context.Context, gid, uid string, private bool) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT into annotated_game_metadata 
			(game_uuid, creator_uuid, private_broadcast, finished)
		VALUES ($1, $2, $3, $4)
		`, gid, uid, private, false)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *AnnotatedDBStore) GetGamesForEditor(ctx context.Context, uid string, limit, offset int) error {
	// if we want to get the player names we do need quick data,
	// or the new game service db structures.
	// rows, err := s.dbPool.Query(ctx)

	return nil
}

func (s *AnnotatedDBStore) GetUnfinishedGamesForEditor(uid string) ([]string, error) {

	return nil, nil
}

// func (s *AnnotatedDBStore) AddEditor(gid string, uid string) error {

// 	return nil
// }

// func (s *AnnotatedDBStore) RemoveEditor(gid string, uid string) error {
// 	return nil
// }

// DeleteGame should delete the game from the original game table, plus
// the relationship with any editors. It should be rejected if
// the game is already finished.
func (s *AnnotatedDBStore) DeleteGame(gid, uid string) error {

	return nil
}
