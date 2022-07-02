package user

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	commontest "github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/glicko"
	"github.com/domino14/liwords/pkg/stores/common"
	cpb "github.com/domino14/liwords/rpc/api/proto/config_service"
	"github.com/domino14/liwords/rpc/api/proto/mod_service"
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	"github.com/domino14/liwords/rpc/api/proto/user_service"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

func recreateDB() (*DBStore, *pgxpool.Pool, context.Context) {
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
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	// Insert some dummy profiles into the database so that
	// user_id != id for profiles
	for i := 1; i <= 10; i++ {
		createDummyProfile(ctx, pool, 1)
	}

	for _, u := range []*entity.User{
		{Username: "cesar", Email: "cesar@woogles.io", UUID: "mozEwaVMvTfUA2oxZfYN8k"},
		{Username: "mina", Email: "mina@gmail.com", UUID: "iW7AaqNJDuaxgcYnrFfcJF"},
		{Username: "jesse", Email: "jesse@woogles.io", UUID: "3xpEkpRAy3AizbVmDg3kdi"},
		{Username: "HastyBot", Email: "hastybot@woogles.io", UUID: "hasty_bot_uuid", IsBot: true, Profile: &entity.Profile{BirthDate: "1400-01-01"}},
		{Username: "adult", Email: "adult@woogles.io", UUID: "adult_uuid", IsMod: true, IsAdmin: true, Profile: &entity.Profile{AvatarUrl: "adult_avatar_url", BirthDate: "1900-01-01", FirstName: "vincent", LastName: "adultman"}},
		{Username: "child", Email: "child@woogles.io", UUID: "child_uuid", Profile: &entity.Profile{AvatarUrl: "child_avatar_url", BirthDate: "2015-01-01", LastName: "kid"}},
		{Username: "winter", Email: "winter@woogles.io", UUID: "winter_uuid", IsMod: true, Profile: &entity.Profile{AvatarUrl: "winter_avatar_url", BirthDate: "1972-03-20", FirstName: "winter"}},
		{Username: "smith", Email: "smith@woogles.io", UUID: "smith_uuid", IsAdmin: true, Profile: &entity.Profile{AvatarUrl: "smith_avatar_url", BirthDate: "1950-07-08", LastName: "smith"}},
		{Username: "noprofile", Email: "noprofile@woogles.io", UUID: "noprofile_uuid", Profile: &entity.Profile{}},
		{Username: "mo", Email: "mo@woogles.io", UUID: "mo_uuid"},
		{Username: "mod", Email: "mod@woogles.io", UUID: "mod_uuid", Actions: &entity.Actions{History: []*mod_service.ModAction{{Type: mod_service.ModActionType_SUSPEND_ACCOUNT}}}},
		{Username: "mot", Email: "mot@woogles.io", UUID: "mot_uuid"},
		{Username: "mode", Email: "mode@woogles.io", UUID: "mode_uuid", IsBot: true},
		{Username: "moder", Email: "moder@woogles.io", UUID: "moder_uuid", Actions: &entity.Actions{Current: map[string]*mod_service.ModAction{mod_service.ModActionType_SUSPEND_GAMES.String(): {Type: mod_service.ModActionType_SUSPEND_GAMES}}}},
		{Username: "modern", Email: "modern@woogles.io", UUID: "modern_uuid"},
		{Username: "moderne", Email: "moderne@woogles.io", UUID: "moderne_uuid", Actions: &entity.Actions{Current: map[string]*mod_service.ModAction{mod_service.ModActionType_SUSPEND_ACCOUNT.String(): {Type: mod_service.ModActionType_SUSPEND_ACCOUNT, EndTime: timestamppb.New(time.Now())}}}},
		{Username: "moderns", Email: "moderns@woogles.io", UUID: "moderns_uuid"},
		{Username: "modernes", Email: "modernes@woogles.io", UUID: "modernes_uuid", Actions: &entity.Actions{Current: map[string]*mod_service.ModAction{mod_service.ModActionType_SUSPEND_ACCOUNT.String(): {Type: mod_service.ModActionType_SUSPEND_ACCOUNT}}}},
		{Username: "modernest", Email: "modernest@woogles.io", UUID: "modernest_uuid"},
	} {
		err = ustore.New(context.Background(), u)
		if err != nil {
			log.Fatal().Err(err).Msg("error")
		}
	}

	return ustore, pool, ctx
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestGet(t *testing.T) {
	is := is.New(t)
	ustore, pool, ctx := recreateDB()

	cesarByUsername, err := ustore.Get(ctx, "cesar")
	is.NoErr(err)

	cesarByUUID, err := ustore.GetByUUID(ctx, cesarByUsername.UUID)
	is.NoErr(err)
	is.Equal(cesarByUsername.UUID, cesarByUUID.UUID)

	cesarByEmail, err := ustore.GetByEmail(ctx, cesarByUsername.Email)
	is.NoErr(err)
	is.Equal(cesarByUsername.UUID, cesarByEmail.UUID)

	apiKey := "somekey"
	err = common.UpdateWithPool(ctx, pool, []string{"api_key"}, []interface{}{apiKey}, &common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByUUID, Value: cesarByUsername.UUID})
	is.NoErr(err)

	cesarByAPIKey, err := ustore.GetByAPIKey(ctx, apiKey)
	is.NoErr(err)
	is.Equal(cesarByUsername.UUID, cesarByAPIKey.UUID)

	profileUUIDs := []string{"hasty_bot_uuid", "adult_uuid", "child_uuid", "winter_uuid", "smith_uuid", "noprofile_uuid"}
	profiles, err := ustore.GetBriefProfiles(ctx, profileUUIDs)
	is.NoErr(err)
	is.Equal(profiles["hasty_bot_uuid"].AvatarUrl, "https://woogles-prod-assets.s3.amazonaws.com/macondog.png")
	is.Equal(profiles["hasty_bot_uuid"].FullName, "")
	is.Equal(profiles["adult_uuid"].AvatarUrl, "adult_avatar_url")
	is.Equal(profiles["adult_uuid"].FullName, "vincent adultman")
	is.Equal(profiles["child_uuid"].AvatarUrl, "")
	is.Equal(profiles["child_uuid"].FullName, "")
	is.Equal(profiles["winter_uuid"].AvatarUrl, "winter_avatar_url")
	is.Equal(profiles["winter_uuid"].FullName, "winter")
	is.Equal(profiles["smith_uuid"].AvatarUrl, "smith_avatar_url")
	is.Equal(profiles["smith_uuid"].FullName, "smith")
	is.Equal(profiles["noprofile_uuid"].Username, "noprofile")
	is.Equal(profiles["noprofile_uuid"].FullName, "")
	is.Equal(profiles["noprofile_uuid"].CountryCode, "")
	is.Equal(profiles["noprofile_uuid"].AvatarUrl, "")

	bot, err := ustore.GetBot(ctx, macondopb.BotRequest_HASTY_BOT)
	is.NoErr(err)
	is.Equal(bot.Username, botNames[macondopb.BotRequest_HASTY_BOT])

	// This bot will not be found, so GetBot should pick a random bot
	// Since there is only one bot in the db, it will pick that bot,
	// which is HastyBot.
	bot, err = ustore.GetBot(ctx, macondopb.BotRequest_LEVEL2_CEL_BOT)
	is.NoErr(err)
	is.True(bot.Username == botNames[macondopb.BotRequest_HASTY_BOT] || bot.Username == "mode")

	// GetModList
	modList, err := ustore.GetModList(ctx)
	is.NoErr(err)
	sort.Strings(modList.AdminUserIds)
	sort.Strings(modList.ModUserIds)
	is.Equal(modList.AdminUserIds, []string{"adult_uuid", "smith_uuid"})
	is.Equal(modList.ModUserIds, []string{"adult_uuid", "winter_uuid"})
	ustore.Disconnect()
}

