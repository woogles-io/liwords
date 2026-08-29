package xwordgame

// What a clock does to a position.
//
// This package deliberately does not model time. Timers, increments, overtime
// allowances, time banks and correspondence deadlines are wall-clock IO and
// Woogles policy; they live in liwords, where the Nower and the timer state
// already are, and keeping them there is what lets everything here stay pure
// and deterministic. macondo has no clock either, and analysis and the
// annotator have no use for one.
//
// But time does reach into the position in three ways, and those belong here,
// because they change the very things a snapshot is authoritative for:
//
//   - an overtime penalty changes a score,
//   - running out of time ends a game and decides who won,
//   - and a move the referee generates has to be charged to somebody's clock.
//
// The third is the one that removes an existing wart. macondo synthesises an
// unsuccessful-challenge turn loss without knowing clocks exist, so
// pkg/gameplay/game.go:437 flips the player on turn, records the time, and
// flips back to work out whose move it was. A result that simply says whose
// clock it was makes that unnecessary.

import "fmt"

// ApplyTimePenalty deducts an overtime penalty from a player's score.
//
// It is not a turn: the penalised player still owes a move, so nothing else
// about the position changes. How large the penalty is and when it accrues are
// liwords' to decide -- this only records the consequence.
func (s *State) ApplyTimePenalty(r *Rules, player int, points int32) (*ApplyResult, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	if player < 0 || player >= MaxPlayers {
		return nil, fmt.Errorf("xwordgame: player index %d out of range", player)
	}
	if points < 0 {
		return nil, fmt.Errorf("xwordgame: time penalty of %d is negative; pass the magnitude", points)
	}
	res := newApplyResult()
	res.Score = -points
	res.TimeAttributedTo = int8(player)
	s.Scores[player] -= points
	return res, nil
}

// ApplyTimeout ends a game because the given player ran out of time. They lose
// regardless of the score, so the winner is reported rather than inferred.
//
// Scores are untouched: liwords settles a timeout on the clock, not on the
// tiles, and the losing player's remaining rack is not deducted. That differs
// from every other ending here, which is exactly why it is worth stating.
func (s *State) ApplyTimeout(r *Rules, player int) (*ApplyResult, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	if player < 0 || player >= MaxPlayers {
		return nil, fmt.Errorf("xwordgame: player index %d out of range", player)
	}
	if s.PlayState == GameOver {
		return nil, errGameIsOver
	}
	res := newApplyResult()
	res.GameOver = true
	res.Winner = int8(otherPlayer(uint8(player)))
	res.TimeAttributedTo = int8(player)

	s.PlayState = GameOver
	// A game that ended on the clock has nothing left to challenge.
	s.LastWordsFormed = nil
	return res, nil
}
