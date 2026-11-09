package organizations

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"
)

// SGTitlesURL is the Singapore Scrabble Association's titles list, as one
// JSON file: {"players": [{"id": "1234", "name": "...", "title": "NM"}, ...]}.
// A player's title may be "" for none, and the list may be empty.
const SGTitlesURL = "https://www.toucanet.com/archives/p/bytitle.json"

// sgMemberIDRegex matches a membership number. The capture, the number
// without its leading zeros ("0" stays "0"), is the member ID; it is kept as
// a string, so any length works.
var sgMemberIDRegex = regexp.MustCompile(`^0*(\d+)$`)

// SGIntegration looks members up in the Singapore Scrabble Association's
// titles list. A member missing from it, or listed with an empty title, holds
// no title.
type SGIntegration struct {
	TitlesURL  string
	HTTPClient *http.Client

	mu      sync.Mutex
	players map[string]sgPlayer // Key: member ID; loaded once per instance
	loadErr error               // a failed load, not retried by this instance
}

// sgPlayer is one entry of the titles list.
type sgPlayer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
}

// NewSGIntegration creates a new SG integration instance
func NewSGIntegration() *SGIntegration {
	return &SGIntegration{
		TitlesURL: SGTitlesURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// sgMemberID returns the member ID for a membership number, or false when
// s is not one.
func sgMemberID(s string) (string, bool) {
	m := sgMemberIDRegex.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// parseSGTitles reads the titles list. An empty list is valid, and so is an
// empty title: nobody, or that player, holds a title. Anything wrong in it (no
// players array, a bad or repeated number, a missing or unknown title, a
// missing name) refuses the whole list, so that a broken list changes no
// one's title.
func parseSGTitles(body []byte) (map[string]sgPlayer, error) {
	// The array and each title are read through pointers, so that a missing
	// key or a null is told apart from an empty array or an empty title.
	var list struct {
		Players *[]struct {
			ID    string  `json:"id"`
			Name  string  `json:"name"`
			Title *string `json:"title"`
		} `json:"players"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("failed to parse SG titles list: %w", err)
	}
	if list.Players == nil {
		return nil, fmt.Errorf("SG titles list has no players array")
	}
	players := make(map[string]sgPlayer, len(*list.Players))
	for _, p := range *list.Players {
		id, ok := sgMemberID(p.ID)
		if !ok {
			return nil, fmt.Errorf("SG titles list has a bad member ID %q", p.ID)
		}
		if _, repeated := players[id]; repeated {
			return nil, fmt.Errorf("SG titles list repeats member ID %s", id)
		}
		if p.Title == nil {
			return nil, fmt.Errorf("SG titles list has no title for member ID %s", id)
		}
		if _, known := TitleRegistry[OrgSG][*p.Title]; !known && *p.Title != "" {
			return nil, fmt.Errorf("SG titles list has an unknown title %q for member ID %s", *p.Title, id)
		}
		if p.Name == "" {
			return nil, fmt.Errorf("SG titles list has no name for member ID %s", id)
		}
		players[id] = sgPlayer{ID: id, Name: p.Name, Title: *p.Title}
	}
	return players, nil
}

// load downloads and parses the titles list, once per instance: a failure is
// kept too, so that a scheduled refresh of every linked user does not
// download a broken list again for each of them.
func (s *SGIntegration) load() (map[string]sgPlayer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.players == nil && s.loadErr == nil {
		s.players, s.loadErr = s.download()
	}
	return s.players, s.loadErr
}

// download fetches and parses the titles list.
func (s *SGIntegration) download() (map[string]sgPlayer, error) {
	req, err := http.NewRequest("GET", s.TitlesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "woogles.io")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download SG titles list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SG titles list returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read SG titles list: %w", err)
	}

	return parseSGTitles(body)
}

// FetchTitle looks the member up in the titles list. A member missing from
// it comes back with an empty RawTitle and FullName, not an error, so that a
// refresh takes away a title that is no longer listed.
func (s *SGIntegration) FetchTitle(memberID string, credentials map[string]string) (*TitleInfo, error) {
	id, ok := sgMemberID(memberID)
	if !ok {
		return nil, fmt.Errorf("%q is not a membership number", memberID)
	}
	players, err := s.load()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	info := &TitleInfo{
		Organization:     OrgSG,
		OrganizationName: OrganizationRegistry[OrgSG].Name,
		MemberID:         id,
		LastFetched:      &now,
	}
	if p, ok := players[id]; ok {
		info.RawTitle = p.Title
		info.FullName = p.Name
	}
	return info, nil
}

// GetOrganizationCode returns the organization code
func (s *SGIntegration) GetOrganizationCode() OrganizationCode {
	return OrgSG
}

// GetRealName returns the member's name from the titles list. A member
// missing from it has none, which is an error, so that only a listed member
// can be linked.
func (s *SGIntegration) GetRealName(memberID string, credentials map[string]string) (string, error) {
	info, err := s.FetchTitle(memberID, credentials)
	if err != nil {
		return "", err
	}
	if info.FullName == "" {
		return "", fmt.Errorf("member ID %s is not in the SG titles list", info.MemberID)
	}
	return info.FullName, nil
}
