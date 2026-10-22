package gameplay_test

import (
	"testing"
	"time"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/macondo/move"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/gameplay"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func withCorrespondence(gr *pb.GameRequest) {
	gr.GameMode = pb.GameMode_CORRESPONDENCE
}

// Ending a game must load it inside the game's lock. A correspondence game is
// a fresh copy on every load, so a copy loaded before the lock is out of date
// as soon as someone else moves, and ending the game from it saves over the
// move.
func TestAdjudicateLoadsTheGameInsideTheLock(t *testing.T) {
	is := is.New(t)
	gs := setupNewGame(withCorrespondence)
	defer teardownGame(gs)
	ctx := ctxForTests()
	gid := gs.g.GameID()

	// Hold the lock, as a move in progress would.
	gameLock, err := gs.stores.GameStore.LockGame(ctx, gid)
	is.NoErr(err)

	done := make(chan error, 1)
	go func() { done <- gameplay.AdjudicateGame(ctx, gid, gs.stores) }()

	select {
	case err := <-done:
		t.Fatalf("adjudicated while the game was locked: %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	// The move in progress: a pass, played and saved under the lock.
	mover, err := gs.stores.GameStore.Get(ctx, gid)
	is.NoErr(err)
	mover.Lock()
	onTurn := mover.Game.PlayerOnTurn()
	pass := move.NewPassMove(mover.Game.RackFor(onTurn).TilesOn(), mover.Game.Alphabet())
	err = gameplay.PlayMove(ctx, mover, gs.stores, mover.Game.PlayerIDOnTurn(), onTurn,
		mover.TimeRemaining(onTurn), pass)
	mover.Unlock()
	is.NoErr(err)
	gameLock.Unlock(ctx)

	is.NoErr(<-done)

	final, err := gs.stores.GameStore.Get(ctx, gid)
	is.NoErr(err)
	is.Equal(final.GameEndReason, pb.GameEndReason_ADJUDICATED)
	evts := final.History().Events
	is.True(len(evts) > 0)
	is.Equal(evts[0].Type, macondopb.GameEvent_PASS) // the move was not saved over

	// Ending it again does nothing.
	is.NoErr(gameplay.AdjudicateGame(ctx, gid, gs.stores))
	is.NoErr(gameplay.ForfeitGame(ctx, gid, 0, gs.stores))
	again, err := gs.stores.GameStore.Get(ctx, gid)
	is.NoErr(err)
	is.Equal(again.GameEndReason, pb.GameEndReason_ADJUDICATED)
	is.Equal(len(again.History().Events), len(evts))

	gs.cancel()
	<-gs.donechan
}
