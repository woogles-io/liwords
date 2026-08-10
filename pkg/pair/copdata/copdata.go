package copdata

import (
	"fmt"
	"math"

	"golang.org/x/exp/rand"

	"strconv"
	"strings"

	pkgstnd "github.com/woogles-io/liwords/pkg/pair/standings"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

var (
	standingsHeader = []string{"Rank", "Num", "Name", "Wins", "Spr"}
)

const (
	// runawayLeadersWinPctThreshold is the combined win% for the leader and
	// 2nd place above which other players only need half as many simulated
	// wins to be considered hopeful contenders for 1st or 2nd.
	runawayLeadersWinPctThreshold = 0.80
)

// IsLastQuarter reports whether roundPairingsRemaining rounds still to be
// paired puts the tournament in its last quarter of rounds.
func IsLastQuarter(roundPairingsRemaining int, rounds int32) bool {
	return roundPairingsRemaining*4 <= int(rounds)
}

type PrecompData struct {
	Standings     *pkgstnd.Standings
	PairingCounts map[string]int

	// Indexed by player id
	RepeatCounts []int

	// The remaining following fields are indexed by player rank

	HighestRankHopefully  []int
	HighestRankAbsolutely []int
	LowestRankAbsolutely  []int
	LowestPossibleHopeNth []int
	DestinysChild         int
	GibsonGroups          []int
	GibsonizedPlayers     []bool
	CompletePairings      int
}

func GetPrecompData(req *pb.PairRequest, copRand *rand.Rand, logsb *strings.Builder) (*PrecompData, pb.PairError) {
	standings := pkgstnd.CreateInitialStandings(req)

	// Use the initial results to get a tighter bound on the maximum factor
	initialFactor := pkgstnd.GetRoundsRemaining(req)
	initialSimResults, pairErr := standings.SimFactorPairAll(req, copRand, int(req.DivisionSims), initialFactor, -1, nil)
	if pairErr != pb.PairError_SUCCESS {
		return nil, pairErr
	}
	WriteFinalRankResultsToLog(fmt.Sprintf("Initial Sim Results (factor ceiling of %d)", initialFactor), initialSimResults.FinalRanks, standings, req, logsb)

	numPlayers := standings.GetNumPlayers()

	// Attempt to tighten the max factor.
	// When tightening the factor bounds, the  maximum factor should
	// be with respect to the highest non-gibsonized rank.
	highestNongibsonizedRank := 0
	for rankIdx, isGibsonized := range initialSimResults.GibsonizedPlayers {
		if !isGibsonized {
			highestNongibsonizedRank = rankIdx
			break
		}
	}

	// If the initial factor is N, then 1st played (1 + N)th, but if (1 + N)th never achieved
	// 1st, then (1 + N)th should never have played 1st at all, so we use the initial results
	// to get a tighter bound on the maximum factor.
	maxFactor := 0
	for playerRankIdx := highestNongibsonizedRank + 1; playerRankIdx < numPlayers; playerRankIdx++ {
		if initialSimResults.FinalRanks[playerRankIdx][highestNongibsonizedRank] > 0 {
			maxFactor++
		} else {
			break
		}
	}

	// Get the number of players in the highest gibson group where the factor would be applied
	numPlayersInhighestNongibsonGroup := 0
	highestNongibsonizedRankGroup := initialSimResults.GibsonGroups[highestNongibsonizedRank]
	for rankIdx := highestNongibsonizedRank; rankIdx < numPlayers; rankIdx++ {
		if initialSimResults.GibsonGroups[rankIdx] == highestNongibsonizedRankGroup {
			numPlayersInhighestNongibsonGroup++
		} else {
			break
		}
	}

	var improvedFactorSimResults *pkgstnd.SimResults

	// Only re-sim with the improved bound if it actually makes the max factor smaller
	// for the highest gibson group.
	if maxFactor*2 < numPlayersInhighestNongibsonGroup {
		improvedFactorSimResults, pairErr = standings.SimFactorPairAll(req, copRand, int(req.DivisionSims), maxFactor, -1, initialSimResults.SegmentRoundFactors)
		if pairErr != pb.PairError_SUCCESS {
			return nil, pairErr
		}
	}

	if improvedFactorSimResults == nil {
		improvedFactorSimResults = initialSimResults
		logsb.WriteString("\n\nNo factor improvement made.\n\n")
	} else {
		WriteFinalRankResultsToLog(fmt.Sprintf("Improved Factor Sim Results (factor ceiling of %d)", maxFactor), improvedFactorSimResults.FinalRanks, standings, req, logsb)
	}

	// numCompletePairings is needed early to compute the leader/2nd win% below;
	// it's recomputed (identically) further down alongside pairingCounts.
	numCompletePairings := 0
completePairingsLoop:
	for _, roundPairings := range req.DivisionPairings {
		for playerIdx := range roundPairings.Pairings {
			oppIdx := int(roundPairings.Pairings[playerIdx])
			if oppIdx == -1 {
				break completePairingsLoop
			}
		}
		numCompletePairings++
	}

	minWinsForHopeful := int(math.Round(float64(improvedFactorSimResults.TotalSims) * req.HopefulnessThreshold))
	// If the leader and 2nd place have run away with the tournament (their
	// combined probability of finishing 1st exceeds 80%), other players only
	// need half as many simulated wins to be considered hopeful contenders
	// for 1st or 2nd.
	//
	// This must be computed from the simulated probability of finishing in
	// 1st place (FinalRanks[...][0]/TotalSims), not from either player's
	// actual completed-game win rate: only one player can finish 1st, so the
	// simulated figure is a single probability bounded at 100%, while two
	// independent win rates can each approach 100% and sum past it.
	halfMinWinsForHopeful := int(math.Round(float64(minWinsForHopeful) / 2.0))
	runawayLeaders := false
	if numPlayers >= 2 && improvedFactorSimResults.TotalSims > 0 {
		leaderFirstPct := float64(improvedFactorSimResults.FinalRanks[0][0]) / float64(improvedFactorSimResults.TotalSims)
		secondFirstPct := float64(improvedFactorSimResults.FinalRanks[1][0]) / float64(improvedFactorSimResults.TotalSims)
		if leaderFirstPct+secondFirstPct > runawayLeadersWinPctThreshold {
			runawayLeaders = true
			logsb.WriteString(fmt.Sprintf(
				"Leader+2nd combined 1st-place%% (%.1f%%) > %.0f%%: halving the hopeful-for-1st/2nd bar to %.0f%% (normally %.0f%%)\n",
				(leaderFirstPct+secondFirstPct)*100, runawayLeadersWinPctThreshold*100,
				float64(halfMinWinsForHopeful)/float64(improvedFactorSimResults.TotalSims)*100,
				float64(minWinsForHopeful)/float64(improvedFactorSimResults.TotalSims)*100))
		}
	}
	highestRankHopefully := make([]int, numPlayers)
	highestRankAbsolutely := make([]int, numPlayers)
	lowestRankAbsolutely := make([]int, numPlayers)
	for playerRankIdx := 0; playerRankIdx < numPlayers; playerRankIdx++ {
		winsSum := 0
		hopefulRank := numPlayers - 1
		absoluteRank := numPlayers - 1
		for rank := 0; rank < numPlayers; rank++ {
			rankSum := improvedFactorSimResults.FinalRanks[playerRankIdx][rank]
			if winsSum == 0 && rankSum > 0 {
				absoluteRank = rank
			}
			winsSum += rankSum
			threshold := minWinsForHopeful
			if runawayLeaders && rank <= 1 {
				threshold = halfMinWinsForHopeful
			}
			if winsSum >= threshold {
				hopefulRank = rank
				break
			}
		}
		highestRankHopefully[playerRankIdx] = hopefulRank
		highestRankAbsolutely[playerRankIdx] = absoluteRank
		lowestRank := 0
		for rank := numPlayers - 1; rank >= 0; rank-- {
			rankSum := improvedFactorSimResults.FinalRanks[playerRankIdx][rank]
			if rankSum > 0 {
				lowestRank = rank
				break
			}
		}
		lowestRankAbsolutely[playerRankIdx] = lowestRank
	}

	// In the last quarter, an odd number of hopeful-to-cash players leaves one of
	// them with no hopeful-to-cash opponent (the CC weight policy major-penalizes
	// pairing a hopeful-to-cash player against a non-hopeful one). Fix the parity
	// here, at the source, by promoting the highest-ranked non-hopeful player to
	// hopeful for the lowest cash position - the same fix computeDisallowedLeaderOpponent
	// makes for the hopeful-for-1st contender group, but folded into the data
	// instead of layered on as a pairing constraint.
	//
	// This promotion still matters in the true final round even though CC's
	// hopeful-vs-hopeful grouping rule is disabled there (see isFinalRound in
	// cop.go): lowestPossibleHopeCasher, which this feeds via
	// LowestPossibleHopeNth, is also read unconditionally (not gated by
	// last-quarter/final-round) by the BB and computeForcedContenderBye
	// policies in cop.go, so it must stay correct in every round.
	roundPairingsRemaining := int(req.Rounds) - numCompletePairings
	if IsLastQuarter(roundPairingsRemaining, req.Rounds) {
		numHopefulToCash := 0
		highestNonHopefulRankIdx := -1
		for playerRankIdx, place := range highestRankHopefully {
			if place < int(req.PlacePrizes) {
				numHopefulToCash++
			} else if highestNonHopefulRankIdx == -1 {
				highestNonHopefulRankIdx = playerRankIdx
			}
		}
		if numHopefulToCash%2 == 1 && highestNonHopefulRankIdx >= 0 {
			lowestCashPlace := int(req.PlacePrizes) - 1
			logsb.WriteString(fmt.Sprintf(
				"Odd number of hopeful-to-cash players (%d) in the last quarter: %s (rank %d) altered from hopeful-for-%d to hopeful-for-%d (lowest cash position)\n",
				numHopefulToCash, req.PlayerNames[standings.GetPlayerIndex(highestNonHopefulRankIdx)], highestNonHopefulRankIdx+1,
				highestRankHopefully[highestNonHopefulRankIdx]+1, lowestCashPlace+1))
			highestRankHopefully[highestNonHopefulRankIdx] = lowestCashPlace
		}
	}

	lowestPossibleHopeNth := make([]int, len(highestRankHopefully))
	prevPlace := 0
	for playerRankIdx, place := range highestRankHopefully {
		if playerRankIdx > lowestPossibleHopeNth[place] {
			lowestPossibleHopeNth[place] = playerRankIdx
		}
		for i := prevPlace + 1; i < place; i++ {
			lowestPossibleHopeNth[i] = playerRankIdx - 1
		}
		prevPlace = place
	}
	for i := prevPlace + 1; i < len(highestRankHopefully); i++ {
		lowestPossibleHopeNth[i] = len(highestRankHopefully) - 1
	}

	pairingCounts := make(map[string]int)
	repeatCounts := make([]int, int(req.AllPlayers))
	for roundIdx := range numCompletePairings {
		roundPairings := req.DivisionPairings[roundIdx]
		for playerIdx := range roundPairings.Pairings {
			oppIdx := int(roundPairings.Pairings[playerIdx])

			if oppIdx > playerIdx {
				continue
			}
			pairingKey := GetPairingKey(playerIdx, oppIdx)
			if pairingCounts[pairingKey] > 0 {
				repeatCounts[playerIdx]++
				if playerIdx != oppIdx {
					repeatCounts[oppIdx]++
				}
			}
			pairingCounts[pairingKey]++
		}
	}

	var controlLossSimResults *pkgstnd.SimResults
	var allControlLosses map[int]int
	var vsFirstWins map[int]int
	destinysChild := -1
	if numCompletePairings >= int(req.ControlLossActivationRound) && !improvedFactorSimResults.GibsonizedPlayers[0] && initialFactor > 1 && maxFactor > 0 {
		controlLossSimResults, pairErr = standings.SimFactorPairAll(req, copRand, int(req.ControlLossSims), maxFactor, lowestPossibleHopeNth[0], nil)
		if pairErr != pb.PairError_SUCCESS {
			return nil, pairErr
		}
		allControlLosses = controlLossSimResults.AllControlLosses
		vsFirstWins = controlLossSimResults.VsFirstWins
		if controlLossSimResults.HighestControlLossRankIdx >= 0 {
			destinysChild = controlLossSimResults.HighestControlLossRankIdx
		}
	}

	writePrecompDataToLog("Precomp Data", improvedFactorSimResults, allControlLosses, vsFirstWins, highestRankHopefully, highestRankAbsolutely, standings, req, logsb)

	return &PrecompData{
		Standings:             standings,
		PairingCounts:         pairingCounts,
		RepeatCounts:          repeatCounts,
		HighestRankHopefully:  highestRankHopefully,
		HighestRankAbsolutely: highestRankAbsolutely,
		LowestRankAbsolutely:  lowestRankAbsolutely,
		LowestPossibleHopeNth: lowestPossibleHopeNth,
		DestinysChild:         destinysChild,
		GibsonGroups:          improvedFactorSimResults.GibsonGroups,
		GibsonizedPlayers:     improvedFactorSimResults.GibsonizedPlayers,
		CompletePairings:      numCompletePairings,
	}, pb.PairError_SUCCESS
}

func GetPairingKey(playerIdx int, oppIdx int) string {
	var pairingKey string
	if playerIdx == oppIdx || oppIdx == pkgstnd.ByePlayerIndex {
		pairingKey = fmt.Sprintf("%d:BYE", playerIdx)
	} else {
		if oppIdx > playerIdx {
			playerIdx, oppIdx = oppIdx, playerIdx
		}
		pairingKey = fmt.Sprintf("%d:%d", playerIdx, oppIdx)
	}
	return pairingKey
}

func writePrecompDataToLog(title string, simResults *pkgstnd.SimResults, allControlLosses map[int]int, vsFirstWins map[int]int, highestRankHopefully []int, highestRankAbsolutely []int, standings *pkgstnd.Standings, req *pb.PairRequest, logsb *strings.Builder) {
	numPlayers := len(highestRankHopefully)
	matrix := make([][]string, numPlayers)

	useControlLoss := allControlLosses != nil
	var header []string
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		if useControlLoss {
			matrix[rankIdx] = make([]string, 6)
			header = append(standingsHeader, []string{"Gb", "Gr", "H", "A", "vs1st", "vsFactor"}...)
		} else {
			matrix[rankIdx] = make([]string, 4)
			header = append(standingsHeader, []string{"Gb", "Gr", "H", "A"}...)
		}
		matrix[rankIdx][0] = boolToYesEmpty(simResults.GibsonizedPlayers[rankIdx])
		matrix[rankIdx][1] = strconv.Itoa(simResults.GibsonGroups[rankIdx] + 1)
		matrix[rankIdx][2] = strconv.Itoa(highestRankHopefully[rankIdx] + 1)
		matrix[rankIdx][3] = strconv.Itoa(highestRankAbsolutely[rankIdx] + 1)
		if useControlLoss {
			matrix[rankIdx][4] = ""
			matrix[rankIdx][5] = ""
			vsFirstWinsCount, exists := vsFirstWins[rankIdx]
			if exists {
				if vsFirstWinsCount < 0 {
					matrix[rankIdx][4] = "-"
				} else {
					matrix[rankIdx][4] = strconv.Itoa(vsFirstWinsCount)
				}
			}
			playerControlLosses, exists := allControlLosses[rankIdx]
			if exists {
				if playerControlLosses < 0 {
					matrix[rankIdx][5] = "-"
				} else {
					matrix[rankIdx][5] = strconv.Itoa(playerControlLosses)
				}
			}
		}
	}

	WriteStringDataToLog(title, header, combineStringMatrices(standings.StringData(req), matrix), logsb)
}

