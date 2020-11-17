package entity

const ProhibitiveWeight int = 1000000000

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
}

type UnpairedPoolMembers struct {
	PoolMembers   []*PoolMember
	RoundControls *RoundControls
	Repeats       map[string]int
}
