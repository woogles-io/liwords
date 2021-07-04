package gameplay_test

import (
	"context"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	pkgmod "github.com/domino14/liwords/pkg/mod"
	pkgstats "github.com/domino14/liwords/pkg/stats"
	"github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/pkg/stores/mod"
	"github.com/domino14/liwords/pkg/stores/stats"
	ts "github.com/domino14/liwords/pkg/stores/tournament"
	"github.com/domino14/liwords/pkg/stores/user"
	"github.com/domino14/liwords/pkg/tournament"
	pkguser "github.com/domino14/liwords/pkg/user"
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
	recreateDB()
	cstr := TestingDBConnStr + " dbname=liwords_test"

	ustore := userStore(cstr)
	nstore := notorietyStore(cstr)
	lstore := listStatStore(cstr)
	cfg, gstore := gameStore(cstr, ustore)
	tstore := tournamentStore(cfg, gstore)

	g, nower, cancel, donechan, consumer := makeGame(cfg, ustore, gstore)

	return &gamesetup{
		g, nower, cancel, donechan, consumer, ustore, nstore, lstore, gstore, tstore,
	}
}

func teardownGame(g *gamesetup) {
	g.ustore.(*user.DBStore).Disconnect()
	g.nstore.(*mod.NotorietyStore).Disconnect()
	g.lstore.(*stats.ListStatStore).Disconnect()
	g.gstore.(*game.Cache).Disconnect()
	g.tstore.(*ts.Cache).Disconnect()
}
