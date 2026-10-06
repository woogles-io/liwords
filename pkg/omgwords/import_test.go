package omgwords

import (
	"context"
	"os"
	"testing"

	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/macondo/gcgio"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/cwgame"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// A GCG whose last line is a time penalty imports as a finished game with the
// penalty applied after the end-of-game rack points.
func TestReplayImportedHistoryTimePenalty(t *testing.T) {
	is := is.New(t)
	cfg := config.DefaultConfig()
	ctx := log.Logger.WithContext(context.Background())
	ctx = cfg.WithContext(ctx)

	f, err := os.Open("./testdata/vs_josh_time_penalty.gcg")
	is.NoErr(err)
	defer f.Close()
	mcfg := macondoconfig.DefaultConfig()
	mcfg.SetDefault(macondoconfig.ConfigDefaultLexicon, "CSW21")
	mcfg.SetDefault(macondoconfig.ConfigDefaultLetterDistribution, "english")
	gh, err := gcgio.ParseGCGFromReader(mcfg, f)
	is.NoErr(err)

	dist, err := tilemapping.GetDistribution(cfg.WGLConfig(), "english")
	is.NoErr(err)
	gdoc, err := cwgame.NewGame(cfg.WGLConfig(), cwgame.NewBasicGameRules(
		"CSW21", "CrosswordGame", "english", ipc.ChallengeRule_ChallengeRule_FIVE_POINT,
		"", []int{0, 0}, 0, 0, true),
		[]*ipc.GameDocument_MinimalPlayerInfo{
			{Nickname: "cesar", UserId: "internal-cesar"},
			{Nickname: "josh", UserId: "internal-josh"},
		})
	is.NoErr(err)
	gdoc.Type = ipc.GameType_ANNOTATED

	err = replayImportedHistory(ctx, cfg.WGLConfig(), gdoc, gh, dist)
	is.NoErr(err)
	is.Equal(gdoc.PlayState, ipc.PlayState_GAME_OVER)
	is.Equal(gdoc.CurrentScores, []int32{387, 397})
	is.Equal(gdoc.Winner, int32(1))
	n := len(gdoc.Events)
	is.Equal(gdoc.Events[n-2].Type, ipc.GameEvent_END_RACK_PTS)
	is.Equal(gdoc.Events[n-1].Type, ipc.GameEvent_TIME_PENALTY)
	is.Equal(gdoc.Events[n-1].LostScore, int32(70))
}
