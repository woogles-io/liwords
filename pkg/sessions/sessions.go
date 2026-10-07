package sessions

import (
	"context"

	"github.com/woogles-io/liwords/pkg/entity"
)

// SessionStore is a session store
type SessionStore interface {
	Get(ctx context.Context, sessionID string) (*entity.Session, error)
	New(ctx context.Context, user *entity.User) (*entity.Session, error)
	SetCSRFToken(ctx context.Context, sess *entity.Session, csrfToken string) error
	Delete(ctx context.Context, sess *entity.Session) error
	// DeleteForUser deletes every session belonging to the user, logging them
	// out everywhere. It returns the number of sessions deleted.
	DeleteForUser(ctx context.Context, userUUID string) (int64, error)
	ExtendExpiry(ctx context.Context, s *entity.Session) error
}
