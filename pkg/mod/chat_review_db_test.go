package mod_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	mspb "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

func TestUserGameIDs(t *testing.T) {
	is := is.New(t)
	pool, stores, cfg := recreateDB()
	defer stores.Disconnect()
	ctx := DefaultConfig.WithContext(context.Background())
	ms := mod.NewModService(stores.UserStore, stores.ChatStore, stores.Queries)

	cesar, err := stores.UserStore.Get(ctx, "cesar")
	is.NoErr(err)

	g, cancel, donechan, _ := makeGame(cfg, stores, 60, pb.RatingMode_RATED)

	// in progress: found among the active games, on the first page.
	ids, more, err := mod.UserGameIDs(ms, ctx, cesar, 0)
	is.NoErr(err)
	is.True(slices.Contains(ids, g.GameID()))
	is.True(!more)

	// ended: found among the recent games, however long ago, as its chat
	// is kept.
	_, err = gameplay.HandleEvent(ctx, stores, cesar.UUID, &pb.ClientGameplayEvent{
		Type: pb.ClientGameplayEvent_RESIGN, GameId: g.GameID(),
	})
	is.NoErr(err)
	for _, table := range []string{"games", "game_players"} {
		_, err = pool.Exec(ctx, "UPDATE "+table+" SET updated_at = updated_at - interval '400 days'")
		is.NoErr(err)
	}
	ids, more, err = mod.UserGameIDs(ms, ctx, cesar, 0)
	is.NoErr(err)
	is.Equal(ids, []string{g.GameID()})
	is.True(!more)
	// the next page holds nothing more.
	ids, more, err = mod.UserGameIDs(ms, ctx, cesar, 1)
	is.NoErr(err)
	is.Equal(len(ids), 0)
	is.True(!more)

	cancel()
	<-donechan
}

func TestChatReviewSummaries(t *testing.T) {
	is := is.New(t)
	_, stores, _ := recreateDB()
	defer stores.Disconnect()
	ctx := DefaultConfig.WithContext(context.Background())
	ms := mod.NewModService(stores.UserStore, stores.ChatStore, stores.Queries)
	cesar, err := stores.UserStore.Get(ctx, "cesar")
	is.NoErr(err)
	jesse, err := stores.UserStore.Get(ctx, "jesse")
	is.NoErr(err)

	ch := "chat.tournament.room"
	say := func(u *entity.User, msg string) *pb.ChatMessage {
		m, err := stores.ChatStore.AddChat(ctx, u.Username, u.UUID, msg, ch, false)
		is.NoErr(err)
		return m
	}
	first := say(cesar, "hi")
	hello := say(jesse, "hello")
	rude := say(jesse, "rude")
	last := say(cesar, "bye")
	// neither a line the server posted nor a deleted message counts.
	_, err = stores.ChatStore.AddChat(ctx, "", "", "Play was challenged off the board!", ch, false)
	is.NoErr(err)
	is.NoErr(stores.ChatStore.DeleteChat(ctx, ch, rude.Id, cesar.UUID))

	sums, err := mod.Summaries(ms, ctx, []string{ch, "chat.tournament.empty"}, jesse.UUID, "jesse")
	is.NoErr(err)
	is.Equal(len(sums), 2)
	s := sums[0]
	is.Equal(s.Channel, ch)
	is.Equal(s.MessageCount, int32(3))
	is.Equal(s.FirstTimestamp, first.Timestamp)
	is.Equal(s.LastTimestamp, last.Timestamp)
	is.Equal(len(s.Senders), 2)
	is.Equal(s.Senders[0].UserId, cesar.UUID)
	is.Equal(s.Senders[0].Username, "cesar")
	is.Equal(s.Senders[0].Count, int32(2))
	is.Equal(s.Senders[1].UserId, jesse.UUID)
	is.Equal(s.Senders[1].Count, int32(1))
	// the user's own last message, not the channel's.
	is.Equal(s.UserLastTimestamp, hello.Timestamp)
	// a channel with nothing to count is still summarized, as named.
	is.Equal(sums[1].Channel, "chat.tournament.empty")
	is.Equal(sums[1].MessageCount, int32(0))
	is.Equal(len(sums[1].Senders), 0)
}

