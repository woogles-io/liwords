package game

import (
	"context"
	"sync"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/rs/zerolog/log"
)

// same as the GameStore in gameplay package, but this gives us a bit more flexibility
// in defining the backing store (i.e. it may not necessarily be a SQL db store)
type backingStore interface {
	Get(ctx context.Context, id string) (*entity.Game, error)
	Set(context.Context, *entity.Game) error
	Create(context.Context, *entity.Game) error
	ListActive(ctx context.Context) ([]*pb.GameMeta, error)
	SetGameEventChan(ch chan<- *entity.EventWrapper)
}

// Cache will reside in-memory, and will be per-node. If we add more nodes
// we will need to make sure only the right nodes respond to game requests.
type Cache struct {
	sync.Mutex

	games       map[string]*entity.Game
	activeGames []*pb.GameMeta

	activeGamesTTL         time.Duration
	activeGamesLastUpdated time.Time

	backing backingStore
}

func NewCache(backing backingStore) *Cache {
	return &Cache{
		backing: backing,
		games:   make(map[string]*entity.Game),
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
		// few things (especially game metadata).
		activeGamesTTL: time.Second * 5,
	}
}

// Exists lets us know whether the game is in the cache
func (c *Cache) Exists(ctx context.Context, id string) bool {
	_, ok := c.games[id]
	return ok
}

// Unload unloads the game from the cache
func (c *Cache) Unload(ctx context.Context, id string) {
	delete(c.games, id)
}

// SetGameEventChan sets the game event channel to the passed in channel.
func (c *Cache) SetGameEventChan(ch chan<- *entity.EventWrapper) {
	c.backing.SetGameEventChan(ch)
}

// Get gets a game from the cache. It doesn't try to get it from the backing
// store, if it can't find it in the cache.
func (c *Cache) Get(ctx context.Context, id string) (*entity.Game, error) {
	g, ok := c.games[id]
	if !ok {
		// Don't store it in the cache. If we are here, the only reason
		// we wouldn't be able to load from cache is that the game is either
		// over, or it's found in another API node -- in the latter case,
		// this function should ideally not be called. We should create an
		// InCache function.
		return c.backing.Get(ctx, id)
	}
	return g, nil
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
	c.Lock()
	defer c.Unlock()
	c.games[gameID] = game
	return nil
}

func (c *Cache) ListActive(ctx context.Context) ([]*pb.GameMeta, error) {

	if time.Now().Sub(c.activeGamesLastUpdated) < c.activeGamesTTL {
		log.Debug().Msg("returning active games from cache")
		return c.activeGames, nil
	}
	log.Debug().Msg("active games not in cache, fetching from backing")

	games, err := c.backing.ListActive(ctx)
	if err == nil {
		c.activeGames = games
		c.activeGamesLastUpdated = time.Now()
	}
	return games, err
}
