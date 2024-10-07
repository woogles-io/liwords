package pair

import (
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
	records []uint64
}

const (
	PlayerWinsOffset   int    = 48
	PlayerSpreadOffset int    = 16
	InitialSpreadValue int    = 1 << (PlayerWinsOffset - PlayerSpreadOffset - 1)
	PlayerIndexMask    uint64 = 0xFFFF
)

func CreateEmptyStandings(numPlayers int) *Standings {
	standings := &Standings{}
	standings.records = make([]uint64, numPlayers)
	for playerIdx := 0; playerIdx < numPlayers; playerIdx++ {
		standings.records[playerIdx] = uint64(playerIdx)
		standings.IncrementPlayerSpread(playerIdx, InitialSpreadValue)
	}
	return standings
}

func (standings *Standings) IncrementPlayerWins(rankIdx int) {
	standings.records[rankIdx] += uint64(2) << PlayerWinsOffset
}

func (standings *Standings) IncrementPlayerTies(rankIdx int) {
	standings.records[rankIdx] += uint64(1) << PlayerWinsOffset
}

func (standings *Standings) IncrementPlayerSpread(rankIdx int, spread int) {
	if spread < 0 {
		standings.records[rankIdx] -= uint64((-spread)) << PlayerSpreadOffset
	} else {
		standings.records[rankIdx] += uint64(spread) << PlayerSpreadOffset
	}
}

func (standings *Standings) GetPlayerIndex(rankIdx int) int {
	return int(standings.records[rankIdx] & PlayerIndexMask)
}

func (standings *Standings) getPlayerWinInternal(rankIdx int) uint64 {
	return (standings.records[rankIdx] >> PlayerWinsOffset) & 0xFFFF
}

func (standings *Standings) GetPlayerWins(rankIdx int) float64 {
	return float64(standings.getPlayerWinInternal(rankIdx)) / 2
}

func (standings *Standings) getPlayerSpreadInternal(rankIdx int) uint64 {
	return (standings.records[rankIdx] >> PlayerSpreadOffset) & 0xFFFFFFFF
}

func (standings *Standings) GetPlayerSpread(rankIdx int) int {
	spread := standings.getPlayerSpreadInternal(rankIdx)
	if spread > uint64(InitialSpreadValue) {
		return int(spread - uint64(InitialSpreadValue))
	} else {
		return -int(uint64(InitialSpreadValue) - spread)
	}
}

// Assumes the standings are already sorted and i < j
func (standings *Standings) CanCatch(roundsRemaining int, gibsonSpread int, i int, j int) bool {
	winIntDiff := standings.getPlayerWinInternal(i) - standings.getPlayerWinInternal(j)
	highestPossibleWinIntScore := uint64(roundsRemaining) * 2
	if winIntDiff != highestPossibleWinIntScore {
		return winIntDiff < highestPossibleWinIntScore
	}
	piSpread := standings.getPlayerSpreadInternal(i)
	pjSpread := standings.getPlayerSpreadInternal(j)
	if pjSpread > piSpread {
		return true
	}
	return (piSpread - pjSpread) <= uint64(gibsonSpread)
}

func (standings *Standings) Sort() {
	sort.Slice(standings.records, func(i, j int) bool {
		return standings.records[i] > standings.records[j]
	})
}
