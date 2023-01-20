package apiserver

import (
	"context"
	"errors"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
)

var ErrAuthFailed = errors.New("auth-methods-failed")

var ApiKeyFirst = []entity.AuthMethod{entity.AuthMethodAPIKey, entity.AuthMethodCookie}
var CookieOnly = []entity.AuthMethod{entity.AuthMethodCookie}
var CookieFirst = []entity.AuthMethod{entity.AuthMethodCookie, entity.AuthMethodAPIKey}

// AuthUser uses the passed-in auth methods to authenticate the user.
// It uses the first one that passes.
// If the method fails outright though, it returns an error; i.e. if the
// API key is wrong or the cookie has expired, it won't try the other method.
func AuthUser(ctx context.Context, authMethods []entity.AuthMethod, userStore user.Store) (*entity.User, error) {

	for _, method := range authMethods {
		switch method {
		case entity.AuthMethodCookie:
			sess, err := GetSession(ctx)
			if err != nil {
				continue
			}
			user, err := userStore.GetByUUID(ctx, sess.UserUUID)
			if err != nil {
				return nil, err
			}
			return user, nil

		case entity.AuthMethodAPIKey:
			apikey, err := GetAPIKey(ctx)
			if err != nil {
				continue
			}
			user, err := userStore.GetByAPIKey(ctx, apikey)
			if err != nil {
				return nil, err
			}
			return user, nil
		}
	}
	return nil, ErrAuthFailed
}
