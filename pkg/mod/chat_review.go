package mod

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	pb "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

// The channel prefixes a mod may scan. Anything else is refused, so a request
// cannot smuggle LIKE wildcards into the scan's pattern.
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
	// messages walked per step, newest first: only their ids and channels
	// are read.
	chatScanStepCount = 500
	// steps per request; More asks again with the returned cursor.
	chatScanMaxSteps = 2
	// a request stops early once it has this many channels to return; a
	// search by user lists this many of the channels they wrote in a page.
	chatScanMaxChannels = 100
	// ended games are read this many at a time, a page of a search by user,
	// up to the cap.
	chatReviewGamePage      = 100
	chatReviewMaxEndedGames = 2000
	// a search by name lists at most this many tournaments, and as many
	// leagues.
	chatFindMaxMatches = 20
)

// likeEscaper keeps a typed % or _ from acting as a LIKE wildcard.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// said keeps the messages people sent. A line the server posted itself, such
// as a challenged-off play's returned tiles, has no sender and is left out:
// chat review is about what people said, and a game chat holding only such
// lines stays out of the lists, as it was before the server posted them.
func said(msgs []*ipc.ChatMessage) []*ipc.ChatMessage {
	out := make([]*ipc.ChatMessage, 0, len(msgs))
	for _, m := range msgs {
		if m.UserId != "" || m.Username != "" {
			out = append(out, m)
		}
	}
	return out
}

// summaries counts each channel's messages per sender in the database,
// without reading them, leaving out what said leaves out. Every channel
// gets a summary, in the order given; one with nothing to count has no
// senders. With a user given, a summary also tells when they last wrote
// there.
func (ms *ModService) summaries(ctx context.Context, channels []string, uid, username string) ([]*pb.ChatChannelSummary, error) {
	if len(channels) == 0 {
		return nil, nil
	}
	rows, err := ms.queries.ChatChannelSenders(ctx, channels)
	if err != nil {
		return nil, err
	}
	byChannel := make(map[string]*pb.ChatChannelSummary, len(channels))
	out := make([]*pb.ChatChannelSummary, 0, len(channels))
	for _, ch := range channels {
		if byChannel[ch] == nil {
			byChannel[ch] = &pb.ChatChannelSummary{Channel: ch}
			out = append(out, byChannel[ch])
		}
	}
	senders := map[[2]string]*pb.ChatSenderCount{}
	for _, r := range rows {
		s := byChannel[r.Channel]
		if s == nil {
			continue
		}
		first, last := r.FirstAt.Time.UnixMilli(), r.LastAt.Time.UnixMilli()
		if s.MessageCount == 0 || first < s.FirstTimestamp {
			s.FirstTimestamp = first
		}
		s.LastTimestamp = max(s.LastTimestamp, last)
		s.MessageCount += r.MessageCount
		// one count per person, by their id where the message has one: a
		// user who changed names is counted once, under the newest name (the
		// rows come in the order of their last messages).
		key := [2]string{r.Channel, r.SenderUuid}
		if r.SenderUuid == "" {
			key[1] = "name:" + r.SenderUsername
		}
		sc := senders[key]
		if sc == nil {
			sc = &pb.ChatSenderCount{UserId: r.SenderUuid}
			senders[key] = sc
			s.Senders = append(s.Senders, sc)
		}
		sc.Username = r.SenderUsername
		sc.Count += r.MessageCount
		if username != "" && ((uid != "" && r.SenderUuid == uid) || strings.EqualFold(r.SenderUsername, username)) {
			s.UserLastTimestamp = max(s.UserLastTimestamp, last)
		}
	}
	for _, s := range out {
		sort.SliceStable(s.Senders, func(i, j int) bool {
			return s.Senders[i].Count > s.Senders[j].Count
		})
	}
	return out, nil
}

