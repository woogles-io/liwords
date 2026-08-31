package xwordbridge

// Replaying the real corpus.
//
// The differential in bridge_test.go plays random moves, which covers the rules
// broadly but only ever produces game shapes the generator knows how to make.
// This runs the engine over games that were actually played on Woogles, with
// whatever the last several years put in them.
//
// The corpus is not in the repository -- it is production game data, including
// player nicknames. Point XWORDGAME_CORPUS at a directory holding any of the
// files below and the test runs; without it the test skips, so CI stays green
// without prod access. Columns are uuid, base64 history, game_request,
// game_end_reason.
//
// live_games.tsv -- everything not yet archived, so mostly games in progress.
// This is the set that matters for the migration, since these are the positions
// that have to survive the cutover.
//
//	./bin/proddb -q -c "\copy (
//	  WITH live AS (SELECT DISTINCT game_uuid FROM game_turns)
//	  SELECT g.uuid, encode(g.history,'base64'), g.game_request::text, g.game_end_reason
//	  FROM live JOIN games g ON g.uuid = live.game_uuid
//	  WHERE g.history IS NOT NULL AND octet_length(g.history) > 0
//	) TO '/path/to/corpus/live_games.tsv'"
//
// finished_games.tsv -- a random sample of completed games, which exercise the
// endings that in-progress games never reach: going out, the six-scoreless
// stalemate, timeouts, resignations. Archiving keeps the history bytea on the
// row, so these come from the database rather than S3.
//
// TABLESAMPLE is what makes the sample cheap: it reads a fraction of the pages,
// where ORDER BY random() would sort all 12.6M rows of a 67GB table. 0.09% is
// roughly 11k rows, trimmed to 10k.
//
//	./bin/proddb -q -c "\copy (
//	  SELECT uuid, encode(history,'base64'), game_request::text, game_end_reason
//	  FROM games TABLESAMPLE SYSTEM (0.09)
//	  WHERE game_end_reason <> 0 AND history IS NOT NULL AND octet_length(history) > 0
//	  LIMIT 10000
//	) TO '/path/to/corpus/finished_games.tsv'"

import (
	"bufio"
	"encoding/base64"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	macondoboard "github.com/domino14/macondo/board"
	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/word-golib/kwg"
	"github.com/domino14/word-golib/tilemapping"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/xwordgame"
)

type corpusGame struct {
	uuid      string
	hist      *macondopb.GameHistory
	endReason int
}

// corpusFiles are the exports the test will read, in order. Any that are
// missing are simply skipped.
var corpusFiles = []string{"live_games.tsv", "finished_games.tsv"}

func loadCorpus(t *testing.T) []corpusGame {
	t.Helper()
	dir := os.Getenv("XWORDGAME_CORPUS")
	if dir == "" {
		t.Skip("set XWORDGAME_CORPUS to a directory containing a corpus export")
	}
	var games []corpusGame
	for _, name := range corpusFiles {
		n := readCorpusFile(t, filepath.Join(dir, name), &games)
		if n > 0 {
			t.Logf("%s: %d games", name, n)
		}
	}
	if len(games) == 0 {
		t.Skipf("no corpus files found in %s", dir)
	}
	return games
}

// readCorpusFile appends the games in one export, returning how many it added.
func readCorpusFile(t *testing.T, path string, games *[]corpusGame) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	before := len(*games)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<24)
	for sc.Scan() {
		cols := strings.Split(sc.Text(), "\t")
		if len(cols) < 2 {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(cols[1], "\\n", ""))
		if err != nil {
			continue
		}
		hist := &macondopb.GameHistory{}
		if err := proto.Unmarshal(raw, hist); err != nil {
			continue
		}
		cg := corpusGame{uuid: cols[0], hist: hist}
		if len(cols) > 3 {
			cg.endReason, _ = strconv.Atoi(strings.TrimSpace(cols[3]))
		}
		*games = append(*games, cg)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return len(*games) - before
}

