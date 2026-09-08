// Challenge adjudication.
//
// There are two existing implementations of this in the tree and they disagree
// with each other:
//
//   - macondo's game.ChallengeEvent, which live liwords games reach through
//     pkg/gameplay/game.go:409.
//   - pkg/cwgame's challengeEvent, which the annotator uses.
//
// The differences are not cosmetic. cwgame lets a player challenge a subset of
// the words formed and pays the five-point bonus per word; macondo has no such
// concept. macondo carries the scoreless-turn counter through an undo; cwgame
// re-derives it by scanning the event log backwards -- the same
// derive-the-position-from-the-log pattern that corrupted 944 games. macondo
// reshuffles the bag after a phony comes off so the offender does not retain
// knowledge of the tiles they drew; cwgame does not.
//
// This implementation is meant to replace both. Three things are different by
// construction:
//
//  1. It adjudicates a position, not an event log. Nothing is re-derived.
//  2. It returns a description of what happened instead of writing events, so
//     the caller can attribute clocks and end reasons correctly the first time.
//  3. It needs no reshuffle. The bag is a multiset with no order, so there is
//     no ordering for a phony's drawn tiles to leak. The bug class is absent
//     rather than mitigated.
//
// Subset challenges are supported from the start, since that is the one place
// liwords is ahead of macondo and bolting it on later would mean revisiting
// every call site. The two engines' five-point scoring is reconciled by
// FivePointMode rather than by picking a winner: live games keep macondo's flat
// five, the annotator gets cwgame's per-word count, and the default is the
// behaviour liwords has today.

package xwordgame

import (
	"errors"
	"fmt"

	"github.com/domino14/word-golib/tilemapping"
)

// ChallengeRule is how a game settles challenges. The values match
// macondo's ChallengeRule enum so conversion at the boundary stays a cast.
type ChallengeRule uint8

const (
	// ChallengeRuleVoid disallows challenges entirely; invalid words are simply
	// rejected when played.
	ChallengeRuleVoid ChallengeRule = 0
	// ChallengeRuleSingle takes a phony off the board but costs a wrong
	// challenger nothing.
	ChallengeRuleSingle ChallengeRule = 1
	// ChallengeRuleDouble costs a wrong challenger their turn.
	ChallengeRuleDouble ChallengeRule = 2
	// ChallengeRuleFivePoint pays the challenged player five points for a wrong
	// challenge.
	ChallengeRuleFivePoint ChallengeRule = 3
	// ChallengeRuleTenPoint pays the challenged player ten points for a wrong
	// challenge.
	ChallengeRuleTenPoint ChallengeRule = 4
	// ChallengeRuleTriple ends the game: whoever is wrong loses outright.
	ChallengeRuleTriple ChallengeRule = 5
)

func (c ChallengeRule) String() string {
	switch c {
	case ChallengeRuleVoid:
		return "VOID"
	case ChallengeRuleSingle:
		return "SINGLE"
	case ChallengeRuleDouble:
		return "DOUBLE"
	case ChallengeRuleFivePoint:
		return "FIVE_POINT"
	case ChallengeRuleTenPoint:
		return "TEN_POINT"
	case ChallengeRuleTriple:
		return "TRIPLE"
	}
	return fmt.Sprintf("ChallengeRule(%d)", uint8(c))
}

// Variant selects the word-validity rule. The string values match macondo's.
type Variant string

const (
	VarClassic       Variant = "classic"
	VarWordSmog      Variant = "wordsmog"
	VarClassicSuper  Variant = "classic_super"
	VarWordSmogSuper Variant = "wordsmog_super"
)

// MatchesAnagrams reports whether the variant accepts any anagram of a valid
// word rather than requiring the exact spelling.
func (v Variant) MatchesAnagrams() bool {
	return v == VarWordSmog || v == VarWordSmogSuper
}

// FivePointMode selects how the five-point rule pays a wrong challenger. It is
// the one place the two implementations this package replaces disagree on
// scoring, so it is an explicit setting rather than something inferred.
//
// It applies to ChallengeRuleFivePoint and nothing else. Single, ten-point,
// double and triple always treat the play as a single unit and pay at most
// once, however many words were challenged.
type FivePointMode uint8

