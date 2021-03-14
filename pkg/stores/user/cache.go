package user

import (
	"context"

	"github.com/domino14/liwords/pkg/entity"
	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/zerolog/log"

	pb "github.com/domino14/liwords/rpc/api/proto/user_service"
)

// same as the GameStore in gameplay package, but this gives us a bit more flexibility
// in defining the backing store (i.e. it may not necessarily be a SQL db store)
type backingStore interface {
	Get(ctx context.Context, username string) (*entity.User, error)
	GetByUUID(ctx context.Context, uuid string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByAPIKey(ctx context.Context, apikey string) (*entity.User, error)
	// Username by UUID. Good for fast lookups.
	Username(ctx context.Context, uuid string) (string, bool, error)
	New(ctx context.Context, user *entity.User) error
	SetPassword(ctx context.Context, uuid string, hashpass string) error
	SetAbout(ctx context.Context, uuid string, about string) error
	SetAvatarUrl(ctx context.Context, uuid string, avatarUrl string) error
	SetPersonalInfo(ctx context.Context, uuid string, email string, firstName string, lastName string, countryCode string, about string) error
	SetRatings(ctx context.Context, p0uuid string, p1uuid string, variant entity.VariantKey,
		p1Rating entity.SingleRating, p2Rating entity.SingleRating) error
	SetStats(ctx context.Context, p0uuid string, p1uuid string, variant entity.VariantKey,
		p0stats *entity.Stats, p1stats *entity.Stats) error
	ResetRatings(ctx context.Context, uuid string) error
	ResetStats(ctx context.Context, uuid string) error
	GetRandomBot(ctx context.Context) (*entity.User, error)

	AddFollower(ctx context.Context, targetUser, follower uint) error
	RemoveFollower(ctx context.Context, targetUser, follower uint) error
	// GetFollows gets all the users that the passed-in DB ID is following.
	GetFollows(ctx context.Context, uid uint) ([]*entity.User, error)

	AddBlock(ctx context.Context, targetUser, blocker uint) error
	RemoveBlock(ctx context.Context, targetUser, blocker uint) error
	// GetBlocks gets all the users that the passed-in DB ID is blocking
	GetBlocks(ctx context.Context, uid uint) ([]*entity.User, error)
	GetBlockedBy(ctx context.Context, uid uint) ([]*entity.User, error)
	GetFullBlocks(ctx context.Context, uid uint) ([]*entity.User, error)

	UsersByPrefix(ctx context.Context, prefix string) ([]*pb.BasicUser, error)
	Count(ctx context.Context) (int64, error)
	Set(ctx context.Context, u *entity.User) error
}

const (
	// Allow for 400 simultaneously logged on users.
	CacheCap = 400
)

// Cache will reside in-memory, and will be per-node.
type Cache struct {
	cache *lru.Cache

	backing backingStore
}

func NewCache(backing backingStore) *Cache {

	lrucache, err := lru.New(CacheCap)
	if err != nil {
		panic(err)
	}

	return &Cache{
		backing: backing,
		cache:   lrucache,
	}
}

func (c *Cache) Get(ctx context.Context, username string) (*entity.User, error) {
	return c.backing.Get(ctx, username)
}

func (c *Cache) GetByUUID(ctx context.Context, uuid string) (*entity.User, error) {
	u, ok := c.cache.Get(uuid)
	if ok && u != nil {
		return u.(*entity.User), nil
	}
	log.Info().Str("uid", uuid).Msg("not-in-cache")
	uncachedUser, err := c.backing.GetByUUID(ctx, uuid)
	if err == nil {
		c.cache.Add(uuid, uncachedUser)
	}
	return uncachedUser, err
}

func (c *Cache) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	return c.backing.GetByEmail(ctx, email)
}

func (c *Cache) GetByAPIKey(ctx context.Context, apikey string) (*entity.User, error) {
	return c.backing.GetByAPIKey(ctx, apikey)
}

func (c *Cache) Username(ctx context.Context, uuid string) (string, bool, error) {
	return c.backing.Username(ctx, uuid)
}

func (c *Cache) New(ctx context.Context, user *entity.User) error {
	return c.backing.New(ctx, user)
}

func (c *Cache) SetPassword(ctx context.Context, uuid string, hashpass string) error {
	u, err := c.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	u.Password = hashpass
	return c.backing.SetPassword(ctx, uuid, hashpass)
}

func (c *Cache) SetAbout(ctx context.Context, uuid string, about string) error {
	u, err := c.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	u.Profile.About = about
	return c.backing.SetAbout(ctx, uuid, about)
}

func (c *Cache) SetAvatarUrl(ctx context.Context, uuid string, avatarUrl string) error {
	u, err := c.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	u.Profile.AvatarUrl = avatarUrl
	return c.backing.SetAvatarUrl(ctx, uuid, avatarUrl)
}

