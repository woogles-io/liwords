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

func baseParams(t *testing.T, rule ChallengeRule, prev *State, ld *tilemapping.LetterDistribution) ChallengeParams {
	t.Helper()
	return ChallengeParams{
		Rule:               rule,
		Lexicon:            testLexicon(t, "CSW21"),
		Variant:            VarClassic,
		LetterDistribution: ld,
		Prev:               prev,
	}
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

func TestChallengeSubsetPaysPerWord(t *testing.T) {
	is := is.New(t)
	cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
	alph := ld.TileMapping()
	// Pretend the play formed three words, all valid.
	cur.LastWordsFormed = []tilemapping.MachineWord{
		word(t, alph, "FRIENDS"), word(t, alph, "AT"), word(t, alph, "ON"),
	}

	p := baseParams(t, ChallengeRuleFivePoint, prev, ld)
	p.WordIndices = []uint32{1, 2}
	out, err := cur.AdjudicateChallenge(p)
	is.NoErr(err)
	is.True(out.PlayLegal)
	is.Equal(len(out.ChallengedWords), 2)
	// Two words challenged, ten points. Challenging all three would pay the
	// flat five, which is the rule macondo implements and the only behaviour
	// live games have ever seen.
	is.Equal(out.BonusPoints, int32(10))
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
	p2.Variant = VarWordSmog
	out2, err := cur2.AdjudicateChallenge(p2)
	is.NoErr(err)
	is.True(out2.PlayLegal) // wordsmog: accepted
}