// rulesForHistory builds both engines' rules from what the history records.
func rulesForHistory(t *testing.T, hist *macondopb.GameHistory) (*xwordgame.Rules, *macondogame.GameRules, error) {
	t.Helper()
	cfg := testConfig(t)

	layoutName := hist.BoardLayout
	if layoutName == "" {
		layoutName = macondoboard.CrosswordGameLayout
	}
	distName := hist.LetterDistribution
	if distName == "" {
		distName = "english"
	}
	variant := hist.Variant
	if variant == "" {
		variant = "classic"
	}

	mrules, err := macondogame.NewBasicGameRules(cfg, hist.Lexicon, layoutName, distName,
		macondogame.CrossScoreOnly, macondogame.Variant(variant))
	if err != nil {
		return nil, nil, err
	}
	layout, err := xwordgame.NamedLayout(layoutName)
	if err != nil {
		return nil, nil, err
	}
	ld, err := tilemapping.GetDistribution(cfg.WGLConfig(), distName)
	if err != nil {
		return nil, nil, err
	}
	cr, err := ChallengeRuleFromMacondo(hist.ChallengeRule)
	if err != nil {
		return nil, nil, err
	}
	r := &xwordgame.Rules{
		Layout:             layout,
		LetterDistribution: ld,
		Variant:            xwordgame.Variant(variant),
		ChallengeRule:      cr,
		ExchangeLimit:      xwordgame.ExchangeLimitForLexicon(hist.Lexicon),
	}
	if hist.Lexicon != "" {
		if k, err := kwg.GetKWG(cfg.WGLConfig(), hist.Lexicon, kwg.WithDistribution(distName)); err == nil {
			r.Lexicon = kwg.Lexicon{KWG: *k}
		}
	}
	return r, mrules, nil
}

// The headline test: replay every real game through xwordgame and compare the
// resulting position against the one macondo reconstructs from the same
// history.
//
// Divergences are collected and summarised by shape rather than failing on the
// first one. With thousands of games, the distribution of failures is the
// diagnosis -- one root cause usually produces a characteristic cluster, and
// stopping at game 3 would hide that.
func TestCorpusReplayMatchesMacondo(t *testing.T) {
	games := loadCorpus(t)
	t.Logf("corpus: %d games", len(games))

	var (
		matched, skipped, replayFailed, diverged, incomplete, endRackWrong int
		totalEvents, totalChallenges                                       int
		reasons                                                            = map[string]int{}
		fields                                                             = map[string]int{}
		mismatches                                                         = map[string]int{}
		examples                                                           = map[string]string{}
	)
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
		r, mrules, err := rulesForHistory(t, cg.hist)
		if err != nil {
			skipped++
			note(reasons, "rules: "+truncate(err.Error()), cg.uuid)
			continue
		}

		res, err := ReplayHistory(cg.hist, r, nil)
		if err != nil {
			replayFailed++
			note(reasons, "replay: "+truncate(err.Error()), cg.uuid)
			continue
		}
		totalEvents += res.Applied
		totalChallenges += res.Challenges
		if res.EndRackRackUnknown > 0 {
			note(mismatches, "rack we charged was not the rack the log names (replay limit)", cg.uuid)
		}
		if res.EndRackArithmeticWrong > 0 {
			endRackWrong++
			note(mismatches, "knew the rack and still got the number wrong (REAL BUG)", cg.uuid)
		}

		// macondo's own reconstruction of the same history, as the reference.
		mg, err := macondogame.NewFromHistory(cg.hist, mrules, len(cg.hist.Events))
		if err != nil {
			skipped++
			note(reasons, "macondo: "+truncate(err.Error()), cg.uuid)
			continue
		}
		theirs, err := StateFromGame(mg)
		if err != nil {
			skipped++
			note(reasons, "convert: "+truncate(err.Error()), cg.uuid)
			continue
		}

		divs := comparableDivergences(res.State, theirs, cg)
		if why := checkPlayState(res.State, cg); why != "" {
			divs = append(divs, Divergence{Field: why})
		}
		if len(divs) == 0 {
			matched++
			continue
		}
		// Some games simply cannot be rebuilt from their events. That is the
		// premise of the whole migration, so those are counted separately
		// rather than treated as engine failures.
		if why := logCannotExpress(cg, res.State); why != "" {
			incomplete++
			note(reasons, why, cg.uuid)
			continue
		}
		diverged++
		for _, d := range divs {
			note(fields, fieldShape(d.Field), cg.uuid)
		}
	}

	t.Logf("matched %d, log-incomplete %d, diverged %d, replay-failed %d, skipped %d (of %d)",
		matched, incomplete, diverged, replayFailed, skipped, len(games))
	t.Logf("%d events applied, %d challenges adjudicated", totalEvents, totalChallenges)
	// The one number the replay cannot hide behind. Live play has no recorded
	// end-of-game adjustment to read, so anywhere our own figure disagrees with
	// the log is a position we would get wrong for real.
	if endRackWrong > 0 {
		t.Errorf("%d games where we knew the rack and still computed the wrong end-of-game adjustment", endRackWrong)
	}
	report(t, "replay/setup failures", reasons, examples)
	report(t, "divergent fields", fields, examples)
	report(t, "end-of-game arithmetic mismatches", mismatches, examples)

	// Small buckets are worth listing outright: with a handful of games the
	// identifiers are more use than the count.
	for _, m := range []map[string]int{reasons, fields, mismatches} {
		for k, n := range m {
			if n <= 40 {
				t.Logf("  ids[%s]: %s", k, strings.Join(ids[k], " "))
			}
		}
	}

	if matched == 0 {
		t.Fatal("no game replayed to a matching position")
	}
	// Every game must either agree or fall into one of the known shapes the log
	// cannot express -- except for a small, named residue.
	//
	// There is no residue: the games that used to be here were end-of-game
	// adjustments we recomputed instead of reading, and they are now read.
	const knownResidue = 0
	if diverged > knownResidue {
		t.Errorf("%d games diverged, more than the %d known unexplained",
			diverged, knownResidue)
	}
	// And those shapes are supposed to be rare. If the share grows, either
	// something regressed or a new kind of unrepresentable game appeared.
	if incomplete*100 > len(games) {
		t.Errorf("%d of %d games are log-incomplete, over the 1%% expected",
			incomplete, len(games))
	}
}

