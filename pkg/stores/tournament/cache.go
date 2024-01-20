package tournament

import (
	"context"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/entity"

	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

type backingStore interface {
	Get(ctx context.Context, id string) (*entity.Tournament, error)
	GetBySlug(ctx context.Context, id string) (*entity.Tournament, error)
	Set(context.Context, *entity.Tournament) error
	Create(context.Context, *entity.Tournament) error
	GetRecentGames(ctx context.Context, tourneyID string, numGames int, offset int) (*pb.RecentGamesResponse, error)
	Disconnect()
	SetTournamentEventChan(c chan<- *entity.EventWrapper)
	TournamentEventChan() chan<- *entity.EventWrapper
	GetRecentClubSessions(ctx context.Context, clubID string, numSessions int, offset int) (*pb.ClubSessionsResponse, error)
	ListAllIDs(context.Context) ([]string, error)

	AddRegistrants(ctx context.Context, tid string, userIDs []string, division string) error
	RemoveRegistrants(ctx context.Context, tid string, userIDs []string, division string) error
	RemoveRegistrantsForTournament(ctx context.Context, tid string) error
	ActiveTournamentsFor(ctx context.Context, userID string) ([][2]string, error)
}

const (
	// Increase this if we ever think we might be holding more than
	// 50 tournaments at a time.
	CacheCap = 50
)

// Cache will reside in-memory, and will be per-node.
type Cache struct {
	sync.Mutex
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

// Get gets a tournament from the cache. It loads it into the cache if it's not there.
func (c *Cache) Get(ctx context.Context, id string) (*entity.Tournament, error) {
	tm, ok := c.cache.Get(id)
	if ok && tm != nil {
		return tm.(*entity.Tournament), nil
	}

	// Recheck after locking, to ensure it is still not there.
	c.Lock()
	defer c.Unlock()
	tm, ok = c.cache.Get(id)
	if ok && tm != nil {
		return tm.(*entity.Tournament), nil
	}
	log.Info().Str("tournamentid", id).Msg("not-in-cache")
	uncachedTournament, err := c.backing.Get(ctx, id)
	if err == nil {
		c.cache.Add(id, uncachedTournament)
	}
	return uncachedTournament, err
}

func (c *Cache) GetBySlug(ctx context.Context, id string) (*entity.Tournament, error) {
	return c.backing.GetBySlug(ctx, id)
}

// Set sets a tournament in the cache, AND in the backing store. This ensures if the
// node crashes the tournament doesn't just vanish.
func (c *Cache) Set(ctx context.Context, tm *entity.Tournament) error {
	return c.setOrCreate(ctx, tm, false)
}

// Create creates the tournament in the cache as well as the store.
func (c *Cache) Create(ctx context.Context, tm *entity.Tournament) error {
	return c.setOrCreate(ctx, tm, true)
}

func (c *Cache) setOrCreate(ctx context.Context, tm *entity.Tournament, isNew bool) error {
	var err error
	if isNew {
		err = c.backing.Create(ctx, tm)
	} else {
		err = c.backing.Set(ctx, tm)
	}
	if err != nil {
		return err
	}
	c.cache.Add(tm.UUID, tm)
	return nil
}

// Unload unloads the tournament from the cache
func (c *Cache) Unload(ctx context.Context, id string) {
	c.cache.Remove(id)
}

func (c *Cache) Disconnect() {
	c.backing.Disconnect()
}

// SetTournamentEventChan sets the tournament event channel to the passed in channel.
func (c *Cache) SetTournamentEventChan(ch chan<- *entity.EventWrapper) {
	c.backing.SetTournamentEventChan(ch)
}

func (c *Cache) TournamentEventChan() chan<- *entity.EventWrapper {
	return c.backing.TournamentEventChan()
}

func (c *Cache) GetRecentGames(ctx context.Context, tourneyID string, numGames int, offset int) (*pb.RecentGamesResponse, error) {
	return c.backing.GetRecentGames(ctx, tourneyID, numGames, offset)
}

func (c *Cache) GetRecentClubSessions(ctx context.Context, clubID string, numSessions int, offset int) (*pb.ClubSessionsResponse, error) {
	return c.backing.GetRecentClubSessions(ctx, clubID, numSessions, offset)
}

func (c *Cache) ListAllIDs(ctx context.Context) ([]string, error) {
	return c.backing.ListAllIDs(ctx)
}

func (c *Cache) AddRegistrants(ctx context.Context, tid string, userIDs []string, division string) error {
	return c.backing.AddRegistrants(ctx, tid, userIDs, division)
}

func (c *Cache) RemoveRegistrants(ctx context.Context, tid string, userIDs []string, division string) error {
	return c.backing.RemoveRegistrants(ctx, tid, userIDs, division)
}

func (c *Cache) RemoveRegistrantsForTournament(ctx context.Context, tid string) error {
	return c.backing.RemoveRegistrantsForTournament(ctx, tid)
}

func (c *Cache) ActiveTournamentsFor(ctx context.Context, userID string) ([][2]string, error) {
	return c.backing.ActiveTournamentsFor(ctx, userID)
}
