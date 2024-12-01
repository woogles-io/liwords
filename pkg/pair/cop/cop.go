package cop

import (
	"fmt"
	"strings"
	"time"

	"github.com/woogles-io/liwords/pkg/matching"
	copdatapkg "github.com/woogles-io/liwords/pkg/pair/copdata"
	pkgstnd "github.com/woogles-io/liwords/pkg/pair/standings"
	"github.com/woogles-io/liwords/pkg/pair/verifyreq"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	timeFormat   = "2006-01-02T15:04:05.000Z"
	majorPenalty = 1e9
	minorPenalty = majorPenalty / 1e3
)

type policyArgs struct {
	req                      *pb.PairRequest
	copdata                  *copdatapkg.PrecompData
	playerNodes              []int
	kothPairedPlayers        map[int]bool
	prepairedRoundIdx        int
	lowestPossibleAbsCasher  int
	lowestPossibleHopeCasher int
	lowestPossibleHopeNth    []int
	roundsRemaining          int
	gibsonGetsBye            bool
}

type policy struct {
	name    string
	handler func(*policyArgs, int, int) int64
}

// Constraints and Weights:
// repeats
//  - back to back
//  - matchup repeats
//  - total repeats
//  - byes
// rank diff
//  - one can cash
//  - neither can cash
// pair with casher
//  - can catch
//  - cannot catch
// gibson - cashers
//  - gibsons should not player cashers
// *** the following should be constraints:
// gibson - groups
// gibson - bye
//  - gibsons should play byes first
// control loss
// koth
// prepaired

// Test logging:
// with and without bye
// with and without control loss

// For constraint policies:
// -1 means the pairing is not allowed
// 1 means the pairing is forced
// 0 means no constraint is applied
var constraintPolicies = []policy{
	{
		// Prepaired players
		name: "PP",
		handler: func(pargs *policyArgs, playerIRankIdx int, playerJRankIdx int) int64 {
			if pargs.prepairedRoundIdx == -1 {
				return 0
			}
			playerIPlayerIdx := pargs.playerNodes[playerIRankIdx]
			playerJPlayerIdx := pargs.playerNodes[playerJRankIdx]
			oppPlayerIdx := int(pargs.req.DivisionPairings[pargs.prepairedRoundIdx].Pairings[playerIPlayerIdx])
			if oppPlayerIdx == -1 {
				return 0
			}
			if oppPlayerIdx == playerJPlayerIdx ||
				(oppPlayerIdx == playerIPlayerIdx &&
					playerJPlayerIdx == pkgstnd.ByePlayerIndex) {
				return 1
			}
			return -1
		},
	},
	{
		// KOTH
		name: "KH",
		handler: func(pargs *policyArgs, playerIRankIdx int, playerJRankIdx int) int64 {
			// Only apply KOTH in the last round
			if pargs.roundsRemaining != 1 {
				return 0
			}
			// If either player has already been KOTH paired, this pairing is not allowed
			if pargs.kothPairedPlayers[playerIRankIdx] || pargs.kothPairedPlayers[playerJRankIdx] {
				return -1
			}
			if playerIRankIdx != playerJRankIdx-1 ||
				pargs.copdata.GibsonizedPlayers[playerIRankIdx] ||
				(playerIRankIdx > pargs.lowestPossibleAbsCasher) {
				return 0
			}
			pargs.kothPairedPlayers[playerIRankIdx] = true
			pargs.kothPairedPlayers[playerJRankIdx] = true
			return 1
		},
	},
	{
		// Control loss
		name: "CL",
		handler: func(pargs *policyArgs, playerIRankIdx int, playerJRankIdx int) int64 {
			if pargs.copdata.HighestControlLossRankIdx < 0 {
				return 0
			}
			playerJeligible := (playerJRankIdx == pargs.copdata.HighestControlLossRankIdx ||
				playerJRankIdx == pargs.copdata.HighestControlLossRankIdx-1)
			if playerIRankIdx == 0 && playerJeligible {
				return 0
			}
			if playerIRankIdx == 0 || playerJeligible {
				return -1
			}
			return 0
		},
	},
	{
		// Gibson groups
		name: "GG",
		handler: func(pargs *policyArgs, playerIRankIdx int, playerJRankIdx int) int64 {
			if pargs.copdata.GibsonGroups[playerIRankIdx] != pargs.copdata.GibsonGroups[playerJRankIdx] {
				return -1
			}
			return 0
		},
	},
	{
		// Gibson Bye
		name: "GB",
		handler: func(pargs *policyArgs, playerIRankIdx int, playerJRankIdx int) int64 {
			if !pargs.gibsonGetsBye || pargs.playerNodes[playerJRankIdx] != pkgstnd.ByePlayerIndex ||
				(pargs.copdata.GibsonizedPlayers[playerIRankIdx] && pargs.copdata.GibsonGroups[playerIRankIdx] == 0) {
				return 0
			}
			return -1
		},
	},
}

