package omgwords

import (
	"strings"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestSetAnnotatedGameBoard(t *testing.T) {
	is := is.New(t)
	cases := []struct {
		lexicon, variant          string
		board, tiles, wantVariant string
	}{
		{"CSW24", "classic", "CrosswordGame", "english", "classic"},
		{"CSW24", "", "CrosswordGame", "english", "classic"},
		{"CSW24", "wordsmog", "CrosswordGame", "english", "wordsmog"},
		{"CSW24", "classic_super", "SuperCrosswordGame", "english_super", "classic_super"},
		{"NWL23", "wordsmog_super", "SuperCrosswordGame", "english_super", "wordsmog_super"},
		{"RD28", "classic", "CrosswordGame", "german", "classic"},
	}
	for _, c := range cases {
		// Whatever the client asked for, the variant decides.
		rules := &ipc.GameRules{
			BoardLayoutName:        "SuperCrosswordGame",
			LetterDistributionName: "catalan",
			VariantName:            c.variant,
		}
		is.NoErr(setAnnotatedGameBoard(c.lexicon, rules))
		is.Equal(rules.BoardLayoutName, c.board)
		is.Equal(rules.LetterDistributionName, c.tiles)
		is.Equal(rules.VariantName, c.wantVariant)
	}
	is.True(setAnnotatedGameBoard("CSW24", &ipc.GameRules{VariantName: "chess"}) != nil)
	// The 21x21 board has only the English tiles.
	for _, lexicon := range []string{"RD28", "DISC2"} {
		err := setAnnotatedGameBoard(lexicon, &ipc.GameRules{VariantName: "classic_super"})
		is.True(err != nil)
		is.True(strings.Contains(err.Error(), "ZOMGWords plays with English lexica only"))
	}
}
