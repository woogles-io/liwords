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
	// EndRackRackUnknown counts end-of-game adjustments our figure got wrong
	// because we did not know the rack being charged. This is a limit of
	// replaying a log, not a defect: live play knows both racks.
	EndRackRackUnknown int
	// EndRackArithmeticWrong counts the ones where we knew the rack and still
	// produced the wrong number. These are real defects -- live play would get
	// them wrong too -- and the count is expected to be zero.
	EndRackArithmeticWrong int
}

// isDerivedEvent reports whether an event should be skipped as an input.
//
// Nothing is, any more. The list started with the end-of-game bookkeeping on
// the theory that the state machine regenerates it, but a replay's job is to
// reproduce the game that was played, and the recorded adjustment is what
// actually happened: the log knows racks at the moment a game ended that the
// events cannot otherwise reveal. Our own computation is still checked against
// the recorded one -- see endRackMismatch -- so the arithmetic is verified
// without the position being derived from it.
//
// The same reasoning already applied to the challenge outcomes: liwords records
// no CHALLENGE action, only its consequence.
func isDerivedEvent(t macondopb.GameEvent_Type) bool {
	_ = t
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
	// These moves were accepted by the referee that actually ran, under the
	// lexicon and exchange rules of their day. Re-deciding either now would
	// reject moves that really happened; see Rules.TrustRecordedPlays.
	trusted := *r
	trusted.TrustRecordedPlays = true
	r = &trusted

	alph := r.LetterDistribution.TileMapping()

	s, err := xwordgame.NewState(r.Layout.Dim())
	if err != nil {
		return nil, err
	}
	if err := s.FillBag(r.LetterDistribution); err != nil {
		return nil, err
	}

	res := &ReplayResult{State: s}
	// The most recent rack the log recorded for each player. An event only ever
	// names the mover's rack, so this is the best evidence available about the
	// opponent -- and near the end of a game, where the bag is empty and racks
	// stop changing, it is exact. That matters because the scoreless-turn
	// penalty is charged against both racks at once.
	var lastRack [xwordgame.MaxPlayers]tilemapping.MachineWord
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
		// how large it was and when it happened.
		if evt.Type == macondopb.GameEvent_TIME_PENALTY {
			if _, err := s.ApplyTimePenalty(r, int(evt.PlayerIndex), abs32(evt.LostScore)); err != nil {
				return res, fmt.Errorf("event %d (%s): %w", i, evt.Type, err)
			}
			res.Applied++
			continue
		}

		if isEndRackEvent(evt.Type) {
			if err := applyEndRack(s, evt, res, alph); err != nil {
				return res, fmt.Errorf("event %d (%s): %w", i, evt.Type, err)
			}
			continue
		}

		// The end-rack events that follow a game-ending move carry each player's
		// final rack, and that is rack information the log does not otherwise
		// provide: an ordinary event names only the mover's rack, so the
		// opponent's is a guess by the time a stalemate charges both of them.
		// Read it before applying the move that triggers the ending -- the
		// penalty itself is still ours to compute, which is the part under test.
		for j := i + 1; j < len(hist.Events); j++ {
			nxt := hist.Events[j]
			if nxt.Type != macondopb.GameEvent_END_RACK_PENALTY &&
				nxt.Type != macondopb.GameEvent_END_RACK_PTS {
				break
			}
			if int(nxt.PlayerIndex) < xwordgame.MaxPlayers && nxt.Rack != "" {
				if mw, err := tilemapping.ToMachineWord(nxt.Rack, alph); err == nil {
					lastRack[nxt.PlayerIndex] = mw
				}
			}
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
		if err := replayOne(s, r, rng, evt, alph, prev, res, &lastRack); err != nil {
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
	applyStoredEnding(s, hist)
	return res, nil
}

// replayOne applies a single player event.
func replayOne(s *xwordgame.State, r *xwordgame.Rules, rng xwordgame.Rand,
	evt *macondopb.GameEvent, alph *tilemapping.TileMapping,
	prev *xwordgame.State, res *ReplayResult,
	lastRack *[xwordgame.MaxPlayers]tilemapping.MachineWord) error {

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
		if err := seatMover(s, rng, int(evt.PlayerIndex), rack, *lastRack); err != nil {
			return fmt.Errorf("seating rack %q: %w", evt.Rack, err)
		}
		lastRack[evt.PlayerIndex] = append(tilemapping.MachineWord(nil), rack...)
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

// isEndRackEvent reports whether an event is an end-of-game rack adjustment.
func isEndRackEvent(t macondopb.GameEvent_Type) bool {
	return t == macondopb.GameEvent_END_RACK_PTS || t == macondopb.GameEvent_END_RACK_PENALTY
}

// applyEndRack applies a recorded end-of-game rack adjustment, and records
// whether our own machine had already produced the same number.
//
// The recorded value wins. A game ending charges each rack at the moment it
// ended, and the log is the only place the racks at that moment are written
// down -- an ordinary event names the mover's rack, never the opponent's, and
// the tiles drawn in a final exchange appear nowhere else. Recomputing from a
// rack we had to guess produces a score the game never had.
//
// EndRackMismatches counts the times our figure disagreed, so the arithmetic is
// still under test even though it is not what lands in the position.
func applyEndRack(s *xwordgame.State, evt *macondopb.GameEvent, res *ReplayResult,
	alph *tilemapping.TileMapping) error {
	p := int(evt.PlayerIndex)
	if p < 0 || p >= xwordgame.MaxPlayers {
		return fmt.Errorf("player index %d out of range", p)
	}
	var delta int32
	if evt.Type == macondopb.GameEvent_END_RACK_PTS {
		delta = evt.EndRackPoints
	} else {
		delta = -abs32(evt.LostScore)
	}
	// Cumulative is the player's score after this event, so it says what our
	// own end-of-game handling should already have arrived at.
	//
	// When it does not, distinguish the two possible causes, because only one
	// of them matters for live play. If the rack we charged differs from the
	// one the log names, we simply did not know the tiles -- an ordinary event
	// never reveals the opponent's rack, and a final exchange's draw is
	// invisible until this event. Live play knows both racks exactly, so that
	// gap cannot happen there. If the racks agree and the number still does
	// not, the arithmetic is wrong, and live play would get it wrong too.
	if evt.Cumulative != 0 && s.Scores[p] != evt.Cumulative {
		if evt.Rack != "" && !sameRack(s, p, evt.Rack, alph) {
			res.EndRackRackUnknown++
		} else {
			res.EndRackArithmeticWrong++
		}
	}
	if evt.Cumulative != 0 {
		s.Scores[p] = evt.Cumulative
	} else {
		s.Scores[p] += delta
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

	// Under the triple rule the challenge itself decides the game, whichever
	// way it went: the outcome functions below apply the points and the board,
	// but ending it is the rule's doing, not theirs.
	if r.ChallengeRule == xwordgame.ChallengeRuleTriple {
		defer func() {
			if s.PlayState != xwordgame.GameOver {
				s.EndGameByRule()
			}
		}()
	}

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
func seatMover(s *xwordgame.State, rng xwordgame.Rand, mover int, rack tilemapping.MachineWord,
	lastRack [xwordgame.MaxPlayers]tilemapping.MachineWord) error {

	opp := 1 - mover

	oppSize := s.RackLen(opp)
	if err := s.AssignRack(opp, nil); err != nil {
		return err
	}
	if err := s.AssignRack(mover, rack); err != nil {
		return err
	}
	if oppSize == 0 {
		return nil
	}

	// Prefer the last rack the log recorded for the opponent over an invented
	// one. It can be stale -- they may have drawn since -- but it is real, and
	// once the bag is empty it is exact. The scoreless-turn penalty is charged
	// against this rack, so inventing it silently gets the final scores wrong;
	// that is how a Q went missing from a 2021 endgame.
	if want := lastRack[opp]; len(want) == oppSize {
		if err := s.AssignRack(opp, want); err == nil {
			return nil
		}
		// The bag cannot supply it, so the opponent has drawn since. Fall
		// through to a stand-in, which at least keeps the sizes right.
	}

	if _, err := s.DrawToFull(rng, opp); err != nil {
		return err
	}
	// DrawToFull tops up to a full rack; trim back to the size the opponent
	// actually held.
	if extra := s.RackLen(opp) - oppSize; extra > 0 {
		spare := append(tilemapping.MachineWord(nil), s.Rack(opp)[:extra]...)
		if err := s.TakeFromRack(opp, spare); err != nil {
			return err
		}
		if err := s.PutBack(spare); err != nil {
			return err
		}
	}
	return nil
}

// assignFinalRacks sets both racks from the history's last-known racks, which
// is where the current racks actually live -- events only ever record the
// mover's rack before their own play.
func assignFinalRacks(s *xwordgame.State, hist *macondopb.GameHistory,
	alph *tilemapping.TileMapping) error {

	// Nothing to put back means nothing to take out. A history without
	// last-known racks -- a partial replay, or an old export -- should keep the
	// racks the replay computed rather than have them emptied into the bag.
	known := false
	for _, str := range hist.LastKnownRacks {
		if str != "" {
			known = true
		}
	}
	if !known {
		return nil
	}

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

// applyStoredEnding finishes a game whose ending the events do not show.
//
// Some endings leave no trace in the event list at all. A triple challenge the
// challenger lost is the clearest: macondo's ChallengeEvent sets the winner and
// ends the game without appending anything, so the events are indistinguishable
// from a game still in progress. A player can also go out and the game be over
// with no final pass ever recorded. The corpus has both.
//
// What marks these finished is stored fields rather than events. FinalScores is
// written only when a game concludes, and PlayState is written alongside it, so
// either one saying so is enough.
//
// macondo does not read its own history.PlayState here -- PlayToTurn re-derives
// the state from whether a rack is empty, which is why it reports a finished
// triple challenge as still playing. Reading what was stored is both simpler
// and right, and it is the same principle the migration rests on: the position
// is recorded, not inferred.
func applyStoredEnding(s *xwordgame.State, hist *macondopb.GameHistory) {
	if s.PlayState == xwordgame.GameOver {
		return
	}
	if hist.PlayState == macondopb.PlayState_GAME_OVER || len(hist.FinalScores) > 0 {
		s.EndGameByRule()
		return
	}
	// Not over, but a player is out of tiles, so the game is waiting on the
	// opponent to pass or challenge.
	for p := range xwordgame.MaxPlayers {
		if s.RackLen(p) == 0 {
			s.PlayState = xwordgame.WaitingForFinalPass
			return
		}
	}
}

// sameRack reports whether player p holds exactly the rack the log names.
func sameRack(s *xwordgame.State, p int, recorded string, alph *tilemapping.TileMapping) bool {
	mw, err := tilemapping.ToMachineWord(recorded, alph)
	if err != nil {
		return false
	}
	have := s.Rack(p)
	if len(have) != len(mw) {
		return false
	}
	// Racks are stored sorted, so sort the recorded one to match.
	want := append(tilemapping.MachineWord(nil), mw...)
	for i := 1; i < len(want); i++ {
		for j := i; j > 0 && want[j] < want[j-1]; j-- {
			want[j], want[j-1] = want[j-1], want[j]
		}
	}
	for i := range want {
		if want[i].IsBlanked() {
			want[i] = 0
		}
		if have[i] != want[i] {
			return false
		}
	}
	return true
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