// TestCorpusOneGame dumps the full divergence for a single game. The summary
// above groups by shape, which is right for triage and useless for diagnosis;
// this is the other half.
//
//	XWORDGAME_CORPUS=... XWORDGAME_GAME=<uuid> go test ./pkg/xwordbridge/ -run TestCorpusOneGame -v
func TestCorpusOneGame(t *testing.T) {
	want := os.Getenv("XWORDGAME_GAME")
	if want == "" {
		t.Skip("set XWORDGAME_GAME to a game uuid")
	}
	for _, cg := range loadCorpus(t) {
		if cg.uuid != want {
			continue
		}
		r, mrules, err := rulesForHistory(t, cg.hist)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("lexicon=%s dist=%s layout=%s variant=%s challenge=%s events=%d",
			cg.hist.Lexicon, cg.hist.LetterDistribution, cg.hist.BoardLayout,
			cg.hist.Variant, cg.hist.ChallengeRule, len(cg.hist.Events))
		for i, e := range cg.hist.Events {
			t.Logf("  evt %2d p%d %-28s rack=%-9q played=%q score=%d cume=%d",
				i, e.PlayerIndex, e.Type, e.Rack, e.PlayedTiles, e.Score, e.Cumulative)
		}

		// Replay one event at a time so the tile accounting is visible; a
		// surprising bag count is usually the replay's fault, not the game's.
		for n := 1; n <= len(cg.hist.Events); n++ {
			partial := &macondopb.GameHistory{}
			proto.Merge(partial, cg.hist)
			partial.Events = cg.hist.Events[:n]
			partial.LastKnownRacks = nil
			pr, perr := ReplayHistory(partial, r, nil)
			if perr != nil {
				t.Logf("  after %2d events: %v", n, perr)
				break
			}
			t.Logf("  after %2d events: board=%3d racks=%d/%d bag=%3d total=%3d",
				n, pr.State.TilesOnBoard(), pr.State.RackLen(0), pr.State.RackLen(1),
				pr.State.TilesRemaining(),
				pr.State.TilesOnBoard()+pr.State.RackLen(0)+pr.State.RackLen(1)+pr.State.TilesRemaining())
		}

		res, err := ReplayHistory(cg.hist, r, nil)
		if err != nil {
			t.Fatalf("replay: %v", err)
		}
		mg, err := macondogame.NewFromHistory(cg.hist, mrules, len(cg.hist.Events))
		if err != nil {
			t.Fatal(err)
		}
		theirs, err := StateFromGame(mg)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("applied=%d regenerated=%d challenges=%d", res.Applied, res.Regenerated, res.Challenges)
		t.Logf("ours:    turns=%v scores=%v onTurn=%d play=%s",
			res.State.PlayerTurns, res.State.Scores, res.State.OnTurn, res.State.PlayState)
		t.Logf("macondo: turns=%v scores=%v onTurn=%d play=%s",
			theirs.PlayerTurns, theirs.Scores, theirs.OnTurn, theirs.PlayState)
		for _, d := range CompareStates(res.State, theirs) {
			t.Logf("  DIVERGE %s", d)
		}
		return
	}
	t.Fatalf("game %s not in corpus", want)
}

