package apiserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/twitchtv/twirp"

	"github.com/rs/zerolog"
)

const ApiKeyHeader = "X-Api-Key"
const apikeykey ctxkey = "apikey"

func StoreAPIKeyInContext(ctx context.Context, apikey string) context.Context {
	ctx = context.WithValue(ctx, apikeykey, apikey)
	return ctx
}

// APIKeyMiddlewareGenerator creates a middleware to fetch an API key from
// a header and store it in a context key.
func APIKeyMiddlewareGenerator() (mw func(http.Handler) http.Handler) {
	mw = func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := zerolog.Ctx(ctx)

			apikey := r.Header[ApiKeyHeader]
			if len(apikey) > 1 {
				log.Error().Msg("apikey formatted incorrectly")
				// Serve, unauthenticated
				h.ServeHTTP(w, r)
				return
			} else if len(apikey) == 0 {
				h.ServeHTTP(w, r)
				return
			}
			// Otherwise, an API key was provided. Store it in the context.
			ctx = StoreAPIKeyInContext(ctx, apikey[0])
			r = r.WithContext(ctx)
			h.ServeHTTP(w, r)
		})
	}
	return
}

// GetAPIKey works with APIKeyMiddlewareGenerator to return an API key in the
// passed-in context.
func GetAPIKey(ctx context.Context) (string, error) {
	apikey := ctx.Value(apikeykey)
	if apikey == nil {
		return "", twirp.NewError(twirp.Unauthenticated, "api key required")
	}
	a, ok := apikey.(string)
	if !ok {
		return "", twirp.InternalErrorWith(errors.New("unexpected error with apikey type inference"))
	}
	return a, nil
}
