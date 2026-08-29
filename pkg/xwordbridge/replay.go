package xwordbridge

// Replaying a GameHistory through xwordgame.
//
// This is the other direction from bridge.go: instead of reading a position out
// of a live macondo game, it drives xwordgame's own state machine from an event
// log and reports where it lands. It exists so the engine can be validated
// against the real corpus -- thousands of games that were actually played, with
// whatever shapes production contains -- rather than only against synthetic
// ones.
//
// Two things it deliberately does NOT do:
//
//   - It does not consult macondo. Playthrough squares are resolved against our
//     own board, and every rule decision is our state machine's. A replay that
//     leaned on macondo to interpret the log would not be evidence of anything.
//   - It does not treat the log as a source of position. Derived events --
//     challenge bonuses, phony returns, end-rack adjustments, turn losses -- are
//     *outputs* of a referee, so they are skipped as inputs and regenerated.
//     Replaying them literally would be the derive-position-from-log mistake
//     that this whole package exists to undo; regenerating them is what makes
//     the comparison meaningful.
//
// Racks are the one thing the log cannot supply going forward: an event records
// the mover's rack before their play, never the tiles they drew afterwards. So
// each event's recorded rack is assigned before the move is applied, which is
// exactly the information the log does carry.

import (
	"errors"
	"fmt"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/woogles-io/liwords/pkg/xwordgame"
)

// ErrUnsupportedEvent reports an event kind the replay cannot drive.
var ErrUnsupportedEvent = errors.New("xwordbridge: unsupported event type")

// ReplayResult is where a replay ended up, and what it had to do to get there.
type ReplayResult struct {
	State *xwordgame.State
	// Applied counts events fed to the state machine; Regenerated counts
	// derived events skipped because the machine produces them itself.
	Applied     int
	Regenerated int
	// Challenges counts adjudicated challenges, the case most worth knowing
	// the coverage of.
	Challenges int
}

// isDerivedEvent reports whether an event is something the state machine emits
// for itself when it ends a game, and which therefore must not be fed back in.
//
// The challenge outcomes are deliberately NOT in this list. liwords never
// records a CHALLENGE action -- only its consequence -- so PHONY_TILES_RETURNED
// and CHALLENGE_BONUS are the only evidence a challenge happened, and skipping
// them would leave the phony on the board and the bonus unpaid.
func isDerivedEvent(t macondopb.GameEvent_Type) bool {
	switch t {
	case macondopb.GameEvent_END_RACK_PTS,
		macondopb.GameEvent_END_RACK_PENALTY:
		return true
	}
	return false
}

// isChallengeOutcome reports whether an event records the result of a challenge
// rather than a player's move.
func isChallengeOutcome(t macondopb.GameEvent_Type) bool {
	switch t {
	case macondopb.GameEvent_PHONY_TILES_RETURNED,
		macondopb.GameEvent_CHALLENGE_BONUS,
		macondopb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS:
		return true
	}
	return false
}

// MoveFromEvent converts a macondo GameEvent into an xwordgame Move.
//
// Played-through squares are resolved against s, our own board. The log may
// either mark them with the playthrough marker or spell the covered letter out;
// both forms have to produce the same move, and only the board can say which
// squares are already occupied.
func MoveFromEvent(s *xwordgame.State, evt *macondopb.GameEvent,
	alph *tilemapping.TileMapping) (*xwordgame.Move, error) {

	switch evt.Type {
	case macondopb.GameEvent_PASS:
		return xwordgame.NewPassMove(), nil

	case macondopb.GameEvent_CHALLENGE:
		return xwordgame.NewChallengeMove(), nil

	case macondopb.GameEvent_EXCHANGE:
		tiles, err := exchangedTiles(evt, alph)
		if err != nil {
			return nil, err
		}
		return xwordgame.NewExchangeMove(tiles), nil

	case macondopb.GameEvent_TILE_PLACEMENT_MOVE:
		tiles, err := tilemapping.ToMachineWord(evt.PlayedTiles, alph)
		if err != nil {
			return nil, fmt.Errorf("xwordbridge: parsing played tiles %q: %w", evt.PlayedTiles, err)
		}
		vertical := evt.Direction == macondopb.GameEvent_VERTICAL
		row, col := int(evt.Row), int(evt.Column)
		if err := resolvePlaythrough(s, tiles, vertical, row, col); err != nil {
			return nil, err
		}
		return xwordgame.NewPlacementMove(row, col, vertical, tiles), nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedEvent, evt.Type)
}

