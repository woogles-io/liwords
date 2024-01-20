package sessions

import (
	"context"

	"github.com/woogles-io/liwords/pkg/entity"
)

// SessionStore is a session store
type SessionStore interface {
	Get(ctx context.Context, sessionID string) (*entity.Session, error)
	New(ctx context.Context, user *entity.User) (*entity.Session, error)
	Delete(ctx context.Context, sess *entity.Session) error
	ExtendExpiry(ctx context.Context, s *entity.Session) error
}
