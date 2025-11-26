package organizations

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// WESPAIntegration handles WESPA (manual verification, scrapes HTML for name)
type WESPAIntegration struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewWESPAIntegration creates a new WESPA integration instance
func NewWESPAIntegration() *WESPAIntegration {
	return &WESPAIntegration{
		BaseURL: "https://wespa.org",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchTitle returns an error since WESPA doesn't have titles yet
func (w *WESPAIntegration) FetchTitle(memberID string, credentials map[string]string) (*TitleInfo, error) {
	return nil, fmt.Errorf("WESPA does not support title fetching - titles not yet implemented by WESPA")
}

// NormalizeTitle converts a WESPA title to a normalized title
func (w *WESPAIntegration) NormalizeTitle(rawTitle string) NormalizedTitle {
	// WESPA title mapping
	lower := strings.ToLower(strings.TrimSpace(rawTitle))

	switch {
	case strings.Contains(lower, "grandmaster") || strings.Contains(lower, "grand master") || lower == "gm":
		return TitleGrandmaster
	case strings.Contains(lower, "master"):
		return TitleMaster
	case strings.Contains(lower, "expert"):
		return TitleExpert
	default:
		return TitleNone
	}
}

// GetOrganizationCode returns the organization code
func (w *WESPAIntegration) GetOrganizationCode() OrganizationCode {
	return OrgWESPA
}

// GetRealName fetches the user's real name from WESPA by scraping their player page
// Returns the name extracted from the <title> tag of the player page
func (w *WESPAIntegration) GetRealName(memberID string, credentials map[string]string) (string, error) {
	// Fetch the player page HTML
	playerURL := fmt.Sprintf("%s/aardvark/html/players/%s.html", w.BaseURL, memberID)

	resp, err := w.HTTPClient.Get(playerURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch WESPA player page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("WESPA player page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read player page: %w", err)
	}

	// Extract name from <title> tag
	name, err := w.extractNameFromHTML(string(body))
	if err != nil {
		return "", fmt.Errorf("failed to extract name from page: %w", err)
	}

	return name, nil
}

// extractNameFromHTML extracts the player name from the <title> tag
func (w *WESPAIntegration) extractNameFromHTML(html string) (string, error) {
	// Match <title>Player Name</title>
	re := regexp.MustCompile(`<title>\s*([^<]+?)\s*</title>`)
	matches := re.FindStringSubmatch(html)

	if len(matches) < 2 {
		return "", fmt.Errorf("could not find title tag in HTML")
	}

	name := strings.TrimSpace(matches[1])
	if name == "" {
		return "", fmt.Errorf("title tag is empty")
	}

	return name, nil
}
