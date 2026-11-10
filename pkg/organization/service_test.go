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
		w.Write([]byte(`{"players": [{"id": "5", "name": "Promoted, Tony", "title": "GM"}, {"id": "7", "name": "Untitled, Uma", "title": ""}]}`))
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

	// a member listed with no title is linked, holding none
	is.NoErr(assign("7"))
	data = stored(t, q, "stays", organizations.OrgSG)
	is.Equal(data.MemberID, "7")
	is.Equal(data.FullName, "Untitled, Uma")
	is.Equal(data.RawTitle, "")
}

// abspServer serves an ABSP ratings list, and a login for evan@example.com,
// member 02333
func abspServer(t *testing.T, ratings *string, status *int) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/current.txt":
			w.WriteHeader(*status)
			w.Write([]byte(*ratings))
		case "/absp_log_on_or_off.php":
			if r.FormValue("username") == "evan@example.com" && r.FormValue("password") == "right" {
				http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "evan"})
				w.Write([]byte("<p style='color:green'>ABSP Member</p>"))
			}
		case "/edit_absp_member.php":
			if c, err := r.Cookie("PHPSESSID"); err == nil && c.Value == "evan" {
				w.Write([]byte("<td>absp_number:</td>\n<td>\n02333</td>\n" +
					"<td>first_name:</td>\n<td>\nEvan</td>\n<td>last_name:</td>\n<td>\nCohen</td>"))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("ABSP_BASE_URL", server.URL)
}

const abspRatings = "ELDAR(GM) DAVID 205 1 02279 \n" +
	"TEITELBAUM JAKOB 204 2  15\n" +
	"BALOGUN RASHEED 185 23 02524 111\n" +
	"COHEN(EXP) EVAN 183 28 02333 \n"

func TestConnectOrganizationABSP(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	pool, q := recreateDB(t)
	us, err := user.NewDBStore(pool)
	is.NoErr(err)
	ctx = apiserver.PlaceInContext(ctx, &entity.Session{
		ID: "stays-session", Username: "stays", UserUUID: "stays", Expiry: time.Now().Add(time.Minute)})

	ratings, status := abspRatings, http.StatusOK
	abspServer(t, &ratings, &status)

	s := NewOrganizationService(us, q, nil)
	connectABSP := func(memberID string, creds map[string]string) error {
		_, err := s.ConnectOrganization(ctx, connect.NewRequest(&pb.ConnectOrganizationRequest{
			OrganizationCode: string(organizations.OrgABSP),
			MemberId:         memberID,
			Credentials:      creds,
		}))
		return err
	}

	// a member number alone, with no login, proves nothing
	is.True(connectABSP("02279", nil) != nil)
	is.True(connectABSP("02279", map[string]string{}) != nil)
	is.True(connectABSP("", map[string]string{"username": "evan@example.com", "password": "wrong"}) != nil)
	_, err = q.GetIntegrationData(ctx, models.GetIntegrationDataParams{UserUuid: "stays", IntegrationName: "absp"})
	is.True(err != nil) // nothing linked

	// the login gives the number, and the list the title; the login is not kept
	is.NoErr(connectABSP("", map[string]string{"username": "evan@example.com", "password": "right"}))
	data := stored(t, q, "stays", organizations.OrgABSP)
	is.Equal(data.MemberID, "2333")
	is.Equal(data.FullName, "Evan Cohen")
	is.Equal(data.RawTitle, "EXP")
	is.Equal(data.EncryptedCredentials, "")
	is.True(data.Verified)

	title, org := profileTitle(t, pool, "stays")
	is.Equal(title, "Expert")
	is.Equal(org, "absp")
}

