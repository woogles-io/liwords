package soughtgame

import (
	"context"
	"encoding/json"

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

// New stores the seek. A real-time match request is refused with
// entity.ErrMatchRequestOpen if its seeker already has one open to the same
// receiver, or if another of their match requests is being made, or one of
// their seeks accepted, at that moment.
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

	params := models.InsertSoughtGameParams{
		Uuid:           pgtype.Text{String: id, Valid: true},
		Seeker:         pgtype.Text{String: seeker, Valid: true},
		SeekerConnID:   pgtype.Text{String: seekerConnID, Valid: true},
		Receiver:       pgtype.Text{String: receiver, Valid: true},
		ReceiverConnID: pgtype.Text{String: receiverConnID, Valid: true},
		Request:        request,
		GameMode:       gameMode,
	}
	realtime := !gameMode.Valid || pb.GameMode(gameMode.Int32) == pb.GameMode_REAL_TIME
	if !realtime || receiver == "" {
		return s.queries.InsertSoughtGame(ctx, params)
	}

	// One open real-time match request per seeker and receiver: each one
	// reaches the receiver with a sound and a row in their lobby, so a
	// second one adds nothing but noise.
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)

	// Checked under the seeker's lock: each socket message is handled in
	// its own goroutine, so a burst of match requests would otherwise all
	// be checked before any was stored. Not waited for, since each waiter
	// would hold a pool connection and a burst would starve every other
	// request; a match request that finds it taken is refused instead.
	locked, err := qtx.TryLockSoughtGameSeeker(ctx, seeker)
	if err != nil {
		return err
	}
	if !locked {
		return entity.ErrMatchRequestOpen
	}
	open, err := qtx.RealtimeMatchRequestExists(ctx, models.RealtimeMatchRequestExistsParams{
		SeekerID:   pgtype.Text{String: seeker, Valid: true},
		ReceiverID: pgtype.Text{String: receiver, Valid: true},
	})
	if err != nil {
		return err
	}
	if open {
		return entity.ErrMatchRequestOpen
	}
	if err := qtx.InsertSoughtGame(ctx, params); err != nil {
		return err
	}
	return tx.Commit(ctx)
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

// DeleteForUser deletes the user's real-time seeks and match requests, and
// returns them. Correspondence seeks are not deleted when user leaves - they
// persist for the receiver to accept later.
func (s *DBStore) DeleteForUser(ctx context.Context, userID string) ([]*entity.SoughtGame, error) {
	rows, err := s.queries.DeleteRealtimeSoughtGamesBySeekerID(ctx, pgtype.Text{String: userID, Valid: true})
	if err != nil {
		return nil, err
	}
	return soughtGamesFromRows(rows)
}

// DeleteForSeekerConnID deletes the real-time seeks and match requests made
// from a connection, and returns them. Correspondence seeks are not deleted
// when seeker disconnects - they persist for the receiver to accept later.
func (s *DBStore) DeleteForSeekerConnID(ctx context.Context, connID string) ([]*entity.SoughtGame, error) {
	rows, err := s.queries.DeleteRealtimeSoughtGamesBySeekerConnID(ctx, pgtype.Text{String: connID, Valid: true})
	if err != nil {
		return nil, err
	}
	return soughtGamesFromRows(rows)
}

func soughtGamesFromRows(rows [][]byte) ([]*entity.SoughtGame, error) {
	sgs := make([]*entity.SoughtGame, 0, len(rows))
	for _, data := range rows {
		sg, err := soughtGameFromBytes(data)
		if err != nil {
			return nil, err
		}
		sgs = append(sgs, sg)
	}
	return sgs, nil
}

// Claim takes the seek with this ID for a game that is about to start, so
// that it can be accepted once only. For a real-time seek or match request it
// also takes the seeker's other real-time ones, since they are now playing;
// those come back as others. Claims on one seeker's seeks are serialized, so
// of two accepts of their seeks at the same moment, the second finds the seek
// gone and gets ErrSeekGone.
func (s *DBStore) Claim(ctx context.Context, id string) (claimed *entity.SoughtGame, others []*entity.SoughtGame, err error) {
	uuid := pgtype.Text{String: id, Valid: true}
	seeker, err := s.queries.GetSoughtGameSeekerByUUID(ctx, uuid)
	if err == pgx.ErrNoRows {
		return nil, nil, entity.ErrSeekGone
	} else if err != nil {
		return nil, nil, err
	}

	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)

	if err := qtx.LockSoughtGameSeeker(ctx, seeker.String); err != nil {
		return nil, nil, err
	}
	row, err := qtx.ClaimSoughtGameByUUID(ctx, uuid)
	if err == pgx.ErrNoRows {
		return nil, nil, entity.ErrSeekGone
	} else if err != nil {
		return nil, nil, err
	}
	claimed, err = soughtGameFromBytes(row.Request)
	if err != nil {
		return nil, nil, err
	}
	if !row.GameMode.Valid || pb.GameMode(row.GameMode.Int32) == pb.GameMode_REAL_TIME {
		rows, err := qtx.DeleteRealtimeSoughtGamesBySeekerID(ctx, seeker)
		if err != nil {
			return nil, nil, err
		}
		if others, err = soughtGamesFromRows(rows); err != nil {
			return nil, nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return claimed, others, nil
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
