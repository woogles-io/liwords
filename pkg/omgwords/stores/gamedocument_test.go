package stores

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	commondb "github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

var pkg = "stores"
var RedisUrl = os.Getenv("REDIS_URL")

func newPool(addr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
		Dial: func() (redis.Conn, error) { return redis.DialURL(addr) },
	}
}

func TestNewAndGet(t *testing.T) {
	is := is.New(t)
	store, err := NewGameDocumentStore(&DefaultConfig, newPool(RedisUrl), nil)
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

	otherdoc, err := store.GetDocument(ctx, gdoc.Uid, false)
	is.NoErr(err)
	is.True(proto.Equal(gdoc, otherdoc))
}

func TestRedisLocking(t *testing.T) {
	is := is.New(t)
	store, err := NewGameDocumentStore(&DefaultConfig, newPool(RedisUrl), nil)
	is.NoErr(err)
	ctx := context.Background()

	documentfile := "document-earlygame.pb"
	content, err := os.ReadFile("../../cwgame/testdata/" + documentfile)
	is.NoErr(err)
	origDoc := &ipc.GameDocument{}
	err = proto.Unmarshal(content, origDoc)
	is.NoErr(err)
	err = store.SetDocument(ctx, origDoc)
	is.NoErr(err)

	// After setting the document, let's get a bunch of threads to get it
	// and modify it.
	wg := sync.WaitGroup{}
	for i := 0; i < 25; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			// We are some player spamming pass
			doc, err := store.GetDocument(ctx, origDoc.Uid, true)
			is.NoErr(err)

			doc.Events = append(doc.Events, &ipc.GameEvent{
				Type:        ipc.GameEvent_PASS,
				PlayerIndex: 0,
			})
			err = store.UpdateDocument(ctx, doc)
			is.NoErr(err)
		}()
	}

	wg.Wait()
	doc2, err := store.GetDocument(ctx, origDoc.Uid, false)
	is.NoErr(err)

	for i := 0; i < 25; i++ {
		origDoc.Events = append(origDoc.Events, &ipc.GameEvent{
			Type:        ipc.GameEvent_PASS,
			PlayerIndex: 0,
		})
	}
	// All 25 passes get added to the document. This is because we are not
	// running logic here to check that the player is on turn before adding
	// a pass. The distributed lock worked by only allowing one thread to modify
	// the document at a time.
	is.True(proto.Equal(doc2, origDoc))
}

func TestRedisLockingWithTurnLogic(t *testing.T) {
	is := is.New(t)
	store, err := NewGameDocumentStore(&DefaultConfig, newPool(RedisUrl), nil)
	is.NoErr(err)
	ctx := context.Background()

	documentfile := "document-earlygame.pb"
	content, err := os.ReadFile("../../cwgame/testdata/" + documentfile)
	is.NoErr(err)
	origDoc := &ipc.GameDocument{}
	err = proto.Unmarshal(content, origDoc)
	is.NoErr(err)
	err = store.SetDocument(ctx, origDoc)
	is.NoErr(err)

	// After setting the document, let's get a bunch of threads to get it
	// and modify it.
	wg := sync.WaitGroup{}
	for i := 0; i < 25; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			// We are some player spamming pass
			doc, err := store.GetDocument(ctx, origDoc.Uid, true)
			is.NoErr(err)
			// Some very simple logic here to only add the pass if we're on turn.
			if doc.PlayerOnTurn == 0 {
				doc.Events = append(doc.Events, &ipc.GameEvent{
					Type:        ipc.GameEvent_PASS,
					PlayerIndex: 0,
				})
				doc.PlayerOnTurn = 1
				log.Debug().Msg("added evt")
			} else {
				log.Debug().Msg("player not on turn")
			}
			err = store.UpdateDocument(ctx, doc)
			is.NoErr(err)
		}()
	}

	wg.Wait()
	doc2, err := store.GetDocument(ctx, origDoc.Uid, false)
	is.NoErr(err)

	origDoc.Events = append(origDoc.Events, &ipc.GameEvent{
		Type:        ipc.GameEvent_PASS,
		PlayerIndex: 0,
	})
	origDoc.PlayerOnTurn = 1

	// Only 1 pass gets added to the document.
	is.True(proto.Equal(doc2, origDoc))
}

func TestDBGetAndSet(t *testing.T) {
	is := is.New(t)

	err := commondb.RecreateTestDB(pkg)
	if err != nil {
		panic(err)
	}

	dbPool, err := pgxpool.New(context.Background(), commondb.TestingPostgresConnUri(pkg))
	is.NoErr(err)
	store, err := NewGameDocumentStore(&DefaultConfig, newPool(RedisUrl), dbPool)
	is.NoErr(err)
	ctx := context.Background()

	documentfile := "document-gameover.json"
	content, err := os.ReadFile("../../cwgame/testdata/" + documentfile)
	is.NoErr(err)
	origDoc := &ipc.GameDocument{}
	err = protojson.Unmarshal(content, origDoc)
	is.NoErr(err)

	// Lock value doesn't matter. The mutex unlock will fail, but that's ok,
	// we're not testing that part.
	err = store.UpdateDocument(ctx, &MaybeLockedDocument{GameDocument: origDoc, LockValue: "foo"})
	is.NoErr(err)

	fromS3, err := store.getFromDatabase(ctx, origDoc.Uid)
	is.NoErr(err)
	is.True(proto.Equal(fromS3, origDoc))
}
