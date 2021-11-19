package service

import (
	"context"

	"os"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/user"
	pkguser "github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/mod_service"
	"github.com/twitchtv/twirp"
)

var TestDBHost = os.Getenv("TEST_DB_HOST")
var TestingDBConnStr = "host=" + TestDBHost + " port=5432 user=postgres password=pass sslmode=disable"

func recreateDB() {
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

	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")

	for _, u := range []*entity.User{
		{Username: "Spammer", Email: os.Getenv("TEST_EMAIL_USERNAME") + "+spammer@woogles.io", UUID: "Spammer"},
		{Username: "Sandbagger", Email: "sandbagger@gmail.com", UUID: "Sandbagger"},
		{Username: "Cheater", Email: os.Getenv("TEST_EMAIL_USERNAME") + "@woogles.io", UUID: "Cheater"},
		{Username: "Hacker", Email: "hacker@woogles.io", UUID: "Hacker"},
		{Username: "Deleter", Email: "deleter@woogles.io", UUID: "Deleter"},
		{Username: "Moderator", Email: "admin@woogles.io", UUID: "Moderator", IsMod: true},
	} {
		err = ustore.New(context.Background(), u)
		if err != nil {
			log.Fatal().Err(err).Msg("error")
		}
	}
	ustore.(*user.DBStore).Disconnect()
}

func userStore(dbURL string) pkguser.Store {
	ustore, err := user.NewDBStore(TestingDBConnStr + " dbname=liwords_test")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	return ustore
}

func TestAuthenticateMod(t *testing.T) {
	is := is.New(t)

	session := &entity.Session{
		ID:       "abcdef",
		Username: "Moderator",
		UserUUID: "Moderator",
		Expiry:   time.Now().Add(time.Second * 100)}
	ctx := context.Background()
	ctx = apiserver.PlaceInContext(ctx, session)

	cstr := TestingDBConnStr + " dbname=liwords_test"
	recreateDB()
	us := userStore(cstr)
	ms := &ModService{userStore: us}

	err := authenticateMod(ctx, ms, &pb.ModActionsList{
		Actions: []*pb.ModAction{},
	})
	is.NoErr(err)
	us.(*user.DBStore).Disconnect()
}

func TestAuthenticateModNoAuth(t *testing.T) {
	is := is.New(t)

	session := &entity.Session{
		ID:       "defghi",
		Username: "Cheater",
		UserUUID: "Cheater",
		Expiry:   time.Now().Add(time.Second * 100)}
	ctx := context.Background()
	ctx = apiserver.PlaceInContext(ctx, session)

	cstr := TestingDBConnStr + " dbname=liwords_test"
	recreateDB()
	us := userStore(cstr)
	ms := &ModService{userStore: us}

	err := authenticateMod(ctx, ms, &pb.ModActionsList{
		Actions: []*pb.ModAction{},
	})
	is.Equal(err, twirp.NewError(twirp.Unauthenticated, errNotAuthorized.Error()))
	us.(*user.DBStore).Disconnect()
}
