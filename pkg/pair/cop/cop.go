package cop

import (
	"fmt"
	"hash/fnv"
	"math"

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
	timeFormat                           = "2006-01-02T15:04:05.000Z"
	majorPenalty                         = 1e9
	minorPenalty                         = majorPenalty / 1e3
	byePlayerName                        = "BYE"
	controlLossLowestContenderOnlyRounds = 4
)

type policyArgs struct {
	req                      *pb.PairRequest
	copdata                  *copdatapkg.PrecompData
	playerNodes              []int
	lowestPossibleAbsCasher  int
	lowestPossibleHopeCasher int
	roundsRemaining          int
	gibsonGetsBye            bool
	prepairedRoundIdx        int
	prepairedPlayerIndexes   map[int]int
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
			numPlayers := len(pargs.playerNodes)
			disallowedPairings := [][2]int{}
			for playerIdx := range pargs.prepairedPlayerIndexes {
				for i := 0; i < numPlayers; i++ {
					disallowedPairings = append(disallowedPairings, [2]int{playerIdx, pargs.playerNodes[i]})
				}
			}
			return [][2]int{}, disallowedPairings
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
				pi := pargs.playerNodes[playerRankIdx]
				pj := pargs.playerNodes[playerRankIdx+1]
				if pi == pkgstnd.ByePlayerIndex || pj == pkgstnd.ByePlayerIndex {
					continue
				}
				if pargs.copdata.GibsonizedPlayers[playerRankIdx] || pargs.copdata.GibsonizedPlayers[playerRankIdx+1] {
					continue
				}
				forcedPairings = append(forcedPairings, [2]int{pi, pj})
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
			if pargs.copdata.DestinysChild < 0 {
				return [][2]int{}, [][2]int{}
			}
			disallowedPairings := [][2]int{}
			numPlayers := len(pargs.playerNodes)
			for playerRankIdx := 1; playerRankIdx < numPlayers; playerRankIdx++ {
				// This if statement implements the following logic:
				//
				// Force pair the player in first with the bottom contender if:
				//  - There are controlLossLowestContenderOnlyRounds or fewer rounds remaining, OR
				//  - This is the first round where control loss is active
				// Force pair the player in first with the bottom 2 contenders:
				//  - otherwise
				if playerRankIdx == pargs.copdata.DestinysChild ||
					(pargs.roundsRemaining > controlLossLowestContenderOnlyRounds && int(pargs.req.ControlLossActivationRound) != pargs.copdata.CompletePairings &&
						playerRankIdx == pargs.copdata.DestinysChild-1) {
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
			// Do not consider the bye as a player in this case
			if pargs.playerNodes[numPlayers-1] == pkgstnd.ByePlayerIndex {
				numPlayers--
			}
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
			numPlayers := len(pargs.playerNodes)
			// Do not consider the bye as a player in this case
			if pargs.playerNodes[numPlayers-1] == pkgstnd.ByePlayerIndex {
				numPlayers--
			}
			for pri := 0; pri < numPlayers; pri++ {
				if pargs.copdata.GibsonizedPlayers[pri] {
					continue
				}
				disallowedPairings = append(disallowedPairings, [2]int{pargs.playerNodes[pri], pkgstnd.ByePlayerIndex})
			}
			return [][2]int{}, disallowedPairings
		},
	},
	{
		// Top Down Byes
		name: "TB",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			numPlayers := len(pargs.playerNodes)
			if !pargs.req.TopDownByes || pargs.playerNodes[numPlayers-1] != pkgstnd.ByePlayerIndex {
				return [][2]int{}, [][2]int{}
			}
			forcedBye := [][2]int{{-1, pkgstnd.ByePlayerIndex}}
			leastByes := int(pargs.req.Rounds + 1)
			// Use numPlayers - 1 to exclude the bye
			for playerRankIdx := range numPlayers - 1 {
				pi := pargs.playerNodes[playerRankIdx]
				pairingKey := copdatapkg.GetPairingKey(pi, pkgstnd.ByePlayerIndex)
				numByes := pargs.copdata.PairingCounts[pairingKey]
				if numByes < leastByes {
					leastByes = numByes
					forcedBye[0][0] = pi
				}
			}
			return forcedBye, [][2]int{}
		},
	},
}

