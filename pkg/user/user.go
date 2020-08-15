package user

import (
	"context"

	"github.com/domino14/liwords/pkg/entity"
)

// Store is an interface that user stores should implement.
type Store interface {
	Get(ctx context.Context, username string) (*entity.User, error)
	GetByUUID(ctx context.Context, uuid string) (*entity.User, error)
	// Username by UUID. Good for fast lookups.
	Username(ctx context.Context, uuid string) (string, error)
	New(ctx context.Context, user *entity.User) error
	SetPassword(ctx context.Context, uuid string, hashpass string) error
	SetRating(ctx context.Context, uuid string, variant entity.VariantKey, rating entity.SingleRating) error
	SetStats(ctx context.Context, uuid string, variant entity.VariantKey, stats *entity.Stats) error
	AddFollower(ctx context.Context, targetUser, follower uint) error
	RemoveFollower(ctx context.Context, targetUser, follower uint) error
	// GetFollows gets all the users that the passed-in DB ID is following.
	GetFollows(ctx context.Context, uid uint) ([]*entity.User, error)
}

// SessionStore is a session store
type SessionStore interface {
	Get(ctx context.Context, sessionID string) (*entity.Session, error)
	New(ctx context.Context, user *entity.User) (*entity.Session, error)
	Delete(ctx context.Context, sess *entity.Session) error
}

// PresenceStore stores user presence
type PresenceStore interface {
	SetPresence(ctx context.Context, uuid, channel string) error
	GetInChannel(ctx context.Context, channel string) ([]*entity.User, error)
	// BatchGetPresence returns a list of the users with their presence.
	BatchGetPresence(ctx context.Context, users []*entity.User) ([]*entity.User, error)
}
