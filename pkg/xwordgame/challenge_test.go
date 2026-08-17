package xwordgame

import (
	"testing"

	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
)

// challengeFixture builds the position immediately after player 0 has played a
// word at row 7, and returns both that position and the one before the play, so
// a successful challenge has something exact to roll back to.
//
// Player 1 is on turn and is therefore the challenger.
func challengeFixture(t *testing.T, played string, score int32) (cur, prev *State, ld *tilemapping.LetterDistribution) {
	t.Helper()
	ld = testLetterDistribution(t, "english")
	alph := ld.TileMapping()

	prev, err := NewState(15)
	if err != nil {
		t.Fatal(err)
	}
	if err := prev.FillBag(ld); err != nil {
		t.Fatal(err)
	}
	if err := prev.AssignRack(0, word(t, alph, played)); err != nil {
		t.Fatal(err)
	}
	if err := prev.AssignRack(1, word(t, alph, "AEIOUST")); err != nil {
		t.Fatal(err)
	}
	prev.Scores = [MaxPlayers]int32{120, 140}
	prev.OnTurn = 0
	prev.TurnNum = 10
	prev.PlayerTurns = [MaxPlayers]uint16{5, 5}
	prev.ScorelessTurns = 2

	// Now play it.
	cur = prev.Clone()
	m := NewPlacementMove(7, 7-len(played)/2, false, word(t, alph, played))
	words, err := cur.FormedWords(m)
	if err != nil {
		t.Fatal(err)
	}
	cur.PlaceMoveTiles(m)
	if err := cur.TakeFromRack(0, m.Tiles); err != nil {
		t.Fatal(err)
	}
	if _, err := cur.DrawToFull(seededRand(), 0); err != nil {
		t.Fatal(err)
	}
	cur.Scores[0] += score
	cur.ScorelessTurns = 0
	cur.PlayerTurns[0]++
	cur.TurnNum++
	cur.OnTurn = 1
	cur.LastWordsFormed = words

	if err := cur.ValidateTileConservation(ld); err != nil {
		t.Fatal(err)
	}
	return cur, prev, ld
}

// testRules is a classic 15x15 english game with a real lexicon.
func testRules(t *testing.T, rule ChallengeRule, ld *tilemapping.LetterDistribution) *Rules {
	t.Helper()
	layout, err := NamedLayout(CrosswordGameLayout)
	if err != nil {
		t.Fatal(err)
	}
	return &Rules{
		Layout:             layout,
		LetterDistribution: ld,
		Lexicon:            testLexicon(t, "CSW21"),
		Variant:            VarClassic,
		ChallengeRule:      rule,
	}
}

// boardFixture is an empty board with a full english bag.
func boardFixture(t *testing.T, rule ChallengeRule) (*State, *Rules) {
	t.Helper()
	ld := testLetterDistribution(t, "english")
	r := testRules(t, rule, ld)
	s, err := NewState(r.Layout.Dim())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FillBag(ld); err != nil {
		t.Fatal(err)
	}
	return s, r
}

// playPlacement runs a tile play through the real state machine and returns the
// position from before it, which challenge adjudication needs. Deriving
// LastWordsFormed from the board rather than setting it by hand is the point of
// the tests that use this.
func playPlacement(t *testing.T, s *State, r *Rules, m *Move) *State {
	t.Helper()
	prev := s.Clone()
	if _, err := s.ApplyPlacement(r, seededRand(), m); err != nil {
		t.Fatal(err)
	}
	if err := s.ValidateTileConservation(r.LetterDistribution); err != nil {
		t.Fatal(err)
	}
	return prev
}

func baseParams(t *testing.T, rule ChallengeRule, prev *State, ld *tilemapping.LetterDistribution) ChallengeParams {
	t.Helper()
	return ChallengeParams{Rules: testRules(t, rule, ld), Prev: prev}
}

