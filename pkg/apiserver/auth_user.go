package apiserver

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
)

var ErrAuthFailed = errors.New("auth-methods-failed")

// AuthUser uses several auth methods to authenticate the user.
// It uses the first one that passes.
// If the method fails outright though, it returns an error; i.e. if the
// API key is wrong or the cookie has expired, it won't try the other method.
func AuthUser(ctx context.Context, userStore user.Store) (*entity.User, error) {
	for _, method := range []entity.AuthMethod{entity.AuthMethodCookie, entity.AuthMethodAPIKey} {
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
	return nil, Unauthenticated(ErrAuthFailed.Error())
}

func AuthenticateAdmin(ctx context.Context, userStore user.Store, q *models.Queries) error {
	u, err := AuthUser(ctx, userStore)
	if err != nil {
		return err
	}
	roles, err := q.GetUserRoles(ctx, pgtype.Text{Valid: true, String: u.Username})
	if err != nil {
		return err
	}
	for _, r := range roles {
		if r.Name == string(rbac.Admin) {
			log.Info().Str("username", u.Username).Msg("admin-authenticated")
			return nil
		}
	}
	return PermissionDenied("not an admin")
}
