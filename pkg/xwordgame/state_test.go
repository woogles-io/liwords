package xwordgame

import (
	"encoding/binary"
	"math/rand/v2"
	"testing"

	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
)

// fullState builds a state exercising every field, on a board of the given
// dimension.
func fullState(t *testing.T, dim int) *State {
	t.Helper()
	s, err := NewState(dim)
	if err != nil {
		t.Fatal(err)
	}
	s.PlayState = WaitingForFinalPass
	s.OnTurn = 1
	s.TurnNum = 37
	s.ScorelessTurns = 3
	s.Scores = [MaxPlayers]int32{412, 389}
	s.Bingos = [MaxPlayers]uint16{2, 1}
	s.PlayerTurns = [MaxPlayers]uint16{19, 18}

	// A handful of tiles, including a played blank (high bit set).
	s.SetTileAt(7, 7, 3)
	s.SetTileAt(7, 8, 1)
	s.SetTileAt(7, 9, 20|tilemapping.BlankMask)
	s.SetTileAt(dim-1, dim-1, 26)

	if err := s.SetRack(0, []tilemapping.MachineLetter{1, 5, 9, 15, 21, 0, 26}); err != nil {
		t.Fatal(err)
	}
	// A short rack, endgame style.
	if err := s.SetRack(1, []tilemapping.MachineLetter{17, 22}); err != nil {
		t.Fatal(err)
	}

	counts := make([]uint8, AlphabetSize)
	counts[0] = 2
	counts[1] = 9
	counts[5] = 12
	counts[26] = 1
	if err := s.SetBagCounts(counts); err != nil {
		t.Fatal(err)
	}

	s.LastWordsFormed = []tilemapping.MachineWord{
		{3, 1, 20},
		{17, 21, 9, 26},
	}
	return s
}

func TestRoundTrip15x15(t *testing.T) {
	is := is.New(t)
	orig := fullState(t, 15)

	buf := orig.Encode()
	is.Equal(len(buf), orig.EncodedSize())
	is.True(len(buf) < MaxEncodedSize)

	var got State
	is.NoErr(got.Decode(buf))
	is.Equal(got.Digest(), orig.Digest())
	is.True(got.Equal(orig))

	// Spot-check that fields actually survived, not just that the digests of
	// two identically-broken encodings match.
	is.Equal(got.Dim(), 15)
	is.Equal(got.PlayState, WaitingForFinalPass)
	is.Equal(got.OnTurn, uint8(1))
	is.Equal(got.TurnNum, uint16(37))
	is.Equal(got.ScorelessTurns, uint8(3))
	is.Equal(got.Scores, [MaxPlayers]int32{412, 389})
	is.Equal(got.Bingos, [MaxPlayers]uint16{2, 1})
	is.Equal(got.PlayerTurns, [MaxPlayers]uint16{19, 18})
	is.Equal(got.TileAt(7, 9), tilemapping.MachineLetter(20|tilemapping.BlankMask))
	is.Equal(got.TileAt(14, 14), tilemapping.MachineLetter(26))
	is.Equal(got.Rack(0), []tilemapping.MachineLetter{1, 5, 9, 15, 21, 0, 26})
	is.Equal(got.Rack(1), []tilemapping.MachineLetter{17, 22})
	is.Equal(got.TilesRemaining(), 24)
	is.Equal(got.LastWordsFormed, orig.LastWordsFormed)
}

func TestRoundTrip21x21SuperBoard(t *testing.T) {
	is := is.New(t)
	orig := fullState(t, MaxBoardDim)

	// Fill the board densely to hit the worst case.
	for i := range orig.Board() {
		orig.Board()[i] = tilemapping.MachineLetter(i%26 + 1)
	}
	// A 200-tile super-english bag.
	counts := make([]uint8, AlphabetSize)
	for i := 1; i <= 25; i++ {
		counts[i] = 8
	}
	is.NoErr(orig.SetBagCounts(counts))
	is.Equal(orig.TilesRemaining(), 200)

	buf := orig.Encode()
	is.Equal(len(buf), orig.EncodedSize())
	// The whole point of the format: a worst-case snapshot stays comfortably
	// under Postgres's ~2KB TOAST threshold.
	is.True(len(buf) < MaxEncodedSize)
	t.Logf("worst-case 21x21 snapshot: %d bytes", len(buf))

	var got State
	is.NoErr(got.Decode(buf))
	is.Equal(got.Dim(), MaxBoardDim)
	is.Equal(got.Digest(), orig.Digest())
	is.Equal(got.TilesRemaining(), 200)
}

