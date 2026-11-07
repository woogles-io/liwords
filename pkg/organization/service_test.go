package organization

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
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
	for _, u := range []string{"stays", "other"} {
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
