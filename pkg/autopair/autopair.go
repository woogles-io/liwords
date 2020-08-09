package autopair

import (
	"errors"
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

	pairings, err := minWeightMatching(edges, true)
	if err != nil {
		// Log error here, be sure to record edges
		return nil, err
	}
	if len(pairings) != len(members) {
		// Log error here, be sure to record edges
		return nil, errors.New("Pairings and members are not the same length")
	}

	for index, pairing := range pairings {
		if pairing == -1 {
			members[index].Misses++
		}
	}

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
	return ratingDiff - missBonus - rangeBonus
}

func missBonus(p *entity.PoolMember) int {
	return min(p.Misses*12, 400)
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
