package registration

import (
	"context"
	"errors"
	"strings"

	"github.com/domino14/liwords/pkg/auth"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
)

// RegisterUser registers a user.
func RegisterUser(ctx context.Context, username string, password string, email string,
	userStore user.Store, bot bool) error {
	// username = strings.Rep
	if len(username) < 1 || len(username) > 20 {
		return errors.New("username must be between 1 and 20 letters in length")
	}
	if len(strings.Fields(username)) != 1 {
		return errors.New("username must not contain any whitespace")
	}
	// Should we have other unacceptable usernames?
	if strings.ToLower(username) == "anonymous" {
		return errors.New("username is not acceptable")
	}
	if len(password) < 8 {
		return errors.New("your new password is too short, use 8 or more characters")
	}
	if len(email) < 3 {
		return errors.New("please use a valid email address")
	}

	// time, memory, threads, keyLen for argon2:
	config := auth.NewPasswordConfig(1, 64*1024, 4, 32)
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
	return err
}
