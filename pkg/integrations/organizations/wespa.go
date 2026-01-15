package organizations

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// wespaPlayer represents a player in the WESPA database
type wespaPlayer struct {
	Name  string
	Title string
}

// wespaCache holds the in-memory database with TTL
type wespaCache struct {
	mu        sync.RWMutex
	players   map[string]*wespaPlayer // Key: lowercase normalized name
	lastFetch time.Time
	cacheTTL  time.Duration
}

// WESPAIntegration handles WESPA (manual verification, scrapes HTML for name)
type WESPAIntegration struct {
	BaseURL    string
	HTTPClient *http.Client
	cache      *wespaCache
}

// NewWESPAIntegration creates a new WESPA integration instance
func NewWESPAIntegration() *WESPAIntegration {
	return &WESPAIntegration{
		BaseURL: "https://legacy.wespa.org",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: &wespaCache{
			players:  make(map[string]*wespaPlayer),
			cacheTTL: 24 * time.Hour,
		},
	}
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

// ensureCacheValid checks if cache needs refresh and updates if necessary
func (w *WESPAIntegration) ensureCacheValid() error {
	w.cache.mu.RLock()
	needsRefresh := time.Since(w.cache.lastFetch) > w.cache.cacheTTL || len(w.cache.players) == 0
	w.cache.mu.RUnlock()

	if !needsRefresh {
		return nil
	}

	// Need to refresh - acquire write lock
	return w.refreshCache()
}

// refreshCache downloads and parses the WESPA database
func (w *WESPAIntegration) refreshCache() error {
	// Download the database file
	resp, err := w.HTTPClient.Get(w.BaseURL + "/latest.txt")
	if err != nil {
		return fmt.Errorf("failed to download WESPA database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("WESPA database returned status %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read WESPA database: %w", err)
	}

	// Parse the file
	players := parseLatestTxt(string(body))

	// Update cache with write lock
	w.cache.mu.Lock()
	w.cache.players = players
	w.cache.lastFetch = time.Now()
	w.cache.mu.Unlock()

	return nil
}

// parseLatestTxt parses the WESPA latest.txt file
func parseLatestTxt(data string) map[string]*wespaPlayer {
	players := make(map[string]*wespaPlayer)
	scanner := bufio.NewScanner(strings.NewReader(data))

	// Skip header line (first line)
	if !scanner.Scan() {
		return players
	}

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 30 {
			continue // Line too short
		}

		// Extract name from fixed-width column (chars 9-29, 21 chars)
		name := strings.TrimSpace(line[9:30])
		if name == "" {
			continue
		}

		// Extract remaining fields after position 30
		remainder := strings.TrimSpace(line[30:])
		fields := strings.Fields(remainder)

		// Title is at index 4 of the remaining fields (column 8 overall)
		title := ""
		if len(fields) >= 5 {
			title = fields[4]
			if title == "--" {
				title = ""
			}
		}

		// Use lowercase name as key for prefix matching
		key := strings.ToLower(name)
		players[key] = &wespaPlayer{
			Name:  name,
			Title: title,
		}
	}

	return players
}

// findTitleByName searches for a title by name using prefix matching
// The WESPA page has the full real name, but database names may be shortened
// e.g., "Conrad Bassett-Bouch" in database matches "Conrad Bassett-Bouchard" from page
func (w *WESPAIntegration) findTitleByName(fullName string) string {
	if fullName == "" {
		return ""
	}

	// Ensure cache is valid before lookup
	if err := w.ensureCacheValid(); err != nil {
		// If cache fetch fails, return empty title
		return ""
	}

	// Normalize the full name for matching
	normalizedFullName := strings.ToLower(strings.TrimSpace(fullName))

	w.cache.mu.RLock()
	defer w.cache.mu.RUnlock()

	for key, player := range w.cache.players {
		// Check if the database name is a prefix of the full name from WESPA page
		if strings.HasPrefix(normalizedFullName, key) {
			return player.Title
		}
	}

	return ""
}