func TestSuccessfulChallengeReturnsThePhony(t *testing.T) {
	is := is.New(t)
	// ZZZ is not a word in any lexicon we ship, but the board only cares that
	// the geometry is legal.
	cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
	// Overwrite the formed word with something the lexicon rejects.
	cur.LastWordsFormed = []tilemapping.MachineWord{word(t, ld.TileMapping(), "QZJXV")}

	out, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleDouble, prev, ld))
	is.NoErr(err)
	is.True(!out.PlayLegal)
	is.True(out.PhonyReturned)
	is.Equal(out.LostScore, int32(74))
	is.Equal(len(out.IllegalWords), 1)

	// The position is exactly what it was before the play.
	is.Equal(cur.Scores, [MaxPlayers]int32{120, 140})
	is.Equal(cur.TurnNum, prev.TurnNum)
	is.Equal(cur.PlayerTurns, prev.PlayerTurns)
	is.Equal(cur.Rack(0), prev.Rack(0))
	is.Equal(cur.BagCounts(), prev.BagCounts())
	is.True(cur.IsBoardEmpty())
	is.NoErr(cur.ValidateTileConservation(ld))

	// But the challenger keeps the turn, and the forfeited turn is scoreless.
	is.Equal(cur.OnTurn, uint8(1))
	is.Equal(cur.ScorelessTurns, prev.ScorelessTurns+1)
	// The words are spent; a second challenge has nothing to work on.
	is.Equal(len(cur.LastWordsFormed), 0)
}

func TestSuccessfulChallengeRestoresTheDrawnTiles(t *testing.T) {
	is := is.New(t)
	cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
	cur.LastWordsFormed = []tilemapping.MachineWord{word(t, ld.TileMapping(), "QZJXV")}

	// The offender drew seven replacements. Those go back, and because the bag
	// is a multiset there is no draw order for them to have learned -- macondo
	// has to reshuffle at this point, and we have nothing to reshuffle.
	is.Equal(cur.TilesRemaining(), prev.TilesRemaining()-7)

	_, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleSingle, prev, ld))
	is.NoErr(err)
	is.Equal(cur.TilesRemaining(), prev.TilesRemaining())
	is.Equal(cur.BagCounts(), prev.BagCounts())
}

func TestUnsuccessfulChallengeBonusesByRule(t *testing.T) {
	for _, tc := range []struct {
		rule      ChallengeRule
		wantBonus int32
		wantTurn  uint8 // who is on turn afterwards
		turnLoss  bool
	}{
		{ChallengeRuleSingle, 0, 1, false},
		{ChallengeRuleFivePoint, 5, 1, false},
		{ChallengeRuleTenPoint, 10, 1, false},
		{ChallengeRuleDouble, 0, 0, true},
	} {
		t.Run(tc.rule.String(), func(t *testing.T) {
			is := is.New(t)
			cur, prev, ld := challengeFixture(t, "FRIENDS", 74)

			out, err := cur.AdjudicateChallenge(baseParams(t, tc.rule, prev, ld))
			is.NoErr(err)
			is.True(out.PlayLegal)
			is.True(!out.PhonyReturned)
			is.Equal(out.BonusPoints, tc.wantBonus)
			is.Equal(out.TurnLoss, tc.turnLoss)
			is.Equal(cur.Scores[0], 120+74+tc.wantBonus)
			is.Equal(cur.Scores[1], int32(140))
			// Under the points rules the challenger still owes a move; under
			// DOUBLE they have just spent it.
			is.Equal(cur.OnTurn, tc.wantTurn)
			is.NoErr(cur.ValidateTileConservation(ld))
		})
	}
}

func TestDoubleTurnLossIsAScorelessTurn(t *testing.T) {
	is := is.New(t)
	cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
	is.Equal(cur.ScorelessTurns, uint8(0)) // the tile play reset it

	_, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleDouble, prev, ld))
	is.NoErr(err)
	is.Equal(cur.ScorelessTurns, uint8(1))
	is.Equal(cur.PlayerTurns, [MaxPlayers]uint16{6, 6})
}