// exchangedTiles reads the tiles an exchange gave up. Older events record only
// a count, in which case the specific tiles were never written down; any tiles
// from the rack will do, since an exchange returns them to an unordered bag.
func exchangedTiles(evt *macondopb.GameEvent, alph *tilemapping.TileMapping) (tilemapping.MachineWord, error) {
	if evt.Exchanged != "" {
		if tiles, err := tilemapping.ToMachineWord(evt.Exchanged, alph); err == nil {
			return tiles, nil
		}
		// Not tiles; fall through to the count form below.
	}
	n := int(evt.NumTilesFromRack)
	if n == 0 {
		return nil, fmt.Errorf("xwordbridge: exchange event records neither tiles nor a count")
	}
	rack, err := tilemapping.ToMachineWord(evt.Rack, alph)
	if err != nil {
		return nil, fmt.Errorf("xwordbridge: parsing rack %q: %w", evt.Rack, err)
	}
	if n > len(rack) {
		return nil, fmt.Errorf("xwordbridge: exchange of %d tiles from a rack of %d", n, len(rack))
	}
	return rack[:n], nil
}

// resolvePlaythrough rewrites tiles that are already on the board to the
// playthrough marker, and verifies that a spelled-out playthrough matches what
// is actually there. Mirrors macondo's modifyForPlaythrough, against our board.
func resolvePlaythrough(s *xwordgame.State, tiles tilemapping.MachineWord,
	vertical bool, row, col int) error {

	for idx := range tiles {
		r, c := row, col+idx
		if vertical {
			r, c = row+idx, col
		}
		if !s.PosExists(r, c) {
			return fmt.Errorf("xwordbridge: play runs off the board at row %d col %d", r, c)
		}
		if tiles[idx] == 0 || !s.HasLetter(r, c) {
			continue
		}
		// The square is occupied and the event named a letter for it, so the
		// letter must be the one already there.
		onBoard := s.TileAt(r, c)
		if onBoard != tiles[idx] && onBoard.Unblank() != tiles[idx].Unblank() {
			return fmt.Errorf("xwordbridge: play-through tile at row %d col %d is %d, event says %d",
				r, c, onBoard, tiles[idx])
		}
		tiles[idx] = 0
	}
	return nil
}

// ReplayHistory drives xwordgame's state machine through a game's events and
// returns the resulting position.
//
// Passing a nil Rand uses a deterministic source: replacement draws are
// overwritten by the next event's recorded rack, so their contents never
// matter, but making them deterministic keeps a replay reproducible.
func ReplayHistory(hist *macondopb.GameHistory, r *xwordgame.Rules, rng xwordgame.Rand) (*ReplayResult, error) {
	if hist == nil {
		return nil, errors.New("xwordbridge: nil history")
	}
	if err := rulesOK(r); err != nil {
		return nil, err
	}
	if rng == nil {
		rng = deterministicRand{}
	}
	alph := r.LetterDistribution.TileMapping()

	s, err := xwordgame.NewState(r.Layout.Dim())
	if err != nil {
		return nil, err
	}
	if err := s.FillBag(r.LetterDistribution); err != nil {
		return nil, err
	}

	res := &ReplayResult{State: s}
	// The position before the most recent move, which a challenge needs in
	// order to take a phony off the board. Replay knows it exactly: it is where
	// this machine was one move ago, so there is nothing to reconstruct.
	var prev *xwordgame.State

	for i, evt := range hist.Events {
		if isDerivedEvent(evt.Type) {
			res.Regenerated++
			continue
		}
		// A time penalty is not something this package can regenerate: xwordgame
		// models the position, not the clock, so the log is the only source for
		// it. It adjusts a score without being a turn -- the player who ran low
		// still owes a move -- so nothing else about the position changes.
		if evt.Type == macondopb.GameEvent_TIME_PENALTY {
			if int(evt.PlayerIndex) >= xwordgame.MaxPlayers {
				return res, fmt.Errorf("event %d: player index %d out of range", i, evt.PlayerIndex)
			}
			s.Scores[evt.PlayerIndex] -= abs32(evt.LostScore)
			res.Applied++
			continue
		}

		if isChallengeOutcome(evt.Type) {
			if err := replayChallengeOutcome(s, r, evt, prev, res); err != nil {
				return res, fmt.Errorf("event %d (%s): %w", i, evt.Type, err)
			}
			// The challenge consumed the challengeable play either way.
			prev = nil
			continue
		}

		before := s.Clone()
		if err := replayOne(s, r, rng, evt, alph, prev, res); err != nil {
			return res, fmt.Errorf("event %d (%s): %w", i, evt.Type, err)
		}
		// Only a tile placement leaves something challengeable behind, which is
		// the same condition ongoing_games.prev_state is stored under.
		if evt.Type == macondopb.GameEvent_TILE_PLACEMENT_MOVE {
			prev = before
		} else {
			prev = nil
		}
	}

	// The log's last word on the racks. Every event before this only recorded
	// the mover's rack, so the final position needs both.
	if err := assignFinalRacks(s, hist, alph); err != nil {
		return res, err
	}
	return res, nil
}

