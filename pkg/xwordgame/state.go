// Package xwordgame implements the rules engine and state machine for
// crossword board games (the OMGWords / Scrabble family).
//
// The package is deliberately dependency-light. It depends only on
// github.com/domino14/word-golib for tile mappings, letter distributions and
// lexicon lookups -- all of which are pure calculation with no state of their
// own. It imports no protobufs, and nothing from the rest of liwords, so it can
// eventually be lifted into a standalone module shared by liwords, macondo and
// the annotator.
//
// # Why a state snapshot at all
//
// A game has two distinct authoritative records, and conflating them is what
// caused the May 2026 data loss:
//
//   - The event log (game_turns, assembled into a GameHistory for the S3
//     archive) is authoritative for *what happened*. It is the whole game: the
//     archive, GCG export, replay, annotation and analysis are all built from
//     it.
//   - This State is authoritative for *the current position*: board, bag,
//     racks, scores, counters, whose turn it is, and what phase the game is in.
//
// Neither is derivable from the other. A position cannot tell you how it was
// reached, and -- less obviously -- the log cannot fully reconstruct the
// position. WAITING_FOR_FINAL_PASS, timeouts, resignations, the contents of the
// bag, the scoreless-turn counter and the words formed by the last play are all
// state changes with no corresponding event. The old design inferred them by
// replaying the log, which corrupted roughly 944 correspondence games. See
// docs/mikado/liwords_referee.md for the full post-mortem.
//
// So the log keeps its job and this keeps its own. What changed is only that
// the position is stored rather than re-derived.
package xwordgame

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/cespare/xxhash/v2"
	"github.com/domino14/word-golib/tilemapping"
)

const (
	// MaxBoardDim is the largest board dimension we support. SuperCrosswordGame
	// is 21x21; the classic board is 15x15.
	MaxBoardDim = 21
	// BoardCells is the in-memory capacity reserved for a board. A State always
	// reserves the maximum so that decoding never allocates for the board,
	// regardless of the variant.
	BoardCells = MaxBoardDim * MaxBoardDim
	// RackTileLimit is the number of tiles a full rack holds. This is 7 for
	// every variant we support, including the 21x21 super board.
	RackTileLimit = 7
	// MaxPlayers is the number of players in a game.
	MaxPlayers = 2
	// AlphabetSize bounds the distinct tile values in any distribution we
	// support (the largest, Polish, has 31 including the blank).
	AlphabetSize = tilemapping.MaxAlphabetSize
)

// PlayState is the phase of the game's lifecycle. Unlike in macondo, where the
// equivalent value is an internal field that only sometimes survives a round
// trip through storage, this is a first-class persisted concept: it is encoded
// in every snapshot and mirrored into a column on ongoing_games.
type PlayState uint8

const (
	// Playing is a game in progress.
	Playing PlayState = 0
	// WaitingForFinalPass is set when a player has gone out via tile placement
	// under a non-VOID challenge rule. The opponent must pass or challenge
	// before the game is actually over. This state has no corresponding event
	// in the game log, which is precisely why it must be persisted.
	WaitingForFinalPass PlayState = 1
	// GameOver is a finished game.
	GameOver PlayState = 2
)

// maxPlayState is the highest valid PlayState, used for validation.
const maxPlayState = GameOver

func (p PlayState) String() string {
	switch p {
	case Playing:
		return "PLAYING"
	case WaitingForFinalPass:
		return "WAITING_FOR_FINAL_PASS"
	case GameOver:
		return "GAME_OVER"
	}
	return fmt.Sprintf("PlayState(%d)", uint8(p))
}

