package game

// A move must reach the database all at once or not at all.
//
// Before this, AppendTurns wrote the event rows and Set wrote the games row in
// separate transactions, separated on the end-of-game path by two user-mutex
// acquisitions, a ratings computation and a NATS publish. A load in that window
// saw a finished event log beside a row saying the game was still being played.
// These tests hold the two halves together by breaking each one in turn and
// checking that the other did not land.
//
// Which of them actually proves the transaction, checked by removing
// WithGameLock and re-running: only TestSetRollsBackTurnsWhenGameUpdateFails
// fails (it finds 1 turn row where there should be 0). The others still pass
// without a transaction, because the turn insert runs first -- so a failure
// there stops the games update from ever being attempted, transaction or not.
// They guard the surrounding behaviour, not the atomicity.

import (
	"context"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
)

const atomGID = "wJxURccCgSAPivUvj4QdYL"

func countTurns(t *testing.T, gstore *DBStore, gid string) int {
	t.Helper()
	n, err := gstore.queries.CountGameTurns(context.Background(), gid)
	if err != nil {
		t.Fatal(err)
	}
	return int(n)
}

func playOneMove(t *testing.T, gstore *DBStore, gid, coords, word string) *entity.Game {
	t.Helper()
	entGame, err := gstore.Get(context.Background(), gid)
	if err != nil {
		t.Fatal(err)
	}
	before := len(entGame.History().Events)
	if _, err := entGame.PlayScoringMove(coords, word, true); err != nil {
		t.Fatal(err)
	}
	if err := gstore.StageTurns(entGame, before, entGame.History().Events[before:]); err != nil {
		t.Fatal(err)
	}
	return entGame
}

// The happy path: both halves land, and the staged rows are cleared so a second
// save does not write them again.
func TestSetWritesTurnsAndGameTogether(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := context.Background()

	is.Equal(countTurns(t, gstore, atomGID), 0)

	entGame := playOneMove(t, gstore, atomGID, "8E", "AGUE")
	is.Equal(len(entGame.PendingTurns()), 1)
	is.Equal(countTurns(t, gstore, atomGID), 0) // staged, not written

	is.NoErr(gstore.Set(ctx, entGame))
	is.Equal(countTurns(t, gstore, atomGID), 1)
	is.Equal(len(entGame.PendingTurns()), 0)

	// A second save must not re-write the same row; the primary key would
	// reject it, which is exactly how we would find out.
	is.NoErr(gstore.Set(ctx, entGame))
	is.Equal(countTurns(t, gstore, atomGID), 1)
}

// If the games row cannot be written, the event rows must not be either --
// otherwise the log runs ahead of the row that says where the game is.
func TestSetRollsBackTurnsWhenGameUpdateFails(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := context.Background()

	entGame := playOneMove(t, gstore, atomGID, "8E", "AGUE")

	// fk_games_player0 references users(id); this id does not exist, so
	// UpdateGame fails *after* the turn rows have been inserted in the same
	// transaction. That ordering is the point of the test.
	entGame.PlayerDBIDs[0] = 999999999

	err := gstore.Set(ctx, entGame)
	is.True(err != nil)
	is.Equal(countTurns(t, gstore, atomGID), 0)
	// Still staged: a rollback must not lose the move, it must let the next
	// save retry it.
	is.Equal(len(entGame.PendingTurns()), 1)
}

// And the other direction: if the event rows cannot be written, the games row
// must not advance past them.
func TestSetRollsBackGameWhenTurnsFail(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := context.Background()

	// Occupy turn_idx 0 so the staged insert collides with it.
	is.NoErr(gstore.queries.AppendGameTurn(ctx, models.AppendGameTurnParams{
		GameUuid: atomGID, TurnIdx: 0, Event: []byte(`{"type":"PASS"}`),
	}))

	entGame := playOneMove(t, gstore, atomGID, "8E", "AGUE")
	scoreBefore := entGame.PointsFor(0)

	err := gstore.Set(ctx, entGame)
	is.True(err != nil)
	is.Equal(countTurns(t, gstore, atomGID), 1) // only the row we planted

	// The games row still holds the position from before the move.
	reloaded, err := gstore.Get(ctx, atomGID)
	is.NoErr(err)
	is.Equal(len(reloaded.History().Events), 0)
	is.True(scoreBefore > 0) // the move did score, in memory
}

// The staged rows must be consecutive, because AppendGameTurns derives each
// index from start_idx plus the ordinal and so cannot express a gap. A gap has
// to fail loudly rather than write the wrong indices.
func TestSetRejectsNonConsecutiveStagedTurns(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := context.Background()

	entGame, err := gstore.Get(ctx, atomGID)
	is.NoErr(err)
	entGame.StageTurns([]entity.PendingTurn{
		{Idx: 0, Event: []byte(`{"type":"PASS"}`)},
		{Idx: 7, Event: []byte(`{"type":"PASS"}`)},
	})

	err = gstore.Set(ctx, entGame)
	is.True(err != nil)
	is.Equal(countTurns(t, gstore, atomGID), 0)
}
