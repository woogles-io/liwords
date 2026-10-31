package omgwords

import (
	"fmt"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/cwgame"
	"github.com/woogles-io/liwords/pkg/cwgame/board"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// setAnnotatedGameBoard sets a new annotated game's board and tiles from its
// variant rather than taking the client's, as a seek's are set
// (entity/sought_game.go): a super variant plays on the 21x21 board with its
// lexicon's 200-tile set, any other on the standard board with its lexicon's
// own tiles. No variant means classic. Only the English and Catalan tiles
// have a 200-tile set, so a super variant takes only a lexicon that uses
// one of those.
func setAnnotatedGameBoard(lexicon string, rules *ipc.GameRules) error {
	ld, err := entity.LetterDistributionForLexicon(lexicon)
	if err != nil {
		return apiserver.InvalidArg(err.Error())
	}
	switch cwgame.Variant(rules.VariantName) {
	case cwgame.VarClassicSuper, cwgame.VarWordSmogSuper:
		super, ok := entity.SuperLetterDistribution(ld)
		if !ok {
			return apiserver.InvalidArg("ZOMGWords plays with English and Catalan lexica only")
		}
		rules.BoardLayoutName = board.SuperCrosswordGameLayout
		rules.LetterDistributionName = super
	case "", cwgame.VarClassic, cwgame.VarWordSmog:
		if rules.VariantName == "" {
			rules.VariantName = string(cwgame.VarClassic)
		}
		rules.BoardLayoutName = board.CrosswordGameLayout
		rules.LetterDistributionName = ld
	default:
		return apiserver.InvalidArg(fmt.Sprintf("unknown variant %q", rules.VariantName))
	}
	return nil
}
