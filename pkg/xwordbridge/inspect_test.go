package xwordbridge

// A single game, both ways, for when one game in the corpus needs explaining.
//
//	XWORDGAME_GAME=TzeCLa8q go test ./pkg/xwordbridge/ -run TestInspectGame -v
//
// The corpus tests report a game's *shape* -- how many diverged, in which
// fields -- which is the right summary for a hundred thousand games and no help
// at all for the one that is interesting. This prints what each path produced
// for a named game, including the production turns path with the ending taken
// from game_end_reason, so "would this game load correctly today" can be
// answered by looking rather than by reasoning.

import (
	"os"
	"testing"

	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

func TestInspectGame(t *testing.T) {
	want := os.Getenv("XWORDGAME_GAME")
	if want == "" {
		t.Skip("set XWORDGAME_GAME to a corpus game id")
	}
	games := loadCorpus(t)

	var cg *corpusGame
	for i := range games {
		if games[i].uuid == want {
			cg = &games[i]
			break
		}
	}
	if cg == nil {
		t.Fatalf("game %s is not in the corpus", want)
	}

	r, mrules, err := rulesForHistory(t, cg.hist)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("game %s: %d events, lexicon %s, end reason %d, stored play state %s",
		cg.uuid, len(cg.hist.Events), cg.hist.Lexicon, cg.endReason, cg.hist.PlayState)
	for i, e := range cg.hist.Events {
		t.Logf("  evt %2d p%d %-28s rack=%-9q played=%-8q score=%3d cume=%d",
			i, e.PlayerIndex, e.Type, e.Rack, e.PlayedTiles, e.Score, e.Cumulative)
	}

	// 1. The replay used by the corpus test: history in, position out.
	res, err := ReplayHistory(cg.hist, r, nil)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	t.Logf("from history:  scores=%v playState=%v onTurn=%d scoreless=%d",
		res.State.Scores, res.State.PlayState, res.State.OnTurn, res.State.ScorelessTurns)
	t.Logf("               endRack: rackUnknown=%d gameNotOver=%d arithmeticWrong=%d",
		res.EndRackRackUnknown, res.EndRackGameNotOver, res.EndRackArithmeticWrong)

	// 2. The production path: events from the turns table, ending stated from
	// games.game_end_reason rather than inferred from the log.
	tres, err := StateFromTurns(TurnsInput{
		Events:         cg.hist.Events,
		LastKnownRacks: cg.hist.LastKnownRacks,
		GameEnded:      cg.endReason != int(macondopb.PlayState_PLAYING),
	}, r, nil)
	if err != nil {
		t.Fatalf("from turns: %v", err)
	}
	t.Logf("from turns:    scores=%v playState=%v onTurn=%d",
		tres.State.Scores, tres.State.PlayState, tres.State.OnTurn)

	// 3. What macondo makes of the same history, as the reference.
	mcg, err := macondogame.NewFromHistory(cg.hist, mrules, len(cg.hist.Events))
	if err != nil {
		t.Logf("macondo could not load this game: %v", err)
		return
	}
	t.Logf("macondo:       scores=[%d %d] playState=%v onTurn=%d",
		mcg.PointsFor(0), mcg.PointsFor(1), mcg.Playing(), mcg.PlayerOnTurn())

	if divs := CompareStates(tres.State, res.State); len(divs) > 0 {
		t.Errorf("turns path and history path disagree: %s", joinDivs(divs))
	}
	t.Logf("final scores recorded in the log: %v", cg.hist.FinalScores)
}
