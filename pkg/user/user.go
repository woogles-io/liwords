package user

import (
	"context"

	"github.com/domino14/liwords/pkg/entity"
)

// Store is an interface that user stores should implement.
type Store interface {
	Get(ctx context.Context, username string) (*entity.User, error)
	GetByUUID(ctx context.Context, uuid string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	// Username by UUID. Good for fast lookups.
	Username(ctx context.Context, uuid string) (string, bool, error)
	New(ctx context.Context, user *entity.User) error
	SetPassword(ctx context.Context, uuid string, hashpass string) error
	SetRating(ctx context.Context, uuid string, variant entity.VariantKey, rating entity.SingleRating) error
	SetRatings(ctx context.Context, p0uuid string, p1uuid string, variant entity.VariantKey,
		p1Rating entity.SingleRating, p2Rating entity.SingleRating) error
	SetStats(ctx context.Context, uuid string, variant entity.VariantKey, stats *entity.Stats) error
	GetRandomBot(ctx context.Context) (*entity.User, error)

	AddFollower(ctx context.Context, targetUser, follower uint) error
	RemoveFollower(ctx context.Context, targetUser, follower uint) error
	// GetFollows gets all the users that the passed-in DB ID is following.
	GetFollows(ctx context.Context, uid uint) ([]*entity.User, error)

	AddBlock(ctx context.Context, targetUser, blocker uint) error
	RemoveBlock(ctx context.Context, targetUser, blocker uint) error
	// GetBlocks gets all the users that the passed-in DB ID is blocking
	GetBlocks(ctx context.Context, uid uint) ([]*entity.User, error)
	GetBlockedBy(ctx context.Context, uid uint) ([]*entity.User, error)
	GetFullBlocks(ctx context.Context, uid uint) ([]*entity.User, error)

	UsernamesByPrefix(ctx context.Context, prefix string) ([]string, error)
}

// PresenceStore stores user presence. Since it is meant to be easily user-visible,
// we deal with unique usernames in addition to UUIDs.
type PresenceStore interface {
	// SetPresence sets the presence. If channel is the string NULL this is
	// equivalent to saying the user logged off.
	SetPresence(ctx context.Context, uuid, username string, anon bool, channel string) error
	ClearPresence(ctx context.Context, uuid, username string, anon bool) (string, error)
	GetPresence(ctx context.Context, uuid string) (string, error)

	CountInChannel(ctx context.Context, channel string) (int, error)
	GetInChannel(ctx context.Context, channel string) ([]*entity.User, error)
	// BatchGetPresence returns a list of the users with their presence.
	// Can use for buddy/follower lists.
	BatchGetPresence(ctx context.Context, users []*entity.User) ([]*entity.User, error)
}