func WriteFinalRankResultsToLog(title string, finalRanks [][]int, standings *pkgstnd.Standings, req *pb.PairRequest, logsb *strings.Builder) {
	header := append([]string{}, standingsHeader[:]...)
	numPlayers := standings.GetNumPlayers()
	totalSims := 0
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		header = append(header, strconv.Itoa(rankIdx+1))
		totalSims += finalRanks[rankIdx][0]
	}

	finalRanksStrPct := make([][]string, numPlayers)
	finalRanksStrRaw := make([][]string, numPlayers)
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		finalRanksStrPct[rankIdx] = make([]string, numPlayers)
		finalRanksStrRaw[rankIdx] = make([]string, numPlayers)
		for colIdx, value := range finalRanks[rankIdx] {
			finalRanksStrPct[rankIdx][colIdx] = fmt.Sprintf("%.4f%%", float64(value)*100/float64(totalSims))
			finalRanksStrRaw[rankIdx][colIdx] = fmt.Sprintf("%d", value)
		}
	}

	WriteStringDataToLog(title, header, combineStringMatrices(standings.StringData(req), finalRanksStrPct), logsb)
	WriteStringDataToLog("Totals", header, combineStringMatrices(standings.StringData(req), finalRanksStrRaw), logsb)
	logsb.WriteString(fmt.Sprintf("Total Sims: %d\n\n", totalSims))
}