const (
	// FivePointPerPlay pays a flat five points however many words the play
	// formed or the challenger named. This is macondo's behaviour and what
	// live liwords games have always done, so it is the zero value: a caller
	// that says nothing keeps the behaviour it has today.
	FivePointPerPlay FivePointMode = 0
	// FivePointPerWord pays five points for every word challenged. This is
	// what pkg/cwgame does and what the annotator needs.
	//
	// It requires ChallengeParams.WordIndices to name the words. "Per word,
	// but the caller did not say which words" has two plausible readings --
	// every word formed, or cwgame's flat five -- so it is refused rather than
	// guessed at. Callers that mean every word say so by listing them all.
	//
	// There is no "per good word" variant, because the two cannot differ: the
	// bonus is only ever paid when the challenge fails, and a challenge only
	// fails when every word it named was valid.
	FivePointPerWord FivePointMode = 1
)

// LegacyFivePointMode maps pkg/cwgame's wire format onto an explicit mode.
//
// cwgame has no mode field. It infers per-word scoring from the presence of
// challenged-word indices, so an event naming no words pays a flat five while
// one naming every word pays five apiece -- the same action, scored two ways
// (pkg/cwgame/game.go:633, commented there as "Legacy behavior"). Reproducing
// that inference is necessary to replay historical annotated games without
// rescoring them, but it is a property of the old wire format, not of the
// rules, so it lives here in one named place rather than inside the engine.
//
// New callers should set the mode explicitly instead. Live liwords games use
// FivePointPerPlay and never send indices at all.
func LegacyFivePointMode(indices []uint32) FivePointMode {
	if len(indices) > 0 {
		return FivePointPerWord
	}
	return FivePointPerPlay
}

func (f FivePointMode) String() string {
	switch f {
	case FivePointPerPlay:
		return "per-play"
	case FivePointPerWord:
		return "per-word"
	}
	return fmt.Sprintf("FivePointMode(%d)", uint8(f))
}

// Lexicon is the word-validity oracle. It is declared here rather than imported
// so this package keeps its only dependency on word-golib; macondo's
// lexicon.Lexicon satisfies it as-is.
type Lexicon interface {
	HasWord(word tilemapping.MachineWord) bool
	HasAnagram(word tilemapping.MachineWord) bool
}

var (
	errChallengeInVoid    = errors.New("xwordgame: challenges are not valid under the void rule")
	errNoWordsToChallenge = errors.New("xwordgame: there are no words to challenge")
	// ErrPerWordNeedsIndices is returned when per-word five-point scoring is
	// asked for without naming the words it should count.
	ErrPerWordNeedsIndices = errors.New("xwordgame: per-word five-point scoring requires ChallengeParams.WordIndices; list every word to challenge the whole play")
	errNoPrevPosition      = errors.New("xwordgame: returning a phony needs the position from before the challenged play; set ChallengeParams.Prev")
)

// ChallengeParams are the inputs to adjudicating a challenge. The player on turn
// is the challenger.
type ChallengeParams struct {
	// Rules is the game's configuration; its Lexicon is required here.
	Rules *Rules

	// Prev is the position immediately before the challenged play. It is
	// required only when the play might come off the board -- that is, whenever
	// the challenge could succeed -- and supplying it is what makes the undo
	// exact instead of reconstructed. Restoring it recovers the board, the
	// challenged player's original rack, the tiles they drew and the
	// scoreless-turn counter in one step, with no event-log replay.
	Prev *State

	// WordIndices optionally restricts the challenge to a subset of
	// State.LastWordsFormed, by index. Empty means challenge every word.
	//
	// This is independent of FivePointMode: WordIndices decides what gets
	// validated, FivePointMode decides how the bonus is counted. Conflating
	// the two -- inferring per-word scoring from the presence of indices --
	// would mean that naming every word scored differently from naming none,
	// for the same action. See LegacyFivePointMode for the one place that
	// inference is still reproduced, and why.
	WordIndices []uint32
}

