package apiserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"
)

func TestClientIPFromRequest(t *testing.T) {
	cases := []struct {
		name       string
		viewer     string
		xff        []string
		remoteAddr string
		want       string
	}{
		{"viewer header", "203.0.113.7:51234", []string{"203.0.113.7, 10.0.0.1"}, "10.0.0.2:80", "203.0.113.7"},
		{"viewer header ipv6", "2001:db8::1:443", nil, "10.0.0.2:80", "2001:db8::1"},
		{"cdn and lb hops", "", []string{"203.0.113.7, 198.51.100.9"}, "10.0.0.2:80", "203.0.113.7"},
		{"client-supplied prefix ignored", "", []string{"1.2.3.4, 203.0.113.7, 198.51.100.9"}, "10.0.0.2:80", "203.0.113.7"},
		{"multiple header lines", "", []string{"1.2.3.4", "203.0.113.7, 198.51.100.9"}, "10.0.0.2:80", "203.0.113.7"},
		{"single hop", "", []string{"203.0.113.7"}, "10.0.0.2:80", "203.0.113.7"},
		{"no proxy", "", nil, "192.0.2.5:4321", "192.0.2.5"},
		{"garbage", "", []string{"not-an-ip, 198.51.100.9"}, "10.0.0.2:80", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			is := is.New(t)
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = c.remoteAddr
			if c.viewer != "" {
				r.Header.Set("CloudFront-Viewer-Address", c.viewer)
			}
			for _, v := range c.xff {
				r.Header.Add("X-Forwarded-For", v)
			}
			is.Equal(ClientIPFromRequest(r), c.want)
		})
	}
}

func TestClientInfoMiddleware(t *testing.T) {
	is := is.New(t)
	var gotID, gotIP string
	h := ClientInfoMiddlewareGenerator(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = ClientID(r.Context())
		gotIP = ClientIP(r.Context())
	}))

	// No cookie: one is issued and placed in the context.
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.5:4321"
	h.ServeHTTP(rec, r)
	cookies := rec.Result().Cookies()
	is.Equal(len(cookies), 1)
	is.Equal(cookies[0].Name, ClientIDCookie)
	is.True(cookies[0].HttpOnly)
	is.True(ValidClientID(cookies[0].Value))
	is.Equal(gotID, cookies[0].Value)
	is.Equal(gotIP, "192.0.2.5")

	// Valid cookie: reused, not reissued.
	issued := cookies[0].Value
	rec = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: ClientIDCookie, Value: issued})
	h.ServeHTTP(rec, r)
	is.Equal(len(rec.Result().Cookies()), 0)
	is.Equal(gotID, issued)

	// Malformed cookie: replaced.
	rec = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: ClientIDCookie, Value: "bogus"})
	h.ServeHTTP(rec, r)
	is.Equal(len(rec.Result().Cookies()), 1)
	is.True(gotID != "bogus")
}
