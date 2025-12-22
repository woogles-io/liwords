package cwgame

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"

	wglconfig "github.com/domino14/word-golib/config"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/config"

	"github.com/woogles-io/liwords/pkg/cwgame/tiles"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

var DefaultConfig = config.DefaultConfig()

func restoreGlobalNower() {
	globalNower = GameTimer{}
}

func ctxForTests() context.Context {
	ctx := context.Background()
	ctx = log.Logger.WithContext(ctx)
	ctx = DefaultConfig.WithContext(ctx)
	return ctx
}

func loadGDoc(testfilename string) *ipc.GameDocument {
	content, err := os.ReadFile("./testdata/" + testfilename)
	if err != nil {
		panic(err)
	}
	gdoc := &ipc.GameDocument{}
	err = protojson.Unmarshal(content, gdoc)
	if err != nil {
		panic(err)
	}
	return gdoc
}

// ReconcileAllTiles returns an error if the tiles on the board and on
// player racks do not match the letter distribution. It is not meant to
// be used in production, but for debugging purposes only.
func ReconcileAllTiles(cfg *wglconfig.Config, gdoc *ipc.GameDocument) error {
	dist, err := tilemapping.GetDistribution(cfg, gdoc.LetterDistribution)
	if err != nil {
		return err
	}

	bag := tiles.TileBag(dist)

	for _, t := range gdoc.Board.Tiles {
		toRm := []byte{t}
		if int8(t) < 0 {
			toRm = []byte{0}
		} else if t == 0 {
			continue
		}
		err := tiles.RemoveTiles(bag, tilemapping.FromByteArr(toRm))
		if err != nil {
			return fmt.Errorf("removing-from-board error: %w", err)
		}
	}
	for idx, rack := range gdoc.Racks {
		err := tiles.RemoveTiles(bag, tilemapping.FromByteArr(rack))
		if err != nil {
			return fmt.Errorf("removing-from-rack-%d error: %w", idx, err)
		}
	}
	if len(gdoc.Bag.Tiles) != len(bag.Tiles) {
		return fmt.Errorf("lengths dont match %d %d", len(gdoc.Bag.Tiles), len(bag.Tiles))
	}

	// No error if both bags are empty
	if len(bag.Tiles) == 0 && len(gdoc.Bag.Tiles) == 0 {
		return nil
	}
	// Otherwise sort and check the tile bags.

	sort.Slice(gdoc.Bag.Tiles, func(i, j int) bool {
		return gdoc.Bag.Tiles[i] < gdoc.Bag.Tiles[j]
	})
	sort.Slice(bag.Tiles, func(i, j int) bool {
		return bag.Tiles[i] < bag.Tiles[j]
	})

	if !reflect.DeepEqual(bag.Tiles, gdoc.Bag.Tiles) {
		return fmt.Errorf("bags aren't equal: (%v) (%v)", bag.Tiles, gdoc.Bag.Tiles)
	}
	return nil
}

func TestNewGame(t *testing.T) {
	is := is.New(t)
	rules := NewBasicGameRules("NWL20", "CrosswordGame", "english", ipc.ChallengeRule_ChallengeRule_FIVE_POINT,
		"classic", []int{300, 300}, 1, 0, false)
	g, err := NewGame(DefaultConfig.WGLConfig(), rules, []*ipc.GameDocument_MinimalPlayerInfo{
		{Nickname: "Cesitar", RealName: "Cesar", UserId: "cesar1"},
		{Nickname: "Lucas", RealName: "Lucas", UserId: "lucas1"},
	})
	is.NoErr(err)
	is.Equal(len(g.Board.Tiles), 225)
	is.Equal(len(g.Bag.Tiles), 100)
}

func TestStartGame(t *testing.T) {
	is := is.New(t)

	globalNower = &FakeNower{fakeMeow: 12345}
	defer restoreGlobalNower()

	rules := NewBasicGameRules("NWL20", "CrosswordGame", "english", ipc.ChallengeRule_ChallengeRule_FIVE_POINT,
		"classic", []int{300, 300}, 1, 0, false)
	g, _ := NewGame(DefaultConfig.WGLConfig(), rules, []*ipc.GameDocument_MinimalPlayerInfo{
		{Nickname: "Cesitar", RealName: "Cesar", UserId: "cesar1"},
		{Nickname: "Lucas", RealName: "Lucas", UserId: "lucas1"},
	})
	err := StartGame(ctxForTests(), DefaultConfig.WGLConfig(), g)
	is.NoErr(err)
	is.True(g.TimersStarted)
	is.Equal(g.Timers, &ipc.Timers{
		TimeOfLastUpdate: 12345,
		TimeStarted:      12345,
		TimeRemaining:    []int64{300000, 300000},
		MaxOvertime:      1,
		IncrementSeconds: 0,
	})
	is.Equal(len(g.Racks[0]), 7)
	is.Equal(len(g.Racks[1]), 7)
	is.Equal(len(g.Bag.Tiles), 86)
}

func englishBytes(tiles string) []byte {
	ld, err := tilemapping.GetDistribution(DefaultConfig.WGLConfig(), "english")
	if err != nil {
		panic(err)
	}
	mw, err := tilemapping.ToMachineWord(tiles, ld.TileMapping())
	if err != nil {
		panic(err)
	}
	return mw.ToByteArr()
}

func norwegianBytes(tiles string) []byte {
	ld, err := tilemapping.GetDistribution(DefaultConfig.WGLConfig(), "norwegian")
	if err != nil {
		panic(err)
	}
	mw, err := tilemapping.ToMachineWord(tiles, ld.TileMapping())
	if err != nil {
		panic(err)
	}
	return mw.ToByteArr()
}

