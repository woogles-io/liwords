package organizations

import (
	"encoding/json"
	"time"
)

// NormalizedTitle represents the standardized title levels
type NormalizedTitle string

const (
	TitleGrandmaster NormalizedTitle = "Grandmaster"
	TitleMaster      NormalizedTitle = "Master"
	TitleExpert      NormalizedTitle = "Expert"
	TitleNone        NormalizedTitle = ""
)

// TitleHierarchy maps normalized titles to their ranking (higher is better)
var TitleHierarchy = map[NormalizedTitle]int{
	TitleGrandmaster: 3,
	TitleMaster:      2,
	TitleExpert:      1,
	TitleNone:        0,
}

// OrganizationCode represents the code for a scrabble organization
type OrganizationCode string

const (
	OrgNASPA OrganizationCode = "naspa"
	OrgWESPA OrganizationCode = "wespa"
	OrgABSP  OrganizationCode = "absp"
)

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
		HasAPI:               false,
		RequiresAuth:         false,
		RequiresVerification: true,
	},
	OrgABSP: {
		Code:                 OrgABSP,
		Name:                 "ABSP",
		HasAPI:               true,
		RequiresAuth:         true,  // Uses login to absp-database.org
		RequiresVerification: false,
	},
}

// OrganizationIntegrationData represents the JSONB data stored in integrations table
type OrganizationIntegrationData struct {
	MemberID             string          `json:"member_id"`
	FullName             string          `json:"full_name"`
	EncryptedCredentials string          `json:"encrypted_credentials,omitempty"`
	Verified             bool            `json:"verified"`
	VerificationMethod   string          `json:"verification_method"` // "api", "manual", "admin"
	VerifiedAt           *time.Time      `json:"verified_at,omitempty"`
	VerifiedBy           string          `json:"verified_by,omitempty"` // UUID of verifier
	RawTitle             string          `json:"raw_title"`
	NormalizedTitle      NormalizedTitle `json:"normalized_title"`
	LastFetched          *time.Time      `json:"last_fetched,omitempty"`
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
	NormalizedTitle  NormalizedTitle
	MemberID         string
	FullName         string
	LastFetched      *time.Time
}

// OrganizationIntegration is the interface that all organization integrations must implement
type OrganizationIntegration interface {
	// FetchTitle fetches the title for a user from the organization's API
	FetchTitle(memberID string, credentials map[string]string) (*TitleInfo, error)

	// NormalizeTitle converts a raw title from the organization to a normalized title
	NormalizeTitle(rawTitle string) NormalizedTitle

	// GetOrganizationCode returns the organization code
	GetOrganizationCode() OrganizationCode

	// GetRealName fetches the user's real name from the organization
	// For orgs with APIs (NASPA), credentials are required
	// For orgs without APIs (WESPA), credentials can be nil and data is scraped from public pages
	GetRealName(memberID string, credentials map[string]string) (string, error)
}
