package xwordbridge

// What a macondo history reconstruction cannot be held to.
//
// Comparing a replayed position against a macondo game built with
// NewFromHistory turns up a handful of fields where macondo's *reconstruction*
// disagrees with macondo's own *live play*. Every one of them is state that
// live play maintains and rebuilding silently does not -- the same shape as the
// bug that caused the May 2026 incident, which is why they are listed here
// individually with a reason each rather than filtered by a blanket rule.
//
// None of these apply to a differential against a live macondo game, which is
// compared field for field with nothing excluded.

import (
	"strings"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/woogles-io/liwords/pkg/xwordgame"
)

// LastMeaningfulEvent is the type of the last event that is a player's move,
// which is what several of the quirks below depend on.
func LastMeaningfulEvent(events []*macondopb.GameEvent) macondopb.GameEvent_Type {
	for i := len(events) - 1; i >= 0; i-- {
		if t := events[i].Type; !isDerivedEvent(t) {
			return t
		}
	}
	return macondopb.GameEvent_TILE_PLACEMENT_MOVE
}

// ReconstructionOpts describes how much of the macondo side to trust.
type ReconstructionOpts struct {
	// LastEvent is the last move in the log; LastMeaningfulEvent computes it.
	LastEvent macondopb.GameEvent_Type

	// PlayStateReliable says the macondo game's play state was set from a
	// stored value rather than derived by the reconstruction.
	//
	// This distinction decides whether the single most important field in the
	// comparison is checked or skipped, so it is required to be stated.
	// PlayTurn never runs end-of-game logic -- it says so outright, relying on
	// the end-rack events to carry the score changes -- so a game that ended on
	// six scoreless turns comes back PLAYING. That is the May 2026 bug almost
	// exactly, and comparing against it would be comparing against a known
	// wrong answer. But liwords' own load path immediately overwrites it with
	// the stored play state (SetPlaying in DBStore.Get), and *that* value is
	// ground truth worth diffing against.
	PlayStateReliable bool
}

// DivergencesVsReconstruction diffs two positions and drops the fields the
// macondo side cannot be held to.
//
// The exclusions:
//
//   - turns[]: macondo maintains per-player turn counts during live play --
//     playMove increments players[i].turns for placements, passes and exchanges
//     alike -- but PlayTurn, the reconstruction path, increments it only in the
//     exchange branch. A rebuilt game reports a turn count equal to its number
//     of exchanges.
//
//   - lastWordsFormed, unless the last move was a tile placement: macondo
//     clears this field on a phony return, a challenge bonus and game start,
//     but never on a pass or an exchange, in either the live or the
//     reconstruction path. So after a pass it still believes the play before it
//     is challengeable -- and ChallengeEvent's only guard is that the list is
//     non-empty, so it would adjudicate a challenge against a play from two
//     turns ago. We clear it, which is what the rules say: only the
//     immediately preceding play can be challenged.
//
//   - onTurn, when the last event was a time penalty: PlayToTurn advances the
//     player on turn after every event it processes, including a penalty, which
//     is not a turn -- the penalised player still owes a move.
//
//   - onTurn, scorelessTurns and lastWordsFormed once the game is over. None
//     mean anything in a finished position, and macondo's reconstruction
//     disagrees with its own live play on all three.
//
//   - playState, unless the caller says otherwise. See
//     ReconstructionOpts.PlayStateReliable.
func DivergencesVsReconstruction(ours, theirs *xwordgame.State, o ReconstructionOpts) []Divergence {
	var out []Divergence
	for _, d := range CompareStates(ours, theirs) {
		switch {
		case strings.HasPrefix(d.Field, "turns["):
			continue
		case d.Field == "lastWordsFormed" && o.LastEvent != macondopb.GameEvent_TILE_PLACEMENT_MOVE:
			continue
		case d.Field == "onTurn" && o.LastEvent == macondopb.GameEvent_TIME_PENALTY:
			continue
		case d.Field == "playState" && !o.PlayStateReliable:
			continue
		case (d.Field == "onTurn" || d.Field == "scorelessTurns" ||
			d.Field == "lastWordsFormed") && ours.PlayState == xwordgame.GameOver:
			continue
		}
		out = append(out, d)
	}
	return out
}
