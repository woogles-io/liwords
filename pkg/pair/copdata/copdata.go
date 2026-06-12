package copdata

import (
	"fmt"
	"math"
	"sort"

	"golang.org/x/exp/rand"

	"strconv"
	"strings"

	pkgstnd "github.com/woogles-io/liwords/pkg/pair/standings"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

var (
	standingsHeader = []string{"Rank", "Num", "Name", "Wins", "Spr"}
)

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
	writeFinalRankResultsToLog(fmt.Sprintf("Initial Sim Results (factor ceiling of %d)", initialFactor), initialSimResults.FinalRanks, standings, req, logsb)

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
		writeFinalRankResultsToLog(fmt.Sprintf("Improved Factor Sim Results (factor ceiling of %d)", maxFactor), improvedFactorSimResults.FinalRanks, standings, req, logsb)
	}

	minWinsForHopeful := int(math.Round(float64(improvedFactorSimResults.TotalSims) * req.HopefulnessThreshold))
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
			if winsSum >= minWinsForHopeful {
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
	numCompletePairings := 0
divisionPairingLoop:
	for _, roundPairings := range req.DivisionPairings {
		for playerIdx := range roundPairings.Pairings {
			oppIdx := int(roundPairings.Pairings[playerIdx])
			if oppIdx == -1 {
				break divisionPairingLoop
			}
		}
		numCompletePairings++
	}
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
	if numCompletePairings >= int(req.ControlLossActivationRound) && !improvedFactorSimResults.GibsonizedPlayers[0] && initialFactor > 1 {
		controlLossSimResults, pairErr = standings.SimFactorPairAll(req, copRand, int(req.ControlLossSims), maxFactor, lowestPossibleHopeNth[0], nil)
		if pairErr != pb.PairError_SUCCESS {
			return nil, pairErr
		}
		allControlLosses = controlLossSimResults.AllControlLosses
		vsFirstWins = controlLossSimResults.VsFirstWins
		if controlLossSimResults.HighestControlLossRankIdx >= 0 {
			destinysChild = controlLossSimResults.HighestControlLossRankIdx
		}
		writeControlLossDebugToLog(controlLossSimResults, allControlLosses, vsFirstWins, standings, req, logsb)
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

func writeFinalRankResultsToLog(title string, finalRanks [][]int, standings *pkgstnd.Standings, req *pb.PairRequest, logsb *strings.Builder) {
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

// writeControlLossDebugToLog prints the base factor pairings and, for each rank
// tested by the binary search, shows what opponent the tested player faces in
// vsFirst mode (always rank 1) vs vsFactor mode (swapped away from rank 1).
// Note: pairings shown are from initial standings; actual sims re-rank after each round.
func writeControlLossDebugToLog(simResults *pkgstnd.SimResults, allControlLosses map[int]int, vsFirstWins map[int]int, standings *pkgstnd.Standings, req *pb.PairRequest, logsb *strings.Builder) {
	roundsRemaining := pkgstnd.GetRoundsRemaining(req)
	numPlayers := standings.GetNumPlayers()

	// Safe name lookup: pairings can include a bye slot at index numPlayers (which
	// is out of range after standings shrinks back post-sim).
	rankName := func(rankIdx int) string {
		if rankIdx >= numPlayers {
			return "BYE"
		}
		playerIdx := standings.GetPlayerIndex(rankIdx)
		if playerIdx == pkgstnd.ByePlayerIndex {
			return "BYE"
		}
		return req.PlayerNames[playerIdx]
	}

	logsb.WriteString("** Control Loss Debug **\n\n")

	// Print base factor pairings.
	logsb.WriteString("Base factor pairings (ranks are 1-indexed, from initial standings):\n")
	for roundIdx := 0; roundIdx < roundsRemaining; roundIdx++ {
		logsb.WriteString(fmt.Sprintf("  Round %d:", roundIdx+1))
		pairings := simResults.Pairings[roundIdx]
		for pairIdx := 0; pairIdx < len(pairings); pairIdx += 2 {
			ri := pairings[pairIdx]
			rj := pairings[pairIdx+1]
			logsb.WriteString(fmt.Sprintf("  r%d(%s) vs r%d(%s)", ri+1, rankName(ri), rj+1, rankName(rj)))
		}
		logsb.WriteString("\n")
	}
	logsb.WriteString("\n")

	// For each rank tested by the binary search, show the vsFirst/vsFactor swap.
	tested := make([]int, 0, len(allControlLosses))
	for rankIdx := range allControlLosses {
		tested = append(tested, rankIdx)
	}
	sort.Ints(tested)

	for _, rankIdx := range tested {
		vsFirst := vsFirstWins[rankIdx]
		vsFactor := allControlLosses[rankIdx]
		diff := float64(vsFirst-vsFactor) * 100 / float64(req.ControlLossSims)
		threshold := float64(req.ControlLossThreshold) * 100
		status := ""
		if diff >= threshold {
			status = "  ← CONTROL LOSS"
		}
		logsb.WriteString(fmt.Sprintf("r%d (%s): vs1st=%d (%.1f%%)  vsFactor=%d (%.1f%%)  diff=%.1f%%  threshold=%.1f%%%s\n",
			rankIdx+1, rankName(rankIdx),
			vsFirst, float64(vsFirst)*100/float64(req.ControlLossSims),
			vsFactor, float64(vsFactor)*100/float64(req.ControlLossSims),
			diff, threshold, status))
		logsb.WriteString("  Per-round opponent (from initial standings; re-ranks mid-sim change this):\n")

		for roundIdx := 0; roundIdx < roundsRemaining; roundIdx++ {
			pairings := simResults.Pairings[roundIdx]

			// Find where rankIdx sits in this round's pairings.
			switchPairingIdx := -1
			for idx, r := range pairings {
				if r == rankIdx {
					switchPairingIdx = idx
					break
				}
			}

			// vsFirst: tested player is always swapped to position 1, so they play rank 0.
			vsFirstOppName := rankName(0)

			// vsFactor: if tested player is at position 1 (would face rank 0), swap them out.
			var vsFactorDesc string
			if switchPairingIdx == 1 {
				targetRank := 1
				if rankIdx == 1 {
					targetRank = 2
				}
				vsFactorOpp := -1
				for pairIdx := 0; pairIdx < len(pairings); pairIdx += 2 {
					if pairings[pairIdx] == targetRank {
						vsFactorOpp = pairings[pairIdx+1]
						break
					} else if pairings[pairIdx+1] == targetRank {
						vsFactorOpp = pairings[pairIdx]
						break
					}
				}
				vsFactorOppStr := "?"
				if vsFactorOpp >= 0 {
					vsFactorOppStr = fmt.Sprintf("r%d(%s)", vsFactorOpp+1, rankName(vsFactorOpp))
				}
				vsFactorDesc = fmt.Sprintf("%s  [was at r1's slot → r1 now plays r%d(%s)]", vsFactorOppStr, targetRank+1, rankName(targetRank))
			} else {
				// Already not at position 1; no swap in vsFactor mode.
				var origOpp int
				if switchPairingIdx%2 == 0 {
					origOpp = pairings[switchPairingIdx+1]
				} else {
					origOpp = pairings[switchPairingIdx-1]
				}
				vsFactorDesc = fmt.Sprintf("r%d(%s)  [no swap needed]", origOpp+1, rankName(origOpp))
			}

			logsb.WriteString(fmt.Sprintf("    Round %d: vsFirst→r1(%s)  vsFactor→%s\n", roundIdx+1, vsFirstOppName, vsFactorDesc))
		}
		logsb.WriteString("\n")
	}
}
