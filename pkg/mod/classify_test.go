package mod

// What a finished game says about each player's conduct.
//
// Classify reads a game and nothing else, so these cases build the end state
// directly instead of playing a game to reach it. That is not only faster: the
// old approach played games through the whole store and then forced the parts
// it could not reach -- most notably SetPlayerOnTurn, so that a chosen player
// could be timed out even when it was not their turn -- and asserted on
// positions no real game arrives at. Nothing here can drift out of a legal
// state, because nothing here pretends to be a game in progress.

import (
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

const (
	p0ID = "player-zero"
	p1ID = "player-one"
	// Comfortably over UnreasonableTime (5 minutes), which is the threshold
	// below which a game is too short to judge anyone by.
	longEnough = int32(25 * 60)
)

// play is a move by a player, with the time they had left when they made it.
type play struct {
	player uint32
	kind   macondopb.GameEvent_Type
	msLeft int32
}

// finishedGame builds a game in its end state.
func finishedGame(reason ipc.GameEndReason, loserIdx int, plays []play, opts ...func(*entity.Game)) *entity.Game {
	g := &entity.Game{}
	events := make([]*macondopb.GameEvent, len(plays))
	for i, p := range plays {
		events[i] = &macondopb.GameEvent{
			PlayerIndex:     p.player,
			Type:            p.kind,
			MillisRemaining: p.msLeft,
		}
	}
	g.SetHistory(&macondopb.GameHistory{
		Uid: "test-game",
		Players: []*macondopb.PlayerInfo{
			{UserId: p0ID, Nickname: "zero"},
			{UserId: p1ID, Nickname: "one"},
		},
		Events: events,
	})
	g.GameEndReason = reason
	g.LoserIdx = loserIdx
	g.WinnerIdx = 1 - loserIdx
	g.GameReq = &entity.GameRequest{GameRequest: &ipc.GameRequest{
		InitialTimeSeconds: longEnough,
	}}
	g.Timers = entity.Timers{TimeRemaining: []int{0, 0}}
	g.MetaEvents = &entity.MetaEventData{}
	for _, o := range opts {
		o(g)
	}
	return g
}

func deniedNudge(userID string) func(*entity.Game) {
	return func(g *entity.Game) {
		g.MetaEvents.Events = append(g.MetaEvents.Events, &ipc.GameMetaEvent{
			PlayerId: userID,
			Type:     ipc.GameMetaEvent_ABORT_DENIED,
		})
	}
}

func shortGame(g *entity.Game) {
	// Below UnreasonableTime: too short to hold anyone's clock against them.
	g.GameReq.InitialTimeSeconds = 60
}

func timeLeftAtResignation(p0, p1 int) func(*entity.Game) {
	return func(g *entity.Game) { g.Timers.TimeRemaining = []int{p0, p1} }
}

func TestClassify(t *testing.T) {
	const (
		plenty = int32(20 * 60 * 1000) // 20 minutes left: they walked away
		barely = int32(3 * 1000)       // 3 seconds left: they played it out
	)
	tile := macondopb.GameEvent_TILE_PLACEMENT_MOVE

	for _, tc := range []struct {
		name       string
		game       *entity.Game
		isBotGame  bool
		wantLoser  ms.NotoriousGameType
		wantWinner ms.NotoriousGameType
	}{
		{
			name: "lost on time having never played",
			game: finishedGame(ipc.GameEndReason_TIME, 1, []play{
				{player: 0, kind: tile, msLeft: barely},
			}),
			wantLoser: ms.NotoriousGameType_NO_PLAY,
		},
		{
			name: "never played and refused to be let off",
			game: finishedGame(ipc.GameEndReason_TIME, 1, []play{
				{player: 0, kind: tile, msLeft: barely},
			}, deniedNudge(p1ID)),
			wantLoser: ms.NotoriousGameType_NO_PLAY_DENIED_NUDGE,
		},
		{
			name: "walked away with most of the clock left",
			game: finishedGame(ipc.GameEndReason_TIME, 1, []play{
				{player: 0, kind: tile, msLeft: barely},
				{player: 1, kind: tile, msLeft: plenty},
			}),
			wantLoser: ms.NotoriousGameType_SITTING,
		},
		{
			name: "played it out and simply ran out",
			game: finishedGame(ipc.GameEndReason_TIME, 1, []play{
				{player: 0, kind: tile, msLeft: barely},
				{player: 1, kind: tile, msLeft: barely},
			}),
			wantLoser: ms.NotoriousGameType_GOOD,
		},
		{
			name: "a bot game is nobody's fault",
			game: finishedGame(ipc.GameEndReason_TIME, 1, []play{
				{player: 0, kind: tile, msLeft: barely},
			}),
			isBotGame: true,
			wantLoser: ms.NotoriousGameType_GOOD,
		},
		{
			name: "too short a game to judge",
			game: finishedGame(ipc.GameEndReason_TIME, 1, []play{
				{player: 0, kind: tile, msLeft: barely},
			}, shortGame),
			wantLoser: ms.NotoriousGameType_GOOD,
		},
		{
			name: "resigned after barely playing",
			game: finishedGame(ipc.GameEndReason_RESIGNED, 1, []play{
				{player: 0, kind: tile, msLeft: barely},
				{player: 1, kind: tile, msLeft: barely},
			}, timeLeftAtResignation(0, int(barely))),
			wantLoser: ms.NotoriousGameType_SANDBAG,
		},
		{
			name: "resigned long after their last move",
			game: finishedGame(ipc.GameEndReason_RESIGNED, 1, []play{
				{player: 0, kind: tile, msLeft: barely},
				{player: 1, kind: tile, msLeft: plenty},
				{player: 1, kind: tile, msLeft: plenty},
				{player: 1, kind: tile, msLeft: plenty},
			}, timeLeftAtResignation(0, 0)),
			wantLoser: ms.NotoriousGameType_SITTING,
		},
		{
			name: "a game that simply ended",
			game: finishedGame(ipc.GameEndReason_STANDARD, 1, []play{
				{player: 0, kind: tile, msLeft: barely},
				{player: 1, kind: tile, msLeft: barely},
			}),
			wantLoser: ms.NotoriousGameType_GOOD,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			v, err := Classify(tc.game, nil, tc.isBotGame)
			is.NoErr(err)
			is.Equal(v.Loser, tc.wantLoser)
			// GOOD is the zero value, so a case that says nothing about the
			// winner is asserting that the winner did nothing wrong.
			is.Equal(v.Winner, tc.wantWinner)
		})
	}
}

