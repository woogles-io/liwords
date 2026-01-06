package organizations

import (
	"encoding/json"
	"strings"
	"time"
)

// OrganizationCode represents the code for a scrabble organization
type OrganizationCode string

const (
	OrgNASPA OrganizationCode = "naspa"
	OrgWESPA OrganizationCode = "wespa"
	OrgABSP  OrganizationCode = "absp"
)

// TitleDisplay contains display information for a title
type TitleDisplay struct {
	Abbreviation string
	FullName     string
	Color        string
	Rank         int // Higher is better
}

// TitleRegistry maps (orgCode, rawTitle) to display information
// Raw titles are stored uppercase for matching
var TitleRegistry = map[OrganizationCode]map[string]TitleDisplay{
	OrgNASPA: {
		"GM": {Abbreviation: "GM", FullName: "Grandmaster", Color: "gold", Rank: 4},
		"SM": {Abbreviation: "SM", FullName: "NASPA Master", Color: "blue", Rank: 2},
		"EX": {Abbreviation: "EX", FullName: "Expert", Color: "green", Rank: 1},
	},
	OrgWESPA: {
		"GM": {Abbreviation: "GM", FullName: "Grandmaster", Color: "gold", Rank: 4},
		"IM": {Abbreviation: "IM", FullName: "International Master", Color: "purple", Rank: 3},
		"M":  {Abbreviation: "M", FullName: "Master", Color: "blue", Rank: 2},
	},
	OrgABSP: {
		"GM":  {Abbreviation: "GM", FullName: "Grandmaster", Color: "gold", Rank: 4},
		"EXP": {Abbreviation: "EXP", FullName: "Expert", Color: "green", Rank: 1},
	},
}

// GetTitleDisplay returns display information for a title
func GetTitleDisplay(orgCode OrganizationCode, rawTitle string) *TitleDisplay {
	if rawTitle == "" {
		return nil
	}

	orgTitles, ok := TitleRegistry[orgCode]
	if !ok {
		return nil
	}

	// Normalize the raw title for lookup (uppercase, trimmed)
	normalized := strings.ToUpper(strings.TrimSpace(rawTitle))

	display, ok := orgTitles[normalized]
	if !ok {
		return nil
	}

	return &display
}

// GetTitleRank returns the rank for a title (0 if no title or unknown)
func GetTitleRank(orgCode OrganizationCode, rawTitle string) int {
	display := GetTitleDisplay(orgCode, rawTitle)
	if display == nil {
		return 0
	}
	return display.Rank
}

// OrganizationMetadata contains information about an organization
type OrganizationMetadata struct {
	Code                 OrganizationCode
	Name                 string
	HasAPI               bool
	RequiresAuth         bool
	RequiresVerification bool
}

// OrganizationRegistry contains metadata for all supported organizations
var OrganizationRegistry = map[OrganizationCode]OrganizationMetadata{
	OrgNASPA: {
		Code:                 OrgNASPA,
		Name:                 "NASPA",
		HasAPI:               true,
		RequiresAuth:         true,
		RequiresVerification: false,
	},
	OrgWESPA: {
		Code:                 OrgWESPA,
		Name:                 "WESPA",
		HasAPI:               true, // Uses titlists with HTML scraping (no auth required)
		RequiresAuth:         false,
		RequiresVerification: true,
	},
	OrgABSP: {
		Code:                 OrgABSP,
		Name:                 "ABSP",
		HasAPI:               true,
		RequiresAuth:         true, // Uses login to absp-database.org
		RequiresVerification: false,
	},
}

// OrganizationIntegrationData represents the JSONB data stored in integrations table
type OrganizationIntegrationData struct {
	MemberID             string     `json:"member_id"`
	FullName             string     `json:"full_name"`
	EncryptedCredentials string     `json:"encrypted_credentials,omitempty"`
	Verified             bool       `json:"verified"`
	VerificationMethod   string     `json:"verification_method"` // "api", "manual", "admin"
	VerifiedAt           *time.Time `json:"verified_at,omitempty"`
	VerifiedBy           string     `json:"verified_by,omitempty"` // UUID of verifier
	RawTitle             string     `json:"raw_title"`
	LastFetched          *time.Time `json:"last_fetched,omitempty"`
}

// MarshalJSON serializes OrganizationIntegrationData to JSON
func (d *OrganizationIntegrationData) ToJSON() ([]byte, error) {
	return json.Marshal(d)
}

// FromJSON deserializes OrganizationIntegrationData from JSON
func (d *OrganizationIntegrationData) FromJSON(data []byte) error {
	return json.Unmarshal(data, d)
}

// TitleInfo represents title information for display
type TitleInfo struct {
	Organization     OrganizationCode
	OrganizationName string
	RawTitle         string
	MemberID         string
	FullName         string
	LastFetched      *time.Time
}

// OrganizationIntegration is the interface that all organization integrations must implement
type OrganizationIntegration interface {
	// FetchTitle fetches the title for a user from the organization's API
	FetchTitle(memberID string, credentials map[string]string) (*TitleInfo, error)

	// GetOrganizationCode returns the organization code
	GetOrganizationCode() OrganizationCode

	// GetRealName fetches the user's real name from the organization
	// For orgs with APIs (NASPA), credentials are required
	// For orgs without APIs (WESPA), credentials can be nil and data is scraped from public pages
	GetRealName(memberID string, credentials map[string]string) (string, error)
}

// GetTitleAbbreviation returns the abbreviation for a given organization and full title name
// Returns empty string if not found
func GetTitleAbbreviation(orgCode OrganizationCode, fullName string) string {
	// Get titles for this organization
	orgTitles, exists := TitleRegistry[orgCode]
	if !exists {
		return ""
	}

	// Find the matching title by full name
	for _, display := range orgTitles {
		if display.FullName == fullName {
			return display.Abbreviation
		}
	}
	return ""
}