func TestSizeIsSmallForTypicalGame(t *testing.T) {
	is := is.New(t)
	s, err := NewState(15)
	is.NoErr(err)
	counts := make([]uint8, AlphabetSize)
	counts[0] = 2
	for i := 1; i <= 26; i++ {
		counts[i] = 3
	}
	is.NoErr(s.SetBagCounts(counts))
	is.NoErr(s.SetRack(0, []tilemapping.MachineLetter{1, 2, 3, 4, 5, 6, 7}))
	is.NoErr(s.SetRack(1, []tilemapping.MachineLetter{1, 2, 3, 4, 5, 6, 7}))

	n := len(s.Encode())
	t.Logf("typical 15x15 snapshot: %d bytes", n)
	is.True(n < 400)
}

func TestEmptyBagRoundTrips(t *testing.T) {
	is := is.New(t)
	s, err := NewState(15)
	is.NoErr(err)
	is.Equal(s.TilesRemaining(), 0)

	var got State
	is.NoErr(got.Decode(s.Encode()))
	is.Equal(got.TilesRemaining(), 0)
	is.Equal(got.Digest(), s.Digest())
}

// A blank on a rack is the zero tile; a blank on the board carries a designated
// letter in its low bits. Both must survive, and they must stay distinct.
func TestBlanksAreDistinguishable(t *testing.T) {
	is := is.New(t)
	s, err := NewState(15)
	is.NoErr(err)
	is.NoErr(s.SetRack(0, []tilemapping.MachineLetter{0, 0}))
	s.SetTileAt(0, 0, 5|tilemapping.BlankMask)
	s.SetTileAt(0, 1, 5)

	var got State
	is.NoErr(got.Decode(s.Encode()))
	is.Equal(got.Rack(0), []tilemapping.MachineLetter{0, 0})
	is.True(got.TileAt(0, 0).IsBlanked())
	is.True(!got.TileAt(0, 1).IsBlanked())
	is.Equal(got.TileAt(0, 0).Unblank(), got.TileAt(0, 1))
}

// Polish has 31 distinct tiles, well above English's 27; Catalan has multi-rune
// tiles. Both are just larger machine-letter values as far as the codec cares.
func TestLargeAlphabetRoundTrips(t *testing.T) {
	is := is.New(t)
	s, err := NewState(15)
	is.NoErr(err)
	counts := make([]uint8, AlphabetSize)
	for i := range AlphabetSize {
		counts[i] = 1
	}
	is.NoErr(s.SetBagCounts(counts))
	s.SetTileAt(0, 0, AlphabetSize-1)
	is.NoErr(s.SetRack(0, []tilemapping.MachineLetter{AlphabetSize - 1, AlphabetSize - 2}))

	var got State
	is.NoErr(got.Decode(s.Encode()))
	is.Equal(got.TilesRemaining(), AlphabetSize)
	is.Equal(got.TileAt(0, 0), tilemapping.MachineLetter(AlphabetSize-1))
}

// A newer node writes fields this build does not know about. We must not strip
// them on the way back out, or a rolling deploy silently destroys data --
// correspondence games run for weeks, so mixed versions are unavoidable.
func TestUnknownTrailingBytesArePreserved(t *testing.T) {
	is := is.New(t)
	orig := fullState(t, 15)

	buf := orig.Encode()
	future := append(buf, 0xde, 0xad, 0xbe, 0xef)
	binary.LittleEndian.PutUint16(future[1:3], uint16(len(future)))

	var got State
	is.NoErr(got.Decode(future))
	is.Equal(got.unknownTail, []byte{0xde, 0xad, 0xbe, 0xef})

	// Re-encoding preserves them byte for byte.
	is.Equal(got.Encode(), future)
}

func TestUnsupportedVersionIsRefused(t *testing.T) {
	is := is.New(t)
	buf := fullState(t, 15).Encode()
	buf[0] = 99

	var got State
	err := got.Decode(buf)
	is.True(err != nil)
	t.Log(err)
}

func TestTruncatedSnapshotIsRefused(t *testing.T) {
	is := is.New(t)
	buf := fullState(t, 15).Encode()

	// Chopping bytes off must be caught by the length header, not misparsed.
	for _, cut := range []int{1, 5, 50, len(buf) - 1} {
		var got State
		err := got.Decode(buf[:cut])
		is.True(err != nil)
	}

	// A length header that disagrees with the buffer is also rejected, even
	// though every field would parse fine.
	bad := append([]byte(nil), buf...)
	binary.LittleEndian.PutUint16(bad[1:3], uint16(len(bad)+1))
	var got State
	is.True(got.Decode(bad) != nil)
}

