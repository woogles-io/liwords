package soughtgame

import (
	"context"
	"fmt"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/common"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"

	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
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

func (s *DBStore) New(ctx context.Context, game *entity.SoughtGame) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// For open seek requests, receiverConnID
	// might return errors. This is okay, when setting
	// sought games, we just want to set whatever is available
	// and avoid conditional checks for open/closed seeks.
	id, _ := game.ID()
	seekerConnID, _ := game.SeekerConnID()
	seeker, _ := game.SeekerUserID()
	receiver, _ := game.ReceiverUserID()
	receiverConnID, _ := game.ReceiverConnID()

	_, err = tx.Exec(ctx, `INSERT INTO soughtgames (uuid, seeker, seeker_conn_id, receiver, receiver_conn_id, request) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, seeker, seekerConnID, receiver, receiverConnID, game.SeekRequest)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// Get gets the sought game with the given ID.
func (s *DBStore) Get(ctx context.Context, id string) (*entity.SoughtGame, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	sg, err := getSoughtGameBy(ctx, tx, &common.CommonDBConfig{SelectByType: common.SelectByUUID, Value: id})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, entity.NewWooglesError(pb.WooglesError_GAME_NO_LONGER_AVAILABLE)
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return sg, nil
}

// GetBySeekerConnID gets the sought game with the given socket connection ID for the seeker.
func (s *DBStore) GetBySeekerConnID(ctx context.Context, connID string) (*entity.SoughtGame, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	sg, err := getSoughtGameBy(ctx, tx, &common.CommonDBConfig{SelectByType: common.SelectBySeekerConnID, Value: connID})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return sg, nil
}

// GetByReceiverConnID gets the sought game with the given socket connection ID for the receiver.
func (s *DBStore) GetByReceiverConnID(ctx context.Context, connID string) (*entity.SoughtGame, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	sg, err := getSoughtGameBy(ctx, tx, &common.CommonDBConfig{SelectByType: common.SelectByReceiverConnID, Value: connID})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return sg, nil
}

func (s *DBStore) Delete(ctx context.Context, id string) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = common.Delete(ctx, tx, &common.CommonDBConfig{TableType: common.SoughtGamesTable, SelectByType: common.SelectByUUID, RowsAffectedType: common.AnyRowsAffected, Value: id})
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// ExpireOld expires old seek requests. Usually this shouldn't be necessary
// unless something weird happens.
func (s *DBStore) ExpireOld(ctx context.Context) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx, `DELETE FROM soughtgames WHERE created_at < NOW() - INTERVAL '1 hour'`)
	if err != nil {
		return err
	}
	if result.RowsAffected() > 0 {
		log.Info().Int("rows-affected", int(result.RowsAffected())).Msg("expire-old-seeks")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil
	}
	return nil
}

// DeleteForUser deletes the game by seeker ID
func (s *DBStore) DeleteForUser(ctx context.Context, userID string) (*entity.SoughtGame, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	sg, err := getSoughtGameBy(ctx, tx, &common.CommonDBConfig{SelectByType: common.SelectBySeekerID, Value: userID})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	err = common.Delete(ctx, tx, &common.CommonDBConfig{TableType: common.SoughtGamesTable, SelectByType: common.SelectBySeekerID, RowsAffectedType: common.AnyRowsAffected, Value: userID})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return sg, nil
}

// UpdateForReceiver updates the receiver's status when the receiver leaves
func (s *DBStore) UpdateForReceiver(ctx context.Context, receiverID string) (*entity.SoughtGame, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	sg, err := getSoughtGameBy(ctx, tx, &common.CommonDBConfig{SelectByType: common.SelectByReceiverID, Value: receiverID})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	sg.SeekRequest.ReceiverState = pb.SeekState_ABSENT
	result, err := tx.Exec(ctx, `UPDATE soughtgames SET request = jsonb_set(request, array['receiver_state'], $1) WHERE receiver = $2`, pb.SeekState_ABSENT, receiverID)
	if err != nil {
		return nil, err
	}
	if result.RowsAffected() != 1 {
		return nil, fmt.Errorf("failed to update receiver status: %s (%d rows affected)", receiverID, result.RowsAffected())
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return sg, nil
}

// DeleteForSeekerConnID deletes the game by connection ID
func (s *DBStore) DeleteForSeekerConnID(ctx context.Context, connID string) (*entity.SoughtGame, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	sg, err := getSoughtGameBy(ctx, tx, &common.CommonDBConfig{SelectByType: common.SelectBySeekerConnID, Value: connID})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	err = common.Delete(ctx, tx, &common.CommonDBConfig{TableType: common.SoughtGamesTable, SelectByType: common.SelectBySeekerConnID, RowsAffectedType: common.AnyRowsAffected, Value: connID})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return sg, nil
}

func (s *DBStore) UpdateForReceiverConnID(ctx context.Context, connID string) (*entity.SoughtGame, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	sg, err := getSoughtGameBy(ctx, tx, &common.CommonDBConfig{SelectByType: common.SelectByReceiverConnID, Value: connID})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	sg.SeekRequest.ReceiverState = pb.SeekState_ABSENT
	result, err := tx.Exec(ctx, `UPDATE soughtgames SET request = jsonb_set(request, array['receiver_state'], $1) WHERE receiver_conn_id = $2`, pb.SeekState_ABSENT, connID)
	if err != nil {
		return nil, err
	}
	if result.RowsAffected() != 1 {
		return nil, fmt.Errorf("failed to update receiver status: %s (%d rows affected)", connID, result.RowsAffected())
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return sg, nil
}

// ListOpenSeeks lists all open seek requests for receiverID, in tourneyID (optional)
func (s *DBStore) ListOpenSeeks(ctx context.Context, receiverID, tourneyID string) ([]*entity.SoughtGame, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var rows pgx.Rows
	if tourneyID != "" {
		rows, err = tx.Query(ctx, `SELECT request FROM soughtgames WHERE receiver = $1 AND request->>'tournament_id' = $2`, receiverID, tourneyID)
	} else {
		rows, err = tx.Query(ctx, `SELECT request FROM soughtgames WHERE receiver = $1 OR receiver = ''`, receiverID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	games := []*entity.SoughtGame{}

	for rows.Next() {
		var req pb.SeekRequest
		if err := rows.Scan(&req); err != nil {
			return nil, err
		}
		games = append(games, &entity.SoughtGame{SeekRequest: &req})
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return games, nil
}

// ExistsForUser returns true if the user already has an outstanding seek request.
func (s *DBStore) ExistsForUser(ctx context.Context, userID string) (bool, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM soughtgames WHERE seeker = $1)`, userID).Scan(&exists)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return exists, nil
}

// UserMatchedBy returns true if there is an open seek request from matcher for user
func (s *DBStore) UserMatchedBy(ctx context.Context, userID, matcher string) (bool, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM soughtgames WHERE receiver = $1 AND seeker = $2)`, userID, matcher).Scan(&exists)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return exists, nil
}

func getSoughtGameBy(ctx context.Context, tx pgx.Tx, cfg *common.CommonDBConfig) (*entity.SoughtGame, error) {
	req := pb.SeekRequest{}
	err := tx.QueryRow(ctx, fmt.Sprintf("SELECT request FROM soughtgames WHERE %s = $1", common.SelectByTypeToString[cfg.SelectByType]), cfg.Value).Scan(&req)
	if err != nil {
		return nil, err
	}
	return &entity.SoughtGame{SeekRequest: &req}, nil
}
