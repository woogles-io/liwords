package mod_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/gameplay"
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

	g, cancel, donechan, _ := makeGame(cfg, stores, 60, pb.RatingMode_RATED)

	// in progress: found among the active games.
	ids, err := mod.UserGameIDs(ms, ctx, cesar)
	is.NoErr(err)
	is.True(slices.Contains(ids, g.GameID()))

	// ended: found among the recent games.
	_, err = gameplay.HandleEvent(ctx, stores, cesar.UUID, &pb.ClientGameplayEvent{
		Type: pb.ClientGameplayEvent_RESIGN, GameId: g.GameID(),
	})
	is.NoErr(err)
	// a day after the game ended it is recent, a month after it is not.
	ended := time.Now()
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