func TestGetNullValues(t *testing.T) {
	is := is.New(t)
	ustore, pool, ctx := recreateDB()

	cesarByUsername, err := ustore.Get(ctx, "cesar")
	is.NoErr(err)

	setNullValues(ctx, pool, cesarByUsername.UUID)

	cesarByUsername, err = ustore.Get(ctx, "cesar")
	is.NoErr(err)

	cesarByUUID, err := ustore.GetByUUID(ctx, cesarByUsername.UUID)
	is.NoErr(err)
	is.Equal(cesarByUsername.UUID, cesarByUUID.UUID)

	cesarEmail := "cesar@woogles.io"
	err = common.UpdateWithPool(ctx, pool, []string{"email"}, []interface{}{cesarEmail}, &common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByUUID, Value: cesarByUsername.UUID})
	is.NoErr(err)

	cesarByEmail, err := ustore.GetByEmail(ctx, cesarEmail)
	is.NoErr(err)
	is.Equal(cesarByUsername.UUID, cesarByEmail.UUID)

	apiKey := "somekey"
	err = common.UpdateWithPool(ctx, pool, []string{"api_key"}, []interface{}{apiKey}, &common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByUUID, Value: cesarByUsername.UUID})
	is.NoErr(err)

	cesarByAPIKey, err := ustore.GetByAPIKey(ctx, apiKey)
	is.NoErr(err)
	is.Equal(cesarByUsername.UUID, cesarByAPIKey.UUID)

	profileUUIDs := []string{cesarByUsername.UUID}
	profiles, err := ustore.GetBriefProfiles(ctx, profileUUIDs)
	is.NoErr(err)
	is.Equal(profiles[cesarByUsername.UUID].Username, "cesar")
	is.Equal(profiles[cesarByUsername.UUID].FullName, "")
	is.Equal(profiles[cesarByUsername.UUID].CountryCode, "")
	is.Equal(profiles[cesarByUsername.UUID].AvatarUrl, "")

	err = common.UpdateWithPool(ctx, pool, []string{"is_mod", "is_admin"}, []interface{}{nil, true}, &common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByUUID, Value: cesarByUsername.UUID})
	is.NoErr(err)
	// GetModList
	modList, err := ustore.GetModList(ctx)
	is.NoErr(err)
	sort.Strings(modList.AdminUserIds)
	sort.Strings(modList.ModUserIds)
	is.Equal(modList.AdminUserIds, []string{"adult_uuid", cesarByUsername.UUID, "smith_uuid"})
	is.Equal(modList.ModUserIds, []string{"adult_uuid", "winter_uuid"})
	ustore.Disconnect()
}

