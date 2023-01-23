package comments

import (
	"context"
	"database/sql"

	"github.com/domino14/liwords/pkg/stores/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type DBStore struct {
	dbPool  *pgxpool.Pool
	queries *models.Queries
}

func NewDBStore(p *pgxpool.Pool) (*DBStore, error) {
	queries := models.New(p)
	return &DBStore{dbPool: p, queries: queries}, nil
}

func (s *DBStore) Disconnect() {
	s.dbPool.Close()
}

func (s *DBStore) AddComment(ctx context.Context, gameID string, authorID int,
	eventNumber int, comment string) (string, error) {

	id, err := s.queries.AddComment(ctx, models.AddCommentParams{
		// how do i rename this default field uuid?
		Uuid:        sql.NullString{gameID, true},
		AuthorID:    int32(authorID),
		EventNumber: int32(eventNumber),
		Comment:     comment,
	})
	if err != nil {
		return "", err
	}

	return id.String(), nil
}

func (s *DBStore) GetComments(ctx context.Context, gameID string) ([]models.GetCommentsForGameRow, error) {
	return s.queries.GetCommentsForGame(ctx, sql.NullString{gameID, true})
}

func (s *DBStore) UpdateComment(ctx context.Context, commentID string, comment string) error {
	uuid, err := uuid.Parse(commentID)
	if err != nil {
		return err
	}
	return s.queries.UpdateComment(ctx, models.UpdateCommentParams{
		Comment: comment,
		ID:      uuid,
	})
}

func (s *DBStore) DeleteComment(ctx context.Context, commentID string) error {
	uuid, err := uuid.Parse(commentID)
	if err != nil {
		return err
	}
	return s.queries.DeleteComment(ctx, uuid)
}
