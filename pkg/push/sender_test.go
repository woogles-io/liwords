package push

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

// fakeQueries keeps subscriptions in memory.
type fakeQueries struct {
	mu       sync.Mutex
	subs     map[string][]models.GetPushSubscriptionsRow
	upserts  []models.UpsertPushSubscriptionParams
	deleted  []string
	noUser   bool
	removals []models.DeleteUserPushSubscriptionParams
}

func (f *fakeQueries) UpsertPushSubscription(_ context.Context, arg models.UpsertPushSubscriptionParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.noUser {
		return 0, nil
	}
	f.upserts = append(f.upserts, arg)
	return 1, nil
}

func (f *fakeQueries) GetPushSubscriptions(_ context.Context, userUUID string) ([]models.GetPushSubscriptionsRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.subs[userUUID], nil
}

func (f *fakeQueries) DeleteUserPushSubscription(_ context.Context, arg models.DeleteUserPushSubscriptionParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removals = append(f.removals, arg)
	return 1, nil
}

func (f *fakeQueries) DeletePushSubscription(_ context.Context, endpoint string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = append(f.deleted, endpoint)
	return nil
}

// browserKeys makes the keys a browser would hand over with a subscription.
func browserKeys(t *testing.T) (p256dh, auth string) {
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(k.PublicKey().Bytes()),
		base64.RawURLEncoding.EncodeToString(secret)
}

func subscription(t *testing.T, endpoint string) models.GetPushSubscriptionsRow {
	p256dh, auth := browserKeys(t)
	return models.GetPushSubscriptionsRow{Endpoint: endpoint, P256dh: p256dh, Auth: auth}
}

type pushRequest struct {
	method string
	path   string
	header http.Header
	body   []byte
}

// pushService stands in for a push service: each path answers with its own
// status, and every request is recorded.
type pushService struct {
	mu       sync.Mutex
	requests []pushRequest
	status   map[string]int
	got      chan struct{}
}

func newPushService(t *testing.T, status map[string]int) (*httptest.Server, *pushService) {
	ps := &pushService{status: status, got: make(chan struct{}, 100)}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		ps.mu.Lock()
		ps.requests = append(ps.requests, pushRequest{
			method: r.Method, path: r.URL.Path, header: r.Header.Clone(), body: body})
		code, ok := ps.status[r.URL.Path]
		ps.mu.Unlock()
		if !ok {
			code = http.StatusCreated
		}
		w.WriteHeader(code)
		ps.got <- struct{}{}
	}))
	t.Cleanup(srv.Close)
	return srv, ps
}

func (ps *pushService) received() []pushRequest {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return slices.Clone(ps.requests)
}

func testSender(t *testing.T, q Queries, srv *httptest.Server) *Sender {
	settings, problems := NewSettings(validConfig(t))
	if problems != nil {
		t.Fatal(problems)
	}
	s := NewSender(settings, q)
	// The test service listens on a loopback address, which the real client
	// refuses.
	s.client = srv.Client()
	return s
}

func TestSendSignsAndEncrypts(t *testing.T) {
	is := is.New(t)
	srv, ps := newPushService(t, nil)
	q := &fakeQueries{subs: map[string][]models.GetPushSubscriptionsRow{
		"u1": {subscription(t, srv.URL+"/ok")},
	}}
	s := testSender(t, q, srv)

	s.send(context.Background(), "u1", []byte(`{"gameId":"g1"}`))

	requests := ps.received()
	is.Equal(len(requests), 1)
	r := requests[0]
	is.Equal(r.method, http.MethodPost)
	is.Equal(r.path, "/ok")
	is.Equal(r.header.Get("Content-Encoding"), "aes128gcm")
	is.Equal(r.header.Get("TTL"), "172800") // 48h
	is.True(len(r.body) > 0)

	authz := r.header.Get("Authorization")
	is.True(strings.HasPrefix(authz, "vapid t="))
	jwt, key, ok := strings.Cut(strings.TrimPrefix(authz, "vapid t="), ", k=")
	is.True(ok)
	is.Equal(key, s.settings.PublicKey)
	parts := strings.Split(jwt, ".")
	is.Equal(len(parts), 3)
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	is.NoErr(err)
	var claims struct {
		Aud string `json:"aud"`
		Sub string `json:"sub"`
	}
	is.NoErr(json.Unmarshal(claimsJSON, &claims))
	is.Equal(claims.Sub, "mailto:push@example.com") // said once, not "mailto:mailto:"
	is.Equal(claims.Aud, srv.URL)
	is.Equal(len(q.deleted), 0)
}

