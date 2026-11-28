package ratelimit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
)

type fakeChecker struct {
	got    []Check
	result Result
	err    error
}

func (f *fakeChecker) Allow(ctx context.Context, checks []Check) (Result, error) {
	f.got = checks
	return f.result, f.err
}

func keys(cs []Check) []string {
	var ks []string
	for _, c := range cs {
		ks = append(ks, c.Key)
	}
	return ks
}

const analysisPath = "/analysis_service.AnalysisService/GetAnalysisResult"

func knownKey(ctx context.Context, key string) (string, bool) {
	if key == "good-key" {
		return "key-user", true
	}
	return "", false
}

// serve runs one request through the middleware. It returns the response and
// whether the handler behind it ran.
func serve(f *fakeChecker, r *http.Request) (*httptest.ResponseRecorder, bool) {
	ran := false
	// As in production: the client info middleware reads the IP and the
	// client ID cookie first.
	h := apiserver.ClientInfoMiddlewareGenerator(false)(
		Middleware(f, knownKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ran = true
		})))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w, ran
}

func request(path, ip string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Connect-Protocol-Version", "1")
	r.Header.Set("CloudFront-Viewer-Address", ip+":5555")
	return r
}

// A valid client ID: 22 base64url characters.
const kidA = "AAAAAAAAAAAAAAAAAAAAAA"
const kidB = "BBBBBBBBBBBBBBBBBBBBBB"

func withBrowser(r *http.Request, cid string) *http.Request {
	r.AddCookie(&http.Cookie{Name: "client_id", Value: cid})
	return r
}

func withSession(r *http.Request, uuid string) *http.Request {
	return r.WithContext(apiserver.PlaceInContext(r.Context(), &entity.Session{UserUUID: uuid}))
}

func withAPIKey(r *http.Request, key string) *http.Request {
	return r.WithContext(apiserver.StoreAPIKeyInContext(r.Context(), key))
}

// Without a client ID (scripts, first visits), one bucket per IP, under the
// IP ceiling.
func TestAnonymousWithoutCookieCountedPerIP(t *testing.T) {
	is := is.New(t)
	f := &fakeChecker{result: Result{Allowed: true}}
	_, ran := serve(f, request(analysisPath, "1.2.3.4"))
	is.True(ran)
	is.Equal(keys(f.got), []string{
		"rl:general:ip:1.2.3.4", "rl:general:ipmax:1.2.3.4",
		"rl:heavy:ip:1.2.3.4", "rl:heavy:ipmax:1.2.3.4"})
	is.Equal(f.got[0].Limit, generalBucket.Anon)
	is.Equal(f.got[1].Limit, generalBucket.IPCeiling)
	is.Equal(f.got[2].Limit, heavyBucket.Anon)
	is.Equal(f.got[3].Limit, heavyBucket.IPCeiling)
}

// Two browsers on one IP (a school) get their own buckets, and share the
// IP ceiling.
func TestAnonymousBrowsersCountedSeparately(t *testing.T) {
	is := is.New(t)
	f := &fakeChecker{result: Result{Allowed: true}}
	serve(f, withBrowser(request("/x.Y/Z", "1.2.3.4"), kidA))
	is.Equal(keys(f.got), []string{"rl:general:ip:1.2.3.4:c:" + kidA, "rl:general:ipmax:1.2.3.4"})
	serve(f, withBrowser(request("/x.Y/Z", "1.2.3.4"), kidB))
	is.Equal(keys(f.got), []string{"rl:general:ip:1.2.3.4:c:" + kidB, "rl:general:ipmax:1.2.3.4"})
	// A malformed cookie counts as none.
	serve(f, withBrowser(request("/x.Y/Z", "1.2.3.4"), "nope"))
	is.Equal(keys(f.got), []string{"rl:general:ip:1.2.3.4", "rl:general:ipmax:1.2.3.4"})
}

func TestLoggedInCountedPerUser(t *testing.T) {
	is := is.New(t)
	f := &fakeChecker{result: Result{Allowed: true}}
	serve(f, withBrowser(withSession(request(analysisPath, "1.2.3.4"), "abc"), kidA))
	is.Equal(keys(f.got), []string{"rl:general:u:abc", "rl:heavy:u:abc"})
	is.Equal(f.got[0].Limit, generalBucket.User)
	is.Equal(f.got[1].Limit, heavyBucket.User)
}

func TestAPIKeys(t *testing.T) {
	is := is.New(t)
	f := &fakeChecker{result: Result{Allowed: true}}
	serve(f, withAPIKey(request("/x.Y/Z", "1.2.3.4"), "good-key"))
	is.Equal(keys(f.got), []string{"rl:general:u:key-user"})
	// A made-up key is anonymous.
	serve(f, withAPIKey(request("/x.Y/Z", "1.2.3.4"), "made-up"))
	is.Equal(keys(f.got), []string{"rl:general:ip:1.2.3.4", "rl:general:ipmax:1.2.3.4"})
}

