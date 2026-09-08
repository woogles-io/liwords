package mod_test

// Applying a verdict, and applying it only once.
//
// Notoriety is not idempotent: a bad game adds to a player's score and files a
// row against them, a good one subtracts. Nothing in the database used to stop
// that happening twice for one game; what stopped it was every caller of
// performEndgameDuties checking whether the game had already ended before
// getting that far. performEndgameDuties itself has no such check, so the
// protection was a property of who called it. These tests hold it as a property
// of the operation instead, which is why they run against the real store -- the
// guarantee is a unique constraint, and a fake store would prove nothing.

import (
	"context"
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/stores"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

// abandonedGame is the simplest notorious game: player 1 never moved and lost
// on time. Built directly rather than played, because Automod reads a finished
// game and nothing else.
func abandonedGame(uid string) *entity.Game {
	g := &entity.Game{}
	g.SetHistory(&macondopb.GameHistory{
		Uid: uid,
		Players: []*macondopb.PlayerInfo{
			{UserId: playerIds[0], Nickname: "cesar"},
			{UserId: playerIds[1], Nickname: "jesse"},
		},
		Events: []*macondopb.GameEvent{{
			PlayerIndex:     0,
			Type:            macondopb.GameEvent_TILE_PLACEMENT_MOVE,
			MillisRemaining: 1000,
		}},
	})
	g.GameEndReason = pb.GameEndReason_TIME
	g.LoserIdx = 1
	g.WinnerIdx = 0
	g.GameReq = &entity.GameRequest{GameRequest: &pb.GameRequest{
		InitialTimeSeconds: 25 * 60,
	}}
	g.Timers = entity.Timers{TimeRemaining: []int{0, 0}}
	g.MetaEvents = &entity.MetaEventData{}
	return g
}

func notorietyOf(t *testing.T, st *stores.Stores, uuid string) int {
	t.Helper()
	score, _, err := mod.GetNotorietyReport(context.Background(), st.UserStore, st.NotorietyStore, uuid, 10)
	if err != nil {
		t.Fatal(err)
	}
	return score
}

func notoriousGameCount(t *testing.T, st *stores.Stores, uuid string) int {
	t.Helper()
	_, games, err := mod.GetNotorietyReport(context.Background(), st.UserStore, st.NotorietyStore, uuid, 100)
	if err != nil {
		t.Fatal(err)
	}
	return len(games)
}

func TestAutomodAppliesAVerdictOnce(t *testing.T) {
	is := is.New(t)
	_, st, cfg := recreateDB()
	defer st.Disconnect()
	ctx := cfg.WithContext(context.Background())

	u0, err := st.UserStore.GetByUUID(ctx, playerIds[0])
	is.NoErr(err)
	u1, err := st.UserStore.GetByUUID(ctx, playerIds[1])
	is.NoErr(err)

	g := abandonedGame("abandoned-1")
	is.NoErr(mod.Automod(ctx, st.UserStore, st.NotorietyStore, u0, u1, g))

	// NO_PLAY is worth 6.
	is.Equal(notorietyOf(t, st, playerIds[1]), mod.BehaviorToScore[ms.NotoriousGameType_NO_PLAY])
	is.Equal(notoriousGameCount(t, st, playerIds[1]), 1)

	// Running it again for the same game must change nothing -- not the score,
	// and not the list of games held against them.
	u1, err = st.UserStore.GetByUUID(ctx, playerIds[1])
	is.NoErr(err)
	is.NoErr(mod.Automod(ctx, st.UserStore, st.NotorietyStore, u0, u1, g))
	is.Equal(notorietyOf(t, st, playerIds[1]), mod.BehaviorToScore[ms.NotoriousGameType_NO_PLAY])
	is.Equal(notoriousGameCount(t, st, playerIds[1]), 1)

	// A different game does count.
	u1, err = st.UserStore.GetByUUID(ctx, playerIds[1])
	is.NoErr(err)
	is.NoErr(mod.Automod(ctx, st.UserStore, st.NotorietyStore, u0, u1, abandonedGame("abandoned-2")))
	is.Equal(notorietyOf(t, st, playerIds[1]), 2*mod.BehaviorToScore[ms.NotoriousGameType_NO_PLAY])
	is.Equal(notoriousGameCount(t, st, playerIds[1]), 2)
}

// The forgiving path needs the same protection. A good game decrements, so
// replaying one would forgive twice -- which is why GOOD verdicts are recorded
// too rather than only the notorious ones.
func TestAutomodForgivesOnlyOnce(t *testing.T) {
	is := is.New(t)
	_, st, cfg := recreateDB()
	defer st.Disconnect()
	ctx := cfg.WithContext(context.Background())

	u0, err := st.UserStore.GetByUUID(ctx, playerIds[0])
	is.NoErr(err)
	u1, err := st.UserStore.GetByUUID(ctx, playerIds[1])
	is.NoErr(err)

	// Two bad games to build up a score worth decrementing.
	for _, uid := range []string{"bad-1", "bad-2"} {
		u1, err = st.UserStore.GetByUUID(ctx, playerIds[1])
		is.NoErr(err)
		is.NoErr(mod.Automod(ctx, st.UserStore, st.NotorietyStore, u0, u1, abandonedGame(uid)))
	}
	built := notorietyOf(t, st, playerIds[1])
	is.True(built > 1)

	// A clean game: both played it out, nobody ran the clock down.
	good := abandonedGame("clean-1")
	good.History().Events = append(good.History().Events, &macondopb.GameEvent{
		PlayerIndex:     1,
		Type:            macondopb.GameEvent_TILE_PLACEMENT_MOVE,
		MillisRemaining: 1000,
	})

	u1, err = st.UserStore.GetByUUID(ctx, playerIds[1])
	is.NoErr(err)
	is.NoErr(mod.Automod(ctx, st.UserStore, st.NotorietyStore, u0, u1, good))
	once := notorietyOf(t, st, playerIds[1])
	is.Equal(once, built-mod.NotorietyDecrement)

	u1, err = st.UserStore.GetByUUID(ctx, playerIds[1])
	is.NoErr(err)
	is.NoErr(mod.Automod(ctx, st.UserStore, st.NotorietyStore, u0, u1, good))
	is.Equal(notorietyOf(t, st, playerIds[1]), once)
}
