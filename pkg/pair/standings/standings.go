package standings

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"golang.org/x/exp/rand"
)

const (
	playerWinsOffset   int    = 48
	initialWinsValue   int    = 1 << (64 - playerWinsOffset - 1)
	playerSpreadOffset int    = 16
	initialSpreadValue int    = 1 << (playerWinsOffset - playerSpreadOffset - 1)
	playerIndexMask    uint64 = 0xFFFF
	maxSpread          int    = 300
	byeSpread          int    = 50
)

type Standings struct {
	records            []uint64
	recordsBackup      []uint64
	possibleResults    []uint64
	roundsPlayed       int
	roundsPlayedBackup int
}

// Exported functions

func CreateInitialStandings(req *pb.PairRequest) *Standings {
	// Create empty standings
	standings := &Standings{}
	standings.roundsPlayed = len(req.DivisionResults)
	standings.records = make([]uint64, int(req.AllPlayers))
	for playerIdx := 0; playerIdx < int(req.AllPlayers); playerIdx++ {
		standings.records[playerIdx] = getRecordFromWinsAndSpread(initialWinsValue, initialSpreadValue)
		standings.records[playerIdx] += uint64(playerIdx)
	}

	// Initialize the possible results
	standings.possibleResults = make([]uint64, maxSpread+1)
	for spread := 1; spread < maxSpread; spread++ {
		standings.possibleResults[spread] = getRecordFromWinsAndSpread(1, spread)
	}

	// Update the standings wuth the pairing results
	for roundIdx, roundResults := range req.DivisionResults {
		for playerIdx, playerScore := range roundResults.Results {
			oppIdx := int(req.DivisionPairings[roundIdx].Pairings[playerIdx])
			pScore := int(playerScore)
			if playerIdx == int(oppIdx) {
				// Bye
				if pScore > 0 {
					standings.incrementPlayerRecord(playerIdx, getRecordFromWinsAndSpread(1, pScore))
				} else if pScore < 0 {
					standings.decrementPlayerRecord(playerIdx, getRecordFromWinsAndSpread(1, -pScore))
				}
			} else if playerIdx < oppIdx {
				playerSpread := pScore - int(roundResults.Results[oppIdx])
				if playerSpread > 0 {
					record := getRecordFromWinsAndSpread(1, playerSpread)
					standings.incrementPlayerRecord(playerIdx, record)
					standings.decrementPlayerRecord(oppIdx, record)
				} else if playerSpread < 0 {
					record := getRecordFromWinsAndSpread(1, -playerSpread)
					standings.incrementPlayerRecord(oppIdx, record)
					standings.decrementPlayerRecord(playerIdx, record)
				}
			}
		}
	}

	validRecords := make([]uint64, int(req.ValidPlayers))
	validPlayersIdx := 0
	removedPlayersMap := make(map[int]struct{}, len(req.RemovedPlayers))
	for _, removedPlayer := range req.RemovedPlayers {
		removedPlayersMap[int(removedPlayer)] = struct{}{}
	}
	for playerIdx := 0; playerIdx < int(req.AllPlayers); playerIdx++ {
		if _, exists := removedPlayersMap[playerIdx]; exists {
			continue
		}
		validRecords[validPlayersIdx] = standings.records[playerIdx]
		validPlayersIdx++
	}

	standings.records = validRecords
	standings.recordsBackup = make([]uint64, int(req.ValidPlayers))

	standings.Sort()

	return standings
}

func (standings *Standings) Copy() *Standings {
	standingsCopy := &Standings{
		records:            make([]uint64, len(standings.records)),
		recordsBackup:      make([]uint64, len(standings.recordsBackup)),
		possibleResults:    standings.possibleResults,
		roundsPlayed:       standings.roundsPlayed,
		roundsPlayedBackup: standings.roundsPlayedBackup,
	}
	copy(standingsCopy.records, standings.records)
	copy(standingsCopy.recordsBackup, standings.recordsBackup)
	return standingsCopy
}

func (standings *Standings) Backup() {
	copy(standings.recordsBackup, standings.records)
	standings.roundsPlayedBackup = standings.roundsPlayed
}

func (standings *Standings) RestoreFromBackup() {
	copy(standings.records, standings.recordsBackup)
	standings.roundsPlayed = standings.roundsPlayedBackup
}

func (standings *Standings) GetPlayerIndex(rankIdx int) int {
	return getIndex(standings.records[rankIdx])
}

