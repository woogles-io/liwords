package entity

// These weights were made very large
// out of an abundance of caution so that
// any single win weight outweighs the sum
// of all possible spread weights

const ProhibitiveWeight int64 = 1 << 52

// If spreads are greater than this number
// stuff will break
const MaxSpreadWeight int64 = 1 << 12

// Win weight must be much greater than
// spread weight
const WinWeightScaling int64 = 1 << 22
const MaxRelativeWeight int = 100

type PairingMethod int

const (
	Random PairingMethod = iota
	RoundRobin
	KingOfTheHill
	Elimination
	Factor
	Swiss
	Quickpair
	// Need to implement eventually
	// Performance

	// Manual simply does not make any
	// pairings at all. The director
	// has to make all the pairings themselves.
	Manual
)

type PoolMember struct {
	Id          string
	Rating      int
	RatingRange [2]int
	Blocking    []string
	Misses      int
	Wins        int
	Draws       int
	Spread      int
}

type UnpairedPoolMembers struct {
	PoolMembers   []*PoolMember
	RoundControls *RoundControls
	Repeats       map[string]int
}
