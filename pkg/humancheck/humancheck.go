// Package humancheck verifies a Cloudflare Turnstile token and, if it is
// valid, sets a signed cookie that the CDN edge accepts as proof that the
// browser passed the check.
package humancheck

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/apiserver"
)

const (
	Prefix     = "/api/human-check"
	CookieName = "wg_ok"

	verifyURL      = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	cookieLifetime = 30 * 24 * time.Hour
	maxTokenLength = 2048
)

type Service struct {
	turnstileSecret string
	cookieKey       []byte
	secureCookies   bool
	allowedHosts    map[string]bool
	client          *http.Client
	now             func() time.Time
	verifyURL       string
}

// NewService returns nil if either secret is empty; the endpoint is then not
// registered.
func NewService(turnstileSecret, cookieKey string, secureCookies bool, allowedHosts []string) *Service {
	if turnstileSecret == "" || cookieKey == "" {
		return nil
	}
	hosts := map[string]bool{}
	for _, h := range allowedHosts {
		hosts[h] = true
	}
	return &Service{
		turnstileSecret: turnstileSecret,
		cookieKey:       []byte(cookieKey),
		secureCookies:   secureCookies,
		allowedHosts:    hosts,
		client:          &http.Client{Timeout: 10 * time.Second},
		now:             time.Now,
		verifyURL:       verifyURL,
	}
}

// CookieValue is "<unix expiry>.<hex HMAC-SHA256 of the expiry>". The edge
// function recomputes the HMAC with the same key.
func CookieValue(key []byte, expiry time.Time) string {
	exp := strconv.FormatInt(expiry.Unix(), 10)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(exp))
	return exp + "." + hex.EncodeToString(mac.Sum(nil))
}

type verifyResponse struct {
	Success    bool     `json:"success"`
	Hostname   string   `json:"hostname"`
	ErrorCodes []string `json:"error-codes"`
}

func (s *Service) verify(r *http.Request, token string) (bool, error) {
	form := url.Values{
		"secret":   {s.turnstileSecret},
		"response": {token},
	}
	if ip := apiserver.ClientIPFromRequest(r); ip != "" {
		form.Set("remoteip", ip)
	}
	resp, err := s.client.PostForm(s.verifyURL, form)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var vr verifyResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&vr); err != nil {
		return false, err
	}
	if !vr.Success {
		log.Debug().Strs("errors", vr.ErrorCodes).Msg("human-check-rejected")
		return false, nil
	}
	if len(s.allowedHosts) > 0 && !s.allowedHosts[vr.Hostname] {
		log.Debug().Str("hostname", vr.Hostname).Msg("human-check-wrong-host")
		return false, nil
	}
	return true, nil
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil ||
		body.Token == "" || len(body.Token) > maxTokenLength {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	ok, err := s.verify(r, body.Token)
	if err != nil {
		log.Err(err).Msg("human-check-verify-error")
		http.Error(w, "Verification unavailable", http.StatusBadGateway)
		return
	}
	if !ok {
		http.Error(w, "Verification failed", http.StatusForbidden)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    CookieValue(s.cookieKey, s.now().Add(cookieLifetime)),
		Path:     "/",
		MaxAge:   int(cookieLifetime.Seconds()),
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}
