package auth

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	commondb "github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

var pkg = "auth"

var DefaultConfig = config.DefaultConfig()

type DBController struct {
	pool *pgxpool.Pool
	us   *user.DBStore
	q    *models.Queries
}

func (dbc *DBController) cleanup() {
	dbc.us.Disconnect()
	dbc.pool.Close()
}

func RecreateDB() *DBController {
	cfg := DefaultConfig
	cfg.DBConnUri = commondb.TestingPostgresConnUri(pkg)
	err := commondb.RecreateTestDB(pkg)
	if err != nil {
		panic(err)
	}
	// Reconnect to the new test database
	pool, err := commondb.OpenTestingDB(pkg)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	userStore, err := user.NewDBStore(pool)
	if err != nil {
		panic(err)
	}
	err = userStore.New(context.Background(), &entity.User{Username: "SomeAdmin", Email: "someadmin@woogles.io", UUID: "someadmin"})
	if err != nil {
		panic(err)
	}
	_, err = userStore.ResetAPIKey(ctx, "someadmin")
	if err != nil {
		panic(err)
	}
	err = userStore.New(context.Background(), &entity.User{Username: "NotAnAdmin", Email: "notadmin@woogles.io", UUID: "notadmin"})
	if err != nil {
		panic(err)
	}
	_, err = userStore.ResetAPIKey(ctx, "notadmin")
	if err != nil {
		panic(err)
	}
	err = userStore.New(context.Background(), &entity.User{Username: "SomeMod", Email: "mod@woogles.io", UUID: "somemod"})
	if err != nil {
		panic(err)
	}

	q := models.New(pool)
	err = q.AssignRole(ctx, models.AssignRoleParams{
		Username: pgtype.Text{String: "SomeAdmin", Valid: true},
		RoleName: string(rbac.Admin),
	})
	if err != nil {
		panic(err)
	}
	err = q.AssignRole(ctx, models.AssignRoleParams{
		Username: pgtype.Text{String: "SomeMod", Valid: true},
		RoleName: string(rbac.Moderator),
	})
	if err != nil {
		panic(err)
	}
	return &DBController{pool: pool, q: q, us: userStore}
}

func TestAssignRole(t *testing.T) {
	is := is.New(t)
	dbc := RecreateDB()
	defer func() {
		dbc.cleanup()
	}()
	ctx := context.Background()
	svc := NewAuthorizationService(dbc.us, dbc.q)

	// Try assigning a role to themselves, if not an admin
	_, err := svc.AssignRole(ctx, connect.NewRequest(&pb.AssignRoleRequest{
		Username: "NotAnAdmin",
		RoleName: string(rbac.SpecialAccessPlayer),
	}))
	is.Equal(err.Error(), "unauthenticated: auth-methods-failed")

	apikey, err := dbc.us.GetAPIKey(context.Background(), "notadmin")
	is.NoErr(err)
	ctx = apiserver.StoreAPIKeyInContext(ctx, apikey)

	_, err = svc.AssignRole(ctx, connect.NewRequest(&pb.AssignRoleRequest{
		Username: "NotAnAdmin",
		RoleName: string(rbac.SpecialAccessPlayer),
	}))
	is.Equal(err.Error(), "permission_denied: not an admin")

	// Now try it with the admin.
	apikey, err = dbc.us.GetAPIKey(context.Background(), "someadmin")
	is.NoErr(err)
	ctx = apiserver.StoreAPIKeyInContext(ctx, apikey)

	_, err = svc.AssignRole(ctx, connect.NewRequest(&pb.AssignRoleRequest{
		Username: "NotAnAdmin",
		RoleName: string(rbac.SpecialAccessPlayer),
	}))
	is.NoErr(err)

	resp, err := svc.GetUserRoles(ctx, connect.NewRequest(&pb.GetUserRolesRequest{
		Username: "NotAnAdmin",
	}))
	is.NoErr(err)
	is.Equal(resp.Msg.Roles, []string{string(rbac.SpecialAccessPlayer)})
	// Try assigning the same role again. It should fail.
	_, err = svc.AssignRole(ctx, connect.NewRequest(&pb.AssignRoleRequest{
		Username: "NotAnAdmin",
		RoleName: string(rbac.SpecialAccessPlayer),
	}))
	is.Equal(err.Error(), "already_exists: role already assigned to user")
}

func TestGetAdminsAndMods(t *testing.T) {
	is := is.New(t)
	dbc := RecreateDB()
	defer func() {
		dbc.cleanup()
	}()
	ctx := context.Background()
	svc := NewAuthorizationService(dbc.us, dbc.q)
	resp, err := svc.GetModList(ctx, connect.NewRequest(&pb.GetModListRequest{}))
	is.NoErr(err)
	is.Equal(resp.Msg.AdminUserIds, []string{"someadmin"})
	is.Equal(resp.Msg.ModUserIds, []string{"somemod"})
}

func TestUnassignRole(t *testing.T) {
	is := is.New(t)
	dbc := RecreateDB()
	defer func() {
		dbc.cleanup()
	}()
	ctx := context.Background()
	svc := NewAuthorizationService(dbc.us, dbc.q)

	apikey, err := dbc.us.GetAPIKey(context.Background(), "someadmin")
	is.NoErr(err)
	ctx = apiserver.StoreAPIKeyInContext(ctx, apikey)

	_, err = svc.UnassignRole(ctx, connect.NewRequest(&pb.UnassignRoleRequest{
		Username: "SomeMod",
		RoleName: string(rbac.Moderator),
	}))
	is.NoErr(err)

	resp, err := svc.GetUserRoles(ctx, connect.NewRequest(&pb.GetUserRolesRequest{
		Username: "SomeMod",
	}))
	is.NoErr(err)
	is.Equal(resp.Msg.Roles, []string{})

	_, err = svc.UnassignRole(ctx, connect.NewRequest(&pb.UnassignRoleRequest{
		Username: "SomeMod",
		RoleName: string(rbac.Moderator),
	}))
	is.Equal(err.Error(), "not_found: role assignment not found")
}
