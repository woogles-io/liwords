package copdata

import (
	"fmt"
	"math"
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
	HighestControlLossRankIdx int
	GibsonGroups              []int
}

func GetPrecompData(req *pb.PairRequest, logsb *strings.Builder) *PrecompData {
	standings := pkgstnd.CreateInitialStandings(req)

	writeStandingsToLog("Initial Standings", standings, req, logsb)
	// Use the initial results to get a tighter bound on the maximum factor
	initialFactor := int(req.AllPlayers) - 1
	initialSimResults := standings.SimFactorPairAll(req, int(req.DivisionSims), initialFactor, false, nil, logsb)

	writeFinalRankResultsToLog(fmt.Sprintf("Initial Sim Results (factor %d)", initialFactor), initialSimResults.FinalRanks, standings, req, logsb)

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
	maxFactor := 0
	for playerRankIdx := highestNongibsonizedRank + 1; playerRankIdx < numPlayers; playerRankIdx++ {
		if initialSimResults.FinalRanks[playerRankIdx][highestNongibsonizedRank] >= minWinsForHopeful {
			maxFactor++
		} else {
			break
		}
	}

	useControlLoss := req.UseControlLoss && !initialSimResults.GibsonizedPlayers[0]

	improvedFactorSimResults := standings.SimFactorPairAll(req, int(req.DivisionSims), maxFactor, useControlLoss, initialSimResults.SegmentRoundFactors, logsb)

	// FIXME: test this
	if improvedFactorSimResults == nil {
		improvedFactorSimResults = initialSimResults
		logsb.WriteString("\n\nNo factor improvement made.\n\n")
	} else {
		writeFinalRankResultsToLog(fmt.Sprintf("Improved Factor Sim Results (factor %d)", maxFactor), initialSimResults.FinalRanks, standings, req, logsb)
	}

	highestControlLossRankIdx := -1
	if improvedFactorSimResults.HighestControlLossRankIdx >= 0 &&
		1.0-float64(improvedFactorSimResults.LowestFactorPairWins)/float64(req.DivisionSims) >= req.ControlLossThreshold {
		highestControlLossRankIdx = improvedFactorSimResults.HighestControlLossRankIdx
	}

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

	writePrecompDataToLog("Precomp Data", improvedFactorSimResults, highestRankHopefully, highestRankAbsolutely, standings, req, logsb)

	return &PrecompData{
		Standings:                 standings,
		PairingCounts:             pairingCounts,
		RepeatCounts:              repeatCounts,
		HighestRankHopefully:      highestRankHopefully,
		HighestRankAbsolutely:     highestRankAbsolutely,
		HighestControlLossRankIdx: highestControlLossRankIdx,
		GibsonGroups:              improvedFactorSimResults.GibsonGroups,
	}
}

func GetPairingKey(playerIdx int, oppIdx int) string {
	var pairingKey string
	if playerIdx == oppIdx {
		pairingKey = fmt.Sprintf("%d:BYE", playerIdx)
	} else {
		if oppIdx > playerIdx {
			playerIdx, oppIdx = oppIdx, playerIdx
		}
		pairingKey = fmt.Sprintf("%d:%d", playerIdx, oppIdx)
	}
	return pairingKey
}

func writePrecompDataToLog(title string, simResults *pkgstnd.SimResults, highestRankHopefully []int, highestRankAbsolutely []int, standings *pkgstnd.Standings, req *pb.PairRequest, logsb *strings.Builder) {
	numPlayers := len(simResults.FinalRanks)
	matrix := make([][]string, numPlayers)

	useControlLoss := simResults.AllControlLosses != nil
	var header []string
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		if useControlLoss {
			matrix[rankIdx] = make([]string, 7)
			header = append(standingsHeader, []string{"Gb", "Gr", "H", "A", "CL1", "CLf", "D"}...)
		} else {
			matrix[rankIdx] = make([]string, 4)
			header = append(standingsHeader, []string{"Gb", "Gr", "H", "A"}...)
		}
		matrix[rankIdx][0] = boolToYesEmpty(simResults.GibsonizedPlayers[rankIdx])
		matrix[rankIdx][1] = strconv.Itoa(simResults.GibsonGroups[rankIdx] + 1)
		matrix[rankIdx][2] = strconv.Itoa(highestRankHopefully[rankIdx] + 1)
		matrix[rankIdx][3] = strconv.Itoa(highestRankAbsolutely[rankIdx] + 1)
		if useControlLoss {
			matrix[rankIdx][4] = strconv.Itoa(simResults.AllControlLosses[rankIdx][0])
			matrix[rankIdx][5] = strconv.Itoa(simResults.AllControlLosses[rankIdx][1])
			matrix[rankIdx][6] = boolToYesEmpty(simResults.HighestControlLossRankIdx == rankIdx)
		}
	}

	writeStringDataToLog(title, header, combineStringMatrices(standings.StringData(req), matrix), logsb)
}

func writeFinalRankResultsToLog(title string, finalRanks [][]int, standings *pkgstnd.Standings, req *pb.PairRequest, logsb *strings.Builder) {
	header := append([]string{}, standingsHeader[:]...)
	numPlayers := len(finalRanks)
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		header = append(header, strconv.Itoa(rankIdx+1))
	}

	finalRanksStr := make([][]string, numPlayers)
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		finalRanksStr[rankIdx] = make([]string, len(finalRanks[rankIdx]))
		for colIdx, value := range finalRanks[rankIdx] {
			finalRanksStr[rankIdx][colIdx] = strconv.Itoa(value)
		}
	}

	writeStringDataToLog(title, header, combineStringMatrices(standings.StringData(req), finalRanksStr), logsb)
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

func writeStringDataToLog(title string, header []string, data [][]string, logsb *strings.Builder) {
	titleLine := fmt.Sprintf("** %s **", title)
	border := strings.Repeat("*", len(titleLine))
	logsb.WriteString(fmt.Sprintf("%s\n%s\n%s\n\n", border, titleLine, border))
	logsb.WriteString(formatStringData(header, data))
}

func writeStandingsToLog(title string, standings *pkgstnd.Standings, req *pb.PairRequest, logsb *strings.Builder) {
	writeStringDataToLog(title, standingsHeader, standings.StringData(req), logsb)
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
