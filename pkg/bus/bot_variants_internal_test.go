package bus

import (
	"testing"

	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestBotsDeclineSuperSet(t *testing.T) {
	is := is.New(t)
	for _, c := range []struct {
		variant, lexicon string
		declined         bool
	}{
		{"classic_super", "CSW24", false},
		{"wordsmog_super", "NWL23", false},
		{"classic_super", "DISC2", true},
		{"wordsmog_super", "DISC2", true},
		{"classic", "DISC2", false},
		{"wordsmog", "RD29", false},
	} {
		err := botsDeclineSuperSet(&pb.GameRules{VariantName: c.variant}, c.lexicon)
		is.Equal(err != nil, c.declined)
		if c.declined {
			is.Equal(err.Error(), "bots play this variant with English lexica only")
		}
	}
}