var weightPolicies = []policy{
	{
		// Rank diff
		name: "RD",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			diff := int64(rj - ri)
			if ri > pargs.lowestPossibleAbsCasher {
				return diff
			}
			return diff * diff * diff
		},
	},
	{
		// Pair with Casher
		name: "PC",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			if pargs.copdata.GibsonizedPlayers[ri] || pargs.copdata.GibsonizedPlayers[rj] ||
				ri > pargs.lowestPossibleHopeCasher {
				return 0
			}
			if rj <= pargs.lowestPossibleHopeNth[ri] ||
				(pargs.lowestPossibleHopeNth[ri] == ri && ri == rj-1) {
				casherDiff := pargs.lowestPossibleHopeNth[ri] - rj
				if casherDiff < 0 {
					casherDiff *= -1
				}
				return int64(intPow(casherDiff, 3) * 2)
			}
			return majorPenalty
		},
	},
	{
		// Gibson cashers
		name: "GC",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			if pargs.copdata.GibsonGroups[ri] != 0 || pargs.copdata.GibsonGroups[rj] != 0 ||
				(pargs.copdata.GibsonizedPlayers[ri] && pargs.copdata.GibsonizedPlayers[rj]) {
				return 0
			}
			if pargs.copdata.GibsonizedPlayers[ri] && rj <= pargs.lowestPossibleAbsCasher ||
				pargs.copdata.GibsonizedPlayers[rj] && ri <= pargs.lowestPossibleAbsCasher {
				return majorPenalty
			}
			return 0
		},
	},
	{
		// Repeats
		name: "RE",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			pi := pargs.playerNodes[ri]
			pj := pargs.playerNodes[rj]
			pairingKey := copdatapkg.GetPairingKey(pi, pj)
			timesPlayed := pargs.copdata.PairingCounts[pairingKey]
			if timesPlayed == 0 {
				return 0
			}
			return int64(intPow(timesPlayed, 2) * intPow(pargs.copdata.Standings.GetNumPlayers()/3, 3))
		},
	},
	{
		// Back-to-back repeats for non-cashers
		name: "BB",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			if ri <= pargs.lowestPossibleHopeCasher {
				return 0
			}
			prevRound := len(pargs.req.DivisionPairings) - 2
			if pargs.prepairedRoundIdx >= 0 {
				prevRound = pargs.prepairedRoundIdx - 1
			}
			if prevRound < 0 {
				return 0
			}
			pi := pargs.playerNodes[ri]
			pj := pargs.playerNodes[rj]
			if int(pargs.req.DivisionPairings[prevRound].Pairings[pi]) != pj {
				return 0
			}
			return minorPenalty
		},
	},
	{
		// Total repeats between both players
		name: "TR",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			pj := pargs.playerNodes[rj]
			if pj == pkgstnd.ByePlayerIndex {
				return 0
			}
			pi := pargs.playerNodes[ri]
			return int64(pargs.copdata.RepeatCounts[pi]+pargs.copdata.RepeatCounts[pj]) * 2
		},
	},
	{
		// Bye Repeats
		name: "BR",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			if pargs.req.AllowRepeatByes {
				return 0
			}
			pj := pargs.playerNodes[rj]
			if pj != pkgstnd.ByePlayerIndex {
				return 0
			}
			pi := pargs.playerNodes[ri]
			if pargs.copdata.PairingCounts[copdatapkg.GetPairingKey(pi, pj)] == 0 {
				return 0
			}
			return majorPenalty
		},
	},
}

