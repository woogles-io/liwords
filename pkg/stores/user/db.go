package user

import (
	"context"
	"errors"

	"github.com/domino14/liwords/pkg/auth"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/jinzhu/gorm"
	"github.com/lithammer/shortuuid"

	// postgres
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

// DBStore is a postgres-backed store for users.
type DBStore struct {
	db *gorm.DB
}

// NewDBStore creates a new DB store
func NewDBStore(dbURL string) (*DBStore, error) {
	db, err := gorm.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&entity.User{})

	return &DBStore{db: db}, nil
}

// Get gets a user by username.
func (s *DBStore) Get(ctx context.Context, username string) (*entity.User, error) {
	u := &entity.User{}
	if result := s.db.Where("username = ?", username).First(u); result.Error != nil {
		return nil, result.Error
	}
	return u, nil
}

// GetUUID gets user by UUID
func (s *DBStore) GetUUID(ctx context.Context, uuid string) (*entity.User, error) {
	u := &entity.User{}
	if result := s.db.Where("uuid = ?", uuid).First(u); result.Error != nil {
		return nil, result.Error
	}
	return u, nil
}

func (s *DBStore) New(ctx context.Context, user *entity.User) error {
	user.UUID = shortuuid.New()
	result := s.db.Create(user)
	return result.Error
}

// CheckPassword checks the password. If the password matches, return the User.
// If not, return an error.
func (s *DBStore) CheckPassword(ctx context.Context, username string, password string) (*entity.User, error) {
	u := &entity.User{}
	if result := s.db.Where("username = ?", username).First(u); result.Error != nil {
		return nil, result.Error
	}
	matches, err := auth.ComparePassword(password, u.Password)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, errors.New("password does not match")
	}
	return u, nil
}