// moveDrivenEnding reports whether a game's ending is implied by its own moves.
//
// A game that timed out, was resigned, aborted, cancelled, force-forfeited or
// adjudicated had its ending imposed from outside the event log. xwordgame
// models the position the moves produce; those endings live in liwords'
// GameEndReason, which is the correct division -- so a play state derived from
// the log genuinely should not match one stamped on by an external ruling.
func moveDrivenEnding(endReason int) bool {
	switch endReason {
	case 0, // NONE, still playing
		2, // STANDARD, someone went out
		3, // CONSECUTIVE_ZEROES
		6: // TRIPLE_CHALLENGE
		return true
	}
	return false
}

// lastMeaningfulEvent returns the type of the last event that moves a game
// along, ignoring the end-of-game bookkeeping macondo appends.
func lastMeaningfulEvent(hist *macondopb.GameHistory) macondopb.GameEvent_Type {
	for i := len(hist.Events) - 1; i >= 0; i-- {
		if t := hist.Events[i].Type; !isDerivedEvent(t) {
			return t
		}
	}
	return macondopb.GameEvent_TILE_PLACEMENT_MOVE
}

// comparableDivergences drops the fields a macondo *history reconstruction*
// cannot be held to. Each exclusion is narrow and has a reason; none of them
// apply to the live-game differential in bridge_test.go, which drives a real
// macondo game and compares every field including these.
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
//   - onTurn, when the last event was not a move: PlayToTurn advances the
//     player on turn after every event it processes, including a time penalty,
//     which is not a turn -- the penalised player still owes a move.
//
//   - lastWordsFormed once the game is over. Nothing can be challenged in a
//     finished game, so we clear it; macondo ends a game in PlayToTurn without
//     clearing, and reports a challengeable play in a position where no
//     challenge is possible. Same bug as the pass/exchange case above.
//
//   - onTurn and scorelessTurns once the game is over. Neither means anything
//     in a finished position -- nobody is on turn and no further turn can be
//     scoreless -- and macondo's reconstruction disagrees with its own live
//     play on both: PlayToTurn advances the player after the end-of-game events
//     it appends, and PlayTurn increments the scoreless counter on the final
//     pass where playMove deliberately does not.
//
//   - playState, always. macondo's reconstruction never runs end-of-game logic
//     -- PlayTurn says so outright, relying on the end-rack events to carry the
//     score changes -- so a game that ended on six scoreless turns comes back as
//     PLAYING. That is the May 2026 bug almost exactly. Rather than compare
//     against a reference that is known wrong here, the play state is checked
//     directly against the database's game_end_reason, which is ground truth;
//     see checkPlayState.
//
// The first three are macondo reconstruction bugs, all of the same shape as the
// one that caused the May 2026 incident: state that live play maintains and
// rebuilding silently does not.
func comparableDivergences(ours, theirs *xwordgame.State, cg corpusGame) []Divergence {
	lastEvt := lastMeaningfulEvent(cg.hist)
	var out []Divergence
	for _, d := range CompareStates(ours, theirs) {
		switch {
		case strings.HasPrefix(d.Field, "turns["):
			continue
		case d.Field == "lastWordsFormed" && lastEvt != macondopb.GameEvent_TILE_PLACEMENT_MOVE:
			continue
		case d.Field == "onTurn" && lastEvt == macondopb.GameEvent_TIME_PENALTY:
			continue
		case d.Field == "playState":
			continue
		case (d.Field == "onTurn" || d.Field == "scorelessTurns" ||
			d.Field == "lastWordsFormed") && ours.PlayState == xwordgame.GameOver:
			continue
		}
		out = append(out, d)
	}
	return out
}

