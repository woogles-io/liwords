package standings

import (
	"fmt"
	"sort"
	"strings"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"

	"golang.org/x/exp/rand"
)

const (
	playerWinsOffset   int    = 48
	playerSpreadOffset int    = 16
	initialSpreadValue int    = 1 << (playerWinsOffset - playerSpreadOffset - 1)
	playerIndexMask    uint64 = 0xFFFF
	maxSpread          int    = 300
	byeSpread          int    = 50
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

// Exported functions

func CreateInitialStandings(req *pb.PairRequest) *Standings {
	// Create empty standings
	standings := &Standings{}
	standings.records = make([]uint64, req.Players)
	for playerIdx := 0; playerIdx < int(req.Players); playerIdx++ {
		standings.records[playerIdx] = uint64(playerIdx)
		standings.IncrementPlayerSpread(playerIdx, initialSpreadValue)
	}

	// Initialize the possible results
	standings.possibleResults = make([]uint64, (maxSpread+1)*2)
	for spread := 0; spread < maxSpread; spread++ {
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

	// Update the standings wuth the pairing results
	for roundIdx, roundResults := range req.DivisionResults {
		for playerIdx, playerScore := range roundResults.Results {
			oppIdx := int(req.DivisionPairings[roundIdx].Pairings[playerIdx])
			if playerIdx == int(oppIdx) {
				// Bye
				if playerScore >= 0 {
					standings.IncrementPlayerWins(playerIdx)
				}
				standings.IncrementPlayerSpread(playerIdx, int(playerScore))
			} else if playerIdx < oppIdx {
				oppScore := roundResults.Results[oppIdx]
				playerSpread := playerScore - oppScore
				oppSpread := oppScore - playerScore
				if playerSpread > 0 {
					standings.IncrementPlayerWins(playerIdx)
				} else if playerSpread < 0 {
					standings.IncrementPlayerWins(oppIdx)
				} else {
					standings.IncrementPlayerTies(playerIdx)
					standings.IncrementPlayerTies(oppIdx)
				}
				standings.IncrementPlayerSpread(playerIdx, int(playerSpread))
				standings.IncrementPlayerSpread(oppIdx, int(oppSpread))
			}
		}
	}

	standings.Sort()

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

// Assumes the standings are already sorted
func (standings *Standings) SimFactorPair(sims int, maxFactor int, roundsRemaining int, gibsonizedPlayers []bool) [][]int {
	numPlayers := len(standings.records)
	results := make([][]int, numPlayers)
	for i := range results {
		results[i] = make([]int, numPlayers)
	}
	// FIXME: probably need better names for i and j
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

func (standings *Standings) String(req *pb.PairRequest) string {
	maxNameLength := 0
	for _, playerName := range req.PlayerNames {
		if len(playerName) > maxNameLength {
			if len(playerName) > 30 {
				maxNameLength = 30
			} else {
				maxNameLength = len(playerName)
			}
		}
	}

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

	for rankIdx := 0; rankIdx < int(req.Players); rankIdx++ {
		playerIdx := standings.GetPlayerIndex(rankIdx)
		wins := standings.GetPlayerWins(rankIdx)
		spread := standings.GetPlayerSpread(rankIdx)
		playerName := req.PlayerNames[playerIdx]
		if len(playerName) > 30 {
			playerName = playerName[:30]
		}
		sb.WriteString(fmt.Sprintf(rowFormat, rankIdx+1, playerName, wins, spread))
	}

	return sb.String()
}

// Unexported functions

func incrementWins(record *uint64) {
	*record += uint64(2) << playerWinsOffset
}

func incrementTies(record *uint64) {
	*record += uint64(1) << playerWinsOffset
}

func incrementSpread(record *uint64, spread int) {
	if spread < 0 {
		*record -= uint64((-spread)) << playerSpreadOffset
	} else {
		*record += uint64(spread) << playerSpreadOffset
	}
}

func incrementRecord(record *uint64, incRecord uint64) {
	*record += incRecord
}

func decrementRecord(record *uint64, incRecord uint64) {
	*record -= incRecord
}

func getIndex(record uint64) int {
	return int(record & playerIndexMask)
}

func getWins(record uint64) int {
	return int((record >> playerWinsOffset) & 0xFFFF)
}

func getWinsHumanReadable(record uint64) float64 {
	return float64(getWins(record)) / 2
}

func getSpread(record uint64) uint64 {
	return (record >> playerSpreadOffset) & 0xFFFFFFFF
}

func getSpreadHumanReadable(record uint64) int {
	spread := getSpread(record)
	if spread > uint64(initialSpreadValue) {
		return int(spread - uint64(initialSpreadValue))
	} else {
		return -int(uint64(initialSpreadValue) - spread)
	}
}

// Private standings methods

func (standings *Standings) incrementPlayerRecord(rankIdx int, incRecord uint64) {
	incrementRecord(&standings.records[rankIdx], incRecord)
}

func (standings *Standings) decrementPlayerRecord(rankIdx int, incRecord uint64) {
	decrementRecord(&standings.records[rankIdx], incRecord)
}

func (standings *Standings) sortRange(i, j int) {
	sort.Slice(standings.records[i:j], func(x, y int) bool {
		return standings.records[i+x] > standings.records[i+y]
	})
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

	numTotalPlayers := len(standings.records)
	oddNumPlayers := numTotalPlayers%2 == 1
	startingRecords := make([]uint64, numTotalPlayers)
	copy(startingRecords, standings.records)

	for simIdx := 0; simIdx < sims; simIdx++ {
		for roundIdx := 0; roundIdx < roundsRemaining; roundIdx++ {
			for pairIdx := 0; pairIdx < numPlayers-1; pairIdx += 2 {
				randomResult := rand.Intn(2)
				p1 := pairings[roundIdx][pairIdx]
				p2 := pairings[roundIdx][pairIdx+1]
				winner := p1*(1-randomResult) + p2*randomResult
				loser := p2*(1-randomResult) + p1*randomResult
				randomSpread := rand.Intn(maxSpread + 1)
				standings.incrementPlayerRecord(winner, standings.possibleResults[randomSpread*2])
				standings.decrementPlayerRecord(loser, standings.possibleResults[randomSpread*2+1])
			}
			// FIXME: Branch predictor should be able to predict this close to 100%
			// of the time since the value of oddNumPlayers never changes in this function
			// but test it to be sure
			if oddNumPlayers {
				standings.incrementPlayerRecord(numPlayers-1, standings.possibleResults[byeSpread*2])
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
