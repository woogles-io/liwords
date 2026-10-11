// Package xwordbridge converts a macondo game into an xwordgame.State and
// compares the two.
//
// It is deliberately a separate package. pkg/xwordgame depends only on
// word-golib and is meant to be liftable into a standalone module; importing
// macondo there would end that. Keeping the bridge outside also means the whole
// migration scaffold is one directory that gets deleted when macondo leaves the
// live path.
//
// # What this is for
//
// A game already in progress has no xwordgame.State. Rather than backfilling
// one, StateFromGame derives it from the macondo game that liwords already
// holds in memory, so any game can acquire a State the first time it is touched
// after a deploy. That is what makes the migration incremental instead of a
// flag day.
//
// Compare is the shadow-mode workhorse: run both engines, diff the positions,
// log what disagrees. It reports every divergence rather than the first, since
// one root cause usually shows up in several fields at once and the shape of
// the set is the diagnosis.
package xwordbridge

import (
	"fmt"

	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/woogles-io/liwords/pkg/xwordgame"
)

// StateFromGame builds a snapshot of the position a macondo game is currently
// in. It reads the game and does not modify it.
func StateFromGame(g *macondogame.Game) (*xwordgame.State, error) {
	if g == nil {
		return nil, fmt.Errorf("xwordbridge: nil game")
	}
	b := g.Board()
	if b == nil {
		return nil, fmt.Errorf("xwordbridge: game has no board")
	}
	dim := b.Dim()
	// macondo transposes a board while scoring vertical plays and transposes
	// back afterwards. It does that by swapping the row and column strides
	// rather than moving squares, and exposes no accessor for the flag -- but
	// the swap is visible through GetSqIdx, which returns the row stride for
	// (1,0). Reading a board mid-flip would silently mirror the position, so
	// refuse rather than guess.
	if b.GetSqIdx(1, 0) != dim {
		return nil, fmt.Errorf("xwordbridge: the board is transposed; read it between moves")
	}
	s, err := xwordgame.NewState(dim)
	if err != nil {
		return nil, err
	}
	for row := range dim {
		for col := range dim {
			s.SetTileAt(row, col, b.GetLetter(row, col))
		}
	}

	hist := g.History()
	if hist == nil {
		return nil, fmt.Errorf("xwordbridge: game has no history")
	}
	if n := len(hist.Players); n != xwordgame.MaxPlayers {
		return nil, fmt.Errorf("xwordbridge: %d players, this package supports %d", n, xwordgame.MaxPlayers)
	}

	for p := range xwordgame.MaxPlayers {
		rack := g.RackFor(p)
		if rack == nil {
			return nil, fmt.Errorf("xwordbridge: player %d has no rack", p)
		}
		if err := s.SetRack(p, rack.TilesOn()); err != nil {
			return nil, err
		}
		s.Scores[p] = int32(g.PointsFor(p))
		// macondo exposes these per nickname rather than per index. Nicknames
		// are unique within a game; NewGame refuses duplicates.
		nick := hist.Players[p].Nickname
		s.Bingos[p] = uint16(g.BingosForNick(nick))
		s.PlayerTurns[p] = uint16(g.TurnsForNick(nick))
	}

	bag := g.Bag()
	if bag == nil {
		return nil, fmt.Errorf("xwordbridge: game has no bag")
	}
	// PeekMap is already a count vector indexed by MachineLetter, which is
	// exactly how State stores the bag.
	if err := s.SetBagCounts(bag.PeekMap()); err != nil {
		return nil, err
	}

	if turn := g.Turn(); turn >= 0 {
		s.TurnNum = uint16(turn)
	}
	if st := g.ScorelessTurns(); st >= 0 {
		s.ScorelessTurns = uint8(st)
	}
	onTurn := g.PlayerOnTurn()
	if onTurn < 0 || onTurn >= xwordgame.MaxPlayers {
		return nil, fmt.Errorf("xwordbridge: player %d is on turn, out of range", onTurn)
	}
	s.OnTurn = uint8(onTurn)

	ps, err := PlayStateFromMacondo(g.Playing())
	if err != nil {
		return nil, err
	}
	s.PlayState = ps

	// Retained so a challenge can be adjudicated without re-deriving the words
	// from the board. macondo treats this as scratch that "does not need to be
	// backed up", which only holds for a process that never reloads a game
	// mid-turn.
	if words := g.LastWordsFormed(); len(words) > 0 {
		s.LastWordsFormed = make([]tilemapping.MachineWord, len(words))
		for i, w := range words {
			s.LastWordsFormed[i] = append(tilemapping.MachineWord(nil), w...)
		}
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

// PlayStateFromMacondo converts macondo's play state. The values happen to
// coincide, but converting explicitly means a change on either side becomes a
// compile or runtime error rather than a silent mismatch.
func PlayStateFromMacondo(ps macondopb.PlayState) (xwordgame.PlayState, error) {
	switch ps {
	case macondopb.PlayState_PLAYING:
		return xwordgame.Playing, nil
	case macondopb.PlayState_WAITING_FOR_FINAL_PASS:
		return xwordgame.WaitingForFinalPass, nil
	case macondopb.PlayState_GAME_OVER:
		return xwordgame.GameOver, nil
	}
	return 0, fmt.Errorf("xwordbridge: unknown macondo play state %v", ps)
}

// ChallengeRuleFromMacondo converts macondo's challenge rule.
func ChallengeRuleFromMacondo(cr macondopb.ChallengeRule) (xwordgame.ChallengeRule, error) {
	switch cr {
	case macondopb.ChallengeRule_VOID:
		return xwordgame.ChallengeRuleVoid, nil
	case macondopb.ChallengeRule_SINGLE:
		return xwordgame.ChallengeRuleSingle, nil
	case macondopb.ChallengeRule_DOUBLE:
		return xwordgame.ChallengeRuleDouble, nil
	case macondopb.ChallengeRule_FIVE_POINT:
		return xwordgame.ChallengeRuleFivePoint, nil
	case macondopb.ChallengeRule_TEN_POINT:
		return xwordgame.ChallengeRuleTenPoint, nil
	case macondopb.ChallengeRule_TRIPLE:
		return xwordgame.ChallengeRuleTriple, nil
	}
	return 0, fmt.Errorf("xwordbridge: unknown macondo challenge rule %v", cr)
}

// Divergence is one field on which the two engines disagree.
type Divergence struct {
	Field  string
	Ours   string
	Theirs string
}

func (d Divergence) String() string {
	return fmt.Sprintf("%s: ours=%s macondo=%s", d.Field, d.Ours, d.Theirs)
}

// Compare reports every field on which the snapshot disagrees with the macondo
// game, in a fixed order.
//
// It returns all of them rather than stopping at the first: one root cause
// usually surfaces in several fields at once, and which fields those are is the
// most useful part of a shadow-mode log line.
func Compare(s *xwordgame.State, g *macondogame.Game) ([]Divergence, error) {
	theirs, err := StateFromGame(g)
	if err != nil {
		return nil, err
	}
	return CompareStates(s, theirs), nil
}

// CompareStates diffs two snapshots. The second is treated as macondo's for
// labelling purposes.
func CompareStates(ours, theirs *xwordgame.State) []Divergence {
	var out []Divergence
	add := func(field string, a, b any) {
		out = append(out, Divergence{Field: field, Ours: fmt.Sprint(a), Theirs: fmt.Sprint(b)})
	}

	if ours.Dim() != theirs.Dim() {
		// Nothing else is comparable across different board sizes.
		add("dim", ours.Dim(), theirs.Dim())
		return out
	}
	dim := ours.Dim()
	for row := range dim {
		for col := range dim {
			if a, b := ours.TileAt(row, col), theirs.TileAt(row, col); a != b {
				add(fmt.Sprintf("board[%d,%d]", row, col), a, b)
			}
		}
	}
	for p := range xwordgame.MaxPlayers {
		if a, b := ours.Rack(p), theirs.Rack(p); !equalTiles(a, b) {
			add(fmt.Sprintf("rack[%d]", p), a, b)
		}
		if a, b := ours.Scores[p], theirs.Scores[p]; a != b {
			add(fmt.Sprintf("score[%d]", p), a, b)
		}
		if a, b := ours.Bingos[p], theirs.Bingos[p]; a != b {
			add(fmt.Sprintf("bingos[%d]", p), a, b)
		}
		if a, b := ours.PlayerTurns[p], theirs.PlayerTurns[p]; a != b {
			add(fmt.Sprintf("turns[%d]", p), a, b)
		}
	}
	if a, b := ours.BagCounts(), theirs.BagCounts(); !equalCounts(a, b) {
		add("bag", a, b)
	}
	// TurnNum is deliberately not compared. macondo's Turn() indexes its event
	// history, so it also advances for the synthetic events written at the end
	// of a game: one for an end-rack bonus, two for the scoreless-turn
	// penalties. State.TurnNum counts plies. They agree throughout normal play
	// and part company exactly when a game ends, which would make every
	// finished game report a divergence that is not one.
	//
	// Nothing is lost by skipping it: PlayerTurns above means the same thing in
	// both engines -- macondo bumps players[i].turns only for a real move, as
	// we do -- so a genuine drift in move counting still shows up.
	if a, b := ours.ScorelessTurns, theirs.ScorelessTurns; a != b {
		add("scorelessTurns", a, b)
	}
	if a, b := ours.OnTurn, theirs.OnTurn; a != b {
		add("onTurn", a, b)
	}
	if a, b := ours.PlayState, theirs.PlayState; a != b {
		add("playState", a, b)
	}
	if a, b := ours.LastWordsFormed, theirs.LastWordsFormed; !equalWords(a, b) {
		add("lastWordsFormed", a, b)
	}
	return out
}

func equalTiles(a, b []tilemapping.MachineLetter) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalCounts(a, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalWords(a, b []tilemapping.MachineWord) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !equalTiles(a[i], b[i]) {
			return false
		}
	}
	return true
}