// gameOf is the id of the game whose players' or viewers' chat channel is.
func gameOf(channel string) (string, bool) {
	for _, p := range []string{"chat.game.", "chat.gametv."} {
		if strings.HasPrefix(channel, p) {
			return strings.TrimPrefix(channel, p), true
		}
	}
	return "", false
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

// leagueChannel is a league's chat channel: its id without the dashes.
func leagueChannel(id string) string {
	return "chat.league." + strings.ReplaceAll(id, "-", "")
}

// pagePath is the path of a page's address, given with or without its scheme
// and host, and without a query or a trailing slash: "woogles.io/club/x/"
// gives "/club/x". Anything else comes back as it is.
func pagePath(s string) string {
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+len("://"):]
	}
	if i := strings.Index(s, "/"); i > 0 && strings.Contains(s[:i], ".") {
		s = s[i:]
	}
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 1 {
		s = strings.TrimRight(s, "/")
	}
	return s
}

// findChannels turns what a mod typed into the channels it names: a channel
// name as it is; else first the exact matches, a game, tournament, club or
// league by its id or by its page's address or path, and then always the
// tournaments, clubs and leagues whose name or slug contains it, newest
// first.
func (ms *ModService) findChannels(ctx context.Context, find string) ([]string, error) {
	find = strings.TrimSpace(find)
	if find == "" {
		return nil, nil
	}
	if strings.HasPrefix(find, "chat.") {
		return []string{find}, nil
	}
	var channels []string
	seen := map[string]bool{}
	add := func(chs ...string) {
		for _, ch := range chs {
			if !seen[ch] {
				seen[ch] = true
				channels = append(channels, ch)
			}
		}
	}
	// A lookup that finds nothing is no error.
	missed := func(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
	game := func(id string) error {
		_, err := ms.queries.GetGameMetadata(ctx, pgtype.Text{String: id, Valid: true})
		if err == nil {
			add("chat.game."+id, "chat.gametv."+id)
		} else if !missed(err) {
			return err
		}
		return nil
	}
	tournamentBySlug := func(slug string) error {
		id, err := ms.queries.GetTournamentUUIDBySlug(ctx, slug)
		if err == nil {
			add("chat.tournament." + id.String)
		} else if !missed(err) {
			return err
		}
		return nil
	}
	leagueBySlug := func(slug string) error {
		lg, err := ms.queries.GetLeagueBySlug(ctx, slug)
		if err == nil {
			add(leagueChannel(lg.Uuid.String()))
		} else if !missed(err) {
			return err
		}
		return nil
	}

	// The exact matches come first: an address's page, or what has the text
	// as its id or as the last part of its address.
	path := pagePath(find)
	switch {
	case strings.HasPrefix(path, "/game/"):
		if err := game(strings.TrimPrefix(path, "/game/")); err != nil {
			return nil, err
		}
	case strings.HasPrefix(path, "/tournament/") || strings.HasPrefix(path, "/club/"):
		if err := tournamentBySlug(path); err != nil {
			return nil, err
		}
	case strings.HasPrefix(path, "/leagues/"):
		if err := leagueBySlug(strings.TrimPrefix(path, "/leagues/")); err != nil {
			return nil, err
		}
	default:
		if err := game(find); err != nil {
			return nil, err
		}
		if _, err := ms.queries.GetTournamentNameAndSlug(ctx, pgtype.Text{String: find, Valid: true}); err == nil {
			add("chat.tournament." + find)
		} else if !missed(err) {
			return nil, err
		}
		if id, err := uuid.Parse(find); err == nil {
			if _, err := ms.queries.GetLeagueByUUID(ctx, id); err == nil {
				add(leagueChannel(id.String()))
			} else if !missed(err) {
				return nil, err
			}
		}
		if !strings.Contains(find, "/") {
			for _, slug := range []string{"/tournament/" + find, "/club/" + find} {
				if err := tournamentBySlug(slug); err != nil {
					return nil, err
				}
			}
			if err := leagueBySlug(find); err != nil {
				return nil, err
			}
		}
	}

	// Then, never instead, whatever has the text (an address, its path) in
	// its name or slug.
	pattern := "%" + likeEscaper.Replace(path) + "%"
	tids, err := ms.queries.SearchTournamentsByNameOrSlug(ctx, models.SearchTournamentsByNameOrSlugParams{
		Pattern: pattern,
		Lim:     chatFindMaxMatches,
	})
	if err != nil {
		return nil, err
	}
	for _, id := range tids {
		add("chat.tournament." + id.String)
	}
	lids, err := ms.queries.SearchLeaguesByNameOrSlug(ctx, models.SearchLeaguesByNameOrSlugParams{
		Pattern: pattern,
		Lim:     chatFindMaxMatches,
	})
	if err != nil {
		return nil, err
	}
	for _, id := range lids {
		add(leagueChannel(id))
	}
	return channels, nil
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
	containing := strings.TrimSpace(req.Msg.Containing)
	resp := &pb.ListChatChannelsResponse{NextCursor: "0"}
	addContexts := func() {
		for _, summary := range resp.Channels {
			summary.Context = ms.channelContext(ctx, summary.Channel)
		}
	}

	// Named channels (a game id or channel looked up directly, or what a
	// find names): summarize just those, listing them even when they hold
	// no chat.
	channels := req.Msg.Channels
	if req.Msg.Find != "" {
		channels, err = ms.findChannels(ctx, req.Msg.Find)
		if err != nil {
			return nil, err
		}
		if len(channels) == 0 {
			return connect.NewResponse(resp), nil
		}
	}
	if len(channels) > 0 {
		for _, ch := range channels {
			if !strings.HasPrefix(ch, "chat.") {
				return nil, apiserver.InvalidArg("channel must start with chat.")
			}
		}
		resp.Channels, err = ms.summaries(ctx, channels, "", "")
		if err != nil {
			return nil, err
		}
		resp.Channels, err = ms.keepMatching(ctx, resp.Channels, containing)
		if err != nil {
			return nil, err
		}
		addContexts()
		return connect.NewResponse(resp), nil
	}

	// One user's chats: a page at a time, the cursor its number.
	if username != "" {
		u, err := ms.userStore.Get(ctx, username)
		if err != nil {
			return nil, apiserver.InvalidArg("user not found: " + username)
		}
		page := 0
		if c := req.Msg.Cursor; c != "" && c != "0" {
			page, err = strconv.Atoi(c)
			if err != nil || page < 0 {
				return nil, apiserver.InvalidArg("bad cursor")
			}
		}
		var more bool
		resp.Channels, more, err = ms.userChannels(ctx, u, username, prefix, page)
		if err != nil {
			return nil, err
		}
		resp.Channels, err = ms.keepMatching(ctx, resp.Channels, containing)
		if err != nil {
			return nil, err
		}
		if more {
			resp.NextCursor = strconv.Itoa(page + 1)
		}
		addContexts()
		return connect.NewResponse(resp), nil
	}

	// Anyone's chats: the channels by their newest message, newest first,
	// found by walking the messages newest first, a batch per step; the
	// cursor is the oldest message walked. A channel may come again on a
	// later page, from its older messages; the page keeps the first.
	var before pgtype.UUID
	if c := req.Msg.Cursor; c != "" && c != "0" {
		id, err := uuid.Parse(c)
		if err != nil {
			return nil, apiserver.InvalidArg("bad cursor")
		}
		before = pgtype.UUID{Bytes: id, Valid: true}
	}
	seen := map[string]bool{}
	var recent []string
	for step := 0; step < chatScanMaxSteps; step++ {
		rows, err := ms.queries.RecentChatChannelMessages(ctx, models.RecentChatChannelMessagesParams{
			Prefix: prefix,
			Before: before,
			Lim:    chatScanStepCount,
		})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			if !seen[r.Channel] {
				seen[r.Channel] = true
				recent = append(recent, r.Channel)
			}
		}
		if len(rows) < chatScanStepCount {
			resp.NextCursor = "0"
			break
		}
		last := rows[len(rows)-1].ID
		before = pgtype.UUID{Bytes: last, Valid: true}
		resp.NextCursor = last.String()
		if len(recent) >= chatScanMaxChannels {
			break
		}
	}
	summaries, err := ms.summaries(ctx, recent, "", "")
	if err != nil {
		return nil, err
	}
	for _, summary := range summaries {
		if summary.MessageCount > 0 {
			resp.Channels = append(resp.Channels, summary)
		}
	}
	resp.Channels, err = ms.keepMatching(ctx, resp.Channels, containing)
	if err != nil {
		return nil, err
	}
	addContexts()
	return connect.NewResponse(resp), nil
}