func TestSet(t *testing.T) {
	is := is.New(t)
	ustore, pool, ctx := recreateDB()

	cesar, err := ustore.Get(ctx, "cesar")
	is.NoErr(err)

	newNotoriety := 7
	err = ustore.SetNotoriety(ctx, cesar.UUID, newNotoriety)
	is.NoErr(err)
	cesar, err = ustore.Get(ctx, "cesar")
	is.NoErr(err)
	is.Equal(cesar.Notoriety, newNotoriety)

	newActions := &entity.Actions{
		Current: map[string]*ms.ModAction{
			"SUSPEND_ACCOUNT": {UserId: cesar.UUID, Type: ms.ModActionType_SUSPEND_ACCOUNT, Duration: 100},
		},
		History: []*ms.ModAction{
			{UserId: cesar.UUID, Type: ms.ModActionType_MUTE, Duration: 1000},
		},
	}
	err = ustore.SetActions(ctx, cesar.UUID, newActions)
	is.NoErr(err)
	cesar, err = ustore.Get(ctx, "cesar")
	is.NoErr(err)
	is.Equal(cesar.Actions, newActions)

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
	err = ustore.SetPassword(ctx, cesar.UUID, newPassword)
	is.NoErr(err)
	cesar, err = ustore.Get(ctx, "cesar")
	is.NoErr(err)
	is.Equal(cesar.Password, newPassword)

	newAvatarURL := "newavatarurl"
	err = ustore.SetAvatarUrl(ctx, cesar.UUID, newAvatarURL)
	is.NoErr(err)
	cesar, err = ustore.Get(ctx, "cesar")
	is.NoErr(err)
	is.Equal(cesar.AvatarUrl(), newAvatarURL)

	newEmail := "cesar@wolges.io"
	newLastName := "del lunar"
	newAbout := "manegar of wolges"
	err = ustore.SetPersonalInfo(ctx, cesar.UUID, newEmail, "", newLastName, "", "", newAbout, false)
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
	newCesarRating := 500.0
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
	err = ustore.SetRatings(ctx, cesar.UUID, mina.UUID, entity.VariantKey(variantKey), newCesarSingleRating, newMinaSingleRating)
	is.NoErr(err)
	mina, err = ustore.Get(ctx, "mina")
	is.NoErr(err)

	cesarDBID, err := common.GetDBIDFromUUID(ctx, pool, &common.CommonDBConfig{
		TableType: common.UsersTable,
		Value:     cesar.UUID,
	})
	is.NoErr(err)
	minaDBID, err := common.GetDBIDFromUUID(ctx, pool, &common.CommonDBConfig{
		TableType: common.UsersTable,
		Value:     mina.UUID,
	})
	is.NoErr(err)
	actualCesarRating, err := common.GetUserRatingWithPool(ctx, pool, cesarDBID, entity.VariantKey(variantKey))
	is.NoErr(err)
	is.True(commontest.WithinEpsilon(newCesarRating, actualCesarRating.Rating))
	actualMinaRating, err := common.GetUserRatingWithPool(ctx, pool, minaDBID, entity.VariantKey(variantKey))
	is.NoErr(err)
	is.True(commontest.WithinEpsilon(newMinaRating, actualMinaRating.Rating))

	// stats

	newCesarStats := &entity.Stats{
		PlayerOneId: "cesar",
	}
	newMinaStats := &entity.Stats{
		PlayerOneId: "mina",
	}
	err = ustore.SetStats(ctx, cesar.UUID, mina.UUID, entity.VariantKey(variantKey), newCesarStats, newMinaStats)
	is.NoErr(err)
	mina, err = ustore.Get(ctx, "mina")
	is.NoErr(err)

	actualCesarStats, err := common.GetUserStatsWithPool(ctx, pool, cesarDBID, entity.VariantKey(variantKey))
	is.NoErr(err)
	is.Equal(newCesarStats.PlayerOneId, actualCesarStats.PlayerOneId)
	actualMinaStats, err := common.GetUserStatsWithPool(ctx, pool, minaDBID, entity.VariantKey(variantKey))
	is.NoErr(err)
	is.Equal(newMinaStats.PlayerOneId, actualMinaStats.PlayerOneId)

	err = ustore.ResetStats(ctx, cesar.UUID)
	is.NoErr(err)
	actualCesarStats, err = common.GetUserStatsWithPool(ctx, pool, cesarDBID, entity.VariantKey(variantKey))
	is.NoErr(err)
	is.Equal(actualCesarStats.PlayerOneId, "")

	err = ustore.ResetRatings(ctx, cesar.UUID)
	is.NoErr(err)
	actualCesarRating, err = common.GetUserRatingWithPool(ctx, pool, cesarDBID, entity.VariantKey(variantKey))
	is.NoErr(err)
	is.True(commontest.WithinEpsilon(actualCesarRating.Rating, float64(glicko.InitialRating)))

	err = ustore.ResetStatsAndRatings(ctx, mina.UUID)
	is.NoErr(err)
	actualMinaStats, err = common.GetUserStatsWithPool(ctx, pool, minaDBID, entity.VariantKey(variantKey))
	is.NoErr(err)
	is.Equal(actualMinaStats.PlayerOneId, "")
	actualMinaRating, err = common.GetUserRatingWithPool(ctx, pool, minaDBID, entity.VariantKey(variantKey))
	is.NoErr(err)
	is.True(commontest.WithinEpsilon(actualMinaRating.Rating, float64(glicko.InitialRating)))

	err = ustore.ResetPersonalInfo(ctx, cesar.UUID)
	is.NoErr(err)
	cesar, err = ustore.Get(ctx, "cesar")
	is.NoErr(err)
	is.Equal(cesar.Email, "cesar@wolges.io")
	is.Equal(cesar.Profile.FirstName, "")
	is.Equal(cesar.Profile.LastName, "")
	is.Equal(cesar.Profile.CountryCode, "")
	is.Equal(cesar.Profile.About, "")
	is.Equal(cesar.Profile.Title, "")

	err = ustore.SetRatings(ctx, cesar.UUID, mina.UUID, entity.VariantKey(variantKey), newCesarSingleRating, newMinaSingleRating)
	is.NoErr(err)
	err = ustore.SetStats(ctx, cesar.UUID, mina.UUID, entity.VariantKey(variantKey), newCesarStats, newMinaStats)
	is.NoErr(err)

	err = ustore.ResetProfile(ctx, mina.UUID)
	is.NoErr(err)

	actualMinaStats, err = common.GetUserStatsWithPool(ctx, pool, minaDBID, entity.VariantKey(variantKey))
	is.NoErr(err)
	is.Equal(actualMinaStats.PlayerOneId, "")

	actualMinaRating, err = common.GetUserRatingWithPool(ctx, pool, minaDBID, entity.VariantKey(variantKey))
	is.NoErr(err)
	is.True(commontest.WithinEpsilon(actualMinaRating.Rating, float64(glicko.InitialRating)))

	mina, err = ustore.Get(ctx, "mina")
	is.NoErr(err)
	is.Equal(mina.Email, "mina@gmail.com")
	is.Equal(mina.Profile.FirstName, "")
	is.Equal(mina.Profile.LastName, "")
	is.Equal(mina.Profile.CountryCode, "")
	is.Equal(mina.Profile.About, "")
	ustore.Disconnect()
}