// replayOne applies a single player event.
func replayOne(s *xwordgame.State, r *xwordgame.Rules, rng xwordgame.Rand,
	evt *macondopb.GameEvent, alph *tilemapping.TileMapping,
	prev *xwordgame.State, res *ReplayResult) error {

	// The event log is authoritative for whose turn it was; trusting our own
	// counter here would hide an ordering bug rather than surface it.
	if int(evt.PlayerIndex) >= xwordgame.MaxPlayers {
		return fmt.Errorf("player index %d out of range", evt.PlayerIndex)
	}
	if uint8(evt.PlayerIndex) != s.OnTurn {
		return fmt.Errorf("event is for player %d but player %d is on turn",
			evt.PlayerIndex, s.OnTurn)
	}

	if evt.Rack != "" {
		rack, err := tilemapping.ToMachineWord(evt.Rack, alph)
		if err != nil {
			return fmt.Errorf("parsing rack %q: %w", evt.Rack, err)
		}
		if err := seatMover(s, rng, int(evt.PlayerIndex), rack); err != nil {
			return fmt.Errorf("seating rack %q: %w", evt.Rack, err)
		}
	}

	m, err := MoveFromEvent(s, evt, alph)
	if err != nil {
		return err
	}

	if m.Type == xwordgame.MoveTypeChallenge {
		res.Challenges++
		// A challenge needs the position before the challenged play. Replay has
		// it exactly: it is the position this machine was in one move ago.
		_, err := s.AdjudicateChallenge(xwordgame.ChallengeParams{
			Rules: r,
			Prev:  prev,
		})
		if err != nil {
			return err
		}
		res.Applied++
		return nil
	}

	if _, err := s.ApplyMove(r, rng, m); err != nil {
		return err
	}
	res.Applied++
	return nil
}

// replayChallengeOutcome applies a challenge whose verdict the log already
// records.
//
// It deliberately does not re-adjudicate. These games were played under the
// lexicon of their day, and lexicons change -- re-deciding an old challenge
// against today's word list would produce a position the game never reached.
// The log is authoritative for what happened; xwordgame is authoritative for
// what that does to the position.
func replayChallengeOutcome(s *xwordgame.State, r *xwordgame.Rules,
	evt *macondopb.GameEvent, prev *xwordgame.State, res *ReplayResult) error {

	res.Challenges++

	switch evt.Type {
	case macondopb.GameEvent_PHONY_TILES_RETURNED:
		// The challenge succeeded: the play comes off and the challenger, who
		// is already on turn, keeps it.
		if prev == nil {
			return errors.New("a phony was returned but no play preceded it")
		}
		if _, err := s.ApplyReturnedPhony(r, prev); err != nil {
			return err
		}
		res.Applied++
		return nil

	case macondopb.GameEvent_CHALLENGE_BONUS:
		// The challenge failed under a points rule. The bonus is taken from the
		// event rather than recomputed from the rule, because the five-point
		// rule has been scored two different ways over the years and the log
		// records which one this game actually used.
		if _, err := s.ApplyChallengeBonus(r, int(evt.PlayerIndex), evt.Bonus); err != nil {
			return err
		}
		res.Applied++
		return nil

	case macondopb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS:
		// The challenge failed under the double rule, so the challenger
		// forfeits their turn. The state transition is a pass; only the event
		// the caller would write differs.
		if _, err := s.ApplyPass(r); err != nil {
			return err
		}
		res.Applied++
		return nil
	}
	return fmt.Errorf("%w: %s", ErrUnsupportedEvent, evt.Type)
}

