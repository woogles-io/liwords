package session

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/entity"
)

type dbSession struct {
	UUID      string    `gorm:"type:varchar(24);primary_key"`
	ExpiresAt time.Time `gorm:"index"`
	Data      postgres.Jsonb
}

// DBStore is a postgres-backed store for user sessions.
type DBStore struct {
	db *gorm.DB
}

// NewDBStore creates a new DB store
func NewDBStore(dbURL string) (*DBStore, error) {
	db, err := gorm.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&dbSession{})

	return &DBStore{db: db}, nil
}

// Get gets a session by session ID
func (s *DBStore) Get(ctx context.Context, sessionID string) (*entity.Session, error) {
	u := &dbSession{}
	if result := s.db.Where("uuid = ?", sessionID).First(u); result.Error != nil {
		return nil, result.Error
	}
	if time.Now().After(u.ExpiresAt) {
		log.Debug().Interface("expires", u.ExpiresAt).Msg("expired?")
		return nil, errors.New("session expired, log in again")
	}

	// Parse JSONB session data.
	var data sessionInfo
	err := json.Unmarshal(u.Data.RawMessage, &data)
	if err != nil {
		return nil, err
	}

	return &entity.Session{
		ID:       u.UUID, // the session ID
		Username: data.Username,
		UserUUID: data.UserUUID,
		Expiry:   u.ExpiresAt,
	}, nil
}

// The data inside the session's Data object.
type sessionInfo struct {
	Username string `json:"username"`
	UserUUID string `json:"uuid"`
}

// New should be called when a user logs in. It'll create a new session.
func (s *DBStore) New(ctx context.Context, user *entity.User) (*entity.Session, error) {

	data := sessionInfo{user.Username, user.UUID}

	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	sess := &dbSession{
		UUID:      shortuuid.New(),
		ExpiresAt: time.Now().Add(entity.SessionExpiration),
		Data:      postgres.Jsonb{RawMessage: bytes},
	}

	result := s.db.Create(sess)
	if result.Error != nil {
		return nil, result.Error
	}
	return &entity.Session{
		ID:       sess.UUID,
		Username: user.Username,
		UserUUID: user.UUID,
	}, nil
}

// Delete deletes the session with the given ID, essentially logging the user out.
func (s *DBStore) Delete(ctx context.Context, sess *entity.Session) error {
	if sess.ID == "" {
		return errors.New("session has a blank ID, cannot be deleted")
	}
	// We want to delete from db_sessions, not delete from sessions
	return s.db.Delete(&dbSession{UUID: sess.ID}).Error
}

// ExtendExpiry extends the expiry of the given cookie.
func (s *DBStore) ExtendExpiry(ctx context.Context, sess *entity.Session) error {
	result := s.db.Table("db_sessions").Where("uuid = ?", sess.ID).Updates(
		map[string]interface{}{"expires_at": time.Now().Add(entity.SessionExpiration)})
	return result.Error
}
