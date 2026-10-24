package integrations

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

func TestHighestTier(t *testing.T) {
	is := is.New(t)
	pm := &PatreonMemberDatum{
		Relationships: PatreonRelationships{
			CurrentlyEntitledTiers: EntitledTiersRelationship{
				Data: []PatreonRelationship{
					{"22998862", "tier"}, {"24128408", "tier"},
				},
			},
		},
	}
	is.Equal(HighestTier(pm), TierGoldenRetriever)

	pm = &PatreonMemberDatum{}
	is.Equal(HighestTier(pm), TierNone)

	pm = &PatreonMemberDatum{
		Relationships: PatreonRelationships{
			CurrentlyEntitledTiers: EntitledTiersRelationship{
				Data: []PatreonRelationship{
					{"22998862", "tier"},
				},
			},
		},
	}
	is.Equal(HighestTier(pm), TierChihuahua)

	pm = &PatreonMemberDatum{
		Relationships: PatreonRelationships{
			CurrentlyEntitledTiers: EntitledTiersRelationship{
				Data: []PatreonRelationship{
					{"24128312", "tier"}, {"10805942", "tier"},
				},
			},
		},
	}
	is.Equal(HighestTier(pm), TierDalmatian)

}

func member(tierIDs []string, attrs PatreonMemberAttributes) *PatreonMemberDatum {
	m := &PatreonMemberDatum{Attributes: attrs}
	for _, id := range tierIDs {
		m.Relationships.CurrentlyEntitledTiers.Data = append(
			m.Relationships.CurrentlyEntitledTiers.Data, PatreonRelationship{id, "tier"})
	}
	return m
}

func TestTierDataFromMember(t *testing.T) {
	is := is.New(t)
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

	// Never a member.
	td, err := tierDataFromMember(nil, now)
	is.NoErr(err)
	is.True(td == nil)

	// Former patron: still in the member list, but no entitled tiers.
	_, err = tierDataFromMember(member(nil, PatreonMemberAttributes{
		PatronStatus: "former_patron", LastChargeDate: "2026-01-01T00:00:00.000+00:00"}), now)
	is.True(errors.Is(err, ErrNotSubscribed))

	_, err = tierDataFromMember(member([]string{"10805942"}, PatreonMemberAttributes{}), now)
	is.True(errors.Is(err, ErrNotPaidTier))

	// Paying member. Patreon sends fractional seconds and a +00:00 offset.
	td, err = tierDataFromMember(member([]string{"24128312"}, PatreonMemberAttributes{
		LastChargeDate:          "2026-10-09T00:33:06.000+00:00",
		LastChargeStatus:        "Paid",
		PledgeRelationshipStart: "2026-06-08T13:09:09.179+00:00",
	}), now)
	is.NoErr(err)
	is.Equal(td.Tier, TierDalmatian)
	is.Equal(td.LastChargeStatus, "Paid")
	is.True(td.LastChargeDate.Equal(time.Date(2026, 10, 9, 0, 33, 6, 0, time.UTC)))

	// Gift recipient: never charged, so the period runs from the latest
	// monthly anniversary of the gift.
	td, err = tierDataFromMember(member([]string{"24128312"}, PatreonMemberAttributes{
		IsGifted:                true,
		PatronStatus:            "active_patron",
		PledgeRelationshipStart: "2025-11-09T09:44:06.213+00:00",
	}), now)
	is.NoErr(err)
	is.Equal(td.Tier, TierDalmatian)
	is.True(td.LastChargeDate.Equal(time.Date(2026, 10, 9, 9, 44, 6, 213000000, time.UTC)))

	// Neither date: an error rather than a zero period start, which would
	// count every BestBot game the user ever played.
	_, err = tierDataFromMember(member([]string{"24128312"}, PatreonMemberAttributes{IsGifted: true}), now)
	is.True(err != nil)
}

