package cwgame

import (
	"sort"
	"testing"

	"github.com/matryer/is"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func newAnnotatedGameForTest(t *testing.T, racks ...[]byte) *ipc.GameDocument {
	rules := NewBasicGameRules("NWL20", "CrosswordGame", "english", ipc.ChallengeRule_ChallengeRule_FIVE_POINT,
		"classic", []int{0, 0}, 0, 0, true)
	g, err := NewGame(DefaultConfig.WGLConfig(), rules, []*ipc.GameDocument_MinimalPlayerInfo{
		{Nickname: "a", UserId: "internal-a"}, {Nickname: "b", UserId: "internal-b"}})
	if err != nil {
		t.Fatal(err)
	}
	g.Type = ipc.GameType_ANNOTATED
	g.PlayState = ipc.PlayState_PLAYING
	if len(racks) > 0 {
		enterRacks(t, g, racks...)
	}
	return g
}

func enterRacks(t *testing.T, g *ipc.GameDocument, racks ...[]byte) {
	if err := AssignRacks(DefaultConfig.WGLConfig(), g, racks, AlwaysAssignEmpty); err != nil {
		t.Fatal(err)
	}
}

// annotate sends a move, or a pass if tiles is empty, and returns the rack
// it recorded.
func annotate(t *testing.T, g *ipc.GameDocument, typ ipc.ClientGameplayEvent_EventType, pos, tiles string) []byte {
	e := &ipc.ClientGameplayEvent{Type: typ, GameId: g.Uid, PositionCoords: pos}
	if tiles != "" {
		e.MachineLetters = englishBytes(tiles)
	}
	if err := ProcessGameplayEvent(ctxForTests(), DefaultConfig.WGLConfig(), e, g.Players[g.PlayerOnTurn].UserId, g); err != nil {
		t.Fatal(err)
	}
	if err := NewTileInventory(g, DefaultConfig.WGLConfig()).ValidateInvariants(); err != nil {
		t.Fatal(err)
	}
	return sorted(g.Events[len(g.Events)-1].Rack)
}

func pass(t *testing.T, g *ipc.GameDocument) []byte {
	return annotate(t, g, ipc.ClientGameplayEvent_PASS, "", "")
}

func sorted(b []byte) []byte {
	c := append([]byte{}, b...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	return c
}

// drainUnseenPoolToBoard leaves `leave` tiles in the unseen pool by moving
// the rest to rows away from the centre.
func drainUnseenPoolToBoard(g *ipc.GameDocument, leave int) {
	cols := int(g.Board.NumCols)
	sq := 0
	for _, tile := range g.Bag.Tiles[leave:] {
		if tile == 0 {
			tile = 0x81 // blanks on the board are designated
		}
		g.Board.Tiles[sq] = tile
		if sq++; sq == 4*cols {
			sq = 11 * cols
		}
	}
	g.Bag.Tiles = g.Bag.Tiles[:leave]
}

func TestAnnotatedRecordsOnlyKnownTiles(t *testing.T) {
	is := is.New(t)
	g := newAnnotatedGameForTest(t, englishBytes("AEINRST"), nil)
	is.Equal(annotate(t, g, ipc.ClientGameplayEvent_TILE_PLACEMENT, "8D", "RETAINS"), sorted(englishBytes("AEINRST")))
	is.Equal(len(g.Racks[0])+len(g.Racks[1]), 0) // nothing random is stored

	// No rack entered: only the tiles the move shows.
	is.Equal(annotate(t, g, ipc.ClientGameplayEvent_EXCHANGE, "", "QV"), sorted(englishBytes("QV")))
	is.Equal(annotate(t, g, ipc.ClientGameplayEvent_TILE_PLACEMENT, "E7", "D.G"), sorted(englishBytes("DG")))

	// A partial rack is recorded as entered, and its leave stays known.
	enterRacks(t, g, nil, englishBytes("QVW"))
	is.Equal(annotate(t, g, ipc.ClientGameplayEvent_EXCHANGE, "", "QV"), sorted(englishBytes("QVW")))
	pass(t, g)
	is.Equal(pass(t, g), englishBytes("W"))
}

func TestAnnotatedPhonyRestoresKnownRack(t *testing.T) {
	is := is.New(t)
	g := newAnnotatedGameForTest(t, englishBytes("AEINRST"), nil)
	annotate(t, g, ipc.ClientGameplayEvent_TILE_PLACEMENT, "8D", "NTRSAIE")
	annotate(t, g, ipc.ClientGameplayEvent_CHALLENGE_PLAY, "", "")
	is.Equal(g.Events[len(g.Events)-1].Type, ipc.GameEvent_PHONY_TILES_RETURNED)
	is.Equal(sorted(g.Racks[0]), sorted(englishBytes("AEINRST")))
	is.Equal(len(g.Racks[1]), 0)
}

// Once the bag is empty, a rack is known if it is the only unknown one.
func TestAnnotatedEndgameRacks(t *testing.T) {
	is := is.New(t)

	g := newAnnotatedGameForTest(t)
	drainUnseenPoolToBoard(g, 2*RackTileLimit)
	pass(t, g)
	is.Equal(len(pass(t, g)), 0) // both unknown: the split stays unknown

	g = newAnnotatedGameForTest(t)
	drainUnseenPoolToBoard(g, 2*RackTileLimit)
	opp := sorted(g.Bag.Tiles[RackTileLimit:])
	enterRacks(t, g, append([]byte{}, g.Bag.Tiles[:RackTileLimit]...), nil)
	pass(t, g)
	is.Equal(pass(t, g), opp)

	// The play that empties the bag leaves its player's rack known.
	g = newAnnotatedGameForTest(t, englishBytes("AEINRST"), englishBytes("BCDFGHL"))
	drainUnseenPoolToBoard(g, 3)
	last := sorted(g.Bag.Tiles)
	annotate(t, g, ipc.ClientGameplayEvent_TILE_PLACEMENT, "8D", "RETAINS")
	is.Equal(sorted(g.Racks[0]), last)
	is.Equal(len(g.Bag.Tiles), 0)
}

// Correcting a play in the endgame must not leave a player more tiles than
// they really hold.
func TestAnnotatedEndgameCorrectionGoesOut(t *testing.T) {
	is := is.New(t)
	g := newAnnotatedGameForTest(t, englishBytes("AEINRST"), englishBytes("BCDFGHL"))
	drainUnseenPoolToBoard(g, 0)
	annotate(t, g, ipc.ClientGameplayEvent_TILE_PLACEMENT, "8D", "TRAIN") // A keeps ES
	pass(t, g)
	// The entered ES must have been wrong: A had only 2 tiles and played B and C.
	annotate(t, g, ipc.ClientGameplayEvent_TILE_PLACEMENT, "E7", "B.C")
	is.Equal(g.PlayState, ipc.PlayState_WAITING_FOR_FINAL_PASS)
	is.Equal(sorted(g.Racks[1]), sorted(englishBytes("DEFGHLS")))
}

// Replaying the events, as amendments do, keeps both players' known tiles.
func TestAnnotatedReplayKeepsKnownTiles(t *testing.T) {
	is := is.New(t)
	g := newAnnotatedGameForTest(t, englishBytes("AEINRST"), nil)
	annotate(t, g, ipc.ClientGameplayEvent_TILE_PLACEMENT, "8D", "RETAINS")
	enterRacks(t, g, nil, englishBytes("QVW"))
	annotate(t, g, ipc.ClientGameplayEvent_EXCHANGE, "", "QV")
	pass(t, g)

	r := proto.Clone(g).(*ipc.GameDocument)
	is.NoErr(ReplayEvents(ctxForTests(), DefaultConfig.WGLConfig(), r, r.Events, false))
	is.Equal(r.Racks[1], englishBytes("W"))
}

// Amending a move forgets the tiles only the old move showed, but keeps the
// rest of a typed rack.
func TestAnnotatedAmendmentForgetsOldMoveTiles(t *testing.T) {
	is := is.New(t)
	for _, tc := range []struct{ typed, want string }{{"", "CT"}, {"DGOXYZ", "COTXYZ"}} {
		g := newAnnotatedGameForTest(t, englishBytes("AEINRST"), nil)
		annotate(t, g, ipc.ClientGameplayEvent_TILE_PLACEMENT, "8D", "RETAINS")
		if tc.typed != "" {
			enterRacks(t, g, nil, englishBytes(tc.typed))
		}
		annotate(t, g, ipc.ClientGameplayEvent_TILE_PLACEMENT, "E7", "D.G")

		old := g.Events[1]
		is.NoErr(ReplayEvents(ctxForTests(), DefaultConfig.WGLConfig(), g, g.Events[:1], false))
		is.NoErr(RestoreRackForAmendment(DefaultConfig.WGLConfig(), g, old))
		got := annotate(t, g, ipc.ClientGameplayEvent_TILE_PLACEMENT, "D7", "C.T")
		is.Equal(got, sorted(englishBytes(tc.want)))
	}
}

// An amendment starts from the rack known before the amended move, so typed
// tiles that an earlier, overfull version of the move discarded come back.
func TestAnnotatedAmendmentKeepsEarlierKnownTiles(t *testing.T) {
	is := is.New(t)
	g := newAnnotatedGameForTest(t, englishBytes("AEINRST"), englishBytes("ABCDEFG"))
	annotate(t, g, ipc.ClientGameplayEvent_TILE_PLACEMENT, "8D", "RETAINS")
	annotate(t, g, ipc.ClientGameplayEvent_TILE_PLACEMENT, "E7", "G.") // b knows ABCDEF
	pass(t, g)
	is.Equal(annotate(t, g, ipc.ClientGameplayEvent_TILE_PLACEMENT, "K8", "OX"), sorted(englishBytes("OX")))

	old := g.Events[3]
	is.NoErr(ReplayEvents(ctxForTests(), DefaultConfig.WGLConfig(), g, g.Events[:3], false))
	is.NoErr(RestoreRackForAmendment(DefaultConfig.WGLConfig(), g, old))
	is.Equal(annotate(t, g, ipc.ClientGameplayEvent_TILE_PLACEMENT, "J7", "X."), sorted(englishBytes("ABCDEFX")))
}

// A failed move must still put the top-up back: an amendment that can't
// re-apply a later move keeps the document as it is at that point.
func TestAnnotatedFailedMoveReturnsFill(t *testing.T) {
	is := is.New(t)
	g := newAnnotatedGameForTest(t, englishBytes("FAH"), nil)
	e := &ipc.ClientGameplayEvent{Type: ipc.ClientGameplayEvent_TILE_PLACEMENT, GameId: g.Uid,
		PositionCoords: "1A", MachineLetters: englishBytes("FAH")} // misses the centre square
	is.True(ProcessGameplayEvent(ctxForTests(), DefaultConfig.WGLConfig(), e, g.Players[0].UserId, g) != nil)
	is.Equal(g.Racks[0], englishBytes("FAH"))
	is.Equal(len(g.Racks[1]), 0)
	is.NoErr(NewTileInventory(g, DefaultConfig.WGLConfig()).ValidateInvariants())
}
