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
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

const (
	ChargeStatusPaid = "Paid"
)

// Use a simple LRU cache to avoid hammering Patreon API
// in case people refresh etc.
var PatreonAPICache *expirable.LRU[string, *PaidTierData]

func init() {
	PatreonAPICache = expirable.NewLRU[string, *PaidTierData](0, nil, time.Second*60)
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
	LastChargeDate   string `json:"last_charge_date"`
	LastChargeStatus string `json:"last_charge_status"`
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

type PatreonMemberData struct {
	Data     PatreonMemberDatum `json:"data"`
	Included []struct {
		Attributes struct {
			Description string `json:"description"`
			Title       string `json:"title"`
		} `json:"attributes"`
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"included"`
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
		UserUuid:        pgtype.Text{String: sess.UserUUID, Valid: true},
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
	tokenURL := "https://www.patreon.com/api/oauth2/token"

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
	tokenURL := "https://www.patreon.com/api/oauth2/token"

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

func fetchPatreonUserData(accessToken string) (*PatreonUserData, error) {
	apiURL := "https://www.patreon.com/api/oauth2/v2/identity"

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

	client := &http.Client{}
	resp, err := client.Do(req)
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

func fetchPatreonMemberData(accessToken, memberID string) (*PatreonMemberData, error) {
	apiURL := "https://www.patreon.com/api/oauth2/v2/members/" + memberID
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	// Set the Authorization header
	req.Header.Set("Authorization", "Bearer "+accessToken)

	query := req.URL.Query() // Get a copy of the query parameters
	query.Set("fields[member]", "full_name,is_follower,last_charge_date,last_charge_status,email")
	query.Set("include", "currently_entitled_tiers")
	query.Set("fields[tier]", "title,description")
	req.URL.RawQuery = query.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
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

	var memberResp PatreonMemberData
	err = json.Unmarshal(bts, &memberResp)
	if err != nil {
		return nil, err
	}
	return &memberResp, nil

}

func GetCampaignSubscribers(ctx context.Context, globalToken string) (*MultiplePatreonMemberData, error) {
	apiURL := fmt.Sprintf(
		"https://www.patreon.com/api/oauth2/v2/campaigns/%d/members", CampaignID)

	paginating := true

	members := &MultiplePatreonMemberData{}
	page1 := true
	for paginating {

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, err
		}
		// Set the Authorization header
		req.Header.Set("Authorization", "Bearer "+globalToken)
		if page1 {
			query := req.URL.Query() // Get a copy of the query parameters
			query.Set("include", "currently_entitled_tiers,user")
			req.URL.RawQuery = query.Encode()
		}
		page1 = false

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}

		bts, err := io.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, &PatreonAPIError{resp.StatusCode, string(bts)}
		}
		thispage := &MultiplePatreonMemberData{}
		err = json.Unmarshal(bts, thispage)
		if err != nil {
			resp.Body.Close()
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
		resp.Body.Close()
	}
	return members, nil
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
		UserUuid:        pgtype.Text{String: userID, Valid: true},
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
	userData, err := fetchPatreonUserData(ptoken.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch user data: %w", err)
	}
	// Update integration rows that had no patreon_user_id
	// XXX: For now. Remove when all Patreon rows have a patreon_user_id
	if ptoken.PatreonUserID == "" {
		tokenUpdate := []byte(fmt.Sprintf(`{"patreon_user_id": "%s"}`, userData.Data.ID))
		err = queries.UpdateIntegrationData(ctx, models.UpdateIntegrationDataParams{
			Uuid: row.Uuid,
			Data: tokenUpdate,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to update user data: %w", err)
		}
	}

	memberships := userData.Data.Relationships.Memberships.Data
	if len(memberships) == 0 {
		// Not subscribed.
		log.Info().Str("userID", userID).Msg("no-memberships")
		PatreonAPICache.Add(userID, nil)
		return nil, nil
	}
	// we should really only find one ID. But, shrug.
	memberIDs := []string{}
	for _, m := range memberships {
		if m.Type == "member" {
			memberIDs = append(memberIDs, m.ID)
		}
	}
	// Now look up the ID in the members endpoint, using the global/creator token.
	bts, err := queries.GetGlobalIntegrationData(ctx, PatreonIntegrationName)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bts, &ptoken)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal global integration data into token: %w", err)
	}
	memberData, err := fetchPatreonMemberData(ptoken.AccessToken, memberIDs[0])
	if err != nil {
		return nil, err
	}
	if len(memberData.Included) == 0 {
		return nil, errors.New("missing-member-data")
	}

	if len(memberData.Data.Relationships.CurrentlyEntitledTiers.Data) == 0 {
		log.Info().Str("userID", userID).Msg("no-currently-entitled-tiers")
		PatreonAPICache.Add(userID, nil)
		return nil, ErrNotSubscribed
	}
	tier := HighestTier(&memberData.Data)
	if tier == TierFree {
		log.Info().Str("userID", userID).Msg("on-free-tier")
		PatreonAPICache.Add(userID, nil)
		return nil, ErrNotPaidTier
	}
	lastChargeDate, err := time.Parse(time.RFC3339, memberData.Data.Attributes.LastChargeDate)
	if err != nil {
		return nil, err
	}

	tierData := &PaidTierData{
		Tier:             tier,
		LastChargeStatus: memberData.Data.Attributes.LastChargeStatus,
		LastChargeDate:   lastChargeDate,
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