// Try a few simple moves and make sure that basic flows work
func TestProcessGameplayEventSanityChecks(t *testing.T) {
	documentfile := "document-earlygame.json"
	testcases := []struct {
		name           string
		cge            *ipc.ClientGameplayEvent
		userID         string
		expectedErr    error
		expectedScores []int32
	}{
		{
			name: "Simple play through a tile",
			cge: &ipc.ClientGameplayEvent{
				Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
				GameId:         "9aK3YgVk",
				PositionCoords: "1D",
				MachineLetters: englishBytes("KNI.E"),
			},
			userID:         "2gJGaYnchL6LbQVTNQ6mjT",
			expectedScores: []int32{113, 137},
		},
		{
			name: "Exchange",
			cge: &ipc.ClientGameplayEvent{
				Type:           ipc.ClientGameplayEvent_EXCHANGE,
				GameId:         "9aK3YgVk",
				MachineLetters: englishBytes("EEIKNTW"),
			},
			userID:         "2gJGaYnchL6LbQVTNQ6mjT",
			expectedScores: []int32{62, 137},
		},
		{
			name: "Pass",
			cge: &ipc.ClientGameplayEvent{
				Type:   ipc.ClientGameplayEvent_PASS,
				GameId: "9aK3YgVk",
			},
			userID:         "2gJGaYnchL6LbQVTNQ6mjT",
			expectedScores: []int32{62, 137},
		},
		{
			name: "Try playing on top of existing tiles",
			cge: &ipc.ClientGameplayEvent{
				Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
				GameId:         "9aK3YgVk",
				PositionCoords: "8D",
				MachineLetters: englishBytes("TWEEN"),
			},
			userID: "2gJGaYnchL6LbQVTNQ6mjT",
			expectedErr: errors.New("tried to play through a letter already on " +
				"the board; please use the played-through marker (.) instead " +
				"(row 7 col 3 ml 6)"),
		},
		{
			name: "Try playing through a tile that doesn't exist",
			cge: &ipc.ClientGameplayEvent{
				Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
				GameId:         "9aK3YgVk",
				PositionCoords: "1D",
				MachineLetters: englishBytes("KNI.."),
			},
			userID: "2gJGaYnchL6LbQVTNQ6mjT",
			expectedErr: errors.New("a played-through marker was specified, but " +
				"there is no tile at the given location"),
		},
		{
			name: "Try playing out in space",
			cge: &ipc.ClientGameplayEvent{
				Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
				GameId:         "9aK3YgVk",
				PositionCoords: "14H",
				MachineLetters: englishBytes("WEEK"),
			},
			userID:      "2gJGaYnchL6LbQVTNQ6mjT",
			expectedErr: errors.New("your play must border a tile already on the board"),
		},
		{
			name: "Try playing a tile we don't have",
			cge: &ipc.ClientGameplayEvent{
				Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
				GameId:         "9aK3YgVk",
				PositionCoords: "1D",
				MachineLetters: englishBytes("KNI.ES"),
			},
			userID:      "2gJGaYnchL6LbQVTNQ6mjT",
			expectedErr: errors.New("you tried to play a tile (S) that is not in your rack"),
		},
		{
			name: "Try exchanging a tile we don't have",
			cge: &ipc.ClientGameplayEvent{
				Type:           ipc.ClientGameplayEvent_EXCHANGE,
				GameId:         "9aK3YgVk",
				MachineLetters: englishBytes("Q"),
			},
			userID:      "2gJGaYnchL6LbQVTNQ6mjT",
			expectedErr: errors.New("you tried to play a tile (Q) that is not in your rack"),
		},
		{
			name: "Try playing as another player not on turn",
			cge: &ipc.ClientGameplayEvent{
				Type:           ipc.ClientGameplayEvent_EXCHANGE,
				GameId:         "9aK3YgVk",
				MachineLetters: englishBytes("EEIKNTW"),
			},
			userID:      "foo",
			expectedErr: errors.New("not on turn"),
		},
		{
			name: "Try another game id",
			cge: &ipc.ClientGameplayEvent{
				Type:           ipc.ClientGameplayEvent_EXCHANGE,
				GameId:         "abc",
				MachineLetters: englishBytes("EEIKNTW"),
			},
			userID:      "2gJGaYnchL6LbQVTNQ6mjT",
			expectedErr: errors.New("game ids do not match"),
		},
	}
	// load config into context; this is needed by the gameplay functions.
	ctx := ctxForTests()

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			gdoc := loadGDoc(documentfile)
			// use a timestamp that's a little bit later than the
			// time_of_last_update in the doc.
			globalNower = &FakeNower{
				fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
			defer restoreGlobalNower()
			onturn := gdoc.PlayerOnTurn

			err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), tc.cge, tc.userID, gdoc)
			is.Equal(err, tc.expectedErr)
			if err != nil {
				// return early; nothing changes
				return
			}
			is.Equal(gdoc.CurrentScores, tc.expectedScores)
			is.Equal(gdoc.PlayerOnTurn, (onturn+uint32(1))%uint32(len(gdoc.Players)))
		})

	}
}

func TestChallengeBadWord(t *testing.T) {
	is := is.New(t)
	ctx := ctxForTests()
	documentfile := "document-earlygame.json"

	gdoc := loadGDoc(documentfile)

	// Let's say we're 5000 ms after the last time of update
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()

	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "1D",
		MachineLetters: englishBytes("KNI.EW"),
	}

	err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(2500) // second player challenges after 2500 ms

	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9aK3YgVk",
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)
	is.Equal(gdoc.ScorelessTurns, uint32(1))

	globalNower.(*FakeNower).Sleep(1500) // second player plays after challenging
	cge3 := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "15C",
		MachineLetters: englishBytes("S.HEROID"),
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge3, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)
	is.Equal(len(gdoc.Events), 7)

	is.Equal(gdoc.Events[4], &ipc.GameEvent{
		Rack:                []byte{5, 5, 9, 11, 14, 20, 23}, // EEIKNTW
		Type:                ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:          125,
		Row:                 0,
		Column:              3,
		Direction:           ipc.GameEvent_HORIZONTAL,
		Position:            "1D",
		PlayedTiles:         []byte{11, 14, 9, 0, 5, 23}, // KNI.EW
		Score:               63,
		WordsFormed:         [][]byte{{11, 14, 9, 22, 5, 23}}, // KNIVEW
		WordsFormedFriendly: []string{"KNIVEW"},
		MillisRemaining:     883808, // 5000 ms after their last time remaining
		PlayerIndex:         0,
	})
	is.Equal(gdoc.Events[5], &ipc.GameEvent{
		Type:            ipc.GameEvent_PHONY_TILES_RETURNED,
		Cumulative:      62,
		PlayerIndex:     0,
		LostScore:       63,
		Rack:            []byte{5, 5, 9, 11, 14, 20, 23}, // EEIKNTW
		PlayedTiles:     []byte{11, 14, 9, 0, 5, 23},     // KNI.EW,
		MillisRemaining: 897414,                          // 2500 ms after
	})
	is.Equal(gdoc.Events[6], &ipc.GameEvent{
		Rack:                []byte{4, 5, 8, 9, 15, 18, 19}, // DEHIORS
		Type:                ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:          229,
		Row:                 14,
		Column:              2,
		Direction:           ipc.GameEvent_HORIZONTAL,
		Position:            "15C",
		PlayedTiles:         []byte{19, 0, 8, 5, 18, 15, 9, 4}, // S.HEROID
		Score:               92,
		WordsFormed:         [][]byte{{19, 16, 8, 5, 18, 15, 9, 4}}, // SPHEROID
		WordsFormedFriendly: []string{"SPHEROID"},
		MillisRemaining:     895914, // 5000 ms after their last time remaining
		PlayerIndex:         1,
		IsBingo:             true,
	})
	is.Equal(gdoc.CurrentScores, []int32{62, 229})
	is.Equal(gdoc.ScorelessTurns, uint32(0))
	is.Equal(gdoc.Racks[0], []byte{5, 5, 9, 11, 14, 20, 23})
	is.Equal(gdoc.PlayerOnTurn, uint32(0))
	/*
		dist, err := tilemapping.GetDistribution(DefaultConfig, gdoc.LetterDistribution)
		is.NoErr(err)
		fmt.Println(board.ToUserVisibleString(gdoc.Board, gdoc.BoardLayout, dist.RuneMapping()))*/
}