func TestHTTPSSubjectIsSentAsIs(t *testing.T) {
	is := is.New(t)
	s := &Sender{settings: &Settings{Subject: "https://woogles.io/about"}}
	is.Equal(s.subscriber(), "https://woogles.io/about")
	s.settings.Subject = "mailto:push@example.com"
	is.Equal(s.subscriber(), "push@example.com")
}

func TestGoneSubscriptionsAreDeleted(t *testing.T) {
	is := is.New(t)
	srv, ps := newPushService(t, map[string]int{
		"/gone404": http.StatusNotFound,
		"/gone410": http.StatusGone,
		"/err500":  http.StatusInternalServerError,
		"/bad400":  http.StatusBadRequest,
		"/slow429": http.StatusTooManyRequests,
	})
	q := &fakeQueries{subs: map[string][]models.GetPushSubscriptionsRow{
		"u1": {
			subscription(t, srv.URL+"/gone404"),
			subscription(t, srv.URL+"/err500"),
			subscription(t, srv.URL+"/gone410"),
			subscription(t, srv.URL+"/bad400"),
			subscription(t, srv.URL+"/slow429"),
			subscription(t, srv.URL+"/ok"),
		},
	}}
	s := testSender(t, q, srv)

	s.send(context.Background(), "u1", []byte(`{}`))

	is.Equal(len(ps.received()), 6)
	is.Equal(q.deleted, []string{srv.URL + "/gone404", srv.URL + "/gone410"})
}

func TestNotifySendsInTheBackground(t *testing.T) {
	is := is.New(t)
	srv, ps := newPushService(t, nil)
	q := &fakeQueries{subs: map[string][]models.GetPushSubscriptionsRow{
		"u1": {subscription(t, srv.URL+"/a"), subscription(t, srv.URL+"/b")},
	}}
	s := testSender(t, q, srv)

	s.Notify("u1", []byte(`{}`))
	for range 2 {
		select {
		case <-ps.got:
		case <-time.After(10 * time.Second):
			t.Fatal("the pushes never came")
		}
	}
	is.Equal(len(ps.received()), 2)
}

// The push library writes into a message's spare capacity; the caller's
// array must come back untouched.
func TestSendLeavesThePayloadAlone(t *testing.T) {
	is := is.New(t)
	srv, _ := newPushService(t, nil)
	sub := subscription(t, srv.URL+"/ok")
	s := testSender(t, &fakeQueries{}, srv)

	backing := make([]byte, 64)
	for i := range backing {
		backing[i] = '#'
	}
	payload := backing[:8]
	s.sendOne(context.Background(), sub, payload)
	is.Equal(string(backing), strings.Repeat("#", 64))
}

// Run with -race: the same payload goes to several browsers at once.
func TestConcurrentSendsOfOnePayload(t *testing.T) {
	srv, ps := newPushService(t, nil)
	s := testSender(t, &fakeQueries{}, srv)
	payload := make([]byte, 8, 64)

	var wg sync.WaitGroup
	for i := range 8 {
		sub := subscription(t, srv.URL+"/"+string(rune('a'+i)))
		wg.Go(func() {
			s.sendOne(context.Background(), sub, payload)
		})
	}
	wg.Wait()
	if n := len(ps.received()); n != 8 {
		t.Fatalf("got %d requests", n)
	}
}

