package xwordgame

import (
	"testing"

	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
)

// The unset value is -1, not player 0. Every path sets the field today, so
// this is unreachable in practice -- which is the point: a future transition
// that forgets should report "nobody" rather than silently charging player 0.
func TestUnattributedTimeIsNobody(t *testing.T) {
	is := is.New(t)
	res := newApplyResult()
	is.Equal(res.TimeAttributedTo, int8(-1))
	is.Equal(res.Winner, int8(-1))
	is.Equal(res.EndRackBonusPlayer, int8(-1))
}

func TestApplyTimePenalty(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	alph := r.LetterDistribution.TileMapping()
	is.NoErr(s.AssignRack(0, word(t, alph, "HELLOAB")))
	is.NoErr(s.AssignRack(1, word(t, alph, "AEIOUST")))
	s.Scores = [MaxPlayers]int32{300, 280}
	s.OnTurn = 1
	s.ScorelessTurns = 2

	res, err := s.ApplyTimePenalty(r, 1, 10)
	is.NoErr(err)
	is.Equal(res.Score, int32(-10))
	is.Equal(res.TimeAttributedTo, int8(1))
	is.Equal(s.Scores, [MaxPlayers]int32{300, 270})

	// A penalty is not a turn: the penalised player still owes a move, and
	// nothing else about the position moves either.
	is.Equal(s.OnTurn, uint8(1))
	is.Equal(s.TurnNum, uint16(0))
	is.Equal(s.ScorelessTurns, uint8(2))
	is.Equal(s.PlayState, Playing)
	is.True(!res.GameOver)
	is.NoErr(s.ValidateTileConservation(r.LetterDistribution))

	// Penalties accrue; each one is separate.
	_, err = s.ApplyTimePenalty(r, 1, 10)
	is.NoErr(err)
	is.Equal(s.Scores[1], int32(260))

	// The magnitude is expected, not a signed adjustment -- taking a negative
	// here would silently award points for running late.
	_, err = s.ApplyTimePenalty(r, 1, -10)
	is.True(err != nil)
	is.Equal(s.Scores[1], int32(260))

	_, err = s.ApplyTimePenalty(r, 5, 10)
	is.True(err != nil)
}

func TestApplyTimeout(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	alph := r.LetterDistribution.TileMapping()
	is.NoErr(s.AssignRack(0, word(t, alph, "QUIZ")))    // 22 points of tiles
	is.NoErr(s.AssignRack(1, word(t, alph, "AEIOUST"))) // 7
	s.Scores = [MaxPlayers]int32{300, 400}
	s.OnTurn = 1

	res, err := s.ApplyTimeout(r, 1)
	is.NoErr(err)
	is.True(res.GameOver)
	is.Equal(s.PlayState, GameOver)

	// The clock decides the winner, not the score: player 1 is ahead by a
	// hundred and still loses.
	is.Equal(res.Winner, int8(0))
	is.Equal(res.TimeAttributedTo, int8(1))

	// And it is settled on the clock, not the tiles -- neither rack is
	// deducted, unlike every other way a game can end here.
	is.Equal(s.Scores, [MaxPlayers]int32{300, 400})
	is.Equal(res.EndRackBonusPlayer, int8(-1))
	is.True(res.ScorelessPenalties == nil)
	is.NoErr(s.ValidateTileConservation(r.LetterDistribution))
}

func TestTimeoutClearsTheChallengeableePlay(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	alph := r.LetterDistribution.TileMapping()
	is.NoErr(s.AssignRack(0, word(t, alph, "HELLOAB")))

	_, err := s.ApplyPlacement(r, seededRand(), NewPlacementMove(7, 3, false, word(t, alph, "HELLO")))
	is.NoErr(err)
	is.True(len(s.LastWordsFormed) > 0)

	_, err = s.ApplyTimeout(r, 1)
	is.NoErr(err)
	is.Equal(len(s.LastWordsFormed), 0)
}