func COPPair(req *pb.PairRequest) *pb.PairResponse {
	logsb := &strings.Builder{}
	starttime := time.Now()
	resp := copPairWithLog(req, logsb)
	endtime := time.Now()
	duration := endtime.Sub(starttime)
	if resp.ErrorCode != pb.PairError_SUCCESS {
		logsb.WriteString("\nCOP finished with error:\n\n" + resp.ErrorMessage + "\n\n")
	} else {
		logsb.WriteString("\nCOP finished successfully.\n\n")
	}
	logsb.WriteString(fmt.Sprintf("Started:  %s\nFinished: %s\nDuration: %s",
		starttime.Format(timeFormat), endtime.Format(timeFormat), duration))
	resp.Log = logsb.String()
	return resp
}

func copPairWithLog(req *pb.PairRequest, logsb *strings.Builder) *pb.PairResponse {
	req.Seed = int32(time.Now().Unix())
	marshaler := protojson.MarshalOptions{
		Multiline: true, // Enables pretty printing
		Indent:    "  ", // Sets the indentation level
	}
	jsonData, err := marshaler.Marshal(req)
	if err != nil {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_REQUEST_TO_JSON_FAILED,
			ErrorMessage: err.Error(),
		}
	}

	logsb.WriteString("Pairings Request:\n\n" + string(jsonData) + "\n\n")

	resp := verifyreq.Verify(req)
	if resp != nil {
		return resp
	}

	copdata := copdatapkg.GetPrecompData(req, logsb)

	pairings, resp := copMinWeightMatching(req, copdata, logsb)

	if resp != nil {
		return resp
	}

	return &pb.PairResponse{
		ErrorCode: pb.PairError_SUCCESS,
		Pairings:  pairings,
	}
}

