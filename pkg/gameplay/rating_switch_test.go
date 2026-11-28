package gameplay_test

import (
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/gameplay"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// With ratings paused, a rated game still ends, but nobody's rating changes.
func TestPausedRatingsSkipRating(t *testing.T) {
	is := is.New(t)
	gs := setupNewGame(func(gr *pb.GameRequest) { gr.RatingMode = pb.RatingMode_RATED })
	defer teardownGame(gs)
	ctx := ctxForTests()
	is.NoErr(gs.stores.ConfigStore.SetRatingSystemSetting(ctx, "paused"))
	gameplay.SetRatingSystemSettings(gs.stores.ConfigStore)
	defer gameplay.SetRatingSystemSettings(nil)

	const jesse = "3xpEkpRAy3AizbVmDg3kdi"
	key, err := gs.g.RatingKey()
	is.NoErr(err)
	u, err := gs.stores.UserStore.GetByUUID(ctx, jesse)
	is.NoErr(err)
	before, hadRating := u.Profile.Ratings.Data[key]

	_, err = gameplay.HandleEvent(ctx, gs.stores, jesse, &pb.ClientGameplayEvent{
		Type: pb.ClientGameplayEvent_RESIGN, GameId: gs.g.GameID(),
	})
	is.NoErr(err)
	ended, err := gs.stores.GameStore.Get(ctx, gs.g.GameID())
	is.NoErr(err)
	is.Equal(ended.GameEndReason, pb.GameEndReason_RESIGNED)
	is.Equal(len(ended.Quickdata.NewRatings), 0)

	u, err = gs.stores.UserStore.GetByUUID(ctx, jesse)
	is.NoErr(err)
	after, hasRating := u.Profile.Ratings.Data[key]
	is.Equal(hasRating, hadRating)
	is.Equal(after, before)

	gs.cancel()
	<-gs.donechan
}