// The five-point rule is the only one whose payout varies, and it varies by
// mode rather than by whether the caller happened to name indices.
func TestFivePointModes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mode    FivePointMode
		indices []uint32
		want    int32
	}{
		// macondo, and live liwords today: the whole play, five points, once.
		{"per-play, whole play", FivePointPerPlay, nil, 5},
		// Naming words must not change the payout in this mode. This is the
		// case that was wrong before: it used to pay 10.
		{"per-play, two words named", FivePointPerPlay, []uint32{1, 2}, 5},
		{"per-play, every word named", FivePointPerPlay, []uint32{0, 1, 2}, 5},
		// cwgame, for the annotator.
		{"per-word, two words named", FivePointPerWord, []uint32{1, 2}, 10},
		{"per-word, every word named", FivePointPerWord, []uint32{0, 1, 2}, 15},
	} {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
			alph := ld.TileMapping()
			// A play that formed three words, all valid.
			cur.LastWordsFormed = []tilemapping.MachineWord{
				word(t, alph, "FRIENDS"), word(t, alph, "AT"), word(t, alph, "ON"),
			}

			p := baseParams(t, ChallengeRuleFivePoint, prev, ld)
			p.WordIndices = tc.indices
			p.Rules.FivePointMode = tc.mode
			out, err := cur.AdjudicateChallenge(p)
			is.NoErr(err)
			is.True(out.PlayLegal)
			is.Equal(out.BonusPoints, tc.want)
			is.Equal(cur.Scores[0], 120+74+tc.want)
		})
	}
}

// The tests above set LastWordsFormed by hand, which is how the referee
// receives it in production but leaves the fixture's board holding a different
// word from the one being challenged. These play a real phony and derive the
// words from the board, so the position and the challenge actually correspond.

func TestChallengeAPhonyPlayedOnTheBoard(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	ld := r.LetterDistribution
	alph := ld.TileMapping()

	// FRIENDZ is not in CSW21, but it is a perfectly legal opening play as far
	// as the board is concerned. Nothing but a challenge can take it off.
	is.NoErr(s.AssignRack(0, word(t, alph, "FRIENDZ")))
	rackBefore := append([]tilemapping.MachineLetter(nil), s.Rack(0)...)
	bagBefore := s.BagCounts()

	m := NewPlacementMove(7, 7, false, word(t, alph, "FRIENDZ"))
	prev := playPlacement(t, s, r, m)

	// The words under challenge came off the board, not out of the test.
	is.Equal(len(s.LastWordsFormed), 1)
	is.Equal(s.LastWordsFormed[0], word(t, alph, "FRIENDZ"))
	is.True(!r.Lexicon.HasWord(s.LastWordsFormed[0]))
	is.True(!s.IsBoardEmpty())
	is.Equal(s.Bingos[0], uint16(1))
	scored := s.Scores[0]
	is.True(scored > 0)
	is.Equal(s.OnTurn, uint8(1)) // the opponent, who is about to challenge

	out, err := s.AdjudicateChallenge(baseParams(t, ChallengeRuleDouble, prev, ld))
	is.NoErr(err)
	is.True(!out.PlayLegal)
	is.True(out.PhonyReturned)
	is.Equal(out.LostScore, scored)
	is.Equal(len(out.IllegalWords), 1)

	// Everything the play touched is back: board, rack, bag, score, bingo count.
	is.True(s.IsBoardEmpty())
	is.Equal(s.Rack(0), rackBefore)
	is.Equal(s.BagCounts(), bagBefore)
	is.Equal(s.Scores, [MaxPlayers]int32{0, 0})
	is.Equal(s.Bingos[0], uint16(0))
	is.NoErr(s.ValidateTileConservation(ld))
}

