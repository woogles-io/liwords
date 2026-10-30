package integrations

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

//go:generate stringer -type=Tier -linecomment
type Tier int

// Comments below are the user-readable names of these tiers. See
// -linecomment flag above.
const (
	TierNone            Tier = iota
	TierFree                 // Free
	TierChihuahua            // Chihuahua
	TierDalmatian            // Dalmatian
	TierGoldenRetriever      // Golden Retriever
)

// This is specific to Woogles.io Patreon tier data.
var PatreonTierIDs = map[string]Tier{
	"10805942": TierFree,
	"22998862": TierChihuahua,
	"24128312": TierDalmatian,
	"24128408": TierGoldenRetriever,
}

// CampaignID is a hard-coded id for our specific Woogles.io Patreon campaign.
var CampaignID = 6109248

var ErrNotSubscribed = errors.New("user not subscribed")
var ErrNotPaidTier = errors.New("user not subscribed on paid tier")

// ErrMembershipCheckUnavailable means we could not fetch the campaign's
// member list with the creator token. It is our problem, not the user's.
var ErrMembershipCheckUnavailable = errors.New("unable to check patreon membership")

const (
	ChargeStatusPaid = "Paid"
)

// patreonAPIBase is a var so tests can point it at a fake server.
var patreonAPIBase = "https://www.patreon.com"

var patreonHTTPClient = &http.Client{Timeout: 15 * time.Second}

// Use a simple LRU cache to avoid hammering Patreon API
// in case people refresh etc.
var PatreonAPICache *expirable.LRU[string, *PaidTierData]

func init() {
	PatreonAPICache = expirable.NewLRU[string, *PaidTierData](0, nil, time.Second*60)
}

const (
	// How long a fetched member list is trusted for users with a current tier.
	memberListTTL = 5 * time.Minute
	// Anyone else may have just subscribed, so we refetch for them, but no
	// more often than this.
	memberListMinRefetch = 30 * time.Second
)

// memberListCache holds the campaign's member list, keyed by Patreon user ID.
// Fetching it is a single request with the creator token, so we don't need
// any per-user tokens to check a user's tier.
var memberListCache struct {
	sync.Mutex
	byUserID  map[string]*PatreonMemberDatum
	fetchedAt time.Time
}

