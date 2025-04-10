package game

import (
	"crypto/rand"
	"unsafe"

	wglconfig "github.com/domino14/word-golib/config"
	"github.com/domino14/word-golib/tilemapping"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/woogles-io/liwords/pkg/omgwords/game/board"
	"github.com/woogles-io/liwords/pkg/omgwords/game/gamestate"
	"github.com/woogles-io/liwords/pkg/omgwords/game/tiles"
)

// When playing a game, the most important info is the ongoing game state.
// This can be represented in a flatbuffer GameState object. It contains the
// game board, the tile bag, the player racks, and current scores. It also
// contains the game timers.
// If this was the only thing that was deserialized, we could still run the game logic.
//
// There is a lot of other information that is part of a game, like
// the game rules, the game history, cancel/abort metadata, game request info,
// and so forth. Not all of this needs to be loaded and transmitted every time
// anyone asks for a game.
//
// The database game columns should be refactored a bit for this:
// - Remove player IDs (this goes in the separate player to game table)
// - Remove game history (this goes in the separate game history table, as a
// list of events, and then transferred to S3 when the game is over).
// - Add a column to store the game state as a flatbuffer. This is meant to be
// temporary and small.
//   - Remove the JSONB timers column (it's part of the flatbuffer now).
//   - Keep the JSONB game request column (migrate from bytea).
//   - Add a JSONB column with combined game metadata:
//     Stats, QuickData, TournamentData, MetaEvents
//
// When a game is loaded on first load, all this info could be sent over to the client.
// When a user takes a turn, only the game state and game history need to be updated.
// type Game struct {
// 	DBID uint
// 	Type pb.GameType

// 	Started   bool
// 	GameState *gamestate.GameState
// 	nower     Nower
// 	Events    []*pb.GameEvent
// }

func createGameState(cfg *wglconfig.Config, layoutName string, numPlayers int, rules *GameRules) ([]byte, error) {

	builder := flatbuffers.NewBuilder(512)
	var seed [32]byte
	rand.Read(seed[:])

	dist, err := tilemapping.GetDistribution(cfg, rules.distname)
	if err != nil {
		return nil, err
	}

	tileBagOffset := tiles.BuildTileBag(builder, dist, seed)
	gamestate.GameStateStart(builder)
	gamestate.GameStateAddBag(builder, tileBagOffset)

	boardOffset, err := board.BuildBoard(builder, layoutName)
	if err != nil {
		return nil, err
	}
	gamestate.GameStateAddBoard(builder, boardOffset)
	gamestate.GameStateAddNumPlayers(builder, uint8(numPlayers))

	rackVector := builder.CreateByteVector(make([]byte, numPlayers*7))
	gamestate.GameStateAddRacks(builder, rackVector)

	gamestate.GameStateStartPlayerScoresVector(builder, numPlayers)
	for range numPlayers {
		builder.PrependInt32(0)
	}
	playerScores := builder.EndVector(numPlayers)
	gamestate.GameStateAddPlayerScores(builder, playerScores)

	gamestate.TimersStartTimeRemainingMsVector(builder, numPlayers)
	for _, v := range rules.secondsPerPlayer {
		builder.PrependInt64(int64(v * 1000))
	}
	tremainvector := builder.EndVector(numPlayers)

	gamestate.TimersStart(builder)
	gamestate.TimersAddTimeOfLastUpdateMs(builder, uint64(0))
	gamestate.TimersAddTimeStartedMs(builder, uint64(0))
	gamestate.TimersAddTimeRemainingMs(builder, tremainvector)
	gamestate.TimersAddMaxOvertimeMinutes(builder, int32(rules.maxOvertimeMins))
	gamestate.TimersAddIncrementSeconds(builder, int32(rules.incrementSeconds))
	gamestate.TimersAddResetToIncrementAfterTurn(builder, false)
	gamestate.TimersAddTimeBankSeconds(builder, uint64(0))
	timers := gamestate.TimersEnd(builder)

	gamestate.GameStateAddTimers(builder, timers)

	tt := gamestate.GameStateEnd(builder)
	builder.Finish(tt)

	return builder.FinishedBytes(), nil
}

func createGameStateFromBytes(bytes []byte) (*gamestate.GameState, error) {
	st := gamestate.GetRootAsGameState(bytes, 0)
	return st, nil
}

func racksFromGameState(st *gamestate.GameState) []tilemapping.MachineLetter {
	// Note that the rack must be of the correct size!
	racksBytes := st.RacksBytes()
	// Note that MachineLetter is just a byte, so we can do this ugly unsafe pointer magic and avoid allocating.
	racks := *(*[]tilemapping.MachineLetter)(unsafe.Pointer(&racksBytes))
	return racks
}
