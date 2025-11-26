package organizations

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NASPAIntegration handles NASPA title fetching
type NASPAIntegration struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewNASPAIntegration creates a new NASPA integration instance
func NewNASPAIntegration(baseURL string) *NASPAIntegration {
	return &NASPAIntegration{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NASPAAPIResponse represents the actual NASPA API response structure
type NASPAAPIResponse struct {
	Alive      int     `json:"alive"`
	City       string  `json:"city"`
	Country    string  `json:"country"`
	Expiry     string  `json:"expiry"`
	FirstName  string  `json:"firstName"`
	ID         int     `json:"id"`
	LastName   string  `json:"lastName"`
	NASPA      string  `json:"naspa"`       // Member ID
	Photo      string  `json:"photo"`
	RatingCSW  int     `json:"ratingCSW"`
	RatingOWL  int     `json:"ratingOWL"`
	State      string  `json:"state"`
	Suffix     string  `json:"suffix"`
	Title      string  `json:"title"`       // e.g., "SM", "GM", "M", "E"
	Version    float64 `json:"version"`
}

// FetchTitle fetches the title for a user from NASPA's API
// This method authenticates the user and then fetches their player data
func (n *NASPAIntegration) FetchTitle(memberID string, credentials map[string]string) (*TitleInfo, error) {
	username, ok := credentials["username"]
	if !ok {
		return nil, fmt.Errorf("username not provided")
	}

	password, ok := credentials["password"]
	if !ok {
		return nil, fmt.Errorf("password not provided")
	}

	// Step 1: Authenticate with NASPA to verify credentials
	if err := n.authenticate(username, password); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Step 2: Fetch player data from public API (authentication proves it's their account)
	playerData, err := n.fetchPlayerData(memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch player data: %w", err)
	}

	// Verify the authenticated username matches the requested member ID
	if playerData.NASPA != username {
		return nil, fmt.Errorf("member ID mismatch: authenticated as %s but requested data for %s", username, memberID)
	}

	now := time.Now()
	fullName := playerData.FirstName + " " + playerData.LastName
	if playerData.Suffix != "" {
		fullName += " " + playerData.Suffix
	}

	return &TitleInfo{
		Organization:     OrgNASPA,
		OrganizationName: "NASPA",
		RawTitle:         playerData.Title,
		NormalizedTitle:  n.NormalizeTitle(playerData.Title),
		MemberID:         playerData.NASPA,
		FullName:         fullName,
		LastFetched:      &now,
	}, nil
}

// authenticate authenticates with NASPA using username and password
func (n *NASPAIntegration) authenticate(username, password string) error {
	// POST form data to NASPA login endpoint
	formData := url.Values{}
	formData.Set("username", username)
	formData.Set("password", password)
	formData.Set("poslid_submit_login", "Log in")
	formData.Set("email", "")

	req, err := http.NewRequest("POST", n.BaseURL+"/cgi-bin/services.pl", strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := n.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make auth request: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	// NASPA returns 200 even for failed login, so check the response body
	// Failed login returns the login form again with 'poslid_submit_login' button
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	bodyStr := string(body)

	// If the response contains the login form, authentication failed
	if strings.Contains(bodyStr, "poslid_submit_login") || strings.Contains(bodyStr, `class=login`) || strings.Contains(bodyStr, `class="login"`) {
		return fmt.Errorf("invalid username or password")
	}

	return nil
}

// fetchPlayerData fetches player data from the public NASPA API
func (n *NASPAIntegration) fetchPlayerData(memberID string) (*NASPAAPIResponse, error) {
	apiURL := fmt.Sprintf("%s/rest/v1/player/%s", n.BaseURL, memberID)

	resp, err := n.HTTPClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch player data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp NASPAAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	return &apiResp, nil
}

// NormalizeTitle converts a NASPA title to a normalized title
// NASPA uses three title codes: GM (Grandmaster), SM (Scrabble Master), EX (Expert)
func (n *NASPAIntegration) NormalizeTitle(rawTitle string) NormalizedTitle {
	upper := strings.ToUpper(strings.TrimSpace(rawTitle))

	switch upper {
	case "GM":
		return TitleGrandmaster
	case "SM":
		return TitleMaster
	case "EX":
		return TitleExpert
	default:
		return TitleNone
	}
}

// GetOrganizationCode returns the organization code
func (n *NASPAIntegration) GetOrganizationCode() OrganizationCode {
	return OrgNASPA
}

// GetRealName fetches the user's real name from NASPA using their credentials
// Returns the full name in format "FirstName LastName" or "FirstName LastName Suffix"
func (n *NASPAIntegration) GetRealName(memberID string, credentials map[string]string) (string, error) {
	username, ok := credentials["username"]
	if !ok {
		return "", fmt.Errorf("username not provided")
	}

	password, ok := credentials["password"]
	if !ok {
		return "", fmt.Errorf("password not provided")
	}

	// Authenticate to verify credentials
	if err := n.authenticate(username, password); err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	// Fetch player data
	playerData, err := n.fetchPlayerData(memberID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch player data: %w", err)
	}

	// Verify the authenticated username matches the requested member ID
	if playerData.NASPA != username {
		return "", fmt.Errorf("member ID mismatch: authenticated as %s but requested data for %s", username, memberID)
	}

	// Build full name
	fullName := playerData.FirstName + " " + playerData.LastName
	if playerData.Suffix != "" {
		fullName += " " + playerData.Suffix
	}

	return fullName, nil
}
