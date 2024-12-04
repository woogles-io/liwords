package session

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lithammer/shortuuid/v4"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
)

// The data inside the session's Data object.
type sessionInfo struct {
	Username  string `json:"username"`
	UserUUID  string `json:"uuid"`
	CSRFToken string `json:"csrf_token"`
}

type DBStore struct {
	dbPool *pgxpool.Pool
}

func NewDBStore(p *pgxpool.Pool) (*DBStore, error) {
	return &DBStore{dbPool: p}, nil
}

func (s *DBStore) Disconnect() {
	s.dbPool.Close()
}

// Get gets a session by session ID
func (s *DBStore) Get(ctx context.Context, sessionID string) (*entity.Session, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var expiry time.Time
	var data sessionInfo
	err = tx.QueryRow(ctx, `SELECT expires_at, data FROM db_sessions WHERE uuid = $1`, sessionID).Scan(&expiry, &data)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &entity.Session{
		ID:        sessionID,
		Username:  data.Username,
		UserUUID:  data.UserUUID,
		Expiry:    expiry,
		CSRFToken: data.CSRFToken,
	}, nil
}

// New should be called when a user logs in. It'll create a new session.
func (s *DBStore) New(ctx context.Context, user *entity.User) (*entity.Session, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	newSessionID := shortuuid.New()
	expiresAt := time.Now().Add(entity.SessionExpiration)
	data := sessionInfo{Username: user.Username, UserUUID: user.UUID}

	_, err = tx.Exec(ctx, `INSERT INTO db_sessions (uuid, expires_at, data) VALUES ($1, $2, $3)`, newSessionID, expiresAt, data)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
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
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM db_sessions WHERE uuid = $1`, sess.ID)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// ExtendExpiry extends the expiry of the given cookie.
func (s *DBStore) ExtendExpiry(ctx context.Context, sess *entity.Session) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx, `UPDATE db_sessions SET expires_at = $1 WHERE uuid = $2`, time.Now().Add(entity.SessionExpiration), sess.ID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("could not extend expiry of session %s", sess.ID)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *DBStore) SetCSRFToken(ctx context.Context, sess *entity.Session, csrfToken string) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Update the CSRFToken in the JSONB blob
	query := `
		UPDATE db_sessions
		SET data = jsonb_set(data, '{csrf_token}', to_jsonb($1::text))
		WHERE uuid = $2
	`
	result, err := tx.Exec(ctx, query, csrfToken, sess.ID)
	if err != nil {
		return err
	}

	if result.RowsAffected() != 1 {
		return fmt.Errorf("could not update CSRF token for session %s", sess.ID)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}