func TestChallengeGoodWordSingle(t *testing.T) {
	is := is.New(t)
	ctx := ctxForTests()
	documentfile := "document-earlygame.json"

	gdoc := loadGDoc(documentfile)

	// Let's say we're 5000 ms after the last time of update
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()

	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "1D",
		MachineLetters: englishBytes("KNI.E"),
	}

	err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(2500)
	// second player challenges after 2500 ms, but nothing happens since
	// this is single challenge

	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9aK3YgVk",
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(1500)
	// second player can still play
	cge3 := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "15C",
		MachineLetters: englishBytes("S.HEROID"),
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge3, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)
	is.Equal(len(gdoc.Events), 7)

	is.Equal(gdoc.Events[4], &ipc.GameEvent{
		Rack:                []byte{5, 5, 9, 11, 14, 20, 23}, // EEIKNTW
		Type:                ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:          113,
		Row:                 0,
		Column:              3,
		Direction:           ipc.GameEvent_HORIZONTAL,
		Position:            "1D",
		PlayedTiles:         []byte{11, 14, 9, 0, 5}, // KNI.E
		Score:               51,
		WordsFormed:         [][]byte{{11, 14, 9, 22, 5}}, // KNIVE
		WordsFormedFriendly: []string{"KNIVE"},
		MillisRemaining:     883808, // 5000 ms after their last time remaining
		PlayerIndex:         0,
	})

	// Don't compare this event exactly because this event has a new random rack.
	is.Equal(gdoc.Events[5].Type, ipc.GameEvent_CHALLENGE_BONUS)
	is.Equal(gdoc.Events[5].Bonus, int32(0))
	is.Equal(gdoc.Events[5].Cumulative, int32(113))
	is.Equal(gdoc.Events[5].PlayerIndex, uint32(0))
	is.Equal(gdoc.Events[5].MillisRemaining, int32(897414))

	is.Equal(gdoc.Events[6], &ipc.GameEvent{
		Rack:                []byte{4, 5, 8, 9, 15, 18, 19}, // DEHIORS
		Type:                ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:          229,
		Row:                 14,
		Column:              2,
		Direction:           ipc.GameEvent_HORIZONTAL,
		Position:            "15C",
		PlayedTiles:         []byte{19, 0, 8, 5, 18, 15, 9, 4}, // S.HEROID
		Score:               92,
		WordsFormed:         [][]byte{{19, 16, 8, 5, 18, 15, 9, 4}}, // SPHEROID
		WordsFormedFriendly: []string{"SPHEROID"},
		MillisRemaining:     895914, // 5000 ms after their last time remaining
		PlayerIndex:         1,
		IsBingo:             true,
	})
	is.Equal(gdoc.CurrentScores, []int32{113, 229})
	is.Equal(gdoc.ScorelessTurns, uint32(0))
	is.Equal(gdoc.PlayerOnTurn, uint32(0))
}

func TestChallengeGoodWordDouble(t *testing.T) {
	is := is.New(t)
	ctx := ctxForTests()
	documentfile := "document-earlygame.json"
	gdoc := loadGDoc(documentfile)

	// overwrite challenge rule to double
	gdoc.ChallengeRule = ipc.ChallengeRule_ChallengeRule_DOUBLE

	// Let's say we're 5000 ms after the last time of update
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()

	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "1D",
		MachineLetters: englishBytes("KNI.E"),
	}

	err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(2500)
	// second player challenges after 2500 ms, and they should lose
	// their turn
	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9aK3YgVk",
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(1500)
	// first player plays again
	cge3 := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "C11",
		MachineLetters: englishBytes("TEW"), // this is their leave from their last move.
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge3, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)
	is.Equal(len(gdoc.Events), 7)

	is.Equal(gdoc.Events[4], &ipc.GameEvent{
		Rack:                []byte{5, 5, 9, 11, 14, 20, 23}, // EEIKNTW
		Type:                ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:          113,
		Row:                 0,
		Column:              3,
		Direction:           ipc.GameEvent_HORIZONTAL,
		Position:            "1D",
		PlayedTiles:         []byte{11, 14, 9, 0, 5}, // KNI.E
		Score:               51,
		WordsFormed:         [][]byte{{11, 14, 9, 22, 5}}, // KNIVE
		WordsFormedFriendly: []string{"KNIVE"},
		MillisRemaining:     883808, // 5000 ms after their last time remaining
		PlayerIndex:         0,
	})
	is.Equal(gdoc.Events[5], &ipc.GameEvent{
		Type:            ipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS,
		Cumulative:      137,
		MillisRemaining: 897414,
		PlayerIndex:     1,
		Rack:            []byte{4, 5, 8, 9, 15, 18, 19}, // DEHIORS
	})

	// Don't compare this event exactly because this event has a new random rack.
	is.Equal(gdoc.Events[6].Type, ipc.GameEvent_TILE_PLACEMENT_MOVE)
	is.Equal(gdoc.Events[6].Cumulative, int32(143))
	is.Equal(gdoc.Events[6].PlayerIndex, uint32(0))
	is.Equal(gdoc.Events[6].MillisRemaining, int32(882308))
	is.Equal(gdoc.Events[6].Score, int32(30))
	is.Equal(gdoc.Events[6].Row, int32(10))
	is.Equal(gdoc.Events[6].Column, int32(2))

	is.Equal(gdoc.CurrentScores, []int32{143, 137})
	is.Equal(gdoc.ScorelessTurns, uint32(0))
	is.Equal(gdoc.PlayerOnTurn, uint32(1))
}

func TestChallengeGoodWordNorwegian(t *testing.T) {
	is := is.New(t)
	globalNower = &FakeNower{fakeMeow: 12345}
	defer restoreGlobalNower()
	ctx := ctxForTests()
	rules := NewBasicGameRules("NSF22", "CrosswordGame", "norwegian", ipc.ChallengeRule_ChallengeRule_TEN_POINT,
		"classic", []int{300, 300}, 1, 0, false)
	g, _ := NewGame(DefaultConfig.WGLConfig(), rules, []*ipc.GameDocument_MinimalPlayerInfo{
		{Nickname: "Cesitar", RealName: "Cesar", UserId: "cesar1"},
		{Nickname: "Lucas", RealName: "Lucas", UserId: "lucas1"},
	})
	err := StartGame(ctxForTests(), DefaultConfig.WGLConfig(), g)
	is.NoErr(err)
	is.True(g.TimersStarted)
	is.Equal(g.Timers, &ipc.Timers{
		TimeOfLastUpdate: 12345,
		TimeStarted:      12345,
		TimeRemaining:    []int64{300000, 300000},
		MaxOvertime:      1,
		IncrementSeconds: 0,
	})

	// Let's say we're 5000 ms after the last time of update
	globalNower = &FakeNower{
		fakeMeow: g.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()

	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         g.Uid,
		PositionCoords: "8G",
		MachineLetters: norwegianBytes("ÅMA"),
	}
	ld, err := tilemapping.GetDistribution(DefaultConfig.WGLConfig(), "norwegian")
	is.NoErr(err)
	rack, err := tilemapping.ToMachineLetters("AÅM", ld.TileMapping())
	is.NoErr(err)

	err = AssignRacks(DefaultConfig.WGLConfig(), g, [][]byte{
		tilemapping.MachineWord(rack).ToByteArr(),
		{},
	}, AlwaysAssignEmpty)
	is.NoErr(err)

	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, "cesar1", g)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(2500)
	// second player challenges after 2500 ms, and first player should gain 10 pts
	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: g.Uid,
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge2, "lucas1", g)
	is.NoErr(err)

	fmt.Println(g.Events)
	is.Equal(len(g.Events), 2)
	// Play was worth 14 but P1 got a 10-pt bonus
	is.Equal(g.CurrentScores, []int32{24, 0})
	// Player 2 is still on turn. (0-indexed)
	is.Equal(g.PlayerOnTurn, uint32(1))

}