func getLowestCasherIndex(pargs *policyArgs) int {
	return int(pargs.req.PlacePrizes) - 1
}

var weightPolicies = []weightPolicy{
	{
		// Rank diff
		name: "RD",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			diff := int64(rj - ri)
			// rj might be the Bye, which is out of range for this array
			rjGibsonized := false
			if rj < len(pargs.copdata.GibsonizedPlayers) {
				rjGibsonized = pargs.copdata.GibsonizedPlayers[rj]
			}
			// If
			//
			// - either play is gibsonized, or
			// - neither player cashed even once in the simulation, then
			//
			// the rank difference should squared
			// so it doesn't overwhelm the repeat penalty.
			if pargs.copdata.GibsonizedPlayers[ri] || rjGibsonized ||
				ri >= int(pargs.req.PlacePrizes) {
				return diff * diff
			}
			return diff * diff * diff
		},
	},
	{
		// Pair with Casher
		name: "PC",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			// rj might be the Bye, which is out of range for this array
			rjGibsonized := false
			if rj < len(pargs.copdata.GibsonizedPlayers) {
				rjGibsonized = pargs.copdata.GibsonizedPlayers[rj]
			}
			if pargs.copdata.GibsonizedPlayers[ri] || rjGibsonized || ri > pargs.lowestPossibleHopeCasher {
				return 0
			}
			// Distance is ceil(numPlayers/3)
			lowestPCIndex := getLowestCasherIndex(pargs)
			if ri > lowestPCIndex {
				dist := int(pargs.copdata.Standings.GetNumPlayers()+2) / 3
				if rj-ri <= dist {
					return 0
				}
				return majorPenalty
			}
			if rj <= pargs.copdata.LowestPossibleHopeNth[ri] ||
				(pargs.copdata.LowestPossibleHopeNth[ri] == ri && ri == rj-1) {
				casherDiff := pargs.copdata.LowestPossibleHopeNth[ri] - rj
				if casherDiff < 0 {
					casherDiff *= -1
				}
				return int64(math.Pow(float64(casherDiff), 3) * 2)
			}
			return majorPenalty
		},
	},
	{
		// Gibson cashers
		name: "GC",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			// rj might be the Bye, which is out of range for these arrays
			rjGibsonGroup := 0
			if rj < len(pargs.copdata.GibsonGroups) {
				rjGibsonGroup = pargs.copdata.GibsonGroups[rj]
			}
			rjGibsonized := false
			if rj < len(pargs.copdata.GibsonizedPlayers) {
				rjGibsonized = pargs.copdata.GibsonizedPlayers[rj]
			}
			if pargs.copdata.GibsonGroups[ri] != 0 || rjGibsonGroup != 0 ||
				(pargs.copdata.GibsonizedPlayers[ri] && rjGibsonized) {
				return 0
			}
			if pargs.copdata.GibsonizedPlayers[ri] && rj <= pargs.lowestPossibleAbsCasher ||
				rjGibsonized && ri <= pargs.lowestPossibleAbsCasher {
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
			unitWeight := int64(2 * int(math.Pow(float64(pargs.copdata.Standings.GetNumPlayers())/3.0, 3)))
			totalWeight := int64(timesPlayed) * unitWeight
			// If both players are outside of cash, add an extra unit weight
			// to the repeat weight.
			if ri > getLowestCasherIndex(pargs) {
				totalWeight += unitWeight
			}
			return totalWeight
		},
	},
	{
		// Back-to-back repeats for non-cashers
		name: "BB",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			if ri <= pargs.lowestPossibleHopeCasher {
				return 0
			}
			mostRecentCompletedRound := pargs.copdata.CompletePairings - 1
			if pargs.prepairedRoundIdx >= 0 {
				mostRecentCompletedRound = pargs.prepairedRoundIdx - 1
			}
			if mostRecentCompletedRound < 0 {
				return 0
			}
			pi := pargs.playerNodes[ri]
			pj := pargs.playerNodes[rj]
			if int(pargs.req.DivisionPairings[mostRecentCompletedRound].Pairings[pi]) != pj {
				return 0
			}
			return minorPenalty
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
			numTimesWithBye := pargs.copdata.PairingCounts[copdatapkg.GetPairingKey(pi, pj)]
			if numTimesWithBye == 0 {
				return 0
			}
			return majorPenalty * int64(numTimesWithBye)
		},
	},
}