func copMinWeightMatching(req *pb.PairRequest, copdata *copdatapkg.PrecompData, logsb *strings.Builder) ([]int32, *pb.PairResponse) {
	numPlayers := copdata.Standings.GetNumPlayers()
	playerNodes := []int{}
	divisionPlayerData := [][]string{}
	for i := 0; i < numPlayers; i++ {
		playerNodes = append(playerNodes, copdata.Standings.GetPlayerIndex(i))
		divisionPlayerData = append(divisionPlayerData, copdata.Standings.StringDataForPlayer(req, i))
	}

	addBye := numPlayers%2 == 1
	if addBye {
		playerNodes = append(playerNodes, pkgstnd.ByePlayerIndex)
		divisionPlayerData = append(divisionPlayerData, []string{"", "", "BYE", "", ""})
	}

	prepairedRoundIdx := -1
	numDivPairings := len(req.DivisionPairings)
	for _, oppIdx := range req.DivisionPairings[numDivPairings-1].Pairings {
		if oppIdx == -1 {
			prepairedRoundIdx = numDivPairings - 1
			break
		}
	}

	lowestPossibleAbsCasher := 0
	for playerRankIdx, place := range copdata.HighestRankAbsolutely {
		if place < int(req.PlacePrizes) {
			lowestPossibleAbsCasher = playerRankIdx
		}
	}

	lowestPossibleHopeCasher := 0
	lowestPossibleHopeNth := make([]int, len(copdata.HighestRankHopefully))
	for playerRankIdx, place := range copdata.HighestRankHopefully {
		if lowestPossibleHopeNth[place] < playerRankIdx {
			lowestPossibleHopeNth[place] = playerRankIdx
		}
		if place < int(req.PlacePrizes) {
			lowestPossibleHopeCasher = playerRankIdx
		}
	}

	numGibsonsInBaseGroup := 0
	for i := 0; i < numPlayers; i++ {
		if copdata.GibsonizedPlayers[i] && copdata.GibsonGroups[i] == 0 {
			numGibsonsInBaseGroup++
		}
	}

	pargs := &policyArgs{
		req:                      req,
		copdata:                  copdata,
		playerNodes:              playerNodes,
		prepairedRoundIdx:        prepairedRoundIdx,
		kothPairedPlayers:        map[int]bool{},
		lowestPossibleAbsCasher:  lowestPossibleAbsCasher,
		lowestPossibleHopeCasher: lowestPossibleHopeCasher,
		lowestPossibleHopeNth:    lowestPossibleHopeNth,
		roundsRemaining:          int(req.Rounds) - len(req.DivisionResults),
		gibsonGetsBye:            numGibsonsInBaseGroup%2 == 1,
	}

	logsb.WriteString(fmt.Sprintf("Lowest Hopeful Casher: %s\n", req.PlayerNames[playerNodes[lowestPossibleHopeCasher]]))
	logsb.WriteString(fmt.Sprintf("Lowest Absolute Casher: %s\n", req.PlayerNames[playerNodes[lowestPossibleAbsCasher]]))
	logsb.WriteString(fmt.Sprintf("Rounds Remaining: %d\n", pargs.roundsRemaining))
	logsb.WriteString(fmt.Sprintf("Gibson Gets Bye: %t\n\n", pargs.gibsonGetsBye))

	numPlayerNodes := len(playerNodes)

	edges := []*matching.Edge{}

	matchupHeader := []string{"Player", "W", "S", "Player", "W", "S"}
	pairingDetails := [][]string{}
	pairingDetailsheader := append(matchupHeader, []string{"S", "C", "Repeats", "Total"}...)
	for _, weightPolicy := range weightPolicies {
		pairingDetailsheader = append(pairingDetailsheader, weightPolicy.name)
	}
	numColums := len(pairingDetailsheader)
	pairingsToDetailsIndex := map[string]int{}
	for rankIdxI := 0; rankIdxI < numPlayerNodes; rankIdxI++ {
		for rankIdxJ := rankIdxI + 1; rankIdxJ < numPlayerNodes; rankIdxJ++ {
			pairingDataRow := getMatchupStrArray(divisionPlayerData, rankIdxI, rankIdxJ)
			constraintPolicyOutcome := int64(0)
			constraintPolicyName := ""
			for _, constraintPolicy := range constraintPolicies {
				constraintPolicyOutcome = constraintPolicy.handler(pargs, rankIdxI, rankIdxJ)
				if constraintPolicyOutcome != 0 {
					constraintPolicyName = constraintPolicy.name
					break
				}
			}
			pairingDataRow = append(pairingDataRow, "")
			pairingDataRow = append(pairingDataRow, constraintPolicyName)
			weightSum := int64(0)
			if constraintPolicyOutcome == 0 {
				// Add the number of repeats for convenience
				pairingDataRow = append(pairingDataRow, fmt.Sprintf("%d", copdata.PairingCounts[copdatapkg.GetPairingKey(playerNodes[rankIdxI], playerNodes[rankIdxJ])]))
				// Placeholder for total weight
				pairingDataRow = append(pairingDataRow, "")
				for _, weightPolicy := range weightPolicies {
					weight := weightPolicy.handler(pargs, rankIdxI, rankIdxJ)
					weightSum += weight
					pairingDataRow = append(pairingDataRow, fmt.Sprintf("%d", weight))
				}
				pairingDataRow[len(matchupHeader)+3] = fmt.Sprintf("%d", weightSum)
			} else {
				emptyColsToAdd := numColums - len(pairingDataRow)
				for i := 0; i < emptyColsToAdd; i++ {
					pairingDataRow = append(pairingDataRow, "")
				}
			}
			if constraintPolicyOutcome != -1 {
				edges = append(edges, matching.NewEdge(rankIdxI, rankIdxJ, weightSum))
			}
			pairingsToDetailsIndex[fmt.Sprintf("%d:%d", rankIdxI, rankIdxJ)] = len(pairingDetails)
			pairingDetails = append(pairingDetails, pairingDataRow)
		}
		if rankIdxI < numPlayerNodes-2 {
			spacingRow := make([]string, numColums)
			pairingDetails = append(pairingDetails, spacingRow)
		}
	}

	pairings, totalWeight, err := matching.MinWeightMatching(edges, true)

	if err != nil {
		return nil, &pb.PairResponse{
			ErrorCode:    pb.PairError_MIN_WEIGHT_MATCHING,
			ErrorMessage: fmt.Sprintf("min weight matching error: %s\n", err.Error()),
		}
	}

	if addBye {
		pairings = pairings[:len(pairings)-1]
	}

	if len(pairings) != numPlayers {
		return nil, &pb.PairResponse{
			ErrorCode:    pb.PairError_INVALID_PAIRINGS_LENGTH,
			ErrorMessage: fmt.Sprintf("invalid pairings length %d for %d players", len(pairings), numPlayers),
		}
	}

	for playerRankIdx, oppRankIdx := range pairings {
		if oppRankIdx < playerRankIdx {
			continue
		}
		pairingDetails[pairingsToDetailsIndex[fmt.Sprintf("%d:%d", playerRankIdx, oppRankIdx)]][len(matchupHeader)] = "*"
	}

	copdatapkg.WriteStringDataToLog("Pairing Weights", pairingDetailsheader, pairingDetails, logsb)

	unpairedRankIdxes := []int{}
	for playerRankIdx, oppRankIdx := range pairings {
		if oppRankIdx < 0 {
			unpairedRankIdxes = append(unpairedRankIdxes, playerRankIdx)
		}
	}

	if len(unpairedRankIdxes) > 0 {
		msg := "COP pairings could not be completed because there were too many constraints. The unpaired players by rank index are:\n\n"
		for idx, unpairedRankIdx := range unpairedRankIdxes {
			msg += fmt.Sprintf("%d", unpairedRankIdx+1)
			if idx < len(unpairedRankIdxes)-1 {
				msg += ", "
			}
		}
		return nil, &pb.PairResponse{
			ErrorCode:    pb.PairError_OVERCONSTRAINED,
			ErrorMessage: msg,
		}
	}

	pairingsLogMx := [][]string{}
	for playerRankIdx := 0; playerRankIdx < len(pairings); playerRankIdx++ {
		oppRankIdx := pairings[playerRankIdx]
		if oppRankIdx < playerRankIdx {
			continue
		}
		pairingsLogMxRow := getMatchupStrArray(divisionPlayerData, playerRankIdx, oppRankIdx)
		playerIdx := playerNodes[playerRankIdx]
		oppIdx := playerNodes[oppRankIdx]
		pairingsLogMxRow = append(pairingsLogMxRow, fmt.Sprintf("%d", copdata.PairingCounts[copdatapkg.GetPairingKey(playerIdx, oppIdx)]))
		pairingsLogMx = append(pairingsLogMx, pairingsLogMxRow)
	}

	copdatapkg.WriteStringDataToLog("Final Pairings", append(matchupHeader, []string{"Repeats"}...), pairingsLogMx, logsb)

	logsb.WriteString(fmt.Sprintf("Total Weight: %d\n", totalWeight))

	allPlayerPairings := make([]int32, req.AllPlayers)

	// Convert rank indexes to player indexes and convert the bye format from ByePlayerIndex to player index
	for playerRankIdx := 0; playerRankIdx < len(pairings); playerRankIdx++ {
		oppRankIdx := pairings[playerRankIdx]
		playerIdx := playerNodes[playerRankIdx]
		oppIdx := playerNodes[oppRankIdx]
		if oppIdx == pkgstnd.ByePlayerIndex {
			oppIdx = playerIdx
		}
		allPlayerPairings[playerIdx] = int32(oppIdx)
	}

	return allPlayerPairings, nil
}

func getPlayerRecordStrArray(playerData []string) []string {
	return []string{fmt.Sprintf("%s (#%s) %s", playerData[0], playerData[1], playerData[2]), playerData[3], playerData[4]}
}

func getMatchupStrArray(divisionPlayerData [][]string, i int, j int) []string {
	return append(getPlayerRecordStrArray(divisionPlayerData[i]), getPlayerRecordStrArray(divisionPlayerData[j])...)
}

func intPow(base, exp int) int {
	result := 1
	for {
		if exp&1 == 1 {
			result *= base
		}
		exp >>= 1
		if exp == 0 {
			break
		}
		base *= base
	}

	return result
}
