package gameplay_test

import (
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/gameplay"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	jesseID = "3xpEkpRAy3AizbVmDg3kdi" // on turn first
	cesarID = "xjCWug7EZtDxDHX5fRZTLo"
)

// Only the play immediately before a challenge can be challenged. The server
// has to enforce that itself: macondo rebuilds a loaded game through a path
// that never clears the last play's words on a pass or an exchange, and every
// event now runs against a game loaded from the database. Without the check a
// player could pass, let the opponent pass, and then challenge the opponent's
// play from two turns ago.
func TestChallengeOnlyTheLastPlay(t *testing.T) {
	type move struct {
		userID string
		cge    *pb.ClientGameplayEvent
	}
	place := func(userID, coords string, letters ...byte) move {
		return move{userID, &pb.ClientGameplayEvent{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: coords,
			MachineLetters: letters,
		}}
	}
	pass := func(userID string) move {
		return move{userID, &pb.ClientGameplayEvent{Type: pb.ClientGameplayEvent_PASS}}
	}
	exchange := func(userID string, letters ...byte) move {
		return move{userID, &pb.ClientGameplayEvent{
			Type:           pb.ClientGameplayEvent_EXCHANGE,
			MachineLetters: letters,
		}}
	}
	challenge := func(userID string) move {
		return move{userID, &pb.ClientGameplayEvent{Type: pb.ClientGameplayEvent_CHALLENGE_PLAY}}
	}
	banjoer := place(jesseID, "8D", 2, 1, 14, 10, 15, 5, 18) // a phony

	for _, tc := range []struct {
		name  string
		moves []move
	}{
		{"opponent's play, after both passed", []move{banjoer, pass(cesarID), pass(jesseID)}},
		{"own play, after the opponent passed", []move{banjoer, pass(cesarID)}},
		{"opponent's play, after an exchange and a pass",
			[]move{banjoer, exchange(cesarID, 25, 25, 26), pass(jesseID)}},
		{"the same play twice", []move{banjoer, challenge(cesarID)}},
		{"before any play", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			ctx := ctxForTests()
			gs := setupNewGame()
			defer teardownGame(gs)

			gs.g.SetChallengeRule(macondopb.ChallengeRule_DOUBLE)
			gs.g.SetRacksForBoth([]*tilemapping.Rack{
				tilemapping.RackFromString("ABEJNOR", gs.g.Alphabet()),
				tilemapping.RackFromString("AGLSYYZ", gs.g.Alphabet()),
			})
			is.NoErr(gs.stores.GameStore.Set(ctx, gs.g))

			for _, m := range tc.moves {
				m.cge.GameId = gs.g.GameID()
				gs.nower.Sleep(1000)
				_, err := gameplay.HandleEvent(ctx, gs.stores, m.userID, m.cge)
				is.NoErr(err)
			}
			before := gs.reload(t)
			onTurn := before.Game.PlayerIDOnTurn()
			scores := []int{before.Game.PointsFor(0), before.Game.PointsFor(1)}

			_, err := gameplay.HandleEvent(ctx, gs.stores, onTurn, &pb.ClientGameplayEvent{
				Type:   pb.ClientGameplayEvent_CHALLENGE_PLAY,
				GameId: gs.g.GameID(),
			})
			is.True(err != nil)
			is.Equal(err.Error(), "only the last play can be challenged")

			after := gs.reload(t)
			is.Equal(len(after.History().Events), len(before.History().Events))
			is.Equal(after.Game.PlayerIDOnTurn(), onTurn)
			is.Equal([]int{after.Game.PointsFor(0), after.Game.PointsFor(1)}, scores)

			gs.cancel()
			<-gs.donechan
		})
	}
}
