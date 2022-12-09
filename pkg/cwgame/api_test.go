package cwgame

import (
	"context"
	"os"
	"testing"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/matryer/is"
)

var DataDir = os.Getenv("DATA_PATH")
var DefaultConfig = &config.Config{DataPath: DataDir}

func restoreGlobalNower() {
	globalNower = GameTimer{}
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
	err := StartGame(context.Background(), g)
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

func TestProcessGameplayEvent(t *testing.T) {

}
