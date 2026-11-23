package gameplay

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/domino14/macondo/board"
	macondoconfig "github.com/domino14/macondo/config"
	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// wordCheckGame is a CSW21 game between u1 and u2 under rule, in variant.
func wordCheckGame(t *testing.T, rule macondopb.ChallengeRule, variant macondogame.Variant) *entity.Game {
	t.Helper()
	return wordCheckGameIn(t, rule, variant, "CSW21", "english")
}

// wordCheckGameIn is a game between u1 and u2 under rule, in variant, with
// the lexicon and the letter distribution named.
func wordCheckGameIn(t *testing.T, rule macondopb.ChallengeRule, variant macondogame.Variant, lexicon, dist string) *entity.Game {
	t.Helper()
	cfg := macondoconfig.DefaultConfig()
	if p := os.Getenv("MACONDO_DATA_PATH"); p != "" {
		cfg.Set(macondoconfig.ConfigDataPath, p)
	}
	rules, err := macondogame.NewBasicGameRules(cfg, lexicon, board.CrosswordGameLayout,
		dist, macondogame.CrossScoreOnly, variant)
	if err != nil {
		t.Fatal(err)
	}
	mcg, err := macondogame.NewGame(rules, []*macondopb.PlayerInfo{
		{Nickname: "one", UserId: "u1"},
		{Nickname: "two", UserId: "u2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	mcg.StartGame()
	return entity.NewGame(mcg, &ipc.GameRequest{ChallengeRule: rule})
}

func deniedFor(err error) bool {
	var ce *connect.Error
	return errors.As(err, &ce) && ce.Code() == connect.CodePermissionDenied
}

func TestWordCheckAllowed(t *testing.T) {
	is := is.New(t)
	g := wordCheckGame(t, macondopb.ChallengeRule_VOID, macondogame.VarClassic)
	is.NoErr(wordCheckAllowed(g, "u1"))
	is.NoErr(wordCheckAllowed(g, "u2"))
	// not a player of it
	is.True(deniedFor(wordCheckAllowed(g, "u3")))
	// a game that can be challenged
	five := wordCheckGame(t, macondopb.ChallengeRule_FIVE_POINT, macondogame.VarClassic)
	is.True(deniedFor(wordCheckAllowed(five, "u1")))
	// over
	g.SetGameEndReason(ipc.GameEndReason_RESIGNED)
	is.True(deniedFor(wordCheckAllowed(g, "u1")))
}

func TestCheckWord(t *testing.T) {
	is := is.New(t)
	g := wordCheckGame(t, macondopb.ChallengeRule_VOID, macondogame.VarClassic)
	word, valid, err := checkWord(g, " qi ")
	is.NoErr(err)
	is.Equal(word, "QI")
	is.True(valid)
	_, valid, err = checkWord(g, "QX")
	is.NoErr(err)
	is.True(!valid)
	// not in the game's letters, empty, or longer than the board; and a
	// blank, a gap, or two words is no word
	for _, w := range []string{"Q1", "", strings.Repeat("A", 16), "Q?", "?", "QI.", "QI ZA", "Q I"} {
		_, _, err = checkWord(g, w)
		var ce *connect.Error
		is.True(errors.As(err, &ce) && ce.Code() == connect.CodeInvalidArgument)
	}
	// WordSmog takes any anagram of a word, as its plays do
	smog := wordCheckGame(t, macondopb.ChallengeRule_VOID, macondogame.VarWordSmog)
	_, valid, err = checkWord(smog, "IQ")
	is.NoErr(err)
	is.True(valid)
	_, valid, err = checkWord(g, "IQ")
	is.NoErr(err)
	is.True(!valid)
}

func TestCheckWordReadsTiles(t *testing.T) {
	is := is.New(t)
	valid := func(g *entity.Game, w string) bool {
		t.Helper()
		_, ok, err := checkWord(g, w)
		is.NoErr(err)
		return ok
	}
	// Spanish: the CH tile is written [CH], as the game writes it; CH is a C
	// and an H, another spelling
	spanish := wordCheckGameIn(t, macondopb.ChallengeRule_VOID, macondogame.VarClassic, "FILE2017", "spanish")
	is.True(valid(spanish, "[CH]ICO"))
	is.True(valid(spanish, "[ch]ico"))
	is.True(!valid(spanish, "chico"))
	// read back by each tile's own label
	word, _, err := checkWord(spanish, "[ch]ico")
	is.NoErr(err)
	is.Equal(word, "[CH]ICO")
	word, _, err = checkWord(spanish, "chico")
	is.NoErr(err)
	is.Equal(word, "CHICO")
	// a label is as it is written, or as a blank writes it: never in other
	// case
	_, _, err = checkWord(spanish, "[Ch]ICO")
	var ce *connect.Error
	is.True(errors.As(err, &ce) && ce.Code() == connect.CodeInvalidArgument)
	// Catalan: L·L, NY and QU are tiles as written; LL is two L tiles; a
	// period is a gap, not the dot
	catalan := wordCheckGameIn(t, macondopb.ChallengeRule_VOID, macondogame.VarClassic, "DISC2", "catalan")
	for _, w := range []string{"COL·LEGI", "anys", "QUATRE", "LLIBRE"} {
		is.True(valid(catalan, w))
	}
	word, _, err = checkWord(catalan, "col·legi")
	is.NoErr(err)
	is.Equal(word, "COL·LEGI")
	_, _, err = checkWord(catalan, "col.legi")
	is.True(errors.As(err, &ce) && ce.Code() == connect.CodeInvalidArgument)
}

func TestWordCheckLimiter(t *testing.T) {
	is := is.New(t)
	l := &checkLimiter{}
	now := time.Now()
	for range wordChecksPerWindow {
		is.True(l.allow("u1", now))
	}
	is.True(!l.allow("u1", now.Add(time.Second)))
	// each user is counted on their own
	is.True(l.allow("u2", now))
	// a window on, again
	is.True(l.allow("u1", now.Add(wordCheckWindow)))
}

// Once a window, users with no check in it are let go, and the others keep
// exactly their checks still in it.
func TestWordCheckLimiterSweep(t *testing.T) {
	is := is.New(t)
	l := &checkLimiter{}
	now := time.Now()
	half := wordCheckWindow / 2
	is.True(l.allow("idle", now))
	for range wordChecksPerWindow / 2 {
		is.True(l.allow("busy", now))
		is.True(l.allow("busy", now.Add(half)))
	}
	later := now.Add(wordCheckWindow + time.Second)
	// the half of busy's checks made half a window in still count
	for range wordChecksPerWindow / 2 {
		is.True(l.allow("busy", later))
	}
	is.True(!l.allow("busy", later))
	_, kept := l.recent["idle"]
	is.True(!kept)
}