func TestChallengeGoodWordDoubleWithTimeIncrement(t *testing.T) {
	is := is.New(t)
	ctx := ctxForTests()
	documentfile := "document-earlygame.json"
	gdoc := loadGDoc(documentfile)

	// overwrite challenge rule to double
	gdoc.ChallengeRule = ipc.ChallengeRule_ChallengeRule_DOUBLE
	gdoc.Timers.IncrementSeconds = 6

	// Let's say we're 5000 ms after the last time of update
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()

	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "1D",
		MachineLetters: englishBytes("KNI.E"),
	}

	err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(2500)
	// second player challenges after 2500 ms, and they should lose
	// their turn
	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9aK3YgVk",
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)

	is.Equal(len(gdoc.Events), 6)

	is.True(proto.Equal(gdoc.Events[4], &ipc.GameEvent{
		Rack:                []byte{5, 5, 9, 11, 14, 20, 23}, // EEIKNTW
		Type:                ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:          113,
		Row:                 0,
		Column:              3,
		Direction:           ipc.GameEvent_HORIZONTAL,
		Position:            "1D",
		PlayedTiles:         []byte{11, 14, 9, 0, 5}, // KNI.E
		Score:               51,
		WordsFormed:         [][]byte{{11, 14, 9, 22, 5}}, // KNIVE
		WordsFormedFriendly: []string{"KNIVE"},
		// 5000 ms after their last time remaining
		MillisRemaining: 883808,
		PlayerIndex:     0,
	}))
	is.True(proto.Equal(gdoc.Events[5], &ipc.GameEvent{
		Type:       ipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS,
		Cumulative: 137,
		// 2.5 second sleep.
		MillisRemaining: 897414,
		PlayerIndex:     1,
		Rack:            []byte{4, 5, 8, 9, 15, 18, 19}, // DEHIORS
	}))

	is.Equal(gdoc.CurrentScores, []int32{113, 137})
	is.Equal(gdoc.ScorelessTurns, uint32(1))
	is.Equal(gdoc.PlayerOnTurn, uint32(0))
	// Player 1 (who made the move) gets 6 seconds back from the increment
	// Player 2 (who challenged) loses their turn, but still gets the 6 seconds
	is.Equal(gdoc.Timers.TimeRemaining, []int64{889808, 903414})
}

func TestChallengeBadWordWithTimeIncrement(t *testing.T) {
	is := is.New(t)
	ctx := ctxForTests()
	documentfile := "document-earlygame.json"
	gdoc := loadGDoc(documentfile)

	gdoc.Timers.IncrementSeconds = 6

	// Let's say we're 5000 ms after the last time of update
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()

	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "1D",
		MachineLetters: englishBytes("KNI.EW"),
	}

	err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(2500)
	// second player challenges after 2500 ms, and they should lose
	// their turn
	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9aK3YgVk",
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)

	is.Equal(len(gdoc.Events), 6)

	is.Equal(gdoc.Events[4], &ipc.GameEvent{
		Rack:                []byte{5, 5, 9, 11, 14, 20, 23}, // EEIKNTW
		Type:                ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:          125,
		Row:                 0,
		Column:              3,
		Direction:           ipc.GameEvent_HORIZONTAL,
		Position:            "1D",
		PlayedTiles:         []byte{11, 14, 9, 0, 5, 23}, // KNI.EW
		Score:               63,
		WordsFormed:         [][]byte{{11, 14, 9, 22, 5, 23}}, // KNIVEW
		WordsFormedFriendly: []string{"KNIVEW"},
		// 5000 ms after their last time remaining
		MillisRemaining: 883808,
		PlayerIndex:     0,
	})
	is.Equal(gdoc.Events[5], &ipc.GameEvent{
		Type:       ipc.GameEvent_PHONY_TILES_RETURNED,
		Cumulative: 62,
		// 2.5 second sleep for the person who challenged. This timer is
		// the timer for the challenger:
		MillisRemaining: 897414,
		PlayerIndex:     0,
		LostScore:       63,
		Rack:            []byte{5, 5, 9, 11, 14, 20, 23}, // EEIKNTW
		PlayedTiles:     []byte{11, 14, 9, 0, 5, 23},     // KNI.EW,
	})

	is.Equal(gdoc.CurrentScores, []int32{62, 137})
	is.Equal(gdoc.ScorelessTurns, uint32(1))
	is.Equal(gdoc.PlayerOnTurn, uint32(1))
	// Player 1 (who made the move) gets 6 seconds back from the increment
	// Player 2 (who challenged) does not get any time back until they make a move:
	is.Equal(gdoc.Timers.TimeRemaining, []int64{889808, 897414})
}

func TestTimeRanOut(t *testing.T) {
	is := is.New(t)
	ctx := ctxForTests()
	documentfile := "document-earlygame.json"

	gdoc := loadGDoc(documentfile)

	// Let's say we're 5000 ms after the last time of update
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()

	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "1D",
		MachineLetters: englishBytes("KNI.EW"),
	}

	err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(899914 + 60001)
	// second player challenges way too late, they went overtime
	// by a minute and 1 millisecond (remember this doc has max permitted
	// overtime equal to 1 minute)
	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9aK3YgVk",
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)

	is.Equal(len(gdoc.Events), 6)

	is.Equal(gdoc.Events[4], &ipc.GameEvent{
		Rack:                []byte{5, 5, 9, 11, 14, 20, 23}, // EEIKNTW
		Type:                ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:          125,
		Row:                 0,
		Column:              3,
		Direction:           ipc.GameEvent_HORIZONTAL,
		Position:            "1D",
		PlayedTiles:         []byte{11, 14, 9, 0, 5, 23}, // KNI.EW
		Score:               63,
		WordsFormed:         [][]byte{{11, 14, 9, 22, 5, 23}}, // KNIVEW
		MillisRemaining:     883808,                           // 5000 ms after their last time remaining
		PlayerIndex:         0,
		WordsFormedFriendly: []string{"KNIVEW"},
	})

	is.Equal(gdoc.Events[5], &ipc.GameEvent{
		Type:            ipc.GameEvent_TIMED_OUT,
		PlayerIndex:     1,
		MillisRemaining: -60000,
	})
	is.Equal(gdoc.Winner, int32(0))
	is.Equal(gdoc.EndReason, ipc.GameEndReason_TIME)
	is.Equal(gdoc.PlayState, ipc.PlayState_GAME_OVER)
	is.Equal(gdoc.CurrentScores, []int32{125, 137}) // but P2 ran out of time and lost

	// Try to make another move. It should be an error
	cge3 := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "1D",
		MachineLetters: englishBytes("FOO"), // doesn't matter
	}

	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge3, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.Equal(err, errGameNotActive)
}

