package user

import (
	"context"
	"encoding/json"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lithammer/shortuuid"

	"github.com/domino14/liwords/pkg/entity"
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

// A user profile is in a one-to-one relationship with a user. It is the
// profile that should have all the extra data we don't want to / shouldn't stick
// in the user model.
type profile struct {
	gorm.Model
	// `profile` belongs to `user`, `UserID` is the foreign key.
	UserID uint
	User   user

	FirstName string `gorm:"type:varchar(32)"`
	LastName  string `gorm:"type:varchar(64)"`

	CountryCode string `gorm:"type:varchar(3)"`
	// Title is some sort of acronym/shorthand for a title. Like GM, EX, SM, UK-GM (UK Grandmaster?)
	Title string `gorm:"type:varchar(8)"`
	// There will be no avatar URL; a user's avatar will be located at a fixed
	// URL based on the user ID.

	// About is profile notes.
	About string `gorm:"type:varchar(2048)"`
	// It's ok to have a few JSON bags here. Postgres makes these easy and fast.
	// XXX: Come up with a model for friend list.
	Ratings postgres.Jsonb // A complex dictionary of ratings with variants/lexica/etc.
	Stats   postgres.Jsonb // Profile stats such as average score per variant, bingos, a lot of other simple stats.
	// More complex stats might be in a separate place.
}

// NewDBStore creates a new DB store
func NewDBStore(dbURL string) (*DBStore, error) {
	db, err := gorm.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&user{}, &profile{})
	// Can't get GORM to auto create these foreign keys, so do it myself /shrug
	db.Model(&profile{}).AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT")

	return &DBStore{db: db}, nil
}

// Get gets a user by username.
func (s *DBStore) Get(ctx context.Context, username string) (*entity.User, error) {
	u := &user{}
	p := &profile{}
	if result := s.db.Where("username = ?", username).First(u); result.Error != nil {
		return nil, result.Error
	}
	if result := s.db.Model(u).Related(p); result.Error != nil {
		return nil, result.Error
	}
	profile, err := dbProfileToProfile(p)
	if err != nil {
		return nil, err
	}
	entu := &entity.User{
		ID:        u.ID,
		Username:  u.Username,
		UUID:      u.UUID,
		Email:     u.Email,
		Password:  u.Password,
		Anonymous: false,
		Profile:   profile,
	}

	return entu, nil
}

func dbProfileToProfile(p *profile) (*entity.Profile, error) {
	var rdata entity.Ratings
	err := json.Unmarshal(p.Ratings.RawMessage, &rdata)
	if err != nil {
		return nil, err
	}
	return &entity.Profile{
		FirstName:   p.FirstName,
		LastName:    p.LastName,
		CountryCode: p.CountryCode,
		Title:       p.Title,
		About:       p.About,
		Ratings:     rdata,
	}, nil
}

// GetByUUID gets user by UUID
func (s *DBStore) GetByUUID(ctx context.Context, uuid string) (*entity.User, error) {
	u := &user{}
	p := &profile{}
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
		if result := s.db.Model(u).Related(p); result.Error != nil {
			return nil, result.Error
		}
		profile, err := dbProfileToProfile(p)
		if err != nil {
			return nil, err
		}

		entu = &entity.User{
			ID:       u.ID,
			Username: u.Username,
			UUID:     u.UUID,
			Email:    u.Email,
			Password: u.Password,
			Profile:  profile,
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
	if result.Error != nil {
		return result.Error
	}
	// Create profile

	rdata := entity.Ratings{}
	bytes, err := json.Marshal(rdata)
	if err != nil {
		return err
	}

	dbp := &profile{
		User:    *dbu,
		Ratings: postgres.Jsonb{RawMessage: bytes},
	}
	result = s.db.Create(dbp)
	return result.Error
}

// SetRating sets the specific rating for the given variant.
func (s *DBStore) SetRating(ctx context.Context, uuid string, variant entity.VariantKey,
	rating entity.SingleRating) error {
	u := &user{}
	p := &profile{}

	if result := s.db.Where("uuid = ?", uuid).First(u); result.Error != nil {
		return result.Error
	}
	if result := s.db.Model(u).Related(p); result.Error != nil {
		return result.Error
	}

	var existingRatings entity.Ratings
	err := json.Unmarshal(p.Ratings.RawMessage, &existingRatings)
	if err != nil {
		return nil
	}

	if existingRatings.Data == nil {
		existingRatings.Data = make(map[entity.VariantKey]entity.SingleRating)
	}
	existingRatings.Data[variant] = rating

	bytes, err := json.Marshal(existingRatings)
	if err != nil {
		return err
	}

	return s.db.Model(p).Update("ratings", postgres.Jsonb{RawMessage: bytes}).Error
}

// CheckPassword checks the password. If the password matches, return the User.
// If not, return an error.
// func (s *DBStore) CheckPassword(ctx context.Context, username string, password string) (*entity.User, error) {
// 	dbu := &user{}

// 	u := &entity.User{}
// 	if result := s.db.Where("username = ?", username).First(dbu); result.Error != nil {
// 		return nil, result.Error
// 	}
// 	matches, err := auth.ComparePassword(password, dbu.Password)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if !matches {
// 		return nil, errors.New("password does not match")
// 	}

// 	u.Username = dbu.Username
// 	u.UUID = dbu.UUID
// 	u.Email = dbu.Email
// 	u.Password = dbu.Password

// 	return u, nil
// }