func addPairRequestAsJSONToLog(req *pb.PairRequest, logsb *strings.Builder, includeResultsAndPairings bool) {
	divisionPairings := req.DivisionPairings
	divisionResults := req.DivisionResults
	playerClasses := req.PlayerClasses
	playerNames := req.PlayerNames
	if !includeResultsAndPairings {
		req.DivisionPairings = nil
		req.DivisionResults = nil
		req.PlayerClasses = nil
		req.PlayerNames = nil
	}
	marshaler := protojson.MarshalOptions{
		Multiline:    true, // Enables pretty printing
		Indent:       "  ", // Sets the indentation level
		AllowPartial: true,
	}
	jsonData, err := marshaler.Marshal(req)
	if err != nil {
		logsb.WriteString("error writing pair request to log: " + err.Error() + "\n\n")
		return
	}
	if !includeResultsAndPairings {
		logsb.WriteString("Abridged pair request:\n\n")
	} else {
		logsb.WriteString("\n\nPair request:\n\n")
	}
	logsb.Write(jsonData)
	logsb.WriteString("\n\n")
	req.DivisionPairings = divisionPairings
	req.DivisionResults = divisionResults
	req.PlayerClasses = playerClasses
	req.PlayerNames = playerNames
}

func COPPair(req *pb.PairRequest) *pb.PairResponse {
	logsb := &strings.Builder{}
	starttime := time.Now()
	if req.Seed == 0 {
		// Create a seed from the player names and the length of the pairings
		hash := fnv.New64()
		for _, name := range req.PlayerNames {
			_, _ = hash.Write([]byte(name))
		}
		req.Seed = int64(hash.Sum64()) + int64(len(req.DivisionPairings))
	}
	addPairRequestAsJSONToLog(req, logsb, false)
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
	addPairRequestAsJSONToLog(req, logsb, true)
	resp.Log = logsb.String()
	return resp
}

func copPairWithLog(req *pb.PairRequest, logsb *strings.Builder) *pb.PairResponse {
	resp := verifyreq.Verify(req)
	if resp != nil {
		return resp
	}

	copdata, pairErr := copdatapkg.GetPrecompData(req, rand.New(rand.NewSource(uint64(req.Seed))), logsb)

	if pairErr != pb.PairError_SUCCESS {
		return &pb.PairResponse{
			ErrorCode:    pairErr,
			ErrorMessage: fmt.Sprintf("error computing required inputs: %s", pb.PairError_name[int32(pairErr)]),
		}
	}

	pairings, resp := copMinWeightMatching(req, copdata, logsb)

	if resp != nil {
		return resp
	}

	return &pb.PairResponse{
		ErrorCode:         pb.PairError_SUCCESS,
		Pairings:          pairings,
		GibsonizedPlayers: copdata.GibsonizedPlayers,
	}
}

