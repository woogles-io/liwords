package xwordgame

import (
	"errors"
	"math/rand/v2"
	"testing"

	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
)

func TestApplyPlacement(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	alph := r.LetterDistribution.TileMapping()

	is.NoErr(s.AssignRack(0, word(t, alph, "HELLOAB")))
	s.ScorelessTurns = 3
	bagBefore := s.TilesRemaining()

	// HELLO at row 7 cols 3..7: 24 points, as TestOpeningPlayScore computes by
	// hand.
	m := NewPlacementMove(7, 3, false, word(t, alph, "HELLO"))
	res, err := s.ApplyPlacement(r, seededRand(), m)
	is.NoErr(err)

	is.Equal(res.Score, int32(24))
	is.Equal(len(res.WordsFormed), 1)
	is.True(!res.Bingo)
	is.True(!res.WentOut)
	is.True(!res.GameOver)
	is.Equal(len(res.Drew), 5)

	is.Equal(s.Scores, [MaxPlayers]int32{24, 0})
	is.Equal(s.RackLen(0), RackTileLimit) // AB kept, five drawn
	is.Equal(s.TilesRemaining(), bagBefore-5)
	is.Equal(s.PlayerTurns, [MaxPlayers]uint16{1, 0})
	is.Equal(s.TurnNum, uint16(1))
	is.Equal(s.OnTurn, uint8(1))
	// A tile play always resets the counter, whatever it scored.
	is.Equal(s.ScorelessTurns, uint8(0))
	is.Equal(s.LastWordsFormed, res.WordsFormed)
	is.NoErr(s.ValidateTileConservation(r.LetterDistribution))
}

func TestApplyPlacementBingo(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	alph := r.LetterDistribution.TileMapping()

	is.NoErr(s.AssignRack(0, word(t, alph, "FRIENDS")))
	res, err := s.ApplyPlacement(r, seededRand(), NewPlacementMove(7, 7, false, word(t, alph, "FRIENDS")))
	is.NoErr(err)
	is.True(res.Bingo)
	is.Equal(res.Score, int32(74))
	is.Equal(s.Bingos, [MaxPlayers]uint16{1, 0})
	// Seven played, seven drawn back.
	is.Equal(s.RackLen(0), RackTileLimit)
	is.True(!res.WentOut) // the bag refilled the rack
}

// Going out is the transition with no corresponding event, and losing it is
// what corrupted 944 games.
func TestApplyPlacementGoingOut(t *testing.T) {
	is := is.New(t)

	t.Run("under a challenge rule it waits for the final pass", func(t *testing.T) {
		s, r := boardFixture(t, ChallengeRuleDouble)
		alph := r.LetterDistribution.TileMapping()
		is.NoErr(s.AssignRack(0, word(t, alph, "HELLO")))
		is.NoErr(s.AssignRack(1, word(t, alph, "QUIZ")))
		// Empty the bag so there is nothing to draw back.
		var scratch []tilemapping.MachineLetter
		_, n := s.DrawAtMost(seededRand(), s.TilesRemaining(), scratch)
		is.True(n > 0)
		is.Equal(s.TilesRemaining(), 0)

		res, err := s.ApplyPlacement(r, seededRand(), NewPlacementMove(7, 3, false, word(t, alph, "HELLO")))
		is.NoErr(err)
		is.True(res.WentOut)
		is.True(!res.GameOver) // the opponent may still pass or challenge
		is.Equal(s.PlayState, WaitingForFinalPass)
		is.Equal(s.RackLen(0), 0)
		// The turn passes: the opponent is the one who has to act.
		is.Equal(s.OnTurn, uint8(1))
		is.Equal(res.EndRackBonusPlayer, int8(-1))

		// And the pass closes it, paying the out player double the opponent's
		// rack. QUIZ is 22.
		passRes, err := s.ApplyPass(r)
		is.NoErr(err)
		is.True(passRes.GameOver)
		is.Equal(passRes.EndRackBonus, int32(44))
		is.Equal(passRes.EndRackBonusPlayer, int8(0))
		is.Equal(s.PlayState, GameOver)
		// The turn still flips. It means nothing once the game is over, but
		// macondo flips here and a byte-level state comparison would report a
		// divergence that is not one. Caught by the bridge parity test.
		is.Equal(s.OnTurn, uint8(0))
	})

	t.Run("under void the game ends immediately", func(t *testing.T) {
		s, r := boardFixture(t, ChallengeRuleVoid)
		alph := r.LetterDistribution.TileMapping()
		is.NoErr(s.AssignRack(0, word(t, alph, "HELLO")))
		is.NoErr(s.AssignRack(1, word(t, alph, "QUIZ")))
		var scratch []tilemapping.MachineLetter
		s.DrawAtMost(seededRand(), s.TilesRemaining(), scratch)

		res, err := s.ApplyPlacement(r, seededRand(), NewPlacementMove(7, 3, false, word(t, alph, "HELLO")))
		is.NoErr(err)
		is.True(res.WentOut)
		is.True(res.GameOver)
		is.Equal(s.PlayState, GameOver)
		is.Equal(res.EndRackBonus, int32(44))
		is.Equal(res.EndRackBonusPlayer, int8(0))
		is.Equal(s.Scores[0], int32(24+44))
	})
}

