package humancheck

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matryer/is"
)

func fakeTurnstile(t *testing.T, reply string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("secret") != "s3cret" || r.Form.Get("response") != "tok" {
			w.Write([]byte(`{"success":false,"error-codes":["invalid-input-response"]}`))
			return
		}
		w.Write([]byte(reply))
	}))
}

func newTestService(verifyURL string) *Service {
	s := NewService("s3cret", "key", true, []string{"woogles.io"})
	s.verifyURL = verifyURL
	s.now = func() time.Time { return time.Unix(1_800_000_000, 0) }
	return s
}

func post(s *Service, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, Prefix, strings.NewReader(body)))
	return rec
}

func TestCookieValue(t *testing.T) {
	is := is.New(t)
	// Must match what the edge function computes:
	// crypto.createHmac("sha256", "key").update("1800000000").digest("hex")
	is.Equal(CookieValue([]byte("key"), time.Unix(1_800_000_000, 0)),
		"1800000000.69781f7c57f4d3f41375dd613b2c01feb230c0353398e870babb15af1572d3a4")
}

func TestValidToken(t *testing.T) {
	is := is.New(t)
	ts := fakeTurnstile(t, `{"success":true,"hostname":"woogles.io"}`)
	defer ts.Close()
	rec := post(newTestService(ts.URL), `{"token":"tok"}`)
	is.Equal(rec.Code, http.StatusNoContent)
	cookies := rec.Result().Cookies()
	is.Equal(len(cookies), 1)
	is.Equal(cookies[0].Name, CookieName)
	is.Equal(cookies[0].Value, CookieValue([]byte("key"), time.Unix(1_800_000_000, 0).Add(cookieLifetime)))
	is.True(cookies[0].HttpOnly)
	is.True(cookies[0].Secure)
}

func TestRejectedToken(t *testing.T) {
	is := is.New(t)
	ts := fakeTurnstile(t, `{"success":true,"hostname":"woogles.io"}`)
	defer ts.Close()
	rec := post(newTestService(ts.URL), `{"token":"forged"}`)
	is.Equal(rec.Code, http.StatusForbidden)
	is.Equal(len(rec.Result().Cookies()), 0)
}

func TestWrongHostname(t *testing.T) {
	is := is.New(t)
	ts := fakeTurnstile(t, `{"success":true,"hostname":"evil.example"}`)
	defer ts.Close()
	rec := post(newTestService(ts.URL), `{"token":"tok"}`)
	is.Equal(rec.Code, http.StatusForbidden)
}

func TestBadRequests(t *testing.T) {
	is := is.New(t)
	s := newTestService("http://127.0.0.1:0")
	is.Equal(post(s, `not json`).Code, http.StatusBadRequest)
	is.Equal(post(s, `{"token":""}`).Code, http.StatusBadRequest)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Prefix, nil))
	is.Equal(rec.Code, http.StatusMethodNotAllowed)
}

func TestDisabledWithoutSecrets(t *testing.T) {
	is := is.New(t)
	is.True(NewService("", "key", true, nil) == nil)
	is.True(NewService("s3cret", "", true, nil) == nil)
}
