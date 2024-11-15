package copdata

import (
	"fmt"
	"math"
	"strings"

	pkgstnd "github.com/woogles-io/liwords/pkg/pair/standings"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type PrecompData struct {
	standings                 *pkgstnd.Standings
	pairingCounts             map[string]int
	repeatCounts              []int
	highestRankHopefully      []int
	highestRankAbsolutely     []int
	highestControlLossRankIdx int
	gibsonGroups              []int
}

// Exported functions

func GetPrecompData(req *pb.PairRequest, logsb *strings.Builder) *PrecompData {
	standings := pkgstnd.CreateInitialStandings(req)

	// Use the initial results to get a tighter bound on the maximum factor
	initialSimResults := standings.SimFactorPairAll(req, int(req.DivisionSims), int(req.AllPlayers), false, logsb)

	numPlayers := len(initialSimResults.FinalRanks)
	// The maximum factor should be with respect to the highest non-gibsonized rank
	highestNongibsonizedRank := 0
	for rankIdx, isGibsonized := range initialSimResults.GibsonizedPlayers {
		if !isGibsonized {
			highestNongibsonizedRank = rankIdx
			break
		}
	}

	minWinsForHopeful := int(math.Round(float64(req.DivisionSims) * req.HopefulnessThreshold))
	maxFactor := 1
	for i := highestNongibsonizedRank + 1; i < numPlayers; i++ {
		numReachedHighestNongibsonizedRank := 0
		for j := 0; j <= numReachedHighestNongibsonizedRank; j++ {
			numReachedHighestNongibsonizedRank += initialSimResults.FinalRanks[i][j]
		}
		if numReachedHighestNongibsonizedRank >= minWinsForHopeful {
			maxFactor++
		} else {
			break
		}
	}

	useControlLoss := req.UseControlLoss && !initialSimResults.GibsonizedPlayers[0]

	improvedFactorSimResults := standings.SimFactorPairAll(req, int(req.DivisionSims), maxFactor, useControlLoss, logsb)

	highestRankHopefully := make([]int, numPlayers)
	highestRankAbsolutely := make([]int, numPlayers)
	for playerRankIdx := 0; playerRankIdx < numPlayers; playerRankIdx++ {
		winsSum := 0
		hopefulRank := numPlayers - 1
		absoluteRank := numPlayers - 1
		for rank := 0; rank < numPlayers; rank++ {
			winsSum += improvedFactorSimResults.FinalRanks[playerRankIdx][rank]
			if winsSum > 0 {
				absoluteRank = rank
				if winsSum >= minWinsForHopeful {
					hopefulRank = rank
					break
				}
			}
		}
		highestRankHopefully[playerRankIdx] = hopefulRank
		highestRankAbsolutely[playerRankIdx] = absoluteRank
	}

	// FIXME: calculate control loss from tourney wins

	pairingCounts := make(map[string]int)
	repeatCounts := make([]int, numPlayers)
	for _, roundPairings := range req.DivisionPairings {
		for playerIdx := 0; playerIdx < len(roundPairings.Pairings); playerIdx++ {
			oppIdx := int(roundPairings.Pairings[playerIdx])
			var pairingKey string
			if playerIdx == oppIdx {
				// FIXME: keymakers should be their own functions
				pairingKey = fmt.Sprintf("%d:BYE", playerIdx)
			} else if playerIdx < oppIdx {
				pairingKey = fmt.Sprintf("%d:%d", playerIdx, oppIdx)
			}
			if pairingCounts[pairingKey] > 0 {
				repeatCounts[playerIdx]++
				if playerIdx != oppIdx {
					repeatCounts[oppIdx]++
				}
			}
			pairingCounts[pairingKey]++
		}
	}

	return &PrecompData{
		standings:                 standings,
		pairingCounts:             pairingCounts,
		repeatCounts:              repeatCounts,
		highestRankHopefully:      highestRankHopefully,
		highestRankAbsolutely:     highestRankAbsolutely,
		highestControlLossRankIdx: -1,
		gibsonGroups:              improvedFactorSimResults.GibsonGroups,
	}
}
