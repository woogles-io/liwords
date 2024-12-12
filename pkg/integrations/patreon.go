package integrations

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

type PatreonTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type PatreonUserData struct {
	ID    string `json:"id"`
	Email string `json:"email"`
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

func fetchPatreonUserData(accessToken string) (*PatreonUserData, error) {
	apiURL := "https://www.patreon.com/api/oauth2/v2/identity?fields[user]=email,first_name,last_name&fields[campaign]=summary,is_monthly&include=memberships,campaign"

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userResp struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Email     string `json:"email"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			} `json:"attributes"`
		} `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&userResp)
	if err != nil {
		return nil, err
	}

	userData := &PatreonUserData{
		ID:    userResp.Data.ID,
		Email: userResp.Data.Attributes.Email,
	}

	return userData, nil
}
