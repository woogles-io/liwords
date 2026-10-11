package bus

import (
	"context"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Accepting a correspondence game cancels the real-time seeks of the tabs
// that get redirected to it, and nothing else (#1792).
func TestCancelRealtimeSeeksOfConns(t *testing.T) {
	is := is.New(t)
	st := recreateInternalTestDB()
	defer st.Disconnect()
	ctx := context.Background()
	b := &Bus{stores: st}

	newSeek := func(id, user, conn string, mode pb.GameMode) {
		is.NoErr(st.SoughtGameStore.New(ctx, &entity.SoughtGame{
			SeekRequest: &pb.SeekRequest{
				User:               &pb.MatchUser{UserId: user},
				SeekerConnectionId: conn,
				GameRequest:        &pb.GameRequest{Lexicon: "CSW21", GameMode: mode, RequestId: id},
			},
		}))
	}
	exists := func(id string) bool {
		_, err := st.SoughtGameStore.Get(ctx, id)
		return err == nil
	}

	// The seeker made both seeks in one tab; that tab follows them to the
	// correspondence game.
	newSeek("seeker-live", "seeker", "seeker-tab", pb.GameMode_REAL_TIME)
	newSeek("seeker-corres", "seeker", "seeker-tab", pb.GameMode_CORRESPONDENCE)
	// The accepter's live seek is up in another of the accepter's tabs,
	// which is not redirected.
	newSeek("accepter-live", "accepter", "accepter-lobby-tab", pb.GameMode_REAL_TIME)
	// A seek with no tab is never matched by an empty connection ID.
	newSeek("tabless-live", "someone", "", pb.GameMode_REAL_TIME)

	canceled, err := b.cancelRealtimeSeeksOfConns(ctx, "accepter-game-tab", "seeker-tab", "")
	is.NoErr(err)
	is.Equal(len(canceled), 1)
	id, err := canceled[0].ID()
	is.NoErr(err)
	is.Equal(id, "seeker-live")

	is.True(!exists("seeker-live"))
	is.True(exists("seeker-corres"))
	is.True(exists("accepter-live"))
	is.True(exists("tabless-live"))
}
