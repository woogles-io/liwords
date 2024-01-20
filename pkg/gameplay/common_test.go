package gameplay_test

import (
	"context"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	pkgmod "github.com/woogles-io/liwords/pkg/mod"
	pkgstats "github.com/woogles-io/liwords/pkg/stats"
	"github.com/woogles-io/liwords/pkg/stores/game"
	"github.com/woogles-io/liwords/pkg/stores/mod"
	"github.com/woogles-io/liwords/pkg/stores/stats"
	ts "github.com/woogles-io/liwords/pkg/stores/tournament"
	"github.com/woogles-io/liwords/pkg/stores/user"
	"github.com/woogles-io/liwords/pkg/tournament"
	pkguser "github.com/woogles-io/liwords/pkg/user"
)

type gamesetup struct {
	g        *entity.Game
	nower    *entity.FakeNower
	cancel   context.CancelFunc
	donechan chan bool
	consumer *evtConsumer
	ustore   pkguser.Store
	nstore   pkgmod.NotorietyStore
	lstore   pkgstats.ListStatStore
	gstore   gameplay.GameStore
	tstore   tournament.TournamentStore
}

func setupNewGame() *gamesetup {
	_, ustore, lstore, nstore := recreateDB()
	cfg, gstore := gameStore(ustore)
	tstore := tournamentStore(cfg, gstore)

	g, nower, cancel, donechan, consumer := makeGame(cfg, ustore, gstore)

	return &gamesetup{
		g, nower, cancel, donechan, consumer, ustore, nstore, lstore, gstore, tstore,
	}
}

func teardownGame(g *gamesetup) {
	g.ustore.(*user.DBStore).Disconnect()
	g.nstore.(*mod.DBStore).Disconnect()
	g.lstore.(*stats.DBStore).Disconnect()
	g.gstore.(*game.Cache).Disconnect()
	g.tstore.(*ts.Cache).Disconnect()
}