// A play whose main word is fine but which hooks a phony onto an existing word.
// This is the case that makes subset challenges mean anything.
func TestChallengeAPhonyCrossWord(t *testing.T) {
	is := is.New(t)

	setup := func(t *testing.T) (*State, *State, *tilemapping.LetterDistribution, Lexicon) {
		t.Helper()
		s, r := boardFixture(t, ChallengeRuleSingle)
		ld := r.LetterDistribution
		alph := ld.TileMapping()

		// AT across the centre.
		if err := s.AssignRack(0, word(t, alph, "AT")); err != nil {
			t.Fatal(err)
		}
		playPlacement(t, s, r, NewPlacementMove(7, 6, false, word(t, alph, "AT")))

		// AW parallel above it: valid itself, but it hooks A onto A and W onto T.
		if err := s.AssignRack(1, word(t, alph, "AW")); err != nil {
			t.Fatal(err)
		}
		prev := playPlacement(t, s, r, NewPlacementMove(6, 6, false, word(t, alph, "AW")))
		return s, prev, ld, r.Lexicon
	}

	t.Run("the board really does form one phony among three words", func(t *testing.T) {
		s, _, ld, lex := setup(t)
		alph := ld.TileMapping()
		is.Equal(len(s.LastWordsFormed), 3)
		is.Equal(s.LastWordsFormed[0], word(t, alph, "AW")) // main word
		is.Equal(s.LastWordsFormed[1], word(t, alph, "AA")) // hook on the A
		is.Equal(s.LastWordsFormed[2], word(t, alph, "WT")) // hook on the T
		is.True(lex.HasWord(s.LastWordsFormed[0]))
		is.True(lex.HasWord(s.LastWordsFormed[1]))
		is.True(!lex.HasWord(s.LastWordsFormed[2]))
	})

	t.Run("challenging the whole play catches it", func(t *testing.T) {
		s, prev, ld, _ := setup(t)
		scored := s.Scores[1]
		out, err := s.AdjudicateChallenge(baseParams(t, ChallengeRuleSingle, prev, ld))
		is.NoErr(err)
		is.True(!out.PlayLegal)
		is.True(out.PhonyReturned)
		is.Equal(out.LostScore, scored)
		is.Equal(len(out.IllegalWords), 1)
		// Only the AW comes off; the AT underneath stays.
		is.Equal(s.TilesOnBoard(), 2)
		is.NoErr(s.ValidateTileConservation(ld))
	})

	t.Run("challenging only the valid words lets the phony stand", func(t *testing.T) {
		s, prev, ld, _ := setup(t)
		p := baseParams(t, ChallengeRuleFivePoint, prev, ld)
		p.WordIndices = []uint32{0, 1} // AW and AA, not WT
		p.Rules.FivePointMode = FivePointPerWord
		out, err := s.AdjudicateChallenge(p)
		is.NoErr(err)
		is.True(out.PlayLegal)
		is.True(!out.PhonyReturned)
		is.Equal(out.BonusPoints, int32(10)) // two words named, per-word
		is.Equal(s.TilesOnBoard(), 4)        // the phony is still there
	})

	t.Run("naming the phony catches it even in a subset", func(t *testing.T) {
		s, prev, ld, _ := setup(t)
		p := baseParams(t, ChallengeRuleFivePoint, prev, ld)
		p.WordIndices = []uint32{2} // just WT
		p.Rules.FivePointMode = FivePointPerWord
		out, err := s.AdjudicateChallenge(p)
		is.NoErr(err)
		is.True(!out.PlayLegal)
		is.True(out.PhonyReturned)
		is.Equal(s.TilesOnBoard(), 2)
	})
}

// "Per word, but I will not say which words" has two plausible readings, so it
// is refused rather than guessed at.
func TestPerWordWithoutIndicesIsRefused(t *testing.T) {
	is := is.New(t)
	cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
	alph := ld.TileMapping()
	cur.LastWordsFormed = []tilemapping.MachineWord{
		word(t, alph, "FRIENDS"), word(t, alph, "AT"), word(t, alph, "ON"),
	}
	before := cur.Clone()

	p := baseParams(t, ChallengeRuleFivePoint, prev, ld)
	p.Rules.FivePointMode = FivePointPerWord
	_, err := cur.AdjudicateChallenge(p)
	is.Equal(err, ErrPerWordNeedsIndices)
	// Rejected before anything was touched.
	is.True(cur.Equal(before))
	t.Log(err)
}

// The mode is only meaningful under the five-point rule, so carrying it around
// for a whole session must not trip up the rules that ignore it.
func TestPerWordWithoutIndicesIsFineForOtherRules(t *testing.T) {
	is := is.New(t)
	for _, rule := range []ChallengeRule{
		ChallengeRuleSingle, ChallengeRuleTenPoint, ChallengeRuleDouble,
	} {
		t.Run(rule.String(), func(t *testing.T) {
			cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
			p := baseParams(t, rule, prev, ld)
			p.Rules.FivePointMode = FivePointPerWord
			_, err := cur.AdjudicateChallenge(p)
			is.NoErr(err)
		})
	}
}

