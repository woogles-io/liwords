package organizations

import (
	"sort"
)

// GetHighestTitleFromInfos returns the TitleInfo with the highest ranked title
// Returns nil if the list is empty or no titles have ranks
func GetHighestTitleFromInfos(infos []TitleInfo) *TitleInfo {
	if len(infos) == 0 {
		return nil
	}

	var highest *TitleInfo
	highestRank := 0

	for i := range infos {
		rank := GetTitleRank(infos[i].Organization, infos[i].RawTitle)
		if rank > highestRank {
			highest = &infos[i]
			highestRank = rank
		}
	}

	return highest
}

// SortTitlesByRank sorts a slice of TitleInfo by title rank (highest first)
func SortTitlesByRank(infos []TitleInfo) {
	sort.Slice(infos, func(i, j int) bool {
		rankI := GetTitleRank(infos[i].Organization, infos[i].RawTitle)
		rankJ := GetTitleRank(infos[j].Organization, infos[j].RawTitle)
		return rankI > rankJ
	})
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
