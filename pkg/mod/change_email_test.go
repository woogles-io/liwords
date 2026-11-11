package mod

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

func TestChangeEmail(t *testing.T) {
	is := is.New(t)
	session := &entity.Session{
		ID:       "mnopqr",
		Username: "Moderator",
		UserUUID: "Moderator",
		Expiry:   time.Now().Add(time.Second * 100)}
	ctx := apiserver.PlaceInContext(context.Background(), session)

	recreateDB()
	us := userStore()
	defer us.(*user.DBStore).Disconnect()
	pool, err := common.OpenTestingDB(pkg)
	is.NoErr(err)
	defer pool.Close()
	q := models.New(pool)
	svc := &ModService{userStore: us, queries: q}
	is.NoErr(us.SetPersonalInfo(ctx, "Hacker", "hacker@woogles.io", fullProfile.FirstName,
		fullProfile.LastName, fullProfile.BirthDate, fullProfile.CountryCode, fullProfile.About))

	change := func(email string) error {
		_, err := svc.ChangeEmail(ctx, connect.NewRequest(&ms.ChangeEmailRequest{
			UserId: "Hacker", Email: email}))
		return err
	}
	emailIs := func(want string) {
		u, err := us.GetByUUID(ctx, "Hacker")
		is.NoErr(err)
		is.Equal(u.Email, want)
	}

	// Neither a moderator nor a manager may.
	for _, role := range []rbac.Role{rbac.Moderator, rbac.Manager} {
		is.NoErr(q.AssignRole(ctx, models.AssignRoleParams{Username: "Moderator", RoleName: string(role)}))
		is.Equal(change("new@woogles.io"), apiserver.Unauthenticated(errNotAuthorized.Error()))
		emailIs("hacker@woogles.io")
	}

	is.NoErr(q.AssignRole(ctx, models.AssignRoleParams{Username: "Moderator", RoleName: string(rbac.Admin)}))

	// An address another account has, in any case, is refused, as is one
	// that is not an address.
	for _, bad := range []string{"Deleter@woogles.io", "", "not an address", "Hacker <new@woogles.io>"} {
		is.Equal(connect.CodeOf(change(bad)), connect.CodeInvalidArgument)
		emailIs("hacker@woogles.io")
	}
	none, err := q.ListEmailChanges(ctx, "Hacker")
	is.NoErr(err)
	is.Equal(len(none), 0)

	// An admin may, and the rest of the profile stays as it was.
	is.NoErr(change(" new@woogles.io "))
	emailIs("new@woogles.io")
	u, err := us.GetByUUID(ctx, "Hacker")
	is.NoErr(err)
	is.Equal(u.Profile.FirstName, fullProfile.FirstName)
	is.Equal(u.Profile.About, fullProfile.About)

	// The change is in the email changes, by the admin, and not in the
	// mod actions.
	changes, err := q.ListEmailChanges(ctx, "Hacker")
	is.NoErr(err)
	is.Equal(len(changes), 1)
	is.Equal(changes[0].OldEmail, "hacker@woogles.io")
	is.Equal(changes[0].NewEmail, "new@woogles.io")
	is.Equal(changes[0].ChangedByUuid, "Moderator")
	is.True(changes[0].ChangedAt.Valid)
	history, err := GetActionHistory(ctx, us, "Hacker")
	is.NoErr(err)
	for _, a := range history {
		is.True(a.Type != ms.ModActionType_CHANGE_EMAIL)
	}

	// A second change lists first.
	is.NoErr(change("newer@woogles.io"))
	changes, err = q.ListEmailChanges(ctx, "Hacker")
	is.NoErr(err)
	is.Equal(len(changes), 2)
	is.Equal(changes[0].OldEmail, "new@woogles.io")
	is.Equal(changes[0].NewEmail, "newer@woogles.io")

	// It cannot go in as a plain action.
	is.True(ApplyActions(ctx, us, chatStore(), "Moderator",
		[]*ms.ModAction{{UserId: "Hacker", Type: ms.ModActionType_CHANGE_EMAIL}}) != nil)
}

func TestListEmailChanges(t *testing.T) {
	is := is.New(t)
	session := &entity.Session{
		ID:       "stuvwx",
		Username: "Moderator",
		UserUUID: "Moderator",
		Expiry:   time.Now().Add(time.Second * 100)}
	ctx := apiserver.PlaceInContext(context.Background(), session)

	recreateDB()
	us := userStore()
	defer us.(*user.DBStore).Disconnect()
	pool, err := common.OpenTestingDB(pkg)
	is.NoErr(err)
	defer pool.Close()
	q := models.New(pool)
	svc := &ModService{userStore: us, queries: q}
	is.NoErr(us.AddEmailChange(ctx, "Hacker", "old@woogles.io", "hacker@woogles.io", "Hacker"))

	list := func() (*connect.Response[ms.ListEmailChangesResponse], error) {
		return svc.ListEmailChanges(ctx, connect.NewRequest(&ms.ListEmailChangesRequest{UserId: "Hacker"}))
	}

	// A moderator may see private user data, but not this.
	is.NoErr(q.AssignRole(ctx, models.AssignRoleParams{Username: "Moderator", RoleName: string(rbac.Moderator)}))
	_, err = list()
	is.Equal(connect.CodeOf(err), connect.CodePermissionDenied)

	// An admin may.
	is.NoErr(q.AssignRole(ctx, models.AssignRoleParams{Username: "Moderator", RoleName: string(rbac.Admin)}))
	resp, err := list()
	is.NoErr(err)
	is.Equal(len(resp.Msg.Changes), 1)
	c := resp.Msg.Changes[0]
	is.Equal(c.OldEmail, "old@woogles.io")
	is.Equal(c.NewEmail, "hacker@woogles.io")
	is.Equal(c.ChangedByUserId, "Hacker")
	is.Equal(c.ChangedByUsername, "Hacker")
	is.True(time.Since(c.ChangedAt.AsTime()) < time.Minute)
}
