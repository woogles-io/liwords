package xwordbridge

import (
	"encoding/binary"
	"math/rand/v2"
	"os"
	"strings"
	"testing"

	macondoboard "github.com/domino14/macondo/board"
	macondoconfig "github.com/domino14/macondo/config"
	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	macondomove "github.com/domino14/macondo/move"
	"github.com/domino14/word-golib/kwg"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
	"github.com/woogles-io/liwords/pkg/xwordgame"
)

func TestMain(m *testing.M) {
	// macondo logs at info on every move, which buries failures.
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	os.Exit(m.Run())
}

func testConfig(t *testing.T) *macondoconfig.Config {
	t.Helper()
	cfg := macondoconfig.DefaultConfig()
	if p := os.Getenv("MACONDO_DATA_PATH"); p != "" {
		cfg.Set(macondoconfig.ConfigDataPath, p)
	}
	return cfg
}

// parityConfig is one board/distribution/lexicon/variant combination to run the
// differential against.
//
// Woogles is not an english-only 15x15 server: it runs 21x21 super boards with
// 200-tile bags, nine letter distributions, and the wordsmog variant. A
// differential that has only ever seen classic english CSW21 cannot support a
// claim about any of them.
type parityConfig struct {
	name          string
	macondoLayout string
	xwordLayout   string
	dist          string
	lexicon       string
	variant       xwordgame.Variant
	rule          macondopb.ChallengeRule
	seeds         int
	minPlays      int
	minRejected   int
}

// parityConfigs covers every distribution shipped in data/letterdistributions,
// both board sizes, and both variants.
//
// The minPlays/minRejected floors are calibrated from observed runs with
// margin. They exist so a configuration that silently stops exercising anything
// -- as the VOID case originally did, landing zero plays and zero rejections --
// fails instead of passing quietly.
//
// Most run under DOUBLE, where the lexicon is not consulted at play time, so
// random plays actually land and the state machine gets exercised. The VOID
// entries are the opposite: almost nothing lands, and what they prove is that
// we refuse exactly the plays macondo refuses -- including under wordsmog,
// where validity means "is an anagram of a word" rather than "is a word".
var parityConfigs = []parityConfig{
	{name: "classic-english", macondoLayout: macondoboard.CrosswordGameLayout,
		xwordLayout: xwordgame.CrosswordGameLayout, dist: "english", lexicon: "CSW21",
		variant: xwordgame.VarClassic, rule: macondopb.ChallengeRule_DOUBLE,
		seeds: 30, minPlays: 200},
	{name: "classic-english-nwl", macondoLayout: macondoboard.CrosswordGameLayout,
		xwordLayout: xwordgame.CrosswordGameLayout, dist: "english", lexicon: "NWL23",
		variant: xwordgame.VarClassic, rule: macondopb.ChallengeRule_DOUBLE,
		seeds: 10, minPlays: 50},

	// 21x21, 200 tiles. The board is nearly twice the width and the bag twice
	// the size, so racks refill for far longer and games run much deeper.
	{name: "super-english", macondoLayout: macondoboard.SuperCrosswordGameLayout,
		xwordLayout: xwordgame.SuperCrosswordGameLayout, dist: "english_super", lexicon: "CSW21",
		variant: xwordgame.VarClassicSuper, rule: macondopb.ChallengeRule_DOUBLE,
		seeds: 16, minPlays: 100},

	// Distributions with alphabets larger than english's 26 -- norwegian and
	// polish reach tile 32. A blank designated as one of those letters only
	// occurs if the generator knows the real alphabet size, which is why
	// randomPlacementFromRack takes maxLtr rather than assuming 26.
	{name: "french", macondoLayout: macondoboard.CrosswordGameLayout,
		xwordLayout: xwordgame.CrosswordGameLayout, dist: "french", lexicon: "FRA24",
		variant: xwordgame.VarClassic, rule: macondopb.ChallengeRule_DOUBLE,
		seeds: 8, minPlays: 40},
	{name: "german", macondoLayout: macondoboard.CrosswordGameLayout,
		xwordLayout: xwordgame.CrosswordGameLayout, dist: "german", lexicon: "RD29",
		variant: xwordgame.VarClassic, rule: macondopb.ChallengeRule_DOUBLE,
		seeds: 8, minPlays: 40},
	{name: "norwegian", macondoLayout: macondoboard.CrosswordGameLayout,
		xwordLayout: xwordgame.CrosswordGameLayout, dist: "norwegian", lexicon: "NSF25",
		variant: xwordgame.VarClassic, rule: macondopb.ChallengeRule_DOUBLE,
		seeds: 8, minPlays: 40},
	{name: "polish", macondoLayout: macondoboard.CrosswordGameLayout,
		xwordLayout: xwordgame.CrosswordGameLayout, dist: "polish", lexicon: "OSPS52",
		variant: xwordgame.VarClassic, rule: macondopb.ChallengeRule_DOUBLE,
		seeds: 8, minPlays: 40},
	{name: "spanish", macondoLayout: macondoboard.CrosswordGameLayout,
		xwordLayout: xwordgame.CrosswordGameLayout, dist: "spanish", lexicon: "FILE2017",
		variant: xwordgame.VarClassic, rule: macondopb.ChallengeRule_DOUBLE,
		seeds: 8, minPlays: 40},
	{name: "catalan", macondoLayout: macondoboard.CrosswordGameLayout,
		xwordLayout: xwordgame.CrosswordGameLayout, dist: "catalan", lexicon: "DISC2",
		variant: xwordgame.VarClassic, rule: macondopb.ChallengeRule_DOUBLE,
		seeds: 8, minPlays: 40},
	{name: "slovene", macondoLayout: macondoboard.CrosswordGameLayout,
		xwordLayout: xwordgame.CrosswordGameLayout, dist: "slovene", lexicon: "SLV26",
		variant: xwordgame.VarClassic, rule: macondopb.ChallengeRule_DOUBLE,
		seeds: 8, minPlays: 40},

	{name: "void-english", macondoLayout: macondoboard.CrosswordGameLayout,
		xwordLayout: xwordgame.CrosswordGameLayout, dist: "english", lexicon: "CSW21",
		variant: xwordgame.VarClassic, rule: macondopb.ChallengeRule_VOID,
		seeds: 40, minRejected: 200},
	{name: "void-wordsmog", macondoLayout: macondoboard.CrosswordGameLayout,
		xwordLayout: xwordgame.CrosswordGameLayout, dist: "english", lexicon: "CSW21",
		variant: xwordgame.VarWordSmog, rule: macondopb.ChallengeRule_VOID,
		seeds: 30, minRejected: 200},
	{name: "void-super", macondoLayout: macondoboard.SuperCrosswordGameLayout,
		xwordLayout: xwordgame.SuperCrosswordGameLayout, dist: "english_super", lexicon: "CSW21",
		variant: xwordgame.VarClassicSuper, rule: macondopb.ChallengeRule_VOID,
		seeds: 14, minRejected: 50},
}

