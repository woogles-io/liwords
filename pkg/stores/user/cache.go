package user

import (
	"context"
	"sync"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/utilities"
	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/zerolog/log"

	cpb "github.com/domino14/liwords/rpc/api/proto/config_service"
	pb "github.com/domino14/liwords/rpc/api/proto/user_service"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

// same as the GameStore in gameplay package, but this gives us a bit more flexibility
// in defining the backing store (i.e. it may not necessarily be a SQL db store)
type backingStore interface {
	Get(ctx context.Context, username string) (*entity.User, error)
	GetByUUID(ctx context.Context, uuid string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByAPIKey(ctx context.Context, apikey string) (*entity.User, error)
	// Username by UUID. Good for fast lookups.
	Username(ctx context.Context, uuid string) (string, error)
	New(ctx context.Context, user *entity.User) error
	SetPassword(ctx context.Context, uuid string, hashpass string) error
	SetAvatarUrl(ctx context.Context, uuid string, avatarUrl string) error
	GetBriefProfiles(ctx context.Context, uuids []string) (map[string]*pb.BriefProfile, error)
	SetPersonalInfo(ctx context.Context, uuid string, email string, firstName string, lastName string, birthDate string, countryCode string, about string, silentMode bool) error
	SetRatings(ctx context.Context, p0uuid string, p1uuid string, variant entity.VariantKey,
		p1Rating *entity.SingleRating, p2Rating *entity.SingleRating) error
	SetStats(ctx context.Context, p0uuid string, p1uuid string, variant entity.VariantKey,
		p0stats *entity.Stats, p1stats *entity.Stats) error
	SetNotoriety(ctx context.Context, uuid string, notoriety int) error
	SetActions(ctx context.Context, uuid string, actions *entity.Actions) error
	ResetRatings(ctx context.Context, uuid string) error
	ResetStats(ctx context.Context, uuid string) error
	ResetProfile(ctx context.Context, uuid string) error
	ResetPersonalInfo(ctx context.Context, uuid string) error
	GetBot(ctx context.Context, botType macondopb.BotRequest_BotCode) (*entity.User, error)

	AddFollower(ctx context.Context, targetUser, follower uint) error
	RemoveFollower(ctx context.Context, targetUser, follower uint) error
	// GetFollows gets all the users that the passed-in DB ID is following.
	GetFollows(ctx context.Context, uid uint) ([]*entity.User, error)
	GetFollowedBy(ctx context.Context, uid uint) ([]*entity.User, error)

	AddBlock(ctx context.Context, targetUser, blocker uint) error
	RemoveBlock(ctx context.Context, targetUser, blocker uint) error
	// GetBlocks gets all the users that the passed-in DB ID is blocking
	GetBlocks(ctx context.Context, uid uint) ([]*entity.User, error)
	GetBlockedBy(ctx context.Context, uid uint) ([]*entity.User, error)
	GetFullBlocks(ctx context.Context, uid uint) ([]*entity.User, error)

	UsersByPrefix(ctx context.Context, prefix string) ([]*pb.BasicUser, error)
	Count(ctx context.Context) (int64, error)
	SetPermissions(ctx context.Context, req *cpb.PermissionsRequest) error

	GetModList(ctx context.Context) (*pb.GetModListResponse, error)
}

const (
	// Allow for 400 simultaneously logged on users.
	CacheCap = 400
)

type BriefProfileCache struct {
	sync.Mutex
	cache map[string]*pb.BriefProfile
}

type modListCache struct {
	sync.Mutex
	expiry   time.Time
	response *pb.GetModListResponse
}

// Cache will reside in-memory, and will be per-node.
type Cache struct {
	sync.Mutex
	cache *lru.Cache

	backing backingStore

	briefProfileCache *BriefProfileCache

	cachedModList modListCache
}

func NewCache(backing backingStore) *Cache {

	lrucache, err := lru.New(CacheCap)
	if err != nil {
		panic(err)
	}

	// good thing this doesn't auto-expire...
	bpc := make(map[string]*pb.BriefProfile)

	bpc[utilities.CensoredUsername] = &pb.BriefProfile{
		Username:    utilities.CensoredUsername,
		FullName:    utilities.CensoredUsername,
		CountryCode: "",
		AvatarUrl:   utilities.CensoredAvatarUrl,
	}

	bpc[utilities.AnotherCensoredUsername] = &pb.BriefProfile{
		Username:    utilities.AnotherCensoredUsername,
		FullName:    utilities.AnotherCensoredUsername,
		CountryCode: "",
		AvatarUrl:   utilities.CensoredAvatarUrl,
	}

	return &Cache{
		backing: backing,
		cache:   lrucache,
		briefProfileCache: &BriefProfileCache{
			cache: bpc,
		},
	}
}

func (c *Cache) uncacheBriefProfile(uuid string) {
	c.briefProfileCache.Lock()
	defer c.briefProfileCache.Unlock()

	delete(c.briefProfileCache.cache, uuid)
}

func (c *Cache) Get(ctx context.Context, username string) (*entity.User, error) {
	return c.backing.Get(ctx, username)
}

func (c *Cache) GetByUUID(ctx context.Context, uuid string) (*entity.User, error) {
	u, ok := c.cache.Get(uuid)
	if ok && u != nil {
		return u.(*entity.User), nil
	}

	// Recheck after locking, to ensure it is still not there.
	c.Lock()
	defer c.Unlock()
	u, ok = c.cache.Get(uuid)
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

func (c *Cache) Username(ctx context.Context, uuid string) (string, error) {
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

func (c *Cache) SetAvatarUrl(ctx context.Context, uuid string, avatarUrl string) error {
	u, err := c.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	u.Profile.AvatarUrl = avatarUrl
	if err = c.backing.SetAvatarUrl(ctx, uuid, avatarUrl); err != nil {
		return err
	}
	c.uncacheBriefProfile(uuid)
	return nil
}

func (c *Cache) GetBriefProfiles(ctx context.Context, uuids []string) (map[string]*pb.BriefProfile, error) {
	c.briefProfileCache.Lock()
	defer c.briefProfileCache.Unlock()

	missingUuids := make([]string, 0, len(uuids))
	ret := make(map[string]*pb.BriefProfile)
	for _, uuid := range uuids {
		if cached, ok := c.briefProfileCache.cache[uuid]; ok {
			ret[uuid] = cached
		} else {
			missingUuids = append(missingUuids, uuid)
		}
	}

	if len(missingUuids) > 0 {
		additionalStuffs, err := c.backing.GetBriefProfiles(ctx, missingUuids)
		if err != nil {
			return nil, err
		}

		if len(additionalStuffs) > 0 {
			// only cache existing values, so cache size is bounded by database size.

			for uuid, value := range additionalStuffs {
				ret[uuid] = value
				c.briefProfileCache.cache[uuid] = value
			}

			// this one goroutine will evict all of these values at the same time
			go func() {
				time.Sleep(5 * time.Minute)

				c.briefProfileCache.Lock()
				defer c.briefProfileCache.Unlock()

				for uuid := range additionalStuffs {
					delete(c.briefProfileCache.cache, uuid)
				}
			}()
		}
	}

	return ret, nil
}

func (c *Cache) SetPersonalInfo(ctx context.Context, uuid string, email string, firstName string, lastName string, birthDate string, countryCode string, about string, silentMode bool) error {
	u, err := c.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}

	if err = c.backing.SetPersonalInfo(ctx, uuid, email, firstName, lastName, birthDate, countryCode, about, silentMode); err != nil {
		return err
	}
	u.Email = email
	u.Profile.FirstName = firstName
	u.Profile.LastName = lastName
	u.Profile.BirthDate = birthDate
	u.Profile.CountryCode = countryCode
	u.Profile.About = about
	u.Profile.SilentMode = silentMode
	c.uncacheBriefProfile(uuid)
	return nil
}

func (c *Cache) ResetPersonalInfo(ctx context.Context, uuid string) error {
	u, err := c.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}

	if err = c.backing.ResetPersonalInfo(ctx, uuid); err != nil {
		return err
	}
	u.Profile.FirstName = ""
	u.Profile.LastName = ""
	u.Profile.BirthDate = ""
	u.Profile.CountryCode = ""
	u.Profile.Title = ""
	u.Profile.AvatarUrl = ""
	u.Profile.About = ""
	u.Profile.SilentMode = false
	c.uncacheBriefProfile(uuid)
	return nil
}

func (c *Cache) SetRatings(ctx context.Context, p0uuid string, p1uuid string, variant entity.VariantKey, p0Rating *entity.SingleRating, p1Rating *entity.SingleRating) error {
	u0, err := c.GetByUUID(ctx, p0uuid)
	if err != nil {
		return err
	}
	u1, err := c.GetByUUID(ctx, p1uuid)
	if err != nil {
		return err
	}

	err = c.backing.SetRatings(ctx, p0uuid, p1uuid, variant, p0Rating, p1Rating)
	if err != nil {
		return err
	}

	if u0.Profile.Ratings.Data == nil {
		u0.Profile.Ratings.Data = make(map[entity.VariantKey]entity.SingleRating)
	}

	if u1.Profile.Ratings.Data == nil {
		u1.Profile.Ratings.Data = make(map[entity.VariantKey]entity.SingleRating)
	}

	u0.Profile.Ratings.Data[variant] = *p0Rating
	u1.Profile.Ratings.Data[variant] = *p1Rating

	return nil
}

func (c *Cache) ResetRatings(ctx context.Context, uuid string) error {
	u, err := c.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	err = c.backing.ResetRatings(ctx, uuid)
	if err != nil {
		return err
	}
	u.Profile.Ratings.Data = nil
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

	err = c.backing.SetStats(ctx, p0uuid, p1uuid, variant, p0Stats, p1Stats)
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
	return nil
}

func (c *Cache) ResetStats(ctx context.Context, uuid string) error {
	u, err := c.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	err = c.backing.ResetStats(ctx, uuid)
	if err != nil {
		return err
	}
	u.Profile.Stats.Data = nil
	return nil
}

func (c *Cache) ResetProfile(ctx context.Context, uuid string) error {
	err := c.ResetStats(ctx, uuid)
	if err != nil {
		return err
	}
	err = c.ResetRatings(ctx, uuid)
	if err != nil {
		return err
	}
	return c.ResetPersonalInfo(ctx, uuid)
}

func (c *Cache) GetBot(ctx context.Context, botType macondopb.BotRequest_BotCode) (*entity.User, error) {
	return c.backing.GetBot(ctx, botType)
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

func (c *Cache) GetFollowedBy(ctx context.Context, uid uint) ([]*entity.User, error) {
	return c.backing.GetFollowedBy(ctx, uid)
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

func (c *Cache) SetActions(ctx context.Context, uuid string, actions *entity.Actions) error {
	err := c.backing.SetActions(ctx, uuid, actions)
	if err != nil {
		return err
	}
	u, err := c.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	u.Actions = actions
	return nil
}

// This was written to avoid the zero value trap
func (c *Cache) SetNotoriety(ctx context.Context, uuid string, notoriety int) error {
	err := c.backing.SetNotoriety(ctx, uuid, notoriety)
	if err != nil {
		return err
	}
	u, err := c.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	u.Notoriety = notoriety
	return nil
}

func (c *Cache) SetPermissions(ctx context.Context, req *cpb.PermissionsRequest) error {
	return c.backing.SetPermissions(ctx, req)
}

func (c *Cache) GetModList(ctx context.Context) (*pb.GetModListResponse, error) {
	c.cachedModList.Lock()
	defer c.cachedModList.Unlock()
	if c.cachedModList.response != nil && time.Now().Before(c.cachedModList.expiry) {
		return c.cachedModList.response, nil
	}

	resp, err := c.backing.GetModList(ctx)
	if err != nil {
		return nil, err
	}

	c.cachedModList.response = resp
	c.cachedModList.expiry = time.Now().Add(time.Minute)
	return resp, nil
}
