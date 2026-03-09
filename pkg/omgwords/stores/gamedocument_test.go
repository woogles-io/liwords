package stores

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	commondb "github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

var pkg = "stores"

func TestNewAndGet(t *testing.T) {
	is := is.New(t)

	err := commondb.RecreateTestDB(pkg)
	if err != nil {
		panic(err)
	}

	dbPool, err := pgxpool.New(context.Background(), commondb.TestingPostgresConnUri(pkg))
	is.NoErr(err)
	defer dbPool.Close()
	store, err := NewGameDocumentStore(DefaultConfig, dbPool)
	is.NoErr(err)
	ctx := context.Background()

	documentfile := "document-earlygame.pb"
	content, err := os.ReadFile("../../cwgame/testdata/" + documentfile)
	is.NoErr(err)
	gdoc := &ipc.GameDocument{}
	err = proto.Unmarshal(content, gdoc)
	is.NoErr(err)
	err = store.SetDocument(ctx, gdoc)
	is.NoErr(err)

	otherdoc, err := store.GetDocument(ctx, gdoc.Uid)
	is.NoErr(err)
	is.True(proto.Equal(gdoc, otherdoc))
}

func TestDBGetAndSet(t *testing.T) {
	is := is.New(t)

	err := commondb.RecreateTestDB(pkg)
	if err != nil {
		panic(err)
	}

	dbPool, err := pgxpool.New(context.Background(), commondb.TestingPostgresConnUri(pkg))
	is.NoErr(err)
	defer dbPool.Close()
	store, err := NewGameDocumentStore(DefaultConfig, dbPool)
	is.NoErr(err)
	ctx := context.Background()

	documentfile := "document-gameover.json"
	content, err := os.ReadFile("../../cwgame/testdata/" + documentfile)
	is.NoErr(err)
	origDoc := &ipc.GameDocument{}
	err = protojson.Unmarshal(content, origDoc)
	is.NoErr(err)

	err = store.UpdateDocument(ctx, origDoc)
	is.NoErr(err)

	fromDB, err := store.getFromDatabase(ctx, origDoc.Uid)
	is.NoErr(err)
	is.True(proto.Equal(fromDB, origDoc))
}
