package league

import (
	"context"
	"errors"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/user"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

// countingUsers counts the lookups that reach it, and fails those for UUIDs
// in fail. Its other methods are the wrapped store's: nil in the unit tests,
// which don't call them.
type countingUsers struct {
	user.Store
	getByUUID, isBot, getActions map[string]int
	fail                         map[string]bool
}

func newCountingUsers(s user.Store) *countingUsers {
	return &countingUsers{Store: s, getByUUID: map[string]int{}, isBot: map[string]int{},
		getActions: map[string]int{}, fail: map[string]bool{}}
}

func (c *countingUsers) GetByUUID(ctx context.Context, uuid string) (*entity.User, error) {
	c.getByUUID[uuid]++
	if c.fail[uuid] {
		return nil, errors.New("db down")
	}
	if c.Store == nil {
		return &entity.User{UUID: uuid, IsBot: uuid == "bot"}, nil
	}
	return c.Store.GetByUUID(ctx, uuid)
}

func (c *countingUsers) IsBot(ctx context.Context, uuid string) (bool, error) {
	c.isBot[uuid]++
	if c.fail[uuid] {
		return false, errors.New("db down")
	}
	if c.Store == nil {
		return uuid == "bot", nil
	}
	return c.Store.IsBot(ctx, uuid)
}

func (c *countingUsers) GetActions(ctx context.Context, uuid string) (map[string]*ms.ModAction, error) {
	c.getActions[uuid]++
	if c.fail[uuid] {
		return nil, errors.New("db down")
	}
	if c.Store == nil {
		return map[string]*ms.ModAction{}, nil
	}
	return c.Store.GetActions(ctx, uuid)
}

func TestStartUsersLooksEachUpOnce(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	under := newCountingUsers(nil)
	s := newStartUsers(under)

	for range 3 {
		for _, id := range []string{"a", "b", "bot"} {
			u, err := s.GetByUUID(ctx, id)
			is.NoErr(err)
			is.Equal(u.UUID, id)
			_, err = s.GetActions(ctx, id)
			is.NoErr(err)
		}
	}
	for _, id := range []string{"a", "b", "bot"} {
		is.Equal(under.getByUUID[id], 1)
		is.Equal(under.getActions[id], 1)
	}

	a1, _ := s.GetByUUID(ctx, "a")
	a2, _ := s.GetByUUID(ctx, "a")
	is.True(a1 == a2) // the same user each time
}

// A failed lookup is an error each time it is asked, until one succeeds: it
// isn't remembered as an empty answer.
func TestStartUsersDoesNotRememberFailures(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	under := newCountingUsers(nil)
	under.fail["a"] = true
	s := newStartUsers(under)

	_, err := s.GetByUUID(ctx, "a")
	is.True(err != nil)
	_, err = s.GetActions(ctx, "a")
	is.True(err != nil)

	under.fail["a"] = false
	u, err := s.GetByUUID(ctx, "a")
	is.NoErr(err)
	is.Equal(u.UUID, "a")
	_, err = s.GetActions(ctx, "a")
	is.NoErr(err)
	is.Equal(under.getByUUID["a"], 2)
	is.Equal(under.getActions["a"], 2)

	_, err = s.GetByUUID(ctx, "a")
	is.NoErr(err)
	is.Equal(under.getByUUID["a"], 2)
}