func TestSubscribe(t *testing.T) {
	is := is.New(t)
	srv, _ := newPushService(t, nil)
	q := &fakeQueries{}
	s := testSender(t, q, srv)
	p256dh, auth := browserKeys(t)
	ctx := context.Background()

	is.NoErr(s.Subscribe(ctx, "u1", "https://push.example.com/a", p256dh, auth))
	is.Equal(len(q.upserts), 1)
	is.Equal(q.upserts[0], models.UpsertPushSubscriptionParams{
		Endpoint: "https://push.example.com/a", P256dh: p256dh, Auth: auth, UserUuid: "u1"})

	for _, endpoint := range []string{
		"http://push.example.com/a",
		"https:///a",
		"https://user:pass@push.example.com/a",
		"push.example.com/a",
		"",
	} {
		is.Equal(s.Subscribe(ctx, "u1", endpoint, p256dh, auth), errBadEndpoint)
	}
	is.Equal(s.Subscribe(ctx, "u1", "https://push.example.com/a", "", auth), errBadKeys)
	is.Equal(s.Subscribe(ctx, "u1", "https://push.example.com/a", p256dh, ""), errBadKeys)
	is.Equal(s.Subscribe(ctx, "u1", "https://push.example.com/a", auth, auth), errBadKeys)
	is.Equal(s.Subscribe(ctx, "u1", "https://push.example.com/a", p256dh, p256dh), errBadKeys)
	is.Equal(len(q.upserts), 1)

	q.noUser = true
	is.Equal(s.Subscribe(ctx, "nobody", "https://push.example.com/b", p256dh, auth), errNoUser)
}

func TestUnsubscribe(t *testing.T) {
	is := is.New(t)
	srv, _ := newPushService(t, nil)
	q := &fakeQueries{}
	s := testSender(t, q, srv)
	is.NoErr(s.Unsubscribe(context.Background(), "u1", "https://push.example.com/a"))
	is.Equal(q.removals, []models.DeleteUserPushSubscriptionParams{
		{Endpoint: "https://push.example.com/a", UserUuid: "u1"}})
}

// An endpoint's path is the subscription's secret, and the client's errors
// quote the whole URL.
func TestFailedSendLogsOnlyTheHost(t *testing.T) {
	is := is.New(t)
	var logged bytes.Buffer
	saved := log.Logger
	log.Logger = zerolog.New(&logged)
	defer func() { log.Logger = saved }()

	settings, problems := NewSettings(validConfig(t))
	is.Equal(len(problems), 0)
	s := NewSender(settings, &fakeQueries{})
	// The real client refuses the loopback address before connecting.
	s.sendOne(context.Background(), subscription(t, "https://127.0.0.1:1/the-subscription-secret"), []byte(`{}`))

	out := logged.String()
	is.True(strings.Contains(out, "web-push-send"))
	is.True(strings.Contains(out, "127.0.0.1:1"))
	is.True(strings.Contains(out, "is not a public address"))
	is.True(!strings.Contains(out, "the-subscription-secret"))
}

func TestClientRefusesInsideAddresses(t *testing.T) {
	is := is.New(t)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the request got through")
	}))
	defer srv.Close()
	_, err := newClient().Get(srv.URL)
	is.True(err != nil)
	is.True(strings.Contains(err.Error(), "is not a public address"))
}

func TestIsPublic(t *testing.T) {
	for addr, public := range map[string]bool{
		"8.8.8.8":           true,
		"142.250.80.42":     true,
		"2607:f8b0:4004::1": true,
		"127.0.0.1":         false,
		"10.1.2.3":          false,
		"172.16.0.1":        false,
		"192.168.1.1":       false,
		"169.254.169.254":   false,
		"100.64.0.1":        false,
		"0.0.0.0":           false,
		"255.255.255.255":   false,
		"224.0.0.1":         false,
		"::1":               false,
		"::":                false,
		"fe80::1":           false,
		"fd00::1":           false,
		"::ffff:10.0.0.1":   false,
		"::ffff:8.8.8.8":    true,
		"64:ff9b::a00:1":    false,
		"64:ff9b::7f00:1":   false,
		"2002:a00:1::1":     false,
		"::7f00:1":          false,
		"fec0::1":           false,
		"3fff::1":           false,
		"2607:f8b0::1":      true,
	} {
		if got := isPublic(netip.MustParseAddr(addr)); got != public {
			t.Errorf("isPublic(%s) = %v, want %v", addr, got, public)
		}
	}
}