func TestMisc(t *testing.T) {
	is := is.New(t)
	ustore, _, ctx := recreateDB()

	users, err := ustore.UsersByPrefix(ctx, "m")
	is.NoErr(err)
	usernames := getUsernamesFromBasicUsers(users)
	sort.Strings(usernames)
	is.Equal(usernames, []string{"mina", "mo", "mod", "moder", "modern", "moderne", "modernest", "moderns", "mot"})

	users, err = ustore.UsersByPrefix(ctx, "mo")
	is.NoErr(err)
	usernames = getUsernamesFromBasicUsers(users)
	sort.Strings(usernames)
	is.Equal(usernames, []string{"mo", "mod", "moder", "modern", "moderne", "modernest", "moderns", "mot"})

	users, err = ustore.UsersByPrefix(ctx, "mod")
	is.NoErr(err)
	usernames = getUsernamesFromBasicUsers(users)
	sort.Strings(usernames)
	is.Equal(usernames, []string{"mod", "moder", "modern", "moderne", "modernest", "moderns"})

	users, err = ustore.UsersByPrefix(ctx, "mode")
	is.NoErr(err)
	usernames = getUsernamesFromBasicUsers(users)
	sort.Strings(usernames)
	is.Equal(usernames, []string{"moder", "modern", "moderne", "modernest", "moderns"})

	users, err = ustore.UsersByPrefix(ctx, "moder")
	is.NoErr(err)
	usernames = getUsernamesFromBasicUsers(users)
	sort.Strings(usernames)
	is.Equal(usernames, []string{"moder", "modern", "moderne", "modernest", "moderns"})

	users, err = ustore.UsersByPrefix(ctx, "modern")
	is.NoErr(err)
	usernames = getUsernamesFromBasicUsers(users)
	sort.Strings(usernames)
	is.Equal(usernames, []string{"modern", "moderne", "modernest", "moderns"})

	users, err = ustore.UsersByPrefix(ctx, "moderne")
	is.NoErr(err)
	usernames = getUsernamesFromBasicUsers(users)
	sort.Strings(usernames)
	is.Equal(usernames, []string{"moderne", "modernest"})

	users, err = ustore.UsersByPrefix(ctx, "modernes")
	is.NoErr(err)
	usernames = getUsernamesFromBasicUsers(users)
	sort.Strings(usernames)
	is.Equal(usernames, []string{"modernest"})

	users, err = ustore.UsersByPrefix(ctx, "modernest")
	is.NoErr(err)
	usernames = getUsernamesFromBasicUsers(users)
	sort.Strings(usernames)
	is.Equal(usernames, []string{"modernest"})

	users, err = ustore.UsersByPrefix(ctx, "modernist")
	is.NoErr(err)
	is.Equal(len(users), 0)

	// ListAllIDs
	allUsers, err := ustore.ListAllIDs(ctx)
	is.NoErr(err)
	is.Equal(len(allUsers), 19)

	winterUsername, err := ustore.Username(ctx, "winter_uuid")
	is.NoErr(err)
	is.Equal(winterUsername, "winter")

	count, err := ustore.Count(ctx)
	is.NoErr(err)
	is.Equal(count, int64(19))
	ustore.Disconnect()
}