// ChallengeOutcome describes what a challenge did. The State has already been
// updated; this is what the caller needs to write events and tell the players.
type ChallengeOutcome struct {
	// Challenger is the player who challenged; Challengee played the challenged
	// word.
	Challenger uint8
	Challengee uint8

	// PlayLegal reports whether every challenged word was valid.
	PlayLegal bool
	// ChallengedWords is the subset actually validated, and IllegalWords is
	// whichever of those failed.
	ChallengedWords []tilemapping.MachineWord
	IllegalWords    []tilemapping.MachineWord

	// PhonyReturned reports that the play came off the board, and LostScore is
	// what the challengee gave back.
	PhonyReturned bool
	LostScore     int32

	// BonusPoints is what the challengee gained from a wrong challenge, and
	// TurnLoss reports that the challenger forfeited their turn instead.
	BonusPoints int32
	TurnLoss    bool

	// GameOver reports whether the challenge ended the game.
	GameOver bool
	// TripleChallenge marks the triple-challenge ending specifically, which the
	// caller reports as its own end reason.
	TripleChallenge bool
	// Winner is set only when the challenge itself decided the game, as it does
	// under the triple rule. It is -1 otherwise, including for games that ended
	// here on score.
	Winner int8

	// EndRackBonus, EndRackBonusPlayer and ScorelessPenalties carry the same
	// end-of-game adjustments an ApplyResult does, for the cases where a
	// challenge finishes the game.
	EndRackBonus       int32
	EndRackBonusPlayer int8
	ScorelessPenalties *[MaxPlayers]int32

	// TimeAttributedTo is the player whose clock this challenge should be
	// charged to: the challenger, in every case, including the DOUBLE rule
	// where the referee spends their turn for them.
	//
	// This is what pkg/gameplay/game.go:437 currently works out by flipping the
	// player on turn, recording the time and flipping back, because macondo
	// synthesises the turn loss without knowing clocks exist.
	TimeAttributedTo int8
}

// AdjudicateChallenge settles a challenge by the player on turn against the
// previous play, updating the State and describing what changed.
func (s *State) AdjudicateChallenge(p ChallengeParams) (*ChallengeOutcome, error) {
	if err := p.Rules.validate(); err != nil {
		return nil, err
	}
	if p.Rules.ChallengeRule == ChallengeRuleVoid {
		return nil, errChallengeInVoid
	}
	if s.PlayState == GameOver {
		return nil, errGameIsOver
	}
	if p.Rules.Lexicon == nil {
		return nil, fmt.Errorf("xwordgame: a lexicon is required to adjudicate a challenge")
	}
	if len(s.LastWordsFormed) == 0 {
		return nil, errNoWordsToChallenge
	}
	// Only enforced where the mode has an effect, so a caller that carries one
	// mode for a whole session is not tripped up by rules that ignore it.
	if p.Rules.ChallengeRule == ChallengeRuleFivePoint && p.Rules.FivePointMode == FivePointPerWord && len(p.WordIndices) == 0 {
		return nil, ErrPerWordNeedsIndices
	}

	words, err := selectChallengedWords(s.LastWordsFormed, p.WordIndices)
	if err != nil {
		return nil, err
	}
	illegal := invalidWords(p.Rules.Lexicon, words, p.Rules.Variant)

	out := &ChallengeOutcome{
		Challenger:         s.OnTurn,
		Challengee:         otherPlayer(s.OnTurn),
		TimeAttributedTo:   int8(s.OnTurn),
		PlayLegal:          len(illegal) == 0,
		ChallengedWords:    words,
		IllegalWords:       illegal,
		Winner:             -1,
		EndRackBonusPlayer: -1,
	}

	switch {
	case p.Rules.ChallengeRule == ChallengeRuleTriple:
		// Somebody always loses outright.
		out.TripleChallenge = true
		out.GameOver = true
		if out.PlayLegal {
			out.Winner = int8(out.Challengee)
		} else {
			out.Winner = int8(out.Challenger)
			if err := s.returnPhony(p.Prev, out); err != nil {
				return nil, err
			}
		}
		s.PlayState = GameOver

	case !out.PlayLegal:
		// Successful challenge: the play comes off and the challenger keeps the
		// turn they were already on. Shared with the replay path so there is
		// one copy of the transition.
		res, err := s.ApplyReturnedPhony(p.Rules, p.Prev)
		if err != nil {
			return nil, err
		}
		out.PhonyReturned = true
		out.LostScore = res.Score
		if res.GameOver {
			out.GameOver = true
			out.ScorelessPenalties = res.ScorelessPenalties
		}

	case p.Rules.ChallengeRule == ChallengeRuleDouble:
		// Unsuccessful challenge, draconian variety: the challenger loses their
		// turn. Applying it as a pass is what handles the case where the
		// challengee had already gone out -- the pass closes the game and pays
		// them the end-rack bonus -- and the case where this is the sixth
		// scoreless turn.
		out.TurnLoss = true
		res, err := s.ApplyPass(p.Rules)
		if err != nil {
			return nil, err
		}
		out.GameOver = res.GameOver
		out.EndRackBonus = res.EndRackBonus
		out.EndRackBonusPlayer = res.EndRackBonusPlayer
		out.ScorelessPenalties = res.ScorelessPenalties

	default:
		// Unsuccessful challenge under a points rule. The challenger does not
		// lose their turn; they still have to move.
		switch p.Rules.ChallengeRule {
		case ChallengeRuleSingle:
			out.BonusPoints = 0
		case ChallengeRuleFivePoint:
			// The only rule whose payout depends on the word count, and only
			// when the caller asks for it.
			out.BonusPoints = 5
			if p.Rules.FivePointMode == FivePointPerWord {
				out.BonusPoints = int32(5 * len(words))
			}
		case ChallengeRuleTenPoint:
			out.BonusPoints = 10
		default:
			return nil, fmt.Errorf("xwordgame: unknown challenge rule %d", p.Rules.ChallengeRule)
		}
		res, err := s.ApplyChallengeBonus(p.Rules, int(out.Challengee), out.BonusPoints)
		if err != nil {
			return nil, err
		}
		out.GameOver = res.GameOver
		out.EndRackBonus = res.EndRackBonus
		out.EndRackBonusPlayer = res.EndRackBonusPlayer
	}

	// The words are spent either way: they cannot be challenged twice.
	s.LastWordsFormed = nil
	return out, nil
}

