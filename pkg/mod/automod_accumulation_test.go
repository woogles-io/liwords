package mod_test

// How verdicts accumulate into a notoriety score, and what happens when it gets
// too high.
//
// This is the half of automod that has state. What a given game *means* is
// decided by Classify and covered exhaustively in classify_test.go against
// games built directly; here every game is a fixture with a verdict that
// classify_test has already pinned down, so a failure means the accumulation
// changed rather than that a simulated game drifted.
//
// The test this replaces played whole games through the store to reach each
// verdict, which meant it depended on one *entity.Game being shared between the
// test and every store call. It also forced states no real game reaches -- most
// visibly SetPlayerOnTurn(loserIdx), so a chosen player could be timed out when
// it was not their turn -- and asserted on the result.
//
// One thing it covered incidentally and this does not: which games reach
// automod at all. That gate is in pkg/gameplay/end.go:253 -- tournament games
// go to HandleTournamentGameEnded instead, and only RATED games are judged, so
// a casual game changes nobody's notoriety. It belongs to the end-of-game path
// rather than to automod.

import (
	"context"
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	pkgmod "github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/stores"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

// judgedGame builds a finished game that Classify will judge as `want` for the
// player at loserIdx, and checks that it does. Building the fixture and
// asserting its verdict together is what stops the two drifting apart.
func judgedGame(t *testing.T, uid string, loserIdx int, want ms.NotoriousGameType) *entity.Game {
	t.Helper()

	const (
		plenty = int32(20 * 60 * 1000) // walked away
		barely = int32(1000)           // played it to the end
	)

	g := &entity.Game{}
	winnerIdx := 1 - loserIdx

	// The winner always plays a move, so they are never the one being judged.
	events := []*macondopb.GameEvent{{
		PlayerIndex:     uint32(winnerIdx),
		Type:            macondopb.GameEvent_TILE_PLACEMENT_MOVE,
		MillisRemaining: barely,
	}}
	switch want {
	case ms.NotoriousGameType_NO_PLAY:
		// The loser never moved at all.
	case ms.NotoriousGameType_SITTING:
		events = append(events, &macondopb.GameEvent{
			PlayerIndex:     uint32(loserIdx),
			Type:            macondopb.GameEvent_TILE_PLACEMENT_MOVE,
			MillisRemaining: plenty,
		})
	case ms.NotoriousGameType_GOOD:
		events = append(events, &macondopb.GameEvent{
			PlayerIndex:     uint32(loserIdx),
			Type:            macondopb.GameEvent_TILE_PLACEMENT_MOVE,
			MillisRemaining: barely,
		})
	default:
		t.Fatalf("judgedGame does not know how to produce %s", want)
	}

	g.SetHistory(&macondopb.GameHistory{
		Uid: uid,
		Players: []*macondopb.PlayerInfo{
			{UserId: playerIds[0], Nickname: "cesar"},
			{UserId: playerIds[1], Nickname: "jesse"},
		},
		Events: events,
	})
	g.GameEndReason = pb.GameEndReason_TIME
	g.LoserIdx = loserIdx
	g.WinnerIdx = winnerIdx
	g.GameReq = &entity.GameRequest{GameRequest: &pb.GameRequest{
		InitialTimeSeconds: 25 * 60,
	}}
	g.Timers = entity.Timers{TimeRemaining: []int{0, 0}}
	g.MetaEvents = &entity.MetaEventData{}

	v, err := pkgmod.Classify(g, pkgmod.DefaultConfig.WGLConfig(), false)
	if err != nil {
		t.Fatal(err)
	}
	if v.Loser != want {
		t.Fatalf("fixture %q was meant to be %s but classifies as %s", uid, want, v.Loser)
	}
	return g
}

// judge runs automod over a fixture, reloading both users first because their
// notoriety changes underneath.
func judge(t *testing.T, st *stores.Stores, ctx context.Context, g *entity.Game) {
	t.Helper()
	u0, err := st.UserStore.GetByUUID(ctx, playerIds[0])
	if err != nil {
		t.Fatal(err)
	}
	u1, err := st.UserStore.GetByUUID(ctx, playerIds[1])
	if err != nil {
		t.Fatal(err)
	}
	if err := pkgmod.Automod(ctx, st.UserStore, st.NotorietyStore, u0, u1, g); err != nil {
		t.Fatal(err)
	}
}

func TestNotorietyAccumulates(t *testing.T) {
	is := is.New(t)
	_, st, _ := recreateDB()
	defer st.Disconnect()
	ctx := pkgmod.DefaultConfig.WithContext(context.Background())

	noPlay := pkgmod.BehaviorToScore[ms.NotoriousGameType_NO_PLAY]  // 6
	sitting := pkgmod.BehaviorToScore[ms.NotoriousGameType_SITTING] // 4

	// Jesse abandons a game.
	judge(t, st, ctx, judgedGame(t, "g1", 1, ms.NotoriousGameType_NO_PLAY))
	is.NoErr(comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: int32(noPlay), Games: []*ms.NotoriousGame{{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, st))

	// Two games played out properly bring it down, one point each. The list of
	// games held against them does not shrink -- the score decays, the record
	// stays.
	judge(t, st, ctx, judgedGame(t, "g2", 1, ms.NotoriousGameType_GOOD))
	judge(t, st, ctx, judgedGame(t, "g3", 1, ms.NotoriousGameType_GOOD))
	is.NoErr(comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: int32(noPlay - 2*pkgmod.NotorietyDecrement),
			Games: []*ms.NotoriousGame{{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, st))

	// Sitting on the clock adds again, and the newest game is listed first.
	judge(t, st, ctx, judgedGame(t, "g4", 1, ms.NotoriousGameType_SITTING))
	is.NoErr(comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: int32(noPlay - 2*pkgmod.NotorietyDecrement + sitting), Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, st))

	// A good player's score never goes below zero however well they behave.
	for i, uid := range []string{"g5", "g6", "g7"} {
		_ = i
		judge(t, st, ctx, judgedGame(t, uid, 1, ms.NotoriousGameType_GOOD))
	}
	score, _, err := pkgmod.GetNotorietyReport(ctx, st.UserStore, st.NotorietyStore, playerIds[0], 100)
	is.NoErr(err)
	is.Equal(score, 0)
}

// Crossing the threshold suspends the player from rated games, for longer the
// further over they are.
func TestNotorietyThresholdSuspends(t *testing.T) {
	is := is.New(t)
	_, st, _ := recreateDB()
	defer st.Disconnect()
	ctx := pkgmod.DefaultConfig.WithContext(context.Background())

	noPlay := pkgmod.BehaviorToScore[ms.NotoriousGameType_NO_PLAY]

	// Below the threshold, nothing happens.
	judge(t, st, ctx, judgedGame(t, "s1", 1, ms.NotoriousGameType_NO_PLAY))
	_, err := pkgmod.ActionExists(ctx, st.UserStore, playerIds[1], false,
		[]ms.ModActionType{ms.ModActionType_SUSPEND_RATED_GAMES})
	is.NoErr(err)

	// 12 is over the threshold of 10, so a suspension of 24h per point over.
	judge(t, st, ctx, judgedGame(t, "s2", 1, ms.NotoriousGameType_NO_PLAY))
	score, _, err := pkgmod.GetNotorietyReport(ctx, st.UserStore, st.NotorietyStore, playerIds[1], 100)
	is.NoErr(err)
	is.Equal(score, 2*noPlay)

	_, err = pkgmod.ActionExists(ctx, st.UserStore, playerIds[1], false,
		[]ms.ModActionType{ms.ModActionType_SUSPEND_RATED_GAMES})
	is.True(err != nil) // an action now exists, so this reports one

	// A further offence supersedes it with a longer one, and the first drops
	// into the history.
	judge(t, st, ctx, judgedGame(t, "s3", 1, ms.NotoriousGameType_NO_PLAY))
	history, err := pkgmod.GetActionHistory(ctx, st.UserStore, playerIds[1])
	is.NoErr(err)
	is.NoErr(equalActionHistories(history, []*ms.ModAction{{
		UserId:   playerIds[1],
		Type:     ms.ModActionType_SUSPEND_RATED_GAMES,
		Duration: int32(pkgmod.DurationMultiplier * (2*noPlay - pkgmod.NotorietyThreshold)),
	}}))
}

// Both players are judged, independently, in the same game.
func TestNotorietyTracksBothPlayers(t *testing.T) {
	is := is.New(t)
	_, st, _ := recreateDB()
	defer st.Disconnect()
	ctx := pkgmod.DefaultConfig.WithContext(context.Background())

	noPlay := pkgmod.BehaviorToScore[ms.NotoriousGameType_NO_PLAY]

	judge(t, st, ctx, judgedGame(t, "b1", 1, ms.NotoriousGameType_NO_PLAY))
	judge(t, st, ctx, judgedGame(t, "b2", 0, ms.NotoriousGameType_NO_PLAY))

	// Each was the loser once and the winner once, so each carries one
	// abandonment and one point of forgiveness for the game they played out.
	is.NoErr(comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: int32(noPlay), Games: []*ms.NotoriousGame{{Type: ms.NotoriousGameType_NO_PLAY}}},
		{Score: int32(noPlay - pkgmod.NotorietyDecrement),
			Games: []*ms.NotoriousGame{{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, st))
}