// maxTile is the highest real tile in a distribution. Blanks are designated as
// some letter when they go on the board, and which letters are reachable
// depends on the alphabet, not on english.
func maxTile(ld *tilemapping.LetterDistribution) tilemapping.MachineLetter {
	var maxLtr tilemapping.MachineLetter
	for ml, ct := range ld.Distribution() {
		if ct > 0 && ml > 0 {
			maxLtr = tilemapping.MachineLetter(ml)
		}
	}
	return maxLtr
}

// newGamePair builds a macondo game and the xwordgame view of it.
// newGamePair builds a macondo game and an equivalent xwordgame state.
//
// seed fixes macondo's bag as well as ours. Without it the test's failure
// messages lie: they report "seed N", but macondo's draws came from the global
// RNG, so re-running with that seed reproduced a different game -- and a
// differential test whose failures cannot be reproduced is very little use. It
// also made the coverage floors below flap, since how many random plays get
// rejected depends on what is on the racks.
func newGamePair(t *testing.T, pc parityConfig, seed int) (*macondogame.Game, *xwordgame.State, *xwordgame.Rules) {
	t.Helper()
	cfg := testConfig(t)
	mrules, err := macondogame.NewBasicGameRules(
		cfg, pc.lexicon, pc.macondoLayout, pc.dist,
		macondogame.CrossScoreOnly, macondogame.Variant(pc.variant))
	if err != nil {
		t.Fatal(err)
	}
	g, err := macondogame.NewGame(mrules, []*macondopb.PlayerInfo{
		{Nickname: "p1", RealName: "Player One"},
		{Nickname: "p2", RealName: "Player Two"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// SeedBag, not Bag().SetRNG: StartGame throws the current bag away and
	// builds a new one, so an RNG set on the bag beforehand is discarded and
	// the draws quietly fall back to the global generator.
	var bagSeed [32]byte
	binary.LittleEndian.PutUint64(bagSeed[:], uint64(seed))
	g.SeedBag(bagSeed)
	g.StartGame()
	g.SetChallengeRule(pc.rule)

	s, err := StateFromGame(g)
	if err != nil {
		t.Fatal(err)
	}

	layout, err := xwordgame.NamedLayout(pc.xwordLayout)
	if err != nil {
		t.Fatal(err)
	}
	cr, err := ChallengeRuleFromMacondo(pc.rule)
	if err != nil {
		t.Fatal(err)
	}
	ld, err := tilemapping.GetDistribution(cfg.WGLConfig(), pc.dist)
	if err != nil {
		t.Fatal(err)
	}
	// Load the lexicon the way macondo does, with the distribution -- the
	// mapping from letters to tiles is distribution-specific.
	k, err := kwg.GetKWG(cfg.WGLConfig(), pc.lexicon, kwg.WithDistribution(pc.dist))
	if err != nil {
		t.Fatal(err)
	}
	return g, s, &xwordgame.Rules{
		Layout:             layout,
		LetterDistribution: ld,
		Lexicon:            kwg.Lexicon{KWG: *k},
		Variant:            pc.variant,
		ChallengeRule:      cr,
	}
}

// englishClassic is the configuration the non-differential tests use.
var englishClassic = parityConfigs[0]

// A freshly started game must translate exactly.
func TestStateFromGameAtStart(t *testing.T) {
	is := is.New(t)
	g, s, r := newGamePair(t, englishClassic, 1)

	is.Equal(s.Dim(), 15)
	is.True(s.IsBoardEmpty())
	is.Equal(s.OnTurn, uint8(g.PlayerOnTurn()))
	is.Equal(s.PlayState, xwordgame.Playing)
	is.Equal(s.RackLen(0), xwordgame.RackTileLimit)
	is.Equal(s.RackLen(1), xwordgame.RackTileLimit)
	is.Equal(s.TilesRemaining(), 100-2*xwordgame.RackTileLimit)
	is.Equal(s.TilesRemaining(), g.Bag().TilesRemaining())
	is.NoErr(s.ValidateTileConservation(r.LetterDistribution))

	divs, err := Compare(s, g)
	is.NoErr(err)
	is.Equal(len(divs), 0)
}

// Every enum value, not just the two the parity test happens to drive. These
// conversions look like they could be casts, and the point of writing them out
// is that a change on either side becomes an error rather than a silent
// mis-mapping -- which only holds if all of them are checked.
func TestEnumConversions(t *testing.T) {
	is := is.New(t)

	for _, tc := range []struct {
		in   macondopb.ChallengeRule
		want xwordgame.ChallengeRule
	}{
		{macondopb.ChallengeRule_VOID, xwordgame.ChallengeRuleVoid},
		{macondopb.ChallengeRule_SINGLE, xwordgame.ChallengeRuleSingle},
		{macondopb.ChallengeRule_DOUBLE, xwordgame.ChallengeRuleDouble},
		{macondopb.ChallengeRule_FIVE_POINT, xwordgame.ChallengeRuleFivePoint},
		{macondopb.ChallengeRule_TEN_POINT, xwordgame.ChallengeRuleTenPoint},
		{macondopb.ChallengeRule_TRIPLE, xwordgame.ChallengeRuleTriple},
	} {
		got, err := ChallengeRuleFromMacondo(tc.in)
		is.NoErr(err)
		is.Equal(got, tc.want)
		// The names have to line up too, or a log line would mislead.
		is.Equal(got.String(), tc.in.String())
	}
	_, err := ChallengeRuleFromMacondo(macondopb.ChallengeRule(99))
	is.True(err != nil)

	for _, tc := range []struct {
		in   macondopb.PlayState
		want xwordgame.PlayState
	}{
		{macondopb.PlayState_PLAYING, xwordgame.Playing},
		{macondopb.PlayState_WAITING_FOR_FINAL_PASS, xwordgame.WaitingForFinalPass},
		{macondopb.PlayState_GAME_OVER, xwordgame.GameOver},
	} {
		got, err := PlayStateFromMacondo(tc.in)
		is.NoErr(err)
		is.Equal(got, tc.want)
		is.Equal(got.String(), tc.in.String())
	}
	_, err = PlayStateFromMacondo(macondopb.PlayState(99))
	is.True(err != nil)
}

// Reading a board mid-transposition would mirror the position, so it is
// refused rather than silently mistranslated.
func TestTransposedBoardIsRefused(t *testing.T) {
	is := is.New(t)
	g, _, _ := newGamePair(t, englishClassic, 1)

	g.Board().Transpose()
	_, err := StateFromGame(g)
	is.True(err != nil)
	t.Log(err)

	g.Board().Transpose()
	_, err = StateFromGame(g)
	is.NoErr(err)
}

func TestCompareReportsEveryDivergence(t *testing.T) {
	is := is.New(t)
	g, s, _ := newGamePair(t, englishClassic, 1)

	// Break several unrelated things at once. A real root cause usually shows
	// up in more than one field, and the set is the diagnosis.
	s.Scores[0] += 7
	s.OnTurn = 1 - s.OnTurn
	s.ScorelessTurns = 4
	s.SetTileAt(0, 0, 5)
	s.Bingos[1]++
	s.PlayerTurns[0] += 2
	// The field the whole exercise is about: a play state that disagrees is
	// exactly the corruption this package exists to detect.
	s.PlayState = xwordgame.WaitingForFinalPass
	is.NoErr(s.SetRack(0, []tilemapping.MachineLetter{1, 2, 3}))
	is.NoErr(s.SetBagCounts(make([]uint8, 30)))

	divs, err := Compare(s, g)
	is.NoErr(err)
	fields := map[string]bool{}
	for _, d := range divs {
		fields[d.Field] = true
		t.Log(d)
	}
	for _, want := range []string{
		"score[0]", "onTurn", "scorelessTurns", "board[0,0]",
		"bingos[1]", "turns[0]", "playState", "rack[0]", "bag",
	} {
		if !fields[want] {
			t.Errorf("Compare did not report %s", want)
		}
	}
}

// The state machine, differentially. Same move into both engines, positions
// compared after every one.
//
// The two draw replacement tiles from their own bags with their own RNGs, so
// rack and bag contents cannot match. Everything else must, and the rack and
// bag *sizes* must -- which is what actually exercises the replenishment rule.
// Racks are resynced from macondo after each move so the next move is played
// from an identical position.
func TestStateMachineParityAgainstMacondo(t *testing.T) {
	for _, pc := range parityConfigs {
		t.Run(pc.name, func(t *testing.T) {
			t.Parallel()
			plays, exchanges, passes, games, endings, rejected := 0, 0, 0, 0, 0, 0

			for seed := range pc.seeds {
				rng := rand.New(rand.NewPCG(uint64(seed), 2024))
				g, s, r := newGamePair(t, pc, seed)
				alph := r.LetterDistribution.TileMapping()
				maxLtr := maxTile(r.LetterDistribution)

				maxTurns := 8 * s.Dim() * s.Dim() / 15
				for turn := 0; turn < maxTurns && s.PlayState != xwordgame.GameOver; turn++ {
					// Probe: ask both engines about a candidate neither has
					// vetted, and require them to agree. The move actually
					// played below is pre-filtered through our own
					// ValidateMove so the game makes progress, which means
					// nothing rejected would otherwise ever reach macondo.
					// Under VOID this is where the lexicon check is compared;
					// under DOUBLE it covers geometric illegality.
					if probe := randomPlacementFromRack(s, rng, maxLtr); probe != nil &&
						s.PlayState == xwordgame.Playing {
						_, ourProbeErr := s.ValidateMove(r, probe)
						mp, err := macondoProbeMove(g, probe, alph)
						if err != nil {
							t.Fatalf("seed %d turn %d: building probe: %v", seed, turn, err)
						}
						_, theirProbeErr := g.ValidateMove(mp)
						if (ourProbeErr == nil) != (theirProbeErr == nil) {
							t.Fatalf("seed %d turn %d: probe legality ours=%v macondo=%v (move %+v)",
								seed, turn, ourProbeErr, theirProbeErr, probe)
						}
						if ourProbeErr != nil {
							rejected++
						}
					}

					m := randomLegalMove(s, r, rng, maxLtr)
					switch m.Type {
					case xwordgame.MoveTypePlay:
						plays++
					case xwordgame.MoveTypeExchange:
						exchanges++
					default:
						passes++
					}

					// macondo needs the score and the leave computed for it.
					// The score comes from macondo's own board so this test
					// does not quietly assert our arithmetic into it.
					mm, err := macondoMove(g, s, m, alph, r.LetterDistribution)
					if err != nil {
						t.Fatalf("seed %d turn %d: building macondo move: %v", seed, turn, err)
					}

					res, ourErr := s.ApplyMove(r, rng, m)
					theirErr := g.PlayMove(mm, true, 0)
					if (ourErr == nil) != (theirErr == nil) {
						t.Fatalf("seed %d turn %d: ours=%v macondo=%v (move %+v)",
							seed, turn, ourErr, theirErr, m)
					}
					if ourErr != nil {
						// Both refused. Neither engine mutates on a rejected
						// move, so the next iteration starts from the same
						// position in both.
						rejected++
						continue
					}
					if m.Type == xwordgame.MoveTypePlay && res.Score != int32(mm.Score()) {
						t.Fatalf("seed %d turn %d: score %d != macondo %d", seed, turn, res.Score, mm.Score())
					}

					theirs, err := StateFromGame(g)
					if err != nil {
						t.Fatalf("seed %d turn %d: %v", seed, turn, err)
					}
					// Sizes are RNG-independent and are what the replenishment
					// rule actually decides.
					for p := range xwordgame.MaxPlayers {
						if s.RackLen(p) != theirs.RackLen(p) {
							t.Fatalf("seed %d turn %d: rack %d has %d tiles, macondo has %d (after %s)",
								seed, turn, p, s.RackLen(p), theirs.RackLen(p), m.Type)
						}
					}
					if s.TilesRemaining() != theirs.TilesRemaining() {
						t.Fatalf("seed %d turn %d: bag has %d, macondo has %d (after %s)",
							seed, turn, s.TilesRemaining(), theirs.TilesRemaining(), m.Type)
					}
					if divs := diffIgnoringDraws(s, theirs, res.GameOver); len(divs) > 0 {
						t.Fatalf("seed %d turn %d after %s: %s", seed, turn, m.Type, joinDivs(divs))
					}
					if res.GameOver {
						// Ending a game adjusts scores by the value of tiles
						// still on racks, and those racks came out of two
						// different bags -- so the adjustments legitimately
						// differ. What both engines must agree on is the score
						// before the adjustment, so add each side's own back.
						ourPre := scoresBeforeEndgameAdjustment(s, res, r.LetterDistribution)
						theirPre := scoresBeforeEndgameAdjustment(theirs, res, r.LetterDistribution)
						if ourPre != theirPre {
							t.Fatalf("seed %d turn %d: pre-adjustment scores %v != macondo %v (final %v vs %v)",
								seed, turn, ourPre, theirPre, s.Scores, theirs.Scores)
						}
						endings++
					}
					if err := s.ValidateTileConservation(r.LetterDistribution); err != nil {
						t.Fatalf("seed %d turn %d: %v", seed, turn, err)
					}
					// Every reachable position must survive the wire format.
					// This is the cheapest place to cover that across all nine
					// distributions and both board sizes at once.
					var decoded xwordgame.State
					if err := decoded.Decode(s.Encode()); err != nil {
						t.Fatalf("seed %d turn %d: decoding our own snapshot: %v", seed, turn, err)
					}
					if !decoded.Equal(s) {
						t.Fatalf("seed %d turn %d: snapshot did not round trip", seed, turn)
					}

					if res.GameOver {
						// Nothing follows, and the scores now hold two
						// different rack adjustments, already accounted for
						// above.
						break
					}
					// Adopt macondo's draws so the next move starts from an
					// identical position.
					for p := range xwordgame.MaxPlayers {
						if err := s.AssignRack(p, theirs.Rack(p)); err != nil {
							t.Fatalf("seed %d turn %d: resync rack %d: %v", seed, turn, p, err)
						}
					}
					// With board and racks equal and every tile accounted for,
					// the bags must now agree exactly -- so this is a full
					// comparison with nothing excluded.
					if divs := CompareStates(s, theirs); len(divs) > 0 {
						t.Fatalf("seed %d turn %d: resync left %s", seed, turn, joinDivs(divs))
					}
				}
				if s.PlayState == xwordgame.GameOver {
					games++
				}
			}
			// Configurations sharing a board size and a bag size report
			// identical play/reject/pass counts, and that is correct rather
			// than a sign the distribution is being ignored: under DOUBLE the
			// lexicon is not consulted at play time, so whether a random
			// placement is legal is purely geometric. French and German have
			// 102 tiles and land on one set of numbers; the 100-tile
			// distributions land on another. What does differ per
			// distribution is the scoring, and that is compared after every
			// single move.
			t.Logf("%s: %d/%d games finished (%d endgame adjustments verified); %d plays, %d rejected, %d exchanges, %d passes",
				pc.name, games, pc.seeds, endings, plays, rejected, exchanges, passes)
			if plays < pc.minPlays {
				t.Errorf("only %d plays landed, wanted at least %d -- this configuration is not measuring what it claims",
					plays, pc.minPlays)
			}
			if rejected < pc.minRejected {
				t.Errorf("only %d plays were rejected, wanted at least %d", rejected, pc.minRejected)
			}
			if games != pc.seeds {
				t.Errorf("%d of %d games did not reach a conclusion", pc.seeds-games, pc.seeds)
			}
			if endings != games {
				t.Errorf("%d games ended but only %d endgame adjustments were verified", games, endings)
			}
		})
	}
}

// diffIgnoringDraws drops the fields that legitimately differ because the two
// engines drew replacement tiles from their own bags with their own RNGs.
//
// On the move that ends a game, scores join that set: the endgame adjustment is
// computed from those racks. The caller compares pre-adjustment scores instead,
// so nothing goes unchecked.
func diffIgnoringDraws(ours, theirs *xwordgame.State, gameOver bool) []Divergence {
	var out []Divergence
	for _, d := range CompareStates(ours, theirs) {
		if strings.HasPrefix(d.Field, "rack[") || d.Field == "bag" {
			continue
		}
		if gameOver && strings.HasPrefix(d.Field, "score[") {
			continue
		}
		out = append(out, d)
	}
	return out
}

// scoresBeforeEndgameAdjustment undoes the rack-based adjustment a finished
// game applied, using the state's own racks. Both engines run the same rule --
// each player loses their own rack on a scoreless stalemate, the player who
// went out gains twice the opponent's -- so applying it in reverse to each side
// separately leaves two numbers that must match.
func scoresBeforeEndgameAdjustment(st *xwordgame.State, res *xwordgame.ApplyResult,
	ld *tilemapping.LetterDistribution) [xwordgame.MaxPlayers]int32 {

	scores := st.Scores
	switch {
	case res.ScorelessPenalties != nil:
		for p := range scores {
			scores[p] += st.RackScoreFor(ld, p)
		}
	case res.EndRackBonusPlayer >= 0:
		out := int(res.EndRackBonusPlayer)
		scores[out] -= 2 * st.RackScoreFor(ld, 1-out)
	}
	return scores
}

func joinDivs(divs []Divergence) string {
	parts := make([]string, len(divs))
	for i, d := range divs {
		parts[i] = d.String()
	}
	return strings.Join(parts, "; ")
}

// macondoMove translates our move into macondo's, computing the score from
// macondo's own board and the leave from macondo's own rack.
func macondoMove(g *macondogame.Game, s *xwordgame.State, m *xwordgame.Move,
	alph *tilemapping.TileMapping, ld *tilemapping.LetterDistribution) (*macondomove.Move, error) {

	rack := g.RackFor(g.PlayerOnTurn()).TilesOn()
	switch m.Type {
	case xwordgame.MoveTypePass:
		return macondomove.NewPassMove(rack, alph), nil

	case xwordgame.MoveTypeExchange:
		leave, err := tilemapping.Leave(rack, m.Tiles, false)
		if err != nil {
			return nil, err
		}
		return macondomove.NewExchangeMove(m.Tiles, leave, alph), nil

	case xwordgame.MoveTypePlay:
		leave, err := tilemapping.Leave(rack, m.Tiles, true)
		if err != nil {
			return nil, err
		}
		score := macondoScore(g, m, ld)
		return macondomove.NewScoringMove(score, m.Tiles, leave, m.Vertical,
			m.TilesPlayed(), alph, m.Row, m.Col), nil
	}
	return nil, nil
}

// macondoProbeMove builds a placement for a legality check only. It skips
// scoring, which is not consulted by ValidateMove and would mean scoring a play
// that may well be off the board.
func macondoProbeMove(g *macondogame.Game, m *xwordgame.Move,
	alph *tilemapping.TileMapping) (*macondomove.Move, error) {

	rack := g.RackFor(g.PlayerOnTurn()).TilesOn()
	leave, err := tilemapping.Leave(rack, m.Tiles, true)
	if err != nil {
		return nil, err
	}
	return macondomove.NewScoringMove(0, m.Tiles, leave, m.Vertical,
		m.TilesPlayed(), alph, m.Row, m.Col), nil
}

// macondoScore scores a play the way game.CreateAndScorePlacementMove does:
// transpose for vertical plays, swap the coordinates, score against the
// opposite cross direction, transpose back.
func macondoScore(g *macondogame.Game, m *xwordgame.Move, ld *tilemapping.LetterDistribution) int {
	b := g.Board()
	row, col := m.Row, m.Col
	crossDir := macondoboard.VerticalDirection
	if m.Vertical {
		crossDir = macondoboard.HorizontalDirection
		row, col = col, row
		b.Transpose()
	}
	score := b.ScoreWord(m.Tiles, row, col, m.TilesPlayed(), crossDir, ld)
	if m.Vertical {
		b.Transpose()
	}
	return score
}

// randomLegalMove prefers a tile play, falls back to an exchange, then a pass.
func randomLegalMove(s *xwordgame.State, r *xwordgame.Rules, rng *rand.Rand,
	maxLtr tilemapping.MachineLetter) *xwordgame.Move {
	if s.PlayState == xwordgame.WaitingForFinalPass {
		return xwordgame.NewPassMove()
	}
	if rng.IntN(12) > 0 {
		for range 12 {
			m := randomPlacementFromRack(s, rng, maxLtr)
			if m == nil {
				break
			}
			if _, err := s.ValidateMove(r, m); err == nil {
				return m
			}
		}
	}
	if rng.IntN(3) == 0 {
		p := int(s.OnTurn)
		if n := s.RackLen(p); n > 0 {
			tiles := append(tilemapping.MachineWord(nil), s.Rack(p)[:1+rng.IntN(n)]...)
			m := xwordgame.NewExchangeMove(tiles)
			if _, err := s.ValidateMove(r, m); err == nil {
				return m
			}
		}
	}
	return xwordgame.NewPassMove()
}

func randomPlacementFromRack(s *xwordgame.State, rng *rand.Rand,
	maxLtr tilemapping.MachineLetter) *xwordgame.Move {
	p := int(s.OnTurn)
	rack := s.Rack(p)
	if len(rack) == 0 {
		return nil
	}
	avail := append([]tilemapping.MachineLetter(nil), rack...)
	rng.Shuffle(len(avail), func(i, j int) { avail[i], avail[j] = avail[j], avail[i] })

	dim := s.Dim()
	vertical := rng.IntN(2) == 0
	dRow, dCol := 0, 1
	if vertical {
		dRow, dCol = 1, 0
	}
	row, col := rng.IntN(dim), rng.IntN(dim)
	want := 1 + rng.IntN(len(avail))

	tiles := make(tilemapping.MachineWord, 0, xwordgame.RackTileLimit+dim)
	used := 0
	for i := 0; used < want; i++ {
		rr, cc := row+dRow*i, col+dCol*i
		if !s.PosExists(rr, cc) {
			break
		}
		if s.HasLetter(rr, cc) {
			tiles = append(tiles, 0)
			continue
		}
		ml := avail[used]
		used++
		if ml == 0 {
			// Designated from the distribution's real alphabet, not english's.
			ml = tilemapping.MachineLetter(1+rng.IntN(int(maxLtr))) | tilemapping.BlankMask
		}
		tiles = append(tiles, ml)
	}
	if used == 0 {
		return nil
	}
	return xwordgame.NewPlacementMove(row, col, vertical, tiles)
}
