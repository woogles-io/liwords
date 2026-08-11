// Applying moves to a state.
//
// This is the state machine: the part that decides what a position becomes
// after a player acts. It is deliberately separate from scoring and legality
// (score.go, board.go), which only ask questions and never mutate.
//
// Every Apply function reports what it did through an ApplyResult rather than
// writing events itself. macondo mutates a GameHistory in place, which is why
// liwords has to patch the result afterwards -- reassigning whose clock ran
// during an unsuccessful-challenge turn loss (pkg/gameplay/game.go:437), and
// setting the triple-challenge end reason macondo cannot know about
// (pkg/gameplay/game.go:469). Returning a description instead lets the caller
// build its events correctly the first time.

package xwordgame

import (
	"errors"
	"fmt"

	"github.com/domino14/word-golib/tilemapping"
)

// MaxScorelessTurns is the number of consecutive scoreless turns that ends a
// game. Six is standard across every variant we support.
const MaxScorelessTurns = 6

var (
	errGameIsOver = errors.New("xwordgame: the game is over")
)

// ApplyResult describes the side effects of applying a move: everything the
// caller needs to write the corresponding events and notify players. The state
// itself has already been updated.
type ApplyResult struct {
	// GameOver reports whether this move ended the game.
	GameOver bool

	// EndRackBonus is the two-times-the-opponent's-rack bonus awarded to the
	// player who went out, and EndRackBonusPlayer is who received it.
	// EndRackBonusPlayer is -1 when no bonus was awarded.
	EndRackBonus       int32
	EndRackBonusPlayer int8

	// ScorelessPenalties is set when the six-scoreless-turns rule ended the
	// game, holding the points deducted from each player for their own
	// remaining rack. It is nil otherwise.
	ScorelessPenalties *[MaxPlayers]int32
}

func newApplyResult() *ApplyResult {
	return &ApplyResult{EndRackBonusPlayer: -1}
}

// ApplyPass applies a pass by the player on turn.
//
// A pass is also how an unsuccessful challenge under the DOUBLE rule takes
// effect: the state transition is identical and only the event the caller
// writes differs. That is how macondo models it too -- MoveTypePass and
// MoveTypeUnsuccessfulChallengePass share a branch in playMove.
func (s *State) ApplyPass(ld *tilemapping.LetterDistribution) (*ApplyResult, error) {
	if s.PlayState == GameOver {
		return nil, errGameIsOver
	}
	if ld == nil {
		return nil, fmt.Errorf("xwordgame: a letter distribution is required to apply a pass")
	}
	res := newApplyResult()

	if s.PlayState == WaitingForFinalPass {
		// The opponent played out their rack and this pass confirms it. The
		// bonus goes to the player who went out, not to the one passing: this
		// pass is a virtual turn that exists only to close the game.
		out := int(otherPlayer(s.OnTurn))
		bonus, err := s.ApplyOutBonus(ld, out)
		if err != nil {
			return nil, err
		}
		res.GameOver = true
		res.EndRackBonus = bonus
		res.EndRackBonusPlayer = int8(out)
		s.TurnNum++
		s.LastWordsFormed = nil
		return res, nil
	}

	s.ScorelessTurns++
	s.PlayerTurns[s.OnTurn]++
	if !s.endIfScorelessStalemate(ld, res) {
		s.OnTurn = otherPlayer(s.OnTurn)
	}
	s.TurnNum++
	s.LastWordsFormed = nil
	return res, nil
}

// endIfScorelessStalemate ends the game if the scoreless-turn counter has just
// reached the limit, and reports whether it did.
//
// The comparison is exact rather than >=, matching macondo. Every path that
// touches the counter moves it by one, so it cannot step over the limit; an
// inexact comparison would only paper over a bug elsewhere.
func (s *State) endIfScorelessStalemate(ld *tilemapping.LetterDistribution, res *ApplyResult) bool {
	if s.ScorelessTurns != MaxScorelessTurns {
		return false
	}
	penalties := s.ApplyScorelessPenalties(ld)
	res.GameOver = true
	res.ScorelessPenalties = &penalties
	return true
}