// Under void there is no challenge to catch a phony, so the referee refuses it
// when it is played.
func TestVoidRejectsInvalidWords(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleVoid)
	alph := r.LetterDistribution.TileMapping()
	is.NoErr(s.AssignRack(0, word(t, alph, "FRIENDZ")))
	before := s.Clone()

	_, err := s.ApplyPlacement(r, seededRand(), NewPlacementMove(7, 7, false, word(t, alph, "FRIENDZ")))
	is.True(err != nil)

	// A typed error, so callers do not have to re-derive the words the way
	// pkg/gameplay does at game.go:511.
	var iwe *InvalidWordsError
	is.True(errors.As(err, &iwe))
	is.Equal(len(iwe.Words), 1)
	is.Equal(iwe.Words[0], word(t, alph, "FRIENDZ"))

	// Nothing was applied.
	is.True(s.Equal(before))

	// The same play is fine when a challenge could catch it.
	r.ChallengeRule = ChallengeRuleDouble
	_, err = s.ApplyPlacement(r, seededRand(), NewPlacementMove(7, 7, false, word(t, alph, "FRIENDZ")))
	is.NoErr(err)
}

// ValidateMove has to reject a move on its own, not lean on the mutating path
// to fail later: callers use it as a pure check before deciding anything.
func TestValidateMoveChecksTheRack(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	alph := r.LetterDistribution.TileMapping()
	is.NoErr(s.AssignRack(0, word(t, alph, "AEIOU")))

	// Geometrically fine, but the rack has no Z.
	_, err := s.ValidateMove(r, NewPlacementMove(7, 5, false, word(t, alph, "ZZZ")))
	is.True(err != nil)
	t.Log(err)

	_, err = s.ValidateMove(r, NewExchangeMove(word(t, alph, "Z")))
	is.True(err != nil)
	t.Log(err)

	// The same shape using tiles the player holds is accepted.
	_, err = s.ValidateMove(r, NewPlacementMove(7, 5, false, word(t, alph, "AIU")))
	is.NoErr(err)
}

// macondo replenishes by drawing at most the number of tiles played rather than
// topping the rack up (game.go, playMove). The two agree in every real game --
// a rack only runs short once the bag is empty -- but they differ on a position
// built by hand, so pin the behaviour the shadow comparison will depend on.
func TestReplenishDrawsOnlyWhatWasPlayed(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	alph := r.LetterDistribution.TileMapping()

	// Three tiles on the rack with a full bag behind them.
	is.NoErr(s.AssignRack(0, word(t, alph, "AIU")))
	res, err := s.ApplyPlacement(r, seededRand(), NewPlacementMove(7, 5, false, word(t, alph, "AIU")))
	is.NoErr(err)

	// Three played, so three drawn -- not seven.
	is.Equal(len(res.Drew), 3)
	is.Equal(s.RackLen(0), 3)
	is.True(!res.WentOut)
	is.NoErr(s.ValidateTileConservation(r.LetterDistribution))
}

func TestApplyExchange(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	alph := r.LetterDistribution.TileMapping()

	is.NoErr(s.AssignRack(0, word(t, alph, "QUIZJXV")))
	bagBefore := s.TilesRemaining()

	res, err := s.ApplyExchange(r, seededRand(), NewExchangeMove(word(t, alph, "JXV")))
	is.NoErr(err)
	is.Equal(len(res.Drew), 3)
	is.Equal(res.Score, int32(0))

	// The rack is still full and the bag is the same size: tiles moved, none
	// were created or destroyed.
	is.Equal(s.RackLen(0), RackTileLimit)
	is.Equal(s.TilesRemaining(), bagBefore)
	is.NoErr(s.ValidateTileConservation(r.LetterDistribution))

	is.Equal(s.ScorelessTurns, uint8(1))
	is.Equal(s.PlayerTurns, [MaxPlayers]uint16{1, 0})
	is.Equal(s.TurnNum, uint16(1))
	is.Equal(s.OnTurn, uint8(1))
	is.Equal(len(s.LastWordsFormed), 0)
}

