package pair

import (
	"fmt"
	"math/rand"
	"sort"
)

// Player standings are implemented as an array of uint64s.
// The 16 most significant bits represent the win score
// The next 32 most significant bits represent the spread
// The last 16 bits represent the player index
// Wins have a value of 2
// Ties have a value of 1
// Losses have a value of 0
type Standings struct {
	records         []uint64
	possibleResults []uint64
}

const (
	PlayerWinsOffset   int    = 48
	PlayerSpreadOffset int    = 16
	InitialSpreadValue int    = 1 << (PlayerWinsOffset - PlayerSpreadOffset - 1)
	PlayerIndexMask    uint64 = 0xFFFF
	MaxSpread          int    = 300
	ByeSpread          int    = 50
)

// Record methods

func incrementWins(record *uint64) {
	*record += uint64(2) << PlayerWinsOffset
}

func incrementTies(record *uint64) {
	*record += uint64(1) << PlayerWinsOffset
}

func incrementSpread(record *uint64, spread int) {
	if spread < 0 {
		*record -= uint64((-spread)) << PlayerSpreadOffset
	} else {
		*record += uint64(spread) << PlayerSpreadOffset
	}
}

func incrementRecord(record *uint64, incRecord uint64) {
	*record += incRecord
}

func decrementRecord(record *uint64, incRecord uint64) {
	*record -= incRecord
}

func getIndex(record uint64) int {
	return int(record & PlayerIndexMask)
}

func getWins(record uint64) int {
	return int((record >> PlayerWinsOffset) & 0xFFFF)
}

func getWinsHumanReadable(record uint64) float64 {
	return float64(getWins(record)) / 2
}

func getSpread(record uint64) uint64 {
	return (record >> PlayerSpreadOffset) & 0xFFFFFFFF
}

func getSpreadHumanReadable(record uint64) int {
	spread := getSpread(record)
	if spread > uint64(InitialSpreadValue) {
		return int(spread - uint64(InitialSpreadValue))
	} else {
		return -int(uint64(InitialSpreadValue) - spread)
	}
}

// Standings Methods

func CreateEmptyStandings(numPlayers int) *Standings {
	standings := &Standings{}
	standings.records = make([]uint64, numPlayers)
	for playerIdx := 0; playerIdx < numPlayers; playerIdx++ {
		standings.records[playerIdx] = uint64(playerIdx)
		standings.IncrementPlayerSpread(playerIdx, InitialSpreadValue)
	}
	standings.possibleResults = make([]uint64, (MaxSpread+1)*2)
	for spread := 0; spread < MaxSpread; spread++ {
		baseResultIdx := spread * 2
		if spread == 0 {
			// Set the tie results
			incrementTies(&standings.possibleResults[baseResultIdx])
			incrementTies(&standings.possibleResults[baseResultIdx+1])
		} else {
			// Set the winning result
			incrementWins(&standings.possibleResults[baseResultIdx])
			incrementSpread(&standings.possibleResults[baseResultIdx], spread)
			// Set the losing result
			incrementSpread(&standings.possibleResults[baseResultIdx+1], spread)
		}
	}
	return standings
}

func (standings *Standings) IncrementPlayerWins(rankIdx int) {
	incrementWins(&standings.records[rankIdx])
}

func (standings *Standings) IncrementPlayerTies(rankIdx int) {
	incrementTies(&standings.records[rankIdx])
}

func (standings *Standings) IncrementPlayerSpread(rankIdx int, spread int) {
	incrementSpread(&standings.records[rankIdx], spread)
}

func (standings *Standings) incrementPlayerRecord(rankIdx int, incRecord uint64) {
	incrementRecord(&standings.records[rankIdx], incRecord)
}

func (standings *Standings) decrementPlayerRecord(rankIdx int, incRecord uint64) {
	decrementRecord(&standings.records[rankIdx], incRecord)
}

func (standings *Standings) GetPlayerIndex(rankIdx int) int {
	return getIndex(standings.records[rankIdx])
}

func (standings *Standings) GetPlayerWins(rankIdx int) float64 {
	return getWinsHumanReadable(standings.records[rankIdx])
}

func (standings *Standings) GetPlayerSpread(rankIdx int) int {
	return getSpreadHumanReadable(standings.records[rankIdx])
}

// Assumes the standings are already sorted and i < j
func (standings *Standings) CanCatch(roundsRemaining int, cumeGibsonSpread int, i int, j int) bool {
	ri := standings.records[i]
	rj := standings.records[j]
	winDiff := getWins(ri) - getWins(rj)
	// Multiply by 2 because wins count as 2 and ties count as 1
	highestPossibleWinScore := roundsRemaining * 2
	if winDiff != highestPossibleWinScore {
		return winDiff < highestPossibleWinScore
	}
	piSpread := getSpread(ri)
	pjSpread := getSpread(rj)
	if pjSpread > piSpread {
		return true
	}
	return (piSpread - pjSpread) <= uint64(cumeGibsonSpread)
}

