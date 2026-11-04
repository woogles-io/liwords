package auth

import (
	"context"
	"os"
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/lithammer/shortuuid/v4"
	"github.com/matryer/is"
)

func testLimiter(t *testing.T) *loginLimiter {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set")
	}
	return newLoginLimiter(&redis.Pool{
		Dial: func() (redis.Conn, error) { return redis.DialURL(url) },
	})
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

func TestLoginLimitWithoutRedis(t *testing.T) {
	is := is.New(t)
	var l *loginLimiter
	is.True(l.allowed(context.Background(), "anyone", "192.0.2.1", ""))
}
