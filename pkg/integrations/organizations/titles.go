package organizations

import (
	"sort"
)

// GetHighestTitle returns the highest normalized title from a list of titles
func GetHighestTitle(titles []NormalizedTitle) NormalizedTitle {
	if len(titles) == 0 {
		return TitleNone
	}

	highest := TitleNone
	highestRank := 0

	for _, title := range titles {
		if rank, ok := TitleHierarchy[title]; ok && rank > highestRank {
			highest = title
			highestRank = rank
		}
	}

	return highest
}

// GetHighestTitleFromInfos returns the highest normalized title from a list of TitleInfo
func GetHighestTitleFromInfos(infos []TitleInfo) NormalizedTitle {
	titles := make([]NormalizedTitle, len(infos))
	for i, info := range infos {
		titles[i] = info.NormalizedTitle
	}
	return GetHighestTitle(titles)
}

// SortTitlesByRank sorts a slice of TitleInfo by normalized title rank (highest first)
func SortTitlesByRank(infos []TitleInfo) {
	sort.Slice(infos, func(i, j int) bool {
		rankI := TitleHierarchy[infos[i].NormalizedTitle]
		rankJ := TitleHierarchy[infos[j].NormalizedTitle]
		return rankI > rankJ
	})
}

// CompareTitles returns:
// -1 if title1 < title2
//  0 if title1 == title2
//  1 if title1 > title2
func CompareTitles(title1, title2 NormalizedTitle) int {
	rank1 := TitleHierarchy[title1]
	rank2 := TitleHierarchy[title2]

	if rank1 < rank2 {
		return -1
	} else if rank1 > rank2 {
		return 1
	}
	return 0
}
