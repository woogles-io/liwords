package soughtgame

import (
	"context"
	"errors"
	"sync"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/rs/zerolog/log"
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

	soughtGames       map[string]*entity.SoughtGame
	soughtGamesByUser map[string]*entity.SoughtGame
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		soughtGames:       make(map[string]*entity.SoughtGame),
		soughtGamesByUser: make(map[string]*entity.SoughtGame),
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
	m.soughtGamesByUser[game.Seeker()] = game
	log.Debug().Interface("by-user", m.soughtGamesByUser).Msg("set-sought-game")
	return nil
}

// Delete deletes the game by game ID.
func (m *MemoryStore) Delete(ctx context.Context, id string) error {
	m.Lock()
	defer m.Unlock()

	g, ok := m.soughtGames[id]
	if !ok {
		log.Warn().Str("game-id", id).Msg("tried-to-delete-nonexistent-game-id")
		return nil
	}

	userID := g.Seeker()
	delete(m.soughtGames, id)
	delete(m.soughtGamesByUser, userID)
	return nil
}

// DeleteForUser deletes the game by user ID.
func (m *MemoryStore) DeleteForUser(ctx context.Context, userID string) (string, error) {
	game, ok := m.soughtGamesByUser[userID]
	if !ok {
		// Do nothing, game never existed
		return "", nil
	}
	m.Lock()
	defer m.Unlock()
	delete(m.soughtGamesByUser, userID)
	delete(m.soughtGames, game.ID())
	return game.ID(), nil
}

func (m *MemoryStore) ListOpen(ctx context.Context) ([]*entity.SoughtGame, error) {
	ret := []*entity.SoughtGame{}
	for _, v := range m.soughtGames {
		ret = append(ret, v)
	}
	return ret, nil
}

func (m *MemoryStore) ExistsForUser(ctx context.Context, userID string) (bool, error) {
	_, ok := m.soughtGamesByUser[userID]
	return ok, nil
}