func TestExchangeRefusals(t *testing.T) {
	is := is.New(t)
	alph := testLetterDistribution(t, "english").TileMapping()

	t.Run("not enough tiles left in the bag", func(t *testing.T) {
		s, r := boardFixture(t, ChallengeRuleDouble)
		is.NoErr(s.AssignRack(0, word(t, alph, "QUIZJXV")))
		// Leave six in the bag, one short of the limit.
		var scratch []tilemapping.MachineLetter
		s.DrawAtMost(seededRand(), s.TilesRemaining()-6, scratch)
		is.Equal(s.TilesRemaining(), 6)

		_, err := s.ApplyExchange(r, seededRand(), NewExchangeMove(word(t, alph, "J")))
		is.True(err != nil)
		t.Log(err)
	})

	t.Run("a custom limit is honoured", func(t *testing.T) {
		s, r := boardFixture(t, ChallengeRuleDouble)
		r.ExchangeLimit = 1
		is.NoErr(s.AssignRack(0, word(t, alph, "QUIZJXV")))
		var scratch []tilemapping.MachineLetter
		s.DrawAtMost(seededRand(), s.TilesRemaining()-1, scratch)
		is.Equal(s.TilesRemaining(), 1)
		_, err := s.ApplyExchange(r, seededRand(), NewExchangeMove(word(t, alph, "J")))
		is.NoErr(err)
	})

	t.Run("tiles not on the rack", func(t *testing.T) {
		s, r := boardFixture(t, ChallengeRuleDouble)
		is.NoErr(s.AssignRack(0, word(t, alph, "AEIOU")))
		_, err := s.ApplyExchange(r, seededRand(), NewExchangeMove(word(t, alph, "Z")))
		is.True(err != nil)
	})

	t.Run("nothing named", func(t *testing.T) {
		s, r := boardFixture(t, ChallengeRuleDouble)
		is.NoErr(s.AssignRack(0, word(t, alph, "AEIOU")))
		_, err := s.ApplyExchange(r, seededRand(), NewExchangeMove(nil))
		is.True(err != nil)
	})
}

// While a player is waiting for the final pass the opponent may only pass or
// challenge.
func TestWaitingForFinalPassRestrictsMoves(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	alph := r.LetterDistribution.TileMapping()
	is.NoErr(s.AssignRack(1, word(t, alph, "HELLOAB")))
	s.PlayState = WaitingForFinalPass
	s.OnTurn = 1

	_, err := s.ValidateMove(r, NewPlacementMove(7, 3, false, word(t, alph, "HELLO")))
	is.Equal(err, errOnlyPassOrChallenge)
	_, err = s.ValidateMove(r, NewExchangeMove(word(t, alph, "AB")))
	is.Equal(err, errOnlyPassOrChallenge)
	_, err = s.ValidateMove(r, NewPassMove())
	is.NoErr(err)
}

func TestNothingAppliesToAFinishedGame(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	alph := r.LetterDistribution.TileMapping()
	is.NoErr(s.AssignRack(0, word(t, alph, "HELLOAB")))
	s.PlayState = GameOver

	for _, m := range []*Move{
		NewPassMove(),
		NewExchangeMove(word(t, alph, "AB")),
		NewPlacementMove(7, 3, false, word(t, alph, "HELLO")),
	} {
		_, err := s.ApplyMove(r, seededRand(), m)
		is.Equal(err, errGameIsOver)
	}
}

func TestApplyMoveDispatch(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	alph := r.LetterDistribution.TileMapping()
	is.NoErr(s.AssignRack(0, word(t, alph, "HELLOAB")))

	res, err := s.ApplyMove(r, seededRand(), NewPlacementMove(7, 3, false, word(t, alph, "HELLO")))
	is.NoErr(err)
	is.Equal(res.Score, int32(24))

	// A challenge needs the previous position, so it does not come through
	// here.
	_, err = s.ApplyMove(r, seededRand(), NewChallengeMove())
	is.True(err != nil)
	t.Log(err)
}

