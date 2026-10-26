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
// (entity/sought_game.go): a super variant plays on the 21x21 board with the
// 200-tile English set, any other on the standard board with its lexicon's
// own tiles. No variant means classic. The English set is the 21x21 board's
// only one, so a super variant takes only a lexicon that uses English tiles.
func setAnnotatedGameBoard(lexicon string, rules *ipc.GameRules) error {
	ld, err := entity.LetterDistributionForLexicon(lexicon)
	if err != nil {
		return apiserver.InvalidArg(err.Error())
	}
	switch cwgame.Variant(rules.VariantName) {
	case cwgame.VarClassicSuper, cwgame.VarWordSmogSuper:
		if ld != "english" {
			return apiserver.InvalidArg("ZOMGWords plays with English lexica only")
		}
		rules.BoardLayoutName = board.SuperCrosswordGameLayout
		rules.LetterDistributionName = "english_super"
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
