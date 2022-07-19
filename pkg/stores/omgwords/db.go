package omgwords

import (
	"context"

	"github.com/domino14/liwords/pkg/entity"
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

func (s *DBStore) Create(ctx context.Context, g *entity.Game) error {
	// Create row in omgwords table
	// Create row in omgwords_requests table
	// Create row in omgwords_tournament_data table
	return nil
}

func (s *DBStore) Set(ctx context.Context, g *entity.Game) error {
	// Conditionally create row in omgwords_turns
	// Conditionally create row in omgwords_meta_events
	// Conditionally modify omgwords table
	return nil
}

func (s *DBStore) HandleGameEnded(ctx context.Context, g *entity.Game) error {
	// Create row in omgwords_stats
	return nil
}
