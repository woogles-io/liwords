package auth

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lithammer/shortuuid/v4"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

const loginLimitTestDB = "authloginlimit"

func testQueries(t *testing.T) (*models.Queries, *pgxpool.Pool) {
	t.Helper()
	if err := common.RecreateTestDB(loginLimitTestDB); err != nil {
		t.Fatal(err)
	}
	db, err := common.OpenTestingDB(loginLimitTestDB)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return models.New(db), db
}

func testLimiter(t *testing.T) *loginLimiter {
	q, _ := testQueries(t)
	return newLoginLimiter(q)
}

func TestLoginLimitPerAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	l := testLimiter(t)
	user := "limit-" + shortuuid.New()

	for range maxFailsUserIP {
		is.True(l.allowed(ctx, user, "192.0.2.1", ""))
		l.failed(ctx, user, "192.0.2.1")
	}
	is.True(!l.allowed(ctx, user, "192.0.2.1", ""))
	// Case doesn't matter, as it doesn't for logging in.
	is.True(!l.allowed(ctx, "LIMIT"+user[5:], "192.0.2.1", ""))
	// The same account from another address is still allowed.
	is.True(l.allowed(ctx, user, "192.0.2.2", ""))

	// One address trying many accounts.
	ip := "198.51.100." + shortuuid.New()
	for i := range maxFailsIP {
		l.failed(ctx, user+string(rune('a'+i)), ip)
	}
	is.True(!l.allowed(ctx, user+"z", ip, ""))
}

func TestLoginLimitPerAccount(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	l := testLimiter(t)
	user := "limit-" + shortuuid.New()

	l.succeeded(ctx, user, "192.0.2.9", "known-browser")
	for i := range maxFailsUser {
		l.failed(ctx, user, "203.0.113."+string(rune('a'+i%26))+shortuuid.New()[:4])
	}
	is.True(!l.allowed(ctx, user, "192.0.2.10", ""))
	is.True(!l.allowed(ctx, user, "192.0.2.10", "other-browser"))
	// A browser that has logged into the account before isn't locked out.
	is.True(l.allowed(ctx, user, "192.0.2.10", "known-browser"))
}

// Requests whose address is unknown don't share a counter.
func TestLoginLimitNoAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	l := testLimiter(t)
	user := "limit-" + shortuuid.New()

	for range maxFailsIP {
		l.failed(ctx, user, "")
	}
	is.True(l.allowed(ctx, "limit-"+shortuuid.New(), "", ""))
}

func TestLoginLimitWithoutStore(t *testing.T) {
	is := is.New(t)
	var l *loginLimiter
	is.True(l.allowed(context.Background(), "anyone", "192.0.2.1", ""))
}

// A window that has ended starts again; the cleanup removes what expired.
func TestLoginLimitWindowEnds(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	q, db := testQueries(t)
	l := newLoginLimiter(q)
	user := "limit-" + shortuuid.New()

	for range maxFailsUserIP {
		l.failed(ctx, user, "192.0.2.1")
	}
	is.True(!l.allowed(ctx, user, "192.0.2.1", ""))
	// Successful login from that address clears its count.
	l.succeeded(ctx, user, "192.0.2.1", "")
	is.True(l.allowed(ctx, user, "192.0.2.1", ""))

	// Age every window past its end.
	_, err := db.Exec(ctx, `UPDATE login_failures SET expires_at = now() - interval '1 second'`)
	is.NoErr(err)
	is.NoErr(CleanupLoginLimits(q)(ctx))
	rows, err := q.GetLoginFailures(ctx, []string{"loginfail:user:" + user})
	is.NoErr(err)
	is.Equal(len(rows), 0)
}
