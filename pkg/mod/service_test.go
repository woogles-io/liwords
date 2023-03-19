package mod

import (
	"context"
	"testing"
	"time"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/user"
	pb "github.com/domino14/liwords/rpc/api/proto/mod_service"
	"github.com/matryer/is"
	"github.com/twitchtv/twirp"
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
