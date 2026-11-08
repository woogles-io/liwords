package organization

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/integrations/organizations"
	commondb "github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

var pkg = "organization"

func recreateDB(t *testing.T) (*pgxpool.Pool, *models.Queries) {
	if err := commondb.RecreateTestDB(pkg); err != nil {
		t.Fatal(err)
	}
	pool, err := commondb.OpenTestingDB(pkg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	userStore, err := user.NewDBStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []string{"stays", "promoted", "lapsed", "other"} {
		err := userStore.New(context.Background(), &entity.User{Username: u, Email: u + "@woogles.io", UUID: u})
		if err != nil {
			t.Fatal(err)
		}
	}
	return pool, models.New(pool)
}

func link(t *testing.T, q *models.Queries, userUUID string, orgCode organizations.OrganizationCode, data organizations.OrganizationIntegrationData) {
	dataJSON, err := data.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	_, err = q.AddOrUpdateIntegration(context.Background(), models.AddOrUpdateIntegrationParams{
		UserUuid:        userUUID,
		IntegrationName: string(orgCode),
		Data:            dataJSON,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestOrganizationIntegrations(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	_, q := recreateDB(t)

	link(t, q, "stays", organizations.OrgWESPA, organizations.OrganizationIntegrationData{
		MemberID: "123", FullName: "Sarah Stays", RawTitle: "M", Verified: true})
	link(t, q, "stays", organizations.OrgNASPA, organizations.OrganizationIntegrationData{
		MemberID: "AA000083", FullName: "Sarah Stays", RawTitle: "EX", Verified: true})
	// integrations that are not organizations are left out
	link(t, q, "stays", "patreon", organizations.OrganizationIntegrationData{RawTitle: "GM", Verified: true})
	link(t, q, "other", "twitch", organizations.OrganizationIntegrationData{RawTitle: "GM", Verified: true})

	s := &OrganizationService{queries: q}
	integs, err := s.organizationIntegrations(ctx, "stays")
	is.NoErr(err)
	names := map[string]bool{}
	for _, integ := range integs {
		names[integ.IntegrationName] = true
	}
	is.Equal(names, map[string]bool{"wespa": true, "naspa": true})

	integs, err = s.organizationIntegrations(ctx, "other")
	is.NoErr(err)
	is.Equal(len(integs), 0)
}

func stored(t *testing.T, q *models.Queries, userUUID string, orgCode organizations.OrganizationCode) organizations.OrganizationIntegrationData {
	row, err := q.GetIntegrationData(context.Background(), models.GetIntegrationDataParams{
		UserUuid:        userUUID,
		IntegrationName: string(orgCode),
	})
	if err != nil {
		t.Fatal(err)
	}
	var data organizations.OrganizationIntegrationData
	if err := data.FromJSON(row.Data); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestManuallySetOrgMembershipSG(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	pool, q := recreateDB(t)
	us, err := user.NewDBStore(pool)
	is.NoErr(err)
	is.NoErr(us.New(ctx, &entity.User{Username: "admin", Email: "admin@woogles.io", UUID: "admin"}))
	is.NoErr(q.AssignRole(ctx, models.AssignRoleParams{Username: "admin", RoleName: string(rbac.Admin)}))
	ctx = apiserver.PlaceInContext(ctx, &entity.Session{
		ID: "admin-session", Username: "admin", UserUUID: "admin", Expiry: time.Now().Add(time.Minute)})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"players": [{"id": "5", "name": "Promoted, Tony", "title": "GM"}]}`))
	}))
	defer server.Close()
	t.Setenv("SG_TITLES_URL", server.URL)

	s := NewOrganizationService(us, q, nil)
	assign := func(memberID string) error {
		_, err := s.ManuallySetOrgMembership(ctx, connect.NewRequest(&pb.ManuallySetOrgMembershipRequest{
			Username:         "stays",
			OrganizationCode: string(organizations.OrgSG),
			MemberId:         memberID,
		}))
		return err
	}

	// stored as the list has it, however it was typed
	is.NoErr(assign("0005"))
	data := stored(t, q, "stays", organizations.OrgSG)
	is.Equal(data.MemberID, "5")
	is.Equal(data.FullName, "Promoted, Tony")
	is.Equal(data.RawTitle, "GM")

	// a member the list does not have, and a number with its type, are refused
	is.True(assign("6") != nil)
	is.True(assign("0005M") != nil)
}

func profileTitle(t *testing.T, pool *pgxpool.Pool, userUUID string) (string, string) {
	var title, org *string
	err := pool.QueryRow(context.Background(),
		`SELECT title, title_organization FROM profiles WHERE user_id = (SELECT id FROM users WHERE uuid = $1)`,
		userUUID).Scan(&title, &org)
	if err != nil {
		t.Fatal(err)
	}
	if title == nil || org == nil {
		return "", ""
	}
	return *title, *org
}

func TestRefreshOrganizationTitles(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	pool, q := recreateDB(t)

	list := `{"players": [
  {"id": "4", "name": "Stays, Sarah", "title": "SM"}
, {"id": "5", "name": "Promoted, Tony", "title": "GM"}
]}
`
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(list))
	}))
	defer server.Close()
	t.Setenv("SG_TITLES_URL", server.URL)

	link(t, q, "stays", organizations.OrgSG, organizations.OrganizationIntegrationData{
		MemberID: "0004", FullName: "Stays, Sarah", RawTitle: "SM", Verified: true})
	link(t, q, "promoted", organizations.OrgSG, organizations.OrganizationIntegrationData{
		MemberID: "5", FullName: "Promoted, Tony", RawTitle: "NM", Verified: true})
	link(t, q, "lapsed", organizations.OrgSG, organizations.OrganizationIntegrationData{
		MemberID: "6", FullName: "Lapsed, Lee", RawTitle: "M", Verified: true})
	// a WESPA title stays untouched, and shows once the SG title is gone
	link(t, q, "lapsed", organizations.OrgWESPA, organizations.OrganizationIntegrationData{
		MemberID: "123", FullName: "Lee Lapsed", RawTitle: "M", Verified: true})

	is.NoErr(RefreshOrganizationTitles(ctx, q, organizations.OrgSG))

	is.Equal(stored(t, q, "stays", organizations.OrgSG).RawTitle, "SM")
	is.Equal(stored(t, q, "promoted", organizations.OrgSG).RawTitle, "GM")
	lapsed := stored(t, q, "lapsed", organizations.OrgSG)
	is.Equal(lapsed.RawTitle, "")
	is.Equal(lapsed.FullName, "Lapsed, Lee") // the stored name is kept
	is.Equal(stored(t, q, "lapsed", organizations.OrgWESPA).RawTitle, "M")

	title, org := profileTitle(t, pool, "promoted")
	is.Equal(title, "Grandmaster")
	is.Equal(org, "sg")
	title, org = profileTitle(t, pool, "stays")
	is.Equal(title, "Senior Master")
	is.Equal(org, "sg")
	title, org = profileTitle(t, pool, "lapsed")
	is.Equal(title, "Master")
	is.Equal(org, "wespa")

	// a failed download, or a broken list, changes no one's title
	for _, broken := range []struct {
		status int
		list   string
	}{
		{http.StatusInternalServerError, list},
		{http.StatusOK, `{"players": [` + "\n]}\n"},
		{http.StatusOK, `{"players": [{"id": "5", "name": "Promoted, Tony", "title": "GM"}`},
	} {
		status, list = broken.status, broken.list
		is.True(RefreshOrganizationTitles(ctx, q, organizations.OrgSG) != nil)
		is.Equal(stored(t, q, "stays", organizations.OrgSG).RawTitle, "SM")
		is.Equal(stored(t, q, "promoted", organizations.OrgSG).RawTitle, "GM")
		title, org = profileTitle(t, pool, "promoted")
		is.Equal(title, "Grandmaster")
		is.Equal(org, "sg")
	}
}

func setProfileTitle(t *testing.T, q *models.Queries, userUUID, title, org string) {
	err := q.UpdateProfileTitle(context.Background(), models.UpdateProfileTitleParams{
		UserUuid:          userUUID,
		Title:             pgtype.Text{String: title, Valid: true},
		TitleOrganization: pgtype.Text{String: org, Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRepickProfileTitles(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	pool, q := recreateDB(t)

	// a NASPA SM and a WESPA M tie, and the old pick showed NASPA's
	link(t, q, "stays", organizations.OrgNASPA, organizations.OrganizationIntegrationData{
		MemberID: "AA000083", FullName: "Sarah Stays", RawTitle: "SM", Verified: true})
	link(t, q, "stays", organizations.OrgWESPA, organizations.OrganizationIntegrationData{
		MemberID: "123", FullName: "Sarah Stays", RawTitle: "M", Verified: true})
	setProfileTitle(t, q, "stays", "NASPA Master", "naspa")
	// one organization, with a stale pick: picked again too
	link(t, q, "other", organizations.OrgWESPA, organizations.OrganizationIntegrationData{
		MemberID: "456", FullName: "Otto Other", RawTitle: "M", Verified: true})
	setProfileTitle(t, q, "other", "Expert", "naspa")
	// patreon only, which is not an organization: left alone
	link(t, q, "promoted", "patreon", organizations.OrganizationIntegrationData{RawTitle: "GM", Verified: true})
	setProfileTitle(t, q, "promoted", "Expert", "naspa")

	is.NoErr(RepickProfileTitles(ctx, q))

	title, org := profileTitle(t, pool, "stays")
	is.Equal(title, "Master")
	is.Equal(org, "wespa")
	title, org = profileTitle(t, pool, "other")
	is.Equal(title, "Master")
	is.Equal(org, "wespa")
	title, org = profileTitle(t, pool, "promoted")
	is.Equal(title, "Expert")
	is.Equal(org, "naspa")

	// picked again with nothing changed: no row is written (same xmin)
	rowVersion := func(userUUID string) string {
		var xmin string
		err := pool.QueryRow(ctx,
			`SELECT xmin::text FROM profiles WHERE user_id = (SELECT id FROM users WHERE uuid = $1)`,
			userUUID).Scan(&xmin)
		if err != nil {
			t.Fatal(err)
		}
		return xmin
	}
	before := []string{rowVersion("stays"), rowVersion("other")}
	is.NoErr(RepickProfileTitles(ctx, q))
	is.Equal([]string{rowVersion("stays"), rowVersion("other")}, before)
}

func TestOrganizationIntegrationsInTieOrder(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	_, q := recreateDB(t)

	for _, org := range []organizations.OrganizationCode{
		organizations.OrgABSP, organizations.OrgSG, organizations.OrgNASPA, organizations.OrgWESPA,
	} {
		link(t, q, "stays", org, organizations.OrganizationIntegrationData{MemberID: "1", Verified: true})
	}

	s := &OrganizationService{queries: q}
	integs, err := s.organizationIntegrations(ctx, "stays")
	is.NoErr(err)
	var names []string
	for _, integ := range integs {
		names = append(names, integ.IntegrationName)
	}
	is.Equal(names, []string{"wespa", "naspa", "sg", "absp"})
}
