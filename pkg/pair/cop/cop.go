package cop

import (
	"fmt"
	"hash/fnv"
	"math"
	"slices"

	"golang.org/x/exp/rand"
	"google.golang.org/protobuf/encoding/protojson"

	"strings"
	"time"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/matching"
	"github.com/woogles-io/liwords/pkg/pair"
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

var pairingMethodMap = map[pb.PairMethod]pb.PairingMethod{
	pb.PairMethod_PAIR_RANDOM:                  pb.PairingMethod_RANDOM,
	pb.PairMethod_PAIR_ROUND_ROBIN:             pb.PairingMethod_ROUND_ROBIN,
	pb.PairMethod_PAIR_KING_OF_THE_HILL:        pb.PairingMethod_KING_OF_THE_HILL,
	pb.PairMethod_PAIR_FACTOR:                  pb.PairingMethod_FACTOR,
	pb.PairMethod_PAIR_INITIAL_FONTES:          pb.PairingMethod_INITIAL_FONTES,
	pb.PairMethod_PAIR_TEAM_ROUND_ROBIN:        pb.PairingMethod_TEAM_ROUND_ROBIN,
	pb.PairMethod_PAIR_INTERLEAVED_ROUND_ROBIN: pb.PairingMethod_INTERLEAVED_ROUND_ROBIN,
}

type policyArgs struct {
	req                      *pb.PairRequest
	copdata                  *copdatapkg.PrecompData
	playerNodes              []int
	lowestPossibleAbsCasher  int
	lowestPossibleHopeCasher int
	roundsRemaining          int
	roundPairingsRemaining   int
	gibsonGetsBye            bool
	prepairedRoundIdx        int
	prepairedPlayerIndexes   map[int]int
	lowestHopeOverride       map[int]int
	factor3ForcedPairings    [][2]int
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

			// First, compute the cash prize KOTH players

			var highestNoncontender int
			for playerRankIdx := 0; playerRankIdx < numPlayers-1; playerRankIdx++ {
				if pargs.lowestPossibleAbsCasher < playerRankIdx {
					highestNoncontender = playerRankIdx
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
				// A pairing with pi and pj was forced, so we need to
				// skip evaluation for player pj in the next iteration
				// by incrementing playerRankIdx, which combined with
				// this for loop effectively performs a playerRankIdx += 2
				playerRankIdx++
			}

			// Then, compute the class prize KOTH players

			// Do not consider the bye as a player in this case
			if pargs.playerNodes[numPlayers-1] == pkgstnd.ByePlayerIndex {
				numPlayers--
			}
			for classPrizesIdx, classPrizes := range pargs.req.ClassPrizes {
				classIdx := classPrizesIdx + 1
				availableClassPrizes := int(classPrizes)
				for pRankIdx := 0; pRankIdx < highestNoncontender; pRankIdx++ {
					pIdx := pargs.copdata.Standings.GetPlayerIndex(pRankIdx)
					if int(pargs.req.PlayerClasses[pIdx]) == classIdx && pargs.copdata.LowestRankAbsolutely[pRankIdx] >= int(pargs.req.PlacePrizes) {
						availableClassPrizes--
					}
				}
				if availableClassPrizes < 1 {
					continue
				}
				ri := highestNoncontender
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
			// Factor 3 expansion already constrains the top-6 pairings; applying
			// control loss on top of it would overconstraint the matching.
			if len(pargs.factor3ForcedPairings) > 0 {
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
			if !pargs.req.TopDownByes || pargs.playerNodes[numPlayers-1] != pkgstnd.ByePlayerIndex || pargs.gibsonGetsBye {
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
	{
		// Factor 3 expansion for the 2nd-to-last round
		name: "F3",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			if len(pargs.factor3ForcedPairings) == 0 {
				return [][2]int{}, [][2]int{}
			}
			return pargs.factor3ForcedPairings, [][2]int{}
		},
	},
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
			// In the fourth quarter, cashers use PC weight exclusively; zero out RD.
			if pargs.roundPairingsRemaining*4 <= int(pargs.req.Rounds) &&
				!pargs.copdata.GibsonizedPlayers[ri] &&
				ri <= pargs.lowestPossibleHopeCasher {
				return 0
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
			lowestContender := pargs.copdata.LowestPossibleHopeNth[ri]
			if override, ok := pargs.lowestHopeOverride[ri]; ok {
				lowestContender = override
			}
			// Check if we should apply an inverse distance penalty
			if rj <= lowestContender || (lowestContender == ri && ri == rj-1) {
				// Only apply PC weight in the fourth quarter.
				if pargs.roundPairingsRemaining*4 > int(pargs.req.Rounds) {
					return 0
				}
				// Calculate the inverse distance penalty
				casherDiff := lowestContender - rj
				if casherDiff < 0 {
					casherDiff *= -1
				}
				return int64(math.Pow(float64(casherDiff), 3) * 2)
			}
			// Apply a major penalty if the lower ranked player cannot catch the higher ranked player
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
			unitWeight := int64(4 * int(math.Pow(float64(pargs.copdata.Standings.GetNumPlayers())/3.0, 3)))
			// We would like the following to always be true:
			//
			// n-peat weight > 2 * (n-1)-peat weight
			//
			// The minimal recursive formula satisfying this is:
			//
			// RE(1) = 1
			// RE(n) = 2 * RE(n-1) + 1
			//
			// which results in the following values for repeats:
			// RE(1) = 1
			// RE(2) = 3
			// RE(3) = 7
			// RE(4) = 15
			// RE(5) = 31
			// ...
			multiplier := 0
			if timesPlayed > 0 {
				multiplier = (1 << timesPlayed) - 1
			}
			return int64(multiplier) * unitWeight
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

	switch req.PairMethod {
	case pb.PairMethod_PAIR_SWISS:
		return swissPair(req, logsb)
	case pb.PairMethod_COP:
		return copMethodPair(req, logsb)
	case pb.PairMethod_PAIR_AUTO:
		return autoPair(req, logsb)
	default:
		return simplePair(req, logsb)
	}
}

func copMethodPair(req *pb.PairRequest, logsb *strings.Builder) *pb.PairResponse {
	copRand := rand.New(rand.NewSource(uint64(req.Seed)))
	copdata, pairErr := copdatapkg.GetPrecompData(req, copRand, logsb)

	if pairErr != pb.PairError_SUCCESS {
		return &pb.PairResponse{
			ErrorCode:    pairErr,
			ErrorMessage: fmt.Sprintf("error computing required inputs: %s", pb.PairError_name[int32(pairErr)]),
		}
	}

	factor3ForcedPairings := computeFactor3ForcedPairings(req, copdata, copRand, logsb)
	pairings, resp := copMinWeightMatching(req, copdata, factor3ForcedPairings, logsb)

	if resp != nil {
		return resp
	}

	return &pb.PairResponse{
		ErrorCode:         pb.PairError_SUCCESS,
		Pairings:          pairings,
		GibsonizedPlayers: copdata.GibsonizedPlayers,
	}
}

// computeFactor3ForcedPairings checks whether the 2nd-to-last round should use
// factor-3 pairings (1v4, 2v5, 3v6). It first checks whether 2nd or 3rd place
// would lose control of their destiny under factor-3 (compared to playing 1st
// directly); if so, only the affected player is paired against 1st. Otherwise it
// checks whether 4th/5th/6th can each reach 1st/2nd/3rd within the hopefulness
// threshold, and if so returns the three factor-3 forced pairs. Returns nil when
// no forced pairings are needed.
func computeFactor3ForcedPairings(req *pb.PairRequest, copdata *copdatapkg.PrecompData, copRand *rand.Rand, logsb *strings.Builder) [][2]int {
	if pkgstnd.GetRoundsRemaining(req) != 2 {
		logsb.WriteString("Factor 3 skipped: not 2 rounds remaining\n")
		return nil
	}
	numPlayers := copdata.Standings.GetNumPlayers()
	if numPlayers < 6 {
		logsb.WriteString(fmt.Sprintf("Factor 3 skipped: fewer than 6 players (%d)\n", numPlayers))
		return nil
	}

	// Build factor-3 pairings for the penultimate round (ranks 0v3, 1v4, 2v5,
	// then factor M/2 for the remaining players) plus KOTH for the final round.
	// pairingsLen is rounded up to even so simRound can always iterate in pairs
	// (RunParallelSimForceWinner adds a dummy bye player for odd player counts).
	pairingsLen := numPlayers
	if pairingsLen%2 == 1 {
		pairingsLen++
	}
	f3Pairings := make([][]int, 2)
	f3Pairings[0] = make([]int, pairingsLen) // penultimate: factor-3
	f3Pairings[1] = make([]int, pairingsLen) // final: KOTH
	// Top 6: factor-3 pairs (0,3), (1,4), (2,5).
	for i := 0; i < 3; i++ {
		f3Pairings[0][2*i] = i
		f3Pairings[0][2*i+1] = i + 3
	}
	// Remaining slots (ranks 6 to pairingsLen-1): factor (pairingsLen-6)/2.
	remCount := pairingsLen - 6
	remFactor := remCount / 2
	for i := 0; i < remFactor; i++ {
		f3Pairings[0][6+2*i] = 6 + i
		f3Pairings[0][6+2*i+1] = 6 + i + remFactor
	}
	// Final round: KOTH (consecutive pairs).
	for i := 0; i < pairingsLen/2; i++ {
		f3Pairings[1][2*i] = 2 * i
		f3Pairings[1][2*i+1] = 2*i + 1
	}

	// Simulate factor-3 pairings using the pre-built f3Pairings so the sim uses the
	// correct factor-3 structure (roundsRemaining=2 < maxFactor=3, so SimFactorPairAll
	// would cap at factor-2 internally).
	factor3FinalRanks, totalSims, err := copdata.Standings.RunSimsWithPairings(copRand, int(req.DivisionSims), 2, f3Pairings)
	if err != pb.PairError_SUCCESS {
		logsb.WriteString(fmt.Sprintf("Factor 3 skipped: sim error %v\n", err))
		return nil
	}
	if totalSims == 0 {
		logsb.WriteString("Factor 3 skipped: zero sims completed\n")
		return nil
	}
	copdatapkg.WriteFinalRankResultsToLog("Factor 3 Sim Results", factor3FinalRanks, copdata.Standings, req, logsb)

	// Check whether 2nd or 3rd place loses control under factor-3 pairings.
	// Each player is assumed to win all their games in both scenarios; the only
	// difference is whether they play 1st (vsFirst) or their factor-3 opponent
	// (vsFactor3) in the penultimate round.
	roundsRemaining := pkgstnd.GetRoundsRemaining(req)
	controlSims := int(req.ControlLossSims)
	threshold := req.ControlLossThreshold * float64(controlSims)
	stopTime := time.Now().Add(6 * time.Second).UnixNano()

	p0 := copdata.Standings.GetPlayerIndex(0)
	p1 := copdata.Standings.GetPlayerIndex(1)
	p2 := copdata.Standings.GetPlayerIndex(2)

	logsb.WriteString(fmt.Sprintf(
		"Factor 3 control loss standings: %s (%.1f/%d) %s (%.1f/%d) %s (%.1f/%d) %s (%.1f/%d) %s (%.1f/%d) %s (%.1f/%d)\n",
		req.PlayerNames[p0], float64(copdata.Standings.GetPlayerWinsIntTimesTwo(0))/2, copdata.Standings.GetPlayerSpread(0),
		req.PlayerNames[p1], float64(copdata.Standings.GetPlayerWinsIntTimesTwo(1))/2, copdata.Standings.GetPlayerSpread(1),
		req.PlayerNames[p2], float64(copdata.Standings.GetPlayerWinsIntTimesTwo(2))/2, copdata.Standings.GetPlayerSpread(2),
		req.PlayerNames[copdata.Standings.GetPlayerIndex(3)], float64(copdata.Standings.GetPlayerWinsIntTimesTwo(3))/2, copdata.Standings.GetPlayerSpread(3),
		req.PlayerNames[copdata.Standings.GetPlayerIndex(4)], float64(copdata.Standings.GetPlayerWinsIntTimesTwo(4))/2, copdata.Standings.GetPlayerSpread(4),
		req.PlayerNames[copdata.Standings.GetPlayerIndex(5)], float64(copdata.Standings.GetPlayerWinsIntTimesTwo(5))/2, copdata.Standings.GetPlayerSpread(5),
	))

	for _, rankIdx := range []int{1, 2} {
		pIdx := copdata.Standings.GetPlayerIndex(rankIdx)
		vsFirst, pairErr := copdata.Standings.RunParallelSimForceWinner(copRand, controlSims, roundsRemaining, 3, f3Pairings, pIdx, true, stopTime)
		if pairErr != pb.PairError_SUCCESS {
			continue
		}
		vsFactor3, pairErr := copdata.Standings.RunParallelSimForceWinner(copRand, controlSims, roundsRemaining, 3, f3Pairings, pIdx, false, stopTime)
		if pairErr != pb.PairError_SUCCESS {
			continue
		}
		logsb.WriteString(fmt.Sprintf(
			"Factor 3 control loss check rank %d (%s): vsFirst=%d/%d vsFactor3=%d/%d threshold=%.0f\n",
			rankIdx+1, req.PlayerNames[pIdx], vsFirst, controlSims, vsFactor3, controlSims, threshold,
		))
		if float64(vsFirst-vsFactor3) >= threshold {
			logsb.WriteString(fmt.Sprintf(
				"Factor 3 control loss: rank %d %s loses control, forcing %s vs %s\n",
				rankIdx+1, req.PlayerNames[pIdx], req.PlayerNames[p0], req.PlayerNames[pIdx],
			))
			return [][2]int{{p0, pIdx}}
		}
	}

	// Neither 2nd nor 3rd loses control under factor-3; check whether 4th/5th/6th
	// can each reach 1st/2nd/3rd respectively within the hopefulness threshold.
	minWins := int(math.Round(float64(totalSims) * float64(req.HopefulnessThreshold)))

	// 4th can get 1st, 5th can get 2nd, 6th can get 3rd.
	p3 := copdata.Standings.GetPlayerIndex(3)
	p4 := copdata.Standings.GetPlayerIndex(4)
	p5 := copdata.Standings.GetPlayerIndex(5)
	if factor3FinalRanks[3][0] < minWins ||
		factor3FinalRanks[4][1] < minWins ||
		factor3FinalRanks[5][2] < minWins {
		logsb.WriteString(fmt.Sprintf(
			"Factor 3 skipped: hopefulness threshold not met (minWins=%d 4th=%s->1st:%d 5th=%s->2nd:%d 6th=%s->3rd:%d)\n",
			minWins,
			req.PlayerNames[p3], factor3FinalRanks[3][0],
			req.PlayerNames[p4], factor3FinalRanks[4][1],
			req.PlayerNames[p5], factor3FinalRanks[5][2],
		))
		return nil
	}

	logsb.WriteString(fmt.Sprintf(
		"Factor 3 expansion: forcing %s vs %s, %s vs %s, %s vs %s\n",
		req.PlayerNames[p0], req.PlayerNames[p3],
		req.PlayerNames[p1], req.PlayerNames[p4],
		req.PlayerNames[p2], req.PlayerNames[p5],
	))
	return [][2]int{{p0, p3}, {p1, p4}, {p2, p5}}
}

// extractPrepairedPlayers returns a map of playerIdx -> oppIdx for all prepaired players
// (where oppIdx == playerIdx means bye), the count of forced byes, and the prepaired round index.
// Players not in this map are unpaired and should be assigned by the pairing method.
func extractPrepairedPlayers(req *pb.PairRequest) (map[int]int, int, int) {
	prepairedPlayerIndexes := map[int]int{}
	numForcedByes := 0
	prepairedRoundIdx := -1
	numDivPairings := len(req.DivisionPairings)
	removedPlayersSet := map[int]bool{}
	for _, idx := range req.RemovedPlayers {
		removedPlayersSet[int(idx)] = true
	}
	if numDivPairings > 0 {
		if slices.Contains(req.DivisionPairings[numDivPairings-1].Pairings, -1) {
			prepairedRoundIdx = numDivPairings - 1
		}
		if prepairedRoundIdx >= 0 {
			for playerIdx, oppIdx := range req.DivisionPairings[prepairedRoundIdx].Pairings {
				if int(oppIdx) < playerIdx || removedPlayersSet[playerIdx] || removedPlayersSet[int(oppIdx)] {
					continue
				}
				prepairedPlayerIndexes[playerIdx] = int(oppIdx)
				prepairedPlayerIndexes[int(oppIdx)] = playerIdx
				if playerIdx == int(oppIdx) {
					numForcedByes++
				}
			}
		}
	}
	return prepairedPlayerIndexes, numForcedByes, prepairedRoundIdx
}

// currentRoundIndex returns the index of the round being paired.
// If the last round in DivisionPairings is incomplete (has a -1 entry), that round is being
// completed, so its index is returned. Otherwise the next new round index is returned.
func currentRoundIndex(req *pb.PairRequest) int32 {
	n := len(req.DivisionPairings)
	if n > 0 && slices.Contains(req.DivisionPairings[n-1].Pairings, -1) {
		return int32(n - 1)
	}
	return int32(n)
}

// computePairingCounts returns a map from pairing key to number of times that pair has played,
// counting only complete rounds (excluding any partially-prepaired last round).
func computePairingCounts(req *pb.PairRequest) map[string]int {
	counts := map[string]int{}
	numDivPairings := len(req.DivisionPairings)
	numComplete := numDivPairings
	if numComplete > 0 && slices.Contains(req.DivisionPairings[numComplete-1].Pairings, -1) {
		numComplete--
	}
	for roundIdx := range numComplete {
		for playerIdx, oppIdx := range req.DivisionPairings[roundIdx].Pairings {
			if int(oppIdx) > playerIdx {
				continue
			}
			key := copdatapkg.GetPairingKey(playerIdx, int(oppIdx))
			counts[key]++
		}
	}
	return counts
}

// simplePairOnce runs a single round of a non-COP pairing method and returns
// the full allPlayers-length pairings slice (bye represented as self-pairing).
func simplePairOnce(req *pb.PairRequest, pairingMethod pb.PairingMethod, poolMembers []*entity.PoolMember, playerOrder []int, roundIdx int32, seed uint64) ([]int32, error) {
	roundControls := &pb.RoundControl{
		PairingMethod:               pairingMethod,
		Round:                       roundIdx,
		Factor:                      req.Factor,
		InitialFontes:               req.InitialNonperfRounds,
		GamesPerRound:               1,
		MaxRepeats:                  1,
		AllowOverMaxRepeats:         true,
		RepeatRelativeWeight:        1,
		WinDifferenceRelativeWeight: 1,
	}
	m := &entity.UnpairedPoolMembers{
		PoolMembers:   poolMembers,
		RoundControls: roundControls,
		Repeats:       map[string]int{},
		Seed:          seed,
	}
	poolPairings, err := pair.Pair(m)
	if err != nil {
		return nil, err
	}
	roundPairings := make([]int32, req.AllPlayers)
	for i := range roundPairings {
		roundPairings[i] = -1
	}
	for poolIdx, poolOppIdx := range poolPairings {
		pi := playerOrder[poolIdx]
		if poolOppIdx == -1 {
			roundPairings[pi] = int32(pi)
		} else {
			roundPairings[pi] = int32(playerOrder[poolOppIdx])
		}
	}
	return roundPairings, nil
}

// simplePair handles pairing methods that delegate entirely to pkg/pair/pair.go:
// Random, Round Robin, King of the Hill, Factor, Initial Fontes, Team Round Robin,
// Interleaved Round Robin.
func simplePair(req *pb.PairRequest, logsb *strings.Builder) *pb.PairResponse {
	removedPlayersSet := map[int]bool{}
	for _, idx := range req.RemovedPlayers {
		removedPlayersSet[int(idx)] = true
	}

	// For rank-dependent methods we need standings to get the correct player order.
	needsRankOrder := req.PairMethod == pb.PairMethod_PAIR_KING_OF_THE_HILL ||
		req.PairMethod == pb.PairMethod_PAIR_FACTOR

	standings := pkgstnd.CreateInitialStandings(req)
	standings.Sort()

	// Build the ordered list of valid players.
	playerOrder := []int{}
	if needsRankOrder {
		numPlayers := standings.GetNumPlayers()
		for rankIdx := range numPlayers {
			pi := standings.GetPlayerIndex(rankIdx)
			if !removedPlayersSet[pi] {
				playerOrder = append(playerOrder, pi)
			}
		}
	} else {
		for pi := 0; pi < int(req.AllPlayers); pi++ {
			if !removedPlayersSet[pi] {
				playerOrder = append(playerOrder, pi)
			}
		}
	}

	pairingMethod, ok := pairingMethodMap[req.PairMethod]
	if !ok {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_UNSUPPORTED_PAIR_METHOD,
			ErrorMessage: fmt.Sprintf("unsupported pair method: %v", req.PairMethod),
		}
	}

	poolMembers := make([]*entity.PoolMember, len(playerOrder))
	for i, pi := range playerOrder {
		poolMembers[i] = &entity.PoolMember{
			Id: req.PlayerNames[pi],
		}
	}

	currentRound := currentRoundIndex(req)
	allPlayerPairings, err := simplePairOnce(req, pairingMethod, poolMembers, playerOrder, currentRound, uint64(req.Seed))
	if err != nil {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_SIMPLE_PAIRING_FAILED,
			ErrorMessage: err.Error(),
		}
	}

	standingsHeader := []string{"Rank", "Num", "Name", "Wins", "Spr"}
	copdatapkg.WriteStringDataToLog("Initial Standings", standingsHeader, standings.StringData(req), logsb)

	logsb.WriteString(fmt.Sprintf("Method: %s\n", req.PairMethod))
	logsb.WriteString(fmt.Sprintf("Seed: %d\n", req.Seed))
	logsb.WriteString(fmt.Sprintf("Round control: PairingMethod=%s Round=%d Factor=%d InitialNonperf=%d\n",
		pairingMethod,
		currentRound,
		req.Factor,
		req.InitialNonperfRounds,
	))
	logsb.WriteString(fmt.Sprintf("Pool members (%d):\n", len(poolMembers)))
	for i, pm := range poolMembers {
		logsb.WriteString(fmt.Sprintf("  [%d] %s\n", i, pm.Id))
	}

	// Log the pairings array (player i plays allPlayerPairings[i]).
	logsb.WriteString("Pairings: [")
	for i, opp := range allPlayerPairings {
		if i > 0 {
			logsb.WriteString(" ")
		}
		logsb.WriteString(fmt.Sprintf("%d", opp))
	}
	logsb.WriteString("]\n")

	// Log pairings by standings rank with player records.
	numStandingsPlayers := standings.GetNumPlayers()
	playerIndexToRankIdx := make(map[int]int, numStandingsPlayers)
	divisionPlayerData := make([][]string, numStandingsPlayers)
	for rankIdx := range numStandingsPlayers {
		pi := standings.GetPlayerIndex(rankIdx)
		playerIndexToRankIdx[pi] = rankIdx
		divisionPlayerData[rankIdx] = standings.StringDataForPlayer(req, rankIdx)
	}
	matchupHeader := []string{"Player", "W", "S", "Player", "W", "S"}
	pairingsLogMx := [][]string{}
	for rankIdx := range numStandingsPlayers {
		pi := standings.GetPlayerIndex(rankIdx)
		opp := int(allPlayerPairings[pi])
		if opp < 0 {
			continue
		}
		oppRankIdx, oppInStandings := playerIndexToRankIdx[opp]
		if !oppInStandings || oppRankIdx < rankIdx {
			continue
		}
		pairingsLogMx = append(pairingsLogMx, getMatchupStrArray(divisionPlayerData, rankIdx, oppRankIdx))
	}
	copdatapkg.WriteStringDataToLog("Final Pairings", matchupHeader, pairingsLogMx, logsb)

	// Compute multiround pairings.
	// If the tournament has existing pairings, multiround is just the current round's pairings.
	// If no pairings exist yet and this is a non-rank-order method, generate N rounds ahead.
	var multiroundPairings []int32
	if len(req.DivisionPairings) > 0 {
		multiroundPairings = append(multiroundPairings, allPlayerPairings...)
	} else if !needsRankOrder {
		N := max(int(req.InitialNonperfRounds), 1)
		multiroundPairings = append(multiroundPairings, allPlayerPairings...)
		for roundIdx := 1; roundIdx < N; roundIdx++ {
			rp, err := simplePairOnce(req, pairingMethod, poolMembers, playerOrder, currentRound+int32(roundIdx), uint64(req.Seed)+uint64(roundIdx))
			if err != nil {
				return &pb.PairResponse{
					ErrorCode:    pb.PairError_SIMPLE_PAIRING_FAILED,
					ErrorMessage: err.Error(),
				}
			}
			multiroundPairings = append(multiroundPairings, rp...)
		}
	}

	return &pb.PairResponse{
		ErrorCode:          pb.PairError_SUCCESS,
		Pairings:           allPlayerPairings,
		MultiroundPairings: multiroundPairings,
	}
}

// swissPair implements a minimum weight matching Swiss pairing where:
//   - prepaired player requests are fulfilled
//   - repeats and bye repeats get a major penalty
//   - win differences of WD get a penalty of WD * minor penalty
//   - spread diff is a bonus when both players have the same wins, otherwise a penalty
func swissPair(req *pb.PairRequest, logsb *strings.Builder) *pb.PairResponse {
	prepairedPlayerIndexes, _, _ := extractPrepairedPlayers(req)
	pairingCounts := computePairingCounts(req)

	removedPlayersSet := map[int]bool{}
	for _, idx := range req.RemovedPlayers {
		removedPlayersSet[int(idx)] = true
	}

	standings := pkgstnd.CreateInitialStandings(req)
	standings.Sort()
	numStandingsPlayers := standings.GetNumPlayers()

	// Build a player-index-to-standings-rank lookup for O(1) access.
	playerToRankIdx := make(map[int]int, numStandingsPlayers)
	for rankIdx := range numStandingsPlayers {
		playerToRankIdx[standings.GetPlayerIndex(rankIdx)] = rankIdx
	}

	// Build rank-ordered list of unpaired valid players.
	playerOrder := []int{}
	for rankIdx := range numStandingsPlayers {
		pi := standings.GetPlayerIndex(rankIdx)
		if !removedPlayersSet[pi] {
			if _, isPrepaired := prepairedPlayerIndexes[pi]; !isPrepaired {
				playerOrder = append(playerOrder, pi)
			}
		}
	}

	poolSize := len(playerOrder)
	addBye := poolSize%2 == 1

	type playerRecord struct {
		wins   int // wins*2 + ties, so wins count 2 and ties count 1
		spread int
	}
	records := make([]playerRecord, poolSize)
	poolPlayerData := make([][]string, poolSize)
	for poolIdx, pi := range playerOrder {
		rankIdx := playerToRankIdx[pi]
		poolPlayerData[poolIdx] = standings.StringDataForPlayer(req, rankIdx)
		records[poolIdx] = playerRecord{
			wins:   standings.GetPlayerWinsIntTimesTwo(rankIdx),
			spread: standings.GetPlayerSpread(rankIdx),
		}
	}

	// The bye is appended as an extra node at index byeIdx when addBye is true.
	byeIdx := poolSize
	numPoolNodes := poolSize
	if addBye {
		numPoolNodes++
	}

	swissWeightHeader := []string{"Player", "W", "S", "Player", "W", "S", "*", "PTP", "Total", "RP", "WD", "SD", "BR"}
	pairingDetails := [][]string{}
	edgeToDetailIdx := map[string]int{}

	edges := []*matching.Edge{}
	for poolIdxI := 0; poolIdxI < numPoolNodes; poolIdxI++ {
		for poolIdxJ := poolIdxI + 1; poolIdxJ < numPoolNodes; poolIdxJ++ {
			var repeatPenalty, winDiffPenalty, spreadComp, byeRepeatPenalty int64

			iIsBye := addBye && poolIdxI == byeIdx
			jIsBye := addBye && poolIdxJ == byeIdx

			var iData, jData []string
			if iIsBye {
				iData = []string{byePlayerName, "", ""}
			} else {
				iData = getPlayerRecordStrArray(poolPlayerData[poolIdxI])
			}
			if jIsBye {
				jData = []string{byePlayerName, "", ""}
			} else {
				jData = getPlayerRecordStrArray(poolPlayerData[poolIdxJ])
			}

			var timesPlayed int
			if !iIsBye && !jIsBye {
				pi := playerOrder[poolIdxI]
				pj := playerOrder[poolIdxJ]

				key := copdatapkg.GetPairingKey(pi, pj)
				timesPlayed = pairingCounts[key]
				if timesPlayed > 0 {
					repeatPenalty = majorPenalty * int64(timesPlayed)
				}

				winDiff := records[poolIdxI].wins - records[poolIdxJ].wins
				if winDiff < 0 {
					winDiff = -winDiff
				}
				winDiffPenalty = int64(winDiff) * int64(winDiff) * int64(minorPenalty) / 2

				spreadDiff := records[poolIdxI].spread - records[poolIdxJ].spread
				if spreadDiff < 0 {
					spreadDiff = -spreadDiff
				}
				// Spread diff is a bonus (reduces weight) when wins are equal,
				// encouraging wide spread gaps, and a penalty otherwise.
				if records[poolIdxI].wins == records[poolIdxJ].wins {
					spreadComp = -int64(spreadDiff)
				} else {
					spreadComp = int64(spreadDiff)
				}
			} else {
				var pi int
				if iIsBye {
					pi = playerOrder[poolIdxJ]
				} else {
					pi = playerOrder[poolIdxI]
				}
				byeKey := copdatapkg.GetPairingKey(pi, pi)
				timesPlayed = pairingCounts[byeKey]
				if timesPlayed > 0 {
					byeRepeatPenalty = majorPenalty * int64(timesPlayed)
				}
			}

			totalWeight := repeatPenalty + winDiffPenalty + spreadComp + byeRepeatPenalty

			row := make([]string, 0, len(swissWeightHeader))
			row = append(row, iData...)
			row = append(row, jData...)
			row = append(row, "")                                  // * placeholder
			row = append(row, fmt.Sprintf("%d", timesPlayed))      // PTP
			row = append(row, fmt.Sprintf("%d", totalWeight))      // Total
			row = append(row, fmt.Sprintf("%d", repeatPenalty))    // RP
			row = append(row, fmt.Sprintf("%d", winDiffPenalty))   // WD
			row = append(row, fmt.Sprintf("%d", spreadComp))       // SD
			row = append(row, fmt.Sprintf("%d", byeRepeatPenalty)) // BR

			edgeKey := getRankPairingKey(poolIdxI, poolIdxJ)
			edgeToDetailIdx[edgeKey] = len(pairingDetails)
			pairingDetails = append(pairingDetails, row)

			edges = append(edges, matching.NewEdge(poolIdxI, poolIdxJ, totalWeight))
		}
		if poolIdxI < numPoolNodes-2 {
			pairingDetails = append(pairingDetails, make([]string, len(swissWeightHeader)))
		}
	}

	poolPairings, _, err := matching.MinWeightMatching(edges, true)
	if err != nil {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_MIN_WEIGHT_MATCHING,
			ErrorMessage: fmt.Sprintf("swiss min weight matching error: %s", err.Error()),
		}
	}

	if addBye {
		poolPairings = poolPairings[:len(poolPairings)-1]
	}

	// Mark selected pairings in the weight table.
	for poolIdx, poolOppIdx := range poolPairings {
		if poolOppIdx < poolIdx {
			continue
		}
		edgeKey := getRankPairingKey(poolIdx, poolOppIdx)
		if detailIdx, ok := edgeToDetailIdx[edgeKey]; ok {
			pairingDetails[detailIdx][6] = "*"
		}
	}

	copdatapkg.WriteStringDataToLog("Swiss Pairing Weights", swissWeightHeader, pairingDetails, logsb)

	allPlayerPairings := make([]int32, req.AllPlayers)
	for idx := range allPlayerPairings {
		allPlayerPairings[idx] = -1
	}

	for poolIdx, poolOppIdx := range poolPairings {
		if poolIdx >= len(playerOrder) {
			break
		}
		pi := playerOrder[poolIdx]
		if poolOppIdx < 0 || (addBye && poolOppIdx == byeIdx) {
			allPlayerPairings[pi] = int32(pi) // bye
		} else if poolOppIdx < len(playerOrder) {
			pj := playerOrder[poolOppIdx]
			allPlayerPairings[pi] = int32(pj)
		}
	}

	// Fill in prepaired players.
	for pi, oppIdx := range prepairedPlayerIndexes {
		allPlayerPairings[pi] = int32(oppIdx)
	}

	// Log the pairings array (player i plays allPlayerPairings[i]).
	logsb.WriteString("Pairings: [")
	for i, opp := range allPlayerPairings {
		if i > 0 {
			logsb.WriteString(" ")
		}
		logsb.WriteString(fmt.Sprintf("%d", opp))
	}
	logsb.WriteString("]\n")

	// Log pairings by standings rank with player records.
	playerIndexToPoolIdx := make(map[int]int, len(playerOrder))
	for poolIdx, pi := range playerOrder {
		playerIndexToPoolIdx[pi] = poolIdx
	}
	matchupHeader := []string{"Player", "W", "S", "Player", "W", "S"}
	pairingsLogMx := [][]string{}
	for poolIdx, pi := range playerOrder {
		opp := int(allPlayerPairings[pi])
		if opp < 0 {
			continue
		}
		oppPoolIdx, oppInPool := playerIndexToPoolIdx[opp]
		if !oppInPool || oppPoolIdx < poolIdx {
			continue
		}
		pairingsLogMx = append(pairingsLogMx, getMatchupStrArray(poolPlayerData, poolIdx, oppPoolIdx))
	}
	copdatapkg.WriteStringDataToLog("Final Swiss Pairings", matchupHeader, pairingsLogMx, logsb)

	logsb.WriteString("Method: SWISS\n")
	return &pb.PairResponse{
		ErrorCode: pb.PairError_SUCCESS,
		Pairings:  allPlayerPairings,
	}
}

// autoPair selects the most appropriate pairing method based on the tournament state.
//
// Compute rrRoundsTotal = floor(numRounds / (numValidPlayers-1)) * (numValidPlayers-1),
// the largest number of rounds that fits one or more complete RR cycles.
// If rrRoundsTotal > 0:
//   - Use Round Robin for rounds 0 through rrRoundsTotal-1.
//   - Use COP for any remaining rounds.
//
// Otherwise (numRounds < numValidPlayers-1):
//   - Rounds 0–2: Initial Fontes
//   - Rounds 3 to numRounds/2-1: Swiss
//   - Round numRounds/2 onward: COP
func autoPair(req *pb.PairRequest, logsb *strings.Builder) *pb.PairResponse {
	origMethod := req.PairMethod
	origInitialNonperfRounds := req.InitialNonperfRounds
	resp := autoPairInner(req, logsb)
	restoreAutoPairReqFields(req, origMethod, origInitialNonperfRounds)
	return resp
}

func autoPairInner(req *pb.PairRequest, logsb *strings.Builder) *pb.PairResponse {
	numValidPlayers := int(req.ValidPlayers)
	numRounds := int(req.Rounds)
	currentRound := int(currentRoundIndex(req))
	rrRounds := numValidPlayers - 1
	rrRoundsTotal := (numRounds / rrRounds) * rrRounds
	if rrRoundsTotal > 0 {
		if currentRound < rrRoundsTotal {
			req.PairMethod = pb.PairMethod_PAIR_ROUND_ROBIN
			if currentRound == 0 {
				req.InitialNonperfRounds = int32(rrRoundsTotal)
			}
			fmt.Fprintf(logsb, "Auto: fitting %d complete RR cycle(s) (%d rounds), using Round Robin for round %d\n", rrRoundsTotal/rrRounds, rrRoundsTotal, currentRound)
			return simplePair(req, logsb)
		}
		req.PairMethod = pb.PairMethod_COP
		fmt.Fprintf(logsb, "Auto: round %d past RR rounds (%d), using COP\n", currentRound, rrRoundsTotal)
		return copMethodPair(req, logsb)
	}

	if currentRound < 3 {
		req.PairMethod = pb.PairMethod_PAIR_INITIAL_FONTES
		if currentRound == 0 {
			req.InitialNonperfRounds = int32(min(3, numRounds))
		}
		fmt.Fprintf(logsb, "Auto: round %d < 3, using Initial Fontes\n", currentRound)
		return simplePair(req, logsb)
	}

	halfway := numRounds / 2
	if currentRound < halfway {
		req.PairMethod = pb.PairMethod_PAIR_SWISS
		fmt.Fprintf(logsb, "Auto: round %d < halfway (%d), using Swiss\n", currentRound, halfway)
		return swissPair(req, logsb)
	}

	req.PairMethod = pb.PairMethod_COP
	fmt.Fprintf(logsb, "Auto: round %d >= halfway (%d), using COP\n", currentRound, halfway)
	return copMethodPair(req, logsb)
}

func restoreAutoPairReqFields(req *pb.PairRequest, origMethod pb.PairMethod, origInitialNonperfRounds int32) {
	req.PairMethod = origMethod
	req.InitialNonperfRounds = origInitialNonperfRounds
}

func copMinWeightMatching(req *pb.PairRequest, copdata *copdatapkg.PrecompData, factor3ForcedPairings [][2]int, logsb *strings.Builder) ([]int32, *pb.PairResponse) {
	prepairedPlayerIndexes, numForcedByes, prepairedRoundIdx := extractPrepairedPlayers(req)

	for playerIdx, oppIdx := range prepairedPlayerIndexes {
		if oppIdx < playerIdx {
			continue
		}
		prepairedPlayersStr := fmt.Sprintf("Forcing (#%d) %s vs ", playerIdx+1, req.PlayerNames[playerIdx])
		if playerIdx == oppIdx {
			prepairedPlayersStr += "BYE\n"
		} else {
			prepairedPlayersStr += fmt.Sprintf("(#%d) %s\n", oppIdx+1, req.PlayerNames[oppIdx])
		}
		logsb.WriteString(prepairedPlayersStr)
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
		roundPairingsRemaining:   int(req.Rounds) - copdata.CompletePairings,
		gibsonGetsBye:            gibsonGetsBye,
		prepairedRoundIdx:        prepairedRoundIdx,
		prepairedPlayerIndexes:   prepairedPlayerIndexes,
		factor3ForcedPairings:    factor3ForcedPairings,
	}

	logsb.WriteString(fmt.Sprintf("Control Loss Sims: %d\n", req.ControlLossSims))
	logsb.WriteString(fmt.Sprintf("Lowest Hopeful Casher: %s\n", req.PlayerNames[playerNodes[lowestPossibleHopeCasher]]))
	logsb.WriteString(fmt.Sprintf("Lowest Absolute Casher: %s\n", req.PlayerNames[playerNodes[lowestPossibleAbsCasher]]))
	logsb.WriteString(fmt.Sprintf("Number of Pairings (including prepaired): %d\n", len(req.DivisionPairings)))
	logsb.WriteString(fmt.Sprintf("Number of Results: %d\n", len(req.DivisionResults)))
	logsb.WriteString(fmt.Sprintf("Rounds Remaining: %d\n", pargs.roundsRemaining))
	logsb.WriteString(fmt.Sprintf("Round Pairings Remaining: %d\n", pargs.roundPairingsRemaining))
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

	matchupHeader := []string{"Player", "W", "S", "Player", "W", "S"}
	pairingDetailsheader := append(matchupHeader, []string{"S", "C", "PTP", "Total"}...)
	for _, weightPolicy := range weightPolicies {
		pairingDetailsheader = append(pairingDetailsheader, weightPolicy.name)
	}
	numColums := len(pairingDetailsheader)

	var pairings []int
	var totalWeight int64
	var pairingDetails [][]string
	var pairingsToDetailsIndex map[string]int

	retried := false
	for {
		edges := []*matching.Edge{}
		edgeWeights := map[string]int64{}
		pcEdgeWeights := map[string]int64{}
		pairingDetails = [][]string{}
		pairingsToDetailsIndex = map[string]int{}

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
						if weightPolicy.name == "PC" {
							rankKey := getRankPairingKey(rankIdxI, rankIdxJ)
							pcEdgeWeights[rankKey] = weight
						}
					}
					pairingDataRow[len(matchupHeader)+3] = fmt.Sprintf("%d", weightSum)
					rankKey := getRankPairingKey(rankIdxI, rankIdxJ)
					edgeWeights[rankKey] = weightSum
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

		var err error
		pairings, totalWeight, err = matching.MinWeightMatching(edges, true)
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

		// For each selected edge with weight >= majorPenalty, expand the PC contender
		// group for both players by incrementing LowestPossibleHopeNth, then retry once.
		// lowestPossibleHopeCasher (the global gate) is intentionally not updated.
		if retried {
			break
		}
		expanded := false
		numStandingsPlayers := copdata.Standings.GetNumPlayers()
		lowestHopeOverride := map[int]int{}
		for playerRankIdx, oppRankIdx := range pairings {
			if oppRankIdx <= playerRankIdx {
				continue
			}
			rankKey := getRankPairingKey(playerRankIdx, oppRankIdx)
			if pcEdgeWeights[rankKey] >= majorPenalty {
				nameForNode := func(rankIdx int) string {
					pi := playerNodes[rankIdx]
					if pi == pkgstnd.ByePlayerIndex {
						return "BYE"
					}
					return req.PlayerNames[pi]
				}
				logsb.WriteString(fmt.Sprintf("Retry: majorPenalty edge %s vs %s (weight %d)\n",
					nameForNode(playerRankIdx),
					nameForNode(oppRankIdx),
					edgeWeights[rankKey]))
				if playerRankIdx < len(copdata.LowestPossibleHopeNth) {
					current := copdata.LowestPossibleHopeNth[playerRankIdx]
					if override, ok := lowestHopeOverride[playerRankIdx]; ok {
						current = override
					}
					if current+1 < numStandingsPlayers {
						lowestHopeOverride[playerRankIdx] = current + 1
						expanded = true
					}
				}
			}
		}
		if !expanded {
			break
		}
		pargs.lowestHopeOverride = lowestHopeOverride
		retried = true
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
