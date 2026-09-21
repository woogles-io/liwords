package mod

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woogles-io/liwords/pkg/stores/models"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
	"google.golang.org/protobuf/types/known/timestamppb"
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

// RecordAutomodVerdict records automod's judgement of one player in one game
// and reports whether this call is the one that recorded it.
//
// False means automod has already judged this pairing, and its effects -- which
// add to or subtract from a notoriety score -- must not be applied again.
func (s *DBStore) RecordAutomodVerdict(ctx context.Context, playerID string, gameID string, verdict int) (bool, error) {
	n, err := s.queries.RecordAutomodVerdict(ctx, models.RecordAutomodVerdictParams{
		PlayerID: playerID,
		GameID:   gameID,
		Verdict:  int32(verdict),
	})
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func (s *DBStore) AddNotoriousGame(ctx context.Context, playerID string, gameID string, gameType int, time int64) error {
	return s.queries.AddNotoriousGame(ctx, models.AddNotoriousGameParams{
		GameID:    pgtype.Text{String: gameID, Valid: true},
		PlayerID:  pgtype.Text{String: playerID, Valid: true},
		Type:      pgtype.Int4{Int32: int32(gameType), Valid: true},
		Timestamp: pgtype.Int8{Int64: time, Valid: true},
	})
}

func ConvertUnixToTimestampPb(unixTime int64) *timestamppb.Timestamp {
	t := time.Unix(unixTime, 0)
	return timestamppb.New(t)
}

func (s *DBStore) GetNotoriousGames(ctx context.Context, playerID string, limit int) ([]*ms.NotoriousGame, error) {
	rows, err := s.queries.GetNotoriousGames(ctx, models.GetNotoriousGamesParams{
		PlayerID: pgtype.Text{Valid: true, String: playerID},
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, err
	}

	games := []*ms.NotoriousGame{}

	for i := range rows {
		games = append(games, &ms.NotoriousGame{
			Id:        rows[i].GameID.String,
			Type:      ms.NotoriousGameType(rows[i].Type.Int32),
			CreatedAt: ConvertUnixToTimestampPb(rows[i].Timestamp.Int64),
		})
	}

	return games, nil
}

func (s *DBStore) DeleteNotoriousGames(ctx context.Context, playerID string) error {
	return s.queries.DeleteNotoriousGamesForPlayer(ctx, pgtype.Text{String: playerID, Valid: true})
}
