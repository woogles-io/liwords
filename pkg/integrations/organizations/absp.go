package organizations

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
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

// ABSPIntegration handles ABSP with in-memory cached database
type ABSPIntegration struct {
	BaseURL    string
	HTTPClient *http.Client
	cache      *abspCache
}

// abspCache holds the in-memory database with TTL
type abspCache struct {
	mu         sync.RWMutex
	players    map[string]*ABSPPlayer // Key: normalized member ID
	lastFetch  time.Time
	cacheTTL   time.Duration
}

// NewABSPIntegration creates a new ABSP integration instance
func NewABSPIntegration() *ABSPIntegration {
	return &ABSPIntegration{
		BaseURL: "https://absp-database.org",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: &abspCache{
			players:   make(map[string]*ABSPPlayer),
			cacheTTL:  24 * time.Hour,
		},
	}
}

// FetchTitle fetches title from cached ABSP database
func (a *ABSPIntegration) FetchTitle(memberID string, credentials map[string]string) (*TitleInfo, error) {
	// Ensure cache is up to date
	if err := a.ensureCacheValid(); err != nil {
		return nil, fmt.Errorf("failed to load ABSP database: %w", err)
	}

	// Look up player
	player, err := a.getPlayer(memberID)
	if err != nil {
		return nil, err
	}

	if player.Title == "" {
		return nil, fmt.Errorf("player has no title in ABSP database")
	}

	now := time.Now()
	fullName := player.FirstName + " " + player.LastName

	return &TitleInfo{
		Organization:     OrgABSP,
		OrganizationName: "ABSP",
		RawTitle:         player.Title,
		NormalizedTitle:  a.NormalizeTitle(player.Title),
		MemberID:         player.MemberID,
		FullName:         fullName,
		LastFetched:      &now,
	}, nil
}

// NormalizeTitle converts an ABSP title to a normalized title
// ABSP has two titles: GM (Grandmaster) and EXP (Expert)
func (a *ABSPIntegration) NormalizeTitle(rawTitle string) NormalizedTitle {
	upper := strings.ToUpper(strings.TrimSpace(rawTitle))

	switch upper {
	case "GM":
		return TitleGrandmaster
	case "EXP":
		return TitleExpert // Note: Should be Master but user requested Expert
	default:
		return TitleNone
	}
}

// GetOrganizationCode returns the organization code
func (a *ABSPIntegration) GetOrganizationCode() OrganizationCode {
	return OrgABSP
}

// GetRealName fetches the user's real name from the cached ABSP database
func (a *ABSPIntegration) GetRealName(memberID string, credentials map[string]string) (string, error) {
	// Ensure cache is up to date
	if err := a.ensureCacheValid(); err != nil {
		return "", fmt.Errorf("failed to load ABSP database: %w", err)
	}

	// Look up player
	player, err := a.getPlayer(memberID)
	if err != nil {
		return "", err
	}

	fullName := player.FirstName + " " + player.LastName
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
