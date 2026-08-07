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

	TurnNum        uint16
	ScorelessTurns uint8
	OnTurn         uint8
	PlayState      PlayState

	// LastWordsFormed is the set of words created by the most recent tile
	// placement, retained so a subsequent challenge can be adjudicated without
	// re-deriving them from the board. macondo treats this as scratch state
	// that "does not need to be backed up", which is only true for a process
	// that never reloads a game mid-turn. liwords reloads constantly.
	LastWordsFormed []tilemapping.MachineWord

	// unknownTail holds trailing bytes written by a newer encoder that this
	// build does not understand. See the note on forward compatibility in
	// Decode.
	unknownTail []byte
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

// Rack returns a mutable view of a player's rack. The slice aliases the State.
func (s *State) Rack(p int) []tilemapping.MachineLetter {
	return s.racks[p][:s.rackLens[p]]
}

// SetRack replaces a player's rack.
func (s *State) SetRack(p int, tiles []tilemapping.MachineLetter) error {
	if p < 0 || p >= MaxPlayers {
		return fmt.Errorf("xwordgame: player index %d out of range", p)
	}
	if len(tiles) > RackTileLimit {
		return fmt.Errorf("xwordgame: rack of %d tiles exceeds limit of %d", len(tiles), RackTileLimit)
	}
	copy(s.racks[p][:], tiles)
	s.rackLens[p] = uint8(len(tiles))
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
	s.unknownTail = nil
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
	if s.unknownTail != nil {
		c.unknownTail = append([]byte(nil), s.unknownTail...)
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
	// The total-length header is a uint16; a real snapshot is two orders of
	// magnitude below that, but check rather than let AppendTo silently write a
	// wrapped length.
	if n := s.EncodedSize(); n > math.MaxUint16 {
		return fmt.Errorf("xwordgame: encoded snapshot of %d bytes exceeds the format limit", n)
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

// -----------------------------------------------------------------------------
// Wire format
// -----------------------------------------------------------------------------

// FormatVersion is the current snapshot wire format.
//
// Bump this ONLY for a breaking layout change; a decoder refuses any version it
// does not recognise rather than guessing, because a misparsed snapshot is
// exactly the failure mode this whole design exists to prevent.
//
// Additive changes do not bump the version: append new fields after every
// existing one, and have the decoder check for remaining bytes before reading
// them. Older builds preserve what they do not understand (see unknownTail), so
// a rolling deploy can run mixed versions against the same live games -- which
// is mandatory, because correspondence games last weeks and can never be
// drained for a cutover.
const FormatVersion = 1

// headerLen is the version byte plus the uint16 total-length field.
const headerLen = 3

// MaxEncodedSize is a generous upper bound on an encoded snapshot. A full 21x21
// board with two full racks and eight formed words comes to well under 700
// bytes. Staying under Postgres's ~2KB TOAST threshold keeps the snapshot
// inline in the heap tuple, so reading or writing it never costs an extra table
// access.
const MaxEncodedSize = 1024

// EncodedSize returns the exact number of bytes Encode will produce.
func (s *State) EncodedSize() int {
	n := headerLen
	n += 1 + 1 + 2 + 1  // play state, on turn, turn num, scoreless turns
	n += 4 * MaxPlayers // scores
	n += 2 * MaxPlayers // bingos
	n += 2 * MaxPlayers // player turns
	n += 1 + int(s.dim)*int(s.dim)
	for p := range s.rackLens {
		n += 1 + int(s.rackLens[p])
	}
	n += AlphabetSize // bag counts
	n++               // formed-word count
	for _, w := range s.LastWordsFormed {
		n += 1 + len(w)
	}
	n += len(s.unknownTail)
	return n
}

// Encode serialises the State into a fresh buffer.
func (s *State) Encode() []byte {
	return s.AppendTo(make([]byte, 0, s.EncodedSize()))
}

// AppendTo serialises the State onto dst and returns the extended buffer,
// letting callers reuse a scratch buffer across moves.
func (s *State) AppendTo(dst []byte) []byte {
	start := len(dst)

	dst = append(dst, FormatVersion, 0, 0) // length is patched in below
	dst = append(dst, byte(s.PlayState), s.OnTurn)
	dst = binary.LittleEndian.AppendUint16(dst, s.TurnNum)
	dst = append(dst, s.ScorelessTurns)
	for p := range MaxPlayers {
		dst = binary.LittleEndian.AppendUint32(dst, uint32(s.Scores[p]))
	}
	for p := range MaxPlayers {
		dst = binary.LittleEndian.AppendUint16(dst, s.Bingos[p])
	}
	for p := range MaxPlayers {
		dst = binary.LittleEndian.AppendUint16(dst, s.PlayerTurns[p])
	}

	dst = append(dst, s.dim)
	dst = appendTiles(dst, s.Board())

	for p := range MaxPlayers {
		dst = append(dst, s.rackLens[p])
		dst = appendTiles(dst, s.Rack(p))
	}

	dst = append(dst, s.bag[:]...)

	dst = append(dst, uint8(len(s.LastWordsFormed)))
	for _, w := range s.LastWordsFormed {
		dst = append(dst, uint8(len(w)))
		dst = appendTiles(dst, w)
	}

	dst = append(dst, s.unknownTail...)

	binary.LittleEndian.PutUint16(dst[start+1:start+3], uint16(len(dst)-start))
	return dst
}

func appendTiles(dst []byte, tiles []tilemapping.MachineLetter) []byte {
	for _, ml := range tiles {
		dst = append(dst, byte(ml))
	}
	return dst
}

// Decode parses a snapshot into s, replacing its current contents. The State
// takes no reference to buf.
//
// Trailing bytes beyond the fields this build knows about are retained verbatim
// and re-emitted by Encode, so that a node running an older binary does not
// silently strip fields written by a newer one during a rolling deploy. Note
// the corollary: any additive field whose value is coupled to the rest of the
// state must NOT rely on this, since an old node will happily play a move and
// write back a stale copy of it. Load-bearing additions require a version bump.
func (s *State) Decode(buf []byte) error {
	if len(buf) < headerLen {
		return fmt.Errorf("xwordgame: snapshot too short: %d bytes", len(buf))
	}
	if v := buf[0]; v != FormatVersion {
		return fmt.Errorf("xwordgame: unsupported snapshot format version %d (this build understands %d)", v, FormatVersion)
	}
	if total := int(binary.LittleEndian.Uint16(buf[1:3])); total != len(buf) {
		return fmt.Errorf("xwordgame: snapshot length mismatch: header says %d, buffer is %d", total, len(buf))
	}

	s.Reset()
	r := reader{buf: buf, pos: headerLen}

	s.PlayState = PlayState(r.u8())
	s.OnTurn = r.u8()
	s.TurnNum = r.u16()
	s.ScorelessTurns = r.u8()
	for p := range MaxPlayers {
		s.Scores[p] = int32(r.u32())
	}
	for p := range MaxPlayers {
		s.Bingos[p] = r.u16()
	}
	for p := range MaxPlayers {
		s.PlayerTurns[p] = r.u16()
	}

	dim := r.u8()
	if r.err == nil && (dim < 1 || dim > MaxBoardDim) {
		return fmt.Errorf("xwordgame: snapshot has invalid board dimension %d", dim)
	}
	s.dim = dim
	r.tilesInto(s.board[:int(dim)*int(dim)])

	for p := range MaxPlayers {
		n := r.u8()
		if r.err == nil && n > RackTileLimit {
			return fmt.Errorf("xwordgame: snapshot has %d tiles on player %d's rack", n, p)
		}
		s.rackLens[p] = n
		r.tilesInto(s.racks[p][:n])
	}

	if b := r.take(AlphabetSize); b != nil {
		copy(s.bag[:], b)
	}

	nWords := int(r.u8())
	if r.err == nil && nWords > 0 {
		s.LastWordsFormed = make([]tilemapping.MachineWord, nWords)
		for i := range nWords {
			wl := int(r.u8())
			if r.err != nil {
				break
			}
			s.LastWordsFormed[i] = make(tilemapping.MachineWord, wl)
			r.tilesInto(s.LastWordsFormed[i])
		}
	}

	if r.err != nil {
		return r.err
	}
	if r.remaining() > 0 {
		s.unknownTail = append([]byte(nil), buf[r.pos:]...)
	}
	return s.Validate()
}

// reader is a bounds-checked sequential reader. Once it overruns it records an
// error and returns zeroes, so callers can read a run of fields and check once.
type reader struct {
	buf []byte
	pos int
	err error
}

func (r *reader) remaining() int { return len(r.buf) - r.pos }

func (r *reader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if r.remaining() < n {
		r.err = fmt.Errorf("xwordgame: snapshot truncated: wanted %d bytes at offset %d, have %d", n, r.pos, r.remaining())
		return nil
	}
	b := r.buf[r.pos : r.pos+n]
	r.pos += n
	return b
}

func (r *reader) u8() uint8 {
	b := r.take(1)
	if b == nil {
		return 0
	}
	return b[0]
}

func (r *reader) u16() uint16 {
	b := r.take(2)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint16(b)
}

func (r *reader) u32() uint32 {
	b := r.take(4)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}

func (r *reader) tilesInto(dst []tilemapping.MachineLetter) {
	b := r.take(len(dst))
	if b == nil {
		return
	}
	for i, v := range b {
		dst[i] = tilemapping.MachineLetter(v)
	}
}

// Digest returns a hash of the full state. Because the bag is a multiset and
// the encoding is otherwise dense and deterministic, the digest is canonical:
// two engines agree if and only if their digests match, with no normalisation
// needed first. The referee parity harness compares states this way.
func (s *State) Digest() uint64 {
	return xxhash.Sum64(s.Encode())
}
