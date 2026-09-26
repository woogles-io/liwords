package cwgame

import (
	"sort"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/cwgame/tiles"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func newAnnotatedGameForTest(t *testing.T) *ipc.GameDocument {
	rules := NewBasicGameRules("NWL20", "CrosswordGame", "english", ipc.ChallengeRule_ChallengeRule_FIVE_POINT,
		"classic", []int{0, 0}, 0, 0, true)
	g, err := NewGame(DefaultConfig.WGLConfig(), rules, []*ipc.GameDocument_MinimalPlayerInfo{
		{Nickname: "a", UserId: "internal-a"}, {Nickname: "b", UserId: "internal-b"}})
	if err != nil {
		t.Fatal(err)
	}
	g.Type = ipc.GameType_ANNOTATED
	g.PlayState = ipc.PlayState_PLAYING
	return g
}

func enterRacks(t *testing.T, g *ipc.GameDocument, racks ...[]byte) {
	if err := AssignRacks(DefaultConfig.WGLConfig(), g, racks, AlwaysAssignEmpty); err != nil {
		t.Fatal(err)
	}
}

func sendAnnotatedEvent(t *testing.T, g *ipc.GameDocument, e *ipc.ClientGameplayEvent) *ipc.GameEvent {
	e.GameId = g.Uid
	if err := ProcessGameplayEvent(ctxForTests(), DefaultConfig.WGLConfig(), e, g.Players[g.PlayerOnTurn].UserId, g); err != nil {
		t.Fatal(err)
	}
	if err := NewTileInventory(g, DefaultConfig.WGLConfig()).ValidateInvariants(); err != nil {
		t.Fatal(err)
	}
	return g.Events[len(g.Events)-1]
}

func pass(t *testing.T, g *ipc.GameDocument) *ipc.GameEvent {
	return sendAnnotatedEvent(t, g, &ipc.ClientGameplayEvent{Type: ipc.ClientGameplayEvent_PASS})
}

