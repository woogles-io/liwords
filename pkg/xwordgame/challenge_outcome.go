package xwordgame

// Applying a challenge outcome that has already been decided.
//
// AdjudicateChallenge answers "is this play legal, and what follows?". These
// two answer only the second half, for a caller that already knows the verdict.
//
// That split exists because replaying a historical game must not re-run the
// lexicon. Woogles games are played under a lexicon that changes over time -- a
// word valid in CSW21 may be gone from CSW24, and vice versa -- so re-deciding
// an old challenge would produce a position the game never actually reached.
// The log is authoritative for what happened; this package is authoritative for
// what that does to the position. These functions are the seam between the two.
//
// AdjudicateChallenge is implemented on top of them, so there is one copy of
// each transition rather than a live path and a replay path that can drift.

import "fmt"

// ApplyReturnedPhony takes a challenged play back off the board, restoring the
// position given in prev, and leaves the challenger on turn.
//
// prev must be the position immediately before the challenged play; it is
// checked against the current turn number and player on turn, because silently
// restoring an unrelated position is the failure mode this design exists to
// prevent.
func (s *State) ApplyReturnedPhony(r *Rules, prev *State) (*ApplyResult, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	tmp := &ChallengeOutcome{Challengee: otherPlayer(s.OnTurn)}
	if err := s.returnPhony(prev, tmp); err != nil {
		return nil, err
	}
	// The challengee's forfeited turn counts as scoreless, on top of whatever
	// had accumulated before the phony.
	s.ScorelessTurns++
	res := newApplyResult()
	// Score carries what the challengee gave back, which is the one number a
	// caller needs that the position no longer shows.
	res.Score = tmp.LostScore
	s.endIfScorelessStalemate(r.LetterDistribution, res)
	s.LastWordsFormed = nil
	return res, nil
}

// ApplyChallengeBonus credits a challenged player for a challenge that failed,
// under one of the points rules. The challenger does not lose their turn, so
// whose turn it is does not change.
//
// If the challengee had already played out their rack, this is the last thing
// standing between them and the win, so the game ends here.
func (s *State) ApplyChallengeBonus(r *Rules, challengee int, bonus int32) (*ApplyResult, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	if challengee < 0 || challengee >= MaxPlayers {
		return nil, fmt.Errorf("xwordgame: player index %d out of range", challengee)
	}
	res := newApplyResult()
	s.Scores[challengee] += bonus

	if s.PlayState == WaitingForFinalPass {
		out, err := s.ApplyOutBonus(r.LetterDistribution, challengee)
		if err != nil {
			return nil, err
		}
		res.GameOver = true
		res.EndRackBonus = out
		res.EndRackBonusPlayer = int8(challengee)
	}
	s.LastWordsFormed = nil
	return res, nil
}
