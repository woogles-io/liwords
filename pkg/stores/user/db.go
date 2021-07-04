package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/entity"
	cpb "github.com/domino14/liwords/rpc/api/proto/config_service"
	pb "github.com/domino14/liwords/rpc/api/proto/user_service"
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
	IsAdmin     bool   `gorm:"default:false;index"`
	IsDirector  bool   `gorm:"default:false"`
	IsMod       bool   `gorm:"default:false;index"`
	ApiKey      string

	Notoriety int
	Actions   postgres.Jsonb
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

	BirthDate string `gorm:"type:varchar(11)"`

	CountryCode string `gorm:"type:varchar(3)"`
	// Title is some sort of acronym/shorthand for a title. Like GM, EX, SM, UK-GM (UK Grandmaster?)
	Title string `gorm:"type:varchar(8)"`

	// AvatarUrl refers to a file in JPEG format.
	AvatarUrl string `gorm:"type:varchar(128)"`

	// About is profile notes.
	About string `gorm:"type:varchar(2048)"`
	// It's ok to have a few JSON bags here. Postgres makes these easy and fast.
	// XXX: Come up with a model for friend list.
	Ratings postgres.Jsonb // A complex dictionary of ratings with variants/lexica/etc.
	Stats   postgres.Jsonb // Profile stats such as average score per variant, bingos, a lot of other simple stats.
	// More complex stats might be in a separate place.
}

type briefProfile struct {
	UUID        string
	Username    string
	InternalBot bool
	CountryCode string
	AvatarUrl   string
	FirstName   string // XXX please add full_name to db instead.
	LastName    string // XXX please add full_name to db instead.
	BirthDate   string
}

type following struct {
	// Follower follows user; pretty straightforward.
	UserID uint
	User   User

	FollowerID uint
	Follower   User
}

type blocking struct {
	// blocker blocks user
	UserID uint
	User   User

	BlockerID uint
	Blocker   User
}

// NewDBStore creates a new DB store
func NewDBStore(dbURL string) (*DBStore, error) {
	db, err := gorm.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&User{}, &profile{}, &following{}, &blocking{})
	db.Model(&User{}).
		AddUniqueIndex("username_idx", "lower(username)").
		AddUniqueIndex("email_idx", "lower(email)").
		AddIndex("api_key_idx", "api_key")

	// Can't get GORM to auto create these foreign keys, so do it myself /shrug
	db.Model(&profile{}).AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT")
	db.Model(&following{}).
		AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT").
		AddForeignKey("follower_id", "users(id)", "RESTRICT", "RESTRICT").
		AddUniqueIndex("user_follower_idx", "user_id", "follower_id")

	db.Model(&blocking{}).
		AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT").
		AddForeignKey("blocker_id", "users(id)", "RESTRICT", "RESTRICT").
		AddUniqueIndex("user_blocker_idx", "user_id", "blocker_id")

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

	var actions entity.Actions
	err = json.Unmarshal(u.Actions.RawMessage, &actions)
	if err != nil {
		log.Err(err).Msg("convert-user-actions")
	}

	entu := &entity.User{
		ID:         u.ID,
		Username:   u.Username,
		UUID:       u.UUID,
		Email:      u.Email,
		Password:   u.Password,
		IsBot:      u.InternalBot,
		Anonymous:  false,
		Profile:    profile,
		IsAdmin:    u.IsAdmin,
		IsDirector: u.IsDirector,
		IsMod:      u.IsMod,
		Notoriety:  u.Notoriety,
		Actions:    &actions,
	}

	return entu, nil
}

func (s *DBStore) Set(ctx context.Context, u *entity.User) error {
	dbu, err := s.toDBObj(u)
	if err != nil {
		return err
	}

	result := s.db.Model(&User{}).Set("gorm:query_option", "FOR UPDATE").
		Where("uuid = ?", u.UUID).Update(dbu)
	return result.Error
}

// This was written to avoid the zero value trap
func (s *DBStore) SetNotoriety(ctx context.Context, u *entity.User, notoriety int) error {
	result := s.db.Model(&User{}).Where("uuid = ?", u.UUID).Update(map[string]interface{}{"notoriety": notoriety})
	return result.Error
}

