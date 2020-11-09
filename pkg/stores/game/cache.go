package game

import (
	"context"
	"sync"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	gs "github.com/domino14/liwords/rpc/api/proto/game_service"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/zerolog/log"
)

// same as the GameStore in gameplay package, but this gives us a bit more flexibility
// in defining the backing store (i.e. it may not necessarily be a SQL db store)
type backingStore interface {
	Get(ctx context.Context, id string) (*entity.Game, error)
	GetMetadata(ctx context.Context, id string) (*gs.GameInfoResponse, error)
	GetRematchStreak(ctx context.Context, originalRequestId string) (*gs.GameInfoResponses, error)
	GetRecentGames(ctx context.Context, username string, numGames int, offset int) (*gs.GameInfoResponses, error)
	Set(context.Context, *entity.Game) error
	Create(context.Context, *entity.Game) error
	Exists(context.Context, string) (bool, error)
	ListActive(ctx context.Context) ([]*pb.GameMeta, error)
	Count(ctx context.Context) (int64, error)
	SetGameEventChan(ch chan<- *entity.EventWrapper)
	Disconnect()
}

const (
	// Assume every game takes up roughly 50KB in memory
	// This is roughly 200 MB and allows for 4000 simultaneous games.
	// We will want to increase this as we grow (as well as the size of our container)

	// Note: above is overly optimistic.
	// It seems each cache slot is taking about 750kB.
	// That's in addition to about 200MB base.
	// Reduced cache cap accordingly.
	CacheCap = 400
)

// Cache will reside in-memory, and will be per-node. If we add more nodes
// we will need to make sure only the right nodes respond to game requests.
type Cache struct {
	sync.RWMutex // used for the activeGames cache.
	cache        *lru.Cache
	activeGames  []*pb.GameMeta

	activeGamesTTL         time.Duration
	activeGamesLastUpdated time.Time

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

		// games:   make(map[string]*entity.Game),
		// Have a non-trivial TTL for the cache of active games.
		// XXX: This might act poorly if the following happens within the TTL:
		//  - active games gets cached
		//  - someone starts playing a game
		//  - new player logs on and fetches active games
		//  - new player will receive the old games and not the new game?
		// One solution: bust the cache or add/subtract directly from cache
		//  when a new game is created/ended.
		// Problem: this won't work for distributed nodes. Once we
		// add multiple nodes we should probably have a Redis cache for a
		// few things (especially game quickdata).
		activeGamesTTL: time.Second * 5,
	}
}

// Unload unloads the game from the cache
func (c *Cache) Unload(ctx context.Context, id string) {
	c.cache.Remove(id)
	// Let's also expire the active games cache. The only time we ever
	// call Unload is when a game is over - so we don't want to go back
	// to the lobby and still show our game as active.
	c.Lock()
	defer c.Unlock()
	c.activeGamesLastUpdated = time.Time{}
}

// SetGameEventChan sets the game event channel to the passed in channel.
func (c *Cache) SetGameEventChan(ch chan<- *entity.EventWrapper) {
	c.backing.SetGameEventChan(ch)
}

// Get gets a game from the cache.. it loads it into the cache if it's not there.
func (c *Cache) Get(ctx context.Context, id string) (*entity.Game, error) {
	g, ok := c.cache.Get(id)
	if ok && g != nil {
		return g.(*entity.Game), nil
	}

	// Recheck after locking, to ensure it is still not there.
	c.Lock()
	defer c.Unlock()
	g, ok = c.cache.Get(id)
	if ok && g != nil {
		return g.(*entity.Game), nil
	}
	log.Info().Str("gameid", id).Msg("not-in-cache")
	uncachedGame, err := c.backing.Get(ctx, id)
	if err == nil {
		c.cache.Add(id, uncachedGame)
	}
	return uncachedGame, err

}

// Just call the DB implementation for now
func (c *Cache) GetRematchStreak(ctx context.Context, originalRequestId string) (*gs.GameInfoResponses, error) {
	return c.backing.GetRematchStreak(ctx, originalRequestId)
}

func (c *Cache) GetRecentGames(ctx context.Context, username string, numGames int, offset int) (*gs.GameInfoResponses, error) {
	return c.backing.GetRecentGames(ctx, username, numGames, offset)
}

// Similar to get but does not unmarshal the stats and timers and does
// not play the game
func (c *Cache) GetMetadata(ctx context.Context, id string) (*gs.GameInfoResponse, error) {
	return c.backing.GetMetadata(ctx, id)
}

// Set sets a game in the cache, AND in the backing store. This ensures if the
// node crashes the game doesn't just vanish.
func (c *Cache) Set(ctx context.Context, game *entity.Game) error {
	return c.setOrCreate(ctx, game, false)
}

// Create creates the game in the cache as well as the store.
func (c *Cache) Create(ctx context.Context, game *entity.Game) error {
	return c.setOrCreate(ctx, game, true)
}

func (c *Cache) Exists(ctx context.Context, id string) (bool, error) {
	return c.backing.Exists(ctx, id)
}

func (c *Cache) setOrCreate(ctx context.Context, game *entity.Game, isNew bool) error {
	gameID := game.History().Uid
	if gameID == "" {
		return errNoID
	}
	var err error
	if isNew {
		err = c.backing.Create(ctx, game)
	} else {
		err = c.backing.Set(ctx, game)
	}
	if err != nil {
		return err
	}
	c.cache.Add(gameID, game)
	return nil
}

func (c *Cache) ListActive(ctx context.Context) ([]*pb.GameMeta, error) {
	c.RLock()
	if time.Now().Sub(c.activeGamesLastUpdated) < c.activeGamesTTL {
		log.Debug().Msg("returning active games from cache")
		c.RUnlock()
		return c.activeGames, nil
	}
	c.RUnlock()
	log.Debug().Msg("active games not in cache, fetching from backing")

	games, err := c.backing.ListActive(ctx)
	if err == nil {
		c.Lock()
		c.activeGames = games
		c.activeGamesLastUpdated = time.Now()
		c.Unlock()
	}
	return games, err
}

func (c *Cache) Count(ctx context.Context) (int64, error) {
	return c.backing.Count(ctx)
}

func (c *Cache) Disconnect() {
	c.backing.Disconnect()
}
