package user

import (
	"context"

	"github.com/domino14/liwords/pkg/entity"
)

// Store is an interface that user stores should implement.
type Store interface {
	Get(ctx context.Context, username string) (*entity.User, error)
	GetByUUID(ctx context.Context, uuid string) (*entity.User, error)
	New(ctx context.Context, user *entity.User) error
}

// SessionStore is a session store
type SessionStore interface {
	Get(ctx context.Context, sessionID string) (*entity.Session, error)
	New(ctx context.Context, user *entity.User) (*entity.Session, error)
	Delete(ctx context.Context, sess *entity.Session) error
}
