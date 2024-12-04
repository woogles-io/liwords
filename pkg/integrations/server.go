package integrations

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/sessions"
)

type IntegrationService struct {
	sessionStore sessions.SessionStore
	cfg          *config.Config
}

func NewIntegrationService(s sessions.SessionStore, cfg *config.Config) *IntegrationService {
	return &IntegrationService{s, cfg}
}

type SaveCSRFRequest struct {
	CSRF string `json:"csrf"`
}

type OAuthState struct {
	CSRF       string `json:"csrfToken"`
	RedirectTo string `json:"redirectTo"`
}

func (s *IntegrationService) patreonCallback(w http.ResponseWriter, r *http.Request) {
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

	// TODO: Associate the Patreon account with your app's user account

	// Redirect the user back to the original page
	http.Redirect(w, r, state.RedirectTo, http.StatusFound)
}

func (s *IntegrationService) integrationsEndpoint(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	switch name {
	case "csrf":
		sess, err := apiserver.GetSession(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		var req SaveCSRFRequest
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil || req.CSRF == "" {
			http.Error(w, "Invalid CSRF token", http.StatusBadRequest)
			return
		}
		err = s.sessionStore.SetCSRFToken(ctx, sess, req.CSRF)
		if err != nil {
			http.Error(w, "Error setting CSRF token", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	case "patreon/callback":
		s.patreonCallback(w, r)
	}

}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (s *IntegrationService) exchangePatreonCodeForToken(code string) (*TokenResponse, error) {
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

	var tokenResp TokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return nil, err
	}
	log.Debug().Interface("tr", tokenResp).Msg("tokenResponse")

	return &tokenResp, nil
}

type UserData struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func fetchPatreonUserData(accessToken string) (*UserData, error) {
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

	userData := &UserData{
		ID:    userResp.Data.ID,
		Email: userResp.Data.Attributes.Email,
	}

	return userData, nil
}

// must end with /
const IntegrationServicePrefix = "/integrations/"

func (s *IntegrationService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, IntegrationServicePrefix) {
		s.integrationsEndpoint(w, r, strings.TrimPrefix(r.URL.Path, IntegrationServicePrefix))
	} else {
		http.NotFound(w, r)
	}
}
