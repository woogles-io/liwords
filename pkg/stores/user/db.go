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

// A user should be a minimal object. All information such as user profile,
// awards, ratings, records, etc should be in a profile object that
// joins 1-1 with this User object.
type user struct {
	gorm.Model

	UUID     string `gorm:"type:varchar(24);index"`
	Username string `gorm:"type:varchar(32);unique_index"`
	Email    string `gorm:"type:varchar(100);unique_index"`
	// Password will be hashed.
	Password string `gorm:"type:varchar(128)"`
}

// NewDBStore creates a new DB store
func NewDBStore(dbURL string) (*DBStore, error) {
	db, err := gorm.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&user{})

	return &DBStore{db: db}, nil
}

// Get gets a user by username.
func (s *DBStore) Get(ctx context.Context, username string) (*entity.User, error) {
	u := &user{}
	if result := s.db.Where("username = ?", username).First(u); result.Error != nil {
		return nil, result.Error
	}

	entu := &entity.User{
		Username:  u.Username,
		UUID:      u.UUID,
		Email:     u.Email,
		Password:  u.Password,
		Anonymous: false,
	}
	return entu, nil
}

// GetByUUID gets user by UUID
func (s *DBStore) GetByUUID(ctx context.Context, uuid string) (*entity.User, error) {
	u := &user{}
	var entu *entity.User
	if result := s.db.Where("uuid = ?", uuid).First(u); result.Error != nil {
		if gorm.IsRecordNotFoundError(result.Error) {
			entu = &entity.User{
				Username:  entity.DeterministicUsername(uuid),
				Anonymous: true,
				UUID:      uuid,
			}
		} else {
			return nil, result.Error
		}
	} else {
		entu = &entity.User{
			Username: u.Username,
			UUID:     u.UUID,
			Email:    u.Email,
			Password: u.Password,
		}
	}

	return entu, nil
}

func (s *DBStore) New(ctx context.Context, u *entity.User) error {
	u.UUID = shortuuid.New()

	dbu := &user{
		UUID:     u.UUID,
		Username: u.Username,
		Email:    u.Email,
		Password: u.Password,
	}

	result := s.db.Create(dbu)
	return result.Error
}

// CheckPassword checks the password. If the password matches, return the User.
// If not, return an error.
func (s *DBStore) CheckPassword(ctx context.Context, username string, password string) (*entity.User, error) {
	dbu := &user{}

	u := &entity.User{}
	if result := s.db.Where("username = ?", username).First(dbu); result.Error != nil {
		return nil, result.Error
	}
	matches, err := auth.ComparePassword(password, dbu.Password)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, errors.New("password does not match")
	}

	u.Username = dbu.Username
	u.UUID = dbu.UUID
	u.Email = dbu.Email
	u.Password = dbu.Password

	return u, nil
}
