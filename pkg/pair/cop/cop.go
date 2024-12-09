package cop

import (
	"fmt"

	"golang.org/x/exp/rand"
	"google.golang.org/protobuf/encoding/protojson"

	"strings"
	"time"

	"github.com/woogles-io/liwords/pkg/matching"
	copdatapkg "github.com/woogles-io/liwords/pkg/pair/copdata"
	pkgstnd "github.com/woogles-io/liwords/pkg/pair/standings"
	"github.com/woogles-io/liwords/pkg/pair/verifyreq"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	timeFormat    = "2006-01-02T15:04:05.000Z"
	majorPenalty  = 1e9
	minorPenalty  = majorPenalty / 1e3
	byePlayerName = "BYE"
)

type policyArgs struct {
	req                      *pb.PairRequest
	copdata                  *copdatapkg.PrecompData
	playerNodes              []int
	prepairedRoundIdx        int
	lowestPossibleAbsCasher  int
	lowestPossibleHopeCasher int
	lowestPossibleHopeNth    []int
	roundsRemaining          int
	gibsonGetsBye            bool
}

type constraintPolicy struct {
	name    string
	handler func(*policyArgs) ([][2]int, [][2]int)
}

type weightPolicy struct {
	name    string
	handler func(*policyArgs, int, int) int64
}

var constraintPolicies = []constraintPolicy{
	{
		// Prepaired players
		name: "PP",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			if pargs.prepairedRoundIdx == -1 {
				return [][2]int{}, [][2]int{}
			}
			forcedPairings := [][2]int{}
			for playerIdx, reqOppIdx := range pargs.req.DivisionPairings[pargs.prepairedRoundIdx].Pairings {
				if int(reqOppIdx) < playerIdx {
					continue
				}
				oppIdx := int(reqOppIdx)
				if oppIdx == playerIdx {
					oppIdx = pkgstnd.ByePlayerIndex
				}
				forcedPairings = append(forcedPairings, [2]int{playerIdx, oppIdx})
			}
			return forcedPairings, [][2]int{}
		},
	},
	{
		// KOTH
		name: "KH",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			if pargs.roundsRemaining != 1 {
				return [][2]int{}, [][2]int{}
			}
			numPlayers := len(pargs.playerNodes)
			forcedPairings := [][2]int{}
			for playerRankIdx := 0; playerRankIdx < numPlayers-1; playerRankIdx++ {
				if pargs.lowestPossibleAbsCasher < playerRankIdx {
					break
				}
				if pargs.copdata.GibsonizedPlayers[playerRankIdx] || pargs.copdata.GibsonizedPlayers[playerRankIdx+1] {
					continue
				}
				forcedPairings = append(forcedPairings, [2]int{pargs.playerNodes[playerRankIdx], pargs.playerNodes[playerRankIdx+1]})
				playerRankIdx++
			}
			return forcedPairings, [][2]int{}
		},
	},
	{
		// KOTH Class Prizes
		name: "KC",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			if pargs.roundsRemaining != 1 {
				return [][2]int{}, [][2]int{}
			}
			numPlayers := len(pargs.playerNodes)
			// Do not consider the bye as a player in this case
			if pargs.playerNodes[numPlayers-1] == pkgstnd.ByePlayerIndex {
				numPlayers--
			}
			forcedPairings := [][2]int{}
			for classPrizesIdx, classPrizes := range pargs.req.ClassPrizes {
				classIdx := classPrizesIdx + 1
				availableClassPrizes := int(classPrizes)
				for pRankIdx := 0; pRankIdx <= pargs.lowestPossibleAbsCasher; pRankIdx++ {
					pIdx := pargs.copdata.Standings.GetPlayerIndex(pRankIdx)
					if int(pargs.req.PlayerClasses[pIdx]) == classIdx && pargs.copdata.LowestRankAbsolutely[pRankIdx] >= int(pargs.req.PlacePrizes) {
						availableClassPrizes--
					}
				}
				if availableClassPrizes < 1 {
					continue
				}
				ri := pargs.lowestPossibleAbsCasher + 1
				numPlayersAhead := 0
				playerToCatch := -1
				KOTHCumeGibsonSpread := int(pargs.req.GibsonSpread * 2)
				for {
					for ri < numPlayers && int(pargs.req.PlayerClasses[pargs.playerNodes[ri]]) != classIdx {
						ri++
					}
					rj := ri + 1
					for rj < numPlayers && int(pargs.req.PlayerClasses[pargs.playerNodes[rj]]) != classIdx {
						rj++
					}
					if rj >= numPlayers {
						break
					}
					if !pargs.copdata.Standings.CanCatch(1, KOTHCumeGibsonSpread, ri, rj) {
						ri = rj
						numPlayersAhead++
						if numPlayersAhead == availableClassPrizes {
							break
						}
						continue
					}
					placesRemaining := availableClassPrizes - numPlayersAhead
					if placesRemaining == 2 {
						playerToCatch = rj
					} else if placesRemaining == 1 {
						playerToCatch = ri
					} else if playerToCatch >= 0 && !pargs.copdata.Standings.CanCatch(1, KOTHCumeGibsonSpread, playerToCatch, rj) {
						break
					}
					forcedPairings = append(forcedPairings, [2]int{pargs.playerNodes[ri], pargs.playerNodes[rj]})
					numPlayersAhead += 2
					ri = rj + 1
				}
			}
			return forcedPairings, [][2]int{}
		},
	},
	{
		// Control loss
		name: "CL",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			if pargs.copdata.HighestControlLossRankIdx < 0 {
				return [][2]int{}, [][2]int{}
			}
			disallowedPairings := [][2]int{}
			numPlayers := len(pargs.playerNodes)
			for playerRankIdx := 1; playerRankIdx < numPlayers; playerRankIdx++ {
				if playerRankIdx == pargs.copdata.HighestControlLossRankIdx || playerRankIdx == pargs.copdata.HighestControlLossRankIdx-1 {
					continue
				}
				disallowedPairings = append(disallowedPairings, [2]int{pargs.playerNodes[0], pargs.playerNodes[playerRankIdx]})
			}
			return [][2]int{}, disallowedPairings
		},
	},
	{
		// Gibson groups
		name: "GG",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			numPlayers := len(pargs.playerNodes)
			disallowedPairings := [][2]int{}
			for pri := 0; pri < numPlayers; pri++ {
				for prj := pri + 1; prj < numPlayers; prj++ {
					if pargs.copdata.GibsonGroups[pri] != pargs.copdata.GibsonGroups[prj] {
						disallowedPairings = append(disallowedPairings, [2]int{pargs.playerNodes[pri], pargs.playerNodes[prj]})
					}
				}
			}
			return [][2]int{}, disallowedPairings
		},
	},
	{
		// Gibson Bye
		name: "GB",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			if !pargs.gibsonGetsBye {
				return [][2]int{}, [][2]int{}
			}
			disallowedPairings := [][2]int{}
			for pri := 0; pri < len(pargs.playerNodes); pri++ {
				if pargs.copdata.GibsonizedPlayers[pri] {
					continue
				}
				disallowedPairings = append(disallowedPairings, [2]int{pargs.playerNodes[pri], pkgstnd.ByePlayerIndex})
			}
			return [][2]int{}, disallowedPairings
		},
	},
}

