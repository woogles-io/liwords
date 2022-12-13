package cwgame

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var DataDir = os.Getenv("DATA_PATH")
var DefaultConfig = &config.Config{DataPath: DataDir}

func restoreGlobalNower() {
	globalNower = GameTimer{}
}

func ctxForTests() context.Context {
	ctx := context.Background()
	ctx = log.Logger.WithContext(ctx)
	ctx = context.WithValue(ctx, config.CtxKeyword, DefaultConfig)
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

func TestNewGame(t *testing.T) {
	is := is.New(t)
	rules := NewBasicGameRules("NWL20", "CrosswordGame", "english", "classic",
		[]int{300, 300}, 1, 0)
	g, err := NewGame(DefaultConfig, rules, []*ipc.GameDocument_MinimalPlayerInfo{
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

	rules := NewBasicGameRules("NWL20", "CrosswordGame", "english", "classic",
		[]int{300, 300}, 1, 0)
	g, _ := NewGame(DefaultConfig, rules, []*ipc.GameDocument_MinimalPlayerInfo{
		{Nickname: "Cesitar", RealName: "Cesar", UserId: "cesar1"},
		{Nickname: "Lucas", RealName: "Lucas", UserId: "lucas1"},
	})
	err := StartGame(ctxForTests(), g)
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
				Tiles:          "KNI.E",
			},
			userID:         "2gJGaYnchL6LbQVTNQ6mjT",
			expectedScores: []int32{113, 137},
		},
		{
			name: "Exchange",
			cge: &ipc.ClientGameplayEvent{
				Type:   ipc.ClientGameplayEvent_EXCHANGE,
				GameId: "9aK3YgVk",
				Tiles:  "EEIKNTW",
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
				Tiles:          "TWEEN",
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
				Tiles:          "KNI..",
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
				Tiles:          "WEEK",
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
				Tiles:          "KNI.ES",
			},
			userID:      "2gJGaYnchL6LbQVTNQ6mjT",
			expectedErr: errors.New("tile in play but not in rack: 19"),
		},
		{
			name: "Try exchanging a tile we don't have",
			cge: &ipc.ClientGameplayEvent{
				Type:   ipc.ClientGameplayEvent_EXCHANGE,
				GameId: "9aK3YgVk",
				Tiles:  "Q",
			},
			userID:      "2gJGaYnchL6LbQVTNQ6mjT",
			expectedErr: errors.New("tile in play but not in rack: 17"),
		},
		{
			name: "Try playing as another player not on turn",
			cge: &ipc.ClientGameplayEvent{
				Type:   ipc.ClientGameplayEvent_EXCHANGE,
				GameId: "9aK3YgVk",
				Tiles:  "EEIKNTW",
			},
			userID:      "foo",
			expectedErr: errors.New("not on turn"),
		},
		{
			name: "Try another game id",
			cge: &ipc.ClientGameplayEvent{
				Type:   ipc.ClientGameplayEvent_EXCHANGE,
				GameId: "abc",
				Tiles:  "EEIKNTW",
			},
			userID:      "2gJGaYnchL6LbQVTNQ6mjT",
			expectedErr: errors.New("game ids do not match"),
		},
	}
	// load config into context; this is needed by the gameplay functions.
	ctx := ctxForTests()

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			gdoc := loadGDoc(documentfile)
			// use a timestamp that's a little bit later than the
			// time_of_last_update in the doc.
			globalNower = &FakeNower{
				fakeMeow: gdoc.Timers.TimeOfLastUpdate + 5000}
			defer restoreGlobalNower()
			onturn := gdoc.PlayerOnTurn

			err := ProcessGameplayEvent(ctx, tc.cge, tc.userID, gdoc)
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
		Tiles:          "KNI.EW",
	}

	err := ProcessGameplayEvent(ctx, cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(2500) // second player challenges after 2500 ms

	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9aK3YgVk",
	}
	err = ProcessGameplayEvent(ctx, cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)
	is.Equal(gdoc.ScorelessTurns, uint32(1))

	globalNower.(*FakeNower).Sleep(1500) // second player plays after challenging
	cge3 := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "15C",
		Tiles:          "S.HEROID",
	}
	err = ProcessGameplayEvent(ctx, cge3, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)
	is.Equal(len(gdoc.Events), 7)

	is.Equal(gdoc.Events[4], &ipc.GameEvent{
		Rack:            []byte{5, 5, 9, 11, 14, 20, 23}, // EEIKNTW
		Type:            ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:      125,
		Row:             0,
		Column:          3,
		Direction:       ipc.GameEvent_HORIZONTAL,
		Position:        "1D",
		PlayedTiles:     []byte{11, 14, 9, 0, 5, 23}, // KNI.EW
		Score:           63,
		WordsFormed:     [][]byte{{11, 14, 9, 22, 5, 23}}, // KNIVEW
		MillisRemaining: 883808,                           // 5000 ms after their last time remaining
		PlayerIndex:     0,
		Leave:           []byte{5, 20}, // ET
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
		Rack:            []byte{4, 5, 8, 9, 15, 18, 19}, // DEHIORS
		Type:            ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:      229,
		Row:             14,
		Column:          2,
		Direction:       ipc.GameEvent_HORIZONTAL,
		Position:        "15C",
		PlayedTiles:     []byte{19, 0, 8, 5, 18, 15, 9, 4}, // S.HEROID
		Score:           92,
		WordsFormed:     [][]byte{{19, 16, 8, 5, 18, 15, 9, 4}}, // SPHEROID
		MillisRemaining: 895914,                                 // 5000 ms after their last time remaining
		PlayerIndex:     1,
		Leave:           []byte{},
		IsBingo:         true,
	})
	is.Equal(gdoc.CurrentScores, []int32{62, 229})
	is.Equal(gdoc.ScorelessTurns, uint32(0))
	is.Equal(gdoc.Racks[0], []byte{5, 5, 9, 11, 14, 20, 23})
	is.Equal(gdoc.PlayerOnTurn, uint32(0))
	/*
		dist, err := tiles.GetDistribution(DefaultConfig, gdoc.LetterDistribution)
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
		Tiles:          "KNI.E",
	}

	err := ProcessGameplayEvent(ctx, cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(2500)
	// second player challenges after 2500 ms, but nothing happens since
	// this is single challenge

	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9aK3YgVk",
	}
	err = ProcessGameplayEvent(ctx, cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(1500)
	// second player can still play
	cge3 := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "15C",
		Tiles:          "S.HEROID",
	}
	err = ProcessGameplayEvent(ctx, cge3, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)
	is.Equal(len(gdoc.Events), 7)

	is.Equal(gdoc.Events[4], &ipc.GameEvent{
		Rack:            []byte{5, 5, 9, 11, 14, 20, 23}, // EEIKNTW
		Type:            ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:      113,
		Row:             0,
		Column:          3,
		Direction:       ipc.GameEvent_HORIZONTAL,
		Position:        "1D",
		PlayedTiles:     []byte{11, 14, 9, 0, 5}, // KNI.E
		Score:           51,
		WordsFormed:     [][]byte{{11, 14, 9, 22, 5}}, // KNIVE
		MillisRemaining: 883808,                       // 5000 ms after their last time remaining
		PlayerIndex:     0,
		Leave:           []byte{5, 20, 23}, // ETW
	})

	// Don't compare this event exactly because this event has a new random rack.
	is.Equal(gdoc.Events[5].Type, ipc.GameEvent_CHALLENGE_BONUS)
	is.Equal(gdoc.Events[5].Bonus, int32(0))
	is.Equal(gdoc.Events[5].Cumulative, int32(113))
	is.Equal(gdoc.Events[5].PlayerIndex, uint32(0))
	is.Equal(gdoc.Events[5].MillisRemaining, int32(897414))

	is.Equal(gdoc.Events[6], &ipc.GameEvent{
		Rack:            []byte{4, 5, 8, 9, 15, 18, 19}, // DEHIORS
		Type:            ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:      229,
		Row:             14,
		Column:          2,
		Direction:       ipc.GameEvent_HORIZONTAL,
		Position:        "15C",
		PlayedTiles:     []byte{19, 0, 8, 5, 18, 15, 9, 4}, // S.HEROID
		Score:           92,
		WordsFormed:     [][]byte{{19, 16, 8, 5, 18, 15, 9, 4}}, // SPHEROID
		MillisRemaining: 895914,                                 // 5000 ms after their last time remaining
		PlayerIndex:     1,
		Leave:           []byte{},
		IsBingo:         true,
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
		Tiles:          "KNI.E",
	}

	err := ProcessGameplayEvent(ctx, cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(2500)
	// second player challenges after 2500 ms, and they should lose
	// their turn
	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9aK3YgVk",
	}
	err = ProcessGameplayEvent(ctx, cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(1500)
	// first player plays again
	cge3 := &ipc.ClientGameplayEvent{
		Type:           ipc.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         "9aK3YgVk",
		PositionCoords: "C11",
		Tiles:          "TEW", // this is their leave from their last move.
	}
	err = ProcessGameplayEvent(ctx, cge3, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)
	is.Equal(len(gdoc.Events), 7)

	is.Equal(gdoc.Events[4], &ipc.GameEvent{
		Rack:            []byte{5, 5, 9, 11, 14, 20, 23}, // EEIKNTW
		Type:            ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:      113,
		Row:             0,
		Column:          3,
		Direction:       ipc.GameEvent_HORIZONTAL,
		Position:        "1D",
		PlayedTiles:     []byte{11, 14, 9, 0, 5}, // KNI.E
		Score:           51,
		WordsFormed:     [][]byte{{11, 14, 9, 22, 5}}, // KNIVE
		MillisRemaining: 883808,                       // 5000 ms after their last time remaining
		PlayerIndex:     0,
		Leave:           []byte{5, 20, 23}, // ETW
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
		Tiles:          "KNI.E",
	}

	err := ProcessGameplayEvent(ctx, cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(2500)
	// second player challenges after 2500 ms, and they should lose
	// their turn
	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9aK3YgVk",
	}
	err = ProcessGameplayEvent(ctx, cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)

	is.Equal(len(gdoc.Events), 6)

	is.Equal(gdoc.Events[4], &ipc.GameEvent{
		Rack:        []byte{5, 5, 9, 11, 14, 20, 23}, // EEIKNTW
		Type:        ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:  113,
		Row:         0,
		Column:      3,
		Direction:   ipc.GameEvent_HORIZONTAL,
		Position:    "1D",
		PlayedTiles: []byte{11, 14, 9, 0, 5}, // KNI.E
		Score:       51,
		WordsFormed: [][]byte{{11, 14, 9, 22, 5}}, // KNIVE
		// 5000 ms after their last time remaining
		MillisRemaining: 883808,
		PlayerIndex:     0,
		Leave:           []byte{5, 20, 23}, // ETW
	})
	is.Equal(gdoc.Events[5], &ipc.GameEvent{
		Type:       ipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS,
		Cumulative: 137,
		// 2.5 second sleep.
		MillisRemaining: 897414,
		PlayerIndex:     1,
		Rack:            []byte{4, 5, 8, 9, 15, 18, 19}, // DEHIORS
	})

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
		Tiles:          "KNI.EW",
	}

	err := ProcessGameplayEvent(ctx, cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(2500)
	// second player challenges after 2500 ms, and they should lose
	// their turn
	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9aK3YgVk",
	}
	err = ProcessGameplayEvent(ctx, cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)

	is.Equal(len(gdoc.Events), 6)

	is.Equal(gdoc.Events[4], &ipc.GameEvent{
		Rack:        []byte{5, 5, 9, 11, 14, 20, 23}, // EEIKNTW
		Type:        ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:  125,
		Row:         0,
		Column:      3,
		Direction:   ipc.GameEvent_HORIZONTAL,
		Position:    "1D",
		PlayedTiles: []byte{11, 14, 9, 0, 5, 23}, // KNI.EW
		Score:       63,
		WordsFormed: [][]byte{{11, 14, 9, 22, 5, 23}}, // KNIVEW
		// 5000 ms after their last time remaining
		MillisRemaining: 883808,
		PlayerIndex:     0,
		Leave:           []byte{5, 20}, // ET
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
		Tiles:          "KNI.EW",
	}

	err := ProcessGameplayEvent(ctx, cge, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.NoErr(err)

	globalNower.(*FakeNower).Sleep(899914 + 60001)
	// second player challenges way too late, they went overtime
	// by a minute and 1 millisecond (remember this doc has max permitted
	// overtime equal to 1 minute)
	cge2 := &ipc.ClientGameplayEvent{
		Type:   ipc.ClientGameplayEvent_CHALLENGE_PLAY,
		GameId: "9aK3YgVk",
	}
	err = ProcessGameplayEvent(ctx, cge2, "FDHvxexaC5QNMfiJnpcnUZ", gdoc)
	is.NoErr(err)

	is.Equal(len(gdoc.Events), 6)

	is.Equal(gdoc.Events[4], &ipc.GameEvent{
		Rack:            []byte{5, 5, 9, 11, 14, 20, 23}, // EEIKNTW
		Type:            ipc.GameEvent_TILE_PLACEMENT_MOVE,
		Cumulative:      125,
		Row:             0,
		Column:          3,
		Direction:       ipc.GameEvent_HORIZONTAL,
		Position:        "1D",
		PlayedTiles:     []byte{11, 14, 9, 0, 5, 23}, // KNI.EW
		Score:           63,
		WordsFormed:     [][]byte{{11, 14, 9, 22, 5, 23}}, // KNIVEW
		MillisRemaining: 883808,                           // 5000 ms after their last time remaining
		PlayerIndex:     0,
		Leave:           []byte{5, 20}, // ET
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
		Tiles:          "FOO", // doesn't matter
	}

	err = ProcessGameplayEvent(ctx, cge3, "2gJGaYnchL6LbQVTNQ6mjT", gdoc)
	is.Equal(err, errGameNotActive)
}

func TestChallengeBadWordEndOfGame(t *testing.T) {

}

func TestChallengeGoodWordEndOfGame(t *testing.T) {

}

// Load a document that has a challenge or pass state, it should work properly.
func TestLoadChallengeOrPassState(t *testing.T) {

}

func BenchmarkLoadDocumentJSON(b *testing.B) {
	is := is.New(b)
	documentfile := "document-earlygame.json"
	content, err := os.ReadFile("./testdata/" + documentfile)
	is.NoErr(err)
	// ~41.7 us per op on themonolith (12th-gen intel box)
	for i := 0; i < b.N; i++ {
		gdoc := &ipc.GameDocument{}
		err = protojson.Unmarshal(content, gdoc)
		is.NoErr(err)
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
		err = proto.Unmarshal(content, gdoc)
		is.NoErr(err)
	}
}