func (s *DBStore) SetPermissions(ctx context.Context, req *cpb.PermissionsRequest) error {
	updates := make(map[string]interface{})
	if req.Bot != nil {
		updates["internal_bot"] = req.Bot.Value
	}
	if req.Admin != nil {
		updates["is_admin"] = req.Admin.Value
	}
	if req.Director != nil {
		updates["is_director"] = req.Director.Value
	}
	if req.Mod != nil {
		updates["is_mod"] = req.Mod.Value
	}

	result := s.db.Table("users").Where("lower(username) = ?", strings.ToLower(req.Username)).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		// gorm also sets result.RowsAffected == 0 if len(updates) == 0
		return gorm.ErrRecordNotFound
	}

	return nil
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
		ID:         u.ID,
		Username:   u.Username,
		UUID:       u.UUID,
		Email:      u.Email,
		Password:   u.Password,
		Anonymous:  false,
		IsBot:      u.InternalBot,
		IsAdmin:    u.IsAdmin,
		IsDirector: u.IsDirector,
		IsMod:      u.IsMod,
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
		BirthDate:   p.BirthDate,
		CountryCode: p.CountryCode,
		Title:       p.Title,
		About:       p.About,
		Ratings:     rdata,
		Stats:       sdata,
		AvatarUrl:   p.AvatarUrl,
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

		var actions entity.Actions
		err = json.Unmarshal(u.Actions.RawMessage, &actions)
		if err != nil {
			log.Err(err).Msg("convert-user-actions")
		}

		entu = &entity.User{
			ID:         u.ID,
			Username:   u.Username,
			UUID:       u.UUID,
			Email:      u.Email,
			Password:   u.Password,
			IsBot:      u.InternalBot,
			Profile:    profile,
			IsAdmin:    u.IsAdmin,
			IsDirector: u.IsDirector,
			IsMod:      u.IsMod,
			Notoriety:  u.Notoriety,
			Actions:    &actions,
		}
	}

	return entu, nil
}

// GetByAPIKey gets a user by api key. It does not try to fetch the profile. We only
// call this for API functions where we care about access levels, etc.
func (s *DBStore) GetByAPIKey(ctx context.Context, apikey string) (*entity.User, error) {
	if apikey == "" {
		return nil, errors.New("api-key is blank")
	}
	u := &User{}
	if result := s.db.Where("api_key = ?", apikey).First(u); result.Error != nil {
		return nil, result.Error
	}

	var actions entity.Actions
	err := json.Unmarshal(u.Actions.RawMessage, &actions)
	if err != nil {
		log.Err(err).Msg("convert-user-actions")
	}

	entu := &entity.User{
		ID:         u.ID,
		Username:   u.Username,
		UUID:       u.UUID,
		Email:      u.Email,
		Password:   u.Password,
		Anonymous:  false,
		IsBot:      u.InternalBot,
		IsAdmin:    u.IsAdmin,
		IsDirector: u.IsDirector,
		IsMod:      u.IsMod,
		Notoriety:  u.Notoriety,
		Actions:    &actions,
	}

	return entu, nil
}

func (s *DBStore) toDBObj(u *entity.User) (*User, error) {
	actions, err := json.Marshal(u.Actions)
	if err != nil {
		return nil, err
	}

	return &User{
		UUID:        u.UUID,
		Username:    u.Username,
		Email:       u.Email,
		Password:    u.Password,
		InternalBot: u.IsBot,
		IsAdmin:     u.IsAdmin,
		IsDirector:  u.IsDirector,
		IsMod:       u.IsMod,
		Notoriety:   u.Notoriety,
		Actions:     postgres.Jsonb{RawMessage: actions},
	}, nil
}

