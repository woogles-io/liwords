package session

import (
	"context"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
)

var pkg = "session"

func TestSession(t *testing.T) {
	err := common.RecreateTestDB(pkg)
	if err != nil {
		panic(err)
	}
	pool, err := common.OpenTestingDB(pkg)
	if err != nil {
		panic(err)
	}

	store, err := NewDBStore(pool)
	if err != nil {
		panic(err)
	}

	is := is.New(t)
	ctx := context.Background()

	cesarUser := &entity.User{Username: "cesar", UUID: "cesar_uuid"}

	cesarSession, err := store.New(ctx, cesarUser)
	is.NoErr(err)
	is.True(cesarSession.ID != "")
	is.Equal(cesarSession.Username, "cesar")
	is.Equal(cesarSession.UserUUID, "cesar_uuid")
	is.True(cesarSession.Expiry.Equal(time.Time{}))

	minaUser := &entity.User{Username: "mina", UUID: "mina_uuid"}

	_, err = store.New(ctx, minaUser)
	is.NoErr(err)

	retrievedSession, err := store.Get(ctx, cesarSession.ID)
	is.NoErr(err)
	// Set the expiry of cesarSession to the expiry of the retrieved
	// session so we can use Equal. We know that they are not equal
	// because New does not set the expiry.
	is.True(!cesarSession.Expiry.Equal(retrievedSession.Expiry))
	cesarSession.Expiry = retrievedSession.Expiry
	is.Equal(cesarSession, retrievedSession)

	err = store.ExtendExpiry(ctx, cesarSession)
	is.NoErr(err)

	extendedCesarSession, err := store.Get(ctx, cesarSession.ID)
	is.NoErr(err)
	is.True(extendedCesarSession.Expiry.After(cesarSession.Expiry))

	err = store.Delete(ctx, extendedCesarSession)
	is.NoErr(err)

	_, err = store.Get(ctx, extendedCesarSession.ID)
	is.True(err != nil)
	store.Disconnect()
}
