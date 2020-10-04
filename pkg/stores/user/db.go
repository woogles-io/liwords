package user

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"sort"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/entity"
)

// DBStore is a postgres-backed store for users.
type DBStore struct {
	db *gorm.DB
}

// User should be a minimal object. All information such as user profile,
// awards, ratings, records, etc should be in a profile object that
// joins 1-1 with this User object.
// User is exported as a Game has Foreign Keys to it.
type User struct {
	gorm.Model

	UUID     string `gorm:"type:varchar(24);index"`
	Username string `gorm:"type:varchar(32)"`
	Email    string `gorm:"type:varchar(100)"`
	// Password will be hashed.
	Password    string `gorm:"type:varchar(128)"`
	InternalBot bool   `gorm:"default:false;index"`
}

// A user profile is in a one-to-one relationship with a user. It is the
// profile that should have all the extra data we don't want to / shouldn't stick
// in the user model.
type profile struct {
	gorm.Model
	// `profile` belongs to `user`, `UserID` is the foreign key.
	UserID uint
	User   User

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

type following struct {
	// Follower follows user; pretty straightforward.
	UserID uint
	User   User

	FollowerID uint
	Follower   User
}

// NewDBStore creates a new DB store
func NewDBStore(dbURL string) (*DBStore, error) {
	db, err := gorm.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&User{}, &profile{}, &following{})
	db.Model(&User{}).
		AddUniqueIndex("username_idx", "lower(username)").
		AddUniqueIndex("email_idx", "lower(email)")

	// Can't get GORM to auto create these foreign keys, so do it myself /shrug
	db.Model(&profile{}).AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT")
	db.Model(&following{}).
		AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT").
		AddForeignKey("follower_id", "users(id)", "RESTRICT", "RESTRICT").
		AddUniqueIndex("user_follower_idx", "user_id", "follower_id")

	return &DBStore{db: db}, nil
}

// Get gets a user by username.
func (s *DBStore) Get(ctx context.Context, username string) (*entity.User, error) {
	u := &User{}
	p := &profile{}
	if result := s.db.Where("lower(username) = ?", strings.ToLower(username)).First(u); result.Error != nil {
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
		IsBot:     u.InternalBot,
		Anonymous: false,
		Profile:   profile,
	}

	return entu, nil
}

// GetByEmail gets the user by email. It does not try to get the profile.
// We don't get the profile here because GetByEmail is only used for things
// like password resets and there is no need.
func (s *DBStore) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	u := &User{}
	if result := s.db.Where("lower(email) = ?", strings.ToLower(email)).First(u); result.Error != nil {
		return nil, result.Error
	}

	entu := &entity.User{
		ID:        u.ID,
		Username:  u.Username,
		UUID:      u.UUID,
		Email:     u.Email,
		Password:  u.Password,
		Anonymous: false,
		IsBot:     u.InternalBot,
	}

	return entu, nil
}

func dbProfileToProfile(p *profile) (*entity.Profile, error) {
	var rdata entity.Ratings
	err := json.Unmarshal(p.Ratings.RawMessage, &rdata)
	if err != nil {
		log.Err(err).Msg("profile had bad rating json, zeroing")
		rdata = entity.Ratings{Data: map[entity.VariantKey]entity.SingleRating{}}
	}
	var sdata entity.ProfileStats
	err = json.Unmarshal(p.Stats.RawMessage, &sdata)
	if err != nil {
		log.Err(err).Msg("profile had bad stats json, zeroing")
		sdata = entity.ProfileStats{Data: map[entity.VariantKey]*entity.Stats{}}
	}
	return &entity.Profile{
		FirstName:   p.FirstName,
		LastName:    p.LastName,
		CountryCode: p.CountryCode,
		Title:       p.Title,
		About:       p.About,
		Ratings:     rdata,
		Stats:       sdata,
	}, nil
}

// GetByUUID gets user by UUID
func (s *DBStore) GetByUUID(ctx context.Context, uuid string) (*entity.User, error) {
	u := &User{}
	p := &profile{}
	var entu *entity.User
	if uuid == "" {
		return nil, errors.New("blank-uuid")
	}

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
			IsBot:    u.InternalBot,
			Profile:  profile,
		}
	}

	return entu, nil
}

// New creates a new user in the DB.
func (s *DBStore) New(ctx context.Context, u *entity.User) error {
	if u.UUID == "" {
		u.UUID = shortuuid.New()
	}
	dbu := &User{
		UUID:        u.UUID,
		Username:    u.Username,
		Email:       u.Email,
		Password:    u.Password,
		InternalBot: u.IsBot,
	}
	result := s.db.Create(dbu)
	if result.Error != nil {
		return result.Error
	}

	// Create profile
	rdata := entity.Ratings{}
	ratbytes, err := json.Marshal(rdata)
	if err != nil {
		return err
	}

	sdata := entity.ProfileStats{}
	statbytes, err := json.Marshal(sdata)
	if err != nil {
		return err
	}
	dbp := &profile{
		User:    *dbu,
		Ratings: postgres.Jsonb{RawMessage: ratbytes},
		Stats:   postgres.Jsonb{RawMessage: statbytes},
	}
	result = s.db.Create(dbp)
	return result.Error
}

