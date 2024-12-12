package standings

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"golang.org/x/exp/rand"
)

const (
	ByePlayerIndex int = 0xFFFF
)

const (
	playerWinsOffset   int    = 48
	initialWinsValue   int    = 1 << (64 - playerWinsOffset - 1)
	playerSpreadOffset int    = 16
	initialSpreadValue int    = 1 << (playerWinsOffset - playerSpreadOffset - 1)
	playerIndexMask    uint64 = 0xFFFF
	byeSpread          int    = 50
)

type Standings struct {
	records            []uint64
	recordsBackup      []uint64
	possibleResults    []uint64
	tieResults         int
	roundsPlayed       int
	roundsPlayedBackup int
}

type SimResults struct {
	FinalRanks                [][]int
	Pairings                  [][]int
	GibsonGroups              []int
	GibsonizedPlayers         []bool
	HighestControlLossRankIdx int
	LowestFactorPairWins      int
	AllControlLosses          map[int]int
	SegmentRoundFactors       []int
}

func CreateInitialStandings(req *pb.PairRequest) *Standings {
	// Create empty standings
	standings := &Standings{}
	standings.roundsPlayed = len(req.DivisionResults)
	standings.records = make([]uint64, int(req.AllPlayers))
	for playerIdx := 0; playerIdx < int(req.AllPlayers); playerIdx++ {
		standings.records[playerIdx] = getRecordFromWinsAndSpread(initialWinsValue, initialSpreadValue)
		standings.records[playerIdx] += uint64(playerIdx)
	}

	// Update the possible results with the maximum gibson spread
	scoreDiffs := GetScoreDifferences()
	numPossibleResults := len(scoreDiffs)
	standings.possibleResults = make([]uint64, numPossibleResults)
	tieResults := 0
	for i := 0; i < numPossibleResults; i++ {
		spread := scoreDiffs[i]
		if spread == 0 {
			tieResults++
			continue
		}
		if spread > uint64(req.GibsonSpread) {
			spread = uint64(req.GibsonSpread)
		}
		standings.possibleResults[i] = getRecordFromWinsAndSpread(1, int(spread))
	}
	standings.tieResults = tieResults

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
		tieResults:         standings.tieResults,
		roundsPlayed:       standings.roundsPlayed,
		roundsPlayedBackup: standings.roundsPlayedBackup,
	}
	copy(standingsCopy.records, standings.records)
	copy(standingsCopy.recordsBackup, standings.recordsBackup)
	return standingsCopy
}

func (standings *Standings) Backup() {
	copy(standings.recordsBackup, standings.records)
}

func (standings *Standings) RestoreFromBackup() {
	copy(standings.records, standings.recordsBackup)
}

