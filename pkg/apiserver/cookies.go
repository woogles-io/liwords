package apiserver

import (
	"context"
	"errors"
	"net/http"
	"time"

	twirp "github.com/twitchtv/twirp"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/sessions"
	"github.com/rs/zerolog"
)

// WithCookies configures an http.Handler (like any Twirp server) to enable
// setting cookies with the SetCookie function.
func WithCookiesMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, rwkey, w)
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}

// SetCookie sets an http Cookie for a response being handled in the given
// context. It returns an error if and only if the context has not been
// configured through the WithCookies function.
func SetCookie(ctx context.Context, cookie *http.Cookie) error {
	w, ok := ctx.Value(rwkey).(http.ResponseWriter)
	if !ok {
		return errors.New("unable to get ResponseWriter from context, middleware might not be set up correctly")
	}
	http.SetCookie(w, cookie)
	return nil
}

func SetDefaultCookie(ctx context.Context, sessID string) error {
	cookie := &http.Cookie{
		Name:  "session",
		Value: sessID,
		// Tell the browser the cookie expires after a year, but the actual
		// session ID in the database will expire sooner than that.
		// We will write middleware to extend the expiration length but maybe
		// it's ok to require the user to log in once a year.
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	}

	return SetCookie(ctx, cookie)
}

type ctxkey string

const rwkey ctxkey = "responsewriter"
const sesskey ctxkey = "session"

const RenewCookieTimer = time.Hour * 24 * 14

// AuthenticationMiddlewareGenerator generates auth middleware that looks up
// a session ID, and attaches a Session to the request context (at `sesskey`)
func AuthenticationMiddlewareGenerator(sessionStore sessions.SessionStore) (mw func(http.Handler) http.Handler) {
	mw = func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := zerolog.Ctx(ctx)
			var session *entity.Session
			var err error
			// Migrate old sessionid to session
			oldSessionID, err := r.Cookie("sessionid")
			authed := false
			if err != nil {
				// Don't worry about it, do nothing.
			} else {
				session, err = sessionStore.Get(ctx, oldSessionID.Value)
				if err != nil {
					log.Err(err).Msg("error-getting-session")
					// Just serve, unauthenticated.
					h.ServeHTTP(w, r)
					return
				}
				authed = true
				// Make new cookie
				SetDefaultCookie(r.Context(), oldSessionID.Value)
			}

			if !authed {
				// Try the new cookie.
				sessionID, err := r.Cookie("session")
				if err != nil {
					if err != http.ErrNoCookie {
						log.Err(err).Msg("error-getting-new-cookie")
					}
					// No problem, this user will not be authenticated.
					log.Debug().Msg("unauthenticated request")
					h.ServeHTTP(w, r)
					return
				}
				session, err = sessionStore.Get(ctx, sessionID.Value)
				if err != nil {
					log.Err(err).Msg("error-getting-session")
					// Just serve, unauthenticated.
					h.ServeHTTP(w, r)
					return
				}
			}
			if time.Until(session.Expiry) < RenewCookieTimer {
				err := sessionStore.ExtendExpiry(ctx, session)
				log.Err(err).Msg("extending-session")
			}
			ctx = context.WithValue(ctx, sesskey, session)
			r = r.WithContext(ctx)
			// printContextInternals(r.Context(), true)
			h.ServeHTTP(w, r)
		})
	}
	return
}
func GetSession(ctx context.Context) (*entity.Session, error) {
	sessval := ctx.Value(sesskey)
	if sessval == nil {
		return nil, twirp.NewError(twirp.Unauthenticated, "authentication required")
	}
	sess, ok := sessval.(*entity.Session)
	if !ok {
		return nil, twirp.InternalErrorWith(errors.New("unexpected error with session type inference"))
	}
	return sess, nil
}
