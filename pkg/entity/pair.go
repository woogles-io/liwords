package entity

import (
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

// These weights were made very large
// out of an abundance of caution so that
// any single win weight outweighs the sum
// of all possible spread weights

const ProhibitiveWeight int64 = 1 << 52
const WinWeightScaling int64 = 1 << 22
const MaxRelativeWeight int = 100

type PoolMember struct {
	Id          string
	Rating      int
	RatingRange [2]int
	Blocking    []string
	Misses      int
	Wins        int
	Draws       int
	Spread      int
	Removed     bool
}

type UnpairedPoolMembers struct {
	PoolMembers   []*PoolMember
	RoundControls *realtime.RoundControl
	Repeats       map[string]int
}
