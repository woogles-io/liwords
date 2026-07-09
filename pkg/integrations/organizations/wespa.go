package organizations

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WESPAIntegration handles player lookup via the WESPA (xerafin) player API
type WESPAIntegration struct {
	HTTPClient *http.Client
}

// NewWESPAIntegration creates a new WESPA integration instance
func NewWESPAIntegration() *WESPAIntegration {
	return &WESPAIntegration{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// wespaPlayerResponse is the subset of the WESPA player API response we use.
type wespaPlayerResponse struct {
	Name  string `json:"name"`
	Title string `json:"title"` // "GM"/"IM"/"M", or "" (JSON null) when untitled
}

// fetchPlayer performs a single GET against the WESPA player API and returns
// the player's name and title.
func (w *WESPAIntegration) fetchPlayer(memberID string) (*wespaPlayerResponse, error) {
	apiURL := fmt.Sprintf("https://wespa.xerafin.net/api/v2/player/%s", memberID)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "woogles.io")

	resp, err := w.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch WESPA player: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WESPA API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read WESPA API response: %w", err)
	}

	var payload wespaPlayerResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse WESPA API response: %w", err)
	}

	return &payload, nil
}

// FetchTitle fetches the player's name and title from the WESPA player API.
func (w *WESPAIntegration) FetchTitle(memberID string, credentials map[string]string) (*TitleInfo, error) {
	payload, err := w.fetchPlayer(memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch player: %w", err)
	}
	if payload.Name == "" {
		return nil, fmt.Errorf("WESPA API returned empty name for player %s", memberID)
	}

	now := time.Now()
	return &TitleInfo{
		Organization:     OrgWESPA,
		OrganizationName: "WESPA",
		RawTitle:         payload.Title,
		MemberID:         memberID,
		FullName:         payload.Name,
		LastFetched:      &now,
	}, nil
}

// GetOrganizationCode returns the organization code
func (w *WESPAIntegration) GetOrganizationCode() OrganizationCode {
	return OrgWESPA
}

// GetRealName fetches the user's real name from the WESPA API.
func (w *WESPAIntegration) GetRealName(memberID string, credentials map[string]string) (string, error) {
	payload, err := w.fetchPlayer(memberID)
	if err != nil {
		return "", err
	}
	if payload.Name == "" {
		return "", fmt.Errorf("WESPA API returned empty name for player %s", memberID)
	}

	return payload.Name, nil
}
