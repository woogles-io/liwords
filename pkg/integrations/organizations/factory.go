package organizations

import (
	"fmt"
	"os"
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