func TestManuallySetOrgMembershipABSP(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	pool, q := recreateDB(t)
	us, err := user.NewDBStore(pool)
	is.NoErr(err)
	is.NoErr(us.New(ctx, &entity.User{Username: "admin", Email: "admin@woogles.io", UUID: "admin"}))
	is.NoErr(q.AssignRole(ctx, models.AssignRoleParams{Username: "admin", RoleName: string(rbac.Admin)}))
	ctx = apiserver.PlaceInContext(ctx, &entity.Session{
		ID: "admin-session", Username: "admin", UserUUID: "admin", Expiry: time.Now().Add(time.Minute)})

	ratings, status := abspRatings, http.StatusOK
	abspServer(t, &ratings, &status)

	s := NewOrganizationService(us, q, nil)
	assign := func(memberID string, creds map[string]string) error {
		_, err := s.ManuallySetOrgMembership(ctx, connect.NewRequest(&pb.ManuallySetOrgMembershipRequest{
			Username:         "stays",
			OrganizationCode: string(organizations.OrgABSP),
			MemberId:         memberID,
			Credentials:      creds,
		}))
		return err
	}

	// stored as the list has it, however it was typed
	is.NoErr(assign("02279", nil))
	data := stored(t, q, "stays", organizations.OrgABSP)
	is.Equal(data.MemberID, "2279")
	is.Equal(data.FullName, "David Eldar")
	is.Equal(data.RawTitle, "GM")
	title, org := profileTitle(t, pool, "stays")
	is.Equal(title, "Grandmaster")
	is.Equal(org, "absp")

	// a number the list does not have as a member is refused: 15 is only a
	// nonmember's count of games
	is.True(assign("15", nil) != nil)
	is.True(assign("2279X", nil) != nil)

	// a member listed with no title is linked, holding none
	is.NoErr(assign("2524", nil))
	data = stored(t, q, "stays", organizations.OrgABSP)
	is.Equal(data.MemberID, "2524")
	is.Equal(data.FullName, "Rasheed Balogun")
	is.Equal(data.RawTitle, "")

	// with the member's login, the profile's number is used, and the login
	// is not kept
	is.NoErr(assign("2524", map[string]string{"username": "evan@example.com", "password": "right"}))
	data = stored(t, q, "stays", organizations.OrgABSP)
	is.Equal(data.MemberID, "2333")
	is.Equal(data.RawTitle, "EXP")
	is.Equal(data.EncryptedCredentials, "")
}

