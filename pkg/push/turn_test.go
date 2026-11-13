package push

import (
	"testing"

	"github.com/matryer/is"
)

func TestTurnMessage(t *testing.T) {
	is := is.New(t)
	is.Equal(string(TurnMessage("g1", "cesar4", 1_700_000_000_000)),
		`{"gameId":"g1","title":"Your turn vs cesar4","body":"It's your turn to play","deadline":1700000000000}`)
	is.Equal(string(TurnMessage("g1", "", 5)),
		`{"gameId":"g1","title":"Your turn vs Opponent","body":"It's your turn to play","deadline":5}`)
}
