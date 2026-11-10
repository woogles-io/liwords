package organizations

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
)

// ABSPBaseURL is absp-database.org, which has both the members' login and
// the ratings list, current.txt: one player per line, the columns separated
// by spaces, as
//
//	LAST(TITLE) FIRST RATING POSITION MEMNO GAMES
//
// The title, GM or EXP, is joined to the last name in parentheses, and only
// a titled player has one. MEMNO is the 5-digit ABSP membership number,
// blank for nonmembers and lapsed members; GAMES is the number of games
// available to be rated, blank for the full quota. A name has underscores for
// its spaces, and a first name may end in a {PLACE} that tells two players of
// the same name apart.
const ABSPBaseURL = "https://absp-database.org"

// abspLineRegex matches a line of current.txt. A blank column leaves its two
// spaces, so MEMNO and GAMES are told apart by position, not by splitting on
// runs of spaces; the names and the numbers before them may be padded.
var abspLineRegex = regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\d+)\s+(\d+) (\d{5}|) ?(\d*)\s*$`)

// abspTitleRegex splits a title off a last name.
var abspTitleRegex = regexp.MustCompile(`^(.+)\(([A-Z]+)\)$`)

// abspPlaceRegex is the {PLACE} that may end a first name.
var abspPlaceRegex = regexp.MustCompile(`\{[^}]*\}$`)

// abspMemberIDRegex matches a membership number. The capture, the number
// without its leading zeros ("0" stays "0"), is the member ID, so that 00745
// and 745 are the same member.
var abspMemberIDRegex = regexp.MustCompile(`^0*(\d+)$`)

// ABSPIntegration connects ABSP members. A member logs in to
// absp-database.org once, to prove who they are; their title, and the
// title of a member an admin links, comes from the public ratings list, so
// that no login is kept for refreshing it. A member missing from the list,
// or listed without a title, holds no title.
type ABSPIntegration struct {
	BaseURL    string
	HTTPClient *http.Client

	mu      sync.Mutex
	players map[string]abspPlayer // Key: member ID; loaded once per instance
	loadErr error                 // a failed load, not retried by this instance
}

// abspPlayer is one ABSP member of the ratings list.
type abspPlayer struct {
	MemberID string
	Name     string
	Title    string
}

// ABSPMemberInfo holds the parsed member information from the profile page
type ABSPMemberInfo struct {
	MemberID  string
	FirstName string
	LastName  string
}

// NewABSPIntegration creates a new ABSP integration instance
func NewABSPIntegration() *ABSPIntegration {
	return &ABSPIntegration{
		BaseURL: ABSPBaseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// abspMemberID returns the member ID for a membership number, or false when
// s is not one.
func abspMemberID(s string) (string, bool) {
	m := abspMemberIDRegex.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// abspName turns the ratings list's form of a name, in capitals with
// underscores for spaces, into the usual one: O'ROURKE to O'Rourke,
// DE_POITIERS to De Poitiers.
func abspName(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	out := []rune(strings.ToLower(s))
	for i, r := range out {
		if i == 0 || out[i-1] == ' ' || out[i-1] == '-' || out[i-1] == '\'' {
			out[i] = unicode.ToUpper(r)
		}
	}
	return string(out)
}

// parseABSPRatings reads current.txt and returns its members, keyed by
// member ID. Anything wrong in it (a line it cannot read, an unknown title,
// a repeated membership number, or no members at all) refuses the whole
// list, so that a broken list changes no one's title.
func parseABSPRatings(body []byte) (map[string]abspPlayer, error) {
	players := make(map[string]abspPlayer)
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for n := 1; scanner.Scan(); n++ {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		m := abspLineRegex.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("ABSP ratings list has a line it cannot read (line %d)", n)
		}
		lastName, firstName, memNo := m[1], m[2], m[5]
		title := ""
		if t := abspTitleRegex.FindStringSubmatch(lastName); t != nil {
			lastName, title = t[1], t[2]
			if _, known := TitleRegistry[OrgABSP][title]; !known {
				return nil, fmt.Errorf("ABSP ratings list has an unknown title %q (line %d)", title, n)
			}
		}
		if memNo == "" {
			continue // a nonmember or a lapsed member
		}
		id, _ := abspMemberID(memNo)
		if _, repeated := players[id]; repeated {
			return nil, fmt.Errorf("ABSP ratings list repeats membership number %s", memNo)
		}
		firstName = abspPlaceRegex.ReplaceAllString(firstName, "")
		players[id] = abspPlayer{
			MemberID: id,
			Name:     abspName(firstName) + " " + abspName(lastName),
			Title:    title,
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read ABSP ratings list: %w", err)
	}
	if len(players) == 0 {
		return nil, fmt.Errorf("ABSP ratings list has no members")
	}
	return players, nil
}

// load downloads and parses the ratings list, once per instance: a failure
// is kept too, so that a scheduled refresh of every linked user does not
// download a broken list again for each of them.
func (a *ABSPIntegration) load() (map[string]abspPlayer, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.players == nil && a.loadErr == nil {
		a.players, a.loadErr = a.download()
	}
	return a.players, a.loadErr
}

// download fetches and parses the ratings list.
func (a *ABSPIntegration) download() (map[string]abspPlayer, error) {
	req, err := http.NewRequest("GET", a.BaseURL+"/current.txt", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "woogles.io")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download ABSP ratings list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ABSP ratings list returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read ABSP ratings list: %w", err)
	}

	return parseABSPRatings(body)
}

// lookUp looks a member up in the ratings list. A member missing from it
// comes back with an empty RawTitle and FullName, not an error, so that a
// refresh takes away a title that is no longer listed.
func (a *ABSPIntegration) lookUp(memberID string) (*TitleInfo, error) {
	id, ok := abspMemberID(memberID)
	if !ok {
		return nil, fmt.Errorf("%q is not an ABSP membership number", memberID)
	}
	players, err := a.load()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	info := &TitleInfo{
		Organization:     OrgABSP,
		OrganizationName: OrganizationRegistry[OrgABSP].Name,
		MemberID:         id,
		LastFetched:      &now,
	}
	if p, ok := players[id]; ok {
		info.RawTitle = p.Title
		info.FullName = p.Name
	}
	return info, nil
}

// FetchTitle returns a member's title from the ratings list. With
// credentials, the member logs in first, and it is the membership number on
// their profile that is looked up, whatever memberID is; their name is the
// profile's, and a member missing from the list is linked holding no title.
// Without, memberID is looked up as it is, for an admin's link or a refresh.
func (a *ABSPIntegration) FetchTitle(memberID string, credentials map[string]string) (*TitleInfo, error) {
	if len(credentials) == 0 {
		return a.lookUp(memberID)
	}

	memberInfo, err := a.authenticate(credentials)
	if err != nil {
		return nil, err
	}
	info, err := a.lookUp(memberInfo.MemberID)
	if err != nil {
		return nil, err
	}
	info.FullName = strings.TrimSpace(memberInfo.FirstName + " " + memberInfo.LastName)
	return info, nil
}

// GetOrganizationCode returns the organization code
func (a *ABSPIntegration) GetOrganizationCode() OrganizationCode {
	return OrgABSP
}

// GetRealName returns the member's name: from their profile, with
// credentials, and otherwise from the ratings list, where a member missing
// from it has none, which is an error, so that an admin links only a listed
// member.
func (a *ABSPIntegration) GetRealName(memberID string, credentials map[string]string) (string, error) {
	info, err := a.FetchTitle(memberID, credentials)
	if err != nil {
		return "", err
	}
	if info.FullName == "" {
		return "", fmt.Errorf("membership number %s is not in the ABSP ratings list", info.MemberID)
	}
	return info.FullName, nil
}

// authenticate logs in to absp-database.org and reads the member's profile.
func (a *ABSPIntegration) authenticate(credentials map[string]string) (*ABSPMemberInfo, error) {
	username, ok := credentials["username"]
	if !ok {
		return nil, fmt.Errorf("username not provided")
	}
	password, ok := credentials["password"]
	if !ok {
		return nil, fmt.Errorf("password not provided")
	}
	memberInfo, err := a.authenticateAndFetchProfile(username, password)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return memberInfo, nil
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
		return nil, fmt.Errorf("authentication failed with status %d", resp.StatusCode)
	}

	// Read the response body to check for successful login
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for successful login indicator
	if !strings.Contains(string(body), "<p style='color:green'>ABSP Member</p>") {
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
	memberInfo, err := parseABSPProfilePage(string(profileBody))
	if err != nil {
		return nil, fmt.Errorf("failed to parse profile page: %w", err)
	}

	return memberInfo, nil
}

// parseABSPProfilePage extracts member information from the ABSP profile page HTML
func parseABSPProfilePage(html string) (*ABSPMemberInfo, error) {
	info := &ABSPMemberInfo{}

	// Pattern to match table cells with label and value
	// Format: <td>label:</td>\n<td>\nvalue</td>
	extractValue := func(label string) string {
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