func (standings *Standings) GetPlayerWins(rankIdx int) float64 {
	return float64(getWinsValue(standings.records[rankIdx])-initialWinsValue+standings.roundsPlayed) / 2
}

func (standings *Standings) GetPlayerSpread(rankIdx int) int {
	spread := getSpreadValue(standings.records[rankIdx])
	if spread > uint64(initialSpreadValue) {
		return int(spread - uint64(initialSpreadValue))
	} else {
		return -int(uint64(initialSpreadValue) - spread)
	}
}

// Assumes the standings are already sorted and i < j
func (standings *Standings) CanCatch(roundsRemaining int, cumeGibsonSpread int, i int, j int) bool {
	ri := standings.records[i]
	rj := standings.records[j]
	winDiff := getWinsValue(ri) - getWinsValue(rj)
	highestPossibleWinScore := roundsRemaining * 2
	if winDiff != highestPossibleWinScore {
		return winDiff < highestPossibleWinScore
	}
	piSpread := getSpreadValue(ri)
	pjSpread := getSpreadValue(rj)
	if pjSpread >= piSpread {
		return true
	}
	return (piSpread - pjSpread) <= uint64(cumeGibsonSpread)
}

// Assumes the standings are already sorted
func (standings *Standings) GetGibsonizedPlayers(req *pb.PairRequest) []bool {
	numPlayers := len(standings.records)
	gibsonizedPlayers := make([]bool, numPlayers)
	roundsRemaining := int(req.Rounds) - len(req.DivisionResults)
	numInputGibonsSpreads := len(req.GibsonSpreads)
	cumeGibsonSpread := 0
	for round := roundsRemaining - 1; round >= 0; round-- {
		// FIXME: should these be multiplied by 2?
		if round >= numInputGibonsSpreads {
			cumeGibsonSpread += int(req.GibsonSpreads[numInputGibonsSpreads-1]) * 2
		} else {
			cumeGibsonSpread += int(req.GibsonSpreads[round]) * 2
		}
	}
	for playerIdx := 0; playerIdx < int(req.PlacePrizes); playerIdx++ {
		gibsonizedPlayers[playerIdx] = true
		if playerIdx > 0 && standings.CanCatch(roundsRemaining, cumeGibsonSpread, playerIdx-1, playerIdx) {
			gibsonizedPlayers[playerIdx] = false
			continue
		}
		if playerIdx < numPlayers-1 && standings.CanCatch(roundsRemaining, cumeGibsonSpread, playerIdx, playerIdx+1) {
			gibsonizedPlayers[playerIdx] = false
			continue
		}
	}
	return gibsonizedPlayers
}

func (standings *Standings) Sort() {
	sort.Slice(standings.records, func(i, j int) bool {
		return standings.records[i] > standings.records[j]
	})
}

func (standings *Standings) GetAllSegments(gibsonizedPlayers []bool) [][]int {
	// FIXME: probably need better names for i and j
	numPlayers := len(standings.records)
	i := 0
	j := 0
	pairingsSegments := [][]int{}
	for j <= numPlayers {
		if j == numPlayers {
			pairingsSegments = append(pairingsSegments, []int{i, j})
			break
		}
		if gibsonizedPlayers[j] {
			if j != i {
				// FIXME: explain this
				if (j-i)%2 == 1 {
					pairingsSegments = append(pairingsSegments, []int{i, j + 1})
				} else {
					pairingsSegments = append(pairingsSegments, []int{i, j})

				}
			}
			j++
			i = j
		} else {
			j++
		}
	}
	return pairingsSegments
}

// Assumes the standings are already sorted
// FIXME: calculate the gibsonizedPlayers in this function instead of passing it in
func (standings *Standings) SimFactorPair(req *pb.PairRequest, sims int, maxFactor int, roundsRemaining int, gibsonizedPlayers []bool) ([][]int, string) {
	numPlayers := len(standings.records)
	results := make([][]int, numPlayers)
	for i := range results {
		results[i] = make([]int, numPlayers)
	}
	playerIdxToRankIdx := map[int]int{}
	for i := 0; i < len(standings.records); i++ {
		playerIdxToRankIdx[standings.GetPlayerIndex(i)] = i
	}
	standings.simFactorPairSegments(results, playerIdxToRankIdx, standings.GetAllSegments(gibsonizedPlayers), roundsRemaining, maxFactor, sims)
	return results, standings.resultsString(results, playerIdxToRankIdx, req)
}