// Byte-exact parity with pkg/cwgame, which has no mode field and infers
// per-word scoring from the presence of indices. These are the numbers
// pkg/cwgame/game.go:633-639 produces for a play forming three valid words.
func TestLegacyFivePointModeMatchesCwgame(t *testing.T) {
	is := is.New(t)
	is.Equal(LegacyFivePointMode(nil), FivePointPerPlay)
	is.Equal(LegacyFivePointMode([]uint32{}), FivePointPerPlay)
	is.Equal(LegacyFivePointMode([]uint32{0}), FivePointPerWord)

	for _, tc := range []struct {
		name    string
		indices []uint32
		want    int32
	}{
		// challengingSubset == false: "Legacy behavior: flat 5 points when
		// challenging all".
		{"no indices", nil, 5},
		// challengingSubset == true: 5 * len(wordsToValidate).
		{"one word", []uint32{1}, 5},
		{"two words", []uint32{1, 2}, 10},
		{"every word listed", []uint32{0, 1, 2}, 15},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
			alph := ld.TileMapping()
			cur.LastWordsFormed = []tilemapping.MachineWord{
				word(t, alph, "FRIENDS"), word(t, alph, "AT"), word(t, alph, "ON"),
			}
			p := baseParams(t, ChallengeRuleFivePoint, prev, ld)
			p.WordIndices = tc.indices
			p.Rules.FivePointMode = LegacyFivePointMode(tc.indices)
			out, err := cur.AdjudicateChallenge(p)
			is.NoErr(err)
			is.Equal(out.BonusPoints, tc.want)
		})
	}
}

// The default must be what live games do today, so a caller that sets no mode
// cannot accidentally change scoring.
func TestFivePointDefaultIsFlat(t *testing.T) {
	is := is.New(t)
	is.Equal(FivePointMode(0), FivePointPerPlay)

	cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
	alph := ld.TileMapping()
	cur.LastWordsFormed = []tilemapping.MachineWord{
		word(t, alph, "FRIENDS"), word(t, alph, "AT"), word(t, alph, "ON"),
	}
	p := baseParams(t, ChallengeRuleFivePoint, prev, ld) // mode left unset
	p.WordIndices = []uint32{0, 1, 2}
	out, err := cur.AdjudicateChallenge(p)
	is.NoErr(err)
	is.Equal(out.BonusPoints, int32(5))
}

// Only the five-point rule varies. Everything else pays once for the whole
// play no matter how many words were named or which mode is set.
func TestOtherRulesIgnoreTheWordCount(t *testing.T) {
	for _, tc := range []struct {
		rule ChallengeRule
		want int32
	}{
		{ChallengeRuleSingle, 0},
		{ChallengeRuleTenPoint, 10},
	} {
		for _, mode := range []FivePointMode{FivePointPerPlay, FivePointPerWord} {
			t.Run(tc.rule.String()+"/"+mode.String(), func(t *testing.T) {
				is := is.New(t)
				cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
				alph := ld.TileMapping()
				cur.LastWordsFormed = []tilemapping.MachineWord{
					word(t, alph, "FRIENDS"), word(t, alph, "AT"), word(t, alph, "ON"),
				}
				p := baseParams(t, tc.rule, prev, ld)
				p.WordIndices = []uint32{0, 1, 2}
				p.Rules.FivePointMode = mode
				out, err := cur.AdjudicateChallenge(p)
				is.NoErr(err)
				is.Equal(out.BonusPoints, tc.want)
			})
		}
	}
}

func TestChallengeSubsetCatchesOnlyTheNamedWords(t *testing.T) {
	is := is.New(t)
	cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
	alph := ld.TileMapping()
	cur.LastWordsFormed = []tilemapping.MachineWord{
		word(t, alph, "QZJXV"), word(t, alph, "AT"),
	}

	// Challenge only the valid word: the phony survives.
	p := baseParams(t, ChallengeRuleSingle, prev, ld)
	p.WordIndices = []uint32{1}
	out, err := cur.AdjudicateChallenge(p)
	is.NoErr(err)
	is.True(out.PlayLegal)
	is.True(!out.PhonyReturned)
}