func formatStringData(header []string, data [][]string) string {
	numRows := len(data)
	numCols := len(header)

	for rowIdx := 0; rowIdx < numRows; rowIdx++ {
		if len(data[rowIdx]) != numCols {
			return fmt.Sprintf("row %d has %d columns, expected %d", rowIdx, len(data[rowIdx]), numCols)
		}
	}

	maxColumnWidths := make([]int, numCols)
	for colIdx := 0; colIdx < numCols; colIdx++ {
		if len(header[colIdx]) > maxColumnWidths[colIdx] {
			maxColumnWidths[colIdx] = len(header[colIdx])
		}
		for rowIdx := 0; rowIdx < numRows; rowIdx++ {
			if len(data[rowIdx][colIdx]) > maxColumnWidths[colIdx] {
				maxColumnWidths[colIdx] = len(data[rowIdx][colIdx])
			}
		}
	}

	var sb strings.Builder

	for colIdx := 0; colIdx < numCols; colIdx++ {
		sb.WriteString(fmt.Sprintf("%-*s", maxColumnWidths[colIdx], header[colIdx]))
		if colIdx < numCols-1 {
			sb.WriteString(" | ")
		}
	}

	sb.WriteString("\n" + strings.Repeat("-", sb.Len()) + "\n")

	for rowIdx := 0; rowIdx < numRows; rowIdx++ {
		for colIdx := 0; colIdx < numCols; colIdx++ {
			sb.WriteString(fmt.Sprintf("%-*s", maxColumnWidths[colIdx], data[rowIdx][colIdx]))
			if colIdx < numCols-1 {
				sb.WriteString(" | ")
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	return sb.String()
}

func WriteStringDataToLog(title string, header []string, data [][]string, logsb *strings.Builder) {
	titleLine := fmt.Sprintf("** %s **", title)
	border := strings.Repeat("*", len(titleLine))
	logsb.WriteString(fmt.Sprintf("%s\n%s\n%s\n\n", border, titleLine, border))
	logsb.WriteString(formatStringData(header, data))
}

func combineStringMatrices(m1, m2 [][]string) [][]string {
	if len(m1) != len(m2) {
		return [][]string{}
	}

	// Create a new matrix to hold the combined rows
	rowCount := len(m1)
	combined := make([][]string, rowCount)

	// Combine rows from m1 and m2
	for i := 0; i < rowCount; i++ {
		combined[i] = append(m1[i], m2[i]...)
	}

	return combined
}

func boolToYesEmpty(value bool) string {
	if value {
		return "Yes"
	}
	return ""
}