func (standings *Standings) SimSingleIteration(pairings [][]int, roundsRemaining int, i int, j int) {
	numPlayers := len(pairings[0])
	segmentHasOddNumPlayers := numPlayers%2 == 1
	divisionHasOddNumPlayers := len(standings.records)%2 == 1
	for roundIdx := 0; roundIdx < roundsRemaining; roundIdx++ {
		for pairIdx := 0; pairIdx < numPlayers-1; pairIdx += 2 {
			randomResult := rand.Intn(2)
			p1 := pairings[roundIdx][pairIdx]
			p2 := pairings[roundIdx][pairIdx+1]
			winner := p1*(1-randomResult) + p2*randomResult
			loser := p2*(1-randomResult) + p1*randomResult
			// FIXME: limit by the gibson spread for this round
			randomSpread := rand.Intn(maxSpread + 1)
			record := standings.possibleResults[randomSpread]
			standings.incrementPlayerRecord(winner, record)
			standings.decrementPlayerRecord(loser, record)
		}
		// FIXME: branches could be slow, think about this some more
		if segmentHasOddNumPlayers {
			bottomPlayerIdx := pairings[roundIdx][numPlayers-1]
			if divisionHasOddNumPlayers {
				standings.incrementPlayerRecord(bottomPlayerIdx, standings.possibleResults[byeSpread])
			} else {
				randomSpread := rand.Intn(maxSpread + 1)
				record := standings.possibleResults[randomSpread]
				if rand.Intn(2) == 0 {
					standings.incrementPlayerRecord(bottomPlayerIdx, record)
				} else {
					standings.decrementPlayerRecord(bottomPlayerIdx, record)
				}
			}
		}
		// FIXME: is sort range inclusive?
		standings.sortRange(i, j)
	}
	standings.roundsPlayed += roundsRemaining
}

func (standings *Standings) UpdateResultsWithFinishedSim(results [][]int, playerIdxToRankIdx map[int]int) {
	numRecords := len(standings.records)
	for rankIdx := 0; rankIdx < numRecords; rankIdx++ {
		results[playerIdxToRankIdx[getIndex(standings.records[rankIdx])]][rankIdx] += 1
	}
}

// Gets the factor pairings for players in for all remaining rounds [i, j)
// Assumes i < j
// Returns pairings in pairs of player indexes
// For example, pairings of [0, 2, 1, 3] indicate player 0 plays player 4
// and player 1 plays player 3.
func GetPairingsForSegment(i int, j int, totalRoundsRemaining int, maxFactor int) [][]int {
	numPlayers := j - i
	pairings := make([][]int, totalRoundsRemaining)
	for i := 0; i < totalRoundsRemaining; i++ {
		pairings[i] = make([]int, numPlayers)
	}

	for roundsRemaining := totalRoundsRemaining; roundsRemaining > 0; roundsRemaining-- {
		roundFactor := roundsRemaining
		if roundFactor > maxFactor {
			roundFactor = maxFactor
		}
		maxPlayerFactor := numPlayers / 2
		if roundFactor > maxPlayerFactor {
			roundFactor = maxPlayerFactor
		}
		roundIdx := totalRoundsRemaining - roundsRemaining
		pairIdx := 0
		for k := i; k < roundFactor+i; k++ {
			pairings[roundIdx][pairIdx] = k
			pairIdx += 1
			pairings[roundIdx][pairIdx] = k + roundFactor
			pairIdx += 1
		}
		for k := roundFactor*2 + i; k < numPlayers+i; k += 2 {
			pairings[roundIdx][pairIdx] = k
			pairIdx += 1
			if pairIdx == numPlayers {
				break
			}
			pairings[roundIdx][pairIdx] = k + 1
			pairIdx += 1
		}
	}
	return pairings
}

