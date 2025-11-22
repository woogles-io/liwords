package user

import (
	"context"
	"time"

	"github.com/woogles-io/liwords/pkg/entity"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
	upb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

// Store is an interface that user stores should implement.
type Store interface {
	Get(ctx context.Context, username string) (*entity.User, error)
	GetByUUID(ctx context.Context, uuid string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*entity.User, error)
	GetByVerificationToken(ctx context.Context, token string) (*entity.User, error)
	// Username by UUID. Good for fast lookups.
	Username(ctx context.Context, uuid string) (string, error)
	New(ctx context.Context, user *entity.User) error
	SetPassword(ctx context.Context, uuid string, hashpass string) error
	SetEmailVerified(ctx context.Context, uuid string, verified bool) error
	UpdateVerificationToken(ctx context.Context, uuid string, token string, expiresAt time.Time) error
	DeleteUnverifiedUsers(ctx context.Context, olderThan time.Duration) (int, error)
	SetAvatarUrl(ctx context.Context, uuid string, avatarUrl string) error
	GetBriefProfiles(ctx context.Context, uuids []string) (map[string]*upb.BriefProfile, error)
	SetPersonalInfo(ctx context.Context, uuid string, email string, firstName string, lastName string, birthDate string, countryCode string, about string) error
	SetRatings(ctx context.Context, p0uuid string, p1uuid string, variant entity.VariantKey,
		p1Rating *entity.SingleRating, p2Rating *entity.SingleRating) error
	SetStats(ctx context.Context, p0uuid string, p1uuid string, variant entity.VariantKey,
		p0stats *entity.Stats, p1stats *entity.Stats) error
	SetNotoriety(ctx context.Context, uuid string, notoriety int) error
	ResetRatings(ctx context.Context, uuid string) error
	ResetStats(ctx context.Context, uuid string) error
	ResetProfile(ctx context.Context, uuid string) error
	ResetPersonalInfo(ctx context.Context, uuid string) error
	GetBot(ctx context.Context, botType macondopb.BotRequest_BotCode) (*entity.User, error)

	AddFollower(ctx context.Context, targetUser, follower uint) error
	RemoveFollower(ctx context.Context, targetUser, follower uint) error
	// GetFollows gets all the users that the passed-in DB ID is following.
	GetFollows(ctx context.Context, uid uint) ([]*entity.User, error)
	// GetFollowedBy gets all the users that are following the passed-in user DB ID.
	GetFollowedBy(ctx context.Context, uid uint) ([]*entity.User, error)
	// IsFollowing checks if followerID is following userID.
	IsFollowing(ctx context.Context, followerID, userID uint) (bool, error)

	AddBlock(ctx context.Context, targetUser, blocker uint) error
	RemoveBlock(ctx context.Context, targetUser, blocker uint) error
	// GetBlocks gets all the users that the passed-in DB ID is blocking
	GetBlocks(ctx context.Context, uid uint) ([]*entity.User, error)
	GetBlockedBy(ctx context.Context, uid uint) ([]*entity.User, error)
	GetFullBlocks(ctx context.Context, uid uint) ([]*entity.User, error)

	UsersByPrefix(ctx context.Context, prefix string) ([]*upb.BasicUser, error)
	CachedCount(ctx context.Context) int

	GetAPIKey(ctx context.Context, uuid string) (string, error)
	ResetAPIKey(ctx context.Context, uuid string) (string, error)
	GetActions(ctx context.Context, userUUID string) (map[string]*ms.ModAction, error)
	GetActionsBatch(ctx context.Context, userUUIDs []string) (map[string]map[string]*ms.ModAction, error)
	GetActionHistory(ctx context.Context, userUUID string) ([]*ms.ModAction, error)
	ApplyActions(ctx context.Context, actions []*ms.ModAction) error
	RemoveActions(ctx context.Context, actions []*ms.ModAction) error
}

// PresenceStore stores user presence. Since it is meant to be easily user-visible,
// we deal with unique usernames in addition to UUIDs.
// Presence applies to chat channels, as well as an overall site-wide presence.
// For example, we'd like to see who's online, as well as who's in our given channel
// (i.e. who's watching a certain game with us?)
type PresenceStore interface {
	// SetPresence sets the presence. If channel is the string NULL this is
	// equivalent to saying the user logged off.
	SetPresence(ctx context.Context, uuid, username string, anon bool, channel string, connID string) ([]string, []string, error)
	ClearPresence(ctx context.Context, uuid, username string, anon bool, connID string) ([]string, []string, []string, error)
	GetPresence(ctx context.Context, uuid string) ([]string, error)
	// RenewPresence prevents the presence store from expiring the relevant keys.
	// Basically, we're telling the presence store "this user and connection are still here".
	// Otherwise, missing a few of these events will destroy the relevant presences.
	RenewPresence(ctx context.Context, uuid, username string, anon bool, connID string) ([]string, []string, error)

	CountInChannel(ctx context.Context, channel string) (int, error)
	GetInChannel(ctx context.Context, channel string) ([]*entity.User, error)
	// BatchGetPresence returns a list of the users with their presence.
	// Can use for buddy/follower lists.
	BatchGetPresence(ctx context.Context, users []*entity.User) ([]*entity.User, error)

	LastSeen(ctx context.Context, uuid string) (int64, error)

	SetEventChan(chan *entity.EventWrapper)
	EventChan() chan *entity.EventWrapper

	BatchGetChannels(ctx context.Context, uuids []string) ([][]string, error)
	UpdateFollower(ctx context.Context, followee, follower *entity.User, following bool) error
	UpdateActiveGame(ctx context.Context, activeGameEntry *pb.ActiveGameEntry) ([][][]string, error)
}

// ChatStore stores user and channel chats and messages
type ChatStore interface {
	AddChat(ctx context.Context, senderUsername, senderUID, msg, channel, channelFriendly string, regulateChat bool) (*pb.ChatMessage, error)
	OldChats(ctx context.Context, channel string, n int) ([]*pb.ChatMessage, error)
	LatestChannels(ctx context.Context, count, offset int, uid, tid, lid string) (*upb.ActiveChatChannels, error)

	GetChat(ctx context.Context, channel string, msgID string) (*pb.ChatMessage, error)
	DeleteChat(ctx context.Context, channel string, msgID string) error
	SetEventChan(chan *entity.EventWrapper)
	EventChan() chan *entity.EventWrapper
}
