package organizations

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ABSPPlayer represents a player in the ABSP database
type ABSPPlayer struct {
	LastName  string
	FirstName string
	Title     string // GM or EXP (extracted from last name if present)
	Rating    int
	MemberID  string // Normalized (leading zeros stripped)
}

// ABSPIntegration handles ABSP authentication and data fetching
type ABSPIntegration struct {
	BaseURL    string
	HTTPClient *http.Client
	cache      *abspCache
}

// abspCache holds the in-memory database with TTL
type abspCache struct {
	mu        sync.RWMutex
	players   map[string]*ABSPPlayer // Key: normalized member ID
	lastFetch time.Time
	cacheTTL  time.Duration
}

// NewABSPIntegration creates a new ABSP integration instance
func NewABSPIntegration() *ABSPIntegration {
	return &ABSPIntegration{
		BaseURL: "https://absp-database.org",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: &abspCache{
			players:  make(map[string]*ABSPPlayer),
			cacheTTL: 24 * time.Hour,
		},
	}
}

// ABSPMemberInfo holds the parsed member information from the profile page
type ABSPMemberInfo struct {
	MemberID  string
	FirstName string
	LastName  string
}

// FetchTitle fetches title from ABSP using authentication to verify identity,
// then looks up the title in the cached database
func (a *ABSPIntegration) FetchTitle(memberID string, credentials map[string]string) (*TitleInfo, error) {
	username, ok := credentials["username"]
	if !ok {
		return nil, fmt.Errorf("username not provided")
	}

	password, ok := credentials["password"]
	if !ok {
		return nil, fmt.Errorf("password not provided")
	}

	// Step 1: Authenticate with ABSP and get member info from profile
	memberInfo, err := a.authenticateAndFetchProfile(username, password)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Step 2: Ensure cache is up to date
	if err := a.ensureCacheValid(); err != nil {
		return nil, fmt.Errorf("failed to load ABSP database: %w", err)
	}

	// Step 3: Look up title in the cached database using the member ID from profile
	player, err := a.getPlayer(memberInfo.MemberID)
	if err != nil {
		// Player might not be in the ratings database yet, but auth succeeded
		// Use info from profile page
		now := time.Now()
		fullName := memberInfo.FirstName + " " + memberInfo.LastName

		return &TitleInfo{
			Organization:     OrgABSP,
			OrganizationName: "ABSP",
			RawTitle:         "",
			NormalizedTitle:  TitleNone,
			MemberID:         memberInfo.MemberID,
			FullName:         fullName,
			LastFetched:      &now,
		}, nil
	}

	now := time.Now()
	// Prefer name from profile page (more authoritative)
	fullName := memberInfo.FirstName + " " + memberInfo.LastName

	return &TitleInfo{
		Organization:     OrgABSP,
		OrganizationName: "ABSP",
		RawTitle:         player.Title,
		NormalizedTitle:  a.NormalizeTitle(player.Title),
		MemberID:         memberInfo.MemberID,
		FullName:         fullName,
		LastFetched:      &now,
	}, nil
}

// authenticateAndFetchProfile authenticates with ABSP and fetches the member's profile
func (a *ABSPIntegration) authenticateAndFetchProfile(username, password string) (*ABSPMemberInfo, error) {
	// Create a cookie jar to store the session cookie
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}

	// Step 1: Authenticate
	formData := url.Values{}
	formData.Set("username", username)
	formData.Set("password", password)
	formData.Set("action", "password_logon")

	req, err := http.NewRequest("POST", a.BaseURL+"/absp_log_on_or_off.php", strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body to check for successful login
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	bodyStr := string(body)

	// Check for successful login indicator
	if !strings.Contains(bodyStr, "<p style='color:green'>ABSP Member</p>") {
		return nil, fmt.Errorf("invalid username or password")
	}

	// Step 2: Fetch profile page to get member info
	profileResp, err := client.Get(a.BaseURL + "/edit_absp_member.php")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile page: %w", err)
	}
	defer profileResp.Body.Close()

	if profileResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("profile page returned status %d", profileResp.StatusCode)
	}

	profileBody, err := io.ReadAll(profileResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile page: %w", err)
	}

	// Parse the member info from the HTML
	memberInfo, err := a.parseProfilePage(string(profileBody))
	if err != nil {
		return nil, fmt.Errorf("failed to parse profile page: %w", err)
	}

	return memberInfo, nil
}

// parseProfilePage extracts member information from the ABSP profile page HTML
func (a *ABSPIntegration) parseProfilePage(html string) (*ABSPMemberInfo, error) {
	info := &ABSPMemberInfo{}

	// Pattern to match table cells with label and value
	// Format: <td>label:</td>\n<td>\nvalue</td>
	extractValue := func(label string) string {
		// Look for pattern: <td>label:</td> followed by <td> with the value
		pattern := regexp.MustCompile(`(?s)<td>` + regexp.QuoteMeta(label) + `:</td>\s*<td>\s*([^<]+?)\s*</td>`)
		matches := pattern.FindStringSubmatch(html)
		if len(matches) >= 2 {
			return strings.TrimSpace(matches[1])
		}
		return ""
	}

	info.MemberID = extractValue("absp_number")
	info.FirstName = extractValue("first_name")
	info.LastName = extractValue("last_name")

	if info.MemberID == "" {
		return nil, fmt.Errorf("could not find member ID in profile page")
	}

	return info, nil
}