func (standings *Standings) String(req *pb.PairRequest) string {
	var playerNames []string

	maxNameLength := 0
	if req != nil {
		for _, playerName := range req.PlayerNames {
			if len(playerName) > maxNameLength {
				if len(playerName) > 30 {
					maxNameLength = 30
				} else {
					maxNameLength = len(playerName)
				}
			}
		}
		playerNames = req.PlayerNames
	} else {
		playerNames = make([]string, len(standings.records))
		for i := 0; i < len(standings.records); i++ {
			playerNames[i] = strconv.Itoa(i + 1)
		}
		maxNameLength = len(playerNames[len(standings.records)-1])
	}
	numPlayers := len(standings.records)

	playerNameColWidth := maxNameLength
	if playerNameColWidth > 30 {
		playerNameColWidth = 30
	}

	headerFormat := fmt.Sprintf("%%-4s | %%-%ds | %%-4s | %%-6s\n", playerNameColWidth)
	rowFormat := fmt.Sprintf("%%-4d | %%-%ds | %%-4.1f | %%-6d\n", playerNameColWidth)

	var sb strings.Builder
	header := fmt.Sprintf(headerFormat, "Rank", "Name", "Wins", "Spread")
	sb.WriteString(header)
	sb.WriteString(strings.Repeat("-", len(header)) + "\n")

	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		playerIdx := standings.GetPlayerIndex(rankIdx)
		wins := standings.GetPlayerWins(rankIdx)
		spread := standings.GetPlayerSpread(rankIdx)
		playerName := playerNames[playerIdx]
		if len(playerName) > 30 {
			playerName = playerName[:30]
		}
		sb.WriteString(fmt.Sprintf(rowFormat, rankIdx+1, playerName, wins, spread))
	}

	return sb.String()
}

func (standings *Standings) resultsString(results [][]int, playerIdxToRankIdx map[int]int, req *pb.PairRequest) string {
	numPlayers := len(results)
	var builder strings.Builder
	nameColumnWidth := 30
	resultColumnWidth := 8

	rankColumnWidth := len(fmt.Sprintf("%d", numPlayers))

	builder.WriteString(fmt.Sprintf("%-*s", rankColumnWidth+1, ""))
	builder.WriteString(fmt.Sprintf("%-*s", nameColumnWidth, ""))
	for pos := 1; pos <= numPlayers; pos++ {
		builder.WriteString(fmt.Sprintf("%-8d", pos))
	}
	builder.WriteString("\n")

	for i, playerRecord := range standings.records {
		playerIdx := getIndex(playerRecord)
		playerName := req.PlayerNames[playerIdx]

		builder.WriteString(fmt.Sprintf("%-*d ", rankColumnWidth, i+1))
		builder.WriteString(fmt.Sprintf("%-*s", nameColumnWidth, playerName))

		for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
			builder.WriteString(fmt.Sprintf("%-*d", resultColumnWidth, results[playerIdxToRankIdx[playerIdx]][rankIdx]))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// Unexported functions

func getRecordFromWinsAndSpread(wins int, spread int) uint64 {
	if wins < 0 || spread < 0 {
		panic("wins and spread must be non-negative")
	}
	return uint64(wins)<<playerWinsOffset | uint64(spread)<<playerSpreadOffset
}

func (standings *Standings) incrementPlayerRecord(playerRank int, incRecord uint64) {
	standings.records[playerRank] += incRecord
}

func (standings *Standings) decrementPlayerRecord(playerRank int, incRecord uint64) {
	standings.records[playerRank] -= incRecord
}

func getIndex(record uint64) int {
	return int(record & playerIndexMask)
}

func getWinsValue(record uint64) int {
	return int((record >> playerWinsOffset) & 0xFFFF)
}

func getSpreadValue(record uint64) uint64 {
	return (record >> playerSpreadOffset) & 0xFFFFFFFF
}

func (standings *Standings) sortRange(i, j int) {
	sort.Slice(standings.records[i:j], func(x, y int) bool {
		return standings.records[i+x] > standings.records[i+y]
	})
}

func (standings *Standings) simFactorPairSegments(results [][]int, playerIdxToRankIdx map[int]int, allSegments [][]int, roundsRemaining int, maxFactor int, sims int) {
	standings.Backup()
	allSegmentPairings := [][][]int{}
	for _, segment := range allSegments {
		i := segment[0]
		j := segment[1]
		pairings := GetPairingsForSegment(i, j, roundsRemaining, maxFactor)
		allSegmentPairings = append(allSegmentPairings, pairings)
	}

	for simIdx := 0; simIdx < sims; simIdx++ {
		for idx, segmentPairings := range allSegmentPairings {
			i := allSegments[idx][0]
			j := allSegments[idx][1]
			standings.SimSingleIteration(segmentPairings, roundsRemaining, i, j)
		}
		standings.UpdateResultsWithFinishedSim(results, playerIdxToRankIdx)
		standings.RestoreFromBackup()
	}
}