// returnPhony rolls the position back to before the challenged play, leaving the
// challenger on turn.
func (s *State) returnPhony(prev *State, out *ChallengeOutcome) error {
	if prev == nil {
		return errNoPrevPosition
	}
	if prev.dim != s.dim {
		return fmt.Errorf("xwordgame: previous position is %dx%d, current is %dx%d",
			prev.dim, prev.dim, s.dim, s.dim)
	}
	// Guard against being handed the wrong snapshot. Exactly one play separates
	// the two positions, and it was the challengee's. Silently restoring an
	// unrelated position is the failure mode this whole design exists to
	// prevent, so it is worth two comparisons.
	if prev.TurnNum+1 != s.TurnNum {
		return fmt.Errorf("xwordgame: previous position is at turn %d, expected %d",
			prev.TurnNum, s.TurnNum-1)
	}
	if prev.OnTurn != out.Challengee {
		return fmt.Errorf("xwordgame: previous position has player %d on turn, expected the challenged player %d",
			prev.OnTurn, out.Challengee)
	}

	out.PhonyReturned = true
	out.LostScore = s.Scores[out.Challengee] - prev.Scores[out.Challengee]

	challenger := s.OnTurn
	s.restorePosition(prev)
	// The challenge does not pass the turn: the challenger was on turn before
	// they challenged and they are on turn after.
	s.OnTurn = challenger
	return nil
}

// restorePosition copies the positional fields of prev into s, leaving OnTurn
// and LastWordsFormed to the caller.
func (s *State) restorePosition(prev *State) {
	s.board = prev.board
	s.racks = prev.racks
	s.rackLens = prev.rackLens
	s.bag = prev.bag
	s.Scores = prev.Scores
	s.Bingos = prev.Bingos
	s.PlayerTurns = prev.PlayerTurns
	s.TurnNum = prev.TurnNum
	s.ScorelessTurns = prev.ScorelessTurns
	s.PlayState = prev.PlayState
}

// selectChallengedWords picks out the words a challenge covers. An empty index
// list means all of them.
func selectChallengedWords(formed []tilemapping.MachineWord, indices []uint32) ([]tilemapping.MachineWord, error) {
	if len(indices) == 0 {
		return formed, nil
	}
	seen := make(map[uint32]bool, len(indices))
	words := make([]tilemapping.MachineWord, 0, len(indices))
	for _, idx := range indices {
		if int(idx) >= len(formed) {
			return nil, fmt.Errorf("xwordgame: challenged word index %d, but the play formed %d words",
				idx, len(formed))
		}
		if seen[idx] {
			// Otherwise a client could inflate the per-word bonus by repeating
			// an index.
			return nil, fmt.Errorf("xwordgame: challenged word index %d more than once", idx)
		}
		seen[idx] = true
		words = append(words, formed[idx])
	}
	return words, nil
}

// invalidWords returns whichever of the given words the lexicon rejects.
func invalidWords(lex Lexicon, words []tilemapping.MachineWord, variant Variant) []tilemapping.MachineWord {
	var bad []tilemapping.MachineWord
	anagrams := variant.MatchesAnagrams()
	for _, w := range words {
		ok := lex.HasWord(w)
		if anagrams {
			ok = lex.HasAnagram(w)
		}
		if !ok {
			bad = append(bad, w)
		}
	}
	return bad
}
