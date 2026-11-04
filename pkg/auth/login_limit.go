package auth

import (
	"context"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"
)

// Failed logins are counted in Redis, and an attempt that is over a limit is
// turned away before its password is hashed. Each limit stops a different
// kind of guessing:
//
//   - one address trying one account: a few tries, then a wait;
//   - one address trying many accounts;
//   - many addresses trying one account. This one could lock the owner out,
//     so it doesn't apply to a browser that has logged into the account before.
const (
	failWindowShort = 15 * time.Minute
	failWindowLong  = time.Hour

	maxFailsUserIP = 5
	maxFailsIP     = 20
	maxFailsUser   = 100

	// How long a browser that logged in is remembered for an account.
	knownClientTTL = 90 * 24 * time.Hour
)

type loginLimiter struct {
	pool *redis.Pool
}

type counter struct {
	key    string
	limit  int
	window time.Duration
}

func newLoginLimiter(pool *redis.Pool) *loginLimiter {
	if pool == nil {
		return nil
	}
	return &loginLimiter{pool: pool}
}

// counters returns the failure counters for an attempt, the per-account one
// last. Without an address only the per-account counter applies; the others
// would lump every such request together.
func counters(username, ip string) []counter {
	u := strings.ToLower(username)
	var cs []counter
	if ip != "" {
		cs = append(cs,
			counter{"loginfail:userip:" + u + ":" + ip, maxFailsUserIP, failWindowShort},
			counter{"loginfail:ip:" + ip, maxFailsIP, failWindowShort})
	}
	return append(cs, counter{"loginfail:user:" + u, maxFailsUser, failWindowLong})
}

func knownClientKey(username, clientID string) string {
	if clientID == "" {
		return ""
	}
	return "loginok:" + strings.ToLower(username) + ":" + clientID
}

// allowed reports whether a login attempt may go on to the password check. It
// lets the attempt through if Redis can't be reached, so an outage doesn't
// stop everyone logging in.
func (l *loginLimiter) allowed(ctx context.Context, username, ip, clientID string) bool {
	if l == nil {
		return true
	}
	conn, err := l.pool.GetContext(ctx)
	if err != nil {
		log.Err(err).Msg("login-limit-redis")
		return true
	}
	defer conn.Close()

	cs := counters(username, ip)
	keys := make([]any, len(cs))
	for i, c := range cs {
		keys[i] = c.key
	}
	counts, err := redis.Ints(conn.Do("MGET", keys...))
	if err != nil {
		log.Err(err).Msg("login-limit-redis")
		return true
	}
	for i, c := range cs[:len(cs)-1] {
		if counts[i] >= c.limit {
			return false
		}
	}
	if counts[len(cs)-1] < maxFailsUser {
		return true
	}
	known := knownClientKey(username, clientID)
	if known == "" {
		return false
	}
	ok, err := redis.Bool(conn.Do("EXISTS", known))
	if err != nil {
		log.Err(err).Msg("login-limit-redis")
		return true
	}
	return ok
}

// failed counts a failed login against the account and the address.
func (l *loginLimiter) failed(ctx context.Context, username, ip string) {
	if l == nil {
		return
	}
	conn, err := l.pool.GetContext(ctx)
	if err != nil {
		log.Err(err).Msg("login-limit-redis")
		return
	}
	defer conn.Close()

	for _, c := range counters(username, ip) {
		n, err := redis.Int(conn.Do("INCR", c.key))
		if err != nil {
			log.Err(err).Msg("login-limit-redis")
			return
		}
		// The window starts at the first failure.
		if n == 1 {
			if _, err := conn.Do("EXPIRE", c.key, int(c.window.Seconds())); err != nil {
				log.Err(err).Msg("login-limit-redis")
				return
			}
		}
	}
}

// succeeded clears this address's failures for the account and remembers the
// browser, which then isn't held back by failures from other addresses.
func (l *loginLimiter) succeeded(ctx context.Context, username, ip, clientID string) {
	if l == nil {
		return
	}
	conn, err := l.pool.GetContext(ctx)
	if err != nil {
		log.Err(err).Msg("login-limit-redis")
		return
	}
	defer conn.Close()

	if ip != "" {
		if _, err := conn.Do("DEL", counters(username, ip)[0].key); err != nil {
			log.Err(err).Msg("login-limit-redis")
		}
	}
	if known := knownClientKey(username, clientID); known != "" {
		if _, err := conn.Do("SET", known, 1, "EX", int(knownClientTTL.Seconds())); err != nil {
			log.Err(err).Msg("login-limit-redis")
		}
	}
}
