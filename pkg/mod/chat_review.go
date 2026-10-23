package mod

import (
	"context"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	pb "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

// chatReviewNow is the clock the recent-game window is measured on.
var chatReviewNow = time.Now

// The channel prefixes a mod may scan. Anything else is refused, so a request
// cannot smuggle glob characters into the SCAN pattern.
var reviewableChannelPrefixes = map[string]bool{
	"":                 true,
	"chat.game.":       true,
	"chat.gametv.":     true,
	"chat.pm.":         true,
	"chat.tournament.": true,
	"chat.league.":     true,
	"chat.lobby":       true,
}

const (
	// keys SCAN is asked to walk per step.
	chatScanStepCount = 500
	// steps per request; the client asks again with the returned cursor.
	chatScanMaxSteps = 20
	// a request stops early once it has this many channels to return.
	chatScanMaxChannels = 100
	// streams are capped near 500 messages; this reads all of one.
	chatReviewMaxMessages = 1000
	// game chat expires 14 days after its last message; a day more covers
	// a game that ended a little after its last chat.
	chatReviewGameWindow = 15 * 24 * time.Hour
	// ended games are read this many at a time, up to the cap.
	chatReviewGamePage      = 100
	chatReviewMaxEndedGames = 2000
)

// summarizeChannel counts a channel's messages per sender. ok is false when
// username is set and the channel neither holds a message from that user nor
// is a private chat naming them.
func summarizeChannel(channel string, msgs []*ipc.ChatMessage, uid, username string) (*pb.ChatChannelSummary, bool) {
	summary := &pb.ChatChannelSummary{Channel: channel, MessageCount: int32(len(msgs))}
	matched := username == ""
	if uid != "" && strings.HasPrefix(channel, "chat.pm.") {
		for _, party := range strings.Split(strings.TrimPrefix(channel, "chat.pm."), "_") {
			if party == uid {
				matched = true
			}
		}
	}
	bySender := map[string]*pb.ChatSenderCount{}
	for i, m := range msgs {
		if i == 0 {
			summary.FirstTimestamp = m.Timestamp
		}
		summary.LastTimestamp = m.Timestamp
		key := m.UserId
		if key == "" {
			key = m.Username
		}
		sc, ok := bySender[key]
		if !ok {
			sc = &pb.ChatSenderCount{UserId: m.UserId, Username: m.Username}
			bySender[key] = sc
			summary.Senders = append(summary.Senders, sc)
		}
		sc.Count++
		if !matched && ((uid != "" && m.UserId == uid) || strings.EqualFold(m.Username, username)) {
			matched = true
		}
	}
	sort.SliceStable(summary.Senders, func(i, j int) bool {
		return summary.Senders[i].Count > summary.Senders[j].Count
	})
	return summary, matched
}

func (ms *ModService) ListChatChannels(ctx context.Context, req *connect.Request[pb.ListChatChannelsRequest],
) (*connect.Response[pb.ListChatChannelsResponse], error) {
	_, err := authenticateMod(ctx, ms, nil)
	if err != nil {
		return nil, err
	}
	prefix := req.Msg.ChannelPrefix
	if !reviewableChannelPrefixes[prefix] {
		return nil, apiserver.InvalidArg("unknown channel prefix")
	}
	username := strings.TrimSpace(req.Msg.Username)
	uid := ""
	resp := &pb.ListChatChannelsResponse{}
	cursor := req.Msg.Cursor
	freshScan := cursor == "" || cursor == "0"
	if cursor == "" {
		cursor = "0"
	}
	// The channels of the games this user played. A game's chat is listed
	// even when the user wrote nothing in it, so a complaint about an
	// unnamed game or opponent still turns it up.
	played := map[string]bool{}
	playedListed := 0
	if username != "" {
		u, err := ms.userStore.Get(ctx, username)
		if err != nil {
			return nil, apiserver.InvalidArg("user not found: " + username)
		}
		uid = u.UUID
		if prefix == "" || prefix == "chat.game." || prefix == "chat.gametv." {
			gameIDs, err := ms.userGameIDs(ctx, u)
			if err != nil {
				return nil, err
			}
			for _, gid := range gameIDs {
				for _, p := range []string{"chat.game.", "chat.gametv."} {
					if prefix == "" || prefix == p {
						played[p+gid] = true
					}
				}
			}
		}
		if freshScan {
			for ch := range played {
				msgs, err := ms.chatStore.OldChats(ctx, ch, chatReviewMaxMessages)
				if err != nil || len(msgs) == 0 {
					continue
				}
				summary, _ := summarizeChannel(ch, msgs, uid, username)
				summary.UserPlayed = true
				resp.Channels = append(resp.Channels, summary)
				playedListed++
			}
		}
	}

	for step := 0; step < chatScanMaxSteps; step++ {
		channels, next, err := ms.chatStore.ScanChannels(ctx, cursor, prefix, chatScanStepCount)
		if err != nil {
			return nil, err
		}
		cursor = next
		for _, ch := range channels {
			if played[ch] {
				continue
			}
			msgs, err := ms.chatStore.OldChats(ctx, ch, chatReviewMaxMessages)
			if err != nil {
				// the key may have expired between SCAN and XREVRANGE.
				continue
			}
			if len(msgs) == 0 {
				continue
			}
			if summary, ok := summarizeChannel(ch, msgs, uid, username); ok {
				resp.Channels = append(resp.Channels, summary)
			}
		}
		if cursor == "0" || len(resp.Channels)-playedListed >= chatScanMaxChannels {
			break
		}
	}
	resp.NextCursor = cursor
	return connect.NewResponse(resp), nil
}

func (ms *ModService) GetChannelChats(ctx context.Context, req *connect.Request[pb.GetChannelChatsRequest],
) (*connect.Response[ipc.ChatMessages], error) {
	_, err := authenticateMod(ctx, ms, nil)
	if err != nil {
		return nil, err
	}
	channel := req.Msg.Channel
	if !strings.HasPrefix(channel, "chat.") {
		return nil, apiserver.InvalidArg("channel must start with chat.")
	}
	msgs, err := ms.chatStore.OldChats(ctx, channel, chatReviewMaxMessages)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ipc.ChatMessages{Messages: msgs}), nil
}

