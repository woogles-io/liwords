package entity

import (
	"context"
	"testing"

	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func newGameRequest(lexicon string) *pb.GameRequest {
	return &pb.GameRequest{
		Lexicon:            lexicon,
		Rules:              &pb.GameRules{},
		InitialTimeSeconds: 60,
	}
}

// AllowedNewGameLexica is the only gate keeping seeks, bot games and
// tournaments off historical lexica.
func TestHistoricalLexicaCannotStartGames(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	is.NoErr(ValidateGameRequest(ctx, newGameRequest("CSW24")))
	for _, lexicon := range []string{"CSW15", "CSW19", "CSW21", "NWL20"} {
		err := ValidateGameRequest(ctx, newGameRequest(lexicon))
		is.True(err != nil)
		is.Equal(err.Error(), lexicon+" is not a supported lexicon")
	}
}