func TestChallengeBadWordEndOfGame(t *testing.T) {
	is := is.New(t)
	ctx := ctxForTests()
	documentfile := "document-game-almost-over.json"

	gdoc := loadGDoc(documentfile)

	// Let's say we're 5000 ms after the last time of update
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()

	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9zaaSuN5",
		PositionCoords: "12A",
		MachineLetters: englishBytes("AER.OITh"),
	}

	err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)

	is.Equal(len(gdoc.Events), 25)
	is.Equal(gdoc.PlayState, ipc.PlayState_WAITING_FOR_FINAL_PASS)
	is.Equal(gdoc.CurrentScores, []int32{446, 303})

	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9zaaSuN5",
	}

	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)

	is.Equal(len(gdoc.Events), 26)
	is.Equal(gdoc.PlayState, ipc.PlayState_PLAYING)
	is.Equal(gdoc.PlayerOnTurn, uint32(0))
	is.Equal(gdoc.Events[25].Type, ipc.GameEvent_PHONY_TILES_RETURNED)
	// CEPRT and ?AEILOR
	is.Equal(gdoc.Racks, [][]byte{{3, 5, 16, 18, 20}, {0, 1, 5, 9, 15, 18, 20}})

	cge3 := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9zaaSuN5",
		PositionCoords: "14J",
		MachineLetters: englishBytes("PR..CE"),
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge3, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)
	is.Equal(gdoc.PlayState, ipc.PlayState_PLAYING)
	is.Equal(gdoc.PlayerOnTurn, uint32(1))
	is.Equal(gdoc.CurrentScores, []int32{478, 245})
}

func TestChallengeGoodWordEndOfGame(t *testing.T) {
	ctx := ctxForTests()
	documentfile := "document-game-almost-over.json"

	for _, chrule := range []ipc.ChallengeRule{
		ipc.ChallengeRule_ChallengeRule_FIVE_POINT,
		ipc.ChallengeRule_ChallengeRule_SINGLE,
		ipc.ChallengeRule_ChallengeRule_DOUBLE,
	} {
		t.Run(ipc.ChallengeRule_name[int32(chrule)], func(t *testing.T) {
			is := is.New(t)

			gdoc := loadGDoc(documentfile)

			// Let's say we're 5000 ms after the last time of update
			globalNower = &FakeNower{
				fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
			defer restoreGlobalNower()
			gdoc.ChallengeRule = chrule

			cge := &ipc.ClientGameplayEvent{
				Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
				GameId:         "9zaaSuN5",
				PositionCoords: "12F",
				MachineLetters: englishBytes("TRIAlO..E"),
			}

			err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
			is.NoErr(err)

			is.Equal(len(gdoc.Events), 25)
			is.Equal(gdoc.PlayState, ipc.PlayState_WAITING_FOR_FINAL_PASS)
			is.Equal(gdoc.CurrentScores, []int32{446, 305})

			cge2 := &ipc.ClientGameplayEvent{
				Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
				GameId: "9zaaSuN5",
			}

			err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
			is.NoErr(err)

			is.Equal(len(gdoc.Events), 27)
			is.Equal(gdoc.PlayState, ipc.PlayState_GAME_OVER)
			bonus := 0
			if chrule == ipc.ChallengeRule_ChallengeRule_FIVE_POINT {
				bonus = 5
			}
			if chrule != ipc.ChallengeRule_ChallengeRule_DOUBLE {
				is.True(proto.Equal(gdoc.Events[25], &ipc.GameEvent{
					Type:            ipc.GameEvent_CHALLENGE_BONUS,
					PlayerIndex:     1, // the person being challenged
					Bonus:           int32(bonus),
					Rack:            []byte{},
					Cumulative:      305 + int32(bonus),
					MillisRemaining: 899671,
				}))
			} else {
				is.True(proto.Equal(gdoc.Events[25], &ipc.GameEvent{
					Type:            ipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS,
					PlayerIndex:     0,
					Rack:            []byte{3, 5, 16, 18, 20},
					Cumulative:      446,
					MillisRemaining: 899671,
				}))
			}

			is.Equal(gdoc.Events[26], &ipc.GameEvent{
				Type:          ipc.GameEvent_END_RACK_PTS,
				EndRackPoints: 18,
				PlayerIndex:   1,
				Rack:          []byte{3, 5, 16, 18, 20},
				Cumulative:    323 + int32(bonus),
			})

			is.Equal(gdoc.CurrentScores, []int32{446, 323 + int32(bonus)})
			is.Equal(gdoc.Winner, int32(0))
			is.Equal(gdoc.EndReason, ipc.GameEndReason_STANDARD)
		})
	}
}

// Load a document that has a challenge or pass state, it should work properly.
func TestLoadChallengeOrPassState(t *testing.T) {
	ctx := ctxForTests()
	documentfile := "document-challenge-or-pass.json"

	for _, finalmove := range []ipc.ClientGameplayEvent_EventType{
		ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		ipc.ClientGameplayEvent_PASS,
	} {
		t.Run(ipc.ClientGameplayEvent_EventType_name[int32(finalmove)], func(t *testing.T) {
			is := is.New(t)

			gdoc := loadGDoc(documentfile)

			// Let's say we're 5000 ms after the last time of update
			globalNower = &FakeNower{
				fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
			defer restoreGlobalNower()

			cge := &ipc.ClientGameplayEvent{
				Type:   finalmove,
				GameId: "9zaaSuN5",
			}

			err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
			is.NoErr(err)

			is.Equal(len(gdoc.Events), 27)
			is.Equal(gdoc.PlayState, ipc.PlayState_GAME_OVER)

			is.Equal(gdoc.Events[26], &ipc.GameEvent{
				Type:          ipc.GameEvent_END_RACK_PTS,
				EndRackPoints: 18,
				PlayerIndex:   1,
				Rack:          []byte{3, 5, 16, 18, 20},
				Cumulative:    322,
			})

			is.Equal(gdoc.CurrentScores, []int32{446, 322})
			is.Equal(gdoc.Winner, int32(0))
			is.Equal(gdoc.EndReason, ipc.GameEndReason_STANDARD)
		})
	}
}

func TestRejectNonChallOrPass(t *testing.T) {
	ctx := ctxForTests()
	documentfile := "document-challenge-or-pass.json"

	is := is.New(t)

	gdoc := loadGDoc(documentfile)

	// Let's say we're 5000 ms after the last time of update
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()

	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		PositionCoords: "14J",
		MachineLetters: englishBytes("PR..CE"),
		GameId:         "9zaaSuN5",
	}

	err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.Equal(err, errOnlyPassOrChallenge)
}

func TestConsecutiveScorelessTurns(t *testing.T) {
	ctx := ctxForTests()
	documentfile := "document-earlygame.json"

	is := is.New(t)
	gdoc := loadGDoc(documentfile)

	// Let's say we're 5000 ms after the last time of update
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()

	passCGE := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_PASS,
		GameId: "9aK3YgVk",
	}
	err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), passCGE, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)
	is.Equal(gdoc.ScorelessTurns, uint32(1))
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), passCGE, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)
	is.Equal(gdoc.ScorelessTurns, uint32(2))

	phonyCGE := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "1D",
		MachineLetters: englishBytes("KNI.EW"),
	}

	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), phonyCGE, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)
	is.Equal(gdoc.ScorelessTurns, uint32(0))
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9aK3YgVk",
	}, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)
	is.Equal(gdoc.ScorelessTurns, uint32(3))
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), passCGE, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)
	is.Equal(gdoc.ScorelessTurns, uint32(4))
	exchangeCGE := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_EXCHANGE,
		GameId:         "9aK3YgVk",
		MachineLetters: englishBytes("KW"),
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), exchangeCGE, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)
	is.Equal(gdoc.ScorelessTurns, uint32(5))
	// now play a phony and challenge it off
	phonyCGE = &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "15C",
		MachineLetters: englishBytes("S.HERIOD"),
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), phonyCGE, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)
	is.Equal(gdoc.ScorelessTurns, uint32(0))
	chCGE := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9aK3YgVk",
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), chCGE, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)
	is.Equal(gdoc.ScorelessTurns, uint32(6))
	is.Equal(gdoc.EndReason, ipc.GameEndReason_CONSECUTIVE_ZEROES)
	// player 1 loses some random amount due to their exchange.
	// player 2 loses 11 pts (DEHIORS)
	is.True(gdoc.CurrentScores[0] < int32(57))
	is.Equal(gdoc.CurrentScores[1], int32(126))
}

