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
//
// The order of operations follows macondo's playMove (game/game.go) closely,
// because the two have to agree tile for tile during the shadow-compare phase.
// Where this deviates, the reason is in a comment.

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
	errGameIsOver          = errors.New("xwordgame: the game is over")
	errOnlyPassOrChallenge = errors.New("xwordgame: you can only pass or challenge")
)

// InvalidWordsError reports a tile play rejected under the void challenge rule,
// carrying the words that failed.
//
// pkg/gameplay currently recovers this information by calling FormedWords again
// after ValidateMove has failed and inferring the case from the pair of results
// (game.go:511). A typed error makes that unnecessary. Rendering the words for
// humans needs an alphabet, which is the caller's to supply.
type InvalidWordsError struct {
	Words []tilemapping.MachineWord
}

func (e *InvalidWordsError) Error() string {
	return fmt.Sprintf("xwordgame: the play formed %d invalid word(s)", len(e.Words))
}

// ApplyResult describes the side effects of applying a move: everything the
// caller needs to write the corresponding events and notify players. The state
// itself has already been updated.
type ApplyResult struct {
	// Score is what the move earned, and WordsFormed is what it spelled. Both
	// are zero-valued for anything but a tile play.
	Score       int32
	WordsFormed []tilemapping.MachineWord
	// Bingo reports that the play used a full rack.
	Bingo bool
	// Drew is the tiles taken from the bag to replenish the rack.
	Drew []tilemapping.MachineLetter
	// WentOut reports that the player played their last tile. Under a non-void
	// challenge rule that means the game is waiting for a final pass rather
	// than over.
	WentOut bool

	// GameOver reports whether this move ended the game.
	GameOver bool
	// Winner is set only when the ending itself decided who won, rather than
	// the score deciding it -- a timeout, currently. It is -1 otherwise.
	Winner int8

	// TimeAttributedTo is the player whose clock this move should be charged
	// to, or -1 if none.
	//
	// It is nearly always the player who was on turn, and saying so explicitly
	// is what makes the referee-generated moves safe. macondo synthesises an
	// unsuccessful-challenge turn loss with no notion of a clock, which is why
	// pkg/gameplay/game.go:437 has to flip the player on turn, record the time
	// and flip back. A caller reading this field does not have to.
	TimeAttributedTo int8

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
	return &ApplyResult{EndRackBonusPlayer: -1, Winner: -1, TimeAttributedTo: -1}
}

// ValidateMove reports whether the player on turn may make this move, and for a
// tile play returns the words it would form. It does not mutate the state.
//
// A challenge is not validated here: settling one needs the position from
// before the challenged play, so it goes through AdjudicateChallenge.
func (s *State) ValidateMove(r *Rules, m *Move) ([]tilemapping.MachineWord, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	if s.PlayState == GameOver {
		return nil, errGameIsOver
	}
	switch m.Type {
	case MoveTypePass:
		// Always legal, and the only way out of WaitingForFinalPass besides a
		// challenge.
		return nil, nil

	case MoveTypeExchange:
		if s.PlayState == WaitingForFinalPass {
			return nil, errOnlyPassOrChallenge
		}
		if len(m.Tiles) == 0 {
			return nil, errors.New("xwordgame: an exchange must name at least one tile")
		}
		if len(m.Tiles) > RackTileLimit {
			return nil, fmt.Errorf("xwordgame: cannot exchange %d tiles, a rack holds %d",
				len(m.Tiles), RackTileLimit)
		}
		if remaining := s.TilesRemaining(); remaining < r.exchangeLimit() {
			return nil, fmt.Errorf("xwordgame: cannot exchange with fewer than %d tiles in the bag, there are %d",
				r.exchangeLimit(), remaining)
		}
		if !s.RackHas(int(s.OnTurn), m.Tiles) {
			return nil, fmt.Errorf("xwordgame: your exchange contained a tile that is not on your rack")
		}
		return nil, nil

	case MoveTypePlay:
		if s.PlayState == WaitingForFinalPass {
			return nil, errOnlyPassOrChallenge
		}
		played := m.TilesPlayed()
		if played > RackTileLimit {
			return nil, fmt.Errorf("xwordgame: your play contained %d tiles, a rack holds %d",
				played, RackTileLimit)
		}
		var buf [RackTileLimit]tilemapping.MachineLetter
		if played <= RackTileLimit && !s.RackHas(int(s.OnTurn), m.PlayedTiles(buf[:0])) {
			return nil, fmt.Errorf("xwordgame: your play contained a tile that is not on your rack")
		}
		if err := s.errorIfIllegalPlay(r.Layout, m, r.TrustRecordedPlays); err != nil {
			return nil, err
		}
		words, err := s.FormedWords(m)
		if err != nil {
			return nil, err
		}
		if r.ChallengeRule == ChallengeRuleVoid && !r.TrustRecordedPlays {
			// Under void there is no challenge to catch a phony, so the
			// referee refuses it up front.
			if r.Lexicon == nil {
				return nil, errors.New("xwordgame: the void challenge rule needs a lexicon to validate plays")
			}
			if bad := invalidWords(r.Lexicon, words, r.Variant); len(bad) > 0 {
				return nil, &InvalidWordsError{Words: bad}
			}
		}
		return words, nil
	}
	return nil, fmt.Errorf("xwordgame: %s is not a move a player can submit", m.Type)
}