// New creates a new user in the DB.
func (s *DBStore) New(ctx context.Context, u *entity.User) error {
	if u.UUID == "" {
		u.UUID = shortuuid.New()
	}
	dbu, err := s.toDBObj(u)
	if err != nil {
		return err
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
	prof := u.Profile
	if prof == nil {
		prof = &entity.Profile{}
	}
	// XXX: no validation for BirthDate etc. :-(
	dbp := &profile{
		User:        *dbu,
		FirstName:   prof.FirstName,
		LastName:    prof.LastName,
		BirthDate:   prof.BirthDate,
		CountryCode: prof.CountryCode,
		Title:       prof.Title,
		AvatarUrl:   prof.AvatarUrl,
		About:       prof.About,
		Ratings:     postgres.Jsonb{RawMessage: ratbytes},
		Stats:       postgres.Jsonb{RawMessage: statbytes},
	}
	// XXX: Should be in transaction.
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

// SetAvatarUrl sets the avatar_url (profile field) for the user.
func (s *DBStore) SetAvatarUrl(ctx context.Context, uuid string, avatarUrl string) error {
	u := &User{}
	p := &profile{}

	if result := s.db.Where("uuid = ?", uuid).First(u); result.Error != nil {
		return result.Error
	}
	if result := s.db.Model(u).Related(p); result.Error != nil {
		return result.Error
	}

	return s.db.Model(p).Update("avatar_url", avatarUrl).Error
}

func (s *DBStore) GetBriefProfiles(ctx context.Context, uuids []string) (map[string]*pb.BriefProfile, error) {
	var profiles []*briefProfile
	if result := s.db.
		Table("users").
		Joins("left join profiles on users.id = profiles.user_id").
		Where("uuid in (?)", uuids).
		Select([]string{"uuid", "username", "internal_bot", "country_code", "avatar_url", "first_name", "last_name", "birth_date"}).
		Find(&profiles); result.Error != nil {
		return nil, result.Error
	}

	profileMap := make(map[string]*briefProfile)
	for _, profile := range profiles {
		profileMap[profile.UUID] = profile
	}

	now := time.Now()

	response := make(map[string]*pb.BriefProfile)
	for _, uuid := range uuids {
		prof, hasProfile := profileMap[uuid]
		if !hasProfile {
			prof = &briefProfile{}
		}
		if prof.AvatarUrl == "" && prof.InternalBot {
			// see entity/user.go
			prof.AvatarUrl = "https://woogles-prod-assets.s3.amazonaws.com/macondog.png"
		}
		subjectIsAdult := entity.IsAdult(prof.BirthDate, now)
		avatarUrl := ""
		fullName := ""
		if subjectIsAdult {
			avatarUrl = prof.AvatarUrl
			// see entity/user.go RealName()
			if prof.FirstName != "" {
				if prof.LastName != "" {
					fullName = prof.FirstName + " " + prof.LastName
				} else {
					fullName = prof.FirstName
				}
			} else {
				fullName = prof.LastName
			}
		}
		response[uuid] = &pb.BriefProfile{
			Username:    prof.Username,
			CountryCode: prof.CountryCode,
			AvatarUrl:   avatarUrl,
			FullName:    fullName,
		}
	}

	return response, nil
}

func (s *DBStore) SetPersonalInfo(ctx context.Context, uuid string, email string, firstName string, lastName string, birthDate string, countryCode string, about string) error {
	u := &User{}
	p := &profile{}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if result := tx.Where("uuid = ?", uuid).First(u); result.Error != nil {
			return result.Error
		}
		if result := tx.Model(u).Update("email", email); result.Error != nil {
			return result.Error
		}
		if result := tx.Model(u).Related(p); result.Error != nil {
			return result.Error
		}

		return tx.Model(p).Update(map[string]interface{}{"first_name": firstName,
			"last_name":    lastName,
			"birth_date":   birthDate,
			"about":        about,
			"country_code": countryCode}).Error
	})

}

// SetRatings set the specific ratings for the given variant in a transaction.
func (s *DBStore) SetRatings(ctx context.Context, p0uuid string, p1uuid string, variant entity.VariantKey,
	p0Rating entity.SingleRating, p1Rating entity.SingleRating) error {

	return s.db.Transaction(func(tx *gorm.DB) error {

		p0Profile, p0RatingBytes, err := getRatingBytes(tx, ctx, p0uuid, variant, p0Rating)
		if err != nil {
			return err
		}

		err = tx.Model(p0Profile).Update("ratings", postgres.Jsonb{RawMessage: p0RatingBytes}).Error

		if err != nil {
			return err
		}

		p1Profile, p1RatingBytes, err := getRatingBytes(tx, ctx, p1uuid, variant, p1Rating)
		if err != nil {
			return err
		}

		err = tx.Model(p1Profile).Update("ratings", postgres.Jsonb{RawMessage: p1RatingBytes}).Error

		if err != nil {
			return err
		}
		// return nil will commit the whole transaction
		return nil
	})
}

