package sessions

import (
	"context"

	"github.com/domino14/liwords/pkg/entity"
)

// SessionStore is a session store
type SessionStore interface {
	Get(ctx context.Context, sessionID string) (*entity.Session, error)
	New(ctx context.Context, user *entity.User) (*entity.Session, error)
	Delete(ctx context.Context, sess *entity.Session) error
}