// ApplyMove validates and applies a move by the player on turn.
//
// Passing a nil Rand uses DefaultRand. Challenges are refused here; see
// AdjudicateChallenge.
func (s *State) ApplyMove(r *Rules, rng Rand, m *Move) (*ApplyResult, error) {
	switch m.Type {
	case MoveTypePlay:
		return s.ApplyPlacement(r, rng, m)
	case MoveTypeExchange:
		return s.ApplyExchange(r, rng, m)
	case MoveTypePass:
		return s.ApplyPass(r)
	case MoveTypeChallenge:
		return nil, errors.New("xwordgame: settle a challenge with AdjudicateChallenge, which needs the previous position")
	}
	return nil, fmt.Errorf("xwordgame: %s is not a move a player can submit", m.Type)
}

// ApplyPlacement plays tiles on the board for the player on turn.
func (s *State) ApplyPlacement(r *Rules, rng Rand, m *Move) (*ApplyResult, error) {
	if m.Type != MoveTypePlay {
		return nil, fmt.Errorf("xwordgame: expected a tile placement, got %s", m.Type)
	}
	words, err := s.ValidateMove(r, m)
	if err != nil {
		return nil, err
	}
	score, err := s.ScoreMove(r.Layout, r.LetterDistribution, m)
	if err != nil {
		return nil, err
	}

	p := int(s.OnTurn)
	res := newApplyResult()
	res.Score = int32(score)
	res.WordsFormed = words
	res.TimeAttributedTo = int8(p)

	s.PlaceMoveTiles(m)
	// Only the fresh tiles come off the rack. m.Tiles is the play's whole span,
	// including zero markers for squares that were already covered; handing
	// those to TakeFromRack would ask the player to give up a blank per
	// played-through square.
	//
	// Validated above, so this cannot fail; checking keeps a future change from
	// silently desynchronising the rack from the board.
	var played [RackTileLimit]tilemapping.MachineLetter
	if err := s.TakeFromRack(p, m.PlayedTiles(played[:0])); err != nil {
		return nil, err
	}

	// Replenish. macondo draws at most TilesPlayed rather than filling the rack
	// (game.go, playMove). The two agree whenever the rack was full, which it
	// always is while the bag still has tiles: once a draw comes up short the
	// bag is empty and neither draws anything. Matching macondo keeps the
	// shadow comparison quiet.
	var buf [RackTileLimit]tilemapping.MachineLetter
	drawn, n := s.DrawAtMost(rng, m.TilesPlayed(), buf[:0])
	if n > 0 {
		if err := s.AddToRack(p, drawn); err != nil {
			return nil, err
		}
		res.Drew = append([]tilemapping.MachineLetter(nil), drawn...)
	}

	s.Scores[p] += res.Score
	s.PlayerTurns[p]++
	if m.TilesPlayed() == RackTileLimit {
		s.Bingos[p]++
		res.Bingo = true
	}
	// A tile play always resets the counter, even when it scores nothing: no
	// international rule treats tiles going onto the board as a scoreless turn.
	// Because of that the six-scoreless-turns rule cannot fire here, which is
	// why this path does not check it.
	s.ScorelessTurns = 0

	// Set before the endgame check below, not after: going out under the void
	// rule ends the game there and clearing the challengeable play is part of
	// ending it. Assigning afterwards would put it straight back.
	s.LastWordsFormed = words

	if s.rackLens[p] == 0 {
		res.WentOut = true
		if r.ChallengeRule != ChallengeRuleVoid {
			// The opponent still gets to pass or challenge, so the game is not
			// over yet. This is the state with no corresponding event, and
			// losing it is what corrupted 944 games.
			s.PlayState = WaitingForFinalPass
		} else {
			bonus, err := s.ApplyOutBonus(r.LetterDistribution, p)
			if err != nil {
				return nil, err
			}
			res.GameOver = true
			res.EndRackBonus = bonus
			res.EndRackBonusPlayer = int8(p)
		}
	}

	// The turn passes even when the game just ended, matching macondo. For
	// WaitingForFinalPass it has to: the opponent is the one who must act.
	s.OnTurn = otherPlayer(s.OnTurn)
	s.TurnNum++
	return res, nil
}

