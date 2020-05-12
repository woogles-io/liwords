package game

import (
	"context"
	"errors"
	"sync"

	"github.com/domino14/crosswords/pkg/entity"
)

var (
	errNoID     = errors.New("game ID was not defined")
	errNotFound = errors.New("that game ID was not found")
)

// MemoryStore is a purely in-memory store of a game. In the real final
// implementation, we will probably use a Postgres-backed memory store.
// Due to the nature of this app, we will always need to have a persistent
// in-memory instantiation of a game.
type MemoryStore struct {
	sync.Mutex

	games map[string]*entity.Game
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		games: make(map[string]*entity.Game),
	}
}

// Get gets the game with the given ID.
func (m *MemoryStore) Get(ctx context.Context, id string) (*entity.Game, error) {
	g, ok := m.games[id]
	if !ok {
		return nil, errNotFound
	}
	return g, nil
}

// Set sets the game in the store.
func (m *MemoryStore) Set(ctx context.Context, game *entity.Game) error {
	gameID := game.History().Uid
	if gameID == "" {
		return errNoID
	}
	m.Lock()
	defer m.Unlock()
	m.games[gameID] = game
	return nil
}
