package bus

import (
	"context"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores/common"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func execOnTestDB(t *testing.T, ctx context.Context, sql string, args ...any) {
	pool, err := common.OpenTestingDB(internalTestPkg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, sql, args...); err != nil {
		t.Fatal(err)
	}
}

// The adjudicator decides from the games rows. These check that what it
// decides from them still ends the games it should, and only those.
func TestAdjudicateRealTimeFromRows(t *testing.T) {
	is := is.New(t)
	ctx, b, st, users := readyTestSetup(t)

	expired := readyTestGame(t, ctx, st, users, pb.GameMode_REAL_TIME)
	is.NoErr(gameplay.StartGame(ctx, st, b.gameEventChan, expired))
	running := readyTestGame(t, ctx, st, users, pb.GameMode_REAL_TIME)
	is.NoErr(gameplay.StartGame(ctx, st, b.gameEventChan, running))
	unstarted := readyTestGame(t, ctx, st, users, pb.GameMode_REAL_TIME)
	fresh := readyTestGame(t, ctx, st, users, pb.GameMode_REAL_TIME)

	// 25 minutes each and no overtime: a turn that began two hours ago is over.
	execOnTestDB(t, ctx, `UPDATE games SET timers = jsonb_set(timers, '{lu}',
		to_jsonb((extract(epoch from now()) * 1000)::bigint - 2 * 3600 * 1000))
		WHERE uuid = $1`, expired.GameID())
	execOnTestDB(t, ctx, `UPDATE games SET created_at = now() - interval '2 minutes'
		WHERE uuid = $1`, unstarted.GameID())

	is.NoErr(b.adjudicateGames(ctx, false))

	endReason := func(id string) pb.GameEndReason {
		g, err := st.GameStore.Get(ctx, id)
		is.NoErr(err)
		return g.GameEndReason
	}
	is.Equal(endReason(expired.GameID()), pb.GameEndReason_TIME)
	is.Equal(endReason(running.GameID()), pb.GameEndReason_NONE)
	is.Equal(endReason(unstarted.GameID()), pb.GameEndReason_CANCELLED)
	is.Equal(endReason(fresh.GameID()), pb.GameEndReason_NONE)
}
