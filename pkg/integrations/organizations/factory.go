package organizations

import (
	"fmt"
	"os"
	"sort"
)

// GetIntegration returns the appropriate integration for the given organization code
func GetIntegration(orgCode OrganizationCode) (OrganizationIntegration, error) {
	switch orgCode {
	case OrgNASPA:
		baseURL := os.Getenv("NASPA_BASE_URL")
		if baseURL == "" {
			baseURL = "https://scrabbleplayers.org" // default
		}
		return NewNASPAIntegration(baseURL), nil

	case OrgWESPA:
		return NewWESPAIntegration(), nil

	case OrgABSP:
		return NewABSPIntegration(), nil

	case OrgSG:
		sg := NewSGIntegration()
		if titlesURL := os.Getenv("SG_TITLES_URL"); titlesURL != "" {
			sg.TitlesURL = titlesURL
		}
		return sg, nil

	default:
		return nil, fmt.Errorf("unsupported organization: %s", orgCode)
	}
}

// GetOrganizationMetadata returns metadata for an organization
func GetOrganizationMetadata(orgCode OrganizationCode) (OrganizationMetadata, error) {
	meta, ok := OrganizationRegistry[orgCode]
	if !ok {
		return OrganizationMetadata{}, fmt.Errorf("unknown organization: %s", orgCode)
	}
	return meta, nil
}

// ListSupportedOrganizations returns a list of all supported organization codes
func ListSupportedOrganizations() []OrganizationCode {
	orgs := make([]OrganizationCode, 0, len(OrganizationRegistry))
	for code := range OrganizationRegistry {
		orgs = append(orgs, code)
	}
	return orgs
}

// SupportedOrganizationCodes returns the codes of OrganizationRegistry as
// strings, sorted, for the queries that must leave out other integrations
// kept in the same table (such as patreon and twitch)
func SupportedOrganizationCodes() []string {
	codes := make([]string, 0, len(OrganizationRegistry))
	for code := range OrganizationRegistry {
		codes = append(codes, string(code))
	}
	sort.Strings(codes)
	return codes
}
