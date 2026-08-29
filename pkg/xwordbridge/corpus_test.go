package xwordbridge

// Replaying the real corpus.
//
// The differential in bridge_test.go plays random moves, which covers the rules
// broadly but only ever produces game shapes the generator knows how to make.
// This runs the engine over games that were actually played on Woogles, with
// whatever the last several years put in them.
//
// The corpus is not in the repository -- it is production game data, including
// player nicknames. Point XWORDGAME_CORPUS at a directory holding
// live_games.tsv (uuid, base64 history, game_request, game_end_reason) and the
// test runs; without it the test skips, so CI stays green without prod access.
//
// Export it with:
//
//	./bin/proddb -q -c "\copy (
//	  WITH live AS (SELECT DISTINCT game_uuid FROM game_turns)
//	  SELECT g.uuid, encode(g.history,'base64'), g.game_request::text, g.game_end_reason
//	  FROM live JOIN games g ON g.uuid = live.game_uuid
//	  WHERE g.history IS NOT NULL AND octet_length(g.history) > 0
//	) TO '/path/to/corpus/live_games.tsv'"

import (
	"bufio"
	"encoding/base64"
	"os"
	"path/filepath"
	"sort"
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
	uuid string
	hist *macondopb.GameHistory
}

func loadCorpus(t *testing.T) []corpusGame {
	t.Helper()
	dir := os.Getenv("XWORDGAME_CORPUS")
	if dir == "" {
		t.Skip("set XWORDGAME_CORPUS to a directory containing live_games.tsv")
	}
	f, err := os.Open(filepath.Join(dir, "live_games.tsv"))
	if err != nil {
		t.Skipf("corpus not readable: %v", err)
	}
	defer f.Close()

	var games []corpusGame
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
		games = append(games, corpusGame{uuid: cols[0], hist: hist})
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("reading corpus: %v", err)
	}
	if len(games) == 0 {
		t.Skip("corpus is empty")
	}
	return games
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
		matched, skipped, replayFailed, diverged int
		totalEvents, totalChallenges             int
		reasons                                  = map[string]int{}
		fields                                   = map[string]int{}
		examples                                 = map[string]string{}
	)
	note := func(m map[string]int, key, uuid string) {
		m[key]++
		if _, ok := examples[key]; !ok {
			examples[key] = uuid
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

		divs := comparableDivergences(res.State, theirs)
		if len(divs) == 0 {
			matched++
			continue
		}
		diverged++
		for _, d := range divs {
			note(fields, fieldShape(d.Field), cg.uuid)
		}
	}

	t.Logf("matched %d, diverged %d, replay-failed %d, skipped %d (of %d)",
		matched, diverged, replayFailed, skipped, len(games))
	t.Logf("%d events applied, %d challenges adjudicated", totalEvents, totalChallenges)
	report(t, "replay/setup failures", reasons, examples)
	report(t, "divergent fields", fields, examples)

	if matched == 0 {
		t.Fatal("no game replayed to a matching position")
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

// comparableDivergences drops the one field a macondo history reconstruction
// cannot be held to.
//
// macondo maintains per-player turn counts during live play -- playMove
// increments players[i].turns for placements, passes and exchanges alike -- but
// its reconstruction path, PlayTurn, increments it *only* in the exchange
// branch. A game rebuilt from history therefore reports a turn count equal to
// the number of exchanges in it, which is why the first corpus run showed a
// turns[] divergence in essentially every game with our (correct) count on one
// side and near-zero on the other.
//
// That is a macondo bug of exactly the kind this project exists to remove:
// state that live play maintains and reconstruction silently does not. It is
// not excluded from CompareStates generally -- the differential in
// bridge_test.go drives a *live* macondo game, where the counter is maintained,
// and compares it successfully. It is excluded only here, where the reference
// is a reconstruction.
func comparableDivergences(ours, theirs *xwordgame.State) []Divergence {
	var out []Divergence
	for _, d := range CompareStates(ours, theirs) {
		if strings.HasPrefix(d.Field, "turns[") {
			continue
		}
		out = append(out, d)
	}
	return out
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
