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
	byePlayerIndex     int    = 0xFFFF
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

func (standings *Standings) SimFactorPairSegment(req *pb.PairRequest, sims int, maxFactor int, numCashers int, computeControlLoss bool, logsb *strings.Builder) ([][]int, [][]int, map[int]int, int, int) {
	if numCashers%2 == 1 {
		numCashers++
	}
	if numCashers >= len(standings.records) {
		return standings.SimFactorPairAll(req, sims, maxFactor, computeControlLoss, logsb)
	}
	recordsAll := make([]uint64, len(standings.records))
	recordsBackupAll := make([]uint64, len(standings.recordsBackup))
	copy(recordsAll, standings.records)
	copy(recordsBackupAll, standings.recordsBackup)

	standings.records = standings.records[:numCashers]
	standings.recordsBackup = standings.recordsBackup[:numCashers]

	results, pairings, gibsonGroups, highestControlLossRankIdx, lowestFactorPairWins := standings.SimFactorPairAll(req, sims, maxFactor, computeControlLoss, logsb)

	standings.records = recordsAll
	standings.recordsBackup = recordsBackupAll

	return results, pairings, gibsonGroups, highestControlLossRankIdx, lowestFactorPairWins
}

// Assumes the standings are already sorted
// FIXME: calculate the gibsonizedPlayers in this function instead of passing it in
func (standings *Standings) SimFactorPairAll(req *pb.PairRequest, sims int, maxFactor int, computeControlLoss bool, logsb *strings.Builder) ([][]int, [][]int, map[int]int, int, int) {
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
		standings.records[numPlayers] += uint64(byePlayerIndex)
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
	highestControlLossRankIdx := -1
	lowestFactorPairWins := sims + 1
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
		standings.logStandingsInfo(req, results, nil, logsb)
	} else {
		// Perform a binary search to find the player with the lowest
		// number of tournament wins while always winning every game in factor pairings
		// who also always wins every tournament while always play first place and win every game.\
		allControlLosses := map[int][]int{}
		leftPlayerRankIdx := 1
		rightPlayerRankIdx := numPlayers - 1
		iters := 0
		for leftPlayerRankIdx <= rightPlayerRankIdx {
			iters++
			forcedWinnerRankIdx := (leftPlayerRankIdx + rightPlayerRankIdx) / 2
			forcedWinnerPlayerIdx := standings.GetPlayerIndex(forcedWinnerRankIdx)
			vsFirstTournamentWins := standings.simForceWinner(sims, roundsRemaining, pairings, forcedWinnerPlayerIdx, maxScoreDiff, true)
			if vsFirstTournamentWins < sims {
				rightPlayerRankIdx = forcedWinnerRankIdx - 1
				allControlLosses[forcedWinnerRankIdx] = []int{vsFirstTournamentWins, -1, iters}
				continue
			}
			vsFactorPairTournamentWins := standings.simForceWinner(sims, roundsRemaining, pairings, forcedWinnerPlayerIdx, maxScoreDiff, false)
			allControlLosses[forcedWinnerRankIdx] = []int{vsFirstTournamentWins, vsFactorPairTournamentWins, iters}
			if vsFactorPairTournamentWins < lowestFactorPairWins {
				leftPlayerRankIdx = forcedWinnerRankIdx + 1
				lowestFactorPairWins = vsFactorPairTournamentWins
				highestControlLossRankIdx = forcedWinnerRankIdx
			} else {
				rightPlayerRankIdx = forcedWinnerRankIdx - 1
			}
		}
		standings.logStandingsInfo(req, nil, allControlLosses, logsb)
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

	return results, pairings, gibsonGroups, highestControlLossRankIdx, lowestFactorPairWins
}

func (standings *Standings) simForceWinnerForRound(pairings [][]int, roundIdx int, forcedWinnerRankIdx int, maxScoreDiff int) {
	numPlayers := len(standings.records)
	for pairIdx := 0; pairIdx < numPlayers; pairIdx += 2 {
		p1 := pairings[roundIdx][pairIdx]
		p2 := pairings[roundIdx][pairIdx+1]
		var winner int
		var loser int
		preventTies := 0
		if p1 == forcedWinnerRankIdx {
			winner = forcedWinnerRankIdx
			loser = p2
			preventTies = 1
		} else if p2 == forcedWinnerRankIdx {
			winner = forcedWinnerRankIdx
			loser = p1
			preventTies = 1
		} else {
			randomResult := rand.Intn(2)
			winner = p1*(1-randomResult) + p2*randomResult
			loser = p2*(1-randomResult) + p1*randomResult
		}
		randomSpread := preventTies + rand.Intn(maxScoreDiff+1-preventTies)
		record := standings.possibleResults[randomSpread]
		standings.incrementPlayerRecord(winner, record)
		standings.decrementPlayerRecord(loser, record)
	}
	standings.Sort()
}