// keepMatching keeps, with words to find, the summaries of the channels with
// a message that holds them, in any case, each saying how many do; with
// none, all of them.
func (ms *ModService) keepMatching(ctx context.Context, summaries []*pb.ChatChannelSummary, containing string) ([]*pb.ChatChannelSummary, error) {
	if containing == "" || len(summaries) == 0 {
		return summaries, nil
	}
	channels := make([]string, len(summaries))
	for i, summary := range summaries {
		channels[i] = summary.Channel
	}
	rows, err := ms.queries.ChatChannelMatches(ctx, models.ChatChannelMatchesParams{
		Channels: channels,
		Pattern:  "%" + likeEscaper.Replace(containing) + "%",
	})
	if err != nil {
		return nil, err
	}
	matches := make(map[string]int32, len(rows))
	for _, r := range rows {
		matches[r.Channel] = r.MatchCount
	}
	kept := make([]*pb.ChatChannelSummary, 0, len(rows))
	for _, summary := range summaries {
		if n := matches[summary.Channel]; n > 0 {
			summary.MatchCount = n
			kept = append(kept, summary)
		}
	}
	return kept, nil
}

// userChannels lists a page of what a search for user u turns up, any age:
// the channels they wrote in, by when they last wrote there; the chats of the
// games they played, most recent first, even where they wrote nothing, so a
// complaint about an unnamed game or opponent still turns it up; and, on the
// first page, their private chats, even ones they never wrote in. A channel
// with nothing to count is left out. more says whether a later page may hold
// more.
func (ms *ModService) userChannels(ctx context.Context, u *entity.User, username, prefix string, page int) ([]*pb.ChatChannelSummary, bool, error) {
	wrote, err := ms.queries.ChatChannelsBySender(ctx, models.ChatChannelsBySenderParams{
		UserUuid: u.UUID,
		Prefix:   prefix,
		Lim:      chatScanMaxChannels,
		Off:      int32(page * chatScanMaxChannels),
	})
	if err != nil {
		return nil, false, err
	}
	more := len(wrote) == chatScanMaxChannels
	channels := append([]string{}, wrote...)

	played := map[string]bool{}
	if prefix == "" || prefix == "chat.game." || prefix == "chat.gametv." {
		gameIDs, gamesMore, err := ms.userGameIDs(ctx, u, page)
		if err != nil {
			return nil, false, err
		}
		more = more || gamesMore
		for _, gid := range gameIDs {
			played[gid] = true
			for _, p := range []string{"chat.game.", "chat.gametv."} {
				if prefix == "" || prefix == p {
					channels = append(channels, p+gid)
				}
			}
		}
	}
	if page == 0 && (prefix == "" || prefix == "chat.pm.") {
		private, err := ms.queries.ChatPrivateChannelsOf(ctx, u.UUID)
		if err != nil {
			return nil, false, err
		}
		channels = append(channels, private...)
	}

	// Whether they played the games they wrote about, which another page's
	// games may hold.
	var asked []string
	for _, ch := range wrote {
		if gid, ok := gameOf(ch); ok && !played[gid] {
			asked = append(asked, gid)
		}
	}
	if len(asked) > 0 {
		ids, err := ms.queries.GamesPlayedBy(ctx, models.GamesPlayedByParams{UserUuid: u.UUID, GameIds: asked})
		if err != nil {
			return nil, false, err
		}
		for _, id := range ids {
			played[id] = true
		}
	}

	summaries, err := ms.summaries(ctx, channels, u.UUID, username)
	if err != nil {
		return nil, false, err
	}
	listed := make([]*pb.ChatChannelSummary, 0, len(summaries))
	for _, summary := range summaries {
		if summary.MessageCount == 0 {
			continue
		}
		if gid, ok := gameOf(summary.Channel); ok && played[gid] {
			summary.UserPlayed = true
		}
		listed = append(listed, summary)
	}
	return listed, more, nil
}

