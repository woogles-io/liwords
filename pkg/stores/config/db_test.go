package config

import (
	"context"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/common"
	pb "github.com/woogles-io/liwords/rpc/api/proto/config_service"
)

const pkg = "configstore"

func setup(t *testing.T) *DBStore {
	t.Helper()
	if err := common.RecreateTestDB(pkg); err != nil {
		t.Fatal(err)
	}
	db, err := common.OpenTestingDB(pkg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return NewDBStore(db)
}

// Unset, the settings have their defaults: games on, no hash, no
// announcements.
func TestDefaults(t *testing.T) {
	is := is.New(t)
	s := setup(t)
	ctx := context.Background()
	on, err := s.GamesEnabled(ctx)
	is.NoErr(err)
	is.True(on)
	h, err := s.FEHash(ctx)
	is.NoErr(err)
	is.Equal(h, "")
	a, err := s.GetAnnouncements(ctx)
	is.NoErr(err)
	is.Equal(len(a), 0)
}

func TestSettings(t *testing.T) {
	is := is.New(t)
	s := setup(t)
	ctx := context.Background()

	is.NoErr(s.SetGamesEnabled(ctx, false))
	on, _ := s.GamesEnabled(ctx)
	is.True(!on)
	is.NoErr(s.SetGamesEnabled(ctx, true))
	on, _ = s.GamesEnabled(ctx)
	is.True(on)

	is.NoErr(s.SetFEHash(ctx, "abc"))
	is.NoErr(s.SetFEHash(ctx, "def"))
	h, err := s.FEHash(ctx)
	is.NoErr(err)
	is.Equal(h, "def")

	is.NoErr(s.SetAnnouncements(ctx, []*pb.Announcement{
		{Title: "Board", Link: blog + "/posts/board/", Pinned: true},
		{Title: "Recap", Link: blog + "/posts/recap/"},
	}))
	is.NoErr(s.SetAnnouncement(ctx, blog, &pb.Announcement{Title: "New", Link: blog + "/posts/new/"}))
	a, err := s.GetAnnouncements(ctx)
	is.NoErr(err)
	is.Equal(a[0].Title, "Board")
	is.Equal(a[1].Title, "New")
	is.True(s.SetAnnouncement(ctx, "https://nowhere", &pb.Announcement{}) != nil)
}
