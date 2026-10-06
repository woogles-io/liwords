package apiserver

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// ClientIDCookie is the name of the long-lived cookie that identifies a
// browser across sessions.
const ClientIDCookie = "client_id"

const clientIDBytes = 16
const clientIDCookieLifetime = 2 * 365 * 24 * time.Hour

const clientipkey ctxkey = "clientip"
const clientidkey ctxkey = "clientid"

// ClientInfoMiddlewareGenerator stores the requesting client's IP address and
// client ID in the context, issuing a client ID cookie if the request has none.
// It must run after ExposeResponseWriterMiddleware.
func ClientInfoMiddlewareGenerator(secureCookies bool) (mw func(http.Handler) http.Handler) {
	mw = func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, clientipkey, ClientIPFromRequest(r))

			cid := ""
			if c, err := r.Cookie(ClientIDCookie); err == nil && ValidClientID(c.Value) {
				cid = c.Value
			} else if cid = newClientID(); cid != "" {
				http.SetCookie(w, &http.Cookie{
					Name:     ClientIDCookie,
					Value:    cid,
					Expires:  time.Now().Add(clientIDCookieLifetime),
					HttpOnly: true,
					Path:     "/",
					SameSite: http.SameSiteLaxMode,
					Secure:   secureCookies,
				})
			}
			ctx = context.WithValue(ctx, clientidkey, cid)
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	return
}

// ClientIP returns the client IP stored by ClientInfoMiddlewareGenerator, or ""
// if unknown.
func ClientIP(ctx context.Context) string {
	ip, _ := ctx.Value(clientipkey).(string)
	return ip
}

// ClientID returns the client ID stored by ClientInfoMiddlewareGenerator, or ""
// if unknown.
func ClientID(ctx context.Context) string {
	cid, _ := ctx.Value(clientidkey).(string)
	return cid
}

// ClientIPFromRequest determines the originating client IP. In production,
// requests pass through CloudFront and then a load balancer, each of which
// appends to X-Forwarded-For, so the entry before the last one is the viewer.
// Entries further left are supplied by the client and can't be trusted.
func ClientIPFromRequest(r *http.Request) string {
	if v := r.Header.Get("CloudFront-Viewer-Address"); v != "" {
		// Formatted as ip:port; IPv6 addresses aren't bracketed.
		if i := strings.LastIndex(v, ":"); i > 0 {
			if ip := net.ParseIP(v[:i]); ip != nil {
				return ip.String()
			}
		}
	}
	var hops []string
	for _, v := range r.Header.Values("X-Forwarded-For") {
		for _, hop := range strings.Split(v, ",") {
			if hop = strings.TrimSpace(hop); hop != "" {
				hops = append(hops, hop)
			}
		}
	}
	candidate := ""
	switch {
	case len(hops) >= 2:
		candidate = hops[len(hops)-2]
	case len(hops) == 1:
		candidate = hops[0]
	default:
		candidate, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	if ip := net.ParseIP(candidate); ip != nil {
		return ip.String()
	}
	return ""
}

// ValidClientID reports whether s looks like an ID issued by newClientID.
func ValidClientID(s string) bool {
	if len(s) != base64.RawURLEncoding.EncodedLen(clientIDBytes) {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(s)
	return err == nil
}

func newClientID() string {
	b := make([]byte, clientIDBytes)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// ClientRecorder persists where users connect from.
type ClientRecorder interface {
	RecordClient(ctx context.Context, uuid string, ip string, clientID string) error
}

// RecordClient records the request's client IP and ID against the user. It is
// best-effort: failures are logged, never returned.
func RecordClient(ctx context.Context, rec ClientRecorder, uuid string) {
	ip := ClientIP(ctx)
	if ip == "" {
		return
	}
	if err := rec.RecordClient(ctx, uuid, ip, ClientID(ctx)); err != nil {
		zerolog.Ctx(ctx).Err(err).Str("userID", uuid).Msg("record-client")
	}
}