func TestChallengeSubsetRejectsBadIndices(t *testing.T) {
	is := is.New(t)
	cur, prev, ld := challengeFixture(t, "FRIENDS", 74)

	p := baseParams(t, ChallengeRuleFivePoint, prev, ld)
	p.WordIndices = []uint32{5}
	_, err := cur.AdjudicateChallenge(p)
	is.True(err != nil)
	t.Log(err)

	// A repeated index would otherwise multiply the per-word bonus for free.
	cur2, prev2, ld2 := challengeFixture(t, "FRIENDS", 74)
	p2 := baseParams(t, ChallengeRuleFivePoint, prev2, ld2)
	p2.WordIndices = []uint32{0, 0, 0, 0}
	_, err = cur2.AdjudicateChallenge(p2)
	is.True(err != nil)
	t.Log(err)
}

func TestTripleChallenge(t *testing.T) {
	is := is.New(t)

	t.Run("challenger wrong loses the game", func(t *testing.T) {
		cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
		out, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleTriple, prev, ld))
		is.NoErr(err)
		is.True(out.PlayLegal)
		is.True(out.TripleChallenge)
		is.True(out.GameOver)
		is.Equal(out.Winner, int8(0)) // the challengee
		is.Equal(cur.PlayState, GameOver)
		// The play stays on the board and the score stands.
		is.Equal(cur.Scores[0], int32(194))
	})

	t.Run("challenger right wins the game", func(t *testing.T) {
		cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
		cur.LastWordsFormed = []tilemapping.MachineWord{word(t, ld.TileMapping(), "QZJXV")}
		out, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleTriple, prev, ld))
		is.NoErr(err)
		is.True(!out.PlayLegal)
		is.True(out.PhonyReturned)
		is.True(out.GameOver)
		is.Equal(out.Winner, int8(1)) // the challenger
		is.Equal(cur.PlayState, GameOver)
		is.Equal(cur.Scores, [MaxPlayers]int32{120, 140})
		is.NoErr(cur.ValidateTileConservation(ld))
	})
}

// A player who goes out under a non-void rule sits in WAITING_FOR_FINAL_PASS
// until the opponent passes or challenges. This is the state that has no
// corresponding event, and getting it wrong is what broke 944 games.
func TestChallengeAgainstAPlayerWhoWentOut(t *testing.T) {
	is := is.New(t)

	t.Run("failed challenge ends the game and pays the out bonus", func(t *testing.T) {
		cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
		cur.PlayState = WaitingForFinalPass
		// The challenger is left holding AEIOUST: 1+1+1+1+1+1+1 = 7.
		out, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleTenPoint, prev, ld))
		is.NoErr(err)
		is.True(out.PlayLegal)
		is.True(out.GameOver)
		is.Equal(out.BonusPoints, int32(10))
		is.Equal(out.EndRackBonus, int32(14))
		is.Equal(out.EndRackBonusPlayer, int8(0))
		is.Equal(cur.PlayState, GameOver)
		is.Equal(cur.Scores[0], int32(120+74+10+14))
	})

	t.Run("failed double challenge ends it through the turn loss", func(t *testing.T) {
		cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
		cur.PlayState = WaitingForFinalPass
		out, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleDouble, prev, ld))
		is.NoErr(err)
		is.True(out.GameOver)
		is.True(out.TurnLoss)
		is.Equal(out.EndRackBonus, int32(14))
		is.Equal(out.EndRackBonusPlayer, int8(0))
		is.Equal(cur.Scores[0], int32(120+74+14))
	})

	t.Run("successful challenge un-ends the game", func(t *testing.T) {
		cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
		cur.PlayState = WaitingForFinalPass
		cur.LastWordsFormed = []tilemapping.MachineWord{word(t, ld.TileMapping(), "QZJXV")}
		out, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleSingle, prev, ld))
		is.NoErr(err)
		is.True(out.PhonyReturned)
		is.True(!out.GameOver)
		// Restoring the previous position restores the play state with it. No
		// event says "we left WAITING_FOR_FINAL_PASS"; the position does.
		is.Equal(cur.PlayState, Playing)
		is.Equal(cur.OnTurn, uint8(1))
	})
}

