package user

import (
	"context"
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/entity"
)

var TestDBHost = os.Getenv("TEST_DB_HOST")

var TestingDBConnStr = "host=" + TestDBHost + " port=5432 user=postgres password=pass sslmode=disable"

func recreateDB() *DBStore {
	// Create a database.
	db, err := gorm.Open("postgres", TestingDBConnStr+" dbname=postgres")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	defer db.Close()
	db = db.Exec("DROP DATABASE IF EXISTS liwords_test")
	if db.Error != nil {
		log.Fatal().Err(db.Error).Msg("error")
	}
	db = db.Exec("CREATE DATABASE liwords_test")
	if db.Error != nil {
		log.Fatal().Err(db.Error).Msg("error")
	}
	// Create a user table. Initialize the user store.
	ustore, err := NewDBStore(TestingDBConnStr + " dbname=liwords_test")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}

	// Insert a couple of users into the table.

	for _, u := range []*entity.User{
		{Username: "cesar", Email: "cesar@woogles.io", UUID: "mozEwaVMvTfUA2oxZfYN8k"},
		{Username: "mina", Email: "mina@gmail.com", UUID: "iW7AaqNJDuaxgcYnrFfcJF"},
		{Username: "jesse", Email: "jesse@woogles.io", UUID: "3xpEkpRAy3AizbVmDg3kdi"},
	} {
		err = ustore.New(context.Background(), u)
		if err != nil {
			log.Fatal().Err(err).Msg("error")
		}
	}

	return ustore
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestAddFollower(t *testing.T) {
	is := is.New(t)
	ustore := recreateDB()
	ctx := context.Background()
	cesar, err := ustore.Get(ctx, "cesar")
	is.NoErr(err)
	mina, err := ustore.Get(ctx, "mina")
	is.NoErr(err)
	jesse, err := ustore.Get(ctx, "jesse")
	is.NoErr(err)

	ustore.AddFollower(ctx, cesar.ID, mina.ID)
	ustore.AddFollower(ctx, jesse.ID, mina.ID)
	ustore.AddFollower(ctx, cesar.ID, jesse.ID)

	followed, err := ustore.GetFollows(ctx, cesar.ID)
	is.Equal(followed, []*entity.User{})

	followed, err = ustore.GetFollows(ctx, mina.ID)
	is.Equal(followed, []*entity.User{
		{Username: "cesar", UUID: "mozEwaVMvTfUA2oxZfYN8k"},
		{Username: "jesse", UUID: "3xpEkpRAy3AizbVmDg3kdi"},
	})

	followed, err = ustore.GetFollows(ctx, jesse.ID)
	is.Equal(followed, []*entity.User{
		{Username: "cesar", UUID: "mozEwaVMvTfUA2oxZfYN8k"},
	})

	ustore.Disconnect()
}

func TestRemoveFollower(t *testing.T) {
	is := is.New(t)
	ustore := recreateDB()
	ctx := context.Background()
	cesar, err := ustore.Get(ctx, "cesar")
	is.NoErr(err)
	mina, err := ustore.Get(ctx, "mina")
	is.NoErr(err)
	jesse, err := ustore.Get(ctx, "jesse")
	is.NoErr(err)

	ustore.AddFollower(ctx, cesar.ID, mina.ID)
	ustore.AddFollower(ctx, jesse.ID, mina.ID)
	ustore.AddFollower(ctx, cesar.ID, jesse.ID)

	ustore.RemoveFollower(ctx, jesse.ID, mina.ID)

	followed, err := ustore.GetFollows(ctx, mina.ID)
	is.NoErr(err)
	is.Equal(followed, []*entity.User{
		{Username: "cesar", UUID: "mozEwaVMvTfUA2oxZfYN8k"},
	})

	ustore.Disconnect()
}

func TestAddDuplicateFollower(t *testing.T) {
	is := is.New(t)
	ustore := recreateDB()
	ctx := context.Background()
	cesar, err := ustore.Get(ctx, "cesar")
	is.NoErr(err)
	mina, err := ustore.Get(ctx, "mina")
	is.NoErr(err)
	jesse, err := ustore.Get(ctx, "jesse")
	is.NoErr(err)

	is.NoErr(ustore.AddFollower(ctx, cesar.ID, mina.ID))
	is.NoErr(ustore.AddFollower(ctx, jesse.ID, mina.ID))
	is.NoErr(ustore.AddFollower(ctx, cesar.ID, jesse.ID))

	err = ustore.AddFollower(ctx, jesse.ID, mina.ID)
	is.True(err != nil)
	ustore.Disconnect()
}

func TestRemoveNonexistentFollower(t *testing.T) {
	is := is.New(t)
	ustore := recreateDB()
	ctx := context.Background()
	cesar, err := ustore.Get(ctx, "cesar")
	is.NoErr(err)
	mina, err := ustore.Get(ctx, "mina")
	is.NoErr(err)
	jesse, err := ustore.Get(ctx, "jesse")
	is.NoErr(err)

	is.NoErr(ustore.AddFollower(ctx, cesar.ID, mina.ID))
	is.NoErr(ustore.AddFollower(ctx, jesse.ID, mina.ID))
	is.NoErr(ustore.AddFollower(ctx, cesar.ID, jesse.ID))

	err = ustore.RemoveFollower(ctx, jesse.ID, cesar.ID)
	is.NoErr(err)
	// Doesn't throw an error...

	ustore.Disconnect()
}