// State is the complete mutable state of an in-progress game: everything you
// need, together with the game's immutable rules, to determine what may legally
// happen next.
//
// Immutable configuration (lexicon, letter distribution, board layout, variant,
// challenge rule, time controls) is deliberately NOT part of State. That data
// lives alongside the snapshot as plain columns and is resolved into shared,
// cached objects at load time.
//
// The zero value is not usable; construct with NewState or Decode.
type State struct {
	// dim is the board's side length. The board is stored row-major with a
	// stride of dim, not MaxBoardDim, so the live region is contiguous and can
	// be copied in one shot.
	dim   uint8
	board [BoardCells]tilemapping.MachineLetter

	racks    [MaxPlayers][RackTileLimit]tilemapping.MachineLetter
	rackLens [MaxPlayers]uint8

	// bag holds the undrawn tiles as a multiset: bag[ml] is how many of that
	// tile remain. It is deliberately NOT an ordered sequence.
	//
	// An ordered bag would imply that draws are determined by position, which
	// makes exchanges incoherent: the exchanging player returns tiles to the
	// bag in the same operation they draw from it, so any append-to-the-end
	// scheme hands them back their own discards. Reinserting at random
	// positions just reintroduces an RNG, defeating the point of storing an
	// order at all. macondo has the same conclusion baked in -- during live
	// play its bag runs with fixedOrder=false and reshuffles at draw time, so
	// its stored order is already meaningless.
	//
	// Counts also make the encoding canonical: two states with the same tiles
	// remaining always produce identical bytes, so digests can be compared
	// directly without normalising first.
	bag [AlphabetSize]uint8

	Scores [MaxPlayers]int32
	// Bingos and PlayerTurns are running per-player accumulators. They cannot
	// be recomputed without the full event log, so they are part of the state
	// even though they are only read for statistics.
	Bingos      [MaxPlayers]uint16
	PlayerTurns [MaxPlayers]uint16

	// TurnNum counts plies: it advances by exactly one per move applied,
	// whatever that move was.
	//
	// This is deliberately not macondo's Turn(), which is an index into its
	// event history and so also advances for the synthetic events it writes at
	// the end of a game -- one for an end-rack bonus, two for the
	// scoreless-turn penalties. The two agree throughout normal play and part
	// company only once a game ends. Counting log entries in a field that
	// describes a position is the coupling this package exists to avoid, so
	// xwordbridge.Compare checks PlayerTurns instead, which does mean the same
	// thing in both engines.
	TurnNum uint16

	ScorelessTurns uint8
	OnTurn         uint8
	PlayState      PlayState

	// LastWordsFormed is the set of words created by the most recent tile
	// placement, retained so a subsequent challenge can be adjudicated without
	// re-deriving them from the board. macondo treats this as scratch state
	// that "does not need to be backed up", which is only true for a process
	// that never reloads a game mid-turn. liwords reloads constantly.
	LastWordsFormed []tilemapping.MachineWord
}

// NewState returns an empty State for a board of the given dimension.
func NewState(dim int) (*State, error) {
	if dim < 1 || dim > MaxBoardDim {
		return nil, fmt.Errorf("xwordgame: board dimension %d out of range [1, %d]", dim, MaxBoardDim)
	}
	return &State{dim: uint8(dim)}, nil
}

// Dim returns the board's side length.
func (s *State) Dim() int { return int(s.dim) }

// Board returns a mutable view of the live region of the board, row-major with
// a stride of Dim(). The slice aliases the State; it is valid until the State
// is reset or decoded into.
func (s *State) Board() []tilemapping.MachineLetter {
	return s.board[:int(s.dim)*int(s.dim)]
}

// TileAt returns the tile at the given coordinates. A zero value means the
// square is empty; a value with the blank bit set is a played blank.
func (s *State) TileAt(row, col int) tilemapping.MachineLetter {
	return s.board[row*int(s.dim)+col]
}

// SetTileAt places a tile at the given coordinates.
func (s *State) SetTileAt(row, col int, ml tilemapping.MachineLetter) {
	s.board[row*int(s.dim)+col] = ml
}

// Rack returns a view of a player's rack, in canonical (sorted) order. The
// slice aliases the State; write to it only through the rack methods, which
// maintain that order.
func (s *State) Rack(p int) []tilemapping.MachineLetter {
	return s.racks[p][:s.rackLens[p]]
}

// SetRack replaces a player's rack. The tiles are stored sorted, and blanks are
// normalised to undesignated -- a blank only carries a letter while it is on the
// board. It does not touch the bag; see AssignRack for the accounting version.
func (s *State) SetRack(p int, tiles []tilemapping.MachineLetter) error {
	if p < 0 || p >= MaxPlayers {
		return fmt.Errorf("xwordgame: player index %d out of range", p)
	}
	if len(tiles) > RackTileLimit {
		return fmt.Errorf("xwordgame: rack of %d tiles exceeds limit of %d", len(tiles), RackTileLimit)
	}
	var counts [AlphabetSize]uint8
	for _, ml := range tiles {
		n, err := normalizeRackTile(ml)
		if err != nil {
			return err
		}
		counts[n]++
	}
	s.setRackFromCounts(p, counts)
	return nil
}

// Reset clears the State for reuse, keeping the board dimension and any
// allocated capacity.
func (s *State) Reset() {
	clear(s.board[:])
	s.racks = [MaxPlayers][RackTileLimit]tilemapping.MachineLetter{}
	s.rackLens = [MaxPlayers]uint8{}
	s.bag = [AlphabetSize]uint8{}
	s.Scores = [MaxPlayers]int32{}
	s.Bingos = [MaxPlayers]uint16{}
	s.PlayerTurns = [MaxPlayers]uint16{}
	s.TurnNum = 0
	s.ScorelessTurns = 0
	s.OnTurn = 0
	s.PlayState = Playing
	s.LastWordsFormed = s.LastWordsFormed[:0]
}