// holdingTransport answers each request once it is released, or fails it
// when its context is canceled first.
type holdingTransport struct {
	started  chan string
	release  chan struct{}
	mu       sync.Mutex
	canceled []string
	answered []string
}

func newHoldingTransport() *holdingTransport {
	return &holdingTransport{started: make(chan string, 8), release: make(chan struct{})}
}

func (h *holdingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	h.started <- r.URL.String()
	select {
	case <-r.Context().Done():
		h.mu.Lock()
		h.canceled = append(h.canceled, r.URL.String())
		h.mu.Unlock()
		return nil, r.Context().Err()
	case <-h.release:
		h.mu.Lock()
		h.answered = append(h.answered, r.URL.String())
		h.mu.Unlock()
		return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader("")),
			Request: r}, nil
	}
}

func (h *holdingTransport) results() (canceled, answered []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.canceled), slices.Clone(h.answered)
}

func holdingSender(t *testing.T, subs map[string][]models.GetPushSubscriptionsRow) (*Sender, *holdingTransport) {
	settings, problems := NewSettings(validConfig(t))
	if problems != nil {
		t.Fatal(problems)
	}
	s := NewSender(settings, &fakeQueries{subs: subs})
	h := newHoldingTransport()
	s.client = &http.Client{Transport: h}
	return s, h
}

func (s *Sender) inFlightCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.inFlight)
}

func TestNewerSendToADeviceCancelsTheOlder(t *testing.T) {
	is := is.New(t)
	endpoint := "https://push.example.com/d1"
	s, h := holdingSender(t, map[string][]models.GetPushSubscriptionsRow{
		"u1": {subscription(t, endpoint)}})

	var first, second sync.WaitGroup
	first.Add(1)
	go func() { defer first.Done(); s.send(context.Background(), "u1", []byte(`{"turn":1}`)) }()
	is.Equal(<-h.started, endpoint)

	// The newer send cancels the one still waiting, then is answered.
	second.Add(1)
	go func() { defer second.Done(); s.send(context.Background(), "u1", []byte(`{"turn":2}`)) }()
	is.Equal(<-h.started, endpoint)
	first.Wait()
	// The canceled send, finishing, leaves the newer one's entry alone.
	is.Equal(s.inFlightCount(), 1)
	close(h.release)
	second.Wait()
	canceled, answered := h.results()
	is.Equal(canceled, []string{endpoint})
	is.Equal(answered, []string{endpoint})
	is.Equal(s.inFlightCount(), 0)
}

func TestSendsToOtherDevicesDoNotCancel(t *testing.T) {
	is := is.New(t)
	s, h := holdingSender(t, map[string][]models.GetPushSubscriptionsRow{
		"u1": {subscription(t, "https://push.example.com/a")},
		"u2": {subscription(t, "https://push.example.com/b")}})

	var wg sync.WaitGroup
	for _, u := range []string{"u1", "u2"} {
		wg.Add(1)
		go func() { defer wg.Done(); s.send(context.Background(), u, []byte(`{}`)) }()
	}
	<-h.started
	<-h.started
	is.Equal(s.inFlightCount(), 2)
	close(h.release)
	wg.Wait()
	canceled, answered := h.results()
	is.Equal(len(canceled), 0)
	slices.Sort(answered)
	is.Equal(answered, []string{"https://push.example.com/a", "https://push.example.com/b"})
	is.Equal(s.inFlightCount(), 0)
}