func (c *Cache) SetPersonalInfo(ctx context.Context, uuid string, email string, firstName string, lastName string, countryCode string, about string) error {
	u, err := c.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	u.Email = email
	u.Profile.FirstName = firstName
	u.Profile.LastName = lastName
	u.Profile.CountryCode = countryCode
	return c.backing.SetPersonalInfo(ctx, uuid, email, firstName, lastName, countryCode, about)
}

func (c *Cache) SetRatings(ctx context.Context, p0uuid string, p1uuid string, variant entity.VariantKey, p0Rating entity.SingleRating, p1Rating entity.SingleRating) error {
	u0, err := c.GetByUUID(ctx, p0uuid)
	if err != nil {
		return err
	}
	u1, err := c.GetByUUID(ctx, p1uuid)
	if err != nil {
		return err
	}

	if u0.Profile.Ratings.Data == nil {
		u0.Profile.Ratings.Data = make(map[entity.VariantKey]entity.SingleRating)
	}

	if u1.Profile.Ratings.Data == nil {
		u1.Profile.Ratings.Data = make(map[entity.VariantKey]entity.SingleRating)
	}

	u0.Profile.Ratings.Data[variant] = p0Rating
	u1.Profile.Ratings.Data[variant] = p1Rating

	err = c.backing.SetRatings(ctx, p0uuid, p1uuid, variant, p0Rating, p1Rating)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cache) ResetRatings(ctx context.Context, uuid string) error {
	u, err := c.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	u.Profile.Ratings.Data = nil
	err = c.backing.ResetRatings(ctx, uuid)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cache) SetStats(ctx context.Context, p0uuid string, p1uuid string, variant entity.VariantKey,
	p0Stats *entity.Stats, p1Stats *entity.Stats) error {
	u0, err := c.GetByUUID(ctx, p0uuid)
	if err != nil {
		return err
	}
	u1, err := c.GetByUUID(ctx, p1uuid)
	if err != nil {
		return err
	}

	if u0.Profile.Stats.Data == nil {
		u0.Profile.Stats.Data = make(map[entity.VariantKey]*entity.Stats)
	}

	if u1.Profile.Stats.Data == nil {
		u1.Profile.Stats.Data = make(map[entity.VariantKey]*entity.Stats)
	}

	u0.Profile.Stats.Data[variant] = p0Stats
	u1.Profile.Stats.Data[variant] = p1Stats

	err = c.backing.SetStats(ctx, p0uuid, p1uuid, variant, p0Stats, p1Stats)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cache) ResetStats(ctx context.Context, uuid string) error {
	u, err := c.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	u.Profile.Stats.Data = nil
	err = c.backing.ResetStats(ctx, uuid)
	if err != nil {
		return err
	}
	return nil
}


func (c *Cache) GetRandomBot(ctx context.Context) (*entity.User, error) {
	return c.backing.GetRandomBot(ctx)
}

func (c *Cache) AddFollower(ctx context.Context, targetUser, follower uint) error {
	return c.backing.AddFollower(ctx, targetUser, follower)
}

func (c *Cache) RemoveFollower(ctx context.Context, targetUser, follower uint) error {
	return c.backing.RemoveFollower(ctx, targetUser, follower)
}

func (c *Cache) GetFollows(ctx context.Context, uid uint) ([]*entity.User, error) {
	return c.backing.GetFollows(ctx, uid)
}

func (c *Cache) AddBlock(ctx context.Context, targetUser, blocker uint) error {
	return c.backing.AddBlock(ctx, targetUser, blocker)
}

func (c *Cache) RemoveBlock(ctx context.Context, targetUser, blocker uint) error {
	return c.backing.RemoveBlock(ctx, targetUser, blocker)
}

func (c *Cache) GetBlocks(ctx context.Context, uid uint) ([]*entity.User, error) {
	return c.backing.GetBlocks(ctx, uid)
}

func (c *Cache) GetBlockedBy(ctx context.Context, uid uint) ([]*entity.User, error) {
	return c.backing.GetBlockedBy(ctx, uid)
}

func (c *Cache) GetFullBlocks(ctx context.Context, uid uint) ([]*entity.User, error) {
	return c.backing.GetFullBlocks(ctx, uid)
}

func (c *Cache) UsersByPrefix(ctx context.Context, prefix string) ([]*pb.BasicUser, error) {
	return c.backing.UsersByPrefix(ctx, prefix)
}

func (c *Cache) Count(ctx context.Context) (int64, error) {
	return c.backing.Count(ctx)
}

func (c *Cache) CachedCount(ctx context.Context) int {
	return c.cache.Len()
}

func (c *Cache) Set(ctx context.Context, u *entity.User) error {

	err := c.backing.Set(ctx, u)
	if err != nil {
		return err
	}
	// readd to cache
	c.cache.Add(u.UUID, u)
	return nil
}
