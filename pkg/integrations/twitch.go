package integrations

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

const TwitchIntegrationName = "twitch"

// TwitchTokenResponse represents the response from Twitch token endpoint
type TwitchTokenResponse struct {
	AccessToken    string `json:"access_token"`
	RefreshToken   string `json:"refresh_token"`
	ExpiresIn      int    `json:"expires_in"`
	TokenType      string `json:"token_type"`
	TwitchUsername string `json:"twitch_username"`
}

// TwitchUser represents the user data fetched from Twitch API
type TwitchUser struct {
	ID          string `json:"id"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email,omitempty"`
}

func (s *OAuthIntegrationService) twitchCallback(w http.ResponseWriter, r *http.Request) {
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
	tokenResp, err := s.exchangeTwitchCodeForToken(code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Token exchange failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Fetch the user's Twitch information
	user, err := s.fetchTwitchUser(tokenResp.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch Twitch user: %v", err), http.StatusInternalServerError)
		return
	}

	log.Info().Interface("ud", user).Msg("userData")
	tokenResp.TwitchUsername = user.Login
	// re-dump token data for saving into table
	td, err := json.Marshal(tokenResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, err := s.queries.AddOrUpdateIntegration(ctx, models.AddOrUpdateIntegrationParams{
		UserUuid:        pgtype.Text{String: sess.UserUUID, Valid: true},
		IntegrationName: TwitchIntegrationName,
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

// exchangeCodeForToken exchanges the authorization code for access and refresh tokens
func (s *OAuthIntegrationService) exchangeTwitchCodeForToken(code string) (*TwitchTokenResponse, error) {
	url := "https://id.twitch.tv/oauth2/token"

	// Prepare the request payload
	data := fmt.Sprintf(
		"client_id=%s&client_secret=%s&code=%s&grant_type=authorization_code&redirect_uri=%s",
		s.cfg.TwitchClientID,
		s.cfg.TwitchClientSecret,
		code,
		s.cfg.TwitchRedirectURI,
	)

	req, err := http.NewRequest("POST", url, strings.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Set the appropriate headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("Token endpoint returned status %d: %s", resp.StatusCode, string(body)))
	}

	var tokenResp TwitchTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// fetchTwitchUser fetches the user's Twitch information using the access token
func (s *OAuthIntegrationService) fetchTwitchUser(accessToken string) (*TwitchUser, error) {
	url := "https://api.twitch.tv/helix/users"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set the required headers
	req.Header.Set("Client-ID", s.cfg.TwitchClientID)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("Twitch API returned status %d: %s", resp.StatusCode, string(body)))
	}

	// Twitch's users endpoint returns a JSON object with a "data" array
	var apiResponse struct {
		Data []TwitchUser `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, err
	}

	if len(apiResponse.Data) == 0 {
		return nil, errors.New("no user data found")
	}

	return &apiResponse.Data[0], nil
}