// NormalizeTitle converts an ABSP title to a normalized title
// ABSP has two titles: GM (Grandmaster) and EXP (Expert)
func (a *ABSPIntegration) NormalizeTitle(rawTitle string) NormalizedTitle {
	upper := strings.ToUpper(strings.TrimSpace(rawTitle))

	switch upper {
	case "GM":
		return TitleGrandmaster
	case "EXP":
		return TitleExpert
	default:
		return TitleNone
	}
}

// GetOrganizationCode returns the organization code
func (a *ABSPIntegration) GetOrganizationCode() OrganizationCode {
	return OrgABSP
}

// GetRealName fetches the user's real name from ABSP using their credentials
func (a *ABSPIntegration) GetRealName(memberID string, credentials map[string]string) (string, error) {
	username, ok := credentials["username"]
	if !ok {
		return "", fmt.Errorf("username not provided")
	}

	password, ok := credentials["password"]
	if !ok {
		return "", fmt.Errorf("password not provided")
	}

	// Authenticate and fetch profile
	memberInfo, err := a.authenticateAndFetchProfile(username, password)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	fullName := memberInfo.FirstName + " " + memberInfo.LastName
	return fullName, nil
}

// ensureCacheValid checks if cache needs refresh and updates if necessary
func (a *ABSPIntegration) ensureCacheValid() error {
	a.cache.mu.RLock()
	needsRefresh := time.Since(a.cache.lastFetch) > a.cache.cacheTTL || len(a.cache.players) == 0
	a.cache.mu.RUnlock()

	if !needsRefresh {
		return nil
	}

	// Need to refresh - acquire write lock
	return a.refreshCache()
}

// refreshCache downloads and parses the ABSP database
func (a *ABSPIntegration) refreshCache() error {
	// Download the database file
	resp, err := a.HTTPClient.Get(a.BaseURL + "/current.txt")
	if err != nil {
		return fmt.Errorf("failed to download ABSP database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ABSP database returned status %d", resp.StatusCode)
	}

	// Parse the file
	players, err := a.parseDatabase(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse ABSP database: %w", err)
	}

	// Update cache with write lock
	a.cache.mu.Lock()
	a.cache.players = players
	a.cache.lastFetch = time.Now()
	a.cache.mu.Unlock()

	return nil
}

// parseDatabase parses the ABSP database text file
func (a *ABSPIntegration) parseDatabase(r io.Reader) (map[string]*ABSPPlayer, error) {
	players := make(map[string]*ABSPPlayer)
	scanner := bufio.NewScanner(r)
	titleRegex := regexp.MustCompile(`^([A-Z]+)\((GM|EXP)\)$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Split by whitespace
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue // Skip malformed lines
		}

		lastName := fields[0]
		firstName := fields[1]
		var title string

		// Check if last name contains title in parentheses
		if matches := titleRegex.FindStringSubmatch(lastName); matches != nil {
			lastName = matches[1]
			title = matches[2]
		}

		// Parse rating (optional)
		rating := 0
		if len(fields) >= 3 {
			if r, err := strconv.Atoi(fields[2]); err == nil {
				rating = r
			}
		}

		// Parse member ID (5th column = index 4, optional)
		// Note: There may be additional columns after this, but we always use the 5th column
		var memberID string
		if len(fields) >= 5 {
			memberID = strings.TrimSpace(fields[4])
			// Normalize member ID by converting to int and back to string (strips leading zeros)
			// This allows "00745" and "745" to both match as "745"
			if memberID != "" {
				if id, err := strconv.Atoi(memberID); err == nil {
					memberID = strconv.Itoa(id)
				}
			}
		}

		// Only add players with member IDs
		if memberID != "" {
			players[memberID] = &ABSPPlayer{
				LastName:  lastName,
				FirstName: firstName,
				Title:     title,
				Rating:    rating,
				MemberID:  memberID,
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading database: %w", err)
	}

	return players, nil
}

// getPlayer looks up a player by member ID (normalized)
func (a *ABSPIntegration) getPlayer(memberID string) (*ABSPPlayer, error) {
	// Normalize the input member ID
	normalizedID := memberID
	if id, err := strconv.Atoi(memberID); err == nil {
		normalizedID = strconv.Itoa(id) // Strip leading zeros
	}

	a.cache.mu.RLock()
	defer a.cache.mu.RUnlock()

	player, ok := a.cache.players[normalizedID]
	if !ok {
		return nil, fmt.Errorf("player not found in ABSP database")
	}

	return player, nil
}