func TestRefreshOrganizationTitlesABSP(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	pool, q := recreateDB(t)

	ratings, status := abspRatings, http.StatusOK
	abspServer(t, &ratings, &status)

	link(t, q, "stays", organizations.OrgABSP, organizations.OrganizationIntegrationData{
		MemberID: "02279", FullName: "David Eldar", RawTitle: "GM", Verified: true})
	// a login kept from before is dropped
	link(t, q, "promoted", organizations.OrgABSP, organizations.OrganizationIntegrationData{
		MemberID: "2333", FullName: "Evan Cohen", RawTitle: "", Verified: true,
		EncryptedCredentials: "kept-from-before"})
	link(t, q, "lapsed", organizations.OrgABSP, organizations.OrganizationIntegrationData{
		MemberID: "1618", FullName: "Paul Gallen", RawTitle: "GM", Verified: true})
	link(t, q, "lapsed", organizations.OrgWESPA, organizations.OrganizationIntegrationData{
		MemberID: "123", FullName: "Paul Gallen", RawTitle: "M", Verified: true})

	is.NoErr(RefreshOrganizationTitles(ctx, q, organizations.OrgABSP))

	is.Equal(stored(t, q, "stays", organizations.OrgABSP).RawTitle, "GM")
	promoted := stored(t, q, "promoted", organizations.OrgABSP)
	is.Equal(promoted.RawTitle, "EXP")
	is.Equal(promoted.EncryptedCredentials, "")
	lapsed := stored(t, q, "lapsed", organizations.OrgABSP)
	is.Equal(lapsed.RawTitle, "")
	is.Equal(lapsed.FullName, "Paul Gallen") // the stored name is kept

	title, org := profileTitle(t, pool, "promoted")
	is.Equal(title, "Expert")
	is.Equal(org, "absp")
	title, org = profileTitle(t, pool, "lapsed")
	is.Equal(title, "Master")
	is.Equal(org, "wespa")

	// a failed download, or a broken list, changes no one's title
	for _, broken := range []struct {
		status  int
		ratings string
	}{
		{http.StatusInternalServerError, abspRatings},
		{http.StatusOK, "<html>down for maintenance</html>\n"},
		{http.StatusOK, ""},
	} {
		status, ratings = broken.status, broken.ratings
		is.True(RefreshOrganizationTitles(ctx, q, organizations.OrgABSP) != nil)
		is.Equal(stored(t, q, "stays", organizations.OrgABSP).RawTitle, "GM")
		is.Equal(stored(t, q, "promoted", organizations.OrgABSP).RawTitle, "EXP")
	}

	// a player's own refresh drops a kept login too
	status, ratings = http.StatusOK, abspRatings
	link(t, q, "promoted", organizations.OrgABSP, organizations.OrganizationIntegrationData{
		MemberID: "2333", FullName: "Evan Cohen", RawTitle: "", Verified: true,
		EncryptedCredentials: "kept-from-before"})
	s := &OrganizationService{queries: q}
	_, errs, err := s.refreshTitlesForUser(ctx, "promoted")
	is.NoErr(err)
	is.Equal(len(errs), 0)
	promoted = stored(t, q, "promoted", organizations.OrgABSP)
	is.Equal(promoted.RawTitle, "EXP")
	is.Equal(promoted.EncryptedCredentials, "")

	// NASPA's titles need a login, so it has no scheduled refresh
	is.True(RefreshOrganizationTitles(ctx, q, organizations.OrgNASPA) != nil)
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
		{http.StatusOK, `{"players": null}` + "\n"},
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

	// an empty list is still a list: it takes every SG title away
	status, list = http.StatusOK, `{"players": [`+"\n]}\n"
	is.NoErr(RefreshOrganizationTitles(ctx, q, organizations.OrgSG))
	is.Equal(stored(t, q, "stays", organizations.OrgSG).RawTitle, "")
	is.Equal(stored(t, q, "promoted", organizations.OrgSG).RawTitle, "")
	title, org = profileTitle(t, pool, "promoted")
	is.Equal(title, "")
	is.Equal(org, "")
	title, org = profileTitle(t, pool, "lapsed")
	is.Equal(title, "Master")
	is.Equal(org, "wespa")
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

func TestGetMyOrganizationsListsPendingRequests(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	pool, q := recreateDB(t)
	us, err := user.NewDBStore(pool)
	is.NoErr(err)
	s := NewOrganizationService(us, q, nil)

	request := func(userUUID string, orgCode organizations.OrganizationCode, memberID, name, title string) int64 {
		vr, err := q.CreateVerificationRequest(ctx, models.CreateVerificationRequestParams{
			UserUuid:        userUUID,
			IntegrationName: string(orgCode),
			MemberID:        memberID,
			FullName:        name,
			ImageUrl:        "verification/" + userUUID,
			Title:           pgtype.Text{String: title, Valid: title != ""},
		})
		is.NoErr(err)
		return vr.ID
	}
	mine := func(userUUID string) []*pb.OrganizationTitle {
		ctx := apiserver.PlaceInContext(ctx, &entity.Session{
			ID: userUUID + "-session", Username: userUUID, UserUUID: userUUID, Expiry: time.Now().Add(time.Minute)})
		resp, err := s.GetMyOrganizations(ctx, connect.NewRequest(&pb.GetMyOrganizationsRequest{}))
		is.NoErr(err)
		return resp.Msg.Titles
	}

	// a connection is listed as before, and a request still waiting for an
	// admin is listed beside it, not verified
	link(t, q, "stays", organizations.OrgWESPA, organizations.OrganizationIntegrationData{
		MemberID: "123", FullName: "Sarah Stays", RawTitle: "M", Verified: true})
	request("stays", organizations.OrgSG, "5", "Promoted, Tony", "GM")
	titles := mine("stays")
	is.Equal(len(titles), 2)
	is.Equal(titles[0].OrganizationCode, "wespa")
	is.True(titles[0].Verified)
	is.Equal(titles[1].OrganizationCode, "sg")
	is.Equal(titles[1].MemberId, "5")
	is.Equal(titles[1].FullName, "Promoted, Tony")
	is.Equal(titles[1].RawTitle, "GM")
	is.Equal(titles[1].NormalizedTitle, "GM")
	is.True(!titles[1].Verified)

	// a request no longer pending is not listed
	rejected := request("other", organizations.OrgSG, "6", "Other, One", "")
	_, err = pool.Exec(ctx, `UPDATE verification_requests SET status = 'rejected' WHERE id = $1`, rejected)
	is.NoErr(err)
	is.Equal(len(mine("other")), 0)

	// nor is one for an organization already connected
	link(t, q, "lapsed", organizations.OrgSG, organizations.OrganizationIntegrationData{
		MemberID: "7", FullName: "Lapsed, Lee", Verified: true})
	request("lapsed", organizations.OrgSG, "7", "Lapsed, Lee", "")
	titles = mine("lapsed")
	is.Equal(len(titles), 1)
	is.True(titles[0].Verified)
}
