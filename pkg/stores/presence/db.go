package presence

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/domino14/liwords/pkg/stores/models"
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

// SetPresence sets a presence and returns oldChannels, newChannels
func (s *DBStore) SetPresence(ctx context.Context, uuid, username string, anon bool, channel string, connID string) ([]string, []string, error) {
	// xxx: username is ignored here. remove from interface.
	// xxx: anon is ignored. the uuid has `anon-` prefixed if user is anon.
	tx, err := s.dbPool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)

	rowsAffected, err := qtx.AddConnection(ctx, models.AddConnectionParams{
		UserID: uuid, ChannelName: channel, ConnectionID: connID,
	})
	if err != nil {
		return nil, nil, err
	}

	beforeChannels, err := qtx.SelectUniqueChannelsNotMatching(ctx, models.SelectUniqueChannelsNotMatchingParams{
		UserID:       uuid,
		ChannelName:  channel,
		ConnectionID: connID,
	})
	if err != nil {
		return nil, nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, nil, err
	}
	// Back-calculate "after" channel based on rowsAffected
	afterChannels := beforeChannels
	if rowsAffected == 1 {
		afterChannels = append(afterChannels, channel)
	}
	return beforeChannels, afterChannels, nil
}

// ClearPresence clears a presence and returns oldChannels, newChannels, removedChannels
func (s *DBStore) ClearPresence(ctx context.Context, uuid, username string, anon bool, connID string) ([]string, []string, []string, error) {
	tx, err := s.dbPool.Begin(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)

	rowsAffected, err := qtx.DeleteConnection(ctx, connID)
	if err != nil {
		return nil, nil, nil, err
	}

	beforeChannels, err := qtx.SelectUniqueChannelsNotMatching(ctx, models.SelectUniqueChannelsNotMatchingParams{
		UserID:       uuid,
		ChannelName:  channel,
		ConnectionID: connID,
	})
	if err != nil {
		return nil, nil, nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, nil, err
	}
	// Back-calculate "after" channel based on rowsAffected
	afterChannels := beforeChannels
	if rowsAffected == 1 {
		afterChannels = append(afterChannels, channel)
	}
	return beforeChannels, afterChannels, nil
}
