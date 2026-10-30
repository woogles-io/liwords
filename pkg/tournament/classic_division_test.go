package tournament

import (
	"math/rand/v2"
	"sort"
	"testing"

	"github.com/matryer/is"

	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestPlayerSorterBreaksTiesByName(t *testing.T) {
	is := is.New(t)
	persons := []*ipc.TournamentPerson{
		{Id: "u3:carol", Rating: 1500},
		{Id: "u1:bob", Rating: 1500},
		{Id: "u2:alice", Rating: 1500},
		{Id: "u4:dave", Rating: 1800},
	}
	for range 20 {
		rand.Shuffle(len(persons), func(i, j int) { persons[i], persons[j] = persons[j], persons[i] })
		sort.Sort(PlayerSorter(persons))
		ids := []string{}
		for _, p := range persons {
			ids = append(ids, p.Id)
		}
		is.Equal(ids, []string{"u4:dave", "u2:alice", "u1:bob", "u3:carol"})
	}
}