// Clone returns a deep copy of the State.
func (s *State) Clone() *State {
	c := *s
	if s.LastWordsFormed != nil {
		c.LastWordsFormed = make([]tilemapping.MachineWord, len(s.LastWordsFormed))
		for i, w := range s.LastWordsFormed {
			c.LastWordsFormed[i] = append(tilemapping.MachineWord(nil), w...)
		}
	}
	return &c
}

// Validate checks the State's structural invariants. It is called automatically
// by Decode; call it directly after constructing a State by hand if you want
// the same guarantees.
func (s *State) Validate() error {
	if s.dim < 1 || s.dim > MaxBoardDim {
		return fmt.Errorf("xwordgame: invalid board dimension %d", s.dim)
	}
	if s.OnTurn >= MaxPlayers {
		return fmt.Errorf("xwordgame: invalid on-turn player %d", s.OnTurn)
	}
	if s.PlayState > maxPlayState {
		return fmt.Errorf("xwordgame: invalid play state %d", s.PlayState)
	}
	for p := range s.rackLens {
		if s.rackLens[p] > RackTileLimit {
			return fmt.Errorf("xwordgame: player %d rack length %d exceeds limit", p, s.rackLens[p])
		}
		for _, ml := range s.Rack(p) {
			// Racks hold undesignated blanks as the zero value, never blanked
			// letters -- a blank is only assigned a letter when it is played.
			if ml.IsBlanked() || int(ml) >= AlphabetSize {
				return fmt.Errorf("xwordgame: player %d rack has invalid tile %d", p, ml)
			}
		}
	}
	for i, ml := range s.Board() {
		if int(ml.Unblank()) >= AlphabetSize {
			return fmt.Errorf("xwordgame: board square %d has invalid tile %d", i, ml)
		}
		if ml.IsBlanked() && ml.Unblank() == 0 {
			return fmt.Errorf("xwordgame: board square %d has a blanked empty tile", i)
		}
	}
	if len(s.LastWordsFormed) > math.MaxUint8 {
		return fmt.Errorf("xwordgame: %d formed words is not encodable", len(s.LastWordsFormed))
	}
	for i, w := range s.LastWordsFormed {
		if len(w) > MaxBoardDim {
			return fmt.Errorf("xwordgame: last-words entry %d is %d tiles long", i, len(w))
		}
	}
	return nil
}

// Equal reports whether two States are identical.
func (s *State) Equal(o *State) bool {
	if s == nil || o == nil {
		return s == o
	}
	return s.Digest() == o.Digest()
}

// Digest returns a hash of the full state.
//
// Because the bag is a multiset and every field is written in a fixed order
// with lengths where they are needed, the digest is canonical: two engines
// agree if and only if their digests match, with no normalisation first. The
// referee parity harness compares states this way, and Equal is built on it.
//
// This used to hash the wire format, back when a position was stored in a
// column. There is no wire format any more -- a position is rebuilt from the
// event log -- so it hashes the fields directly, which is the same thing
// without a version byte, a length header and a forward-compatibility escape
// hatch that existed only for storage.
func (s *State) Digest() uint64 {
	h := xxhash.New()
	var scratch [4]byte
	put8 := func(v uint8) { h.Write([]byte{v}) }
	put16 := func(v uint16) {
		binary.LittleEndian.PutUint16(scratch[:2], v)
		h.Write(scratch[:2])
	}
	put32 := func(v uint32) {
		binary.LittleEndian.PutUint32(scratch[:4], v)
		h.Write(scratch[:4])
	}
	putTiles := func(tiles []tilemapping.MachineLetter) {
		for _, ml := range tiles {
			put8(uint8(ml))
		}
	}

	put8(uint8(s.PlayState))
	put8(s.OnTurn)
	put16(s.TurnNum)
	put8(s.ScorelessTurns)
	for p := range MaxPlayers {
		put32(uint32(s.Scores[p]))
	}
	for p := range MaxPlayers {
		put16(s.Bingos[p])
	}
	for p := range MaxPlayers {
		put16(s.PlayerTurns[p])
	}
	// dim first, so the board that follows has an unambiguous length.
	put8(s.dim)
	putTiles(s.Board())
	for p := range MaxPlayers {
		put8(s.rackLens[p])
		putTiles(s.Rack(p))
	}
	h.Write(s.bag[:])
	put8(uint8(len(s.LastWordsFormed)))
	for _, w := range s.LastWordsFormed {
		put8(uint8(len(w)))
		putTiles(w)
	}
	return h.Sum64()
}
