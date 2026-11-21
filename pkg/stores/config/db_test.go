package config

import (
	"context"
	"os"
	"testing"

	"github.com/gomodule/redigo/redis"
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

// The settings in Redis are copied over once; what Postgres has wins.
func TestImportFromRedis(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set")
	}
	is := is.New(t)
	s := setup(t)
	ctx := context.Background()
	pool := &redis.Pool{Dial: func() (redis.Conn, error) {
		c, err := redis.DialURL(url)
		if err == nil {
			_, err = c.Do("SELECT", 13)
		}
		return c, err
	}}
	conn := pool.Get()
	defer conn.Close()
	_, err := conn.Do("FLUSHDB")
	is.NoErr(err)
	_, err = conn.Do("MSET", GamesDisabledKey, "1", AnnouncementsKey, `[{"title":"From Redis"}]`)
	is.NoErr(err)
	is.NoErr(s.SetFEHash(ctx, "already-here"))
	_, err = conn.Do("SET", FEHashKey, "old")
	is.NoErr(err)

	is.NoErr(ImportFromRedis(ctx, s, pool))
	on, _ := s.GamesEnabled(ctx)
	is.True(!on)
	a, err := s.GetAnnouncements(ctx)
	is.NoErr(err)
	is.Equal(a[0].Title, "From Redis")
	h, _ := s.FEHash(ctx)
	is.Equal(h, "already-here")

	// Changed since: a second import leaves it.
	is.NoErr(s.SetGamesEnabled(ctx, true))
	is.NoErr(ImportFromRedis(ctx, s, pool))
	on, _ = s.GamesEnabled(ctx)
	is.True(on)
	_, _ = conn.Do("FLUSHDB")
}
