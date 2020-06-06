package entity

import (
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	pb "github.com/domino14/crosswords/rpc/api/proto"
	"github.com/domino14/macondo/board"
	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"
)

const (
	// epsilon is in milliseconds.
	epsilon = 12
)

var DefaultConfig = macondoconfig.Config{
	StrategyParamsPath:        os.Getenv("STRATEGY_PARAMS_PATH"),
	LexiconPath:               os.Getenv("LEXICON_PATH"),
	DefaultLexicon:            "NWL18",
	DefaultLetterDistribution: "English",
}

func withinEpsilon(a, b int) bool {
	return math.Abs(float64(a-b)) < float64(epsilon)
}

func newMacondoGame() *game.Game {
	rules, _ := game.NewGameRules(&DefaultConfig, board.CrosswordGameBoard,
		DefaultConfig.DefaultLexicon, DefaultConfig.DefaultLetterDistribution)
	players := []*macondopb.PlayerInfo{
		{Nickname: "p1", RealName: "Player 1"},
		{Nickname: "p2", RealName: "Player 2"},
	}

	mcg, _ := game.NewGame(rules, players)
	return mcg
}

func TestTimeCalc(t *testing.T) {
	is := is.New(t)

	mcg := newMacondoGame()
	g := NewGame(mcg, &pb.GameRequest{InitialTimeSeconds: 60, IncrementSeconds: 0})
	g.ResetTimers()
	g.calculateTimeRemaining(0)
	g.calculateTimeRemaining(1)

	is.True(withinEpsilon(g.TimeRemaining(0), g.TimeRemaining(1)))
	is.True(withinEpsilon(g.TimeRemaining(1), 60000))
}

func TestTimeCalcWithSleep(t *testing.T) {
	is := is.New(t)

	mcg := newMacondoGame()
	g := NewGame(mcg, &pb.GameRequest{InitialTimeSeconds: 60, IncrementSeconds: 0})
	g.ResetTimers()
	g.SetPlayerOnTurn(1)
	time.Sleep(3520 * time.Millisecond)

	g.calculateTimeRemaining(0)
	g.calculateTimeRemaining(1)
	is.True(withinEpsilon(g.TimeRemaining(0), 60000))
	is.True(withinEpsilon(g.TimeRemaining(1), 60000-3520))
}

func TestTimeCalcWithMultipleSleep(t *testing.T) {
	is := is.New(t)

	mcg := newMacondoGame()
	g := NewGame(mcg, &pb.GameRequest{InitialTimeSeconds: 60, IncrementSeconds: 0})
	g.ResetTimers()
	// Simulate a few moves:
	g.SetPlayerOnTurn(1)
	time.Sleep(1520 * time.Millisecond)
	g.RecordTimeOfMove(1)

	g.SetPlayerOnTurn(0)
	time.Sleep(2233 * time.Millisecond)
	g.RecordTimeOfMove(0)

	g.SetPlayerOnTurn(1)
	time.Sleep(1122 * time.Millisecond)
	g.RecordTimeOfMove(1)

	g.SetPlayerOnTurn(0)
	time.Sleep(755 * time.Millisecond)
	g.calculateTimeRemaining(0)

	fmt.Println(g.TimeRemaining(0))
	fmt.Println(g.TimeRemaining(1))

	is.True(withinEpsilon(g.TimeRemaining(0), 60000-2233-755))
	is.True(withinEpsilon(g.TimeRemaining(1), 60000-1520-1122))
}