func TestBlocks(t *testing.T) {
	is := is.New(t)
	ustore, _, ctx := recreateDB()

	adult, err := ustore.Get(ctx, "adult")
	is.NoErr(err)
	smith, err := ustore.Get(ctx, "smith")
	is.NoErr(err)
	winter, err := ustore.Get(ctx, "winter")
	is.NoErr(err)

	is.NoErr(ustore.AddBlock(ctx, adult.ID, smith.ID))
	is.NoErr(ustore.AddBlock(ctx, winter.ID, smith.ID))
	is.NoErr(ustore.AddBlock(ctx, adult.ID, winter.ID))

	// Try to add a block that already exists
	err = ustore.AddBlock(ctx, winter.ID, smith.ID)
	is.True(err != nil)

	blocked, err := ustore.GetBlocks(ctx, adult.ID)
	is.NoErr(err)
	is.Equal(blocked, []*entity.User{})

	blocked, err = ustore.GetBlocks(ctx, smith.ID)
	is.NoErr(err)
	sortUsers(blocked)
	is.Equal(blocked, []*entity.User{
		{Username: "adult", UUID: "adult_uuid"},
		{Username: "winter", UUID: "winter_uuid"},
	})

	blocked, err = ustore.GetBlockedBy(ctx, smith.ID)
	is.NoErr(err)
	is.Equal(blocked, []*entity.User{})

	blocked, err = ustore.GetBlockedBy(ctx, adult.ID)
	is.NoErr(err)
	sortUsers(blocked)
	is.Equal(blocked, []*entity.User{
		{Username: "smith", UUID: "smith_uuid"},
		{Username: "winter", UUID: "winter_uuid"},
	})

	blocked, err = ustore.GetFullBlocks(ctx, winter.ID)
	is.NoErr(err)
	sortUsers(blocked)
	is.Equal(blocked, []*entity.User{
		{Username: "adult", UUID: "adult_uuid"},
		{Username: "smith", UUID: "smith_uuid"},
	})

	blocked, err = ustore.GetFullBlocks(ctx, smith.ID)
	is.NoErr(err)
	sortUsers(blocked)
	is.Equal(blocked, []*entity.User{
		{Username: "adult", UUID: "adult_uuid"},
		{Username: "winter", UUID: "winter_uuid"},
	})

	blocked, err = ustore.GetFullBlocks(ctx, winter.ID)
	is.NoErr(err)
	sortUsers(blocked)
	is.Equal(blocked, []*entity.User{
		{Username: "adult", UUID: "adult_uuid"},
		{Username: "smith", UUID: "smith_uuid"},
	})

	err = ustore.AddBlock(ctx, smith.ID, adult.ID)
	is.NoErr(err)

	blocked, err = ustore.GetFullBlocks(ctx, adult.ID)
	is.NoErr(err)
	sortUsers(blocked)
	is.Equal(blocked, []*entity.User{
		{Username: "smith", UUID: "smith_uuid"},
		{Username: "winter", UUID: "winter_uuid"},
	})

	blocked, err = ustore.GetFullBlocks(ctx, smith.ID)
	is.NoErr(err)
	sortUsers(blocked)
	is.Equal(blocked, []*entity.User{
		{Username: "adult", UUID: "adult_uuid"},
		{Username: "winter", UUID: "winter_uuid"},
	})

	blocked, err = ustore.GetFullBlocks(ctx, winter.ID)
	is.NoErr(err)
	sortUsers(blocked)
	is.Equal(blocked, []*entity.User{
		{Username: "adult", UUID: "adult_uuid"},
		{Username: "smith", UUID: "smith_uuid"},
	})

	err = ustore.RemoveBlock(ctx, smith.ID, adult.ID)
	is.NoErr(err)
	err = ustore.RemoveBlock(ctx, winter.ID, smith.ID)
	is.NoErr(err)

	blocked, err = ustore.GetFullBlocks(ctx, adult.ID)
	is.NoErr(err)
	sortUsers(blocked)
	is.Equal(blocked, []*entity.User{
		{Username: "smith", UUID: "smith_uuid"},
		{Username: "winter", UUID: "winter_uuid"},
	})

	blocked, err = ustore.GetFullBlocks(ctx, smith.ID)
	is.NoErr(err)
	is.Equal(blocked, []*entity.User{{Username: "adult", UUID: "adult_uuid"}})

	blocked, err = ustore.GetFullBlocks(ctx, winter.ID)
	is.NoErr(err)
	is.Equal(blocked, []*entity.User{{Username: "adult", UUID: "adult_uuid"}})

	// Try to remove a block that does not exist
	err = ustore.RemoveBlock(ctx, smith.ID, adult.ID)
	is.True(err != nil)
	is.Equal(err.Error(), "block does not exist")
	ustore.Disconnect()
}