func TestLatestMonthlyAnniversary(t *testing.T) {
	is := is.New(t)
	d := func(y int, m time.Month, day, h int) time.Time { return time.Date(y, m, day, h, 0, 0, 0, time.UTC) }

	start := d(2025, 11, 9, 9)
	is.Equal(latestMonthlyAnniversary(start, d(2026, 10, 23, 0)), d(2026, 10, 9, 9))
	// Same day, before the gift's time of day: still last month's period.
	is.Equal(latestMonthlyAnniversary(start, d(2026, 10, 9, 8)), d(2026, 9, 9, 9))
	is.Equal(latestMonthlyAnniversary(start, d(2026, 10, 9, 9)), d(2026, 10, 9, 9))
	// Within the first month, and a start in the future.
	is.Equal(latestMonthlyAnniversary(start, d(2025, 11, 20, 0)), start)
	is.Equal(latestMonthlyAnniversary(start, d(2025, 10, 1, 0)), start)

	// End-of-month starts clamp instead of spilling into the next month.
	jan31 := d(2026, 1, 31, 12)
	is.Equal(latestMonthlyAnniversary(jan31, d(2026, 3, 1, 0)), d(2026, 2, 28, 12))
	is.Equal(latestMonthlyAnniversary(jan31, d(2026, 3, 31, 12)), d(2026, 3, 31, 12))
	is.Equal(latestMonthlyAnniversary(d(2024, 1, 31, 0), d(2024, 2, 29, 1)), d(2024, 2, 29, 0))
	// Across a year boundary.
	is.Equal(latestMonthlyAnniversary(d(2025, 12, 15, 0), d(2026, 1, 20, 0)), d(2026, 1, 15, 0))
}