func (standings *Standings) Sort() {
	sort.Slice(standings.records, func(i, j int) bool {
		return standings.records[i] > standings.records[j]
	})
}

func (standings *Standings) sortRange(i, j int) {
	sort.Slice(standings.records[i:j], func(x, y int) bool {
		return standings.records[i+x] > standings.records[i+y]
	})
}

// Assumes the standings are already sorted
func (standings *Standings) SimFactorPair(sims int, maxFactor int, roundsRemaining int, gibsonizedPlayers []bool) [][]int {
	numPlayers := len(standings.records)
	results := make([][]int, numPlayers)
	for i := range results {
		results[i] = make([]int, numPlayers)
	}
	i := 0
	j := 0
	for j <= numPlayers {
		if j == numPlayers {
			standings.simFactorPairSegment(results, i, j, roundsRemaining, maxFactor, sims)
			break
		}
		if gibsonizedPlayers[j] {
			// FIXME: add results for gibsonized player
			if j != i {
				// FIXME: explain this
				includeGibsonizedPlayer := 0
				if j-i%2 == 1 {
					includeGibsonizedPlayer = 1
				}
				standings.simFactorPairSegment(results, i, j+includeGibsonizedPlayer, roundsRemaining, maxFactor, sims)
			}
			j++
			i = j
		} else {
			j++
		}
	}

	return results
}

// Factor pair the players in [i, j)
// Assumes i < j
func (standings *Standings) simFactorPairSegment(results [][]int, i int, j int, roundsRemaining int, maxFactor int, sims int) {
	numPlayers := j - i
	pairings := make([][]int, roundsRemaining)
	for i := 0; i < roundsRemaining; i++ {
		pairings[i] = make([]int, numPlayers)
	}

	for factor := roundsRemaining; factor > 0; factor-- {
		roundFactor := factor
		if roundFactor > maxFactor {
			roundFactor = maxFactor
		}
		maxPlayerFactor := numPlayers / 2
		if roundFactor > maxPlayerFactor {
			roundFactor = maxPlayerFactor
		}
		roundIdx := roundsRemaining - factor
		pairIdx := 0
		for k := 0; k < roundFactor; k++ {
			pairings[roundIdx][pairIdx] = k
			pairIdx += 1
			pairings[roundIdx][pairIdx] = k + roundFactor
			pairIdx += 1
		}
		for k := roundFactor * 2; k < numPlayers; k += 2 {
			pairings[roundIdx][pairIdx] = k
			pairIdx += 1
			if pairIdx == numPlayers {
				break
			}
			pairings[roundIdx][pairIdx] = k + 1
			pairIdx += 1
		}
	}

	oddNumPlayers := numPlayers%2 == 1

	startingRecords := make([]uint64, len(standings.records))

	copy(startingRecords, standings.records)

	for simIdx := 0; simIdx < sims; simIdx++ {
		for roundIdx := 0; roundIdx < roundsRemaining; roundIdx++ {
			fmt.Printf("round: %d\n", roundIdx)
			for pairIdx := 0; pairIdx < numPlayers-1; pairIdx += 2 {
				randomResult := rand.Intn(2)
				p1 := pairings[roundIdx][pairIdx]
				p2 := pairings[roundIdx][pairIdx+1]
				fmt.Printf("p1: %d, p2: %d, randomResult: %d\n", p1, p2, randomResult)
				winner := p1*(1-randomResult) + p2*randomResult
				loser := p2*(1-randomResult) + p1*randomResult
				randomSpread := rand.Intn(MaxSpread + 1)
				standings.incrementPlayerRecord(winner, standings.possibleResults[randomSpread*2])
				standings.decrementPlayerRecord(loser, standings.possibleResults[randomSpread*2+1])
			}
			// FIXME: Branch predictor should be able to predict this close to 100%
			// of the time since the value of oddNumPlayers never changes in this function
			// but test it to be sure
			if oddNumPlayers {
				standings.incrementPlayerRecord(numPlayers-1, standings.possibleResults[ByeSpread*2])
			}
			standings.sortRange(i, j)
		}
		for rankIdx := i; rankIdx < j; rankIdx++ {
			playerIdx := getIndex(standings.records[rankIdx])
			results[playerIdx][rankIdx] += 1
		}
		copy(standings.records, startingRecords)
	}
}
