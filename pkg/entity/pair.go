package entity

import (
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// These weights were made very large
// out of an abundance of caution so that
// any single win weight outweighs the sum
// of all possible spread weights

const ProhibitiveWeight int64 = 1 << 52
const WinWeightScaling int64 = 1 << 22
const MaxRelativeWeight int = 100

// These constants control the swiss weighing function
const DifferencePenalty = 1
const DifferencePenaltyMargin = 2

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
	RoundControls *ipc.RoundControl
	Repeats       map[string]int
	// RepeatRounds is the per-round opponent history keyed by the same
	// GetRepeatKey as Repeats: for each pair it lists, in ascending order,
	// the 0-indexed rounds in which the two players met (byes excluded).
	// Repeats carries only totals, which is enough for the matching-based
	// methods, but the Australian Draw needs to know which round a meeting
	// happened in so it can avoid only repeats at or after a reset round.
	// It is populated by the caller solely for that method and may be nil.
	RepeatRounds map[string][]int
	Seed         uint64
}
