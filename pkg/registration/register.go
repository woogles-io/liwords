package registration

import (
	"context"

	"github.com/domino14/liwords/pkg/auth"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
)

// RegisterUser registers a user.
func RegisterUser(ctx context.Context, username string, password string, userStore gameplay.UserStore) error {
	// time, memory, threads, keyLen for argon2:
	config := auth.NewPasswordConfig(1, 64*1024, 4, 32)
	hashPass, err := auth.GeneratePassword(config, password)
	if err != nil {
		return err
	}
	err = userStore.New(ctx, &entity.User{
		Username: username,
		Password: hashPass,
	})
	return err
}