func getRatingBytes(tx *gorm.DB, ctx context.Context, uuid string, variant entity.VariantKey,
	rating entity.SingleRating) (*profile, []byte, error) {
	u := &User{}
	p := &profile{}

	if result := tx.Where("uuid = ?", uuid).First(u); result.Error != nil {
		return nil, nil, result.Error
	}
	if result := tx.Model(u).Related(p); result.Error != nil {
		return nil, nil, result.Error
	}

	existingRatings := getExistingRatings(p)
	existingRatings.Data[variant] = rating

	bytes, err := json.Marshal(existingRatings)
	if err != nil {
		return nil, nil, err
	}
	return p, bytes, nil
}

func getExistingRatings(p *profile) *entity.Ratings {
	var existingRatings entity.Ratings
	err := json.Unmarshal(p.Ratings.RawMessage, &existingRatings)
	if err != nil {
		log.Err(err).Msg("existing ratings missing; initializing...")
		existingRatings = entity.Ratings{Data: map[entity.VariantKey]entity.SingleRating{}}
	}

	if existingRatings.Data == nil {
		existingRatings.Data = make(map[entity.VariantKey]entity.SingleRating)
	}
	return &existingRatings
}

func (s *DBStore) SetStats(ctx context.Context, p0uuid string, p1uuid string, variant entity.VariantKey,
	p0Stats *entity.Stats, p1Stats *entity.Stats) error {

	return s.db.Transaction(func(tx *gorm.DB) error {

		p0Profile, p0StatsBytes, err := getStatsBytes(tx, ctx, p0uuid, variant, p0Stats)
		if err != nil {
			return err
		}

		err = tx.Model(p0Profile).Update("stats", postgres.Jsonb{RawMessage: p0StatsBytes}).Error

		if err != nil {
			return err
		}

		p1Profile, p1StatsBytes, err := getStatsBytes(tx, ctx, p1uuid, variant, p1Stats)
		if err != nil {
			return err
		}

		err = tx.Model(p1Profile).Update("stats", postgres.Jsonb{RawMessage: p1StatsBytes}).Error

		if err != nil {
			return err
		}
		// return nil will commit the whole transaction
		return nil
	})
}

func getStatsBytes(tx *gorm.DB, ctx context.Context, uuid string, variant entity.VariantKey,
	stats *entity.Stats) (*profile, []byte, error) {
	u := &User{}
	p := &profile{}

	if result := tx.Where("uuid = ?", uuid).First(u); result.Error != nil {
		return nil, nil, result.Error
	}
	if result := tx.Model(u).Related(p); result.Error != nil {
		return nil, nil, result.Error
	}

	existingProfileStats := getExistingProfileStats(p)
	existingProfileStats.Data[variant] = stats

	bytes, err := json.Marshal(existingProfileStats)
	if err != nil {
		return nil, nil, err
	}
	return p, bytes, nil
}

