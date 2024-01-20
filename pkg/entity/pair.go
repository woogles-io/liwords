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
}
