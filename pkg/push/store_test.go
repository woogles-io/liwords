package push

import (
	"context"
	"fmt"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	commondb "github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
)

var pkg = "push"

func recreateDB(t *testing.T) *models.Queries {
	if err := commondb.RecreateTestDB(pkg); err != nil {
		t.Fatal(err)
	}
	pool, err := commondb.OpenTestingDB(pkg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	userStore, err := user.NewDBStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []string{"alice", "bob"} {
		err := userStore.New(context.Background(), &entity.User{Username: u, Email: u + "@woogles.io", UUID: u})
		if err != nil {
			t.Fatal(err)
		}
	}
	return models.New(pool)
}

func endpoints(t *testing.T, q *models.Queries, userUUID string) []string {
	rows, err := q.GetPushSubscriptions(context.Background(), userUUID)
	if err != nil {
		t.Fatal(err)
	}
	var es []string
	for _, r := range rows {
		es = append(es, r.Endpoint)
	}
	return es
}

func TestSubscriptionsInTheDB(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	q := recreateDB(t)
	s := &Sender{settings: &Settings{}, queries: q}
	p256dh, auth := browserKeys(t)

	// Every device is kept, however many there are; the newest comes first.
	var want []string
	for i := 1; i <= 40; i++ {
		e := fmt.Sprintf("https://push.example.com/%d", i)
		is.NoErr(s.Subscribe(ctx, "alice", e, p256dh, auth))
		want = append([]string{e}, want...)
	}
	is.Equal(endpoints(t, q, "alice"), want)

	// Subscribing a device again makes it the newest.
	is.NoErr(s.Subscribe(ctx, "alice", "https://push.example.com/2", p256dh, auth))
	is.Equal(endpoints(t, q, "alice")[0], "https://push.example.com/2")
	for i := 1; i <= 40; i++ {
		if i != 4 && i != 2 {
			is.NoErr(s.Unsubscribe(ctx, "alice", fmt.Sprintf("https://push.example.com/%d", i)))
		}
	}
	is.Equal(endpoints(t, q, "alice"), []string{"https://push.example.com/2", "https://push.example.com/4"})

	// The same browser subscribed by another user moves to them, new keys
	// and all.
	newP256dh, newAuth := browserKeys(t)
	is.NoErr(s.Subscribe(ctx, "bob", "https://push.example.com/2", newP256dh, newAuth))
	is.Equal(endpoints(t, q, "alice"), []string{"https://push.example.com/4"})
	rows, err := q.GetPushSubscriptions(ctx, "bob")
	is.NoErr(err)
	is.Equal(rows, []models.GetPushSubscriptionsRow{
		{Endpoint: "https://push.example.com/2", P256dh: newP256dh, Auth: newAuth}})

	// Unsubscribing someone else's device does nothing.
	is.NoErr(s.Unsubscribe(ctx, "alice", "https://push.example.com/2"))
	is.Equal(endpoints(t, q, "bob"), []string{"https://push.example.com/2"})
	is.NoErr(s.Unsubscribe(ctx, "bob", "https://push.example.com/2"))
	is.Equal(len(endpoints(t, q, "bob")), 0)

	// A gone subscription is deleted whoever has it.
	is.NoErr(q.DeletePushSubscription(ctx, "https://push.example.com/4"))
	is.Equal(len(endpoints(t, q, "alice")), 0)

	is.Equal(s.Subscribe(ctx, "nobody", "https://push.example.com/5", p256dh, auth), errNoUser)
}
