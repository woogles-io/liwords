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
	for _, lexicon := range []string{"CSW15", "CSW19", "CSW21", "NSF25", "NWL20"} {
		err := ValidateGameRequest(ctx, newGameRequest(lexicon))
		is.True(err != nil)
		is.Equal(err.Error(), lexicon+" is not a supported lexicon")
	}
}

// A super variant plays its lexicon's 200-tile set on the 21x21 board:
// English and Catalan have one.
func TestSuperVariantsTakeTheLexiconsSuperSet(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	for lexicon, want := range map[string]string{
		"CSW24": "english_super",
		"NWL23": "english_super",
		"DISC2": "catalan_super",
	} {
		for _, variant := range []string{"classic_super", "wordsmog_super"} {
			req := newGameRequest(lexicon)
			// whatever the client sent
			req.Rules.LetterDistributionName = "german"
			req.Rules.VariantName = variant
			is.NoErr(ValidateGameRequest(ctx, req))
			is.Equal(req.Rules.LetterDistributionName, want)
			is.Equal(req.Rules.BoardLayoutName, "SuperCrosswordGame")
			is.NoErr(ValidateLetterDistribution(lexicon, want))
		}
	}
	// no super set; NSWL23 was not offered the super variants before either
	for _, lexicon := range []string{"RD29", "FRA24", "NSWL23"} {
		req := newGameRequest(lexicon)
		req.Rules.VariantName = "classic_super"
		err := ValidateGameRequest(ctx, req)
		is.True(err != nil)
		is.Equal(err.Error(), "that lexicon is not supported for this variant")
	}
	// a super set fits only its own alphabet
	is.True(ValidateLetterDistribution("CSW24", "catalan_super") != nil)
	is.True(ValidateLetterDistribution("DISC2", "english_super") != nil)
}