// ApplyExchange swaps tiles from the rack of the player on turn for fresh ones.
func (s *State) ApplyExchange(r *Rules, rng Rand, m *Move) (*ApplyResult, error) {
	if m.Type != MoveTypeExchange {
		return nil, fmt.Errorf("xwordgame: expected an exchange, got %s", m.Type)
	}
	if _, err := s.ValidateMove(r, m); err != nil {
		return nil, err
	}

	p := int(s.OnTurn)
	res := newApplyResult()
	res.TimeAttributedTo = int8(p)

	// Copy the tiles before touching the rack. A caller can reasonably build an
	// exchange straight from State.Rack, whose slice aliases the rack storage;
	// taking the tiles off rewrites that storage, and the discards handed to
	// the bag afterwards would be whatever landed in those slots. The tiles
	// returned would not be the tiles exchanged, and the count would still
	// balance, so only a conservation check would notice.
	var discards [RackTileLimit]tilemapping.MachineLetter
	n := copy(discards[:], m.Tiles)
	exchanged := discards[:n]

	// Off the rack first, then draw, then the discards go back -- so a player
	// cannot draw back what they just threw away. See State.Exchange.
	if err := s.TakeFromRack(p, exchanged); err != nil {
		return nil, err
	}
	var buf [RackTileLimit]tilemapping.MachineLetter
	drawn, err := s.Exchange(rng, exchanged, buf[:0])
	if err != nil {
		return nil, err
	}
	if err := s.AddToRack(p, drawn); err != nil {
		return nil, err
	}
	res.Drew = append([]tilemapping.MachineLetter(nil), drawn...)

	s.ScorelessTurns++
	s.PlayerTurns[p]++
	if !s.endIfScorelessStalemate(r.LetterDistribution, res) {
		s.OnTurn = otherPlayer(s.OnTurn)
	}
	s.TurnNum++
	s.LastWordsFormed = nil
	return res, nil
}

// ApplyPass applies a pass by the player on turn.
//
// A pass is also how an unsuccessful challenge under the DOUBLE rule takes
// effect: the state transition is identical and only the event the caller
// writes differs. That is how macondo models it too -- MoveTypePass and
// MoveTypeUnsuccessfulChallengePass share a branch in playMove.
func (s *State) ApplyPass(r *Rules) (*ApplyResult, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	if s.PlayState == GameOver {
		return nil, errGameIsOver
	}
	res := newApplyResult()
	res.TimeAttributedTo = int8(s.OnTurn)

	if s.PlayState == WaitingForFinalPass {
		// The opponent played out their rack and this pass confirms it. The
		// bonus goes to the player who went out, not to the one passing: this
		// pass is a virtual turn that exists only to close the game.
		out := int(otherPlayer(s.OnTurn))
		bonus, err := s.ApplyOutBonus(r.LetterDistribution, out)
		if err != nil {
			return nil, err
		}
		res.GameOver = true
		res.EndRackBonus = bonus
		res.EndRackBonusPlayer = int8(out)
		// The turn still passes. Whose turn it is means nothing once the game
		// is over, but macondo flips here -- its playMove flips unconditionally
		// unless the scoreless rule ended the game -- and ApplyScorelessPenalties
		// already matches that. Agreeing keeps the shadow comparison quiet.
		s.OnTurn = otherPlayer(s.OnTurn)
		s.TurnNum++
		s.LastWordsFormed = nil
		return res, nil
	}

	s.ScorelessTurns++
	s.PlayerTurns[s.OnTurn]++
	if !s.endIfScorelessStalemate(r.LetterDistribution, res) {
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
