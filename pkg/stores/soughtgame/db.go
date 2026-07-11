package soughtgame

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"google.golang.org/protobuf/encoding/protojson"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

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

func soughtGameFromBytes(data []byte) (*entity.SoughtGame, error) {
	req := &pb.SeekRequest{}
	if len(data) > 0 {
		if err := protojson.Unmarshal(data, req); err != nil {
			return nil, err
		}
	}
	return &entity.SoughtGame{SeekRequest: req}, nil
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

	// Extract game_mode from the GameRequest (nullable for backwards compatibility)
	var gameMode *int32
	if game.SeekRequest != nil && game.SeekRequest.GameRequest != nil {
		mode := int32(game.SeekRequest.GameRequest.GameMode)
		gameMode = &mode
	}

	_, err = tx.Exec(ctx, `INSERT INTO soughtgames (uuid, seeker, seeker_conn_id, receiver, receiver_conn_id, request, game_mode) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, seeker, seekerConnID, receiver, receiverConnID, game.SeekRequest, gameMode)
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
	data, err := s.queries.GetSoughtGameByUUID(ctx, pgtype.Text{String: id, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, entity.NewWooglesError(pb.WooglesError_GAME_NO_LONGER_AVAILABLE)
		}
		return nil, err
	}
	return soughtGameFromBytes(data)
}

// GetBySeekerConnID gets the sought game with the given socket connection ID for the seeker.
func (s *DBStore) GetBySeekerConnID(ctx context.Context, connID string) (*entity.SoughtGame, error) {
	data, err := s.queries.GetSoughtGameBySeekerConnID(ctx, pgtype.Text{String: connID, Valid: true})
	if err != nil {
		return nil, err
	}
	return soughtGameFromBytes(data)
}

// GetByReceiverConnID gets the sought game with the given socket connection ID for the receiver.
func (s *DBStore) GetByReceiverConnID(ctx context.Context, connID string) (*entity.SoughtGame, error) {
	data, err := s.queries.GetSoughtGameByReceiverConnID(ctx, pgtype.Text{String: connID, Valid: true})
	if err != nil {
		return nil, err
	}
	return soughtGameFromBytes(data)
}

func (s *DBStore) Delete(ctx context.Context, id string) error {
	return s.queries.DeleteSoughtGameByUUID(ctx, pgtype.Text{String: id, Valid: true})
}

// ExpireOld expires old seek requests. Usually this shouldn't be necessary
// unless something weird happens.
// Real-time seeks expire after 2 hours, correspondence seeks expire after 60 hours.
func (s *DBStore) ExpireOld(ctx context.Context) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil
	}
	defer tx.Rollback(ctx)

	// Delete real-time seeks older than 2 hours (game_mode IS NULL or game_mode = 0)
	result, err := tx.Exec(ctx, `DELETE FROM soughtgames WHERE (game_mode IS NULL OR game_mode = 0) AND created_at < NOW() - INTERVAL '2 hours'`)
	if err != nil {
		return err
	}
	if result.RowsAffected() > 0 {
		log.Info().Int("rows-affected", int(result.RowsAffected())).Msg("expire-old-realtime-seeks")
	}

	// Delete correspondence seeks older than 60 hours (game_mode = 1)
	result, err = tx.Exec(ctx, `DELETE FROM soughtgames WHERE game_mode = 1 AND created_at < NOW() - INTERVAL '60 hours'`)
	if err != nil {
		return err
	}
	if result.RowsAffected() > 0 {
		log.Info().Int("rows-affected", int(result.RowsAffected())).Msg("expire-old-correspondence-seeks")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil
	}
	return nil
}

// DeleteForUser deletes the game by seeker ID.
// Correspondence seeks are not deleted when user leaves - they persist for the receiver to accept later.
func (s *DBStore) DeleteForUser(ctx context.Context, userID string) (*entity.SoughtGame, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)
	data, err := qtx.GetSoughtGameBySeekerID(ctx, pgtype.Text{String: userID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	sg, err := soughtGameFromBytes(data)
	if err != nil {
		return nil, err
	}

	// Don't delete correspondence seeks when seeker leaves - they should persist
	if sg.SeekRequest != nil && sg.SeekRequest.GameRequest != nil && sg.SeekRequest.GameRequest.GameMode == pb.GameMode_CORRESPONDENCE {
		log.Debug().Str("userID", userID).Msg("skipping-deletion-of-correspondence-seek-for-user")
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return nil, nil
	}

	if err := qtx.DeleteSoughtGameBySeekerID(ctx, pgtype.Text{String: userID, Valid: true}); err != nil {
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

	data, err := s.queries.WithTx(tx).GetSoughtGameByReceiverID(ctx, pgtype.Text{String: receiverID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	sg, err := soughtGameFromBytes(data)
	if err != nil {
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

// DeleteForSeekerConnID deletes the game by connection ID.
// Correspondence seeks are not deleted when seeker disconnects - they persist for the receiver to accept later.
func (s *DBStore) DeleteForSeekerConnID(ctx context.Context, connID string) (*entity.SoughtGame, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)
	data, err := qtx.GetSoughtGameBySeekerConnID(ctx, pgtype.Text{String: connID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	sg, err := soughtGameFromBytes(data)
	if err != nil {
		return nil, err
	}

	// Don't delete correspondence seeks when seeker disconnects - they should persist
	if sg.SeekRequest != nil && sg.SeekRequest.GameRequest != nil && sg.SeekRequest.GameRequest.GameMode == pb.GameMode_CORRESPONDENCE {
		log.Debug().Str("connID", connID).Msg("skipping-deletion-of-correspondence-seek")
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return nil, nil
	}

	if err := qtx.DeleteSoughtGameBySeekerConnID(ctx, pgtype.Text{String: connID, Valid: true}); err != nil {
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

	data, err := s.queries.WithTx(tx).GetSoughtGameByReceiverConnID(ctx, pgtype.Text{String: connID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	sg, err := soughtGameFromBytes(data)
	if err != nil {
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
// ListOpenSeeks is a read-only query: no explicit transaction needed.
func (s *DBStore) ListOpenSeeks(ctx context.Context, receiverID, tourneyID string) ([]*entity.SoughtGame, error) {
	var rows pgx.Rows
	var err error
	if tourneyID != "" {
		rows, err = s.dbPool.Query(ctx, `SELECT request FROM soughtgames WHERE receiver = $1 AND request->>'tournament_id' = $2`, receiverID, tourneyID)
	} else {
		rows, err = s.dbPool.Query(ctx, `SELECT request FROM soughtgames WHERE receiver = $1 OR receiver = ''`, receiverID)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return games, nil
}

// ListCorrespondenceSeeksForUser lists all correspondence match requests and open seeks for a user.
// Read-only query: no explicit transaction needed.
func (s *DBStore) ListCorrespondenceSeeksForUser(ctx context.Context, userID string) ([]*entity.SoughtGame, error) {
	// Get correspondence seeks where:
	// - user is the seeker (their own open seeks or match requests)
	// - user is the receiver (match requests sent to them)
	// - open seeks available to all (receiver is empty)
	rows, err := s.dbPool.Query(ctx, `SELECT request FROM soughtgames WHERE game_mode = 1 AND (seeker = $1 OR receiver = $1 OR receiver = '')`, userID)
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
	if err := rows.Err(); err != nil {
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

// CanCreateSeek returns true if the user can create a new seek/match request.
// For correspondence match requests, multiple can exist simultaneously.
// For all other types (real-time or open seeks), only one can exist at a time.
// Returns (canCreate, conflictType, error) where conflictType indicates what kind of conflict exists.
func (s *DBStore) CanCreateSeek(ctx context.Context, userID string, gameMode pb.GameMode, receiverID string) (bool, string, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return false, "", err
	}
	defer tx.Rollback(ctx)

	// Check if user has any existing seeks
	var count int
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM soughtgames WHERE seeker = $1`, userID).Scan(&count)
	if err != nil {
		return false, "", err
	}

	// If no existing seeks, always allow
	if count == 0 {
		if err := tx.Commit(ctx); err != nil {
			return false, "", err
		}
		return true, "", nil
	}

	// If this is a correspondence match request (receiver != '' AND game_mode = 1)
	isCorrespondenceMatch := gameMode == pb.GameMode_CORRESPONDENCE && receiverID != ""

	if !isCorrespondenceMatch {
		// For real-time or open seeks, don't allow if any existing seeks exist
		if err := tx.Commit(ctx); err != nil {
			return false, "", err
		}
		return false, "has_other_seek", nil
	}

	// For correspondence match requests, check what kind of conflicts exist
	var hasOpenSeek int
	var hasRealtimeSeek int
	err = tx.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE receiver = ''),
			COUNT(*) FILTER (WHERE game_mode IS NULL OR game_mode != 1)
		FROM soughtgames
		WHERE seeker = $1
	`, userID).Scan(&hasOpenSeek, &hasRealtimeSeek)
	if err != nil {
		return false, "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, "", err
	}

	// Determine conflict type
	if hasOpenSeek > 0 {
		return false, "has_open_seek", nil
	}
	if hasRealtimeSeek > 0 {
		return false, "has_realtime_seek", nil
	}

	// All existing seeks are correspondence matches, allow
	return true, "", nil
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