func (standings *Standings) findRankIdx(playerIdx int) int {
	numPlayers := len(standings.records)
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		if standings.GetPlayerIndex(rankIdx) == playerIdx {
			return rankIdx
		}
	}
	return -1
}

// FIXME: remove req
func (standings *Standings) simForceWinner(sims int, roundsRemaining int, pairings [][]int, forcedWinnerPlayerIdx int, maxScoreDiff int, vsFirst bool) int {
	tournamentWins := 0
	for simIdx := 0; simIdx < sims; simIdx++ {
		for roundIdx := 0; roundIdx < roundsRemaining; roundIdx++ {
			// FIXME: only needed for debugging, remove when done
			standings.roundsPlayed++
			forcedWinnerRankIdx := standings.findRankIdx(forcedWinnerPlayerIdx)
			var switchPairingIdx int
			if vsFirst {
				for idx, rankIdx := range pairings[roundIdx] {
					if rankIdx == forcedWinnerRankIdx {
						switchPairingIdx = idx
						break
					}
				}
			}
			if vsFirst {
				pairings[roundIdx][1], pairings[roundIdx][switchPairingIdx] = pairings[roundIdx][switchPairingIdx], pairings[roundIdx][1]
			}
			standings.simForceWinnerForRound(pairings, roundIdx, forcedWinnerRankIdx, maxScoreDiff)
			if vsFirst {
				pairings[roundIdx][1], pairings[roundIdx][switchPairingIdx] = pairings[roundIdx][switchPairingIdx], pairings[roundIdx][1]
			}
		}
		if standings.GetPlayerIndex(0) == forcedWinnerPlayerIdx {
			tournamentWins++
		}
		standings.RestoreFromBackup()
	}
	return tournamentWins
}

func (standings *Standings) LogStandings(req *pb.PairRequest, logsb *strings.Builder) {
	standings.logStandingsInfo(req, nil, nil, logsb)
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

func (standings *Standings) String(req *pb.PairRequest, results [][]int, allControlLosses map[int][]int) string {
	var playerNames []string

	playerNameColWidth := 0
	if req != nil {
		for _, playerName := range req.PlayerNames {
			if len(playerName) > playerNameColWidth {
				if len(playerName) > 30 {
					playerNameColWidth = 30
				} else {
					playerNameColWidth = len(playerName)
				}
			}
		}
		playerNames = req.PlayerNames
	} else {
		playerNames = make([]string, len(standings.records))
		for i := 0; i < len(standings.records); i++ {
			playerNames[i] = strconv.Itoa(i + 1)
		}
		playerNameColWidth = len(playerNames[len(standings.records)-1])
	}

	headerFormat := fmt.Sprintf("%%-4s | %%-6s | %%-%ds | %%-4s | %%-6s | %%s\n", playerNameColWidth)
	rowFormat := fmt.Sprintf("%%-4d | %%-6d | %%-%ds | %%-4.1f | %%-6d |", playerNameColWidth)

	customHeader := ""
	var playerIdxToRankIdx map[int]int
	if results != nil {
		customHeader = "Results"
		playerIdxToRankIdx = standings.getPlayerIdxToRankIdxMap()
	} else if allControlLosses != nil {
		customHeader = "Control Losses"
	}

	numPlayers := len(standings.records)
	if standings.GetPlayerIndex(numPlayers-1) == byePlayerIndex {
		numPlayers--
	}
	header := fmt.Sprintf(headerFormat, "Rank", "Number", "Name", "Wins", "Spread", customHeader)
	res := header
	res += strings.Repeat("-", len(header)) + "\n"
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		playerIdx := standings.GetPlayerIndex(rankIdx)
		wins := standings.GetPlayerWins(rankIdx)
		spread := standings.GetPlayerSpread(rankIdx)
		playerName := playerNames[playerIdx]
		if len(playerName) > playerNameColWidth {
			playerName = playerName[:playerNameColWidth]
		}
		res += fmt.Sprintf(rowFormat, rankIdx+1, playerIdx+1, playerName, wins, spread)
		if results != nil {
			for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
				res += fmt.Sprintf("%8d", results[playerIdxToRankIdx[playerIdx]][rankIdx])
			}
		} else if allControlLosses != nil {
			controlLossInfo, exists := allControlLosses[rankIdx]
			if exists {
				vsFirstWins := controlLossInfo[0]
				vsFactorPairWins := controlLossInfo[1]
				iter := controlLossInfo[2]
				if vsFactorPairWins >= 0 {
					res += fmt.Sprintf("%8d %-8d %d", vsFirstWins, vsFactorPairWins, iter)
				} else {
					res += fmt.Sprintf("%8d %-8s %d", vsFirstWins, "-", iter)
				}
			}
		}
		res += "\n"
	}
	res += "\n"
	return res
}

func (standings *Standings) logStandingsInfo(req *pb.PairRequest, results [][]int, allControlLosses map[int][]int, logsb *strings.Builder) {
	logsb.WriteString(standings.String(req, results, allControlLosses))
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
