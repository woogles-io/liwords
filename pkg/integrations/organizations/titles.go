package organizations

import (
	"sort"
)

// GetHighestTitleFromInfos returns the TitleInfo with the highest ranked title
// Returns nil if the list is empty or no titles have ranks
// A tie between organizations goes by winsTitleTie, so the order of infos
// does not matter.
func GetHighestTitleFromInfos(infos []TitleInfo) *TitleInfo {
	if len(infos) == 0 {
		return nil
	}

	var highest *TitleInfo
	highestRank := 0

	for i := range infos {
		rank := GetTitleRank(infos[i].Organization, infos[i].RawTitle)
		if rank > highestRank ||
			(rank > 0 && rank == highestRank && winsTitleTie(infos[i].Organization, highest.Organization)) {
			highest = &infos[i]
			highestRank = rank
		}
	}

	return highest
}

// SortTitlesByRank sorts a slice of TitleInfo by title rank (highest first)
// Ties go by winsTitleTie, as in GetHighestTitleFromInfos.
func SortTitlesByRank(infos []TitleInfo) {
	sort.Slice(infos, func(i, j int) bool {
		rankI := GetTitleRank(infos[i].Organization, infos[i].RawTitle)
		rankJ := GetTitleRank(infos[j].Organization, infos[j].RawTitle)
		if rankI != rankJ {
			return rankI > rankJ
		}
		return winsTitleTie(infos[i].Organization, infos[j].Organization)
	})
}

// winsTitleTie reports whether organization a's title shows over an equally
// ranked one from organization b: the higher TiePriority, then the lower
// code, so that an organization left without a TiePriority still has a fixed
// place.
func winsTitleTie(a, b OrganizationCode) bool {
	pa := OrganizationRegistry[a].TiePriority
	pb := OrganizationRegistry[b].TiePriority
	if pa != pb {
		return pa > pb
	}
	return a < b
}

// GetHighestTitleDisplay returns the TitleDisplay for the highest ranked title
// Returns nil if no titles have display info
func GetHighestTitleDisplay(infos []TitleInfo) *TitleDisplay {
	highest := GetHighestTitleFromInfos(infos)
	if highest == nil {
		return nil
	}
	return GetTitleDisplay(highest.Organization, highest.RawTitle)
}
