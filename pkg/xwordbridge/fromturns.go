package xwordbridge

// Reconstructing a live position from the turns table.
//
// This is the production entry point that `pkg/stores/game` calls, and it is
// the replacement for the `buildHistoryFromTurns` that was commented out at
// db.go:193-216 after the May 2026 incident.
//
// The difference from that function is the whole point. It assembled a
// GameHistory and let the referee infer the rest, which meant PlayState was
// left at proto3's zero value -- PLAYING -- and silently overwrote
// WAITING_FOR_FINAL_PASS on games that were waiting for a final pass. Here the
// ending is an explicit input, taken from the games row, and there is no path
// that leaves it to a default.
//
// # What the event log does not carry
//
// Three things, all of which are columns on `games` that are already written
// on every save:
//
//   - the racks, from last_known_racks. An event records the mover's rack
//     *before* their play and never the tiles they drew, so the log alone
//     cannot say what either player is holding now. That column was added in
//     Phase 3a precisely to close this gap.
//   - whether the game is over, from game_end_reason. Several endings leave no
//     event at all: a triple challenge the challenger lost writes nothing, and
//     a game can end by timeout, resignation, abort or adjudication with the
//     ruling recorded only on the games row.
//   - the rules, from game_request.
//
// Everything else -- board, scores, bingos, per-player turn counts, the
// scoreless-turn counter, the bag by conservation, and the words available to
// challenge -- is computed from the events.

import (
	"fmt"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/xwordgame"
)

// TurnsInput is one game's event log plus the state the log cannot carry.
type TurnsInput struct {
	// Events, in turn order.
	Events []*macondopb.GameEvent

	// LastKnownRacks is games.last_known_racks: what each player is holding
	// now, including tiles drawn after their most recent move.
	LastKnownRacks []string

	// GameEnded is games.game_end_reason != NONE. It is a bool rather than the
	// reason itself because the position only cares *that* the game ended; why
	// it ended is liwords' business and lives on the games row.
	GameEnded bool

	// PlayerOnTurn is games.player_on_turn, used as a cross-check rather than
	// as the answer -- see TurnsResult.OnTurnDisagreed.
	PlayerOnTurn int
	// PlayerOnTurnKnown distinguishes "player 0" from "column not set", since
	// zero is a real player index.
	PlayerOnTurnKnown bool
}

// TurnsResult is a ReplayResult plus what the cross-checks found.
type TurnsResult struct {
	*ReplayResult

	// OnTurnDisagreed reports that the replay derived a different player on
	// turn than games.player_on_turn says.
	//
	// The replay's answer is the one returned. The column is kept as a check
	// on the derivation, not as a crutch for it: if this stays at zero across
	// production traffic, the column is redundant and can go.
	OnTurnDisagreed bool
	// StoredOnTurn is what the column said, when it disagreed.
	StoredOnTurn int
}

// StateFromTurns reconstructs the current position of a game from its event
// log and the columns that accompany it.
//
// Passing a nil Rand uses a deterministic source; replacement draws are
// overwritten by the recorded racks, so their contents never matter.
// A game with no events is a valid input -- it was created and never played --
// and reconstructs to the opening position.
//
// The caller must not reach here for a game whose turns have been deleted,
// because that is indistinguishable from a game with no moves and would
// silently produce an empty board. It cannot happen through the intended call
// site: CommitArchival deletes turns and sets history_s3_key in one
// transaction, and archival only ever runs on a finished game, so
// game_end_reason == NONE implies the turns are still there.
func StateFromTurns(in TurnsInput, r *xwordgame.Rules, rng xwordgame.Rand) (*TurnsResult, error) {
	hist := &macondopb.GameHistory{
		Events:         in.Events,
		LastKnownRacks: in.LastKnownRacks,
	}
	// The ending is stated, never inferred. This single assignment is the fix
	// for the May 2026 incident: the old code left PlayState at its zero value
	// and that value happens to mean PLAYING.
	if in.GameEnded {
		hist.PlayState = macondopb.PlayState_GAME_OVER
	} else {
		hist.PlayState = macondopb.PlayState_PLAYING
	}

	res, err := ReplayHistory(hist, r, rng)
	if err != nil {
		return nil, err
	}

	out := &TurnsResult{ReplayResult: res}
	if in.PlayerOnTurnKnown && int(res.State.OnTurn) != in.PlayerOnTurn {
		out.OnTurnDisagreed = true
		out.StoredOnTurn = in.PlayerOnTurn
	}
	return out, nil
}

// DecodeTurn reads one game_turns row.
//
// Rows are protojson today and binary proto after the format change; both
// forms coexist while the two writes overlap, so this accepts either. A
// protojson document always begins with '{' after any leading whitespace,
// which binary proto for this message never does -- field 1 would encode as
// 0x0a, and no field number produces 0x7b in a tag byte position.
func DecodeTurn(raw []byte) (*macondopb.GameEvent, error) {
	evt := &macondopb.GameEvent{}
	if isJSON(raw) {
		if err := protojson.Unmarshal(raw, evt); err != nil {
			return nil, fmt.Errorf("xwordbridge: decoding turn as protojson: %w", err)
		}
		return evt, nil
	}
	if err := proto.Unmarshal(raw, evt); err != nil {
		return nil, fmt.Errorf("xwordbridge: decoding turn as proto: %w", err)
	}
	return evt, nil
}

func isJSON(raw []byte) bool {
	for _, b := range raw {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}

// DecodeTurns reads a whole game's rows in order.
func DecodeTurns(rows [][]byte) ([]*macondopb.GameEvent, error) {
	events := make([]*macondopb.GameEvent, 0, len(rows))
	for i, raw := range rows {
		evt, err := DecodeTurn(raw)
		if err != nil {
			return nil, fmt.Errorf("turn %d: %w", i, err)
		}
		events = append(events, evt)
	}
	return events, nil
}