func TestGetCampaignSubscribersPaginates(t *testing.T) {
	is := is.New(t)
	var srvURL string
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		is.Equal(r.Header.Get("Authorization"), "Bearer creator-token")
		is.Equal(r.URL.Path, fmt.Sprintf("/api/oauth2/v2/campaigns/%d/members", CampaignID))
		if r.URL.Query().Get("page[cursor]") == "" {
			q := r.URL.Query()
			is.Equal(q.Get("include"), "currently_entitled_tiers,user")
			is.True(strings.Contains(q.Get("fields[member]"), "last_charge_date"))
			is.True(strings.Contains(q.Get("fields[member]"), "pledge_relationship_start"))
			is.Equal(q.Get("page[count]"), "1000")
			fmt.Fprintf(w, `{"data":[{"id":"m1","type":"member","attributes":{"last_charge_date":"2026-10-09T00:33:06.000+00:00"},
				"relationships":{"user":{"data":{"id":"u1","type":"user"}},"currently_entitled_tiers":{"data":[{"id":"24128312","type":"tier"}]}}}],
				"links":{"next":"%s/api/oauth2/v2/campaigns/%d/members?page%%5Bcursor%%5D=abc"}}`, srvURL, CampaignID)
			return
		}
		fmt.Fprint(w, `{"data":[{"id":"m2","type":"member","attributes":{"is_gifted":true,"pledge_relationship_start":"2025-11-09T09:44:06.213+00:00"},
			"relationships":{"user":{"data":{"id":"u2","type":"user"}},"currently_entitled_tiers":{"data":[]}}}]}`)
	}))
	defer srv.Close()
	srvURL = srv.URL
	defer func(base string) { patreonAPIBase = base }(patreonAPIBase)
	patreonAPIBase = srv.URL

	members, err := GetCampaignSubscribers(context.Background(), "creator-token")
	is.NoErr(err)
	is.Equal(requests, 2)
	is.Equal(len(members.Data), 2)
	is.Equal(members.Data[0].Relationships.User.Data.ID, "u1")
	is.Equal(members.Data[0].Attributes.LastChargeDate, "2026-10-09T00:33:06.000+00:00")
	is.Equal(members.Data[1].Relationships.User.Data.ID, "u2")
	is.True(members.Data[1].Attributes.IsGifted)

	// Errors surface as PatreonAPIError.
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[]}`, http.StatusUnauthorized)
	})
	_, err = GetCampaignSubscribers(context.Background(), "creator-token")
	var papierr *PatreonAPIError
	is.True(errors.As(err, &papierr))
}

func TestLookupCampaignMemberCache(t *testing.T) {
	is := is.New(t)
	resetCache := func() {
		memberListCache.byUserID = nil
		memberListCache.fetchedAt = time.Time{}
	}
	resetCache()
	defer resetCache()
	defer func(fn func(context.Context, *models.Queries) (map[string]*PatreonMemberDatum, error)) {
		fetchCampaignMembersFn = fn
	}(fetchCampaignMembersFn)

	fetches := 0
	var fetchErr error
	list := map[string]*PatreonMemberDatum{"u1": member([]string{"24128312"}, PatreonMemberAttributes{})}
	fetchCampaignMembersFn = func(context.Context, *models.Queries) (map[string]*PatreonMemberDatum, error) {
		fetches++
		if fetchErr != nil {
			return nil, fetchErr
		}
		return list, nil
	}
	ctx := context.Background()

	m, err := lookupCampaignMember(ctx, nil, "u1")
	is.NoErr(err)
	is.True(m != nil)
	is.Equal(fetches, 1)

	// Cached for members found and, briefly, for users not in the list.
	_, _ = lookupCampaignMember(ctx, nil, "u1")
	m, err = lookupCampaignMember(ctx, nil, "nobody")
	is.NoErr(err)
	is.True(m == nil)
	is.Equal(fetches, 1)

	// A new subscriber is picked up once the miss window passes.
	list = map[string]*PatreonMemberDatum{
		"u1":     list["u1"],
		"newsub": member([]string{"22998862"}, PatreonMemberAttributes{}),
	}
	memberListCache.fetchedAt = time.Now().Add(-memberListMinRefetch - time.Second)
	m, err = lookupCampaignMember(ctx, nil, "newsub")
	is.NoErr(err)
	is.True(m != nil)
	is.Equal(fetches, 2)

	// A former patron who resubscribes is picked up after the short window
	// too, not the full TTL.
	list["former"] = member(nil, PatreonMemberAttributes{PatronStatus: "former_patron"})
	memberListCache.fetchedAt = time.Now().Add(-memberListMinRefetch - time.Second)
	_, _ = lookupCampaignMember(ctx, nil, "former")
	is.Equal(fetches, 3)
	list = map[string]*PatreonMemberDatum{
		"u1":     list["u1"],
		"former": member([]string{"24128408"}, PatreonMemberAttributes{PatronStatus: "active_patron"}),
	}
	memberListCache.fetchedAt = time.Now().Add(-memberListMinRefetch - time.Second)
	m, err = lookupCampaignMember(ctx, nil, "former")
	is.NoErr(err)
	is.Equal(HighestTier(m), TierGoldenRetriever)
	is.Equal(fetches, 4)
	// ...while a current member's entry is still trusted at that age.
	memberListCache.fetchedAt = time.Now().Add(-memberListMinRefetch - time.Second)
	_, _ = lookupCampaignMember(ctx, nil, "u1")
	is.Equal(fetches, 4)

	// Known members are refetched after the TTL. If Patreon is down, the
	// stale list keeps paying users playing.
	fetchErr = errors.New("patreon down")
	memberListCache.fetchedAt = time.Now().Add(-memberListTTL - time.Second)
	m, err = lookupCampaignMember(ctx, nil, "u1")
	is.NoErr(err)
	is.True(m != nil)
	is.Equal(fetches, 5)

	// With no list at all, the failure is reported as ours, not the user's.
	resetCache()
	_, err = lookupCampaignMember(ctx, nil, "u1")
	is.True(errors.Is(err, ErrMembershipCheckUnavailable))
	var papierr *PatreonAPIError
	is.True(!errors.As(err, &papierr))
}
