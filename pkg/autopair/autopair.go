package autopair

import (
	"github.com/domino14/liwords/pkg/entity"
)

func Autopair(members []*entity.PoolMember) ([]int, error) {
	numberOfMembers := len(members)
	edges := []*Edge{}
	for i := 0; i < numberOfMembers; i++ {
		for j := i + 1; j < numberOfMembers; j++ {
			PoolMemberA := members[i]
			PoolMemberB := members[j]
			if pairable(PoolMemberA, PoolMemberB) {
				weight := weigh(PoolMemberA, PoolMemberB)
				edges = append(edges, &Edge{i, j, weight})
			}
		}
	}

	pairings := minWeightMatching(edges, true)
	return pairings, nil
}

func pairable(PoolMemberA *entity.PoolMember, PoolMemberB *entity.PoolMember) bool {
	for _, blockedId := range PoolMemberA.Blocking {
		if PoolMemberB.Id == blockedId {
			return false
		}
	}
	for _, blockedId := range PoolMemberB.Blocking {
		if PoolMemberA.Id == blockedId {
			return false
		}
	}
	return true
}

func weigh(PoolMemberA *entity.PoolMember, PoolMemberB *entity.PoolMember) int {
	ratingDiff := abs(PoolMemberA.Rating - PoolMemberB.Rating)
	missBonus := min(missBonus(PoolMemberA), missBonus(PoolMemberB))
	rangeBonus := rangeBonus(PoolMemberA, PoolMemberB)
	sitBonus := sitBonus(PoolMemberA, PoolMemberB)
	return ratingDiff - missBonus - rangeBonus - sitBonus
}


func missBonus(p *entity.PoolMember) int {
	return min(p.Misses * 12, max((460 + min(p.SitCounter, -3) * 20), 0))
}

func rangeBonus(PoolMemberA *entity.PoolMember, PoolMemberB *entity.PoolMember) int {
	rangeBonus := 0
	if PoolMemberA.RatingRange[0] <= PoolMemberB.Rating &&
	   PoolMemberA.RatingRange[1] >= PoolMemberB.Rating &&
	   PoolMemberB.RatingRange[0] >= PoolMemberA.Rating &&
	   PoolMemberB.RatingRange[1] >= PoolMemberA.Rating {
	   	rangeBonus = 200
	   }
	return rangeBonus
}

func sitBonus(PoolMemberA *entity.PoolMember, PoolMemberB *entity.PoolMember) int {
	// match of good and bad player
	sitBonus := min(abs(PoolMemberA.SitCounter - PoolMemberB.SitCounter), 10) * 30
	if PoolMemberA.SitCounter >= -2 && PoolMemberB.SitCounter >= -2 {
		sitBonus = 30 // good players
	} else if PoolMemberA.SitCounter <= -10 && PoolMemberB.SitCounter <= -10 {
		sitBonus = 80 // very bad players
	} else if PoolMemberA.SitCounter <= -5 && PoolMemberB.SitCounter <= -5 {
		sitBonus = 30 // bad players
	}
	return sitBonus
}