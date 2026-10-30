package mod_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestUserGameIDs(t *testing.T) {
	is := is.New(t)
	_, stores, cfg := recreateDB()
	defer stores.Disconnect()
	ctx := DefaultConfig.WithContext(context.Background())
	ms := mod.NewModService(stores.UserStore, stores.ChatStore, stores.Queries)

	cesar, err := stores.UserStore.Get(ctx, "cesar")
	is.NoErr(err)

	g, _, cancel, donechan, _ := makeGame(cfg, stores, 60, pb.RatingMode_RATED)

	// in progress: found among the active games.
	ids, err := mod.UserGameIDs(ms, ctx, cesar)
	is.NoErr(err)
	is.True(slices.Contains(ids, g.GameID()))

	// ended: found among the recent games.
	turns := []*pb.ClientGameplayEvent{
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "8D",
			MachineLetters: englishBytes("BANJO"),
		},
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "7H",
			MachineLetters: englishBytes("BUSUUTI"),
		},
	}
	err = playGame(ctx, g, stores, turns, 0, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	// the fake clock ended the game at 1970: a day later it is recent, a
	// month later it is not.
	ended := time.UnixMilli(1234)
	restore := mod.SetChatReviewNow(func() time.Time { return ended.Add(24 * time.Hour) })
	defer restore()
	ids, err = mod.UserGameIDs(ms, ctx, cesar)
	is.NoErr(err)
	is.Equal(ids, []string{g.GameID()})

	mod.SetChatReviewNow(func() time.Time { return ended.Add(30 * 24 * time.Hour) })
	ids, err = mod.UserGameIDs(ms, ctx, cesar)
	is.NoErr(err)
	is.Equal(len(ids), 0)

	cancel()
	<-donechan
}

func TestChannelContext(t *testing.T) {
	is := is.New(t)
	_, stores, cfg := recreateDB()
	defer stores.Disconnect()
	ctx := DefaultConfig.WithContext(context.Background())
	ms := mod.NewModService(stores.UserStore, stores.ChatStore, stores.Queries)

	g, _, cancel, donechan, _ := makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	defer func() {
		cancel()
		<-donechan
	}()

	c := mod.ChannelContext(ms, ctx, "chat.game."+g.GameID())
	is.Equal(c.Link, "/game/"+g.GameID())
	is.Equal(c.Label, "cesar vs jesse")
	is.Equal(len(c.Parties), 2)
	is.True(c.GameCreatedAt > 0)
	is.True(c.GameUpdatedAt != 0) // the fake clock dates it 1970

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

	g, _, cancel, donechan, _ := makeGame(cfg, stores, 60, pb.RatingMode_RATED)
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