// SetPassword sets the password for the user. The password is already hashed.
func (s *DBStore) SetPassword(ctx context.Context, uuid string, hashpass string) error {
	u := &User{}
	if result := s.db.Where("uuid = ?", uuid).First(u); result.Error != nil {
		return result.Error
	}
	return s.db.Model(u).Update("password", hashpass).Error
}

// SetRating sets the specific rating for the given variant.
func (s *DBStore) SetRating(ctx context.Context, uuid string, variant entity.VariantKey,
	rating entity.SingleRating) error {
	u := &User{}
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
		log.Err(err).Msg("existing ratings missing; initializing...")
		existingRatings = entity.Ratings{Data: map[entity.VariantKey]entity.SingleRating{}}
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

func (s *DBStore) SetStats(ctx context.Context, uuid string, variant entity.VariantKey,
	stats *entity.Stats) error {

	u := &User{}
	p := &profile{}

	if result := s.db.Where("uuid = ?", uuid).First(u); result.Error != nil {
		return result.Error
	}
	if result := s.db.Model(u).Related(p); result.Error != nil {
		return result.Error
	}

	var existingProfileStats entity.ProfileStats
	err := json.Unmarshal(p.Stats.RawMessage, &existingProfileStats)
	if err != nil {
		log.Err(err).Msg("existing stats missing; initializing...")
		existingProfileStats = entity.ProfileStats{Data: map[entity.VariantKey]*entity.Stats{}}
	}
	if existingProfileStats.Data == nil {
		existingProfileStats.Data = make(map[entity.VariantKey]*entity.Stats)
	}
	existingProfileStats.Data[variant] = stats
	bytes, err := json.Marshal(existingProfileStats)
	if err != nil {
		return err
	}
	return s.db.Model(p).Update("stats", postgres.Jsonb{RawMessage: bytes}).Error
}

func (s *DBStore) GetRandomBot(ctx context.Context) (*entity.User, error) {

	var users []*User
	p := &profile{}

	if result := s.db.Where("internal_bot = ?", true).Find(&users); result.Error != nil {
		return nil, result.Error
	}
	idx := rand.Intn(len(users))
	u := users[idx]

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
		IsBot:     u.InternalBot,
		Profile:   profile,
	}

	return entu, nil
}

// AddFollower creates a follower -> target follow.
func (s *DBStore) AddFollower(ctx context.Context, targetUser, follower uint) error {
	dbf := &following{UserID: targetUser, FollowerID: follower}
	result := s.db.Create(dbf)
	return result.Error
}

// RemoveFollower removes a follower -> target follow.
func (s *DBStore) RemoveFollower(ctx context.Context, targetUser, follower uint) error {
	return s.db.Where("user_id = ? AND follower_id = ?", targetUser, follower).Delete(&following{}).Error
}

// GetFollows gets all the users that the passed-in user DB ID is following.
func (s *DBStore) GetFollows(ctx context.Context, uid uint) ([]*entity.User, error) {
	type followed struct {
		Username string
		Uuid     string
	}

	var users []followed

	if result := s.db.Table("followings").Select("u0.username, u0.uuid").
		Joins("JOIN users as u0 ON u0.id = user_id").
		Where("follower_id = ?", uid).Scan(&users); result.Error != nil {

		return nil, result.Error
	}
	log.Debug().Int("num-followed", len(users)).Msg("found-followed")
	entUsers := make([]*entity.User, len(users))
	for idx, u := range users {
		entUsers[idx] = &entity.User{UUID: u.Uuid, Username: u.Username}
	}
	return entUsers, nil
}

// Username gets the username from the uuid. If not found, return a deterministic username,
// and return true for isAnonymous.
func (s *DBStore) Username(ctx context.Context, uuid string) (string, bool, error) {
	type u struct {
		Username string
	}
	var user u

	if result := s.db.Table("users").Select("username").
		Where("uuid = ?", uuid).Scan(&user); result.Error != nil {

		if gorm.IsRecordNotFoundError(result.Error) {
			return entity.DeterministicUsername(uuid), true, nil
		}
		return "", false, result.Error
	}
	return user.Username, false, nil
}

func (s *DBStore) UsernamesByPrefix(ctx context.Context, prefix string) ([]string, error) {

	type u struct {
		Username string
	}

	var us []u
	if result := s.db.Table("users").Select("username").
		Where("lower(username) like ? AND internal_bot = ?",
			strings.ToLower(prefix)+"%", false).
		Limit(20).
		Scan(&us); result.Error != nil {
		return nil, result.Error
	}
	log.Debug().Str("prefix", prefix).Int("byprefix", len(us)).Msg("found-matches")

	usernames := make([]string, len(us))
	for idx, u := range us {
		usernames[idx] = u.Username
	}
	sort.Strings(usernames)

	return usernames, nil
}

func (s *DBStore) Disconnect() {
	s.db.Close()
}