func TestChatReviewUserChannels(t *testing.T) {
	is := is.New(t)
	pool, stores, cfg := recreateDB()
	defer stores.Disconnect()
	ctx := DefaultConfig.WithContext(context.Background())
	ms := mod.NewModService(stores.UserStore, stores.ChatStore, stores.Queries)
	cesar, err := stores.UserStore.Get(ctx, "cesar")
	is.NoErr(err)
	jesse, err := stores.UserStore.Get(ctx, "jesse")
	is.NoErr(err)

	g, cancel, donechan, _ := makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	defer func() {
		cancel()
		<-donechan
	}()
	add := func(u *entity.User, ch, msg string) {
		_, err := stores.ChatStore.AddChat(ctx, u.Username, u.UUID, msg, ch, false)
		is.NoErr(err)
	}
	played := "chat.game." + g.GameID()
	viewers := "chat.gametv." + g.GameID()
	watched := "chat.gametv.someone-elses-game"
	club := "chat.tournament.club-1"
	private := "chat.pm." + cesar.UUID + "_" + jesse.UUID
	add(cesar, played, "gl")
	add(jesse, viewers, "watching my own game")
	add(cesar, club, "hello club")
	add(cesar, watched, "nice play")
	add(jesse, private, "hey")
	add(jesse, "chat.tournament.elsewhere", "nothing of cesar's")
	// a message of long ago is found as well.
	_, err = pool.Exec(ctx, "UPDATE chat_messages SET created_at = created_at - interval '400 days' WHERE channel = $1", watched)
	is.NoErr(err)

	listed, more, err := mod.UserChannels(ms, ctx, cesar, "cesar", "", 0)
	is.NoErr(err)
	is.True(!more)
	got := map[string]*mspb.ChatChannelSummary{}
	for _, s := range listed {
		got[s.Channel] = s
	}
	is.Equal(len(got), 5)
	// the game played, where they wrote and where they did not.
	is.True(got[played].UserPlayed)
	is.True(got[played].UserLastTimestamp > 0)
	is.True(got[viewers].UserPlayed)
	is.Equal(got[viewers].UserLastTimestamp, int64(0))
	// where they wrote, played or not.
	is.True(!got[club].UserPlayed)
	is.True(!got[watched].UserPlayed)
	is.True(got[watched].UserLastTimestamp < time.Now().Add(-300*24*time.Hour).UnixMilli())
	// a private chat they are a party to, though they never wrote in it.
	is.Equal(got[private].MessageCount, int32(1))
	is.Equal(got[private].UserLastTimestamp, int64(0))

	// by type: only that type.
	listed, _, err = mod.UserChannels(ms, ctx, cesar, "cesar", "chat.pm.", 0)
	is.NoErr(err)
	is.Equal(len(listed), 1)
	is.Equal(listed[0].Channel, private)
	listed, _, err = mod.UserChannels(ms, ctx, cesar, "cesar", "chat.gametv.", 0)
	is.NoErr(err)
	is.Equal(len(listed), 2)
	// the next page holds nothing more.
	listed, more, err = mod.UserChannels(ms, ctx, cesar, "cesar", "", 1)
	is.NoErr(err)
	is.Equal(len(listed), 0)
	is.True(!more)

	// which games of those asked they played.
	ids, err := stores.Queries.GamesPlayedBy(ctx, models.GamesPlayedByParams{
		UserUuid: cesar.UUID, GameIds: []string{g.GameID(), "someone-elses-game"},
	})
	is.NoErr(err)
	is.Equal(ids, []string{g.GameID()})
}

func TestGetChannelChatsPages(t *testing.T) {
	is := is.New(t)
	_, stores, _ := recreateDB()
	defer stores.Disconnect()
	ctx := DefaultConfig.WithContext(context.Background())
	ms := mod.NewModService(stores.UserStore, stores.ChatStore, stores.Queries)
	cesar, err := stores.UserStore.Get(ctx, "cesar")
	is.NoErr(err)
	is.NoErr(stores.Queries.AssignRole(ctx, models.AssignRoleParams{
		Username: "cesar", RoleName: string(rbac.Moderator),
	}))
	modCtx := apiserver.PlaceInContext(ctx, &entity.Session{
		ID: "review", Username: "cesar", UserUUID: cesar.UUID, Expiry: time.Now().Add(time.Hour),
	})

	ch := "chat.tournament.big"
	for i := range 250 {
		_, err := stores.ChatStore.AddChat(ctx, "cesar", cesar.UUID, fmt.Sprintf("m%d", i), ch, false)
		is.NoErr(err)
	}
	// the newest line is the server's, which is not shown.
	_, err = stores.ChatStore.AddChat(ctx, "", "", "Play was challenged off the board!", ch, false)
	is.NoErr(err)

	page := func(before string) *mspb.ChannelChats {
		resp, err := ms.GetChannelChats(modCtx, connect.NewRequest(&mspb.GetChannelChatsRequest{Channel: ch, Before: before}))
		is.NoErr(err)
		return resp.Msg
	}
	seen := map[string]bool{}
	before := ""
	var sizes []int
	var olders []bool
	for {
		p := page(before)
		sizes = append(sizes, len(p.Messages))
		olders = append(olders, p.HasOlder)
		for i, m := range p.Messages {
			is.True(m.Message != "Play was challenged off the board!")
			is.True(!seen[m.Id])
			seen[m.Id] = true
			// oldest first, each page older than the one before.
			if i > 0 {
				is.True(p.Messages[i-1].Id < m.Id)
			}
			if before != "" {
				is.True(m.Id < before)
			}
		}
		if !p.HasOlder {
			break
		}
		before = p.Messages[0].Id
	}
	is.Equal(sizes, []int{99, 100, 51})
	is.Equal(olders, []bool{true, true, false})
	is.Equal(len(seen), 250)
}