// checkPlayState holds the replayed play state against the database rather than
// against macondo, and returns a description of the mismatch or "".
//
// This is the assertion the whole project is about: a game that ended must come
// back ended, and a game still in progress must not come back finished. Getting
// it backwards for 944 correspondence games is what started all this.
//
// Only move-driven endings are checked. A timeout, resignation, abort,
// cancellation, forfeit or adjudication is imposed from outside the event log,
// so a position replayed from moves alone genuinely cannot know about it -- that
// is what GameEndReason is for.
func checkPlayState(ours *xwordgame.State, cg corpusGame) string {
	if !moveDrivenEnding(cg.endReason) {
		return ""
	}
	over := ours.PlayState == xwordgame.GameOver
	switch {
	case cg.endReason == 0 && over:
		return "playState: finished a game the database says is still in progress"
	case cg.endReason != 0 && !over:
		return "playState: did not finish a game the database says ended (reason " +
			strconv.Itoa(cg.endReason) + ")"
	}
	return ""
}

// logCannotExpress names the shapes of game whose ending is simply not in the
// event log, returning "" for anything else.
//
// These are not engine failures. They are the argument for the migration: a
// position that cannot be rebuilt from the events is a position that has to be
// stored, and inferring one anyway is what corrupted 944 games.
func logCannotExpress(cg corpusGame, ours *xwordgame.State) string {
	events := cg.hist.Events

	// A triple challenge that the challenger lost writes nothing at all.
	// macondo's ChallengeEvent sets the winner and ends the game without
	// appending an event, so the log of such a game is indistinguishable from
	// one that is simply still in progress.
	if cg.endReason == 6 {
		for _, e := range events {
			if isChallengeOutcome(e.Type) {
				return ""
			}
		}
		return "log-incomplete: triple challenge leaves no event"
	}

	// A player went out under a challenge rule, so the position is correctly
	// waiting for the opponent to pass or challenge -- but no such event was
	// ever recorded, and the game is marked finished.
	if cg.endReason == 2 && ours.PlayState == xwordgame.WaitingForFinalPass {
		return "log-incomplete: went out with no final pass recorded"
	}

	// The stalemate was triggered by an exchange, and the tiles drawn in it are
	// revealed only by the end-rack event that follows -- too late to charge the
	// penalty against them. Every other stalemate is handled by reading those
	// racks ahead of the triggering move.
	if cg.endReason == 3 {
		for i, e := range events {
			if e.Type != macondopb.GameEvent_EXCHANGE {
				continue
			}
			if i+1 < len(events) && events[i+1].Type == macondopb.GameEvent_END_RACK_PENALTY {
				return "log-incomplete: stalemate on an exchange hides the drawn tiles"
			}
		}
	}
	return ""
}

// fieldShape collapses board[7,9] to board[] and rack[0] to rack[] so the
// summary groups by kind rather than listing every coordinate.
func fieldShape(f string) string {
	if i := strings.IndexByte(f, '['); i >= 0 {
		return f[:i] + "[]"
	}
	return f
}

func truncate(s string) string {
	if len(s) > 90 {
		return s[:90]
	}
	return s
}

func report(t *testing.T, title string, counts map[string]int, examples map[string]string) {
	t.Helper()
	if len(counts) == 0 {
		return
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return counts[keys[i]] > counts[keys[j]] })
	t.Logf("--- %s ---", title)
	for _, k := range keys {
		t.Logf("%6d  %s  (e.g. %s)", counts[k], k, examples[k])
	}
}
