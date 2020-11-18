package pair

import (
	"fmt"
	"github.com/matryer/is"
	"testing"

	"github.com/domino14/liwords/pkg/utilities"
)

// The vast majority of pairing tests are in the tournament package

func TestRoundRobin(t *testing.T) {
	// This test is used to ensure that round robin
	// pairings work correctly.

	is := is.New(t)

	roundRobinPlayers := []int{0, 1, 2, 3, 4, 5, 6, 7}

	is.NoErr(equalRoundRobinPairings([]int{7, 6, 5, 4, 3, 2, 1, 0},
		getRoundRobinPairings(roundRobinPlayers, 0)))
	is.NoErr(equalRoundRobinPairings([]int{6, 4, 3, 2, 1, 7, 0, 5},
		getRoundRobinPairings(roundRobinPlayers, 1)))
	is.NoErr(equalRoundRobinPairings([]int{5, 2, 1, 7, 6, 0, 4, 3},
		getRoundRobinPairings(roundRobinPlayers, 2)))
	is.NoErr(equalRoundRobinPairings([]int{4, 7, 6, 5, 0, 3, 2, 1},
		getRoundRobinPairings(roundRobinPlayers, 3)))
	is.NoErr(equalRoundRobinPairings([]int{3, 5, 4, 0, 2, 1, 7, 6},
		getRoundRobinPairings(roundRobinPlayers, 4)))
	is.NoErr(equalRoundRobinPairings([]int{2, 3, 0, 1, 7, 6, 5, 4},
		getRoundRobinPairings(roundRobinPlayers, 5)))
	is.NoErr(equalRoundRobinPairings([]int{1, 0, 7, 6, 5, 4, 3, 2},
		getRoundRobinPairings(roundRobinPlayers, 6)))

	// Modulus operation should repeat the pairings past the first round robin

	is.NoErr(equalRoundRobinPairings([]int{7, 6, 5, 4, 3, 2, 1, 0},
		getRoundRobinPairings(roundRobinPlayers, 7)))
	is.NoErr(equalRoundRobinPairings([]int{6, 4, 3, 2, 1, 7, 0, 5},
		getRoundRobinPairings(roundRobinPlayers, 8)))
	is.NoErr(equalRoundRobinPairings([]int{5, 2, 1, 7, 6, 0, 4, 3},
		getRoundRobinPairings(roundRobinPlayers, 9)))
	is.NoErr(equalRoundRobinPairings([]int{4, 7, 6, 5, 0, 3, 2, 1},
		getRoundRobinPairings(roundRobinPlayers, 10)))
	is.NoErr(equalRoundRobinPairings([]int{3, 5, 4, 0, 2, 1, 7, 6},
		getRoundRobinPairings(roundRobinPlayers, 11)))
	is.NoErr(equalRoundRobinPairings([]int{2, 3, 0, 1, 7, 6, 5, 4},
		getRoundRobinPairings(roundRobinPlayers, 12)))
	is.NoErr(equalRoundRobinPairings([]int{1, 0, 7, 6, 5, 4, 3, 2},
		getRoundRobinPairings(roundRobinPlayers, 13)))

	// Test first pairing of third round robin just to be sure

	is.NoErr(equalRoundRobinPairings([]int{7, 6, 5, 4, 3, 2, 1, 0},
		getRoundRobinPairings(roundRobinPlayers, 14)))
}

func equalRoundRobinPairings(s1 []int, s2 []int) error {
	if len(s1) != len(s2) {
		return fmt.Errorf("pairing lengths do not match: %d != %d", len(s1), len(s2))
	}
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			return fmt.Errorf("pairings are not equal: %s, %s", utilities.IntArrayToString(s1), utilities.IntArrayToString(s2))
		}
	}
	return nil
}
