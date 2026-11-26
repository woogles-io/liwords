package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/woogles-io/liwords/pkg/apiserver"
)

// APIKeyUser returns the UUID of the user an API key belongs to, and false if
// the key is unknown.
type APIKeyUser func(ctx context.Context, apiKey string) (string, bool)

// identity is who a request is counted against.
type identity struct {
	kind    string // "user", "browser" or "ip"
	user    string // the user UUID
	ip      string // "<address>", or "<prefix>/64" for IPv6
	browser string // the client ID cookie, if the request sent one
}

// checksFor lists the buckets to take a token from, by key.
func checksFor(id identity, buckets []bucket) ([]Check, []string) {
	var checks []Check
	var names []string
	for _, b := range buckets {
		if id.user != "" && !b.AlwaysAnon {
			checks = append(checks, Check{Key: "rl:" + b.Name + ":u:" + id.user, Limit: b.User})
			names = append(names, b.Name)
			continue
		}
		if id.ip == "" {
			continue // nothing to count by
		}
		key := "rl:" + b.Name + ":ip:" + id.ip
		if id.browser != "" {
			key += ":c:" + id.browser
		}
		checks = append(checks,
			Check{Key: key, Limit: b.Anon},
			Check{Key: "rl:" + b.Name + ":ipmax:" + id.ip, Limit: b.IPCeiling})
		names = append(names, b.Name)
	}
	return checks, names
}

// Middleware refuses requests over their buckets' limits with 429 (a Connect
// resource_exhausted error for Connect requests) and a Retry-After header.
// It must run after the auth, API key and client info middlewares. If the
// checker fails, the request goes ahead.
func Middleware(checker Checker, apiKeyUser APIKeyUser) func(http.Handler) http.Handler {
	errWriter := connect.NewErrorWriter()
	var lastErrLog atomic.Int64
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			id := identify(ctx, r, apiKeyUser)
			checks, names := checksFor(id, bucketsFor(r.URL.Path))

			res, err := checker.Allow(ctx, checks)
			span := trace.SpanFromContext(ctx)
			if err != nil {
				// Fail open, logging at most every 10 seconds.
				now := time.Now().Unix()
				if last := lastErrLog.Load(); now-last >= 10 && lastErrLog.CompareAndSwap(last, now) {
					log.Err(err).Msg("rate-limit-check-failed")
				}
				span.SetAttributes(attribute.Bool("ratelimit.error", true))
				h.ServeHTTP(w, r)
				return
			}
			span.SetAttributes(
				attribute.StringSlice("ratelimit.buckets", names),
				attribute.String("ratelimit.identity_kind", id.kind),
				attribute.Bool("ratelimit.allowed", res.Allowed))
			if res.Allowed {
				h.ServeHTTP(w, r)
				return
			}

			secs := int(math.Ceil(res.RetryAfter.Seconds()))
			if secs < 1 {
				secs = 1
			}
			zerolog.Ctx(ctx).Warn().
				Strs("buckets", names).
				Str("identity_kind", id.kind).
				Str("identity_hash", hashIdentity(id)).
				Str("path", r.URL.Path).
				Int("retry_after_s", secs).
				Msg("rate-limited")
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			if errWriter.IsSupported(r) {
				_ = errWriter.Write(w, r, connect.NewError(connect.CodeResourceExhausted,
					errors.New("too many requests; try again in "+strconv.Itoa(secs)+" seconds")))
				return
			}
			http.Error(w, "too many requests", http.StatusTooManyRequests)
		})
	}
}

func identify(ctx context.Context, r *http.Request, apiKeyUser APIKeyUser) identity {
	id := identity{kind: "ip"}
	ip := apiserver.ClientIP(ctx)
	if ip == "" {
		ip = apiserver.ClientIPFromRequest(r)
	}
	if ip != "" {
		id.ip = ipBucket(ip)
	}
	// Only an ID the browser sent back; ClientInfoMiddlewareGenerator leaves
	// out a freshly minted one.
	if cid := apiserver.ClientID(ctx); cid != "" {
		id.kind, id.browser = "browser", cid
	}
	if sess, err := apiserver.GetSession(ctx); err == nil && sess.UserUUID != "" {
		id.kind, id.user = "user", sess.UserUUID
	} else if key, err := apiserver.GetAPIKey(ctx); err == nil && key != "" && apiKeyUser != nil {
		// An unknown key counts as anonymous, so made-up keys can't dodge
		// the per-IP limits.
		if uuid, ok := apiKeyUser(ctx, key); ok {
			id.kind, id.user = "user", uuid
		}
	}
	return id
}

// ipBucket groups IPv6 addresses by /64: one subscriber usually gets a whole
// /64, so counting single addresses would be easy to dodge.
func ipBucket(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip
	}
	if parsed.To4() != nil {
		return parsed.String()
	}
	return parsed.Mask(net.CIDRMask(64, 128)).String() + "/64"
}

func hashIdentity(id identity) string {
	who := id.user
	if who == "" {
		who = id.ip + ":" + id.browser
	}
	sum := sha256.Sum256([]byte(who))
	return hex.EncodeToString(sum[:6])
}

// CachedAPIKeyUser wraps a lookup with a cache, so a script calling with an
// API key costs one database query a minute, not one per request.
func CachedAPIKeyUser(lookup func(ctx context.Context, apiKey string) (string, error)) APIKeyUser {
	const ttl = time.Minute
	const maxEntries = 10000
	type entry struct {
		uuid    string
		ok      bool
		expires time.Time
	}
	var mu sync.Mutex
	cache := map[string]entry{}
	return func(ctx context.Context, apiKey string) (string, bool) {
		now := time.Now()
		mu.Lock()
		e, found := cache[apiKey]
		mu.Unlock()
		if found && now.Before(e.expires) {
			return e.uuid, e.ok
		}
		uuid, err := lookup(ctx, apiKey)
		e = entry{uuid: uuid, ok: err == nil && uuid != "", expires: now.Add(ttl)}
		mu.Lock()
		if len(cache) >= maxEntries {
			cache = map[string]entry{}
		}
		cache[apiKey] = e
		mu.Unlock()
		return e.uuid, e.ok
	}
}
