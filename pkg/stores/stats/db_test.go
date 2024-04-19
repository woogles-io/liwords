package stats

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
	commondb "github.com/woogles-io/liwords/pkg/stores/common"
)

func TestListStats(t *testing.T) {
	err := commondb.RecreateTestDB()
	if err != nil {
		panic(err)
	}

	pool, err := commondb.OpenTestingDB()
	if err != nil {
		panic(err)
	}

	store, err := NewDBStore(pool)
	if err != nil {
		panic(err)
	}

	is := is.New(t)
	ctx := context.Background()

	err = store.AddListItem(ctx, "game1", "player1", 0, 7, newListDatum("A", 2))
	is.NoErr(err)
	err = store.AddListItem(ctx, "game1", "player2", 0, 6, newListDatum("B", 2))
	is.NoErr(err)
	err = store.AddListItem(ctx, "game1", "player1", 1, 2, newListDatum("C", 2))
	is.NoErr(err)
	err = store.AddListItem(ctx, "game2", "player1", 2, 1, newListDatum("D", 2))
	is.NoErr(err)
	err = store.AddListItem(ctx, "game2", "player2", 2, 4, newListDatum("E", 2))
	is.NoErr(err)
	err = store.AddListItem(ctx, "game3", "player1", 2, 3, newListDatum("F", 2))
	is.NoErr(err)
	err = store.AddListItem(ctx, "game4", "player2", 0, 5, newListDatum("G", 2))
	is.NoErr(err)

	stats, err := store.GetListItems(ctx, 0, []string{"game1"}, "")
	is.NoErr(err)
	is.Equal(stats, []*entity.ListItem{
		{GameId: "game1", PlayerId: "player2", Time: 6, Item: entity.ListDatum{Word: "B", Score: 2}},
		{GameId: "game1", PlayerId: "player1", Time: 7, Item: entity.ListDatum{Word: "A", Score: 2}},
	})

	stats, err = store.GetListItems(ctx, 1, []string{"game1"}, "")
	is.NoErr(err)
	is.Equal(len(stats), 1)

	stats, err = store.GetListItems(ctx, 0, []string{"game1", "game4"}, "")
	is.NoErr(err)
	is.Equal(len(stats), 3)

	stats, err = store.GetListItems(ctx, 0, []string{"game1", "game4"}, "player2")
	is.NoErr(err)
	is.Equal(len(stats), 2)

	setNullValues(ctx, pool, []interface{}{"game2", "game3"})

	stats, err = store.GetListItems(ctx, 2, []string{"game2", "game3"}, "")
	is.NoErr(err)
	is.Equal(len(stats), 3)

	store.Disconnect()
}

func setNullValues(ctx context.Context, pool *pgxpool.Pool, gameIds []interface{}) {
	tx, err := pool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)

	inClause := common.BuildIn(len(gameIds), 1)

	updateStmt := fmt.Sprintf("UPDATE liststats SET player_id = NULL, timestamp = NULL, item = NULL WHERE game_id IN (%s)", inClause)
	_, err = tx.Exec(ctx, updateStmt, gameIds...)
	if err != nil {
		panic(err)
	}

	if err := tx.Commit(ctx); err != nil {
		panic(err)
	}
}

func newListDatum(word string, score int) entity.ListDatum {
	return entity.ListDatum{
		Word:  word,
		Score: score,
	}
}