func TestResign(t *testing.T) {
	ctx := ctxForTests()
	documentfile := "document-earlygame.json"

	testcases := []struct {
		name        string
		resignerid  string
		resigneridx int
		winneridx   int32
		millisrem   int
		expectedErr error
	}{
		{
			name:        "player on turn resigns",
			resignerid:  "2gJGaYnchL6LbQVTNQ6mjT",
			resigneridx: 0,
			winneridx:   1,
			millisrem:   883808,
		},
		{
			name:        "player not on turn resigns",
			resignerid:  "FDHvxexaC5QNMfiJnpcnUZ",
			resigneridx: 1,
			winneridx:   0,
			millisrem:   899914,
		},
		{
			name:        "some observer resigns",
			resignerid:  "foo",
			resigneridx: -1,
			expectedErr: errPlayerNotInGame,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			gdoc := loadGDoc(documentfile)

			// Let's say we're 5000 ms after the last time of update
			globalNower = &FakeNower{
				fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
			defer restoreGlobalNower()

			resignCGE := &ipc.ClientGameplayEvent{
				Type:   ipc.ClientGameplayEvent_RESIGN,
				GameId: "9aK3YgVk",
			}
			err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), resignCGE, tc.resignerid, gdoc)
			if err != nil {
				is.Equal(err, tc.expectedErr)
				return
			}
			is.NoErr(err)
			is.Equal(gdoc.EndReason, ipc.GameEndReason_RESIGNED)
			is.Equal(gdoc.PlayState, ipc.PlayState_GAME_OVER)
			is.Equal(gdoc.Winner, tc.winneridx)
			is.Equal(gdoc.CurrentScores, []int32{62, 137})
			is.Equal(len(gdoc.Events), 5)
			is.Equal(gdoc.Events[4], &ipc.GameEvent{
				Type:            ipc.GameEvent_RESIGNED,
				MillisRemaining: int32(tc.millisrem),
				PlayerIndex:     uint32(tc.resigneridx),
			})
		})
	}
}

func TestVoidChallenge(t *testing.T) {
	is := is.New(t)
	ctx := ctxForTests()
	documentfile := "document-earlygame.json"

	gdoc := loadGDoc(documentfile)
	gdoc.ChallengeRule = ipc.ChallengeRule_ChallengeRule_VOID
	// Let's say we're 5000 ms after the last time of update
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()

	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "1D",
		MachineLetters: englishBytes("KNI.EW"),
	}

	err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.Equal(err.Error(), "invalid words: KNIVEW")

	cge2 := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "1D",
		MachineLetters: englishBytes("KNI.E"),
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge2, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)
	is.Equal(gdoc.CurrentScores, []int32{113, 137})
}

func TestVoidChallengeEndOfGame(t *testing.T) {
	is := is.New(t)
	ctx := ctxForTests()
	documentfile := "document-game-almost-over.json"

	gdoc := loadGDoc(documentfile)
	gdoc.ChallengeRule = ipc.ChallengeRule_ChallengeRule_VOID
	// Let's say we're 5000 ms after the last time of update
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()

	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9zaaSuN5",
		PositionCoords: "12F",
		MachineLetters: englishBytes("TRIAlO..E"),
	}

	err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)
	is.Equal(gdoc.EndReason, ipc.GameEndReason_STANDARD)
	is.Equal(gdoc.PlayState, ipc.PlayState_GAME_OVER)
	is.Equal(gdoc.CurrentScores, []int32{446, 323})
	is.Equal(gdoc.Winner, int32(0))
}

func TestTripleChallenge(t *testing.T) {
	is := is.New(t)
	ctx := ctxForTests()
	documentfile := "document-game-almost-over.json"

	gdoc := loadGDoc(documentfile)
	gdoc.ChallengeRule = ipc.ChallengeRule_ChallengeRule_TRIPLE
	// Let's say we're 5000 ms after the last time of update
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()

	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9zaaSuN5",
		PositionCoords: "12F",
		MachineLetters: englishBytes("TRIAlO..E"),
	}

	err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)

	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9zaaSuN5",
	}
	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)

	is.Equal(gdoc.EndReason, ipc.GameEndReason_TRIPLE_CHALLENGE)
	is.Equal(gdoc.PlayState, ipc.PlayState_GAME_OVER)
	is.Equal(gdoc.CurrentScores, []int32{446, 305})
	// Player indexed 1 wins even though they had fewer points, because of
	// the triple challenge rule
	is.Equal(gdoc.Winner, int32(1))
}