func TestAddFollower(t *testing.T) {
	is := is.New(t)
	ustore, _, ctx := recreateDB()

	cesar, err := ustore.Get(ctx, "cesar")
	is.NoErr(err)
	mina, err := ustore.Get(ctx, "mina")
	is.NoErr(err)
	jesse, err := ustore.Get(ctx, "jesse")
	is.NoErr(err)

	is.NoErr(ustore.AddFollower(ctx, cesar.ID, mina.ID))
	is.NoErr(ustore.AddFollower(ctx, jesse.ID, mina.ID))
	is.NoErr(ustore.AddFollower(ctx, cesar.ID, jesse.ID))

	followed, err := ustore.GetFollows(ctx, cesar.ID)
	is.NoErr(err)
	is.Equal(followed, []*entity.User{})

	followed, err = ustore.GetFollows(ctx, mina.ID)
	is.NoErr(err)
	sortUsers(followed)
	is.Equal(followed, []*entity.User{
		{Username: cesar.Username, UUID: cesar.UUID},
		{Username: jesse.Username, UUID: jesse.UUID},
	})

	followed, err = ustore.GetFollows(ctx, jesse.ID)
	is.NoErr(err)
	is.Equal(followed, []*entity.User{
		{Username: cesar.Username, UUID: cesar.UUID},
	})

	followed, err = ustore.GetFollowedBy(ctx, cesar.ID)
	is.NoErr(err)
	sortUsers(followed)
	is.Equal(followed, []*entity.User{
		{Username: jesse.Username, UUID: jesse.UUID},
		{Username: mina.Username, UUID: mina.UUID},
	})
	ustore.Disconnect()
}