func TestChannelContext(t *testing.T) {
	is := is.New(t)
	_, stores, cfg := recreateDB()
	defer stores.Disconnect()
	ctx := DefaultConfig.WithContext(context.Background())
	ms := mod.NewModService(stores.UserStore, stores.ChatStore, stores.Queries)

	g, cancel, donechan, _ := makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	defer func() {
		cancel()
		<-donechan
	}()

	c := mod.ChannelContext(ms, ctx, "chat.game."+g.GameID())
	is.Equal(c.Link, "/game/"+g.GameID())
	is.Equal(c.Label, "cesar vs jesse")
	is.Equal(len(c.Parties), 2)
	is.True(c.GameCreatedAt > 0)
	is.True(c.GameUpdatedAt != 0)

	c = mod.ChannelContext(ms, ctx, "chat.gametv."+g.GameID())
	is.Equal(c.Label, "cesar vs jesse (viewers)")

	cesar, err := stores.UserStore.Get(ctx, "cesar")
	is.NoErr(err)
	jesse, err := stores.UserStore.Get(ctx, "jesse")
	is.NoErr(err)
	c = mod.ChannelContext(ms, ctx, "chat.pm."+cesar.UUID+"_"+jesse.UUID)
	is.Equal(c.Label, "Private: cesar & jesse")
	is.Equal(c.Parties[1].UserId, jesse.UUID)

	// Unknown rooms decode to nothing rather than failing.
	c = mod.ChannelContext(ms, ctx, "chat.league.00000000000000000000000000000000")
	is.Equal(c.Label, "")
	c = mod.ChannelContext(ms, ctx, "chat.tournament.nope")
	is.Equal(c.Label, "")
	c = mod.ChannelContext(ms, ctx, "chat.lobby")
	is.Equal(c.Link, "/")
}

func TestFindChannels(t *testing.T) {
	is := is.New(t)
	_, stores, cfg := recreateDB()
	defer stores.Disconnect()
	ctx := DefaultConfig.WithContext(context.Background())
	ms := mod.NewModService(stores.UserStore, stores.ChatStore, stores.Queries)

	g, cancel, donechan, _ := makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	defer func() {
		cancel()
		<-donechan
	}()
	games := []string{"chat.game." + g.GameID(), "chat.gametv." + g.GameID()}

	is.NoErr(stores.TournamentStore.Create(ctx, &entity.Tournament{
		UUID: "club-1", Name: "Philly Mondays", Slug: "/club/philly", Type: entity.TypeClub,
	}))
	is.NoErr(stores.TournamentStore.Create(ctx, &entity.Tournament{
		UUID: "tourney-1", Name: "Fall 100% Open", Slug: "/tournament/fall-open", Type: entity.TypeStandard,
	}))
	is.NoErr(stores.TournamentStore.Create(ctx, &entity.Tournament{
		UUID: "club-2", Name: "Smoke Club Mondays", Slug: "/club/smokeclub", Type: entity.TypeClub,
	}))
	leagueID := uuid.New()
	_, err := stores.Queries.CreateLeague(ctx, models.CreateLeagueParams{
		Uuid: leagueID, Name: "Smoke League", Slug: "smoke", Settings: []byte("{}"),
	})
	is.NoErr(err)
	hex := strings.ReplaceAll(leagueID.String(), "-", "")
	league := []string{"chat.league." + hex}
	club := []string{"chat.tournament.club-1"}
	tourney := []string{"chat.tournament.tourney-1"}

	find := func(s string) []string {
		chs, err := mod.FindChannels(ms, ctx, s)
		is.NoErr(err)
		return chs
	}
	// A channel, as it is.
	is.Equal(find(" chat.lobby "), []string{"chat.lobby"})
	// A game, by its id or its page's address.
	is.Equal(find(g.GameID()), games)
	is.Equal(find("https://woogles.io/game/"+g.GameID()+"?turn=3"), games)
	// A club or tournament, by its id, its address, or the end of that.
	is.Equal(find("club-1"), club)
	is.Equal(find("woogles.io/club/Philly/"), club)
	is.Equal(find("philly"), club)
	is.Equal(find("fall-open"), tourney)
	// A league, by its id with or without dashes, its slug or its address.
	is.Equal(find(leagueID.String()), league)
	is.Equal(find(hex), league)
	is.Equal(find("/leagues/SMOKE"), league)
	// The exact match first, and what has the word in its name after it,
	// never instead.
	is.Equal(find("smoke"), append(league, "chat.tournament.club-2"))
	// Words of a name or a slug, newest first, wildcards taken literally.
	is.Equal(find("mondays"), []string{"chat.tournament.club-2", "chat.tournament.club-1"})
	is.Equal(find("league"), league)
	is.Equal(find("100%"), tourney)
	is.Equal(len(find("_")), 0)
	is.Equal(len(find("nothing like it")), 0)
	// An address that names nothing finds nothing.
	is.Equal(len(find("/club/nowhere")), 0)
}