func TestExchange(t *testing.T) {
	is := is.New(t)
	documentfile := "document-earlygame.json"
	gdoc := loadGDoc(documentfile)
	// use a timestamp that's a little bit later than the
	// time_of_last_update in the doc.
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()
	ctx := ctxForTests()

	// This player's rack is EEIKNTW
	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_EXCHANGE,
		GameId:         "9aK3YgVk",
		MachineLetters: englishBytes("EKW"),
	}
	userID := "2gJGaYnchL6LbQVTNQ6mjT"

	err := ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, userID, gdoc)
	is.NoErr(err)
	fmt.Println(gdoc.Events[len(gdoc.Events)-1])
	is.True(proto.Equal(gdoc.Events[len(gdoc.Events)-1], &ipc.GameEvent{
		Rack:            []byte{5, 5, 9, 11, 14, 20, 23},
		Type:            ipc.GameEvent_EXCHANGE,
		Cumulative:      62,
		Exchanged:       []byte{5, 11, 23},
		MillisRemaining: 883808,
	}))
	err = ReconcileAllTiles(DefaultConfig.WGLConfig(), gdoc)
	is.NoErr(err)
}

func TestExchangePartialRack(t *testing.T) {
	is := is.New(t)
	documentfile := "document-earlygame.json"
	gdoc := loadGDoc(documentfile)
	// use a timestamp that's a little bit later than the
	// time_of_last_update in the doc.
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()
	ctx := ctxForTests()

	err := AssignRacks(DefaultConfig.WGLConfig(), gdoc, [][]byte{{9, 9, 9, 9, 9}, nil}, NeverAssignEmpty)
	is.NoErr(err)

	// This player's rack is IIIII
	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_EXCHANGE,
		GameId:         "9aK3YgVk",
		MachineLetters: englishBytes("IIIII"),
	}
	userID := "2gJGaYnchL6LbQVTNQ6mjT"

	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, userID, gdoc)
	is.NoErr(err)
	fmt.Println(gdoc.Events[len(gdoc.Events)-1])
	is.True(proto.Equal(gdoc.Events[len(gdoc.Events)-1], &ipc.GameEvent{
		Rack:            []byte{9, 9, 9, 9, 9},
		Type:            ipc.GameEvent_EXCHANGE,
		Cumulative:      62,
		Exchanged:       []byte{9, 9, 9, 9, 9},
		MillisRemaining: 883808,
	}))
	err = ReconcileAllTiles(DefaultConfig.WGLConfig(), gdoc)
	is.NoErr(err)
}

func TestExchangeBlank(t *testing.T) {
	is := is.New(t)
	documentfile := "document-earlygame.json"
	gdoc := loadGDoc(documentfile)
	// use a timestamp that's a little bit later than the
	// time_of_last_update in the doc.
	globalNower = &FakeNower{
		fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
	defer restoreGlobalNower()
	ctx := ctxForTests()

	err := AssignRacks(DefaultConfig.WGLConfig(), gdoc, [][]byte{{0, 1, 2, 3, 4, 5}, nil}, NeverAssignEmpty)
	is.NoErr(err)

	// This player's rack is ?ABCDE
	cge := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_EXCHANGE,
		GameId:         "9aK3YgVk",
		MachineLetters: englishBytes("?CE"),
	}
	userID := "2gJGaYnchL6LbQVTNQ6mjT"

	err = ProcessGameplayEvent(ctx, DefaultConfig.WGLConfig(), cge, userID, gdoc)
	is.NoErr(err)
	fmt.Println(gdoc.Events[len(gdoc.Events)-1])
	is.True(proto.Equal(gdoc.Events[len(gdoc.Events)-1], &ipc.GameEvent{
		Rack:            []byte{0, 1, 2, 3, 4, 5},
		Type:            ipc.GameEvent_EXCHANGE,
		Cumulative:      62,
		Exchanged:       []byte{0, 3, 5},
		MillisRemaining: 883808,
	}))
	err = ReconcileAllTiles(DefaultConfig.WGLConfig(), gdoc)
	is.NoErr(err)
}

func TestAssignRacks(t *testing.T) {
	is := is.New(t)

	dist, err := tilemapping.GetDistribution(DefaultConfig.WGLConfig(), "English")
	is.NoErr(err)

	doc := &ipc.GameDocument{
		Players: []*ipc.GameDocument_MinimalPlayerInfo{
			{Nickname: "abc", RealName: "abc", UserId: "abc"},
			{Nickname: "ijk", RealName: "ijk", UserId: "ijk"},
		},
		Racks:              make([][]byte, 2),
		Bag:                tiles.TileBag(dist),
		LetterDistribution: "English",
	}
	err = AssignRacks(DefaultConfig.WGLConfig(), doc, [][]byte{
		{1, 1, 1, 1, 1, 1, 1},
		{5, 5, 5, 5, 5, 5, 5},
	}, AlwaysAssignEmpty)
	is.NoErr(err)
	is.Equal(len(doc.Bag.Tiles), 86)
	is.Equal(tiles.Count(doc.Bag, 1), 2)
	is.Equal(tiles.Count(doc.Bag, 5), 5)
}

func TestAssignRacksEmptyRack(t *testing.T) {
	is := is.New(t)

	dist, err := tilemapping.GetDistribution(DefaultConfig.WGLConfig(), "English")
	is.NoErr(err)

	doc := &ipc.GameDocument{
		Players: []*ipc.GameDocument_MinimalPlayerInfo{
			{Nickname: "abc", RealName: "abc", UserId: "abc"},
			{Nickname: "ijk", RealName: "ijk", UserId: "ijk"},
		},
		Racks:              make([][]byte, 2),
		Bag:                tiles.TileBag(dist),
		LetterDistribution: "English",
	}
	err = AssignRacks(DefaultConfig.WGLConfig(), doc, [][]byte{
		nil,
		{5, 5, 5, 5, 5, 5, 5},
	}, AlwaysAssignEmpty)
	is.NoErr(err)
	is.Equal(len(doc.Bag.Tiles), 86)
	is.True(tiles.Count(doc.Bag, 5) <= 5)
}

func TestAssignRacksIfBagEmpty(t *testing.T) {
	is := is.New(t)

	gdoc := loadGDoc("document-game-almost-over.json")

	err := AssignRacks(DefaultConfig.WGLConfig(), gdoc, [][]byte{
		nil,
		{1, 3, 5, 9, 15, 18, 20},
	}, AssignEmptyIfUnambiguous)
	is.NoErr(err)
	is.Equal(len(gdoc.Bag.Tiles), 0)
	sort.Slice(gdoc.Racks[0], func(i, j int) bool { return gdoc.Racks[0][i] < gdoc.Racks[0][j] })
	is.Equal(gdoc.Racks, [][]byte{
		{0, 5, 16, 18, 20},
		{1, 3, 5, 9, 15, 18, 20},
	})
	is.NoErr(ReconcileAllTiles(DefaultConfig.WGLConfig(), gdoc))
}

func TestAssignRacksIfBagAmbiguous(t *testing.T) {
	is := is.New(t)

	gdoc := loadGDoc("document-game-almost-over.json")

	err := AssignRacks(DefaultConfig.WGLConfig(), gdoc, [][]byte{
		nil,
		{1, 3, 5, 9},
	}, AssignEmptyIfUnambiguous)
	is.NoErr(err)
	// There are 8 tiles in the bag. We don't know which ones to assign to P1,
	// so we will not assign any.
	is.Equal(len(gdoc.Bag.Tiles), 8)
	is.Equal(gdoc.Racks, [][]byte{
		nil,
		{1, 3, 5, 9},
	})
	is.NoErr(ReconcileAllTiles(DefaultConfig.WGLConfig(), gdoc))
}