func sortedTiles(b []byte) []byte {
	c := append([]byte{}, b...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	return c
}

// drainBagToBoard moves all but `leave` pool tiles onto rows away from the
// centre, keeping tile accounting valid.
func drainBagToBoard(t *testing.T, g *ipc.GameDocument, leave int) {
	squares := []int{}
	cols := int(g.Board.NumCols)
	for _, row := range []int{0, 1, 2, 3, 11, 12, 13, 14} {
		for c := 0; c < cols; c++ {
			squares = append(squares, row*cols+c)
		}
	}
	n := len(g.Bag.Tiles) - leave
	if n > len(squares) {
		t.Fatalf("cannot drain %d tiles", n)
	}
	for i := 0; i < n; i++ {
		tile := g.Bag.Tiles[i]
		if tile == 0 {
			tile = 0x81 // a blank on the board is stored designated
		}
		g.Board.Tiles[squares[i]] = tile
	}
	g.Bag.Tiles = g.Bag.Tiles[n:]
}

// Moves entered without a rack record only the tiles they show, and no
// random tiles are ever stored on a rack.
func TestAnnotatedRacksHoldOnlyKnownTiles(t *testing.T) {
	is := is.New(t)
	g := newAnnotatedGameForTest(t)
	enterRacks(t, g, englishBytes("AEINRST"), nil)
	sendAnnotatedEvent(t, g, &ipc.ClientGameplayEvent{Type: ipc.ClientGameplayEvent_TILE_PLACEMENT,
		PositionCoords: "8D", MachineLetters: englishBytes("RETAINS")})
	is.Equal(len(g.Racks[0]), 0)
	is.Equal(len(g.Racks[1]), 0)

	evt := sendAnnotatedEvent(t, g, &ipc.ClientGameplayEvent{Type: ipc.ClientGameplayEvent_EXCHANGE,
		MachineLetters: englishBytes("QV")})
	is.Equal(sortedTiles(evt.Rack), sortedTiles(englishBytes("QV")))

	evt = sendAnnotatedEvent(t, g, &ipc.ClientGameplayEvent{Type: ipc.ClientGameplayEvent_TILE_PLACEMENT,
		PositionCoords: "E7", MachineLetters: englishBytes("D.G")})
	is.Equal(sortedTiles(evt.Rack), sortedTiles(englishBytes("DG")))

	is.Equal(len(pass(t, g).Rack), 0)
	is.Equal(len(g.Racks[0]), 0)
	is.Equal(len(g.Racks[1]), 0)
}

// An entered rack, full or partial, is recorded as entered, and its leave
// stays known.
func TestAnnotatedEnteredRackIsRecorded(t *testing.T) {
	is := is.New(t)
	g := newAnnotatedGameForTest(t)
	enterRacks(t, g, englishBytes("AEINRST"), nil)
	sendAnnotatedEvent(t, g, &ipc.ClientGameplayEvent{Type: ipc.ClientGameplayEvent_TILE_PLACEMENT,
		PositionCoords: "8D", MachineLetters: englishBytes("RETAINS")})

	enterRacks(t, g, nil, englishBytes("QVW"))
	evt := sendAnnotatedEvent(t, g, &ipc.ClientGameplayEvent{Type: ipc.ClientGameplayEvent_EXCHANGE,
		MachineLetters: englishBytes("QV")})
	is.Equal(sortedTiles(evt.Rack), sortedTiles(englishBytes("QVW")))
	is.Equal(g.Racks[1], englishBytes("W"))

	enterRacks(t, g, englishBytes("BCDEFGH"), nil)
	evt = sendAnnotatedEvent(t, g, &ipc.ClientGameplayEvent{Type: ipc.ClientGameplayEvent_EXCHANGE,
		MachineLetters: englishBytes("B")})
	is.Equal(sortedTiles(evt.Rack), sortedTiles(englishBytes("BCDEFGH")))
	is.Equal(sortedTiles(pass(t, g).Rack), englishBytes("W"))
}

// A successful challenge gives the phony's tiles back as known tiles.
func TestAnnotatedPhonyRestoresKnownRack(t *testing.T) {
	is := is.New(t)
	g := newAnnotatedGameForTest(t)
	enterRacks(t, g, englishBytes("AEINRST"), nil)
	sendAnnotatedEvent(t, g, &ipc.ClientGameplayEvent{Type: ipc.ClientGameplayEvent_TILE_PLACEMENT,
		PositionCoords: "8D", MachineLetters: englishBytes("NTRSAIE")})
	evt := sendAnnotatedEvent(t, g, &ipc.ClientGameplayEvent{Type: ipc.ClientGameplayEvent_CHALLENGE_PLAY})
	is.Equal(evt.Type, ipc.GameEvent_PHONY_TILES_RETURNED)
	is.Equal(sortedTiles(g.Racks[0]), sortedTiles(englishBytes("AEINRST")))
	is.Equal(len(g.Racks[1]), 0)
}

// Once the bag is empty, entering one rack fixes the other.
func TestAnnotatedEndgameOpponentRackIsForced(t *testing.T) {
	is := is.New(t)
	g := newAnnotatedGameForTest(t)
	drainBagToBoard(t, g, 2*RackTileLimit)
	opp := sortedTiles(g.Bag.Tiles[RackTileLimit:])
	enterRacks(t, g, append([]byte{}, g.Bag.Tiles[:RackTileLimit]...), nil)
	is.Equal(sortedTiles(g.Racks[1]), opp)

	pass(t, g)
	is.Equal(sortedTiles(pass(t, g).Rack), opp)
}

// With the bag empty and no rack entered, how the unseen tiles split is
// unknown, so nothing is recorded.
func TestAnnotatedEndgameUnenteredRacksStayUnknown(t *testing.T) {
	is := is.New(t)
	g := newAnnotatedGameForTest(t)
	drainBagToBoard(t, g, 2*RackTileLimit)
	pass(t, g)
	pass(t, g)
	for _, evt := range g.Events {
		is.Equal(len(evt.Rack), 0)
	}
	is.Equal(len(g.Racks[0])+len(g.Racks[1]), 0)
}

// The play that empties the bag leaves its player's new rack forced when the
// opponent's rack is known.
func TestAnnotatedDrawEmptyingBagForcesRack(t *testing.T) {
	is := is.New(t)
	g := newAnnotatedGameForTest(t)
	enterRacks(t, g, englishBytes("AEINRST"), englishBytes("BCDFGHL"))
	drainBagToBoard(t, g, 3)
	last := sortedTiles(g.Bag.Tiles)

	sendAnnotatedEvent(t, g, &ipc.ClientGameplayEvent{Type: ipc.ClientGameplayEvent_TILE_PLACEMENT,
		PositionCoords: "8D", MachineLetters: englishBytes("RETAINS")})
	is.Equal(tiles.InBag(g.Bag), 0)
	is.Equal(sortedTiles(g.Racks[0]), last)

	pass(t, g)
	is.Equal(sortedTiles(pass(t, g).Rack), last)
}