// The sixth consecutive scoreless turn ends the game, and a returned phony can
// be the turn that gets there.
func TestSuccessfulChallengeCanTriggerTheScorelessStalemate(t *testing.T) {
	is := is.New(t)
	cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
	prev.ScorelessTurns = MaxScorelessTurns - 1
	cur.LastWordsFormed = []tilemapping.MachineWord{word(t, ld.TileMapping(), "QZJXV")}

	out, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleSingle, prev, ld))
	is.NoErr(err)
	is.True(out.PhonyReturned)
	is.True(out.GameOver)
	is.Equal(cur.PlayState, GameOver)
	is.True(out.ScorelessPenalties != nil)
	// Each player loses their own rack: FRIENDS is 4+1+1+1+1+2+1 = 11,
	// AEIOUST is 7.
	is.Equal(*out.ScorelessPenalties, [MaxPlayers]int32{11, 7})
	is.Equal(cur.Scores, [MaxPlayers]int32{120 - 11, 140 - 7})
}

func TestChallengeRefusals(t *testing.T) {
	is := is.New(t)

	t.Run("void rule", func(t *testing.T) {
		cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
		_, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleVoid, prev, ld))
		is.True(err != nil)
	})

	t.Run("nothing to challenge", func(t *testing.T) {
		cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
		cur.LastWordsFormed = nil
		_, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleDouble, prev, ld))
		is.True(err != nil)
	})

	t.Run("game already over", func(t *testing.T) {
		cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
		cur.PlayState = GameOver
		_, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleDouble, prev, ld))
		is.True(err != nil)
	})

	t.Run("missing previous position", func(t *testing.T) {
		cur, _, ld := challengeFixture(t, "FRIENDS", 74)
		cur.LastWordsFormed = []tilemapping.MachineWord{word(t, ld.TileMapping(), "QZJXV")}
		p := baseParams(t, ChallengeRuleDouble, nil, ld)
		_, err := cur.AdjudicateChallenge(p)
		is.True(err != nil)
		t.Log(err)
	})

	// A valid play needs no previous position, because nothing comes off the
	// board. Requiring it unconditionally would make the common case harder for
	// no benefit.
	t.Run("valid play without a previous position", func(t *testing.T) {
		cur, _, ld := challengeFixture(t, "FRIENDS", 74)
		out, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleTenPoint, nil, ld))
		is.NoErr(err)
		is.True(out.PlayLegal)
	})
}

// Handing the referee the wrong snapshot is the failure mode this design exists
// to prevent, so it must be loud rather than silent.
func TestPhonyReturnRejectsAMismatchedPreviousPosition(t *testing.T) {
	is := is.New(t)

	t.Run("wrong turn number", func(t *testing.T) {
		cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
		cur.LastWordsFormed = []tilemapping.MachineWord{word(t, ld.TileMapping(), "QZJXV")}
		prev.TurnNum = 3
		_, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleSingle, prev, ld))
		is.True(err != nil)
		t.Log(err)
	})

	t.Run("wrong player on turn", func(t *testing.T) {
		cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
		cur.LastWordsFormed = []tilemapping.MachineWord{word(t, ld.TileMapping(), "QZJXV")}
		prev.OnTurn = 1
		_, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleSingle, prev, ld))
		is.True(err != nil)
		t.Log(err)
	})
}

func TestWordSmogAcceptsAnagrams(t *testing.T) {
	is := is.New(t)
	cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
	alph := ld.TileMapping()
	// SDNEIRF is not a word, but it anagrams to FRIENDS.
	cur.LastWordsFormed = []tilemapping.MachineWord{word(t, alph, "SDNEIRF")}

	p := baseParams(t, ChallengeRuleSingle, prev, ld)
	out, err := cur.AdjudicateChallenge(p)
	is.NoErr(err)
	is.True(!out.PlayLegal) // classic: rejected

	cur2, prev2, ld2 := challengeFixture(t, "FRIENDS", 74)
	cur2.LastWordsFormed = []tilemapping.MachineWord{word(t, alph, "SDNEIRF")}
	p2 := baseParams(t, ChallengeRuleSingle, prev2, ld2)
	p2.Rules.Variant = VarWordSmog
	out2, err := cur2.AdjudicateChallenge(p2)
	is.NoErr(err)
	is.True(out2.PlayLegal) // wordsmog: accepted
}
