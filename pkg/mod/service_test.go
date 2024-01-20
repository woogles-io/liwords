package mod

import (
	"context"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/twitchtv/twirp"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
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
	ms := &ModService{userStore: us}

	_, err := authenticateMod(ctx, ms, &pb.ModActionsList{
		Actions: []*pb.ModAction{},
	})
	is.NoErr(err)
	us.(*user.DBStore).Disconnect()
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
	ms := &ModService{userStore: us}

	_, err := authenticateMod(ctx, ms, &pb.ModActionsList{
		Actions: []*pb.ModAction{},
	})
	is.Equal(err, twirp.NewError(twirp.Unauthenticated, errNotAuthorized.Error()))
	us.(*user.DBStore).Disconnect()
}
