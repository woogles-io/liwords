package tournament_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/stores/common"
)

func TestTourneyStats(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	stores, _ := recreateDB()
	defer func() { cleanup(stores) }()

	pool, err := common.OpenTestingDB(pkg)
	is.NoErr(err)

	conn, err := pool.BeginTx(ctx, pgx.TxOptions{})
	is.NoErr(err)

	tsql, err := os.ReadFile("./testdata/mondaily55.sql")
	is.NoErr(err)
	_, err = conn.Exec(ctx, string(tsql))
	is.NoErr(err)

	usql, err := os.ReadFile("./testdata/mondaily55_users.sql")
	is.NoErr(err)
	_, err = conn.Exec(ctx, string(usql))
	is.NoErr(err)

	gsql, err := os.ReadFile("./testdata/mondaily55_games.sql")
	is.NoErr(err)
	_, err = conn.Exec(ctx, string(gsql))
	is.NoErr(err)

	err = conn.Commit(ctx)
	is.NoErr(err)

	ct, err := stores.GameStore.Count(ctx)
	is.NoErr(err)
	is.Equal(ct, int64(70))

	ty, err := stores.TournamentStore.Get(ctx, "8eeBDqgrVJgM8ryPUVQuh3")
	is.NoErr(err)

	tn := time.Now()

	err = ty.Divisions["NWL"].DivisionManager.CalculateStats(ctx, stores.Queries)
	is.NoErr(err)
	fmt.Println("calculate stats in ", time.Since(tn).String())
	fmt.Println(ty.Divisions["NWL"].DivisionManager.Stats())
	is.True(false)
}
