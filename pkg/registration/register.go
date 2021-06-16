package registration

import (
	"context"
	"errors"
	"strings"

	"github.com/domino14/liwords/pkg/auth"
	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	"github.com/domino14/liwords/pkg/utilities"
	"github.com/lib/pq"
)

// RegisterUser registers a user.
func RegisterUser(ctx context.Context, username string, password string, email string,
	userStore user.Store, bot bool, argonConfig config.ArgonConfig) error {
	// username = strings.Rep
	if len(username) < 3 || len(username) > 20 {
		return errors.New("username must be between 3 and 20 letters in length")
	}
	if strings.IndexFunc(username, func(c rune) bool {
		return !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '.' || c == '_')
	}) != -1 {
		return errors.New("username can only contain letters, digits, period, hyphen or underscore")
	}
	// Should we have other unacceptable usernames?
	if strings.EqualFold(username, "anonymous") || strings.EqualFold(username, utilities.CensoredUsername) {
		return errors.New("username is not acceptable")
	}
	if strings.HasPrefix(username, "-") || strings.HasPrefix(username, ".") || strings.HasPrefix(username, "_") {
		return errors.New("username must start with a number or a letter")
	}
	if strings.HasSuffix(username, "-") || strings.HasSuffix(username, ".") || strings.HasSuffix(username, "_") {
		return errors.New("username must end with a number or a letter")
	}

	if len(password) < 8 {
		return errors.New("your new password is too short, use 8 or more characters")
	}
	if len(email) < 3 {
		return errors.New("please use a valid email address")
	}
	email = strings.TrimSpace(email)

	config := auth.NewPasswordConfig(argonConfig.Time, argonConfig.Memory, argonConfig.Threads, argonConfig.Keylen)
	hashPass, err := auth.GeneratePassword(config, password)
	if err != nil {
		return err
	}
	err = userStore.New(ctx, &entity.User{
		Username: username,
		Password: hashPass,
		Email:    email,
		IsBot:    bot,
	})
	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			// https://www.postgresql.org/docs/current/errcodes-appendix.html
			if err.Code == "23505" {
				if err.Constraint == "username_idx" {
					return errors.New("That username has already been signed up, please log in")
				} else if err.Constraint == "email_idx" {
					return errors.New("That email address has already been signed up, please log in with your existing username")
				}
			}
		}
	}
	return err
}
