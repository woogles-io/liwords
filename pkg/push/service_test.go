package push

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/push_service"
)

func signedIn(userUUID string) context.Context {
	return apiserver.PlaceInContext(context.Background(),
		&entity.Session{Username: userUUID, UserUUID: userUUID})
}

func TestServiceIsOffWithoutSettings(t *testing.T) {
	is := is.New(t)
	svc := NewService(nil)
	ctx := signedIn("alice")
	p256dh, auth := browserKeys(t)

	resp, err := svc.GetPushConfig(ctx, connect.NewRequest(&pb.GetPushConfigRequest{}))
	is.NoErr(err)
	is.Equal(resp.Msg.Enabled, false)
	is.Equal(resp.Msg.PublicKey, "")

	_, err = svc.Subscribe(ctx, connect.NewRequest(&pb.SubscribeRequest{
		Endpoint: "https://push.example.com/1", P256Dh: p256dh, Auth: auth}))
	is.Equal(connect.CodeOf(err), connect.CodeFailedPrecondition)
	is.Equal(err.(*connect.Error).Message(), "push not configured")

	_, err = svc.Unsubscribe(ctx, connect.NewRequest(&pb.UnsubscribeRequest{
		Endpoint: "https://push.example.com/1"}))
	is.Equal(connect.CodeOf(err), connect.CodeFailedPrecondition)
}

func TestServiceSubscribesTheSignedInUser(t *testing.T) {
	is := is.New(t)
	q := recreateDB(t)
	settings := &Settings{PublicKey: "the-public-key"}
	svc := NewService(&Sender{settings: settings, queries: q})
	alice, bob := signedIn("alice"), signedIn("bob")
	p256dh, auth := browserKeys(t)
	subscribe := func(ctx context.Context, endpoint string) error {
		_, err := svc.Subscribe(ctx, connect.NewRequest(&pb.SubscribeRequest{
			Endpoint: endpoint, P256Dh: p256dh, Auth: auth}))
		return err
	}
	unsubscribe := func(ctx context.Context, endpoint string) error {
		_, err := svc.Unsubscribe(ctx, connect.NewRequest(&pb.UnsubscribeRequest{Endpoint: endpoint}))
		return err
	}

	resp, err := svc.GetPushConfig(context.Background(), connect.NewRequest(&pb.GetPushConfigRequest{}))
	is.NoErr(err)
	is.Equal(resp.Msg.Enabled, true)
	is.Equal(resp.Msg.PublicKey, "the-public-key")

	is.NoErr(subscribe(alice, "https://push.example.com/a"))
	is.NoErr(subscribe(bob, "https://push.example.com/b"))
	is.Equal(endpoints(t, q, "alice"), []string{"https://push.example.com/a"})
	is.Equal(endpoints(t, q, "bob"), []string{"https://push.example.com/b"})

	// Nobody signed in, a bad endpoint, or bad keys: nothing kept.
	is.Equal(connect.CodeOf(subscribe(context.Background(), "https://push.example.com/c")),
		connect.CodeUnauthenticated)
	is.Equal(connect.CodeOf(subscribe(alice, "http://push.example.com/c")), connect.CodeInvalidArgument)
	_, err = svc.Subscribe(alice, connect.NewRequest(&pb.SubscribeRequest{
		Endpoint: "https://push.example.com/c", P256Dh: p256dh, Auth: "short"}))
	is.Equal(connect.CodeOf(err), connect.CodeInvalidArgument)
	is.Equal(endpoints(t, q, "alice"), []string{"https://push.example.com/a"})

	// A device bob subscribed, subscribed again by alice (the next person
	// at it), moves to alice and leaves bob.
	is.NoErr(subscribe(bob, "https://push.example.com/shared"))
	is.NoErr(subscribe(alice, "https://push.example.com/shared"))
	is.Equal(endpoints(t, q, "alice"), []string{"https://push.example.com/shared", "https://push.example.com/a"})
	is.Equal(endpoints(t, q, "bob"), []string{"https://push.example.com/b"})
	is.NoErr(unsubscribe(alice, "https://push.example.com/shared"))

	// A user can only unsubscribe their own device.
	is.NoErr(unsubscribe(alice, "https://push.example.com/b"))
	is.Equal(endpoints(t, q, "bob"), []string{"https://push.example.com/b"})
	is.Equal(connect.CodeOf(unsubscribe(context.Background(), "https://push.example.com/b")),
		connect.CodeUnauthenticated)
	is.Equal(endpoints(t, q, "bob"), []string{"https://push.example.com/b"})

	is.NoErr(unsubscribe(alice, "https://push.example.com/a"))
	is.Equal(len(endpoints(t, q, "alice")), 0)
	is.NoErr(unsubscribe(bob, "https://push.example.com/b"))
	is.Equal(len(endpoints(t, q, "bob")), 0)
}
