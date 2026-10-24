package gameplay_test

import (
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/gameplay"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

// The end of a game runs automod, and only for rated games. pkg/mod tests what
// automod decides; this tests that it is reached, through the real end of a
// game, without relying on any one in-memory copy of it.
//
// jesse resigns a 25-minute game without having made a single play, which
// automod treats as abandoning it.
func resignWithoutPlaying(t *testing.T, mode pb.RatingMode) (notoriety int, filed []*ms.NotoriousGame) {
	is := is.New(t)
	gs := setupNewGame(func(gr *pb.GameRequest) { gr.RatingMode = mode })
	defer teardownGame(gs)
	ctx := ctxForTests()

	const jesse = "3xpEkpRAy3AizbVmDg3kdi"
	_, err := gameplay.HandleEvent(ctx, gs.stores, jesse, &pb.ClientGameplayEvent{
		Type: pb.ClientGameplayEvent_RESIGN, GameId: gs.g.GameID(),
	})
	is.NoErr(err)
	ended, err := gs.stores.GameStore.Get(ctx, gs.g.GameID())
	is.NoErr(err)
	is.Equal(ended.GameEndReason, pb.GameEndReason_RESIGNED)

	u, err := gs.stores.UserStore.GetByUUID(ctx, jesse)
	is.NoErr(err)
	filed, err = gs.stores.NotorietyStore.GetNotoriousGames(ctx, jesse, 10)
	is.NoErr(err)

	gs.cancel()
	<-gs.donechan
	return u.Notoriety, filed
}

func TestRatedGameEndRunsAutomod(t *testing.T) {
	is := is.New(t)
	notoriety, filed := resignWithoutPlaying(t, pb.RatingMode_RATED)
	is.True(notoriety > 0)
	is.Equal(len(filed), 1)
	is.Equal(filed[0].Type, ms.NotoriousGameType_NO_PLAY)
}

func TestCasualGameEndSkipsAutomod(t *testing.T) {
	is := is.New(t)
	notoriety, filed := resignWithoutPlaying(t, pb.RatingMode_CASUAL)
	is.Equal(notoriety, 0)
	is.Equal(len(filed), 0)
}
