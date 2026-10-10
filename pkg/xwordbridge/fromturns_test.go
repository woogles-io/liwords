package xwordbridge

// The corpus, driven through the production shape.
//
// TestCorpusReplayMatchesMacondo feeds ReplayHistory a GameHistory. Production
// will feed StateFromTurns a list of game_turns rows plus three columns off the
// games row. Those are not the same input, and the difference is exactly where
// the May 2026 bug lived -- so it is worth verifying the shape that will
// actually run, not the one that is convenient to test.

import (
	"testing"

	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/xwordgame"
)

// asTurnsInput takes a corpus game apart into the pieces production has: the
// events as they would sit in game_turns, and the columns off the games row.
// Nothing else from the GameHistory is allowed through.
func asTurnsInput(t *testing.T, cg corpusGame) TurnsInput {
	t.Helper()
	rows := make([][]byte, len(cg.hist.Events))
	for i, e := range cg.hist.Events {
		raw, err := protojson.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		rows[i] = raw
	}
	events, err := DecodeTurns(rows)
	if err != nil {
		t.Fatal(err)
	}
	return TurnsInput{
		Events:         events,
		LastKnownRacks: cg.hist.LastKnownRacks,
		GameEnded:      cg.endReason != 0,
	}
}

// Reconstructing from turns must land in the same position as reconstructing
// from the history, on every game in the corpus.
func TestCorpusFromTurnsMatchesFromHistory(t *testing.T) {
	games := loadCorpus(t)
	t.Logf("corpus: %d games", len(games))

	var matched, skipped, failed, diverged int
	reasons := map[string]int{}
	fields := map[string]int{}
	examples := map[string]string{}
	ids := map[string][]string{}
	note := func(m map[string]int, key, uuid string) {
		m[key]++
		if _, ok := examples[key]; !ok {
			examples[key] = uuid
		}
		if len(ids[key]) < 40 {
			ids[key] = append(ids[key], uuid)
		}
	}

	for _, cg := range games {
		r, _, err := rulesForHistory(t, cg.hist)
		if err != nil {
			skipped++
			continue
		}
		want, err := ReplayHistory(cg.hist, r, nil)
		if err != nil {
			skipped++
			continue
		}
		got, err := StateFromTurns(asTurnsInput(t, cg), r, nil)
		if err != nil {
			failed++
			note(reasons, "from-turns: "+truncate(err.Error()), cg.uuid)
			continue
		}
		divs := CompareStates(got.State, want.State)
		if len(divs) == 0 {
			matched++
			continue
		}
		diverged++
		for _, d := range divs {
			note(fields, fieldShape(d.Field), cg.uuid)
		}
	}

	t.Logf("matched %d, diverged %d, failed %d, skipped %d (of %d)",
		matched, diverged, failed, skipped, len(games))
	report(t, "from-turns failures", reasons, examples)
	report(t, "divergent fields", fields, examples)
	for _, m := range []map[string]int{reasons, fields} {
		for k, n := range m {
			if n <= 40 {
				t.Logf("  ids[%s]: %v", k, ids[k])
			}
		}
	}

	if diverged > 0 || failed > 0 {
		t.Errorf("%d diverged and %d failed; the turns path must match the history path exactly",
			diverged, failed)
	}
}

// The specific regression. A game waiting for its final pass must come back
// waiting, not playing -- that inversion is what ended 944 games early.
func TestFromTurnsKeepsWaitingForFinalPass(t *testing.T) {
	is := is.New(t)
	games := loadCorpus(t)

	checked := 0
	for _, cg := range games {
		if cg.endReason != 0 {
			continue // only games still in progress can be waiting
		}
		r, _, err := rulesForHistory(t, cg.hist)
		if err != nil {
			continue
		}
		want, err := ReplayHistory(cg.hist, r, nil)
		if err != nil || want.State.PlayState != xwordgame.WaitingForFinalPass {
			continue
		}
		got, err := StateFromTurns(asTurnsInput(t, cg), r, nil)
		is.NoErr(err)
		if got.State.PlayState != xwordgame.WaitingForFinalPass {
			t.Fatalf("%s: reconstructed as %s, should be WAITING_FOR_FINAL_PASS",
				cg.uuid, got.State.PlayState)
		}
		checked++
	}
	t.Logf("%d games waiting for a final pass, all preserved", checked)
}

// games.player_on_turn is carried as a cross-check on the derivation. If it
// fires in production the derivation is wrong; if it never fires, the column is
// redundant and can be dropped.
//
// The stand-in for the column here is macondo's reconstruction, since the
// corpus export does not carry it -- and that stand-in is known to be wrong in
// one specific way: PlayToTurn advances the player on turn after every event it
// processes, including a time penalty, which is not a turn. So disagreement is
// only tolerated on games that are already over, where whose turn it is has no
// meaning.
func TestFromTurnsAgreesWithStoredPlayerOnTurn(t *testing.T) {
	games := loadCorpus(t)
	disagreed, checked := 0, 0

	for _, cg := range games {
		r, mrules, err := rulesForHistory(t, cg.hist)
		if err != nil {
			continue
		}
		// macondo's reconstruction stands in for the games row here, since the
		// corpus export does not carry player_on_turn.
		mg, err := macondogame.NewFromHistory(cg.hist, mrules, len(cg.hist.Events))
		if err != nil {
			continue
		}
		in := asTurnsInput(t, cg)
		in.PlayerOnTurn = mg.PlayerOnTurn()
		in.PlayerOnTurnKnown = true

		got, err := StateFromTurns(in, r, nil)
		if err != nil {
			continue
		}
		checked++
		if !got.OnTurnDisagreed {
			continue
		}
		disagreed++
		t.Logf("  %s: derived %d, stored %d, last event %s, playState %s",
			cg.uuid, got.State.OnTurn, got.StoredOnTurn,
			cg.hist.Events[len(cg.hist.Events)-1].Type, got.State.PlayState)
		if got.State.PlayState != xwordgame.GameOver {
			t.Errorf("%s: player-on-turn disagrees on a game still in progress", cg.uuid)
		}
	}
	t.Logf("%d games checked, %d disagreed on player-on-turn (all on finished games)",
		checked, disagreed)
}

// Both encodings have to be readable while the two writes overlap.
func TestDecodeTurnAcceptsEitherEncoding(t *testing.T) {
	is := is.New(t)
	evt := &macondopb.GameEvent{
		Type:        macondopb.GameEvent_TILE_PLACEMENT_MOVE,
		PlayerIndex: 1,
		Rack:        "AEIOUST",
		PlayedTiles: "HELLO",
		Score:       24,
		Cumulative:  124,
		Row:         7,
		Column:      3,
	}

	asJSON, err := protojson.Marshal(evt)
	is.NoErr(err)
	fromJSON, err := DecodeTurn(asJSON)
	is.NoErr(err)
	is.True(proto.Equal(fromJSON, evt))

	asBin, err := proto.Marshal(evt)
	is.NoErr(err)
	fromBin, err := DecodeTurn(asBin)
	is.NoErr(err)
	is.True(proto.Equal(fromBin, evt))

	// Garbage is refused rather than silently decoded as an empty event.
	_, err = DecodeTurn([]byte("not a turn"))
	is.True(err != nil)
}

// A zero-valued event marshals to binary as zero bytes, which must not be
// mistaken for JSON or for a decode failure.
func TestDecodeTurnHandlesEmptyBinary(t *testing.T) {
	is := is.New(t)
	evt, err := DecodeTurn(nil)
	is.NoErr(err)
	is.Equal(evt.Type, macondopb.GameEvent_TILE_PLACEMENT_MOVE) // the zero value
}
