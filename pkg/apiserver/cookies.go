package apiserver

import (
	"context"
	"errors"
	"net/http"

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

type ctxkey string

const rwkey ctxkey = "responsewriter"
const sesskey ctxkey = "session"

// AuthenticationMiddlewareGenerator generates auth middleware that looks up
// a session ID
func AuthenticationMiddlewareGenerator(sessionStore sessions.SessionStore) (mw func(http.Handler) http.Handler) {
	mw = func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := zerolog.Ctx(r.Context())
			sessionID, err := r.Cookie("sessionid")
			if err != nil {
				if err != http.ErrNoCookie {
					log.Err(err).Msg("error-getting-cookie")
				} else {
					// No problem, this user will not be authenticated.
					log.Debug().Msg("unauthenticated request")
					h.ServeHTTP(w, r)
					return
				}
			}

			ctx := r.Context()
			session, err := sessionStore.Get(ctx, sessionID.Value)
			if err != nil {
				log.Err(err).Msg("error-getting-session")
				// Just serve, unauthenticated.
				h.ServeHTTP(w, r)
				return
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
		return nil, errors.New("authentication required")
	}
	sess, ok := sessval.(*entity.Session)
	if !ok {
		return nil, errors.New("unexpected error with type inference")
	}
	return sess, nil
}