// seatMover installs the rack the log recorded for the player about to move,
// and gives the opponent a stand-in of the right size.
//
// This is where replay has to be honest about what an event log knows. An event
// records the mover's rack before their play and nothing else, so the tiles a
// player drew afterwards are unknown until their next move reveals them. What
// *is* known at every point is how many tiles each player holds, and therefore
// how many are left in the bag.
//
// So both racks go back into the bag, the mover's recorded rack is taken out of
// it -- guaranteed available, since the bag is now everything not on the board
// -- and the opponent is dealt an arbitrary rack of their correct size. The
// opponent's specific tiles are wrong and deliberately so; nothing in the rules
// depends on them, and the next event that reveals them corrects the record.
//
// What this preserves is the part that matters: rack sizes and the bag count.
// Going out, the exchange minimum and the endgame all turn on those, and all
// three would break if the opponent's tiles were simply left in the bag.
func seatMover(s *xwordgame.State, rng xwordgame.Rand, mover int, rack tilemapping.MachineWord) error {
	opp := 1 - mover

	oppSize := s.RackLen(opp)
	if err := s.AssignRack(opp, nil); err != nil {
		return err
	}
	if err := s.AssignRack(mover, rack); err != nil {
		return err
	}
	if oppSize > 0 {
		if _, err := s.DrawToFull(rng, opp); err != nil {
			return err
		}
		// DrawToFull tops up to a full rack; trim back to the size the
		// opponent actually held.
		if extra := s.RackLen(opp) - oppSize; extra > 0 {
			spare := append(tilemapping.MachineWord(nil), s.Rack(opp)[:extra]...)
			if err := s.TakeFromRack(opp, spare); err != nil {
				return err
			}
			if err := s.PutBack(spare); err != nil {
				return err
			}
		}
	}
	return nil
}

// assignFinalRacks sets both racks from the history's last-known racks, which
// is where the current racks actually live -- events only ever record the
// mover's rack before their own play.
func assignFinalRacks(s *xwordgame.State, hist *macondopb.GameHistory,
	alph *tilemapping.TileMapping) error {

	// Clear both first: whichever player did not move last is holding stand-in
	// tiles from seatMover, and those have to go back before the real ones can
	// come out of the bag.
	for p := range xwordgame.MaxPlayers {
		if err := s.AssignRack(p, nil); err != nil {
			return err
		}
	}
	for p, str := range hist.LastKnownRacks {
		if p >= xwordgame.MaxPlayers || str == "" {
			continue
		}
		rack, err := tilemapping.ToMachineWord(str, alph)
		if err != nil {
			return fmt.Errorf("xwordbridge: parsing last known rack %q: %w", str, err)
		}
		if err := s.AssignRack(p, rack); err != nil {
			return fmt.Errorf("xwordbridge: assigning last known rack %q: %w", str, err)
		}
	}
	return nil
}

// abs32 normalises a penalty that may be recorded as either sign.
func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func rulesOK(r *xwordgame.Rules) error {
	if r == nil {
		return errors.New("xwordbridge: rules are required")
	}
	if r.Layout == nil || r.LetterDistribution == nil {
		return errors.New("xwordbridge: rules need a layout and a letter distribution")
	}
	return nil
}

// deterministicRand makes replacement draws reproducible. Their contents are
// always overwritten by the next event's recorded rack, so the sequence is
// irrelevant -- only its repeatability matters.
type deterministicRand struct{}

func (deterministicRand) IntN(n int) int { return 0 }