func TestCorruptFieldsAreRefused(t *testing.T) {
	is := is.New(t)

	// Board dimension out of range.
	s := fullState(t, 15)
	buf := s.Encode()
	dimOff := headerLen + 1 + 1 + 2 + 1 + 4*MaxPlayers + 2*MaxPlayers + 2*MaxPlayers
	is.Equal(buf[dimOff], uint8(15)) // the offset arithmetic itself
	buf[dimOff] = 99
	var got State
	is.True(got.Decode(buf) != nil)

	// On-turn player out of range.
	buf = s.Encode()
	buf[headerLen+1] = 7
	is.True(got.Decode(buf) != nil)

	// Play state out of range.
	buf = s.Encode()
	buf[headerLen] = 42
	is.True(got.Decode(buf) != nil)
}

func TestCloneIsDeep(t *testing.T) {
	is := is.New(t)
	orig := fullState(t, 15)
	c := orig.Clone()
	is.Equal(c.Digest(), orig.Digest())

	c.SetTileAt(0, 0, 12)
	c.LastWordsFormed[0][0] = 9
	is.NoErr(c.PutBack([]tilemapping.MachineLetter{1}))
	is.True(c.Digest() != orig.Digest())
	is.Equal(orig.TileAt(0, 0), tilemapping.MachineLetter(0))
	is.Equal(orig.LastWordsFormed[0][0], tilemapping.MachineLetter(3))
}

func TestDecodeReusesStateWithoutLeakage(t *testing.T) {
	is := is.New(t)
	big := fullState(t, MaxBoardDim)
	small := fullState(t, 15)

	var s State
	is.NoErr(s.Decode(big.Encode()))
	is.NoErr(s.Decode(small.Encode()))

	// No residue from the 21x21 board or the previous bag.
	is.Equal(s.Digest(), small.Digest())
	is.Equal(s.Dim(), 15)
}

func TestNewStateRejectsBadDimension(t *testing.T) {
	is := is.New(t)
	for _, dim := range []int{0, -1, MaxBoardDim + 1, 100} {
		_, err := NewState(dim)
		is.True(err != nil)
	}
}

func FuzzDecode(f *testing.F) {
	f.Add(fullState(&testing.T{}, 15).Encode())
	f.Add(fullState(&testing.T{}, MaxBoardDim).Encode())
	f.Add([]byte{1, 0, 0})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		var s State
		if err := s.Decode(data); err != nil {
			return // rejecting garbage is the correct outcome
		}
		// Anything we accept must survive a re-encode unchanged, and must
		// satisfy its own invariants.
		if err := s.Validate(); err != nil {
			t.Fatalf("decoded a state that fails validation: %v", err)
		}
		out := s.Encode()
		var again State
		if err := again.Decode(out); err != nil {
			t.Fatalf("re-decode of our own output failed: %v", err)
		}
		if again.Digest() != s.Digest() {
			t.Fatalf("round trip changed the state")
		}
	})
}

func FuzzRoundTrip(f *testing.F) {
	f.Add(15, uint8(0), uint8(0), uint16(0), uint8(0), int32(0), int32(0))
	f.Add(21, uint8(2), uint8(1), uint16(65535), uint8(6), int32(-5), int32(999))

	f.Fuzz(func(t *testing.T, dim int, ps, onTurn uint8, turnNum uint16, scoreless uint8, s0, s1 int32) {
		dim = 1 + (dim%MaxBoardDim+MaxBoardDim)%MaxBoardDim
		st, err := NewState(dim)
		if err != nil {
			t.Fatal(err)
		}
		st.PlayState = PlayState(ps % (uint8(maxPlayState) + 1))
		st.OnTurn = onTurn % MaxPlayers
		st.TurnNum = turnNum
		st.ScorelessTurns = scoreless
		st.Scores = [MaxPlayers]int32{s0, s1}

		var got State
		if err := got.Decode(st.Encode()); err != nil {
			t.Fatalf("decode of a valid state failed: %v", err)
		}
		if got.Digest() != st.Digest() {
			t.Fatalf("round trip mismatch for dim=%d", dim)
		}
		if got.Scores != st.Scores {
			t.Fatalf("scores %v != %v", got.Scores, st.Scores)
		}
	})
}

func BenchmarkEncode(b *testing.B) {
	s := fullState(&testing.T{}, 15)
	buf := make([]byte, 0, MaxEncodedSize)
	b.ReportAllocs()
	for b.Loop() {
		buf = s.AppendTo(buf[:0])
	}
}

func BenchmarkDecode(b *testing.B) {
	buf := fullState(&testing.T{}, 15).Encode()
	var s State
	b.ReportAllocs()
	for b.Loop() {
		if err := s.Decode(buf); err != nil {
			b.Fatal(err)
		}
	}
}

// seededRand is a deterministic Rand for tests.
func seededRand() Rand { return rand.New(rand.NewPCG(1, 2)) }
