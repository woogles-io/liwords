package mod

import (
	"context"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

func TestAuthenticateMod(t *testing.T) {
	is := is.New(t)

	session := &entity.Session{
		ID:       "abcdef",
		Username: "Moderator",
		UserUUID: "Moderator",
		Expiry:   time.Now().Add(time.Second * 100)}
	ctx := context.Background()
	ctx = apiserver.PlaceInContext(ctx, session)

	recreateDB()
	us := userStore()

	pool, err := common.OpenTestingDB(pkg)
	is.NoErr(err)
	q := models.New(pool)
	ms := &ModService{userStore: us, queries: q}
	err = q.AssignRole(ctx, models.AssignRoleParams{
		Username: common.ToPGTypeText("Moderator"),
		RoleName: string(rbac.Moderator),
	})
	is.NoErr(err)

	_, err = authenticateMod(ctx, ms, &pb.ModActionsList{
		Actions: []*pb.ModAction{},
	})
	is.NoErr(err)
	us.(*user.DBStore).Disconnect()
	pool.Close()
}

func TestAuthenticateModNoAuth(t *testing.T) {
	is := is.New(t)

	session := &entity.Session{
		ID:       "defghi",
		Username: "Cheater",
		UserUUID: "Cheater",
		Expiry:   time.Now().Add(time.Second * 100)}
	ctx := context.Background()
	ctx = apiserver.PlaceInContext(ctx, session)

	recreateDB()
	us := userStore()

	pool, err := common.OpenTestingDB(pkg)
	is.NoErr(err)
	q := models.New(pool)
	ms := &ModService{userStore: us, queries: q}

	_, err = authenticateMod(ctx, ms, &pb.ModActionsList{
		Actions: []*pb.ModAction{},
	})
	is.Equal(err, apiserver.Unauthenticated(errNotAuthorized.Error()))
	us.(*user.DBStore).Disconnect()
	pool.Close()
}
