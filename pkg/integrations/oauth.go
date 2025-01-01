package integrations

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/sessions"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

type OAuthIntegrationService struct {
	sessionStore sessions.SessionStore
	queries      *models.Queries
	cfg          *config.Config
}

func NewOAuthIntegrationService(s sessions.SessionStore, q *models.Queries, cfg *config.Config) *OAuthIntegrationService {
	return &OAuthIntegrationService{s, q, cfg}
}

type SaveCSRFRequest struct {
	CSRF string `json:"csrf"`
}

type OAuthState struct {
	CSRF       string `json:"csrfToken"`
	RedirectTo string `json:"redirectTo"`
}

func (s *OAuthIntegrationService) integrationsEndpoint(w http.ResponseWriter, r *http.Request, name string) {
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
	case "twitch/callback":
		s.twitchCallback(w, r)
	}

}

// must end with /
const OAuthIntegrationServicePrefix = "/integrations/"

func (s *OAuthIntegrationService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, OAuthIntegrationServicePrefix) {
		s.integrationsEndpoint(w, r, strings.TrimPrefix(r.URL.Path, OAuthIntegrationServicePrefix))
	} else {
		http.NotFound(w, r)
	}
}
