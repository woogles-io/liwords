package copdata

import (
	"context"
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

type PrecompData struct {
	Standings                 *pkgstnd.Standings
	PairingCounts             map[string]int
	RepeatCounts              []int
	HighestRankHopefully      []int
	HighestRankAbsolutely     []int
	LowestRankAbsolutely      []int
	HighestControlLossRankIdx int
	GibsonGroups              []int
	GibsonizedPlayers         []bool
}

func GetPrecompData(ctx context.Context, req *pb.PairRequest, copRand *rand.Rand, logsb *strings.Builder) (*PrecompData, pb.PairError) {
	standings := pkgstnd.CreateInitialStandings(req)

	// Use the initial results to get a tighter bound on the maximum factor
	initialFactor := int(req.Rounds) - len(req.DivisionResults)
	initialSimResults, pairErr := standings.SimFactorPairAll(ctx, req, copRand, int(req.DivisionSims), initialFactor, false, nil)
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
	minWinsForHopeful := int(math.Round(float64(req.DivisionSims) * req.HopefulnessThreshold))
	maxFactor := 0
	for playerRankIdx := highestNongibsonizedRank + 1; playerRankIdx < numPlayers; playerRankIdx++ {
		if initialSimResults.FinalRanks[playerRankIdx][highestNongibsonizedRank] >= minWinsForHopeful {
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
		improvedFactorSimResults, pairErr = standings.SimFactorPairAll(ctx, req, copRand, int(req.DivisionSims), maxFactor, false, initialSimResults.SegmentRoundFactors)
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

	var controlLossSimResults *pkgstnd.SimResults
	var allControlLosses map[int]int
	highestControlLossRankIdx := -1
	if req.UseControlLoss && !improvedFactorSimResults.GibsonizedPlayers[0] {
		controlLossSimResults, pairErr = standings.SimFactorPairAll(ctx, req, copRand, int(req.ControlLossSims), maxFactor, true, nil)
		if pairErr != pb.PairError_SUCCESS {
			return nil, pairErr
		}
		allControlLosses = controlLossSimResults.AllControlLosses
		if controlLossSimResults.HighestControlLossRankIdx >= 0 &&
			1.0-float64(controlLossSimResults.LowestFactorPairWins)/float64(req.ControlLossSims) >= req.ControlLossThreshold {
			highestControlLossRankIdx = controlLossSimResults.HighestControlLossRankIdx
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

	pairingCounts := make(map[string]int)
	repeatCounts := make([]int, int(req.AllPlayers))
	for _, roundPairings := range req.DivisionPairings {
		for playerIdx := 0; playerIdx < len(roundPairings.Pairings); playerIdx++ {
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

	writePrecompDataToLog("Precomp Data", improvedFactorSimResults, allControlLosses, highestRankHopefully, highestRankAbsolutely, standings, req, logsb)

	return &PrecompData{
		Standings:                 standings,
		PairingCounts:             pairingCounts,
		RepeatCounts:              repeatCounts,
		HighestRankHopefully:      highestRankHopefully,
		HighestRankAbsolutely:     highestRankAbsolutely,
		LowestRankAbsolutely:      lowestRankAbsolutely,
		HighestControlLossRankIdx: highestControlLossRankIdx,
		GibsonGroups:              improvedFactorSimResults.GibsonGroups,
		GibsonizedPlayers:         improvedFactorSimResults.GibsonizedPlayers,
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

func writePrecompDataToLog(title string, simResults *pkgstnd.SimResults, allControlLosses map[int]int, highestRankHopefully []int, highestRankAbsolutely []int, standings *pkgstnd.Standings, req *pb.PairRequest, logsb *strings.Builder) {
	numPlayers := len(highestRankHopefully)
	matrix := make([][]string, numPlayers)

	useControlLoss := allControlLosses != nil
	var header []string
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		if useControlLoss {
			matrix[rankIdx] = make([]string, 5)
			header = append(standingsHeader, []string{"Gb", "Gr", "H", "A", "CLf"}...)
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
			playerControlLosses, exists := allControlLosses[rankIdx]
			if exists {
				if playerControlLosses < 0 {
					matrix[rankIdx][4] = "-"
				} else {
					matrix[rankIdx][4] = strconv.Itoa(playerControlLosses)
				}
			}
		}
	}

	WriteStringDataToLog(title, header, combineStringMatrices(standings.StringData(req), matrix), logsb)
}

func writeFinalRankResultsToLog(title string, finalRanks [][]int, standings *pkgstnd.Standings, req *pb.PairRequest, logsb *strings.Builder) {
	header := append([]string{}, standingsHeader[:]...)
	numPlayers := standings.GetNumPlayers()
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		header = append(header, strconv.Itoa(rankIdx+1))
	}

	finalRanksStr := make([][]string, numPlayers)
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		finalRanksStr[rankIdx] = make([]string, numPlayers)
		for colIdx, value := range finalRanks[rankIdx] {
			finalRanksStr[rankIdx][colIdx] = strconv.Itoa(value)
		}
	}

	WriteStringDataToLog(title, header, combineStringMatrices(standings.StringData(req), finalRanksStr), logsb)
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
