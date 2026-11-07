package organizations

import (
	"testing"

	"github.com/matryer/is"
)

func TestTiePriorityPerOrganization(t *testing.T) {
	is := is.New(t)
	seen := map[int]OrganizationCode{}
	for code, meta := range OrganizationRegistry {
		is.True(meta.TiePriority > 0) // every organization needs a TiePriority
		other, taken := seen[meta.TiePriority]
		is.True(!taken) // two organizations share a TiePriority
		if taken {
			t.Logf("%s and %s share TiePriority %d", code, other, meta.TiePriority)
		}
		seen[meta.TiePriority] = code
	}
}

func TestHighestTitleTies(t *testing.T) {
	is := is.New(t)
	for _, tc := range []struct {
		infos []TitleInfo
		org   OrganizationCode
		title string
	}{
		// equal ranks go to WESPA, then NASPA, then SG, then ABSP
		{[]TitleInfo{{Organization: OrgNASPA, RawTitle: "SM"}, {Organization: OrgWESPA, RawTitle: "M"}}, OrgWESPA, "M"},
		{[]TitleInfo{{Organization: OrgWESPA, RawTitle: "M"}, {Organization: OrgNASPA, RawTitle: "SM"}}, OrgWESPA, "M"},
		{[]TitleInfo{{Organization: OrgABSP, RawTitle: "EXP"}, {Organization: OrgNASPA, RawTitle: "EX"}}, OrgNASPA, "EX"},
		{[]TitleInfo{{Organization: OrgNASPA, RawTitle: "EX"}, {Organization: OrgABSP, RawTitle: "EXP"}}, OrgNASPA, "EX"},
		{[]TitleInfo{{Organization: OrgABSP, RawTitle: "GM"}, {Organization: OrgNASPA, RawTitle: "GM"}, {Organization: OrgWESPA, RawTitle: "GM"}}, OrgWESPA, "GM"},
		{[]TitleInfo{{Organization: OrgWESPA, RawTitle: "GM"}, {Organization: OrgABSP, RawTitle: "GM"}, {Organization: OrgNASPA, RawTitle: "GM"}}, OrgWESPA, "GM"},
		{[]TitleInfo{{Organization: OrgSG, RawTitle: "NM"}, {Organization: OrgWESPA, RawTitle: "IM"}}, OrgWESPA, "IM"},
		{[]TitleInfo{{Organization: OrgSG, RawTitle: "SM"}, {Organization: OrgNASPA, RawTitle: "SM"}}, OrgNASPA, "SM"},
		{[]TitleInfo{{Organization: OrgSG, RawTitle: "SM"}, {Organization: OrgWESPA, RawTitle: "M"}}, OrgWESPA, "M"},
		{[]TitleInfo{{Organization: OrgSG, RawTitle: "M"}, {Organization: OrgNASPA, RawTitle: "EX"}}, OrgNASPA, "EX"},
		{[]TitleInfo{{Organization: OrgABSP, RawTitle: "EXP"}, {Organization: OrgSG, RawTitle: "M"}}, OrgSG, "M"},
		{[]TitleInfo{{Organization: OrgABSP, RawTitle: "GM"}, {Organization: OrgSG, RawTitle: "GM"}}, OrgSG, "GM"},
		// different ranks: SG M (1) under WESPA M (2)
		{[]TitleInfo{{Organization: OrgSG, RawTitle: "M"}, {Organization: OrgWESPA, RawTitle: "M"}}, OrgWESPA, "M"},
		// a higher rank still wins
		{[]TitleInfo{{Organization: OrgWESPA, RawTitle: "M"}, {Organization: OrgABSP, RawTitle: "GM"}}, OrgABSP, "GM"},
		// an unranked title never shows
		{[]TitleInfo{{Organization: OrgWESPA, RawTitle: ""}, {Organization: OrgABSP, RawTitle: "EXP"}}, OrgABSP, "EXP"},
	} {
		highest := GetHighestTitleFromInfos(tc.infos)
		is.True(highest != nil)
		is.Equal(highest.Organization, tc.org)
		is.Equal(highest.RawTitle, tc.title)

		SortTitlesByRank(tc.infos)
		is.Equal(tc.infos[0].Organization, tc.org)
	}
	is.Equal(GetHighestTitleFromInfos(nil), nil)
	is.Equal(GetHighestTitleFromInfos([]TitleInfo{{Organization: OrgWESPA, RawTitle: "XX"}}), nil)
}

func TestTitleTieWithoutPriority(t *testing.T) {
	is := is.New(t)
	// codes that are not in OrganizationRegistry fall back to code order
	is.True(winsTitleTie("aa", "bb"))
	is.True(!winsTitleTie("bb", "aa"))
	is.True(winsTitleTie(OrgABSP, "aa"))
}
