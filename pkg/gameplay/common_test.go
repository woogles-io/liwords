package gameplay_test

import (
	"context"
	"testing"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores"
)

type gamesetup struct {
	g        *entity.Game
	nower    *entity.FakeNower
	cancel   context.CancelFunc
	donechan chan bool
	consumer *evtConsumer
	stores   *stores.Stores
}

func setupNewGame(opts ...TestGameOption) *gamesetup {
	_, stores, cfg := recreateDB()

	g, nower, cancel, donechan, consumer := makeGame(cfg, stores, opts...)

	return &gamesetup{
		g, nower, cancel, donechan, consumer, stores,
	}
}

// reload returns the game as the store now holds it.
//
// Games are not kept in memory any more, so a gamesetup's own pointer never
// sees what a store operation did to the game -- the operation loaded its own
// copy, changed that, and wrote it. Any assertion about what actually happened
// has to come from a fresh load. Where a helper returns the game it worked on,
// use that instead; this is for the ones that return nothing.
func (gs *gamesetup) reload(t *testing.T) *entity.Game {
	t.Helper()
	g, err := gs.stores.GameStore.Get(context.Background(), gs.g.GameID())
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func teardownGame(g *gamesetup) {
	g.stores.Disconnect()
}
