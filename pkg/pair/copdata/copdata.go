package copdata

import (
	"fmt"
	"strings"

	pkgstnd "github.com/woogles-io/liwords/pkg/pair/standings"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type PrecompData struct {
	standings         *pkgstnd.Standings
	pairingCounts     map[string]int
	repeatCounts      []int
	gibsonizedPlayers []bool
}

// Exported functions

func GetPrecompData(req *pb.PairRequest, logsb *strings.Builder) *PrecompData {
	standings := pkgstnd.CreateInitialStandings(req)

	logsb.WriteString("\n\nInitial Standings:\n\n" + standings.String(req))

	pairingCounts, repeatCounts := getPairingFrequencies(req)

	gibsonizedPlayers := getGibsonizedPlayers(req, standings)

	_ = standings.SimFactorPair(int(req.DivisionSims), int(req.Players), int(req.Rounds)-len(req.DivisionResults), gibsonizedPlayers)

	return &PrecompData{
		standings:         standings,
		pairingCounts:     pairingCounts,
		repeatCounts:      repeatCounts,
		gibsonizedPlayers: gibsonizedPlayers,
	}
}

// Unexported functions

func getPairingFrequencies(req *pb.PairRequest) (map[string]int, []int) {
	pairingCounts := make(map[string]int)
	totalRepeats := make([]int, req.Players)
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
				totalRepeats[playerIdx]++
				if playerIdx != oppIdx {
					totalRepeats[oppIdx]++
				}
			}
			pairingCounts[pairingKey]++
		}
	}
	return pairingCounts, totalRepeats
}
