package xwordbridge

// The archive contract, held against real games.
//
// Once a finished game is in S3, that object is all a later reconstruction
// gets. These are the fields it has to carry, and this asserts they are there
// across every finished game in the corpus -- which is the evidence the
// production guard in pkg/stores/game/s3.go is calibrated from.

import (
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

// Aborted and cancelled games never produced a score, so they are exempt from
// the FinalScores requirement and only that one.
const (
	endReasonAborted   = 5
	endReasonCancelled = 7
)

func TestArchiveContractHoldsAcrossTheCorpus(t *testing.T) {
	games := loadCorpus(t)

	var finished, endRackEvents int
	problems := map[string]int{}
	examples := map[string]string{}
	note := func(what, uid string) {
		problems[what]++
		if _, seen := examples[what]; !seen {
			examples[what] = uid
		}
	}

	for _, cg := range games {
		// Only finished games are archived, so only they are in scope.
		if cg.endReason == 0 {
			continue
		}
		finished++
		h := cg.hist

		if h.PlayState != macondopb.PlayState_GAME_OVER {
			note("play_state is not GAME_OVER", cg.uuid)
		}
		if len(h.FinalScores) == 0 &&
			cg.endReason != endReasonAborted && cg.endReason != endReasonCancelled {
			note("final_scores is empty", cg.uuid)
		}
		if len(h.LastKnownRacks) != len(h.Players) {
			note("last_known_racks has no entry per player", cg.uuid)
		}
		for _, e := range h.Events {
			if e.Type == macondopb.GameEvent_END_RACK_PTS ||
				e.Type == macondopb.GameEvent_END_RACK_PENALTY {
				endRackEvents++
				if e.Rack == "" {
					note("end-rack event carries no rack", cg.uuid)
				}
			}
		}
	}

	t.Logf("%d finished games, %d end-rack events", finished, endRackEvents)
	for what, n := range problems {
		t.Errorf("%d games: %s (e.g. %s)", n, what, examples[what])
	}
}
