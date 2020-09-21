package entity

import (
	"os"
	"testing"
	"time"

	"github.com/domino14/macondo/board"
	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
)

var DefaultConfig = macondoconfig.Config{
	StrategyParamsPath:        os.Getenv("STRATEGY_PARAMS_PATH"),
	LexiconPath:               os.Getenv("LEXICON_PATH"),
	LetterDistributionPath:    os.Getenv("LETTER_DISTRIBUTION_PATH"),
	DefaultLexicon:            "NWL18",
	DefaultLetterDistribution: "English",
}

func newMacondoGame() *game.Game {
	rules, err := game.NewBasicGameRules(&DefaultConfig, board.CrosswordGameBoard,
		DefaultConfig.DefaultLetterDistribution)
	if err != nil {
		panic(err)
	}
	players := []*macondopb.PlayerInfo{
		{Nickname: "p1", RealName: "Player 1"},
		{Nickname: "p2", RealName: "Player 2"},
	}

	mcg, err := game.NewGame(rules, players)
	if err != nil {
		panic(err)
	}
	return mcg
}

func TestTimeCalc(t *testing.T) {
	is := is.New(t)
	mcg := newMacondoGame()
	g := NewGame(mcg, &pb.GameRequest{InitialTimeSeconds: 60, IncrementSeconds: 0})
	g.SetTimerModule(NewFakeNower(1234))
	g.ResetTimersAndStart()

	now := g.nower.Now()
	g.calculateAndSetTimeRemaining(0, now, false)
	g.calculateAndSetTimeRemaining(1, now, false)

	is.Equal(g.TimeRemaining(0), g.TimeRemaining(1))
	is.Equal(g.TimeRemaining(1), 60000)
}

func TestTimeCalcWithSleep(t *testing.T) {
	is := is.New(t)

	mcg := newMacondoGame()
	g := NewGame(mcg, &pb.GameRequest{InitialTimeSeconds: 60, IncrementSeconds: 0})
	nower := NewFakeNower(1234)
	g.SetTimerModule(nower)

	g.ResetTimersAndStart()
	g.SetPlayerOnTurn(1)
	// "sleep" 3520 ms
	nower.Sleep(3520)
	now := nower.Now()
	g.calculateAndSetTimeRemaining(0, now, false)
	g.calculateAndSetTimeRemaining(1, now, false)
	is.Equal(g.TimeRemaining(0), 60000)
	is.Equal(g.TimeRemaining(1), 60000-3520)
}

func TestTimeCalcWithMultipleSleep(t *testing.T) {
	is := is.New(t)

	mcg := newMacondoGame()
	g := NewGame(mcg, &pb.GameRequest{InitialTimeSeconds: 10, IncrementSeconds: 0})
	nower := NewFakeNower(1234)
	g.SetTimerModule(nower)

	g.ResetTimersAndStart()
	// Simulate a few moves:
	g.SetPlayerOnTurn(1)
	nower.Sleep(1520)
	g.RecordTimeOfMove(1)

	g.SetPlayerOnTurn(0)
	nower.Sleep(2233)
	g.RecordTimeOfMove(0)

	g.SetPlayerOnTurn(1)
	nower.Sleep(1122)
	g.RecordTimeOfMove(1)

	g.SetPlayerOnTurn(0)
	nower.Sleep(755)
	time.Sleep(755 * time.Millisecond)
	now := nower.Now()

	g.calculateAndSetTimeRemaining(0, now, false)

	is.Equal(g.TimeRemaining(0), 10000-2233-755)
	is.Equal(g.TimeRemaining(1), 10000-1520-1122)
}

func TestTimeCalcWithMultipleSleepIncrement(t *testing.T) {
	is := is.New(t)

	mcg := newMacondoGame()
	g := NewGame(mcg, &pb.GameRequest{InitialTimeSeconds: 10, IncrementSeconds: 5})
	nower := NewFakeNower(1234)
	g.SetTimerModule(nower)

	g.ResetTimersAndStart()
	// Simulate a few moves:
	g.SetPlayerOnTurn(1)
	nower.Sleep(1520)
	g.RecordTimeOfMove(1)

	g.SetPlayerOnTurn(0)
	nower.Sleep(2233)
	g.RecordTimeOfMove(0)

	g.SetPlayerOnTurn(1)
	nower.Sleep(1122)
	g.RecordTimeOfMove(1)

	g.SetPlayerOnTurn(0)
	nower.Sleep(755)
	now := nower.Now()

	g.calculateAndSetTimeRemaining(0, now, false)

	is.Equal(g.TimeRemaining(0), 10000-2233-755+5000)
	is.Equal(g.TimeRemaining(1), 10000-1520-1122+5000+5000)
}
