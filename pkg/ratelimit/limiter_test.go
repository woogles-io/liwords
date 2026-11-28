package ratelimit

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/apiserver"
)

var apiserverClientInfo = apiserver.ClientInfoMiddlewareGenerator(false)

// These tests run the Lua script on a real server: REDIS_URL, e.g.
// redis://localhost:16379 with the docker compose stack up.
func testLimiter(t *testing.T) (*Limiter, string) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set")
	}
	prefix := fmt.Sprintf("rltest:%d:", rand.Int63())
	l := NewLimiter(url)
	l.timeout = 2 * time.Second // CI machines can be slow
	t.Cleanup(func() {
		conn := l.pool.Get()
		defer conn.Close()
		keys, _ := redis.Strings(conn.Do("KEYS", prefix+"*"))
		for _, k := range keys {
			conn.Do("DEL", k)
		}
		l.pool.Close()
	})
	return l, prefix
}

func TestBurstThenRefused(t *testing.T) {
	is := is.New(t)
	l, p := testLimiter(t)
	ctx := context.Background()
	c := []Check{{Key: p + "a", Limit: Limit{Rate: 60, Per: time.Minute, Burst: 3}}}
	for i := 0; i < 3; i++ {
		res, err := l.Allow(ctx, c)
		is.NoErr(err)
		is.True(res.Allowed) // within the burst
	}
	res, err := l.Allow(ctx, c)
	is.NoErr(err)
	is.True(!res.Allowed)
	// One token comes back every second.
	is.True(res.RetryAfter > 0)
	is.True(res.RetryAfter <= time.Second)
}

func TestRefills(t *testing.T) {
	is := is.New(t)
	l, p := testLimiter(t)
	ctx := context.Background()
	c := []Check{{Key: p + "a", Limit: Limit{Rate: 20, Per: time.Second, Burst: 1}}} // one per 50 ms
	res, err := l.Allow(ctx, c)
	is.NoErr(err)
	is.True(res.Allowed)
	res, err = l.Allow(ctx, c)
	is.NoErr(err)
	is.True(!res.Allowed)
	time.Sleep(80 * time.Millisecond)
	res, err = l.Allow(ctx, c)
	is.NoErr(err)
	is.True(res.Allowed) // refilled
}

// A request refused by one bucket takes nothing from the others.
func TestRefusedTakesNothing(t *testing.T) {
	is := is.New(t)
	l, p := testLimiter(t)
	ctx := context.Background()
	small := Check{Key: p + "small", Limit: Limit{Rate: 1, Per: time.Hour, Burst: 1}}
	big := Check{Key: p + "big", Limit: Limit{Rate: 5, Per: time.Hour, Burst: 5}}

	res, err := l.Allow(ctx, []Check{small, big})
	is.NoErr(err)
	is.True(res.Allowed) // small 0 left, big 4 left
	for i := 0; i < 10; i++ {
		res, err = l.Allow(ctx, []Check{small, big})
		is.NoErr(err)
		is.True(!res.Allowed) // small is empty
	}
	for i := 0; i < 4; i++ {
		res, err = l.Allow(ctx, []Check{big})
		is.NoErr(err)
		is.True(res.Allowed) // big still had 4
	}
	res, err = l.Allow(ctx, []Check{big})
	is.NoErr(err)
	is.True(!res.Allowed)
}

// Keys don't outlive the time their bucket takes to fill up again.
func TestKeysExpire(t *testing.T) {
	is := is.New(t)
	l, p := testLimiter(t)
	ctx := context.Background()
	lim := Limit{Rate: 60, Per: time.Minute, Burst: 10}
	for i := 0; i < 3; i++ {
		_, err := l.Allow(ctx, []Check{{Key: p + "a", Limit: lim}})
		is.NoErr(err)
	}
	conn := l.pool.Get()
	defer conn.Close()
	pttl, err := redis.Int64(conn.Do("PTTL", p+"a"))
	is.NoErr(err)
	// Three tokens taken, one second each, plus the one-second margin.
	is.True(pttl > 3000)
	is.True(pttl <= 4000)
}

func TestOffSwitch(t *testing.T) {
	is := is.New(t)
	l, p := testLimiter(t)
	ctx := context.Background()
	conn := l.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", "rl:off", "1")
	is.NoErr(err)
	defer conn.Do("DEL", "rl:off")
	c := []Check{{Key: p + "a", Limit: Limit{Rate: 1, Per: time.Hour, Burst: 1}}}
	for i := 0; i < 5; i++ {
		res, err := l.Allow(ctx, c)
		is.NoErr(err)
		is.True(res.Allowed) // over the limit, but switched off
	}
	_, err = conn.Do("DEL", "rl:off")
	is.NoErr(err)
	res, err := l.Allow(ctx, c)
	is.NoErr(err)
	is.True(res.Allowed) // nothing was counted while off
	res, err = l.Allow(ctx, c)
	is.NoErr(err)
	is.True(!res.Allowed)
}

func TestNoChecksAllowed(t *testing.T) {
	is := is.New(t)
	l := &Limiter{} // never touches a server
	res, err := l.Allow(context.Background(), nil)
	is.NoErr(err)
	is.True(res.Allowed)
}

// End to end with the real script: one browser runs out of analyses, another
// browser on the same IP doesn't notice.
func TestMiddlewareWithValkey(t *testing.T) {
	is := is.New(t)
	l, _ := testLimiter(t)
	ip := fmt.Sprintf("10.%d.%d.%d", rand.Intn(250), rand.Intn(250), rand.Intn(250)) // fresh keys
	t.Cleanup(func() {
		conn := l.pool.Get()
		defer conn.Close()
		keys, _ := redis.Strings(conn.Do("KEYS", "rl:*"+ip+"*"))
		for _, k := range keys {
			conn.Do("DEL", k)
		}
	})
	h := apiserverClientInfo(Middleware(l, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	status := func(cid string) int {
		r := withBrowser(request(analysisPath, ip), cid)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	for i := 0; i < heavyBucket.Anon.Burst; i++ {
		is.Equal(status(kidA), http.StatusOK) // within the burst
	}
	is.Equal(status(kidA), http.StatusTooManyRequests)
	is.Equal(status(kidB), http.StatusOK) // another browser on the same IP
}

// Someone who retries a refused sign-up gets ten tries before waiting, and
// the retries don't use up the email bucket that resending verification
// emails draws from.
func TestSignupRetriesWithValkey(t *testing.T) {
	is := is.New(t)
	l, _ := testLimiter(t)
	ip := fmt.Sprintf("10.%d.%d.%d", rand.Intn(250), rand.Intn(250), rand.Intn(250)) // fresh keys
	t.Cleanup(func() {
		conn := l.pool.Get()
		defer conn.Close()
		keys, _ := redis.Strings(conn.Do("KEYS", "rl:*"+ip+"*"))
		for _, k := range keys {
			conn.Do("DEL", k)
		}
	})
	h := apiserverClientInfo(Middleware(l, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	status := func(path string) int {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, withBrowser(request(path, ip), kidA))
		return w.Code
	}
	const register = "/user_service.RegistrationService/Register"
	for i := 0; i < signupBucket.Anon.Burst; i++ {
		is.Equal(status(register), http.StatusOK) // retry i
	}
	is.Equal(status(register), http.StatusTooManyRequests)
	is.Equal(status("/user_service.RegistrationService/ResendVerificationEmail"), http.StatusOK)
}