func TestReplayEvents(t *testing.T) {
	is := is.New(t)

	testcases := []string{
		"document-game-almost-over.json",
		"document-challenge-or-pass.json",
		"document-earlygame.json",
		"document-gameover.json",
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			gdoc := loadGDoc(tc)
			gdocClone := loadGDoc(tc)

			ctx := ctxForTests()
			err := ReplayEvents(ctx, config.DefaultConfig().WGLConfig(), gdoc, gdocClone.Events, true)
			is.NoErr(err)
			tiles.Sort(gdoc.Bag)
			tiles.Sort(gdocClone.Bag)

			is.True(proto.Equal(gdoc, gdocClone))

		})
	}
}

func TestEditOldRack(t *testing.T) {
	is := is.New(t)
	gdoc := loadGDoc("document-game-almost-over.json")
	ctx := ctxForTests()

	err := EditOldRack(ctx, DefaultConfig.WGLConfig(), gdoc, 0, []byte{1, 2, 3, 4, 5, 6, 7})
	is.NoErr(err)
	is.Equal(gdoc.Events[0].Rack, []byte{1, 2, 3, 4, 5, 6, 7})
	is.Equal(gdoc.Events[0].PlayedTiles, []byte{1, 18, 5, 14, 15, 19, 5})
	// The rack doesn't match the played tiles, but this is fine. I can't
	// think of another way to edit an old play.
}

func TestEditOldRackDisallowed(t *testing.T) {
	is := is.New(t)
	gdoc := loadGDoc("document-game-almost-over.json")
	ctx := ctxForTests()

	// Try to set the rack for event indexed 8 to JKL. It shouldn't
	// let you, because event index 7 already used a J.
	err := EditOldRack(ctx, DefaultConfig.WGLConfig(), gdoc, 8, []byte{10, 11, 12})
	is.Equal(err.Error(), "your rack could not be set, you tried to add a tile (J) that is not in the bag")
	err = EditOldRack(ctx, DefaultConfig.WGLConfig(), gdoc, 7, []byte{10, 11, 12})
	is.NoErr(err)
	is.Equal(gdoc.Events[7].Rack, []byte{10, 11, 12})
}

func TestToCGP(t *testing.T) {
	is := is.New(t)
	gdoc := loadGDoc("document-game-almost-over.json")

	cgp, err := ToCGP(DefaultConfig.WGLConfig(), gdoc)
	is.NoErr(err)
	is.Equal(cgp, "V6K7/EL5N1JAGG2/TA1V2HO3RUIN/1Z1I2OWE1QI2I/1YELPED3IT2T/3DON4N2AR/4UM1BUYS2WO/3ARENOSE2SEX/4IS6A2/3MEH6g2/3O8O2/3L7GU2/3D7LI2/FACETE5IN2/3DARAF3B3 ?AEIORT/CEPRT 245/446 0 lex CSW21; ld english;")
}

func BenchmarkLoadDocumentJSON(b *testing.B) {
	is := is.New(b)
	documentfile := "document-earlygame.json"
	content, err := os.ReadFile("./testdata/" + documentfile)
	is.NoErr(err)
	// ~41.7 us per op on themonolith (12th-gen intel box)
	for i := 0; i < b.N; i++ {
		gdoc := &ipc.GameDocument{}
		protojson.Unmarshal(content, gdoc)
	}
}

func BenchmarkLoadDocumentProto(b *testing.B) {
	is := is.New(b)
	documentfile := "document-earlygame.pb"
	content, err := os.ReadFile("./testdata/" + documentfile)
	is.NoErr(err)
	// ~4.3 us per op on themonolith (12th-gen intel box)
	for i := 0; i < b.N; i++ {
		gdoc := &ipc.GameDocument{}
		proto.Unmarshal(content, gdoc)
	}
}

func TestInferRackForPlay(t *testing.T) {
	is := is.New(t)
	cfg := DefaultConfig.WGLConfig()

	// Create a fresh game with empty board
	rules := &GameRules{
		boardLayout:       "CrosswordGame",
		distname:          "english",
		lexicon:           "CSW21",
		variant:           "classic",
		secondsPerPlayer:  []int{900, 900},
		maxOvertimeMins:   10,
		incrementSeconds:  0,
		untimed:           false,
		challengeRule:     ipc.ChallengeRule_ChallengeRule_FIVE_POINT,
	}

	playerInfo := []*ipc.GameDocument_MinimalPlayerInfo{
		{UserId: "player1", Nickname: "Player1"},
		{UserId: "player2", Nickname: "Player2"},
	}

	gdoc, err := NewGame(cfg, rules, playerInfo)
	is.NoErr(err)

	// Scenario 1: Play a word without any through-tiles on empty board
	// The inferred rack should contain all played tiles
	playedTiles := []tilemapping.MachineLetter{1, 2, 3, 4} // A, B, C, D
	inferredRack, err := InferRackForPlay(gdoc, 5, 5, ipc.GameEvent_HORIZONTAL, playedTiles)
	is.NoErr(err)
	is.Equal(len(inferredRack), 4) // All 4 tiles must be from rack

	// Scenario 2: Play through an existing tile using the through-tile marker (0)
	// First, manually place a tile on the board at position (10, 5)
	gdoc.Board.Tiles[10*int(gdoc.Board.NumCols)+5] = byte(tilemapping.MachineLetter('E' - 'A' + 1))

	// Now play "TEST" horizontally starting at (10, 4), which plays through the 'E' at (10, 5)
	// T at (10, 4), 0 at (10, 5) [through-tile marker], S at (10, 6), T at (10, 7)
	// The frontend sends 0 for through-tiles, not the actual tile value
	playedTiles = []tilemapping.MachineLetter{
		tilemapping.MachineLetter('T' - 'A' + 1),
		0, // Through-tile marker
		tilemapping.MachineLetter('S' - 'A' + 1),
		tilemapping.MachineLetter('T' - 'A' + 1),
	}
	inferredRack, err = InferRackForPlay(gdoc, 10, 4, ipc.GameEvent_HORIZONTAL, playedTiles)
	is.NoErr(err)
	// Should only contain T, S, T (3 tiles), not the through-tile E
	is.Equal(len(inferredRack), 3)

	// Scenario 3: Multiple through-tiles
	// Play "TESTS" with two through-tiles: "T0ST0"
	playedTiles = []tilemapping.MachineLetter{
		tilemapping.MachineLetter('T' - 'A' + 1),
		0, // Through-tile marker
		tilemapping.MachineLetter('S' - 'A' + 1),
		tilemapping.MachineLetter('T' - 'A' + 1),
		0, // Through-tile marker
	}
	inferredRack, err = InferRackForPlay(gdoc, 10, 4, ipc.GameEvent_HORIZONTAL, playedTiles)
	is.NoErr(err)
	// Should only contain T, S, T (3 tiles from rack, 2 through-tiles)
	is.Equal(len(inferredRack), 3)
}
