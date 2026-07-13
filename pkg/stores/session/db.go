package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lithammer/shortuuid/v4"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// The data inside the session's Data object.
type sessionInfo struct {
	Username  string `json:"username"`
	UserUUID  string `json:"uuid"`
	CSRFToken string `json:"csrf_token"`
}

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

// Get gets a session by session ID
func (s *DBStore) Get(ctx context.Context, sessionID string) (*entity.Session, error) {
	row, err := s.queries.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	var data sessionInfo
	if len(row.Data) > 0 {
		if err := json.Unmarshal(row.Data, &data); err != nil {
			return nil, err
		}
	}

	return &entity.Session{
		ID:        sessionID,
		Username:  data.Username,
		UserUUID:  data.UserUUID,
		Expiry:    row.ExpiresAt.Time,
		CSRFToken: data.CSRFToken,
	}, nil
}

// New should be called when a user logs in. It'll create a new session.
func (s *DBStore) New(ctx context.Context, user *entity.User) (*entity.Session, error) {
	newSessionID := shortuuid.New()
	expiresAt := time.Now().Add(entity.SessionExpiration)
	data, err := json.Marshal(sessionInfo{Username: user.Username, UserUUID: user.UUID})
	if err != nil {
		return nil, err
	}

	err = s.queries.CreateSession(ctx, models.CreateSessionParams{
		Uuid:      newSessionID,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		Data:      data,
	})
	if err != nil {
		return nil, err
	}
	return &entity.Session{
		ID:       newSessionID,
		Username: user.Username,
		UserUUID: user.UUID,
	}, nil
}

// Delete deletes the session with the given ID, essentially logging the user out.
func (s *DBStore) Delete(ctx context.Context, sess *entity.Session) error {
	return s.queries.DeleteSession(ctx, sess.ID)
}

// ExtendExpiry extends the expiry of the given cookie.
func (s *DBStore) ExtendExpiry(ctx context.Context, sess *entity.Session) error {
	rowsAffected, err := s.queries.ExtendSessionExpiry(ctx, models.ExtendSessionExpiryParams{
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(entity.SessionExpiration), Valid: true},
		Uuid:      sess.ID,
	})
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return fmt.Errorf("could not extend expiry of session %s", sess.ID)
	}
	return nil
}

func (s *DBStore) SetCSRFToken(ctx context.Context, sess *entity.Session, csrfToken string) error {
	rowsAffected, err := s.queries.SetSessionCSRFToken(ctx, models.SetSessionCSRFTokenParams{
		CsrfToken: csrfToken,
		Uuid:      sess.ID,
	})
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return fmt.Errorf("could not update CSRF token for session %s", sess.ID)
	}
	return nil
}