func TestRemoveFollower(t *testing.T) {
	is := is.New(t)
	ustore, _, ctx := recreateDB()
	cesar, err := ustore.Get(ctx, "cesar")
	is.NoErr(err)
	mina, err := ustore.Get(ctx, "mina")
	is.NoErr(err)
	jesse, err := ustore.Get(ctx, "jesse")
	is.NoErr(err)

	is.NoErr(ustore.AddFollower(ctx, cesar.ID, mina.ID))
	is.NoErr(ustore.AddFollower(ctx, jesse.ID, mina.ID))
	is.NoErr(ustore.AddFollower(ctx, cesar.ID, jesse.ID))

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
	ustore, _, ctx := recreateDB()
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
	ustore, _, ctx := recreateDB()
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

func setNullValues(ctx context.Context, pool *pgxpool.Pool, uuid string) {
	tx, err := pool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE users SET password = NULL, internal_bot = NULL, is_admin = NULL, api_key = NULL, is_director = NULL, is_mod = NULL, actions = NULL, notoriety = NULL WHERE uuid = $1`, uuid)
	if err != nil {
		panic(err)
	}

	id, err := common.GetUserDBIDFromUUID(ctx, tx, uuid)
	if err != nil {
		panic(err)
	}

	_, err = tx.Exec(ctx, `UPDATE profiles SET first_name = NULL, last_name = NULL, country_code = NULL, title = NULL, about = NULL, ratings = NULL, stats = NULL, avatar_url = NULL, birth_date = NULL WHERE id = $1`, id)
	if err != nil {
		panic(err)
	}

	if err := tx.Commit(ctx); err != nil {
		panic(err)
	}
}

func getUsernamesFromBasicUsers(basicUsers []*user_service.BasicUser) []string {
	s := make([]string, len(basicUsers))
	for idx, bs := range basicUsers {
		s[idx] = bs.Username
	}
	return s
}

func sortUsers(users []*entity.User) {
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})
}

func createDummyProfile(ctx context.Context, pool *pgxpool.Pool, userId int) error {
	tx, err := pool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO profiles (user_id, first_name, ratings, stats, created_at, updated_at) VALUES ($1, $2, $3, $4, NOW(), NOW())`,
		userId, fmt.Sprintf("firstname-%d", userId), entity.Ratings{}, entity.ProfileStats{})
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