func (ms *ModService) GetChannelChats(ctx context.Context, req *connect.Request[pb.GetChannelChatsRequest],
) (*connect.Response[pb.ChannelChats], error) {
	_, err := authenticateMod(ctx, ms, nil)
	if err != nil {
		return nil, err
	}
	channel := req.Msg.Channel
	if !strings.HasPrefix(channel, "chat.") {
		return nil, apiserver.InvalidArg("channel must start with chat.")
	}
	reply := func(page *user.ChatPage, msgs []*ipc.ChatMessage) (*connect.Response[pb.ChannelChats], error) {
		reported, err := ms.reportedOf(ctx, msgs)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&pb.ChannelChats{Messages: msgs, HasOlder: page.HasOlder,
			HasNewer: page.HasNewer, ReportedIds: reported}), nil
	}
	// With words to find: only the messages that hold them, a page at a
	// time, newest first.
	if containing := strings.TrimSpace(req.Msg.Containing); containing != "" {
		page, err := ms.chatStore.Containing(ctx, channel, containing, req.Msg.Before)
		if err != nil {
			return nil, err
		}
		return reply(page, page.Messages)
	}
	set := 0
	for _, id := range []string{req.Msg.Before, req.Msg.After, req.Msg.Around} {
		if id != "" {
			set++
		}
	}
	if set > 1 {
		return nil, apiserver.InvalidArg("ask before, after or around one message, not more")
	}
	// One page at a time: the newest, older or newer ones on request, or the
	// one around a message: a channel keeps every message, so it may hold
	// more than a page can show. The page is the chat store's own, as the
	// chat pages. A page holding only the server's lines is passed over, the
	// way it was asked, so a page with more behind it always has one to ask
	// from.
	q := user.ChatQuery{Before: req.Msg.Before, After: req.Msg.After, Around: req.Msg.Around}
	for {
		page, err := ms.chatStore.History(ctx, channel, q)
		if err != nil {
			return nil, err
		}
		msgs := said(page.Messages)
		if len(msgs) > 0 || len(page.Messages) == 0 || q.Around != "" {
			return reply(page, msgs)
		}
		if q.After != "" {
			if !page.HasNewer {
				return reply(page, msgs)
			}
			q.After = page.Messages[len(page.Messages)-1].Id
			continue
		}
		if !page.HasOlder {
			return reply(page, msgs)
		}
		q.Before = page.Messages[0].Id
	}
}

// userGameIDs lists a page of the games u played, most recent first, and on
// the first page the games in progress too. more says whether another page
// may hold more, up to chatReviewMaxEndedGames ended games in all.
func (ms *ModService) userGameIDs(ctx context.Context, u *entity.User, page int) ([]string, bool, error) {
	var ids []string
	if page == 0 {
		active, err := ms.queries.ListActiveGames(ctx)
		if err != nil {
			return nil, false, err
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
			return nil, false, err
		}
		for _, g := range corres {
			ids = append(ids, g.Uuid.String)
		}
	}
	offset := page * chatReviewGamePage
	if offset >= chatReviewMaxEndedGames {
		return ids, false, nil
	}
	games, err := ms.queries.GetRecentGamesByUserId(ctx, models.GetRecentGamesByUserIdParams{
		UserID:      int32(u.ID),
		NumGames:    chatReviewGamePage,
		OffsetGames: int32(offset),
	})
	if err != nil {
		return nil, false, err
	}
	for _, g := range games {
		ids = append(ids, g.Uuid.String)
	}
	more := len(games) == chatReviewGamePage && offset+chatReviewGamePage < chatReviewMaxEndedGames
	return ids, more, nil
}
