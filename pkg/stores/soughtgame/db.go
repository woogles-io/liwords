package soughtgame

import (
	"context"
	"encoding/json"
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
	gameMode := pgtype.Int4{Valid: false}
	if game.SeekRequest != nil && game.SeekRequest.GameRequest != nil {
		gameMode = pgtype.Int4{Int32: int32(game.SeekRequest.GameRequest.GameMode), Valid: true}
	}

	// Matches pgx's default jsonb encoding of a Go value (plain encoding/json,
	// using the struct's json tags), which is what this column was written
	// with previously via direct tx.Exec.
	request, err := json.Marshal(game.SeekRequest)
	if err != nil {
		return err
	}

	return s.queries.InsertSoughtGame(ctx, models.InsertSoughtGameParams{
		Uuid:           pgtype.Text{String: id, Valid: true},
		Seeker:         pgtype.Text{String: seeker, Valid: true},
		SeekerConnID:   pgtype.Text{String: seekerConnID, Valid: true},
		Receiver:       pgtype.Text{String: receiver, Valid: true},
		ReceiverConnID: pgtype.Text{String: receiverConnID, Valid: true},
		Request:        request,
		GameMode:       gameMode,
	})
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
	// Delete real-time seeks older than 2 hours (game_mode IS NULL or game_mode = 0)
	rowsAffected, err := s.queries.ExpireOldRealtimeSeeks(ctx)
	if err != nil {
		return err
	}
	if rowsAffected > 0 {
		log.Info().Int64("rows-affected", rowsAffected).Msg("expire-old-realtime-seeks")
	}

	// Delete correspondence seeks older than 60 hours (game_mode = 1)
	rowsAffected, err = s.queries.ExpireOldCorrespondenceSeeks(ctx)
	if err != nil {
		return err
	}
	if rowsAffected > 0 {
		log.Info().Int64("rows-affected", rowsAffected).Msg("expire-old-correspondence-seeks")
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

	qtx := s.queries.WithTx(tx)
	data, err := qtx.GetSoughtGameByReceiverID(ctx, pgtype.Text{String: receiverID, Valid: true})
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
	receiverState, err := json.Marshal(int32(pb.SeekState_ABSENT))
	if err != nil {
		return nil, err
	}
	rowsAffected, err := qtx.UpdateSoughtGameReceiverAbsentByReceiverID(ctx, models.UpdateSoughtGameReceiverAbsentByReceiverIDParams{
		ReceiverState: receiverState,
		ReceiverID:    pgtype.Text{String: receiverID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	if rowsAffected != 1 {
		return nil, fmt.Errorf("failed to update receiver status: %s (%d rows affected)", receiverID, rowsAffected)
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

	qtx := s.queries.WithTx(tx)
	data, err := qtx.GetSoughtGameByReceiverConnID(ctx, pgtype.Text{String: connID, Valid: true})
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
	receiverState, err := json.Marshal(int32(pb.SeekState_ABSENT))
	if err != nil {
		return nil, err
	}
	rowsAffected, err := qtx.UpdateSoughtGameReceiverAbsentByReceiverConnID(ctx, models.UpdateSoughtGameReceiverAbsentByReceiverConnIDParams{
		ReceiverState:  receiverState,
		ReceiverConnID: pgtype.Text{String: connID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	if rowsAffected != 1 {
		return nil, fmt.Errorf("failed to update receiver status: %s (%d rows affected)", connID, rowsAffected)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return sg, nil
}

// ListOpenSeeks lists all open seek requests for receiverID, in tourneyID (optional)
// ListOpenSeeks is a read-only query: no explicit transaction needed.
func (s *DBStore) ListOpenSeeks(ctx context.Context, receiverID, tourneyID string) ([]*entity.SoughtGame, error) {
	var reqs [][]byte
	var err error
	if tourneyID != "" {
		reqs, err = s.queries.ListOpenSeeksByTourney(ctx, models.ListOpenSeeksByTourneyParams{
			ReceiverID:   pgtype.Text{String: receiverID, Valid: true},
			TournamentID: tourneyID,
		})
	} else {
		reqs, err = s.queries.ListOpenSeeksAll(ctx, pgtype.Text{String: receiverID, Valid: true})
	}
	if err != nil {
		return nil, err
	}

	games := []*entity.SoughtGame{}
	for _, data := range reqs {
		sg, err := soughtGameFromBytes(data)
		if err != nil {
			return nil, err
		}
		games = append(games, sg)
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
	reqs, err := s.queries.ListCorrespondenceSeeksForUser(ctx, pgtype.Text{String: userID, Valid: true})
	if err != nil {
		return nil, err
	}

	games := []*entity.SoughtGame{}
	for _, data := range reqs {
		sg, err := soughtGameFromBytes(data)
		if err != nil {
			return nil, err
		}
		games = append(games, sg)
	}
	return games, nil
}

// ExistsForUser returns true if the user already has an outstanding seek request.
func (s *DBStore) ExistsForUser(ctx context.Context, userID string) (bool, error) {
	return s.queries.ExistsSeekForUser(ctx, pgtype.Text{String: userID, Valid: true})
}

// CanCreateSeek returns true if the user can create a new seek/match request.
// For correspondence match requests, multiple can exist simultaneously.
// For all other types (real-time or open seeks), only one can exist at a time.
// Returns (canCreate, conflictType, error) where conflictType indicates what kind of conflict exists.
func (s *DBStore) CanCreateSeek(ctx context.Context, userID string, gameMode pb.GameMode, receiverID string) (bool, string, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.ReadOnlyTxOptions)
	if err != nil {
		return false, "", err
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)

	// Check if user has any existing seeks
	count, err := qtx.CountSeeksForUser(ctx, pgtype.Text{String: userID, Valid: true})
	if err != nil {
		return false, "", err
	}

	// If no existing seeks, always allow
	if count == 0 {
		return true, "", nil
	}

	// If this is a correspondence match request (receiver != '' AND game_mode = 1)
	isCorrespondenceMatch := gameMode == pb.GameMode_CORRESPONDENCE && receiverID != ""

	if !isCorrespondenceMatch {
		// For real-time or open seeks, don't allow if any existing seeks exist
		return false, "has_other_seek", nil
	}

	// For correspondence match requests, check what kind of conflicts exist
	conflicts, err := qtx.CountSeekConflictsForCorrespondence(ctx, pgtype.Text{String: userID, Valid: true})
	if err != nil {
		return false, "", err
	}

	// Determine conflict type
	if conflicts.HasOpenSeek > 0 {
		return false, "has_open_seek", nil
	}
	if conflicts.HasRealtimeSeek > 0 {
		return false, "has_realtime_seek", nil
	}

	// All existing seeks are correspondence matches, allow
	return true, "", nil
}

// UserMatchedBy returns true if there is an open seek request from matcher for user
func (s *DBStore) UserMatchedBy(ctx context.Context, userID, matcher string) (bool, error) {
	exists, err := s.queries.SeekExistsFromMatcher(ctx, models.SeekExistsFromMatcherParams{
		UserID:  pgtype.Text{String: userID, Valid: true},
		Matcher: pgtype.Text{String: matcher, Valid: true},
	})
	if err != nil {
		return false, err
	}

	return exists, nil
}
