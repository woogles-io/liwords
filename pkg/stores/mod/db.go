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