// A drawn game has LoserIdx -1, and nothing downstream may index an array with
// it. The original code squared it; this keeps that behaviour and says so.
func TestClassifyHandlesADraw(t *testing.T) {
	is := is.New(t)
	g := finishedGame(ipc.GameEndReason_STANDARD, -1, []play{
		{player: 0, kind: macondopb.GameEvent_TILE_PLACEMENT_MOVE, msLeft: 1000},
	})
	v, err := Classify(g, nil, false)
	is.NoErr(err)
	is.Equal(v.LoserIdx, 1)
	is.Equal(v.Loser, ms.NotoriousGameType_GOOD)
	is.Equal(v.Winner, ms.NotoriousGameType_GOOD)
}

// The phony check is the one part of classification that needs a word list, so
// it gets a real one.
func TestClassifyExcessivePhonies(t *testing.T) {
	is := is.New(t)
	// A real word list, which is the only thing classification needs beyond
	// the game itself.
	cfg := DefaultConfig

	// Three tile plays, all of them words that are not in NWL20. The threshold
	// is three phonies and more than half of a player's tile plays.
	phonies := []string{"ZZZQJX", "QQXJZV", "JXZQVW"}
	events := make([]*macondopb.GameEvent, 0, len(phonies))
	for _, w := range phonies {
		events = append(events, &macondopb.GameEvent{
			PlayerIndex: 1,
			Type:        macondopb.GameEvent_TILE_PLACEMENT_MOVE,
			WordsFormed: []string{w},
			// Little time left, so this is not also classified as SITTING.
			MillisRemaining: 1000,
		})
	}

	g := finishedGame(ipc.GameEndReason_TIME, 1, nil)
	hist := g.History()
	hist.Lexicon = "NWL20"
	hist.Events = events

	v, err := Classify(g, cfg.WGLConfig(), false)
	is.NoErr(err)
	is.Equal(v.Loser, ms.NotoriousGameType_EXCESSIVE_PHONIES)
	is.Equal(v.Winner, ms.NotoriousGameType_GOOD)
}
