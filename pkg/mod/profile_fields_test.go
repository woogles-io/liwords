package mod

import (
	"context"
	"testing"
	"time"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

// The profile the removals start from, before each case.
var fullProfile = entity.Profile{
	FirstName:   "Hack",
	LastName:    "Er",
	BirthDate:   "1990-01-02",
	CountryCode: "nz",
	About:       "i hack things",
	AvatarUrl:   "https://example.invalid/Hacker-abcdefghi.jpg",
}

func TestRemoveProfileFields(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	recreateDB()
	us := userStore()
	defer us.(*user.DBStore).Disconnect()

	for _, tc := range []struct {
		actionType ms.ModActionType
		oldValue   string
		cleared    func(p entity.Profile) entity.Profile
	}{
		{ms.ModActionType_REMOVE_FIRST_NAME, fullProfile.FirstName,
			func(p entity.Profile) entity.Profile { p.FirstName = ""; return p }},
		{ms.ModActionType_REMOVE_LAST_NAME, fullProfile.LastName,
			func(p entity.Profile) entity.Profile { p.LastName = ""; return p }},
		{ms.ModActionType_REMOVE_ABOUT, fullProfile.About,
			func(p entity.Profile) entity.Profile { p.About = ""; return p }},
		{ms.ModActionType_REMOVE_AVATAR, fullProfile.AvatarUrl,
			func(p entity.Profile) entity.Profile { p.AvatarUrl = ""; return p }},
	} {
		t.Run(tc.actionType.String(), func(t *testing.T) {
			is := is.New(t)
			is.NoErr(us.SetPersonalInfo(ctx, "Hacker", "hacker@woogles.io", fullProfile.FirstName,
				fullProfile.LastName, fullProfile.BirthDate, fullProfile.CountryCode, fullProfile.About))
			is.NoErr(us.SetAvatarUrl(ctx, "Hacker", fullProfile.AvatarUrl))

			is.NoErr(ApplyActions(ctx, us, chatStore(), "Moderator",
				[]*ms.ModAction{{UserId: "Hacker", Type: tc.actionType, Note: "offensive"}}))

			// Exactly the one field is empty; everything else is as it was.
			u, err := us.GetByUUID(ctx, "Hacker")
			is.NoErr(err)
			want := tc.cleared(fullProfile)
			is.Equal(u.Profile.FirstName, want.FirstName)
			is.Equal(u.Profile.LastName, want.LastName)
			is.Equal(u.Profile.BirthDate, want.BirthDate)
			is.Equal(u.Profile.CountryCode, want.CountryCode)
			is.Equal(u.Profile.About, want.About)
			is.Equal(u.Profile.AvatarUrl, want.AvatarUrl)
			is.Equal(u.Email, "hacker@woogles.io")

			// The history keeps the old value, and the action is over at once.
			history, err := GetActionHistory(ctx, us, "Hacker")
			is.NoErr(err)
			var last *ms.ModAction
			for _, a := range history {
				if a.Type == tc.actionType {
					last = a
				}
			}
			is.True(last != nil)
			is.Equal(last.ChatText, tc.oldValue)
			is.Equal(last.Note, "offensive")
			is.Equal(last.ApplierUserId, "Moderator")
			active, err := GetActions(ctx, us, "Hacker")
			is.NoErr(err)
			_, stillActive := active[tc.actionType.String()]
			is.True(!stillActive)
		})
	}
}

func TestRemoveProfileFieldsNeedAdmin(t *testing.T) {
	is := is.New(t)
	session := &entity.Session{
		ID:       "ghijkl",
		Username: "Moderator",
		UserUUID: "Moderator",
		Expiry:   time.Now().Add(time.Second * 100)}
	ctx := apiserver.PlaceInContext(context.Background(), session)

	recreateDB()
	us := userStore()
	defer us.(*user.DBStore).Disconnect()
	pool, err := common.OpenTestingDB(pkg)
	is.NoErr(err)
	defer pool.Close()
	q := models.New(pool)
	svc := &ModService{userStore: us, queries: q}

	removals := []ms.ModActionType{ms.ModActionType_REMOVE_FIRST_NAME,
		ms.ModActionType_REMOVE_LAST_NAME, ms.ModActionType_REMOVE_ABOUT,
		ms.ModActionType_REMOVE_AVATAR}

	// A moderator may remove a chat message but none of the profile fields.
	is.NoErr(q.AssignRole(ctx, models.AssignRoleParams{Username: "Moderator", RoleName: string(rbac.Moderator)}))
	_, err = authenticateMod(ctx, svc, &ms.ModActionsList{
		Actions: []*ms.ModAction{{UserId: "Hacker", Type: ms.ModActionType_REMOVE_CHAT}}})
	is.NoErr(err)
	for _, r := range removals {
		_, err = authenticateMod(ctx, svc, &ms.ModActionsList{
			Actions: []*ms.ModAction{{UserId: "Hacker", Type: r}}})
		is.Equal(err, apiserver.Unauthenticated(errNotAuthorized.Error()))
	}

	// An admin may.
	is.NoErr(q.AssignRole(ctx, models.AssignRoleParams{Username: "Moderator", RoleName: string(rbac.Admin)}))
	for _, r := range removals {
		_, err = authenticateMod(ctx, svc, &ms.ModActionsList{
			Actions: []*ms.ModAction{{UserId: "Hacker", Type: r}}})
		is.NoErr(err)
	}
}
