package organization

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/integrations/organizations"
	commondb "github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
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
