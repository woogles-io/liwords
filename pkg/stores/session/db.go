package session

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lithammer/shortuuid"
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
		return nil, errors.New("session expired, log in again")
	}

	// Parse JSONB session data.
	var data sessionInfo
	err := json.Unmarshal(u.Data.RawMessage, &data)
	if err != nil {
		return nil, err
	}

	return &entity.Session{
		ID:       u.UUID,
		Username: data.Username,
		UserUUID: data.UserUUID,
	}, nil
}

// The data inside the session's Data object.
type sessionInfo struct {
	Username string `json:"username"`
	UserID   uint   `json:"id"`
	UserUUID string `json:"uuid"`
}

// New should be called when a user logs in. It'll create a new session.
func (s *DBStore) New(ctx context.Context, user *entity.User) (*entity.Session, error) {

	data := sessionInfo{user.Username, user.ID, user.UUID}

	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	sess := &dbSession{
		UUID:      shortuuid.New(),
		ExpiresAt: time.Now().Add(entity.SessionExpiration),
		Data:      postgres.Jsonb{RawMessage: bytes},
	}

	user.UUID = shortuuid.New()
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