// userGameIDs lists the games u played whose chat may still be in Redis:
// games in progress, and games that ended within the chat expiry.
func (ms *ModService) userGameIDs(ctx context.Context, u *entity.User) ([]string, error) {
	var ids []string
	active, err := ms.queries.ListActiveGames(ctx)
	if err != nil {
		return nil, err
	}
	for _, g := range active {
		for _, pi := range g.Quickdata.PlayerInfo {
			if pi.UserId == u.UUID {
				ids = append(ids, g.Uuid.String)
				break
			}
		}
	}
	corres, err := ms.queries.ListActiveCorrespondenceGamesForUser(ctx, u.UUID)
	if err != nil {
		return nil, err
	}
	for _, g := range corres {
		ids = append(ids, g.Uuid.String)
	}
	cutoff := chatReviewNow().Add(-chatReviewGameWindow)
	for offset := 0; offset < chatReviewMaxEndedGames; offset += chatReviewGamePage {
		games, err := ms.queries.GetRecentGamesByUserId(ctx, models.GetRecentGamesByUserIdParams{
			UserID:      int32(u.ID),
			NumGames:    chatReviewGamePage,
			OffsetGames: int32(offset),
		})
		if err != nil {
			return nil, err
		}
		for _, g := range games {
			if g.UpdatedAt.Time.Before(cutoff) {
				return ids, nil
			}
			ids = append(ids, g.Uuid.String)
		}
		if len(games) < chatReviewGamePage {
			break
		}
	}
	return ids, nil
}
