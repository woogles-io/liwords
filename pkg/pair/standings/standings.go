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
// FIXME: recheck if some of these need to be exported

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
	// FIXME: only needed for printing
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
	maxPlayerIdx := int(req.PlacePrizes)
	if maxPlayerIdx > numPlayers {
		maxPlayerIdx = numPlayers
	}
	for playerIdx := 0; playerIdx < maxPlayerIdx; playerIdx++ {
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

func (standings *Standings) SimFactorPairCashers(req *pb.PairRequest, sims int, maxFactor int, numCashers int, computeControlLoss bool) ([][]int, [][]int, map[int]int, int, float64) {
	if numCashers%2 == 1 {
		numCashers++
	}
	if numCashers >= len(standings.records) {
		return standings.SimFactorPairAllPlayers(req, sims, maxFactor, computeControlLoss)
	}
	recordsAll := make([]uint64, len(standings.records))
	recordsBackupAll := make([]uint64, len(standings.recordsBackup))
	copy(recordsAll, standings.records)
	copy(recordsBackupAll, standings.recordsBackup)

	standings.records = standings.records[:numCashers]
	standings.recordsBackup = standings.recordsBackup[:numCashers]

	results, pairings, gibsonGroups, highestControlLossPlayerIdx, controlLoss := standings.SimFactorPairAllPlayers(req, sims, maxFactor, computeControlLoss)

	standings.records = recordsAll
	standings.recordsBackup = recordsBackupAll

	return results, pairings, gibsonGroups, highestControlLossPlayerIdx, controlLoss
}

// Assumes the standings are already sorted
// FIXME: calculate the gibsonizedPlayers in this function instead of passing it in
func (standings *Standings) SimFactorPairAllPlayers(req *pb.PairRequest, sims int, maxFactor int, computeControlLoss bool) ([][]int, [][]int, map[int]int, int, float64) {
	numPlayers := len(standings.records)
	roundsRemaining := int(req.Rounds) - len(req.DivisionPairings)
	evenerPlayerAdded := false
	if numPlayers%2 == 1 {
		// If there are an odd number of players, add a dummy player
		// who will act as the "bye" player. In this implementation,
		// players who play the "bye" player could possibly lose, which
		// is okay because repeat byes are usually avoided and players
		// that low rarely cash anyway.
		// FIXME: test that this works
		lowestWins := getWinsValue(standings.records[numPlayers-1])
		standings.records = append(standings.records, getRecordFromWinsAndSpread(lowestWins-(roundsRemaining+1)*2, initialSpreadValue))
		standings.records[numPlayers] += uint64(req.AllPlayers)
		numPlayers++
		standings.recordsBackup = make([]uint64, numPlayers)
		evenerPlayerAdded = true
	}
	results := make([][]int, numPlayers)
	for i := range results {
		results[i] = make([]int, numPlayers)
	}
	pairings := make([][]int, roundsRemaining)
	for i := 0; i < roundsRemaining; i++ {
		pairings[i] = make([]int, numPlayers)
	}
	gibsonizedPlayers := standings.GetGibsonizedPlayers(req)
	gibsonGroups := map[int]int{}
	nextGibsonGroup := 1
	startRankIdx := 0
	endRankIdx := 0
	leftoverGibsonPlayers := []int{}
	pairingsStartIdx := 0
	for endRankIdx <= numPlayers {
		if endRankIdx == numPlayers {
			assignPairingsForSegment(pairingsStartIdx, startRankIdx, endRankIdx-1, roundsRemaining, maxFactor, leftoverGibsonPlayers, pairings)
			for rankIdx := startRankIdx; rankIdx < endRankIdx; rankIdx++ {
				gibsonGroups[rankIdx] = 0
			}
			break
		}
		if gibsonizedPlayers[endRankIdx] {
			gibsonIsLeftover := true
			if endRankIdx != startRankIdx {
				numPlayersInGibsonGroup := endRankIdx - startRankIdx
				if numPlayersInGibsonGroup%2 == 1 {
					// The number of players in the group is odd, so
					// we have to pull in the gibsonized player at endRankIdx
					// to even the group
					assignPairingsForSegment(pairingsStartIdx, startRankIdx, endRankIdx, roundsRemaining, maxFactor, []int{}, pairings)
					pairingsStartIdx += numPlayersInGibsonGroup + 1
					for rankIdx := startRankIdx; rankIdx <= endRankIdx; rankIdx++ {
						gibsonGroups[rankIdx] = nextGibsonGroup
					}
					nextGibsonGroup++
					gibsonIsLeftover = false
				} else {
					// The number of players in the group is even, so
					// the gibsonized player is not included in the group
					// and will be included in the bottom gibson group
					assignPairingsForSegment(pairingsStartIdx, startRankIdx, endRankIdx-1, roundsRemaining, maxFactor, []int{}, pairings)
					pairingsStartIdx += numPlayersInGibsonGroup
					for rankIdx := startRankIdx; rankIdx < endRankIdx; rankIdx++ {
						gibsonGroups[rankIdx] = nextGibsonGroup
					}
					nextGibsonGroup++
					gibsonGroups[endRankIdx] = 0
				}
			}
			if gibsonIsLeftover {
				leftoverGibsonPlayers = append(leftoverGibsonPlayers, endRankIdx)
			}
			startRankIdx = endRankIdx + 1
		}
		endRankIdx++
	}

	var gibsonSpread int
	if roundsRemaining > len(req.GibsonSpreads) {
		gibsonSpread = int(req.GibsonSpreads[len(req.GibsonSpreads)-1])
	} else {
		gibsonSpread = int(req.GibsonSpreads[roundsRemaining-1])
	}
	// FIXME: find a better way to do this
	maxScoreDiff := maxSpread
	if maxScoreDiff > gibsonSpread {
		maxScoreDiff = gibsonSpread
	}

	playerIdxToRankIdx := standings.getPlayerIdxToRankIdxMap()
	standings.Backup()
	numRecords := len(standings.records)
	highestControlLossPlayerIdx := -1
	controlLoss := -1.0
	if !computeControlLoss {
		for simIdx := 0; simIdx < sims; simIdx++ {
			for roundIdx := 0; roundIdx < roundsRemaining; roundIdx++ {
				for pairIdx := 0; pairIdx < numPlayers; pairIdx += 2 {
					// FIXME: consolidate with simForceWinnerForRound
					randomResult := rand.Intn(2)
					p1 := pairings[roundIdx][pairIdx]
					p2 := pairings[roundIdx][pairIdx+1]
					winner := p1*(1-randomResult) + p2*randomResult
					loser := p2*(1-randomResult) + p1*randomResult
					randomSpread := rand.Intn(maxScoreDiff + 1)
					record := standings.possibleResults[randomSpread]
					standings.incrementPlayerRecord(winner, record)
					standings.decrementPlayerRecord(loser, record)
				}
				standings.Sort()
			}
			// FIXME: only needed for debugging, remove when done
			standings.roundsPlayed += roundsRemaining

			// Update results
			for rankIdx := 0; rankIdx < numRecords; rankIdx++ {
				// The rankIdx is the final rank index that the player achieved
				// for the simulation. We need to get the player index at that
				// rankIdx and find the starting rank for the player, since results
				// are ordered by starting rank index.
				results[playerIdxToRankIdx[getIndex(standings.records[rankIdx])]][rankIdx] += 1
			}

			standings.RestoreFromBackup()
		}
	} else {
		// Perform a binary search to find the player with the lowest
		// number of tournament wins while always winning every game in factor pairings
		// who also always wins every tournament while always play first place and win every game.
		leftPlayerIdx := 0
		rightPlayerIdx := numPlayers - 1
		for leftPlayerIdx <= rightPlayerIdx {
			forcedWinner := (leftPlayerIdx + rightPlayerIdx) / 2
			forcedWinnerIdxes := make([]int, len(pairings))
			for roundPairingsIdx, roundPairings := range pairings {
				for pairingIdx, pIdx := range roundPairings {
					if pIdx == forcedWinner {
						forcedWinnerIdxes[roundPairingsIdx] = pairingIdx
						continue
					}
				}
			}
			vsFirstTournamentWins := standings.simForceWinner(sims, roundsRemaining, pairings, forcedWinner, maxScoreDiff, true)
			if vsFirstTournamentWins < sims {
				rightPlayerIdx = forcedWinner - 1
				continue
			}
			vsFactorPairTournamentWins := standings.simForceWinner(sims, roundsRemaining, pairings, forcedWinner, maxScoreDiff, false)
			playerControlLoss := 1.0 - float64(vsFactorPairTournamentWins)/float64(sims)
			if playerControlLoss >= controlLoss {
				leftPlayerIdx = forcedWinner + 1
				controlLoss = playerControlLoss
				highestControlLossPlayerIdx = forcedWinner
			} else {
				rightPlayerIdx = forcedWinner - 1
			}
		}
	}

	if evenerPlayerAdded {
		// FIXME: test that this works
		standings.records = standings.records[:numPlayers-1]
		standings.recordsBackup = make([]uint64, numPlayers-1)
		for i := 0; i < numPlayers; i++ {
			results[i] = results[i][:numPlayers-1]
		}
		results = results[:numPlayers-1]
	}

	return results, pairings, gibsonGroups, highestControlLossPlayerIdx, controlLoss
}

func (standings *Standings) simForceWinnerForRound(pairings [][]int, roundIdx int, forcedWinner int, maxScoreDiff int) {
	numPlayers := len(standings.records)
	for pairIdx := 0; pairIdx < numPlayers; pairIdx += 2 {
		p1 := pairings[roundIdx][pairIdx]
		p2 := pairings[roundIdx][pairIdx+1]
		var winner int
		var loser int
		if p1 == forcedWinner {
			winner = forcedWinner
			loser = p2
		} else if p2 == forcedWinner {
			winner = forcedWinner
			loser = p1
		} else {
			randomResult := rand.Intn(2)
			winner = p1*(1-randomResult) + p2*randomResult
			loser = p2*(1-randomResult) + p1*randomResult
		}
		randomSpread := rand.Intn(maxScoreDiff + 1)
		record := standings.possibleResults[randomSpread]
		standings.incrementPlayerRecord(winner, record)
		standings.decrementPlayerRecord(loser, record)
	}
	standings.Sort()
}

func (standings *Standings) simForceWinner(sims int, roundsRemaining int, pairings [][]int, forcedWinner int, maxScoreDiff int, vsFirst bool) int {
	var forcedWinnerIdxes []int
	if vsFirst {
		forcedWinnerIdxes = make([]int, len(pairings))
		for roundPairingsIdx, roundPairings := range pairings {
			for pairingIdx, pIdx := range roundPairings {
				if pIdx == forcedWinner {
					forcedWinnerIdxes[roundPairingsIdx] = pairingIdx
					continue
				}
			}
		}
	}
	tournamentWins := 0
	for simIdx := 0; simIdx < sims; simIdx++ {
		for roundIdx := 0; roundIdx < roundsRemaining; roundIdx++ {
			// FIXME: assumes player in first is the first player index in the pairings
			if vsFirst {
				pairings[roundIdx][1], pairings[roundIdx][forcedWinnerIdxes[roundIdx]] = pairings[roundIdx][forcedWinnerIdxes[roundIdx]], pairings[roundIdx][1]
			}
			standings.simForceWinnerForRound(pairings, roundIdx, forcedWinner, maxScoreDiff)
			if vsFirst {
				pairings[roundIdx][1], pairings[roundIdx][forcedWinnerIdxes[roundIdx]] = pairings[roundIdx][forcedWinnerIdxes[roundIdx]], pairings[roundIdx][1]
			}

		}
		// FIXME: only needed for debugging, remove when done
		standings.roundsPlayed += roundsRemaining
		if standings.GetPlayerIndex(0) == forcedWinner {
			tournamentWins++
		}
		standings.RestoreFromBackup()
	}
	return tournamentWins
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

// Unexported functions

// Gets the factor pairings for players in [i, j] for all remaining rounds
// Assumes i < j
// Returns pairings in pairs of player indexes
// For example, pairings of [0, 2, 1, 3] indicate player 0 plays player 2
// and player 1 plays player 3.
func assignPairingsForSegment(pairingsStartIdx int, startRankIdx int, endRankIdx int, totalRoundsRemaining int, maxFactor int, leftoverGibsonPlayers []int, pairings [][]int) {
	numPlayers := endRankIdx - startRankIdx + 1

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
		for factorPairing := 0; factorPairing < roundFactor; factorPairing++ {
			basePairingIdx := pairingsStartIdx + 2*factorPairing
			basePlayerIdx := startRankIdx + factorPairing
			pairings[roundIdx][basePairingIdx] = basePlayerIdx
			pairings[roundIdx][basePairingIdx+1] = basePlayerIdx + roundFactor
		}
		numKothPlayers := numPlayers - 2*roundFactor
		numKothPairings := numKothPlayers / 2
		var nextKothPairingIdx int
		var nextKothPairingPlayerIdx int
		for kothPairing := 0; kothPairing < numKothPairings; kothPairing++ {
			basePairingIdx := pairingsStartIdx + 2*roundFactor + 2*kothPairing
			basePlayerIdx := startRankIdx + 2*roundFactor + 2*kothPairing
			pairings[roundIdx][basePairingIdx] = basePlayerIdx
			pairings[roundIdx][basePairingIdx+1] = basePlayerIdx + 1

			nextKothPairingIdx = basePairingIdx + 2
			nextKothPairingPlayerIdx = basePlayerIdx + 2
		}
		if numKothPlayers%2 == 1 {
			pairings[roundIdx][nextKothPairingIdx] = nextKothPairingPlayerIdx
		}
		if len(leftoverGibsonPlayers) > 0 {
			baseGibsonPairingIdx := pairingsStartIdx + numPlayers
			for playerIdx := 0; playerIdx < len(leftoverGibsonPlayers); playerIdx++ {
				pairings[roundIdx][baseGibsonPairingIdx+playerIdx] = leftoverGibsonPlayers[playerIdx]
			}
		}
	}
}

func (standings *Standings) getPlayerIdxToRankIdxMap() map[int]int {
	playerIdxToRankIdx := map[int]int{}
	for i := 0; i < len(standings.records); i++ {
		playerIdxToRankIdx[standings.GetPlayerIndex(i)] = i
	}
	return playerIdxToRankIdx
}

func (standings *Standings) ResultsString(results [][]int, req *pb.PairRequest) string {
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

	playerIdxToRankIdx := standings.getPlayerIdxToRankIdxMap()
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
