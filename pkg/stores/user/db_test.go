package user

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/wrapperspb"

	commontest "github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/common"
	cpb "github.com/domino14/liwords/rpc/api/proto/config_service"
)

func recreateDB() (*DBStore, *pgxpool.Pool) {
	err := common.RecreateTestDB()
	if err != nil {
		panic(err)
	}

	pool, err := common.OpenTestingDB()
	if err != nil {
		panic(err)
	}

	// Create a user table. Initialize the user store.
	ustore, err := NewDBStore(pool)
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

	return ustore, pool
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestGet(t *testing.T) {
	is := is.New(t)
	ustore, pool := recreateDB()
	ctx := context.Background()

	cesarByUsername, err := ustore.Get(ctx, "cesar")
	is.NoErr(err)

	cesarByUUID, err := ustore.GetByUUID(ctx, cesarByUsername.UUID)
	is.NoErr(err)
	is.Equal(cesarByUsername.UUID, cesarByUUID.UUID)

	cesarByEmail, err := ustore.GetByEmail(ctx, cesarByUsername.Email)
	is.NoErr(err)
	is.Equal(cesarByUsername.UUID, cesarByEmail.UUID)

	apiKey := "somekey"
	err = common.UpdateWithPool(ctx, pool, []string{"api_key"}, []string{apiKey}, &common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByAPIKey, Value: apiKey})

	cesarByAPIKey, err := ustore.GetByAPIKey(ctx, apiKey)
	is.NoErr(err)
	is.Equal(cesarByUsername.UUID, cesarByAPIKey.UUID)
}

func TestSet(t *testing.T) {
	is := is.New(t)
	ustore, pool := recreateDB()
	ctx := context.Background()

	cesar, err := ustore.Get(ctx, "cesar")
	is.NoErr(err)

	uuid := cesar.UUID

	newNotoriety := 7
	err = ustore.SetNotoriety(ctx, uuid, newNotoriety)
	is.NoErr(err)
	cesar, err = ustore.Get(ctx, "cesar")
	is.NoErr(err)
	is.Equal(cesar.Notoriety, newNotoriety)

	req := &cpb.PermissionsRequest{
		Username: "cesar",
		Admin:    &wrapperspb.BoolValue{Value: true},
		Mod:      &wrapperspb.BoolValue{Value: true},
	}
	err = ustore.SetPermissions(ctx, req)
	is.NoErr(err)
	cesar, err = ustore.Get(ctx, "cesar")
	is.NoErr(err)
	is.True(cesar.IsAdmin)
	is.True(cesar.IsMod)
	is.True(!cesar.IsDirector)
	is.True(!cesar.IsBot)

	newPassword := "newpassword"
	err = ustore.SetPassword(ctx, uuid, newPassword)
	is.NoErr(err)
	cesar, err = ustore.Get(ctx, "cesar")
	is.NoErr(err)
	is.Equal(cesar.Password, newPassword)

	newAvatarURL := "newavatarurl"
	err = ustore.SetAvatarUrl(ctx, uuid, newAvatarURL)
	is.NoErr(err)
	cesar, err = ustore.Get(ctx, "cesar")
	is.NoErr(err)
	is.Equal(cesar.AvatarUrl(), newAvatarURL)

	newEmail := "cesar@wolges.io"
	newLastName := "del lunar"
	newAbout := "manegar of wolges"
	err = ustore.SetPersonalInfo(ctx, uuid, newEmail, "", newLastName, "", "", newAbout)
	is.NoErr(err)
	cesar, err = ustore.Get(ctx, "cesar")
	is.NoErr(err)
	is.Equal(cesar.Email, newEmail)
	is.Equal(cesar.Profile.FirstName, "")
	is.Equal(cesar.Profile.LastName, newLastName)
	is.Equal(cesar.Profile.CountryCode, "")
	is.Equal(cesar.Profile.About, newAbout)

	mina, err := ustore.Get(ctx, "mina")
	is.NoErr(err)
	variantKey := "burrito"
	newCesarRating := 1500.0
	newCesarSingleRating := &entity.SingleRating{
		Rating:            newCesarRating,
		RatingDeviation:   350.0,
		Volatility:        0.06,
		LastGameTimestamp: 0,
	}
	newMinaRating := 3000.0
	newMinaSingleRating := &entity.SingleRating{
		Rating:            newMinaRating,
		RatingDeviation:   20.0,
		Volatility:        0.06,
		LastGameTimestamp: 0,
	}
	err = ustore.SetRatings(ctx, uuid, mina.UUID, entity.VariantKey(variantKey), newCesarSingleRating, newMinaSingleRating)
	is.NoErr(err)
	cesar, err = ustore.Get(ctx, "cesar")
	is.NoErr(err)
	mina, err = ustore.Get(ctx, "mina")
	is.NoErr(err)

	cesarDBID, err := common.GetDBIDFromUUID(ctx, pool, &common.CommonDBConfig{
		TableType: common.UsersTable,
		Value:     uuid,
	})
	minaDBID, err := common.GetDBIDFromUUID(ctx, pool, &common.CommonDBConfig{
		TableType: common.UsersTable,
		Value:     mina.UUID,
	})
	actualCesarRating, err := common.GetUserRatingWithPool(ctx, pool, cesarDBID, entity.VariantKey(variantKey), &entity.SingleRating{})
	is.NoErr(err)
	is.True(commontest.WithinEpsilon(newCesarRating, actualCesarRating.Rating))
	actualMinaRating, err := common.GetUserRatingWithPool(ctx, pool, minaDBID, entity.VariantKey(variantKey), &entity.SingleRating{})
	is.NoErr(err)
	is.True(commontest.WithinEpsilon(newMinaRating, actualMinaRating.Rating))

	// stats

	newCesarStats := &entity.Stats{
		PlayerOneId: "cesar",
	}
	newMinaStats := &entity.Stats{
		PlayerOneId: "mina",
	}
	err = ustore.SetStats(ctx, uuid, mina.UUID, entity.VariantKey(variantKey), newCesarStats, newMinaStats)
	is.NoErr(err)
	cesar, err = ustore.Get(ctx, "cesar")
	is.NoErr(err)
	mina, err = ustore.Get(ctx, "mina")
	is.NoErr(err)

	actualCesarStats, err := common.GetUserStatsWithPool(ctx, pool, cesarDBID, entity.VariantKey(variantKey))
	is.NoErr(err)
	is.Equal(newCesarStats.PlayerOneId, actualCesarStats.PlayerOneId)
	actualMinaStats, err := common.GetUserStatsWithPool(ctx, pool, minaDBID, entity.VariantKey(variantKey))
	is.NoErr(err)
	is.Equal(newMinaStats.PlayerOneId, actualMinaStats.PlayerOneId)

	err = ustore.ResetStats(ctx, uuid)
	is.NoErr(err)

	// reset stats
	// reset ratings
	// reset stats and ratings
	// reset perfonsl inform
	// reset profile (stats, ratings, and personal info)
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
	is.NoErr(err)
	is.Equal(followed, []*entity.User{})

	followed, err = ustore.GetFollows(ctx, mina.ID)
	is.NoErr(err)
	is.Equal(followed, []*entity.User{
		{Username: "cesar", UUID: "mozEwaVMvTfUA2oxZfYN8k"},
		{Username: "jesse", UUID: "3xpEkpRAy3AizbVmDg3kdi"},
	})

	followed, err = ustore.GetFollows(ctx, jesse.ID)
	is.NoErr(err)
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
