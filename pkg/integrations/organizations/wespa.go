package organizations

import (
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// XXX: replace this with https://legacy.wespa.org/latest.txt soon, when titles are added to it.
//
//go:embed wespa-titlists.txt
var wespaTitlistsData string

// WESPATitlist represents a single entry in the WESPA titlists
type WESPATitlist struct {
	Country string
	Title   string
	Name    string
	Norms   string
}

// WESPAIntegration handles WESPA (manual verification, scrapes HTML for name)
type WESPAIntegration struct {
	BaseURL    string
	HTTPClient *http.Client
	titlists   []WESPATitlist
}

// NewWESPAIntegration creates a new WESPA integration instance
func NewWESPAIntegration() *WESPAIntegration {
	w := &WESPAIntegration{
		BaseURL: "https://legacy.wespa.org",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	w.titlists = parseTitlists(wespaTitlistsData)
	return w
}

// FetchTitle fetches title from the embedded WESPA titlists by matching the player's name
func (w *WESPAIntegration) FetchTitle(memberID string, credentials map[string]string) (*TitleInfo, error) {
	// Fetch the player's full name from their WESPA page
	fullName, err := w.GetRealName(memberID, credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch player name: %w", err)
	}

	// Search for title in titlists using prefix matching
	title := w.findTitleByName(fullName)

	now := time.Now()
	return &TitleInfo{
		Organization:     OrgWESPA,
		OrganizationName: "WESPA",
		RawTitle:         title,
		MemberID:         memberID,
		FullName:         fullName,
		LastFetched:      &now,
	}, nil
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

// parseTitlists parses the embedded WESPA titlists data
func parseTitlists(data string) []WESPATitlist {
	var titlists []WESPATitlist
	lines := strings.Split(data, "\n")

	// Skip header lines (first 2 lines)
	for i := 2; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}

		// Format: [Country] [Title] [Name...] [Norms]
		country := parts[0]
		title := parts[1]
		norms := parts[len(parts)-1]

		// Skip entries with no title
		if title == "--" {
			continue
		}

		// Everything between title and norms is the name
		nameParts := parts[2 : len(parts)-1]
		name := strings.Join(nameParts, " ")

		titlists = append(titlists, WESPATitlist{
			Country: country,
			Title:   title,
			Name:    name,
			Norms:   norms,
		})
	}

	return titlists
}

// findTitleByName searches for a title by name using prefix matching
// The WESPA page has the full real name, but titlist names may be shortened
// e.g., "Conrad Bassett-Bouch" in titlist matches "Conrad Bassett-Bouchard" from page
func (w *WESPAIntegration) findTitleByName(fullName string) string {
	if fullName == "" {
		return ""
	}

	// Normalize the full name for matching
	normalizedFullName := strings.ToLower(strings.TrimSpace(fullName))

	for _, entry := range w.titlists {
		normalizedListName := strings.ToLower(entry.Name)

		// Check if the titlist name is a prefix of the full name from WESPA page
		if strings.HasPrefix(normalizedFullName, normalizedListName) {
			return entry.Title
		}
	}

	return ""
}
