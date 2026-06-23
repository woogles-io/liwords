package standings

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
)

const (
	ByePlayerIndex int = 0xFFFF
)

const (
	playerWinsOffset                     int     = 48
	initialWinsValue                     int     = 1 << (64 - playerWinsOffset - 1)
	playerSpreadOffset                   int     = 16
	initialSpreadValue                   int     = 1 << (playerWinsOffset - playerSpreadOffset - 1)
	playerIndexMask                      uint64  = 0xFFFF
	simConfidence                        float64 = 0.99
	resimBatchSize                       int     = 10000
	initialSimTimeLimit                  int     = 6
	reSimTimeLimit                       int     = 6
	controlSimTimeLimit                  int     = 6
	controlLossWinAllVsFirstRoundsThresh int     = 4
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
	AllControlLosses          map[int]int
	VsFirstWins               map[int]int
	SegmentRoundFactors       []int
	TotalSims                 int
}

func GetRoundsRemaining(req *pb.PairRequest) int {
	return int(req.Rounds) - len(req.DivisionResults)
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
	return float64(standings.GetPlayerWinsIntTimesTwo(rankIdx)) / 2
}

// GetPlayerWinsIntTimesTwo returns wins*2 + ties as an integer, avoiding
// a float64 round-trip when callers need an integer win comparison.
func (standings *Standings) GetPlayerWinsIntTimesTwo(rankIdx int) int {
	return int(getWinsValue(standings.records[rankIdx]) - initialWinsValue + standings.roundsPlayed)
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
	roundsRemaining := GetRoundsRemaining(req)
	cumeGibsonSpread := getCumeGibsonSpread(req)
	maxPlayerRankIdx := int(req.PlacePrizes)
	for playerRankIdx := 0; playerRankIdx < maxPlayerRankIdx; playerRankIdx++ {
		gibsonizedPlayers[playerRankIdx] = true
		if playerRankIdx > 0 && standings.CanCatch(roundsRemaining, cumeGibsonSpread, playerRankIdx-1, playerRankIdx) {
			gibsonizedPlayers[playerRankIdx] = false
			continue
		}
		if playerRankIdx < numPlayers-1 && standings.CanCatch(roundsRemaining, cumeGibsonSpread, playerRankIdx, playerRankIdx+1) {
			gibsonizedPlayers[playerRankIdx] = false
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
func (standings *Standings) SimFactorPairAll(req *pb.PairRequest, copRand *rand.Rand, sims int, maxFactor int, lowestHopeControlLosser int, prevSegmentRoundFactors []int) (*SimResults, pb.PairError) {
	numPlayers := standings.GetNumPlayers()
	roundsRemaining := GetRoundsRemaining(req)
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

	simResults, pairErr := standings.evenedSimFactorPairAll(req, copRand, sims, maxFactor, lowestHopeControlLosser, prevSegmentRoundFactors)
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

func (standings *Standings) evenedSimFactorPairAll(req *pb.PairRequest, copRand *rand.Rand, sims int, maxFactor int, lowestHopeControlLosser int, prevSegmentRoundFactors []int) (*SimResults, pb.PairError) {
	numPlayers := standings.GetNumPlayers()
	results := make([][]int, numPlayers)
	for i := range results {
		results[i] = make([]int, numPlayers)
	}
	roundsRemaining := GetRoundsRemaining(req)
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
	controlLoss := lowestHopeControlLosser >= 0
	pairingsStartIdx := 0
	segmentRoundFactors := []int{}
	for endRankIdx <= numPlayers {
		if endRankIdx == numPlayers {
			assignPairingsForSegment(pairingsStartIdx, startRankIdx, endRankIdx-1, roundsRemaining, maxFactor, leftoverGibsonPlayers, pairings, &segmentRoundFactors, copRand, numPlayers, controlLoss)
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
					assignPairingsForSegment(pairingsStartIdx, startRankIdx, endRankIdx, roundsRemaining, maxFactor, []int{}, pairings, &segmentRoundFactors, copRand, numPlayers, controlLoss)
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
					assignPairingsForSegment(pairingsStartIdx, startRankIdx, endRankIdx-1, roundsRemaining, maxFactor, []int{}, pairings, &segmentRoundFactors, copRand, numPlayers, controlLoss)
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
	highestControlLossRankIdx := -1
	var allControlLosses map[int]int
	var vsFirstWins map[int]int
	totalSims := 0
	if lowestHopeControlLosser < 0 {
		initialSimStopTime := time.Now().UnixNano() + int64(initialSimTimeLimit)*1e9
		count, tl := standings.runParallelSims(sims, copRand, roundsRemaining, pairings, results, playerIdxToRankIdx, initialSimStopTime)
		totalSims += count
		if !tl {
			ranksToCheck := []int{int(req.PlacePrizes) - 1}
			if !gibsonizedPlayers[0] {
				ranksToCheck = append(ranksToCheck, 0)
			}
			reSimStopTime := time.Now().UnixNano() + int64(reSimTimeLimit)*1e9
			for !simResultsReachedThreshold(results, ranksToCheck, totalSims, req.HopefulnessThreshold) {
				batchCount, batchTL := standings.runParallelSims(resimBatchSize, copRand, roundsRemaining, pairings, results, playerIdxToRankIdx, reSimStopTime)
				totalSims += batchCount
				if batchTL {
					break
				}
			}
		}
	} else {
		// Perform a binary search to find the player with the lowest
		// number of tournament wins while always winning every game in factor pairings
		// who also always wins every tournament while always play first place and win every game.\
		allControlLosses = map[int]int{}
		vsFirstWins = map[int]int{}
		leftPlayerRankIdx := 1
		rightPlayerRankIdx := lowestHopeControlLosser
		controlSimStopTime := time.Now().UnixNano() + int64(controlSimTimeLimit)*1e9
		for leftPlayerRankIdx <= rightPlayerRankIdx {
			forcedWinnerRankIdx := (leftPlayerRankIdx + rightPlayerRankIdx) / 2
			vsFirstTournamentWins, vsFactorPairTournamentWins, newSims, pairErr := standings.evaluatePlayerControlLoss(
				copRand, sims, roundsRemaining, maxFactor, pairings, forcedWinnerRankIdx, controlSimStopTime,
				vsFirstWins, allControlLosses,
			)
			if pairErr != pb.PairError_SUCCESS {
				return nil, pairErr
			}
			totalSims += newSims
			// The binary search conditions are different depending on the number of rounds remaining.
			if roundsRemaining > controlLossWinAllVsFirstRoundsThresh {
				if vsFirstTournamentWins < sims {
					rightPlayerRankIdx = forcedWinnerRankIdx - 1
				} else if float64(vsFirstTournamentWins-vsFactorPairTournamentWins) >= req.ControlLossThreshold*float64(sims) {
					rightPlayerRankIdx = forcedWinnerRankIdx - 1
					if highestControlLossRankIdx == -1 || forcedWinnerRankIdx < highestControlLossRankIdx {
						highestControlLossRankIdx = forcedWinnerRankIdx
					}
				} else {
					leftPlayerRankIdx = forcedWinnerRankIdx + 1
				}
			} else {
				if float64(vsFirstTournamentWins-vsFactorPairTournamentWins) >= req.ControlLossThreshold*float64(sims) {
					rightPlayerRankIdx = forcedWinnerRankIdx - 1
					if highestControlLossRankIdx == -1 || forcedWinnerRankIdx < highestControlLossRankIdx {
						highestControlLossRankIdx = forcedWinnerRankIdx
					}
				} else {
					leftPlayerRankIdx = forcedWinnerRankIdx + 1
				}
			}

		}
		if highestControlLossRankIdx == -1 && roundsRemaining == 2 {
			lowestFullWinRankIdx := -1
			for rankIdx := 1; rankIdx <= lowestHopeControlLosser; rankIdx++ {
				vsFirst, vsFactorPair, newSims, pairErr := standings.evaluatePlayerControlLoss(
					copRand, sims, roundsRemaining, maxFactor, pairings, rankIdx, controlSimStopTime,
					vsFirstWins, allControlLosses,
				)
				if pairErr != pb.PairError_SUCCESS {
					return nil, pairErr
				}
				totalSims += newSims
				if vsFirst == sims && vsFactorPair == sims {
					lowestFullWinRankIdx = rankIdx
				} else if lowestFullWinRankIdx != -1 {
					break
				}
			}
			if lowestFullWinRankIdx != -1 && lowestFullWinRankIdx+1 <= lowestHopeControlLosser {
				bRankIdx := lowestFullWinRankIdx + 1
				if vsFirstWins[bRankIdx] > allControlLosses[bRankIdx] {
					highestControlLossRankIdx = bRankIdx
				}
			}
		}
	}
	return &SimResults{
		FinalRanks:                results,
		Pairings:                  pairings,
		GibsonGroups:              gibsonGroups,
		GibsonizedPlayers:         gibsonizedPlayers,
		HighestControlLossRankIdx: highestControlLossRankIdx,
		AllControlLosses:          allControlLosses,
		VsFirstWins:               vsFirstWins,
		SegmentRoundFactors:       segmentRoundFactors,
		TotalSims:                 totalSims,
	}, pb.PairError_SUCCESS
}

func (standings *Standings) evaluatePlayerControlLoss(
	copRand *rand.Rand, sims, roundsRemaining, maxFactor int, pairings [][]int,
	rankIdx int, stopTime int64,
	vsFirstWins, allControlLosses map[int]int,
) (int, int, int, pb.PairError) {
	if vsFirst, ok := vsFirstWins[rankIdx]; ok {
		return vsFirst, allControlLosses[rankIdx], 0, pb.PairError_SUCCESS
	}
	playerIdx := standings.GetPlayerIndex(rankIdx)
	vsFirst, pairErr := standings.runParallelSimForceWinner(copRand, sims, roundsRemaining, maxFactor, pairings, playerIdx, true, stopTime)
	if pairErr != pb.PairError_SUCCESS {
		return 0, 0, 0, pairErr
	}
	vsFirstWins[rankIdx] = vsFirst
	vsFactorPair, pairErr := standings.runParallelSimForceWinner(copRand, sims, roundsRemaining, maxFactor, pairings, playerIdx, false, stopTime)
	if pairErr != pb.PairError_SUCCESS {
		return 0, 0, 0, pairErr
	}
	allControlLosses[rankIdx] = vsFactorPair
	return vsFirst, vsFactorPair, 2, pb.PairError_SUCCESS
}

// Returns true if the time limit has been exceeded
func (standings *Standings) simToEndAndRecordResults(roundsRemaining int, copRand *rand.Rand, pairings [][]int, results [][]int, playerIdxToRankIdx map[int]int, stopTimeNano int64) bool {
	for roundIdx := 0; roundIdx < roundsRemaining; roundIdx++ {
		timeLimitExceeded := standings.simRound(copRand, pairings, roundIdx, -1, stopTimeNano)
		if timeLimitExceeded {
			standings.RestoreFromBackup()
			return true
		}
	}
	// Update results
	numRecords := len(standings.records)
	for rankIdx := 0; rankIdx < numRecords; rankIdx++ {
		// The rankIdx is the final rank index that the player achieved
		// for the simulation. We need to get the player index at that
		// rankIdx and find the starting rank for the player, since results
		// are ordered by starting rank index.
		results[playerIdxToRankIdx[getIndex(standings.records[rankIdx])]][rankIdx] += 1
	}

	standings.RestoreFromBackup()
	return false
}

func getCumeGibsonSpread(req *pb.PairRequest) int {
	return int(req.GibsonSpread) * GetRoundsRemaining(req) * 2
}

// Returns true if the time limit has been exceeded
func (standings *Standings) simRound(copRand *rand.Rand, pairings [][]int, roundIdx int, forcedWinnerRankIdx int, stopTimeNano int64) bool {
	currentTimeNano := time.Now().UnixNano()
	if currentTimeNano > stopTimeNano {
		return true
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
	return false
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

func (standings *Standings) simForceWinner(copRand *rand.Rand, sims int, roundsRemaining int, maxFactor int, pairings [][]int, forcedWinnerPlayerIdx int, vsFirst bool, stopTimeNano int64) (int, pb.PairError) {
	tournamentWins := 0
	for simIdx := 0; simIdx < sims; simIdx++ {
		for roundIdx := 0; roundIdx < roundsRemaining; roundIdx++ {
			forcedWinnerRankIdx := standings.findRankIdx(forcedWinnerPlayerIdx)

			// Find the forced winner's current position in this round's pairings.
			switchPairingIdx := -1
			for idx, rankIdx := range pairings[roundIdx] {
				if rankIdx == forcedWinnerRankIdx {
					switchPairingIdx = idx
					break
				}
			}

			// Determine which swap (if any) to apply before simulating the round.
			// swapIdxA and swapIdxB are the two positions to exchange.
			swapIdxA, swapIdxB := -1, -1
			if vsFirst && roundIdx < roundsRemaining-1 {
				// Move forced winner to position 1 so they play rank 0 (1st place).
				// Skip the final round: it is always KOTH and resolves naturally.
				swapIdxA = 1
				swapIdxB = switchPairingIdx
			} else if switchPairingIdx == 1 && roundIdx < roundsRemaining-1 {
				// Forced winner is currently rank 0's factor-pair opponent.
				// Instead of having them play 1st, swap with another pairing:
				//   - If forced winner is 2nd (rank 1): do 1st vs 3rd, 2nd vs 3rd's opponent.
				//   - Otherwise: do 1st vs 2nd, forced winner vs 2nd's opponent.
				swapIdxA = 1
				targetRank := 1
				if forcedWinnerRankIdx == 1 {
					targetRank = 2
				}
				for idx, rankIdx := range pairings[roundIdx] {
					if rankIdx == targetRank {
						swapIdxB = idx
						break
					}
				}
			}

			if swapIdxA >= 0 && swapIdxB >= 0 {
				pairings[roundIdx][swapIdxA], pairings[roundIdx][swapIdxB] = pairings[roundIdx][swapIdxB], pairings[roundIdx][swapIdxA]
			}

			timeLimitExceeded := standings.simRound(copRand, pairings, roundIdx, forcedWinnerRankIdx, stopTimeNano)
			if timeLimitExceeded {
				return 0, pb.PairError_TIMEOUT
			}

			if swapIdxA >= 0 && swapIdxB >= 0 {
				pairings[roundIdx][swapIdxA], pairings[roundIdx][swapIdxB] = pairings[roundIdx][swapIdxB], pairings[roundIdx][swapIdxA]
			}
		}
		playerInFirst := standings.GetPlayerIndex(0)
		if playerInFirst == forcedWinnerPlayerIdx {
			tournamentWins++
		}
		standings.RestoreFromBackup()
	}
	return tournamentWins, pb.PairError_SUCCESS
}

// runParallelSims distributes numSims iterations of simToEndAndRecordResults across
// runtime.NumCPU() goroutines, each operating on an independent copy of standings.
// Returns (totalSimsCompleted, timeLimitExceeded).
func (standings *Standings) runParallelSims(
	numSims int, copRand *rand.Rand, roundsRemaining int,
	pairings [][]int, results [][]int, playerIdxToRankIdx map[int]int,
	stopTimeNano int64,
) (int, bool) {
	numWorkers := runtime.NumCPU()
	if numWorkers > numSims {
		numWorkers = numSims
	}
	numPlayers := len(standings.records)
	simsPerWorker := numSims / numWorkers
	remainder := numSims % numWorkers

	var wg sync.WaitGroup
	var mu sync.Mutex
	totalSims := 0
	timeLimitExceeded := false

	for w := 0; w < numWorkers; w++ {
		workerSims := simsPerWorker
		if w < remainder {
			workerSims++
		}
		wg.Add(1)
		localStandings := standings.Copy()
		workerRand := rand.New(rand.NewSource(uint64(copRand.Int63())))
		localResults := make([][]int, numPlayers)
		for i := range localResults {
			localResults[i] = make([]int, numPlayers)
		}
		go func(sc *Standings, wr *rand.Rand, lr [][]int, ns int) {
			defer wg.Done()
			localCount := 0
			localTL := false
			for i := 0; i < ns; i++ {
				if sc.simToEndAndRecordResults(roundsRemaining, wr, pairings, lr, playerIdxToRankIdx, stopTimeNano) {
					localTL = true
					break
				}
				localCount++
			}
			mu.Lock()
			totalSims += localCount
			if localTL {
				timeLimitExceeded = true
			}
			for pi := range lr {
				for rank := range lr[pi] {
					results[pi][rank] += lr[pi][rank]
				}
			}
			mu.Unlock()
		}(localStandings, workerRand, localResults, workerSims)
	}
	wg.Wait()
	return totalSims, timeLimitExceeded
}

// runParallelSimForceWinner distributes sims iterations of simForceWinner across
// runtime.NumCPU() goroutines, each with its own standings and pairings copies.
func (standings *Standings) runParallelSimForceWinner(
	copRand *rand.Rand, sims int, roundsRemaining int, maxFactor int,
	pairings [][]int, forcedWinnerPlayerIdx int, vsFirst bool, stopTimeNano int64,
) (int, pb.PairError) {
	numWorkers := runtime.NumCPU()
	if numWorkers > sims {
		numWorkers = sims
	}
	simsPerWorker := sims / numWorkers
	remainder := sims % numWorkers

	var wg sync.WaitGroup
	var mu sync.Mutex
	totalWins := 0
	var firstErr pb.PairError

	for w := 0; w < numWorkers; w++ {
		workerSims := simsPerWorker
		if w < remainder {
			workerSims++
		}
		wg.Add(1)
		localStandings := standings.Copy()
		workerRand := rand.New(rand.NewSource(uint64(copRand.Int63())))
		localPairings := make([][]int, len(pairings))
		for i := range pairings {
			localPairings[i] = make([]int, len(pairings[i]))
			copy(localPairings[i], pairings[i])
		}
		go func(sc *Standings, wr *rand.Rand, lp [][]int, ns int) {
			defer wg.Done()
			wins, pairErr := sc.simForceWinner(wr, ns, roundsRemaining, maxFactor, lp, forcedWinnerPlayerIdx, vsFirst, stopTimeNano)
			mu.Lock()
			totalWins += wins
			if pairErr != pb.PairError_SUCCESS && firstErr == pb.PairError_SUCCESS {
				firstErr = pairErr
			}
			mu.Unlock()
		}(localStandings, workerRand, localPairings, workerSims)
	}
	wg.Wait()
	if firstErr != pb.PairError_SUCCESS {
		return 0, firstErr
	}
	return totalWins, pb.PairError_SUCCESS
}

// RunParallelSimForceWinner runs vsFirst or vsFactor simulations for forcedWinnerPlayerIdx using
// the provided pairings. The player always wins every game; vsFirst=true swaps them to play rank 0
// in each non-final round, vsFirst=false uses the pairings as given. Returns the number of
// tournament wins (finishing in 1st place) out of sims.
//
// When the standings have an odd number of players, a dummy bye player is temporarily appended
// (mirroring SimFactorPairAll) so simRound can iterate in pairs. The caller must supply pairings
// with an even number of entries per round (pairingsLen = numPlayers rounded up to even).
func (standings *Standings) RunParallelSimForceWinner(copRand *rand.Rand, sims int, roundsRemaining int, maxFactor int, pairings [][]int, forcedWinnerPlayerIdx int, vsFirst bool, stopTimeNano int64) (int, pb.PairError) {
	numPlayers := standings.GetNumPlayers()
	evenerAdded := false
	if numPlayers%2 == 1 {
		lowestWins := getWinsValue(standings.records[numPlayers-1])
		standings.records = append(standings.records, getRecordFromWinsAndSpread(lowestWins-(roundsRemaining+1)*2, initialSpreadValue))
		standings.records[numPlayers] += uint64(ByePlayerIndex)
		numPlayers++
		standings.recordsBackup = make([]uint64, numPlayers)
		evenerAdded = true
	}
	wins, pairErr := standings.runParallelSimForceWinner(copRand, sims, roundsRemaining, maxFactor, pairings, forcedWinnerPlayerIdx, vsFirst, stopTimeNano)
	if evenerAdded {
		standings.records = standings.records[:numPlayers-1]
		standings.recordsBackup = make([]uint64, numPlayers-1)
	}
	return wins, pairErr
}

// RunSimsWithPairings runs parallel sims using the provided pre-built pairings and returns
// (FinalRanks, totalSims, error). This is used when the pairings have already been constructed
// externally (e.g. factor-3 pairings where roundsRemaining < maxFactor).
func (standings *Standings) RunSimsWithPairings(copRand *rand.Rand, sims int, roundsRemaining int, pairings [][]int) ([][]int, int, pb.PairError) {
	numPlayers := standings.GetNumPlayers()
	evenerAdded := false
	if numPlayers%2 == 1 {
		lowestWins := getWinsValue(standings.records[numPlayers-1])
		standings.records = append(standings.records, getRecordFromWinsAndSpread(lowestWins-(roundsRemaining+1)*2, initialSpreadValue))
		standings.records[numPlayers] += uint64(ByePlayerIndex)
		numPlayers++
		standings.recordsBackup = make([]uint64, numPlayers)
		evenerAdded = true
	}
	results := make([][]int, numPlayers)
	for i := range results {
		results[i] = make([]int, numPlayers)
	}
	playerIdxToRankIdx := standings.getPlayerIdxToRankIdxMap()
	standings.Backup()
	stopTime := time.Now().UnixNano() + int64(initialSimTimeLimit)*1e9
	totalSims, _ := standings.runParallelSims(sims, copRand, roundsRemaining, pairings, results, playerIdxToRankIdx, stopTime)
	if evenerAdded {
		standings.records = standings.records[:numPlayers-1]
		standings.recordsBackup = make([]uint64, numPlayers-1)
		for i := range results {
			results[i] = results[i][:numPlayers-1]
		}
		results = results[:numPlayers-1]
	}
	return results, totalSims, pb.PairError_SUCCESS
}

// Unexported functions

// assignPairingsForSegment sets the simulation pairings for players in [startRankIdx, endRankIdx]
// for all remaining rounds.
//
// When controlLoss is false (normal sim): each round uses multiple factor pairs followed by
// factor M/2 pairings for the remainder (where M is the number of remaining players).
//
// When controlLoss is true (always-wins sim): each round uses a single factor pair (top of
// segment vs Nth player), then randomly pairs remaining players within totalNumPlayers/2
// rank distance.
func assignPairingsForSegment(pairingsStartIdx int, startRankIdx int, endRankIdx int, initialRoundsRemaining int, maxFactor int, leftoverGibsonPlayers []int, pairings [][]int, segmentRoundFactors *[]int, copRand *rand.Rand, totalNumPlayers int, controlLoss bool) {
	numPlayers := endRankIdx - startRankIdx + 1

	for roundsRemaining := initialRoundsRemaining; roundsRemaining > 0; roundsRemaining-- {
		roundFactor := roundsRemaining
		if roundFactor > maxFactor {
			roundFactor = maxFactor
		}
		maxPlayerFactor := numPlayers / 2
		if roundFactor > maxPlayerFactor {
			roundFactor = maxPlayerFactor
		}
		*segmentRoundFactors = append(*segmentRoundFactors, roundFactor)
		roundIdx := initialRoundsRemaining - roundsRemaining

		// The last simulated round always uses KOTH (consecutive pairing) for all players.
		if roundsRemaining == 1 {
			for kothPairing := 0; kothPairing < numPlayers/2; kothPairing++ {
				basePairingIdx := pairingsStartIdx + 2*kothPairing
				basePlayerIdx := startRankIdx + 2*kothPairing
				pairings[roundIdx][basePairingIdx] = basePlayerIdx
				pairings[roundIdx][basePairingIdx+1] = basePlayerIdx + 1
			}
			if numPlayers%2 == 1 {
				pairings[roundIdx][pairingsStartIdx+numPlayers-1] = startRankIdx + numPlayers - 1
			}
			if len(leftoverGibsonPlayers) > 0 {
				baseGibsonPairingIdx := pairingsStartIdx + numPlayers
				for playerIdx := 0; playerIdx < len(leftoverGibsonPlayers); playerIdx++ {
					pairings[roundIdx][baseGibsonPairingIdx+playerIdx] = leftoverGibsonPlayers[playerIdx]
				}
			}
			continue
		}

		if controlLoss {
			// Single factor pair: top of segment vs Nth player.
			pairings[roundIdx][pairingsStartIdx] = startRankIdx
			pairings[roundIdx][pairingsStartIdx+1] = startRankIdx + roundFactor

			// Collect remaining players (all except the factor pair).
			remaining := make([]int, 0, numPlayers-2)
			for rankIdx := startRankIdx; rankIdx <= endRankIdx; rankIdx++ {
				if rankIdx != startRankIdx && rankIdx != startRankIdx+roundFactor {
					remaining = append(remaining, rankIdx)
				}
			}

			// Randomly pair remaining players within totalNumPlayers/2 rank distance.
			// Shuffle and pair consecutively; retry up to 10 times to reduce violations.
			halfP := totalNumPlayers / 2
			best := make([]int, len(remaining))
			copy(best, remaining)
			bestViolations := countDistanceViolations(best, halfP)
			shuffled := make([]int, len(remaining))
			copy(shuffled, remaining)
			for attempt := 0; attempt < 10 && bestViolations > 0; attempt++ {
				copRand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
				if v := countDistanceViolations(shuffled, halfP); v < bestViolations {
					copy(best, shuffled)
					bestViolations = v
				}
			}

			for i := 0; i+1 < len(best); i += 2 {
				pairings[roundIdx][pairingsStartIdx+2+i] = best[i]
				pairings[roundIdx][pairingsStartIdx+2+i+1] = best[i+1]
			}
			if len(best)%2 == 1 {
				pairings[roundIdx][pairingsStartIdx+2+len(best)-1] = best[len(best)-1]
			}
		} else {
			// Multiple factor pairs followed by factor M/2 pairings for the remainder,
			// where M is the number of remaining players.
			for factorPairing := 0; factorPairing < roundFactor; factorPairing++ {
				basePairingIdx := pairingsStartIdx + 2*factorPairing
				basePlayerIdx := startRankIdx + factorPairing
				pairings[roundIdx][basePairingIdx] = basePlayerIdx
				pairings[roundIdx][basePairingIdx+1] = basePlayerIdx + roundFactor
			}
			numRemainingPlayers := numPlayers - 2*roundFactor
			remainingFactor := numRemainingPlayers / 2
			remainingStartRankIdx := startRankIdx + 2*roundFactor
			remainingStartPairingIdx := pairingsStartIdx + 2*roundFactor
			for remainingPairing := 0; remainingPairing < remainingFactor; remainingPairing++ {
				basePairingIdx := remainingStartPairingIdx + 2*remainingPairing
				basePlayerIdx := remainingStartRankIdx + remainingPairing
				pairings[roundIdx][basePairingIdx] = basePlayerIdx
				pairings[roundIdx][basePairingIdx+1] = basePlayerIdx + remainingFactor
			}
			if numRemainingPlayers%2 == 1 {
				pairings[roundIdx][remainingStartPairingIdx+numRemainingPlayers-1] = remainingStartRankIdx + numRemainingPlayers - 1
			}
		}

		if len(leftoverGibsonPlayers) > 0 {
			baseGibsonPairingIdx := pairingsStartIdx + numPlayers
			for playerIdx := 0; playerIdx < len(leftoverGibsonPlayers); playerIdx++ {
				pairings[roundIdx][baseGibsonPairingIdx+playerIdx] = leftoverGibsonPlayers[playerIdx]
			}
		}
	}
}

func countDistanceViolations(players []int, maxDist int) int {
	violations := 0
	for i := 0; i+1 < len(players); i += 2 {
		d := players[i] - players[i+1]
		if d < 0 {
			d = -d
		}
		if d > maxDist {
			violations++
		}
	}
	return violations
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

// ClopperPearson returns the (lower, upper) exact CI for a Binomial proportion
// using the Clopper–Pearson method at confidence level 1 - alpha.
func clopperPearson(k int, n int, alpha float64) (float64, float64) {
	var lower, upper float64
	if k == 0 {
		lower = 0.0
	} else {
		beta := distuv.Beta{Alpha: float64(k), Beta: float64(n - k + 1)}
		lower = beta.Quantile(alpha / 2)
	}

	if k == n {
		upper = 1.0
	} else {
		beta := distuv.Beta{Alpha: float64(k + 1), Beta: float64(n - k)}
		upper = beta.Quantile(1 - alpha/2)
	}

	return lower, upper
}

// Assumes n > 0
func simResultsReachedThreshold(results [][]int, ranksToCheck []int, n int, y float64) bool {
	numRanks := len(results)
	N := numRanks * len(ranksToCheck)
	alphaPer := (1 - simConfidence) / float64(N)
	for _, rankToCheck := range ranksToCheck {
		for rankIdx := range numRanks {
			lower, upper := clopperPearson(results[rankIdx][rankToCheck], n, alphaPer)
			if !(lower > y || upper < y) {
				// Inconclusive if CI spans threshold
				return false
			}
		}
	}
	return true
}
