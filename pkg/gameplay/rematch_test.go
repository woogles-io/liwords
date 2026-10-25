package gameplay_test

import (
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func rematchStarted(evts []*entity.EventWrapper) []*entity.EventWrapper {
	var out []*entity.EventWrapper
	for _, e := range evts {
		if e.Type == pb.MessageType_REMATCH_STARTED {
			out = append(out, e)
		}
	}
	return out
}

// Starting a rematch tells the viewers of the most recent finished game in the
// streak; the first game of a streak tells no one.
func TestRematchStartTellsPreviousGame(t *testing.T) {
	is := is.New(t)
	_, st, cfg := recreateDB()
	defer st.Disconnect()
	ctx := ctxForTests()
	g, _, cancel, donechan, consumer := makeGame(cfg, st)
	first := g.GameID()

	// Nothing in the streak has finished yet.
	prev, err := st.GameStore.GetLatestRematchGame(ctx, gameReq.OriginalRequestId)
	is.NoErr(err)
	is.Equal(prev, "")

	_, err = gameplay.HandleEvent(ctx, st, "3xpEkpRAy3AizbVmDg3kdi", &pb.ClientGameplayEvent{
		Type: pb.ClientGameplayEvent_RESIGN, GameId: first,
	})
	is.NoErr(err)
	cancel()
	<-donechan
	is.Equal(len(rematchStarted(consumer.evts)), 0)

	prev, err = st.GameStore.GetLatestRematchGame(ctx, gameReq.OriginalRequestId)
	is.NoErr(err)
	is.Equal(prev, first)

	// The rematch shares the first game's original request id.
	second, _, cancel, donechan, consumer := makeGame(cfg, st)
	// Each game's events need a reader until the game is done with them.
	_, err = gameplay.HandleEvent(ctx, st, "3xpEkpRAy3AizbVmDg3kdi", &pb.ClientGameplayEvent{
		Type: pb.ClientGameplayEvent_RESIGN, GameId: second.GameID(),
	})
	is.NoErr(err)
	cancel()
	<-donechan
	started := rematchStarted(consumer.evts)
	is.Equal(len(started), 1)
	is.Equal(started[0].Audience(), []string{string(entity.AudGameTV) + "." + first})
	is.Equal(started[0].Event.(*pb.RematchStartedEvent).RematchGameId, second.GameID())

	// The rematch has finished too, so the next game points at it, not at the
	// first.
	third, _, cancel, donechan, consumer := makeGame(cfg, st)
	cancel()
	<-donechan
	started = rematchStarted(consumer.evts)
	is.Equal(len(started), 1)
	is.Equal(started[0].Audience(), []string{string(entity.AudGameTV) + "." + second.GameID()})
	is.Equal(started[0].Event.(*pb.RematchStartedEvent).RematchGameId, third.GameID())

	// A request id with no finished games has no previous game.
	prev, err = st.GameStore.GetLatestRematchGame(ctx, "no-such-request")
	is.NoErr(err)
	is.Equal(prev, "")
}