func getExistingProfileStats(p *profile) *entity.ProfileStats {
	var existingProfileStats entity.ProfileStats
	err := json.Unmarshal(p.Stats.RawMessage, &existingProfileStats)
	if err != nil {
		log.Err(err).Msg("existing stats missing; initializing...")
		existingProfileStats = entity.ProfileStats{Data: map[entity.VariantKey]*entity.Stats{}}
	}
	if existingProfileStats.Data == nil {
		existingProfileStats.Data = make(map[entity.VariantKey]*entity.Stats)
	}
	return &existingProfileStats
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

// GetFollowedBy gets all the users that are following the passed-in user DB ID.
func (s *DBStore) GetFollowedBy(ctx context.Context, uid uint) ([]*entity.User, error) {
	type followedby struct {
		Username string
		Uuid     string
	}

	var users []followedby

	if result := s.db.Table("followings").Select("u0.username, u0.uuid").
		Joins("JOIN users as u0 ON u0.id = follower_id").
		Where("user_id = ?", uid).Scan(&users); result.Error != nil {

		return nil, result.Error
	}
	log.Debug().Int("num-followed-by", len(users)).Msg("found-followed-by")
	entUsers := make([]*entity.User, len(users))
	for idx, u := range users {
		entUsers[idx] = &entity.User{UUID: u.Uuid, Username: u.Username}
	}
	return entUsers, nil
}

func (s *DBStore) AddBlock(ctx context.Context, targetUser, blocker uint) error {
	dbb := &blocking{UserID: targetUser, BlockerID: blocker}
	result := s.db.Create(dbb)
	return result.Error
}

func (s *DBStore) RemoveBlock(ctx context.Context, targetUser, blocker uint) error {
	return s.db.Where("user_id = ? AND blocker_id = ?", targetUser, blocker).Delete(&blocking{}).Error
}

// GetBlocks gets all the users that the passed-in user DB ID is blocking.
func (s *DBStore) GetBlocks(ctx context.Context, uid uint) ([]*entity.User, error) {
	type blocked struct {
		Username string
		Uuid     string
	}

	var users []blocked

	if result := s.db.Table("blockings").Select("u0.username, u0.uuid").
		Joins("JOIN users as u0 ON u0.id = user_id").
		Where("blocker_id = ?", uid).Scan(&users); result.Error != nil {

		return nil, result.Error
	}
	log.Debug().Int("num-blocked", len(users)).Msg("found-blocked")
	entUsers := make([]*entity.User, len(users))
	for idx, u := range users {
		entUsers[idx] = &entity.User{UUID: u.Uuid, Username: u.Username}
	}
	return entUsers, nil
}

// GetBlockedBy gets all the users that are blocking the passed-in user DB ID.
func (s *DBStore) GetBlockedBy(ctx context.Context, uid uint) ([]*entity.User, error) {
	type blockedby struct {
		Username string
		Uuid     string
	}

	var users []blockedby

	if result := s.db.Table("blockings").Select("u0.username, u0.uuid").
		Joins("JOIN users as u0 ON u0.id = blocker_id").
		Where("user_id = ?", uid).Scan(&users); result.Error != nil {

		return nil, result.Error
	}
	log.Debug().Int("num-blocked-by", len(users)).Msg("found-blocked-by")
	entUsers := make([]*entity.User, len(users))
	for idx, u := range users {
		entUsers[idx] = &entity.User{UUID: u.Uuid, Username: u.Username}
	}
	return entUsers, nil
}

// GetFullBlocks gets users uid is blocking AND users blocking uid
func (s *DBStore) GetFullBlocks(ctx context.Context, uid uint) ([]*entity.User, error) {
	// There's probably a way to do this with one db query but eh.
	players := map[string]*entity.User{}

	blocks, err := s.GetBlocks(ctx, uid)
	if err != nil {
		return nil, err
	}
	blockedby, err := s.GetBlockedBy(ctx, uid)
	if err != nil {
		return nil, err
	}

	for _, u := range blocks {
		players[u.UUID] = u
	}
	for _, u := range blockedby {
		players[u.UUID] = u
	}

	plist := make([]*entity.User, len(players))
	idx := 0
	for _, v := range players {
		plist[idx] = v
		idx++
	}
	return plist, nil
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

func (s *DBStore) UsersByPrefix(ctx context.Context, prefix string) ([]*pb.BasicUser, error) {

	type u struct {
		Username string
		UUID     string
	}

	var us []u
	// This is slightly egregious. Since importing the mod
	// package would result in a circular dependency, we cannot
	// get the string the correct way with ms.ModActionType_SUSPEND_ACCOUNT.String(),
	// so we hard code it here.
	lowerPrefix := strings.ToLower(prefix)
	if result := s.db.Table("users").Select("username, uuid").
		Where("substr(lower(username), 1, length(?)) = ? AND internal_bot = ? AND (actions IS NULL OR actions->'Current' IS NULL OR actions->'Current'->'SUSPEND_ACCOUNT' IS NULL OR actions->'Current'->'SUSPEND_ACCOUNT'->'end_time' IS NOT NULL)",
			lowerPrefix, lowerPrefix, false).
		Limit(20).
		Scan(&us); result.Error != nil {
		return nil, result.Error
	}
	log.Debug().Str("prefix", prefix).Int("byprefix", len(us)).Msg("found-matches")

	users := make([]*pb.BasicUser, len(us))
	for idx, u := range us {
		users[idx] = &pb.BasicUser{Username: u.Username, Uuid: u.UUID}
	}
	sort.Slice(users, func(i int, j int) bool {
		return strings.ToLower(users[i].Username) < strings.ToLower(users[j].Username)
	})

	return users, nil
}

// List all user IDs.
func (s *DBStore) ListAllIDs(ctx context.Context) ([]string, error) {
	var uids []struct{ Uuid string }
	result := s.db.Table("users").Select("uuid").Scan(&uids)

	ids := make([]string, len(uids))
	for idx, uid := range uids {
		ids[idx] = uid.Uuid
	}

	return ids, result.Error
}

func (s *DBStore) ResetStats(ctx context.Context, uid string) error {
	u, err := s.GetByUUID(ctx, uid)
	if err != nil {
		return err
	}
	p := &profile{}
	if result := s.db.Model(u).Related(p); result.Error != nil {
		return fmt.Errorf("Error getting profile for %s", uid)
	}

	emptyStats := &entity.Stats{}
	bytes, err := json.Marshal(emptyStats)
	if err != nil {
		return err
	}
	err = s.db.Model(p).Update("stats", bytes).Error
	if err != nil {
		return err
	}

	return nil
}

func (s *DBStore) ResetRatings(ctx context.Context, uid string) error {
	u, err := s.GetByUUID(ctx, uid)
	if err != nil {
		return err
	}
	p := &profile{}
	if result := s.db.Model(u).Related(p); result.Error != nil {
		return fmt.Errorf("Error getting profile for %s", uid)
	}

	emptyRatings := &entity.Ratings{}
	bytes, err := json.Marshal(emptyRatings)
	if err != nil {
		return err
	}
	err = s.db.Model(p).Update("ratings", bytes).Error
	if err != nil {
		return err
	}

	return nil
}

func (s *DBStore) ResetStatsAndRatings(ctx context.Context, uid string) error {
	u, err := s.GetByUUID(ctx, uid)
	if err != nil {
		return err
	}
	p := &profile{}
	if result := s.db.Model(u).Related(p); result.Error != nil {
		return fmt.Errorf("Error getting profile for %s", uid)
	}

	emptyRatings := &entity.Ratings{}
	bytes, err := json.Marshal(emptyRatings)
	if err != nil {
		return err
	}
	err = s.db.Model(p).Update("ratings", bytes).Error
	if err != nil {
		return err
	}

	emptyStats := &entity.Stats{}
	bytes, err = json.Marshal(emptyStats)
	if err != nil {
		return err
	}
	err = s.db.Model(p).Update("stats", bytes).Error
	if err != nil {
		return err
	}

	return nil
}

func (s *DBStore) ResetPersonalInfo(ctx context.Context, uuid string) error {
	u := &User{}
	p := &profile{}

	if result := s.db.Where("uuid = ?", uuid).First(u); result.Error != nil {
		return result.Error
	}
	if result := s.db.Model(u).Related(p); result.Error != nil {
		return result.Error
	}

	return s.db.Model(p).Update(map[string]interface{}{"first_name": "",
		"last_name":    "",
		"about":        "",
		"title":        "",
		"avatar_url":   "",
		"country_code": ""}).Error
}

func (s *DBStore) ResetProfile(ctx context.Context, uid string) error {
	err := s.ResetStatsAndRatings(ctx, uid)
	if err != nil {
		return err
	}
	return s.ResetPersonalInfo(ctx, uid)
}

func (s *DBStore) Disconnect() {
	s.db.Close()
}

func (s *DBStore) Count(ctx context.Context) (int64, error) {
	var count int64
	result := s.db.Model(&User{}).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (s *DBStore) CachedCount(ctx context.Context) int {
	return 0
}

func (s *DBStore) GetModList(ctx context.Context) (*pb.GetModListResponse, error) {
	var users []User
	if result := s.db.
		Where("is_admin = ?", true).Or("is_mod = ?", true).
		Select([]string{"uuid", "is_admin", "is_mod"}).
		Find(&users); result.Error != nil {
		return nil, result.Error
	}

	var adminUserIds []string
	var modUserIds []string

	for _, user := range users {
		if user.IsAdmin {
			adminUserIds = append(adminUserIds, user.UUID)
		}
		if user.IsMod {
			modUserIds = append(modUserIds, user.UUID)
		}
	}

	return &pb.GetModListResponse{
		AdminUserIds: adminUserIds,
		ModUserIds:   modUserIds,
	}, nil
}
