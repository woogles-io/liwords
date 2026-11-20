package league

import (
	"context"
	"sync"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/user"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

// startUsers is the user store a season start uses. Each player is in about
// fourteen of a division's games, and creating and starting each game looks
// both players up: their user, whether they are a bot, and their moderator
// actions (to censor a suspended player's name). startUsers remembers each
// answer for the rest of the start, so each player is looked up once.
//
// A start takes a minute or two. A player suspended during it may have their
// name shown, not censored, in the start events of their remaining games,
// as if the suspension had come a minute later.
//
// Every other method goes to the store it wraps.
type startUsers struct {
	user.Store

	mu      sync.Mutex
	users   map[string]*entity.User
	bots    map[string]bool
	actions map[string]map[string]*ms.ModAction
}

func newStartUsers(s user.Store) *startUsers {
	return &startUsers{
		Store:   s,
		users:   map[string]*entity.User{},
		bots:    map[string]bool{},
		actions: map[string]map[string]*ms.ModAction{},
	}
}

// GetByUUID returns the same *entity.User for every call with a UUID; game
// creation only reads it.
func (s *startUsers) GetByUUID(ctx context.Context, uuid string) (*entity.User, error) {
	return remember(s, s.users, uuid, func() (*entity.User, error) { return s.Store.GetByUUID(ctx, uuid) })
}

func (s *startUsers) IsBot(ctx context.Context, uuid string) (bool, error) {
	return remember(s, s.bots, uuid, func() (bool, error) { return s.Store.IsBot(ctx, uuid) })
}

func (s *startUsers) GetActions(ctx context.Context, uuid string) (map[string]*ms.ModAction, error) {
	return remember(s, s.actions, uuid, func() (map[string]*ms.ModAction, error) { return s.Store.GetActions(ctx, uuid) })
}

// remember returns m[uuid], looking it up with get the first time. A failed
// lookup isn't remembered.
func remember[T any](s *startUsers, m map[string]T, uuid string, get func() (T, error)) (T, error) {
	s.mu.Lock()
	v, ok := m[uuid]
	s.mu.Unlock()
	if ok {
		return v, nil
	}
	v, err := get()
	if err != nil {
		return v, err
	}
	s.mu.Lock()
	m[uuid] = v
	s.mu.Unlock()
	return v, nil
}
