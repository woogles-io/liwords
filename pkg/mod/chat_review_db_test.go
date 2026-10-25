package mod_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/mod"
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
