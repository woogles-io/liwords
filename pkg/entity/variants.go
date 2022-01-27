package entity

import (
	"errors"

	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/domino14/macondo/game"
)

// Variants, time controls, etc.

type Variant string
type TimeControl string

const (
	TCRegular    TimeControl = "regular"    // > 14/0
	TCRapid                  = "rapid"      // 6/0 to <= 14/0
	TCBlitz                  = "blitz"      // > 2/0 to < 6/0
	TCUltraBlitz             = "ultrablitz" // 2/0 and under
	TCCorres                 = "corres"
)

const (
	// Cutoffs in seconds for different time controls.
	CutoffUltraBlitz = 2 * 60
	CutoffBlitz      = 6 * 60
	CutoffRapid      = 14 * 60
)

// Calculate "total" time assuming there are 16 turns in a game per player.
const turnsPerGame = 16 // just an estimate.

// TotalTimeEstimate estimates the amount of time this game will take, per side.
func TotalTimeEstimate(gamereq *pb.GameRequest) int32 {
	return gamereq.InitialTimeSeconds +
		(gamereq.MaxOvertimeMinutes * 60) +
		(gamereq.IncrementSeconds * turnsPerGame)
}

func VariantFromGameReq(gamereq *pb.GameRequest) (TimeControl, game.Variant, error) {
	// hardcoded values here; fix sometime
	var timefmt TimeControl

	totalTime := TotalTimeEstimate(gamereq)

	if totalTime <= CutoffUltraBlitz {
		timefmt = TCUltraBlitz
	} else if totalTime <= CutoffBlitz {
		timefmt = TCBlitz
	} else if totalTime <= CutoffRapid {
		timefmt = TCRapid
	} else {
		timefmt = TCRegular
	}
	var variant game.Variant
	switch gamereq.Rules.VariantName {
	case "", string(game.VarClassic):
		variant = game.VarClassic
	case string(game.VarWordSmog):
		variant = game.VarWordSmog
	default:
		return "", "", errors.New("unsupported game type")
	}

	return timefmt, variant, nil
}
