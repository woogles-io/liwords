package mod

import (
	"context"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

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

// channelContext decodes a channel name into what it is about: a game's
// players and times, a private chat's two users, a league or tournament room.
// Lookups that fail leave the parts they would have filled empty.
func (ms *ModService) channelContext(ctx context.Context, channel string) *pb.ChatChannelContext {
	c := &pb.ChatChannelContext{}
	userParty := func(uid string) *pb.ChatSenderCount {
		p := &pb.ChatSenderCount{UserId: uid}
		if u, err := ms.userStore.GetByUUID(ctx, uid); err == nil {
			p.Username = u.Username
		}
		return p
	}
	switch {
	case strings.HasPrefix(channel, "chat.game.") || strings.HasPrefix(channel, "chat.gametv."):
		gid := channel[strings.LastIndex(channel, ".")+1:]
		c.Link = "/game/" + gid
		g, err := ms.queries.GetGameMetadata(ctx, pgtype.Text{String: gid, Valid: true})
		if err != nil {
			return c
		}
		c.GameCreatedAt = g.CreatedAt.Time.UnixMilli()
		c.GameUpdatedAt = g.UpdatedAt.Time.UnixMilli()
		var names []string
		for _, pi := range g.Quickdata.PlayerInfo {
			c.Parties = append(c.Parties, &pb.ChatSenderCount{UserId: pi.UserId, Username: pi.Nickname})
			names = append(names, pi.Nickname)
		}
		c.Label = strings.Join(names, " vs ")
		if strings.HasPrefix(channel, "chat.gametv.") {
			c.Label += " (viewers)"
		}
	case strings.HasPrefix(channel, "chat.pm."):
		var names []string
		for _, uid := range strings.Split(strings.TrimPrefix(channel, "chat.pm."), "_") {
			p := userParty(uid)
			c.Parties = append(c.Parties, p)
			names = append(names, p.Username)
		}
		c.Label = "Private: " + strings.Join(names, " & ")
	case strings.HasPrefix(channel, "chat.league."):
		id, err := uuid.Parse(strings.TrimPrefix(channel, "chat.league."))
		if err != nil {
			return c
		}
		if lg, err := ms.queries.GetLeagueByUUID(ctx, id); err == nil {
			c.Label = "League: " + lg.Name
			c.Link = "/leagues/" + lg.Slug
		}
	case strings.HasPrefix(channel, "chat.tournament."):
		t, err := ms.queries.GetTournamentNameAndSlug(ctx, pgtype.Text{String: strings.TrimPrefix(channel, "chat.tournament."), Valid: true})
		if err != nil {
			return c
		}
		c.Link = t.Slug.String
		kind := "Tournament"
		if strings.HasPrefix(t.Slug.String, "/club/") {
			kind = "Club"
		}
		c.Label = kind + ": " + t.Name.String
	case channel == "chat.lobby":
		c.Label = "Lobby"
		c.Link = "/"
	}
	return c
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

	// Named channels (a game id or channel looked up directly): summarize
	// just those, listing them even when they hold no chat.
	if len(req.Msg.Channels) > 0 {
		for _, ch := range req.Msg.Channels {
			if !strings.HasPrefix(ch, "chat.") {
				return nil, apiserver.InvalidArg("channel must start with chat.")
			}
			msgs, err := ms.chatStore.OldChats(ctx, ch, chatReviewMaxMessages)
			if err != nil {
				return nil, err
			}
			summary, _ := summarizeChannel(ch, msgs, "", "")
			summary.Context = ms.channelContext(ctx, ch)
			resp.Channels = append(resp.Channels, summary)
		}
		resp.NextCursor = "0"
		return connect.NewResponse(resp), nil
	}

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
	for _, summary := range resp.Channels {
		summary.Context = ms.channelContext(ctx, summary.Channel)
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