func TestTimeoutRefusesAFinishedGame(t *testing.T) {
	is := is.New(t)
	s, r := boardFixture(t, ChallengeRuleDouble)
	s.PlayState = GameOver
	_, err := s.ApplyTimeout(r, 0)
	is.Equal(err, errGameIsOver)
}

// Every move has to say whose clock it belongs to, because the caller can no
// longer infer it once the referee starts generating moves of its own.
func TestTimeIsAttributedToTheActingPlayer(t *testing.T) {
	is := is.New(t)

	t.Run("placement", func(t *testing.T) {
		s, r := boardFixture(t, ChallengeRuleDouble)
		alph := r.LetterDistribution.TileMapping()
		is.NoErr(s.AssignRack(0, word(t, alph, "HELLOAB")))
		res, err := s.ApplyPlacement(r, seededRand(), NewPlacementMove(7, 3, false, word(t, alph, "HELLO")))
		is.NoErr(err)
		is.Equal(res.TimeAttributedTo, int8(0))
		is.Equal(s.OnTurn, uint8(1)) // the turn moved on; the clock did not
	})

	t.Run("exchange", func(t *testing.T) {
		s, r := boardFixture(t, ChallengeRuleDouble)
		alph := r.LetterDistribution.TileMapping()
		s.OnTurn = 1
		is.NoErr(s.AssignRack(1, word(t, alph, "QUIZJXV")))
		res, err := s.ApplyExchange(r, seededRand(), NewExchangeMove(word(t, alph, "JXV")))
		is.NoErr(err)
		is.Equal(res.TimeAttributedTo, int8(1))
	})

	t.Run("pass", func(t *testing.T) {
		s, r := boardFixture(t, ChallengeRuleDouble)
		s.OnTurn = 1
		res, err := s.ApplyPass(r)
		is.NoErr(err)
		is.Equal(res.TimeAttributedTo, int8(1))
	})

	// The case the field exists for. Under the double rule the referee spends
	// the challenger's turn for them, and macondo's inability to say so is why
	// pkg/gameplay flips the player on turn twice to find out.
	t.Run("unsuccessful challenge turn loss", func(t *testing.T) {
		cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
		out, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleDouble, prev, ld))
		is.NoErr(err)
		is.True(out.TurnLoss)
		// Player 1 challenged and lost the turn, so player 1's clock ran --
		// even though player 1 is no longer the one on turn afterwards.
		is.Equal(out.TimeAttributedTo, int8(1))
		is.Equal(out.Challenger, uint8(1))
		is.Equal(cur.OnTurn, uint8(0))
	})

	// The replay entry points know whose clock ran too, and must say so -- a
	// caller replaying a log has no other way to attribute the time.
	t.Run("returned phony applied directly", func(t *testing.T) {
		cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
		r := testRules(t, ChallengeRuleSingle, ld)
		res, err := cur.ApplyReturnedPhony(r, prev)
		is.NoErr(err)
		is.Equal(res.TimeAttributedTo, int8(1)) // the challenger
		is.Equal(res.Score, int32(74))          // what came off the board
	})

	t.Run("challenge bonus applied directly", func(t *testing.T) {
		cur, _, ld := challengeFixture(t, "FRIENDS", 74)
		r := testRules(t, ChallengeRuleTenPoint, ld)
		// Player 0 was challenged and the challenge failed, so player 1's
		// clock ran even though player 0 collects the points.
		res, err := cur.ApplyChallengeBonus(r, 0, 10)
		is.NoErr(err)
		is.Equal(res.TimeAttributedTo, int8(1))
		is.Equal(cur.Scores[0], int32(120+74+10))
	})

	t.Run("successful challenge", func(t *testing.T) {
		cur, prev, ld := challengeFixture(t, "FRIENDS", 74)
		cur.LastWordsFormed = []tilemapping.MachineWord{word(t, ld.TileMapping(), "QZJXV")}
		out, err := cur.AdjudicateChallenge(baseParams(t, ChallengeRuleSingle, prev, ld))
		is.NoErr(err)
		is.True(out.PhonyReturned)
		is.Equal(out.TimeAttributedTo, int8(1))
	})
}
