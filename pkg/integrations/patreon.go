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

var ErrNotSubscribed = errors.New("user not subscribed")

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

type PatreonMemberData struct {
	Data struct {
		Attributes struct {
			Email            string `json:"email"`
			FullName         string `json:"full_name"`
			IsFollower       bool   `json:"is_follower"`
			LastChargeDate   string `json:"last_charge_date"`
			LastChargeStatus string `json:"last_charge_status"`
		} `json:"attributes"`
		ID            string `json:"id"`
		Relationships struct {
			CurrentlyEntitledTiers struct {
				Data []struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"data"`
			} `json:"currently_entitled_tiers"`
		} `json:"relationships"`
		Type string `json:"type"`
	} `json:"data"`
	Included []struct {
		Attributes struct {
			Description string `json:"description"`
			Title       string `json:"title"`
		} `json:"attributes"`
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"included"`
}

type PaidTierData struct {
	LastChargeDate   time.Time
	LastChargeStatus string
	TierName         string
}

type PatreonTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type PatreonError struct {
	code     int
	errorMsg string
}

func (p *PatreonError) Error() string {
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
		return nil, &PatreonError{resp.StatusCode, string(bts)}
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
		return nil, &PatreonError{resp.StatusCode, string(bts)}
	}

	var memberResp PatreonMemberData
	err = json.Unmarshal(bts, &memberResp)
	if err != nil {
		return nil, err
	}
	return &memberResp, nil

}

// DetermineUserTier determines what tier the user is in.
func DetermineUserTier(ctx context.Context, userID string, queries *models.Queries) (*PaidTierData, error) {
	// This function uses the cache.
	res, ok := PatreonAPICache.Get(userID)
	if ok {
		log.Info().Str("userID", userID).Msg("found-cached-tier-data")
		return res, nil
	}

	bts, err := queries.GetIntegrationData(ctx, models.GetIntegrationDataParams{
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
	err = json.Unmarshal(bts, &ptoken)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal data into token: %w", err)
	}
	userData, err := fetchPatreonUserData(ptoken.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch user data: %w", err)
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
	bts, err = queries.GetGlobalIntegrationData(ctx, PatreonIntegrationName)
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

	lastChargeDate, err := time.Parse(time.RFC3339, memberData.Data.Attributes.LastChargeDate)
	if err != nil {
		return nil, err
	}

	if len(memberData.Data.Relationships.CurrentlyEntitledTiers.Data) == 0 {
		log.Info().Str("userID", userID).Msg("no-currently-entitled-tiers")
		PatreonAPICache.Add(userID, nil)
		return nil, ErrNotSubscribed
	}
	tierID := memberData.Data.Relationships.CurrentlyEntitledTiers.Data[0].ID
	var tierName string
	for _, included := range memberData.Included {
		if included.Type == "tier" && included.ID == tierID {
			tierName = included.Attributes.Title
			break
		}
	}

	tierData := &PaidTierData{
		TierName:         tierName,
		LastChargeStatus: memberData.Data.Attributes.LastChargeStatus,
		LastChargeDate:   lastChargeDate,
	}
	evicted := PatreonAPICache.Add(userID, tierData)
	log.Debug().Bool("evicted", evicted).Str("userID", userID).Msg("added-to-cache")
	return tierData, nil
}
