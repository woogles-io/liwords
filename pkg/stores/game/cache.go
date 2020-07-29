package game

import (
	"context"
	"sync"

	"github.com/domino14/liwords/pkg/entity"
)

// same as the GameStore in gameplay package, but this gives us a bit more flexibility
// in defining the backing store (i.e. it may not necessarily be a SQL db store)
type backingStore interface {
	Get(ctx context.Context, id string) (*entity.Game, error)
	Set(context.Context, *entity.Game) error
	Create(context.Context, *entity.Game) error
}

// Cache will reside in-memory, and will be per-node. If we add more nodes
// we will need to make sure only the right nodes respond to game requests.
type Cache struct {
	sync.Mutex

	games   map[string]*entity.Game
	backing backingStore
}

func NewCache(backing backingStore) *Cache {
	return &Cache{
		backing: backing,
		games:   make(map[string]*entity.Game),
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

// Get gets a game from the cache. It doesn't try to get it from the backing
// store, if it can't find it in the cache.
func (c *Cache) Get(ctx context.Context, id string) (*entity.Game, error) {
	g, ok := c.games[id]
	if !ok {
		return nil, errNotFound
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