// Endpoints used before logging in count as anonymous even with a session.
func TestAlwaysAnonBuckets(t *testing.T) {
	is := is.New(t)
	f := &fakeChecker{result: Result{Allowed: true}}
	serve(f, withBrowser(withSession(request("/user_service.RegistrationService/Register", "1.2.3.4"), "abc"), kidA))
	is.Equal(keys(f.got), []string{"rl:general:u:abc",
		"rl:email:ip:1.2.3.4:c:" + kidA, "rl:email:ipmax:1.2.3.4",
		"rl:password-hash:ip:1.2.3.4:c:" + kidA, "rl:password-hash:ipmax:1.2.3.4"})
	// Tournament registration is a different Register.
	serve(f, request("/tournament_service.TournamentService/Register", "1.2.3.4"))
	is.Equal(keys(f.got), []string{"rl:general:ip:1.2.3.4", "rl:general:ipmax:1.2.3.4"})
}

func TestIPv6GroupedBy64(t *testing.T) {
	is := is.New(t)
	f := &fakeChecker{result: Result{Allowed: true}}
	serve(f, request("/x.Y/Z", "2001:db8:1:2:aaaa:bbbb:cccc:dddd"))
	is.Equal(keys(f.got), []string{"rl:general:ip:2001:db8:1:2::/64", "rl:general:ipmax:2001:db8:1:2::/64"})
}

func TestOtherRoutes(t *testing.T) {
	is := is.New(t)
	f := &fakeChecker{result: Result{Allowed: true}}
	serve(f, request("/gameimg/abc-v2.png", "1.2.3.4"))
	is.Equal(keys(f.got), []string{"rl:general:ip:1.2.3.4", "rl:general:ipmax:1.2.3.4",
		"rl:heavy:ip:1.2.3.4", "rl:heavy:ipmax:1.2.3.4"})
}

func TestRefusedConnectRequest(t *testing.T) {
	is := is.New(t)
	f := &fakeChecker{result: Result{Allowed: false, RetryAfter: 1500 * time.Millisecond}}
	w, ran := serve(f, request(analysisPath, "1.2.3.4"))
	is.True(!ran)
	is.Equal(w.Code, http.StatusTooManyRequests)
	is.Equal(w.Header().Get("Retry-After"), "2") // rounded up
	var body struct{ Code, Message string }
	is.NoErr(json.Unmarshal(w.Body.Bytes(), &body))
	is.Equal(body.Code, "resource_exhausted")
}

func TestRefusedPlainRequest(t *testing.T) {
	is := is.New(t)
	f := &fakeChecker{result: Result{Allowed: false, RetryAfter: 10 * time.Millisecond}}
	r := httptest.NewRequest(http.MethodGet, "/gameimg/abc-v2.png", nil)
	r.Header.Set("CloudFront-Viewer-Address", "1.2.3.4:5555")
	w, ran := serve(f, r)
	is.True(!ran)
	is.Equal(w.Code, http.StatusTooManyRequests)
	is.Equal(w.Header().Get("Retry-After"), "1")
}

// If Valkey fails, requests go ahead.
func TestFailsOpen(t *testing.T) {
	is := is.New(t)
	f := &fakeChecker{err: errors.New("valkey down")}
	_, ran := serve(f, request(analysisPath, "1.2.3.4"))
	is.True(ran)
}

func TestCachedAPIKeyUser(t *testing.T) {
	is := is.New(t)
	calls := 0
	lookup := CachedAPIKeyUser(func(ctx context.Context, key string) (string, error) {
		calls++
		if key == "good" {
			return "u1", nil
		}
		return "", errors.New("user not found")
	})
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		uuid, ok := lookup(ctx, "good")
		is.True(ok)
		is.Equal(uuid, "u1")
		_, ok = lookup(ctx, "bad")
		is.True(!ok)
	}
	is.Equal(calls, 2) // one per key; the rest came from the cache
}

// Every bucket that counts anonymous callers has both limits set, with the
// ceiling above the per-browser limit.
func TestBucketLimits(t *testing.T) {
	is := is.New(t)
	for _, b := range []bucket{generalBucket, heavyBucket, emailBucket, passwordHashBucket} {
		is.True(b.Anon.Rate > 0 && b.Anon.Burst > 0)
		is.True(b.IPCeiling.Rate > b.Anon.Rate)   // b.Name
		is.True(b.IPCeiling.Burst > b.Anon.Burst) // b.Name
		if !b.AlwaysAnon {
			is.True(b.User.Rate > 0 && b.User.Burst > 0)
		}
	}
}

func TestLimitIntervals(t *testing.T) {
	is := is.New(t)
	is.Equal(generalBucket.Anon.intervalMs(), int64(100))
	is.Equal(emailBucket.Anon.intervalMs(), int64(720000))                         // 5 an hour
	is.Equal(Limit{Rate: 5000, Per: time.Second, Burst: 1}.intervalMs(), int64(1)) // never 0
}