// Six consecutive scoreless turns end the game, each player losing their own
// rack. The counter has to survive across both players and both move types.
func TestSixScorelessTurnsEndsTheGame(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	alph := r.LetterDistribution.TileMapping()
	is.NoErr(s.AssignRack(0, word(t, alph, "QI")))  // 11
	is.NoErr(s.AssignRack(1, word(t, alph, "JOT"))) // 10

	// Five scoreless turns: passes and exchanges both count.
	for i := range 5 {
		var res *ApplyResult
		var err error
		if i%2 == 0 {
			res, err = s.ApplyPass(r)
		} else {
			res, err = s.ApplyExchange(r, seededRand(), NewExchangeMove(s.Rack(int(s.OnTurn))[:1]))
		}
		is.NoErr(err)
		is.True(!res.GameOver)
		is.Equal(s.ScorelessTurns, uint8(i+1))
	}

	// Racks have churned through exchanges, so read them now.
	want := [MaxPlayers]int32{s.RackScoreFor(r.LetterDistribution, 0), s.RackScoreFor(r.LetterDistribution, 1)}
	before := s.Scores

	res, err := s.ApplyPass(r)
	is.NoErr(err)
	is.True(res.GameOver)
	is.Equal(s.PlayState, GameOver)
	is.True(res.ScorelessPenalties != nil)
	is.Equal(*res.ScorelessPenalties, want)
	is.Equal(s.Scores, [MaxPlayers]int32{before[0] - want[0], before[1] - want[1]})
	is.NoErr(s.ValidateTileConservation(r.LetterDistribution))
}

// The two endings that a replay of a log cannot get right, proven correct here
// where the racks are known.
//
// In both, the move that ends the game also changes the rack it is charged
// against -- an exchange draws replacements, a returned phony restores the
// pre-play rack -- and an event log does not reveal those tiles until the
// end-rack event, which arrives after the penalty is due. Live play has no such
// gap: the racks are in the position the whole time. That is the difference
// between the 15 corpus games whose scores have to be read and a live game that
// simply computes them.
// drainBagOntoBoard empties the bag onto the board, so a test can reach a
// near-empty bag without destroying tiles and tripping the conservation check.
func drainBagOntoBoard(t *testing.T, s *State) {
	t.Helper()
	var buf []tilemapping.MachineLetter
	buf, _ = s.DrawAtMost(seededRand(), s.TilesRemaining(), buf)
	i := 0
	for _, ml := range buf {
		if ml == 0 {
			// A blank only exists on the board with a letter assigned to it.
			ml = 1 | tilemapping.BlankMask
		}
		for i < s.Dim()*s.Dim() && s.HasLetter(i/s.Dim(), i%s.Dim()) {
			i++
		}
		if i >= s.Dim()*s.Dim() {
			t.Fatal("ran out of board")
		}
		s.SetTileAt(i/s.Dim(), i%s.Dim(), ml)
		i++
	}
}

func TestStalemateChargesTheRealRackAfterAnExchange(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	alph := r.LetterDistribution.TileMapping()

	// Draw the bag down to exactly what the exchange needs, so the tiles drawn
	// back are forced and the expected penalty is unambiguous. The surplus goes
	// onto the board rather than into the void, which is both how a real
	// endgame arrives here and what keeps the tiles conserved.
	is.NoErr(s.AssignRack(0, word(t, alph, "AEIOU")))
	is.NoErr(s.AssignRack(1, word(t, alph, "JOT"))) // 8+1+1 = 10
	is.NoErr(s.RemoveFromBag(word(t, alph, "QI")))  // hold these back
	drainBagOntoBoard(t, s)
	is.NoErr(s.PutBack(word(t, alph, "QI")))
	is.Equal(s.TilesRemaining(), 2)

	s.ScorelessTurns = MaxScorelessTurns - 1
	s.Scores = [MaxPlayers]int32{300, 280}
	r.ExchangeLimit = 1

	// Player 0 exchanges two vowels and necessarily draws the Q and the I, so
	// the rack the penalty is charged against is not the one they held when the
	// move began. A replay reading only the events could not know this.
	res, err := s.ApplyExchange(r, seededRand(), NewExchangeMove(word(t, alph, "AE")))
	is.NoErr(err)
	is.True(res.GameOver)
	is.True(res.ScorelessPenalties != nil)

	// Post-exchange rack is IOUQI: 1+1+1+10+1 = 14, not the 3 the pre-exchange
	// AEIOU would have cost.
	is.Equal(*res.ScorelessPenalties, [MaxPlayers]int32{14, 10})
	is.Equal(s.Scores, [MaxPlayers]int32{286, 270})
	is.NoErr(s.ValidateTileConservation(r.LetterDistribution))
}

