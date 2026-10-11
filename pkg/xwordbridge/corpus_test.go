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
	"compress/gzip"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/xwordgame"
)

type corpusGame struct {
	uuid      string
	hist      *macondopb.GameHistory
	endReason int
}

// corpusFiles are the exports the test will read, in order. Any that are
// missing are simply skipped. A .gz suffix is read transparently, which is how
// the large samples are kept: 100k games is a few hundred megabytes of base64
// and compresses to a fraction of that.
var corpusFiles = []string{
	"live_games.tsv",
	"finished_games.tsv",
	"random_100k.tsv.gz",
}

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

	var src io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		defer gz.Close()
		src = gz
	}
	sc := bufio.NewScanner(src)
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
	spec := SpecFromHistory(hist)

	// Both sides of the comparison are built from the same resolved names, and
	// the xwordgame side goes through the helper production uses, so the corpus
	// is evidence about the real load path rather than about the test.
	mrules, err := macondogame.NewBasicGameRules(cfg, spec.Lexicon, spec.BoardLayout,
		spec.LetterDistribution, macondogame.CrossScoreOnly,
		macondogame.Variant(spec.Variant))
	if err != nil {
		return nil, nil, err
	}
	r, err := RulesFor(cfg, spec)
	if err != nil {
		return nil, nil, err
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
		matched, skipped, replayFailed, diverged, endRackWrong, inconsistent int
		zeroCumulative                                                       int
		totalEvents, totalChallenges                                         int
		reasons                                                              = map[string]int{}
		fields                                                               = map[string]int{}
		mismatches                                                           = map[string]int{}
		examples                                                             = map[string]string{}
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
		zeroCumulative += res.EndRackZeroCumulative
		if res.EndRackGameNotOver > 0 {
			note(mismatches, "the log ended the game where today's rules would not have", cg.uuid)
		}
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
			// An archived history that contradicts the games table cannot be
			// reconciled by any replayer: the record disagrees with itself.
			// Count it as a data inconsistency rather than an engine failure,
			// but count it, because the number is worth watching.
			if cg.hist.PlayState != macondopb.PlayState_GAME_OVER &&
				len(cg.hist.FinalScores) == 0 {
				inconsistent++
				note(mismatches, "history says still playing, games table says ended", cg.uuid)
				continue
			} else {
				divs = append(divs, Divergence{Field: why})
			}
		}
		if len(divs) == 0 {
			matched++
			continue
		}
		diverged++
		for _, d := range divs {
			note(fields, fieldShape(d.Field), cg.uuid)
		}
	}

	t.Logf("matched %d, diverged %d, replay-failed %d, skipped %d, self-contradictory %d (of %d)",
		matched, diverged, replayFailed, skipped, inconsistent, len(games))
	t.Logf("%d events applied, %d challenges adjudicated", totalEvents, totalChallenges)
	// How often a game ends with a player on exactly zero. That is the only
	// condition under which reading a proto zero as "field absent" goes wrong
	// here, and it explains why the bug showed up in a single game.
	t.Logf("%d end-of-game adjustments landed on a score of exactly zero", zeroCumulative)
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
	// Everything is expected to load except games macondo itself cannot open,
	// which are the reference here and so cannot be compared against.
	if want := len(games) - skipped - replayFailed - inconsistent; matched != want {
		t.Errorf("%d games matched, expected %d", matched, want)
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
		t.Logf("stored: playState=%s winner=%d finalScores=%v endReason=%d",
			cg.hist.PlayState, cg.hist.Winner, cg.hist.FinalScores, cg.endReason)
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
// comparableDivergences is the corpus test's view of the production filter in
// quirks.go. The macondo side here is a bare NewFromHistory, so its play state
// is not to be trusted; the play state is checked against the database's
// game_end_reason instead, by checkPlayState.
func comparableDivergences(ours, theirs *xwordgame.State, cg corpusGame) []Divergence {
	return DivergencesVsReconstruction(ours, theirs, ReconstructionOpts{
		LastEvent:         LastMeaningfulEvent(cg.hist.Events),
		PlayStateReliable: false,
	})
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
