package gameplay_test

import (
	"context"

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

func teardownGame(g *gamesetup) {
	g.stores.Disconnect()
}
