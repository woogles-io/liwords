package game

// What the cache does when a save fails. Added while splitting PR #1975; see
// docs/mikado/pr1975_split.md.

import (
	"context"
	"errors"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/user"
)

// lostAckStore commits a save and then reports failure with the game's turn
// rows still staged, as a dropped connection at commit time would leave it.
type lostAckStore struct {
	*DBStore
	fail bool
}

func (s *lostAckStore) Set(ctx context.Context, g *entity.Game) error {
	if !s.fail {
		return s.DBStore.Set(ctx, g)
	}
	staged := append([]entity.PendingTurn(nil), g.PendingTurns()...)
	if err := s.DBStore.Set(ctx, g); err != nil {
		return err
	}
	g.StageTurns(staged)
	return errors.New("connection reset at commit")
}

// A game whose save committed but reported failure must not stay cached with
// its rows staged: every later save would collide with the rows already in
// game_turns, and the game could never be moved again.
func TestCacheDropsGameWhenSaveFails(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()

	const gid = "wJxURccCgSAPivUvj4QdYL"
	ctx := context.Background()
	backing := &lostAckStore{DBStore: gstore}
	cache := NewCache(backing)

	play := func(g *entity.Game, coords, word string) {
		before := len(g.History().Events)
		_, err := g.PlayScoringMove(coords, word, true)
		is.NoErr(err)
		is.NoErr(gstore.StageTurns(g, before, g.History().Events[before:]))
	}

	g, err := cache.Get(ctx, gid)
	is.NoErr(err)
	is.True(!g.IsCorrespondence()) // the precondition: this game is cached

	backing.fail = true
	play(g, "8E", "AGUE")
	is.True(cache.Set(ctx, g) != nil)
	backing.fail = false

	// The next request must get the game as committed, not the object whose
	// rows are still staged.
	g2, err := cache.Get(ctx, gid)
	is.NoErr(err)
	is.True(g2 != g)
	is.Equal(len(g2.PendingTurns()), 0)
	is.Equal(len(g2.History().Events), len(g.History().Events)) // the move did commit

	// And it can still be played.
	play(g2, "E8", "AVE")
	is.NoErr(cache.Set(ctx, g2))
	turns, err := gstore.GetTurns(ctx, gid)
	is.NoErr(err)
	is.Equal(len(turns), len(g2.History().Events))
}
