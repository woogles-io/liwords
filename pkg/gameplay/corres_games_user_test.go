package gameplay

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/user"
)

// usersByName answers Get only; any other user.Store method would panic on
// the nil embedded interface.
type usersByName struct {
	user.Store
	users map[string]*entity.User
}

func (s *usersByName) Get(ctx context.Context, username string) (*entity.User, error) {
	if u, ok := s.users[username]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}

func TestCorrespondenceGamesUser(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	viewer := &entity.User{Username: "Mod", UUID: "mod-uuid"}
	cesar := &entity.User{Username: "cesar", UUID: "cesar-uuid"}
	store := &usersByName{users: map[string]*entity.User{"cesar": cesar}}
	asked := 0
	permission := func(ok bool, err error) func() (bool, error) {
		return func() (bool, error) {
			asked++
			return ok, err
		}
	}

	// The viewer's own games need no permission, by name or without one.
	u, err := correspondenceGamesUser(ctx, store, viewer, "", permission(false, nil))
	is.NoErr(err)
	is.Equal(u, viewer)
	u, err = correspondenceGamesUser(ctx, store, viewer, "mod", permission(false, nil))
	is.NoErr(err)
	is.Equal(u, viewer)
	is.Equal(asked, 0)

	// Someone else's games only with the permission.
	u, err = correspondenceGamesUser(ctx, store, viewer, "cesar", permission(true, nil))
	is.NoErr(err)
	is.Equal(u, cesar)

	_, err = correspondenceGamesUser(ctx, store, viewer, "cesar", permission(false, nil))
	is.Equal(connect.CodeOf(err), connect.CodePermissionDenied)

	_, err = correspondenceGamesUser(ctx, store, viewer, "nobody", permission(true, nil))
	is.Equal(connect.CodeOf(err), connect.CodeInvalidArgument)

	_, err = correspondenceGamesUser(ctx, store, viewer, "cesar", permission(false, errors.New("db down")))
	is.Equal(connect.CodeOf(err), connect.CodeInternal)
}
