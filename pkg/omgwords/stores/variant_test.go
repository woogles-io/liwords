package stores

import (
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestAnnotatedGameVariant(t *testing.T) {
	is := is.New(t)
	rules := func(board, variant string) *ipc.GameRequest {
		return &ipc.GameRequest{Rules: &ipc.GameRules{BoardLayoutName: board, VariantName: variant}}
	}
	is.Equal(annotatedGameVariant(rules("CrosswordGame", "wordsmog")), "wordsmog")
	is.Equal(annotatedGameVariant(rules("SuperCrosswordGame", "classic_super")), "classic_super")
	// stored before the variant was: the board tells
	is.Equal(annotatedGameVariant(rules("SuperCrosswordGame", "")), "classic_super")
	is.Equal(annotatedGameVariant(rules("CrosswordGame", "")), "classic")
	is.Equal(annotatedGameVariant(nil), "classic")
	is.Equal(annotatedGameVariant(&ipc.GameRequest{}), "classic")
}
