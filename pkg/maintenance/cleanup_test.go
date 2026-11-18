package maintenance

import (
	"context"
	"os"
	"testing"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/common"
)

const testDBPkg = "maintenance"

func setup(t *testing.T) (*Tasks, *stores.Stores) {
	t.Helper()
	if os.Getenv("TEST_DB_HOST") == "" {
		t.Skip("no test database (TEST_DB_HOST)")
	}
	if err := common.RecreateTestDB(testDBPkg); err != nil {
		t.Fatal(err)
	}
	pool, err := common.OpenTestingDB(testDBPkg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	cfg := config.DefaultConfig()
	s, err := stores.NewInitializedStores(pool, nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return New(cfg, s, nil, nil), s
}

// The cleanups' SQL runs against the real schema.
func TestCleanupsRun(t *testing.T) {
	tasks, _ := setup(t)
	ctx := context.Background()
	for name, f := range map[string]func(context.Context) error{
		"cancelled games":  tasks.CancelledGamesCleanup,
		"monitoring":       tasks.MonitoringStreamsCleanup,
		"expired sessions": tasks.ExpiredSessionsCleanup,
		"unverified users": tasks.UnverifiedUsersCleanup,
	} {
		if err := f(ctx); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// An unverified account more than 48 hours old goes, with the rows on both
// sides of its followings and blockings; a verified one stays.
func TestUnverifiedUsersCleanup(t *testing.T) {
	tasks, s := setup(t)
	ctx := context.Background()
	for _, u := range []*entity.User{
		{Username: "stale", Email: "stale@example.com", UUID: "stale-uuid"},
		{Username: "kept", Email: "kept@example.com", UUID: "kept-uuid"},
	} {
		if err := s.UserStore.New(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	exec := func(sql string) {
		t.Helper()
		if _, err := s.DBPool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	exec(`UPDATE users SET verified = (username = 'kept'), created_at = NOW() - INTERVAL '3 days'`)
	exec(`INSERT INTO followings (user_id, follower_id)
		SELECT a.id, b.id FROM users a, users b WHERE a.username = 'kept' AND b.username = 'stale'`)
	exec(`INSERT INTO blockings (user_id, blocker_id)
		SELECT a.id, b.id FROM users a, users b WHERE a.username = 'stale' AND b.username = 'kept'`)

	if err := tasks.UnverifiedUsersCleanup(ctx); err != nil {
		t.Fatal(err)
	}
	var names []string
	rows, err := s.DBPool.Query(ctx, `SELECT username FROM users ORDER BY username`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		names = append(names, n)
	}
	var follows, blocks int
	if err := s.DBPool.QueryRow(ctx, `SELECT (SELECT count(*) FROM followings), (SELECT count(*) FROM blockings)`).Scan(&follows, &blocks); err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if n == "stale" {
			t.Fatalf("unverified user kept: %v", names)
		}
	}
	if follows != 0 || blocks != 0 {
		t.Fatalf("left %d followings, %d blockings", follows, blocks)
	}
}