func (standings *Standings) GetNumPlayers() int {
	return len(standings.records)
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
	cumeGibsonSpread := getCumeGibsonSpread(req)
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

// Assumes the standings are already sorted
func (standings *Standings) SimFactorPairAll(ctx context.Context, req *pb.PairRequest, copRand *rand.Rand, sims int, maxFactor int, computeControlLoss bool, prevSegmentRoundFactors []int) (*SimResults, pb.PairError) {
	numPlayers := standings.GetNumPlayers()
	roundsRemaining := int(req.Rounds) - len(req.DivisionPairings)
	evenerPlayerAdded := false
	if numPlayers%2 == 1 {
		// If there are an odd number of players, add a dummy player
		// who will act as the "bye" player. In this implementation,
		// players who play the "bye" player could possibly lose, which
		// is okay because repeat byes are usually avoided and players
		// that low rarely cash anyway.
		lowestWins := getWinsValue(standings.records[numPlayers-1])
		standings.records = append(standings.records, getRecordFromWinsAndSpread(lowestWins-(roundsRemaining+1)*2, initialSpreadValue))
		standings.records[numPlayers] += uint64(ByePlayerIndex)
		numPlayers++
		standings.recordsBackup = make([]uint64, numPlayers)
		evenerPlayerAdded = true
	}

	simResults, pairErr := standings.evenedSimFactorPairAll(ctx, req, copRand, sims, maxFactor, computeControlLoss, prevSegmentRoundFactors, roundsRemaining)
	if pairErr != pb.PairError_SUCCESS {
		return nil, pairErr
	}
	if evenerPlayerAdded {
		standings.records = standings.records[:numPlayers-1]
		standings.recordsBackup = make([]uint64, numPlayers-1)
		if simResults != nil {
			for i := 0; i < numPlayers; i++ {
				simResults.FinalRanks[i] = simResults.FinalRanks[i][:numPlayers-1]
			}
			simResults.FinalRanks = simResults.FinalRanks[:numPlayers-1]
		}
	}
	return simResults, pb.PairError_SUCCESS
}

func (standings *Standings) evenedSimFactorPairAll(ctx context.Context, req *pb.PairRequest, copRand *rand.Rand, sims int, maxFactor int, computeControlLoss bool, prevSegmentRoundFactors []int, roundsRemaining int) (*SimResults, pb.PairError) {
	numPlayers := standings.GetNumPlayers()
	results := make([][]int, numPlayers)
	for i := range results {
		results[i] = make([]int, numPlayers)
	}
	pairings := make([][]int, roundsRemaining)
	for i := 0; i < roundsRemaining; i++ {
		pairings[i] = make([]int, numPlayers)
	}
	gibsonizedPlayers := standings.GetGibsonizedPlayers(req)
	gibsonGroups := make([]int, numPlayers)
	nextGibsonGroup := 1
	startRankIdx := 0
	endRankIdx := 0
	leftoverGibsonPlayers := []int{}
	pairingsStartIdx := 0
	segmentRoundFactors := []int{}
	for endRankIdx <= numPlayers {
		if endRankIdx == numPlayers {
			assignPairingsForSegment(pairingsStartIdx, startRankIdx, endRankIdx-1, roundsRemaining, maxFactor, leftoverGibsonPlayers, pairings, &segmentRoundFactors)
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
					assignPairingsForSegment(pairingsStartIdx, startRankIdx, endRankIdx, roundsRemaining, maxFactor, []int{}, pairings, &segmentRoundFactors)
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
					assignPairingsForSegment(pairingsStartIdx, startRankIdx, endRankIdx-1, roundsRemaining, maxFactor, []int{}, pairings, &segmentRoundFactors)
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

	// If the previous simulation ran with the same parameters, there is no need to rerun
	if prevSegmentRoundFactors != nil && areIntArraysEqual(segmentRoundFactors, prevSegmentRoundFactors) {
		return nil, pb.PairError_SUCCESS
	}

	playerIdxToRankIdx := standings.getPlayerIdxToRankIdxMap()
	standings.Backup()
	numRecords := len(standings.records)
	highestControlLossRankIdx := -1
	lowestFactorPairWins := sims + 1
	var allControlLosses map[int]int
	if !computeControlLoss {
		for simIdx := 0; simIdx < sims; simIdx++ {
			for roundIdx := 0; roundIdx < roundsRemaining; roundIdx++ {
				pairErr := standings.simRound(ctx, copRand, pairings, roundIdx, -1)
				if pairErr != pb.PairError_SUCCESS {
					return nil, pairErr
				}
			}
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
		// who also always wins every tournament while always play first place and win every game.\
		allControlLosses = map[int]int{}
		leftPlayerRankIdx := 1
		rightPlayerRankIdx := 1
		roundsRemaining := int(req.Rounds) - len(req.DivisionResults)
		cumeGibsonSpread := getCumeGibsonSpread(req)
		for rightPlayerRankIdx < numPlayers {
			if !standings.CanCatch(roundsRemaining, cumeGibsonSpread, 0, rightPlayerRankIdx) {
				break
			}
			rightPlayerRankIdx++
		}
		rightPlayerRankIdx--
		for leftPlayerRankIdx <= rightPlayerRankIdx {
			forcedWinnerRankIdx := (leftPlayerRankIdx + rightPlayerRankIdx) / 2
			forcedWinnerPlayerIdx := standings.GetPlayerIndex(forcedWinnerRankIdx)
			vsFirstTournamentWins, pairErr := standings.simForceWinner(ctx, copRand, sims, roundsRemaining, pairings, forcedWinnerPlayerIdx, true)
			if pairErr != pb.PairError_SUCCESS {
				return nil, pairErr
			}
			if vsFirstTournamentWins < sims {
				rightPlayerRankIdx = forcedWinnerRankIdx - 1
				allControlLosses[forcedWinnerRankIdx] = -1
				continue
			}
			vsFactorPairTournamentWins, pairErr := standings.simForceWinner(ctx, copRand, sims, roundsRemaining, pairings, forcedWinnerPlayerIdx, false)
			if pairErr != pb.PairError_SUCCESS {
				return nil, pairErr
			}
			allControlLosses[forcedWinnerRankIdx] = vsFactorPairTournamentWins
			if vsFactorPairTournamentWins < lowestFactorPairWins {
				leftPlayerRankIdx = forcedWinnerRankIdx + 1
				lowestFactorPairWins = vsFactorPairTournamentWins
				highestControlLossRankIdx = forcedWinnerRankIdx
			} else {
				rightPlayerRankIdx = forcedWinnerRankIdx - 1
			}
		}
	}
	return &SimResults{
		FinalRanks:                results,
		Pairings:                  pairings,
		GibsonGroups:              gibsonGroups,
		GibsonizedPlayers:         gibsonizedPlayers,
		HighestControlLossRankIdx: highestControlLossRankIdx,
		LowestFactorPairWins:      lowestFactorPairWins,
		AllControlLosses:          allControlLosses,
		SegmentRoundFactors:       segmentRoundFactors,
	}, pb.PairError_SUCCESS
}

func getCumeGibsonSpread(req *pb.PairRequest) int {
	roundsRemaining := int(req.Rounds) - len(req.DivisionResults)
	return int(req.GibsonSpread) * roundsRemaining * 2
}

func (standings *Standings) simRound(ctx context.Context, copRand *rand.Rand, pairings [][]int, roundIdx int, forcedWinnerRankIdx int) pb.PairError {
	if ctx.Err() != nil {
		// Assume ctx errors are always timeouts
		return pb.PairError_TIMEOUT
	}
	numPlayers := len(standings.records)
	numScoreDiffs := len(standings.possibleResults)
	if forcedWinnerRankIdx < 0 {
		for pairIdx := 0; pairIdx < numPlayers; pairIdx += 2 {
			randomResult := copRand.Intn(2)
			p1 := pairings[roundIdx][pairIdx]
			p2 := pairings[roundIdx][pairIdx+1]
			winner := p1*(1-randomResult) + p2*randomResult
			loser := p2*(1-randomResult) + p1*randomResult
			record := standings.possibleResults[copRand.Intn(numScoreDiffs)]
			standings.incrementPlayerRecord(winner, record)
			standings.decrementPlayerRecord(loser, record)
		}
	} else {
		for pairIdx := 0; pairIdx < numPlayers; pairIdx += 2 {
			p1 := pairings[roundIdx][pairIdx]
			p2 := pairings[roundIdx][pairIdx+1]
			var winner int
			var loser int
			var randResultIdx int
			if p1 == forcedWinnerRankIdx {
				winner = forcedWinnerRankIdx
				loser = p2
				// Prevent tie scores for the forced winner
				randResultIdx = copRand.Intn(numScoreDiffs-standings.tieResults) + standings.tieResults
			} else if p2 == forcedWinnerRankIdx {
				winner = forcedWinnerRankIdx
				loser = p1
				// Prevent tie scores for the forced winner
				randResultIdx = copRand.Intn(numScoreDiffs-standings.tieResults) + standings.tieResults
			} else {
				randomResult := copRand.Intn(2)
				winner = p1*(1-randomResult) + p2*randomResult
				loser = p2*(1-randomResult) + p1*randomResult
				randResultIdx = copRand.Intn(numScoreDiffs)
			}
			record := standings.possibleResults[randResultIdx]
			standings.incrementPlayerRecord(winner, record)
			standings.decrementPlayerRecord(loser, record)
		}
	}
	standings.Sort()
	return pb.PairError_SUCCESS
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

func (standings *Standings) simForceWinner(ctx context.Context, copRand *rand.Rand, sims int, roundsRemaining int, pairings [][]int, forcedWinnerPlayerIdx int, vsFirst bool) (int, pb.PairError) {
	tournamentWins := 0
	for simIdx := 0; simIdx < sims; simIdx++ {
		for roundIdx := 0; roundIdx < roundsRemaining; roundIdx++ {
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
			pairErr := standings.simRound(ctx, copRand, pairings, roundIdx, forcedWinnerRankIdx)
			if pairErr != pb.PairError_SUCCESS {
				return 0, pairErr
			}
			if vsFirst {
				pairings[roundIdx][1], pairings[roundIdx][switchPairingIdx] = pairings[roundIdx][switchPairingIdx], pairings[roundIdx][1]
			}
		}
		playerInFirst := standings.GetPlayerIndex(0)
		if playerInFirst == forcedWinnerPlayerIdx {
			tournamentWins++
		}
		standings.RestoreFromBackup()
		if vsFirst && playerInFirst != forcedWinnerPlayerIdx {
			return 0, pb.PairError_SUCCESS
		}
	}
	return tournamentWins, pb.PairError_SUCCESS
}

// Unexported functions

// Gets the factor pairings for players in [i, j] for all remaining rounds
// Assumes i < j
// Returns pairings in pairs of player indexes
// For example, pairings of [0, 2, 1, 3] indicate player 0 plays player 2
// and player 1 plays player 3.
func assignPairingsForSegment(pairingsStartIdx int, startRankIdx int, endRankIdx int, totalRoundsRemaining int, maxFactor int, leftoverGibsonPlayers []int, pairings [][]int, segmentRoundFactors *[]int) {
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
		*segmentRoundFactors = append(*segmentRoundFactors, roundFactor)
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

func (standings *Standings) StringDataForPlayer(req *pb.PairRequest, rankIdx int) []string {
	playerData := make([]string, 5)
	playerData[0] = strconv.Itoa(rankIdx + 1)
	playerData[1] = strconv.Itoa(standings.GetPlayerIndex(rankIdx) + 1)
	playerData[2] = ""
	if req != nil {
		playerData[2] = req.PlayerNames[standings.GetPlayerIndex(rankIdx)]
	}
	playerData[3] = fmt.Sprintf("%.1f", standings.GetPlayerWins(rankIdx))
	playerData[4] = strconv.Itoa(standings.GetPlayerSpread(rankIdx))
	return playerData
}

func (standings *Standings) StringData(req *pb.PairRequest) [][]string {
	numPlayers := len(standings.records)
	if standings.GetPlayerIndex(numPlayers-1) == ByePlayerIndex {
		numPlayers--
	}
	stringData := [][]string{}
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		stringData = append(stringData, standings.StringDataForPlayer(req, rankIdx))
	}
	return stringData
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

func areIntArraysEqual(arr1, arr2 []int) bool {
	if len(arr1) != len(arr2) {
		return false
	}
	for i := range arr1 {
		if arr1[i] != arr2[i] {
			return false
		}
	}
	return true
}