type PatreonUserData struct {
	Data struct {
		Attributes struct {
			Email     string `json:"email"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"attributes"`
		ID            string `json:"id"`
		Relationships struct {
			Memberships struct {
				Data []struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"data"`
			} `json:"memberships"`
		} `json:"relationships"`
		Type string `json:"type"`
	} `json:"data"`
}

type PatreonMemberAttributes struct {
	Email            string `json:"email"`
	FullName         string `json:"full_name"`
	IsFollower       bool   `json:"is_follower"`
	IsGifted         bool   `json:"is_gifted"`
	LastChargeDate   string `json:"last_charge_date"`
	LastChargeStatus string `json:"last_charge_status"`
	PatronStatus     string `json:"patron_status"`
	// Gift memberships are never charged, so this is the only date they have.
	PledgeRelationshipStart string `json:"pledge_relationship_start"`
}

type PatreonRelationship struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type EntitledTiersRelationship struct {
	Data []PatreonRelationship `json:"data"`
}

type UserRelationship struct {
	Data PatreonRelationship `json:"data"`
}

type PatreonRelationships struct {
	CurrentlyEntitledTiers EntitledTiersRelationship `json:"currently_entitled_tiers"`
	User                   UserRelationship          `json:"user"`
}

type PatreonMemberDatum struct {
	Attributes    PatreonMemberAttributes `json:"attributes"`
	ID            string                  `json:"id"`
	Relationships PatreonRelationships    `json:"relationships"`
	Type          string                  `json:"type"`
}

type MultiplePatreonMemberData struct {
	Data  []PatreonMemberDatum `json:"data"`
	Links struct {
		Next string `json:"next"`
	} `json:"links"`
}

type PaidTierData struct {
	LastChargeDate   time.Time
	LastChargeStatus string
	Tier             Tier
}

type PatreonTokenResponse struct {
	AccessToken   string `json:"access_token"`
	RefreshToken  string `json:"refresh_token"`
	ExpiresIn     int    `json:"expires_in"`
	Scope         string `json:"scope"`
	PatreonUserID string `json:"patreon_user_id,omitempty"` // Don't overwrite with blank ID on refresh.
}

type PatreonAPIError struct {
	code     int
	errorMsg string
}

func (p *PatreonAPIError) Error() string {
	return fmt.Sprintf("patreon error: %s (code %d)", p.errorMsg, p.code)
}

func (s *OAuthIntegrationService) patreonCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	// Get the state parameter
	stateParam := r.URL.Query().Get("state")
	if stateParam == "" {
		http.Error(w, "State parameter is missing", http.StatusBadRequest)
		return
	}

	// Decode the state parameter (Base64 decode first, then JSON decode)
	stateBytes, err := base64.StdEncoding.DecodeString(stateParam)
	if err != nil {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	var state OAuthState
	err = json.Unmarshal(stateBytes, &state)
	if err != nil {
		http.Error(w, "Invalid state JSON", http.StatusBadRequest)
		return
	}
	log.Debug().Bytes("stateBytes", stateBytes).Msg("state-bytes")
	if state.CSRF != sess.CSRFToken {
		log.Debug().Str("state-csrf", state.CSRF).Str("sess-csrf", sess.CSRFToken).Msg("bad-token")
		http.Error(w, "Invalid CSRF token", http.StatusUnauthorized)
		return
	}

	// Get the authorization code from the query parameters
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code not found", http.StatusBadRequest)
		return
	}

	// Exchange the authorization code for an access token
	token, err := s.exchangePatreonCodeForToken(code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Token exchange failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Fetch the user's Patreon data
	userData, err := fetchPatreonUserData(token.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch user data: %v", err), http.StatusInternalServerError)
		return
	}

	log.Info().Interface("ud", userData).Msg("userData")
	token.PatreonUserID = userData.Data.ID
	// re-dump token data for saving into table
	td, err := json.Marshal(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, err := s.queries.AddOrUpdateIntegration(ctx, models.AddOrUpdateIntegrationParams{
		UserUuid:        sess.UserUUID,
		IntegrationName: PatreonIntegrationName,
		Data:            td,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Debug().Str("uuid", id.String()).Msg("integration-table-updated")
	// Redirect the user back to the original page
	http.Redirect(w, r, state.RedirectTo, http.StatusFound)
}

func (s *OAuthIntegrationService) exchangePatreonCodeForToken(code string) (*PatreonTokenResponse, error) {
	tokenURL := patreonAPIBase + "/api/oauth2/token"

	data := url.Values{}
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", s.cfg.PatreonClientID)
	data.Set("client_secret", s.cfg.PatreonClientSecret)
	data.Set("redirect_uri", s.cfg.PatreonRedirectURI)

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp PatreonTokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return nil, err
	}
	log.Debug().Interface("tr", tokenResp).Msg("tokenResponse")

	return &tokenResp, nil
}

func (s *OAuthIntegrationService) RefreshPatreonToken(refreshToken string) (*PatreonTokenResponse, error) {
	tokenURL := patreonAPIBase + "/api/oauth2/token"

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", s.cfg.PatreonClientID)
	data.Set("client_secret", s.cfg.PatreonClientSecret)
	data.Set("refresh_token", refreshToken)

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	var tokenResp PatreonTokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return nil, err
	}
	log.Debug().Interface("tr", tokenResp).Msg("refresh-token-response")

	return &tokenResp, nil
}

// FetchPatreonUserID returns the Patreon user ID that owns accessToken.
func FetchPatreonUserID(accessToken string) (string, error) {
	userData, err := fetchPatreonUserData(accessToken)
	if err != nil {
		return "", err
	}
	if userData.Data.ID == "" {
		return "", errors.New("patreon identity response has no user id")
	}
	return userData.Data.ID, nil
}

func fetchPatreonUserData(accessToken string) (*PatreonUserData, error) {
	apiURL := patreonAPIBase + "/api/oauth2/v2/identity"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	// Set the Authorization header
	req.Header.Set("Authorization", "Bearer "+accessToken)

	query := req.URL.Query() // Get a copy of the query parameters
	query.Set("fields[user]", "email,first_name,last_name")
	query.Set("include", "memberships")
	req.URL.RawQuery = query.Encode()

	resp, err := patreonHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &PatreonAPIError{resp.StatusCode, string(bts)}
	}

	var userResp PatreonUserData
	err = json.Unmarshal(bts, &userResp)
	if err != nil {
		return nil, err
	}
	return &userResp, nil
}

// GetCampaignSubscribers fetches every member of our campaign (current and
// former) with the creator token, along with the attributes needed to work
// out their tier and billing period.
func GetCampaignSubscribers(ctx context.Context, globalToken string) (*MultiplePatreonMemberData, error) {
	apiURL := fmt.Sprintf(
		"%s/api/oauth2/v2/campaigns/%d/members", patreonAPIBase, CampaignID)

	paginating := true

	members := &MultiplePatreonMemberData{}
	page1 := true
	for paginating {

		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			return nil, err
		}
		// Set the Authorization header
		req.Header.Set("Authorization", "Bearer "+globalToken)
		if page1 {
			// The next links carry these params along, so only set them once.
			query := req.URL.Query() // Get a copy of the query parameters
			query.Set("include", "currently_entitled_tiers,user")
			query.Set("fields[member]", "last_charge_date,last_charge_status,patron_status,is_gifted,pledge_relationship_start")
			// The default page size is 20; this fits the whole campaign in one request.
			query.Set("page[count]", "1000")
			req.URL.RawQuery = query.Encode()
		}
		page1 = false

		resp, err := patreonHTTPClient.Do(req)
		if err != nil {
			return nil, err
		}

		bts, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, &PatreonAPIError{resp.StatusCode, string(bts)}
		}
		thispage := &MultiplePatreonMemberData{}
		err = json.Unmarshal(bts, thispage)
		if err != nil {
			return nil, err
		}
		members.Data = append(members.Data, thispage.Data...)
		if thispage.Links.Next != "" {
			apiURL = thispage.Links.Next
			log.Info().Str("nextURL", apiURL).Msg("paginating...")
			time.Sleep(2 * time.Second)
		} else {
			paginating = false
			log.Info().Msg("pagination ending")
		}
	}
	return members, nil
}

// lookupCampaignMember finds a Patreon user in our campaign's member list.
// It returns nil if they have never been a member.
func lookupCampaignMember(ctx context.Context, queries *models.Queries, patreonUserID string) (*PatreonMemberDatum, error) {
	memberListCache.Lock()
	defer memberListCache.Unlock()

	age := time.Since(memberListCache.fetchedAt)
	if memberListCache.byUserID != nil {
		// Users without a current tier may have just (re)subscribed, so they
		// get the shorter window.
		m := memberListCache.byUserID[patreonUserID]
		entitled := m != nil && len(m.Relationships.CurrentlyEntitledTiers.Data) > 0
		if (entitled && age < memberListTTL) || age < memberListMinRefetch {
			return m, nil
		}
	}

	byUserID, err := fetchCampaignMembersFn(ctx, queries)
	if err != nil {
		if memberListCache.byUserID == nil {
			log.Err(err).Msg("patreon-member-list-unavailable")
			return nil, fmt.Errorf("%w: %v", ErrMembershipCheckUnavailable, err)
		}
		// A stale list is better than locking paying users out.
		log.Err(err).Dur("age", age).Msg("patreon-member-list-refetch-failed-using-stale")
		return memberListCache.byUserID[patreonUserID], nil
	}
	memberListCache.byUserID = byUserID
	memberListCache.fetchedAt = time.Now()
	return byUserID[patreonUserID], nil
}

// fetchCampaignMembersFn is a var so tests can stub out the creator token lookup.
var fetchCampaignMembersFn = fetchCampaignMembers

func fetchCampaignMembers(ctx context.Context, queries *models.Queries) (map[string]*PatreonMemberDatum, error) {
	bts, err := queries.GetGlobalIntegrationData(ctx, PatreonIntegrationName)
	if err != nil {
		return nil, err
	}
	var ptoken PatreonTokenResponse
	err = json.Unmarshal(bts, &ptoken)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal global integration data into token: %w", err)
	}
	members, err := GetCampaignSubscribers(ctx, ptoken.AccessToken)
	if err != nil {
		return nil, err
	}
	byUserID := make(map[string]*PatreonMemberDatum, len(members.Data))
	for i := range members.Data {
		m := &members.Data[i]
		byUserID[m.Relationships.User.Data.ID] = m
	}
	log.Info().Int("members", len(byUserID)).Msg("fetched-patreon-member-list")
	return byUserID, nil
}

// tierDataFromMember works out a member's paid tier and the start of their
// current billing period. A nil member means they have never been a member.
func tierDataFromMember(member *PatreonMemberDatum, now time.Time) (*PaidTierData, error) {
	if member == nil {
		return nil, nil
	}
	if len(member.Relationships.CurrentlyEntitledTiers.Data) == 0 {
		return nil, ErrNotSubscribed
	}
	tier := HighestTier(member)
	if tier == TierFree {
		return nil, ErrNotPaidTier
	}
	attrs := member.Attributes
	if attrs.LastChargeDate != "" {
		lastChargeDate, err := time.Parse(time.RFC3339, attrs.LastChargeDate)
		if err != nil {
			return nil, err
		}
		return &PaidTierData{
			Tier:             tier,
			LastChargeStatus: attrs.LastChargeStatus,
			LastChargeDate:   lastChargeDate,
		}, nil
	}
	// Gift recipients are entitled to a tier but are never charged. Treat
	// each monthly anniversary of the gift as the start of a billing period.
	start, err := time.Parse(time.RFC3339, attrs.PledgeRelationshipStart)
	if err != nil {
		return nil, fmt.Errorf("member has no last charge date or pledge start: %w", err)
	}
	return &PaidTierData{
		Tier:           tier,
		LastChargeDate: latestMonthlyAnniversary(start, now),
	}, nil
}

// latestMonthlyAnniversary returns the latest start + N months that is not
// after now (or start itself, if now is before start).
func latestMonthlyAnniversary(start, now time.Time) time.Time {
	months := (now.Year()-start.Year())*12 + int(now.Month()-start.Month())
	for ; months > 0; months-- {
		if a := addMonthsClamped(start, months); !a.After(now) {
			return a
		}
	}
	return start
}

// addMonthsClamped is like t.AddDate(0, months, 0), except a day that doesn't
// exist in the target month becomes that month's last day (Jan 31 -> Feb 28)
// instead of spilling into the next month.
func addMonthsClamped(t time.Time, months int) time.Time {
	firstOfMonth := time.Date(t.Year(), t.Month()+time.Month(months), 1,
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	lastDay := firstOfMonth.AddDate(0, 1, -1).Day()
	return firstOfMonth.AddDate(0, 0, min(t.Day(), lastDay)-1)
}

// DetermineUserTier determines what tier the user is in.
func DetermineUserTier(ctx context.Context, userID string, queries *models.Queries) (*PaidTierData, error) {
	// This function uses the cache.
	res, ok := PatreonAPICache.Get(userID)
	if ok {
		log.Info().Str("userID", userID).Msg("found-cached-tier-data")
		return res, nil
	}

	row, err := queries.GetIntegrationData(ctx, models.GetIntegrationDataParams{
		IntegrationName: PatreonIntegrationName,
		UserUuid:        userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Not an error
			log.Info().Str("userID", userID).Msg("no-patreon-integration")
			PatreonAPICache.Add(userID, nil)
			return nil, nil
		}
		return nil, err
	}
	ptoken := PatreonTokenResponse{}
	err = json.Unmarshal(row.Data, &ptoken)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal data into token: %w", err)
	}
	patreonUserID := ptoken.PatreonUserID
	if patreonUserID == "" {
		// Rows linked before we stored patreon_user_id. The user's own token
		// is the only way to find out who they are; save it for next time.
		// XXX: For now. Remove when all Patreon rows have a patreon_user_id
		userData, err := fetchPatreonUserData(ptoken.AccessToken)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch user data: %w", err)
		}
		patreonUserID = userData.Data.ID
		tokenUpdate := []byte(fmt.Sprintf(`{"patreon_user_id": "%s"}`, patreonUserID))
		err = queries.UpdateIntegrationData(ctx, models.UpdateIntegrationDataParams{
			Uuid: row.Uuid,
			Data: tokenUpdate,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to update user data: %w", err)
		}
	}

	member, err := lookupCampaignMember(ctx, queries, patreonUserID)
	if err != nil {
		return nil, err
	}
	tierData, err := tierDataFromMember(member, time.Now())
	switch {
	case member == nil:
		log.Info().Str("userID", userID).Str("patreonUserID", patreonUserID).Msg("no-memberships")
	case errors.Is(err, ErrNotSubscribed):
		log.Info().Str("userID", userID).Msg("no-currently-entitled-tiers")
	case errors.Is(err, ErrNotPaidTier):
		log.Info().Str("userID", userID).Msg("on-free-tier")
	case err != nil:
		return nil, err
	}
	if err != nil {
		PatreonAPICache.Add(userID, nil)
		return nil, err
	}
	if tierData != nil && member.Attributes.IsGifted {
		log.Info().Str("userID", userID).Time("period-start", tierData.LastChargeDate).Msg("gifted-membership")
	}
	evicted := PatreonAPICache.Add(userID, tierData)
	log.Debug().Bool("evicted", evicted).Str("userID", userID).Msg("added-to-cache")
	return tierData, nil
}

func HighestTier(pm *PatreonMemberDatum) Tier {
	highestTier := TierNone
	for _, t := range pm.Relationships.CurrentlyEntitledTiers.Data {
		tn := PatreonTierIDs[t.ID]
		if tn > highestTier {
			highestTier = tn
		}
	}
	return highestTier
}