var weightPolicies = []weightPolicy{
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
	if req.Seed == 0 {
		req.Seed = time.Now().Unix()
	}
	marshaler := protojson.MarshalOptions{
		Multiline:    true, // Enables pretty printing
		Indent:       "  ", // Sets the indentation level
		AllowPartial: true,
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

	copdata := copdatapkg.GetPrecompData(req, rand.New(rand.NewSource(uint64(req.Seed))), logsb)

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
		if playerRankIdx > lowestPossibleHopeNth[place] {
			lowestPossibleHopeNth[place] = playerRankIdx
		}
	}
	lowestPossibleHopeCasher = lowestPossibleHopeNth[int(req.PlacePrizes)-1]

	gibsonGetsBye := false
	if addBye {
		for i := 0; i < numPlayers; i++ {
			if copdata.GibsonizedPlayers[i] && copdata.GibsonGroups[i] == 0 {
				gibsonGetsBye = true
			}
		}
	}

	pargs := &policyArgs{
		req:                      req,
		copdata:                  copdata,
		playerNodes:              playerNodes,
		prepairedRoundIdx:        prepairedRoundIdx,
		lowestPossibleAbsCasher:  lowestPossibleAbsCasher,
		lowestPossibleHopeCasher: lowestPossibleHopeCasher,
		lowestPossibleHopeNth:    lowestPossibleHopeNth,
		roundsRemaining:          int(req.Rounds) - len(req.DivisionResults),
		gibsonGetsBye:            gibsonGetsBye,
	}

	logsb.WriteString(fmt.Sprintf("Control Loss Sims: %d\n", req.ControlLossSims))
	logsb.WriteString(fmt.Sprintf("Lowest Hopeful Casher: %s\n", req.PlayerNames[playerNodes[lowestPossibleHopeCasher]]))
	logsb.WriteString(fmt.Sprintf("Lowest Absolute Casher: %s\n", req.PlayerNames[playerNodes[lowestPossibleAbsCasher]]))
	logsb.WriteString(fmt.Sprintf("Rounds Remaining: %d\n", pargs.roundsRemaining))
	logsb.WriteString(fmt.Sprintf("Gibson Gets Bye: %t\n\n", pargs.gibsonGetsBye))

	numPlayerNodes := len(playerNodes)

	disallowedPairs := map[string]string{}
	for _, cPol := range constraintPolicies {
		forced, disallowed := cPol.handler(pargs)
		for _, dp := range disallowed {
			setDisallowPairs(disallowedPairs, dp[0], dp[1], cPol.name)
		}
		for _, fp := range forced {
			for pri := 0; pri < numPlayerNodes; pri++ {
				for prj := pri + 1; prj < numPlayerNodes; prj++ {
					pi := pargs.playerNodes[pri]
					pj := pargs.playerNodes[prj]
					if (fp[0] == pi && fp[1] == pj) || (fp[1] == pi && fp[0] == pj) {
						continue
					}
					if fp[0] == pi || fp[1] == pi || fp[0] == pj || fp[1] == pj {
						setDisallowPairs(disallowedPairs, pi, pj, cPol.name)
					}
				}
			}
		}
	}

	edges := []*matching.Edge{}

	matchupHeader := []string{"Player", "W", "S", "Player", "W", "S"}
	pairingDetails := [][]string{}
	pairingDetailsheader := append(matchupHeader, []string{"S", "C", "PTP", "Total"}...)
	for _, weightPolicy := range weightPolicies {
		pairingDetailsheader = append(pairingDetailsheader, weightPolicy.name)
	}
	numColums := len(pairingDetailsheader)
	pairingsToDetailsIndex := map[string]int{}
	for rankIdxI := 0; rankIdxI < numPlayerNodes; rankIdxI++ {
		for rankIdxJ := rankIdxI + 1; rankIdxJ < numPlayerNodes; rankIdxJ++ {
			pairingDataRow := getMatchupStrArray(divisionPlayerData, rankIdxI, rankIdxJ)
			pairKey := copdatapkg.GetPairingKey(playerNodes[rankIdxI], playerNodes[rankIdxJ])
			disallowReason, disallowPair := disallowedPairs[pairKey]
			// Pairing selected bool placeholder
			pairingDataRow = append(pairingDataRow, "")
			if disallowPair {
				pairingDataRow = append(pairingDataRow, disallowReason)
				emptyColsToAdd := numColums - len(pairingDataRow)
				for i := 0; i < emptyColsToAdd; i++ {
					pairingDataRow = append(pairingDataRow, "")
				}
			} else {
				// No disallow reason
				pairingDataRow = append(pairingDataRow, "")
				// Add the number of repeats for convenience
				pairingDataRow = append(pairingDataRow, fmt.Sprintf("%d", copdata.PairingCounts[pairKey]))
				// Placeholder for total weight
				pairingDataRow = append(pairingDataRow, "")
				weightSum := int64(0)
				for _, weightPolicy := range weightPolicies {
					weight := weightPolicy.handler(pargs, rankIdxI, rankIdxJ)
					weightSum += weight
					pairingDataRow = append(pairingDataRow, fmt.Sprintf("%d", weight))
				}
				pairingDataRow[len(matchupHeader)+3] = fmt.Sprintf("%d", weightSum)
				edges = append(edges, matching.NewEdge(rankIdxI, rankIdxJ, weightSum))
			}
			pairingsToDetailsIndex[getRankPairingKey(rankIdxI, rankIdxJ)] = len(pairingDetails)
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
		pairingDetails[pairingsToDetailsIndex[getRankPairingKey(playerRankIdx, oppRankIdx)]][len(matchupHeader)] = "*"
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

	copdatapkg.WriteStringDataToLog("Final Pairings", append(matchupHeader, []string{"Previous Times Played"}...), pairingsLogMx, logsb)

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
	if playerData[2] == byePlayerName {
		return []string{byePlayerName, "", ""}
	}
	return []string{fmt.Sprintf("%s (#%s) %s", playerData[0], playerData[1], playerData[2]), playerData[3], playerData[4]}
}

func getMatchupStrArray(divisionPlayerData [][]string, i int, j int) []string {
	return append(getPlayerRecordStrArray(divisionPlayerData[i]), getPlayerRecordStrArray(divisionPlayerData[j])...)
}

func setDisallowPairs(disallowedPairs map[string]string, playerIdx int, oppIdx int, policyName string) {
	key := copdatapkg.GetPairingKey(playerIdx, oppIdx)
	if _, exists := disallowedPairs[key]; !exists {
		disallowedPairs[key] = policyName
	}
}

func getRankPairingKey(playerRankIdx int, oppRankIdx int) string {
	return fmt.Sprintf("%d:%d", playerRankIdx, oppRankIdx)
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