func copMinWeightMatching(req *pb.PairRequest, copdata *copdatapkg.PrecompData, logsb *strings.Builder) ([]int32, *pb.PairResponse) {
	prepairedRoundIdx := -1
	numDivPairings := len(req.DivisionPairings)
	prepairedPlayerIndexes := map[int]int{}
	numForcedByes := 0
	removedPlayersSet := map[int]bool{}
	for _, removedPlayerIdx := range req.RemovedPlayers {
		removedPlayersSet[int(removedPlayerIdx)] = true
	}
	if numDivPairings > 0 {
		for _, oppIdx := range req.DivisionPairings[numDivPairings-1].Pairings {
			if oppIdx == -1 {
				prepairedRoundIdx = numDivPairings - 1
				break
			}
		}
		if prepairedRoundIdx >= 0 {
			for playerIdx, oppIdx := range req.DivisionPairings[prepairedRoundIdx].Pairings {
				if int(oppIdx) < playerIdx || removedPlayersSet[playerIdx] {
					continue
				}
				prepairedPlayerIndexes[playerIdx] = int(oppIdx)
				prepairedPlayerIndexes[int(oppIdx)] = playerIdx
				prepairedPlayersStr := fmt.Sprintf("Forcing (#%d) %s vs ", playerIdx+1, req.PlayerNames[playerIdx])
				if playerIdx == (int(oppIdx)) {
					numForcedByes++
					prepairedPlayersStr += "BYE\n"
				} else {
					prepairedPlayersStr += fmt.Sprintf("(#%d) %s\n", int(oppIdx)+1, req.PlayerNames[int(oppIdx)])
				}
				logsb.WriteString(prepairedPlayersStr)
			}
		}
	}

	logsb.WriteString(fmt.Sprintf("\nForcing %d bye(s)\n\n", numForcedByes))

	playerNodes := []int{}
	divisionPlayerData := [][]string{}
	numPlayers := copdata.Standings.GetNumPlayers()
	for playerRankIdx := 0; playerRankIdx < numPlayers; playerRankIdx++ {
		playerIdx := copdata.Standings.GetPlayerIndex(playerRankIdx)
		playerNodes = append(playerNodes, playerIdx)
		divisionPlayerData = append(divisionPlayerData, copdata.Standings.StringDataForPlayer(req, playerRankIdx))
	}

	addBye := (numPlayers-numForcedByes)%2 == 1
	if addBye {
		playerNodes = append(playerNodes, pkgstnd.ByePlayerIndex)
		divisionPlayerData = append(divisionPlayerData, []string{"", "", "BYE", "", ""})
	}

	lowestPossibleAbsCasher := 0
	for playerRankIdx, place := range copdata.HighestRankAbsolutely {
		if place < int(req.PlacePrizes) {
			lowestPossibleAbsCasher = playerRankIdx
		}
	}

	lowestPossibleHopeCasher := copdata.LowestPossibleHopeNth[int(req.PlacePrizes)-1]

	gibsonGetsBye := false
	if addBye {
		for i := 0; i < numPlayers; i++ {
			if playerNodes[i] == pkgstnd.ByePlayerIndex {
				break
			}
			if copdata.GibsonizedPlayers[i] && copdata.GibsonGroups[i] == 0 {
				gibsonGetsBye = true
				break
			}
		}
	}

	pargs := &policyArgs{
		req:                      req,
		copdata:                  copdata,
		playerNodes:              playerNodes,
		lowestPossibleAbsCasher:  lowestPossibleAbsCasher,
		lowestPossibleHopeCasher: lowestPossibleHopeCasher,
		roundsRemaining:          pkgstnd.GetRoundsRemaining(req),
		gibsonGetsBye:            gibsonGetsBye,
		prepairedRoundIdx:        prepairedRoundIdx,
		prepairedPlayerIndexes:   prepairedPlayerIndexes,
	}

	logsb.WriteString(fmt.Sprintf("Control Loss Sims: %d\n", req.ControlLossSims))
	logsb.WriteString(fmt.Sprintf("Lowest Hopeful Casher: %s\n", req.PlayerNames[playerNodes[lowestPossibleHopeCasher]]))
	logsb.WriteString(fmt.Sprintf("Lowest Absolute Casher: %s\n", req.PlayerNames[playerNodes[lowestPossibleAbsCasher]]))
	logsb.WriteString(fmt.Sprintf("Number of Pairings (including prepaired): %d\n", numDivPairings))
	logsb.WriteString(fmt.Sprintf("Number of Results: %d\n", len(req.DivisionResults)))
	logsb.WriteString(fmt.Sprintf("Rounds Remaining: %d\n", pargs.roundsRemaining))
	logsb.WriteString(fmt.Sprintf("Using Unforced Bye: %t\n", addBye))
	logsb.WriteString(fmt.Sprintf("Gibson Gets Bye: %t\n", pargs.gibsonGetsBye))
	logsb.WriteString(fmt.Sprintf("Prepaired Round (0 for none): %d\n", pargs.prepairedRoundIdx+1))
	logsb.WriteString("Destinys Child: ")
	if copdata.DestinysChild >= 0 {
		logsb.WriteString(req.PlayerNames[playerNodes[copdata.DestinysChild]])
	} else {
		logsb.WriteString("(none)")
	}
	logsb.WriteString("\n\n")

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

	if len(pairings) > numPlayers {
		return nil, &pb.PairResponse{
			ErrorCode:    pb.PairError_INVALID_PAIRINGS_LENGTH,
			ErrorMessage: fmt.Sprintf("invalid pairings length %d for %d players", len(pairings), numPlayers),
		}
	} else if len(pairings) < numPlayers {
		numUnpairedAtBottom := numPlayers - len(pairings)
		unpairedIndexes := make([]int, numUnpairedAtBottom)
		for i := range unpairedIndexes {
			unpairedIndexes[i] = -1
		}
		pairings = append(pairings, unpairedIndexes...)
	}

	for playerRankIdx, oppRankIdx := range pairings {
		if oppRankIdx < playerRankIdx {
			continue
		}
		pairingDetails[pairingsToDetailsIndex[getRankPairingKey(playerRankIdx, oppRankIdx)]][len(matchupHeader)] = "*"
	}

	copdatapkg.WriteStringDataToLog("Pairing Weights", pairingDetailsheader, pairingDetails, logsb)

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

	copdatapkg.WriteStringDataToLog("Final COP Pairings", append(matchupHeader, []string{"Previous Times Played"}...), pairingsLogMx, logsb)

	logsb.WriteString(fmt.Sprintf("Total Weight: %d\n", totalWeight))

	allPlayerPairings := make([]int32, req.AllPlayers)
	for i := 0; i < int(req.AllPlayers); i++ {
		allPlayerPairings[i] = -1
	}
	unpairedPlayerIndexes := []int{}
	prepairedPlayersStr := ""
	// Convert rank indexes to player indexes and convert the bye format from ByePlayerIndex to player index
	for playerRankIdx := 0; playerRankIdx < len(pairings); playerRankIdx++ {
		oppRankIdx := pairings[playerRankIdx]
		playerIdx := playerNodes[playerRankIdx]
		prepairedOppIdx, playerIsPrepaired := prepairedPlayerIndexes[playerIdx]
		if oppRankIdx < 0 {
			if !playerIsPrepaired {
				unpairedPlayerIndexes = append(unpairedPlayerIndexes, playerIdx)
				continue
			}
			allPlayerPairings[playerIdx] = int32(prepairedOppIdx)
			if playerIdx <= prepairedOppIdx {
				prepairedPlayersStr += fmt.Sprintf("(#%d) %s vs ", playerIdx+1, req.PlayerNames[playerIdx])
				if playerIdx == prepairedOppIdx {
					prepairedPlayersStr += "BYE\n"
				} else {
					prepairedPlayersStr += fmt.Sprintf("(#%d) %s\n", prepairedOppIdx+1, req.PlayerNames[prepairedOppIdx])
				}
			}
		} else if playerIsPrepaired {
			return nil, &pb.PairResponse{
				ErrorCode:    pb.PairError_OVERCONSTRAINED,
				ErrorMessage: fmt.Sprintf("player %s is prepaired but was still paired by COP", req.PlayerNames[playerIdx]),
			}
		} else {
			oppIdx := playerNodes[oppRankIdx]
			if oppIdx == pkgstnd.ByePlayerIndex {
				oppIdx = playerIdx
			}
			allPlayerPairings[playerIdx] = int32(oppIdx)
		}
	}

	if prepairedPlayersStr != "" {
		logsb.WriteString(fmt.Sprintf("\nPrepaired players:\n\n%s", prepairedPlayersStr))
	}

	removedPlayersStr := ""
	for _, removedPlayerIdx := range req.RemovedPlayers {
		if allPlayerPairings[removedPlayerIdx] != -1 {
			return nil, &pb.PairResponse{
				ErrorCode:    pb.PairError_OVERCONSTRAINED,
				ErrorMessage: fmt.Sprintf("player %s was removed but was still paired by COP", req.PlayerNames[removedPlayerIdx]),
			}
		}
		removedPlayersStr += fmt.Sprintf("(#%d) %s\n", removedPlayerIdx+1, req.PlayerNames[removedPlayerIdx])
	}

	if removedPlayersStr != "" {
		logsb.WriteString(fmt.Sprintf("\nRemoved players:\n\n%s\n", removedPlayersStr))
	}

	numUnpairedPlayers := len(unpairedPlayerIndexes)
	if numUnpairedPlayers > 0 {
		msg := "COP pairings could not be completed because there were too many constraints. The unpaired players are:\n\n"
		for _, unpairedPlayerIdx := range unpairedPlayerIndexes {
			msg += fmt.Sprintf("%s\n", req.PlayerNames[unpairedPlayerIdx])
		}
		return nil, &pb.PairResponse{
			ErrorCode:    pb.PairError_OVERCONSTRAINED,
			ErrorMessage: msg,
		}
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
