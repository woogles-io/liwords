package soughtgame

import (
	"context"
	"errors"
	"sync"

	"github.com/domino14/crosswords/pkg/entity"
)

var (
	errNoID     = errors.New("sought game ID was not defined")
	errNotFound = errors.New("sought game ID was not found")
)

// MemoryStore is a purely in-memory store of a sought game. In the real final
// implementation, we will probably use a Postgres-backed store, or perhaps
// something in Redis.
type MemoryStore struct {
	sync.Mutex

	soughtGames map[string]*entity.SoughtGame
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		soughtGames: make(map[string]*entity.SoughtGame),
	}
}

// Get gets the game with the given ID.
func (m *MemoryStore) Get(ctx context.Context, id string) (*entity.SoughtGame, error) {
	g, ok := m.soughtGames[id]
	if !ok {
		return nil, errNotFound
	}
	return g, nil
}

// Set sets the game in the store.
func (m *MemoryStore) Set(ctx context.Context, game *entity.SoughtGame) error {
	m.Lock()
	defer m.Unlock()
	m.soughtGames[game.ID()] = game
	return nil
}

func (m *MemoryStore) Delete(ctx context.Context, id string) error {
	m.Lock()
	defer m.Unlock()
	delete(m.soughtGames, id)
	return nil
}

func (m *MemoryStore) ListOpen(ctx context.Context) ([]*entity.SoughtGame, error) {
	ret := []*entity.SoughtGame{}
	for _, v := range m.soughtGames {
		ret = append(ret, v)
	}
	return ret, nil
}
