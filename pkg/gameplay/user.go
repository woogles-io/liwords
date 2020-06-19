package gameplay

import (
	"context"

	"github.com/domino14/liwords/pkg/entity"
)

// UserStore is an interface that user stores should implement.
type UserStore interface {
	Get(ctx context.Context, username string) (*entity.User, error)
	New(ctx context.Context, user *entity.User) error
}