// A tile play resets the counter, so a stalemate cannot creep up across plays.
func TestATilePlayResetsTheScorelessCounter(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	alph := r.LetterDistribution.TileMapping()
	is.NoErr(s.AssignRack(0, word(t, alph, "HELLOAB")))
	is.NoErr(s.AssignRack(1, word(t, alph, "AEIOUST")))

	for range 5 {
		_, err := s.ApplyPass(r)
		is.NoErr(err)
	}
	is.Equal(s.ScorelessTurns, uint8(5))
	is.Equal(s.OnTurn, uint8(1))

	// Player 1 passing would end it; instead player 1 plays across the centre.
	_, err := s.ApplyPlacement(r, seededRand(), NewPlacementMove(7, 5, false, word(t, alph, "AIT")))
	is.NoErr(err)
	is.Equal(s.ScorelessTurns, uint8(0))
	is.Equal(s.PlayState, Playing)
}

// A full game driven by random legal moves. Conservation and structural
// validity are checked after every single transition, which is the cheapest
// place to catch a position going wrong.
func TestRandomGamesStayConsistent(t *testing.T) {
	is := is.New(t)
	ld := testLetterDistribution(t, "english")
	// DOUBLE rather than VOID so plays are not filtered by the lexicon: this
	// exercises the state machine, not the word list.
	r := testRules(t, ChallengeRuleDouble, ld)

	var maxLtr tilemapping.MachineLetter
	for ml, ct := range ld.Distribution() {
		if ct > 0 && ml > 0 {
			maxLtr = tilemapping.MachineLetter(ml)
		}
	}

	finished, totalMoves := 0, 0
	for game := range 60 {
		rng := rand.New(rand.NewPCG(uint64(game), 99))
		s, err := NewState(r.Layout.Dim())
		is.NoErr(err)
		is.NoErr(s.FillBag(ld))
		for p := range MaxPlayers {
			_, err := s.DrawToFull(rng, p)
			is.NoErr(err)
		}

		for move := 0; move < 300 && s.PlayState != GameOver; move++ {
			m := randomLegalMove(s, r, rng, maxLtr)
			res, err := s.ApplyMove(r, rng, m)
			if err != nil {
				t.Fatalf("game %d move %d: %v (move %+v)", game, move, err, m)
			}
			totalMoves++

			if err := s.Validate(); err != nil {
				t.Fatalf("game %d move %d: %v", game, move, err)
			}
			if err := s.ValidateTileConservation(ld); err != nil {
				t.Fatalf("game %d move %d: %v", game, move, err)
			}
			if s.PlayState == GameOver && !res.GameOver {
				t.Fatalf("game %d move %d: state says over, result does not", game, move)
			}
			// A clone of any reachable position must be indistinguishable
			// from it. That covers Clone's deep copies and the digest at the
			// same time, at every position a real game passes through.
			if c := s.Clone(); !c.Equal(s) {
				t.Fatalf("game %d move %d: clone differs from its original", game, move)
			}
		}
		if s.PlayState == GameOver {
			finished++
		}
	}
	t.Logf("%d/60 games played to completion over %d moves", finished, totalMoves)
	// The endgame paths have to actually be reached for this to mean anything.
	is.True(finished > 20)
}

// randomLegalMove returns something the player on turn may legally do. It
// prefers a tile play, falls back to an exchange, and passes when neither
// works -- so a game always makes progress.
func randomLegalMove(s *State, r *Rules, rng *rand.Rand, maxLtr tilemapping.MachineLetter) *Move {
	if s.PlayState == WaitingForFinalPass {
		return NewPassMove()
	}
	if rng.IntN(12) > 0 {
		for range 10 {
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
			m := NewExchangeMove(append(tilemapping.MachineWord(nil), s.Rack(p)[:1+rng.IntN(n)]...))
			if _, err := s.ValidateMove(r, m); err == nil {
				return m
			}
		}
	}
	return NewPassMove()
}

// randomPlacementFromRack builds a candidate play using only tiles the player
// actually holds. It may still be geometrically illegal; the caller checks.
func randomPlacementFromRack(s *State, rng *rand.Rand, maxLtr tilemapping.MachineLetter) *Move {
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

	tiles := make(tilemapping.MachineWord, 0, RackTileLimit+dim)
	used := 0
	for i := 0; used < want; i++ {
		rr, cc := row+dRow*i, col+dCol*i
		if !s.PosExists(rr, cc) {
			break
		}
		if s.HasLetter(rr, cc) {
			tiles = append(tiles, 0) // played through
			continue
		}
		ml := avail[used]
		used++
		if ml == 0 {
			// A blank has to be designated to go on the board.
			ml = tilemapping.MachineLetter(1+rng.IntN(int(maxLtr))) | tilemapping.BlankMask
		}
		tiles = append(tiles, ml)
	}
	if used == 0 {
		return nil
	}
	return NewPlacementMove(row, col, vertical, tiles)
}
