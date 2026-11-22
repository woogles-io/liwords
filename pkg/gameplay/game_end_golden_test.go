package gameplay_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/matryer/is"
	"google.golang.org/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// goldenGameEnd is what rating three games and adding their stats leaves:
// each game's ratings and stats, and the players' saved ratings and profile
// stats afterward.
type goldenGameEnd struct {
	Games   []goldenGame
	Players map[string]goldenPlayer
}

type goldenGame struct {
	Game            string
	EndReason       string
	Ratings         map[string][2]int32
	OriginalRatings []float64
	NewRatings      []float64
	GameStats       *entity.Stats
}

type goldenPlayer struct {
	Ratings entity.Ratings
	Stats   entity.ProfileStats
}

// playGoldenGames ends three games between cesar4, who starts with a rating
// in the variant, and Mina, who has none: game1 (Mina wins), game2 (the
// players the other way round; cesar4 wins) and game1 again, resigned. Each
// ends as performEndgameDuties ends a game: the players loaded afresh, the
// game rated, then its stats added to theirs.
func playGoldenGames(t *testing.T, st *stores.Stores) goldenGameEnd {
	is := is.New(t)
	ctx := DefaultConfig.WithContext(context.Background())

	load := func(name string) (*macondopb.GameHistory, *pb.GameRequest) {
		histjson, err := os.ReadFile("./testdata/" + name + "/history.json")
		is.NoErr(err)
		hist := &macondopb.GameHistory{}
		is.NoErr(json.Unmarshal(histjson, hist))
		reqjson, err := os.ReadFile("./testdata/" + name + "/game_request.json")
		is.NoErr(err)
		req := &pb.GameRequest{}
		is.NoErr(json.Unmarshal(reqjson, req))
		return hist, req
	}

	_, req1 := load("game1")
	variant := variantKey(req1)
	is.NoErr(seedRatings(ctx, st, "xjCWug7EZtDxDHX5fRZTLo", "3xpEkpRAy3AizbVmDg3kdi", variant,
		&entity.SingleRating{Rating: 1700, RatingDeviation: 80, Volatility: 0.06, LastGameTimestamp: 1700000000},
		&entity.SingleRating{Rating: 1400, RatingDeviation: 120, Volatility: 0.06, LastGameTimestamp: 1700000000}))

	out := goldenGameEnd{Players: map[string]goldenPlayer{}}
	for i, step := range []struct {
		game   string
		reason pb.GameEndReason
		now    int64
	}{
		{"game1", pb.GameEndReason_STANDARD, 1700086400},
		{"game2", pb.GameEndReason_STANDARD, 1700090000},
		{"game1", pb.GameEndReason_RESIGNED, 1700500000},
	} {
		hist, req := load(step.game)
		nicks := []string{hist.Players[0].Nickname, hist.Players[1].Nickname}
		scores := map[string]int32{nicks[0]: hist.FinalScores[0], nicks[1]: hist.FinalScores[1]}
		winner, loser := nicks[hist.Winner], nicks[1-hist.Winner]

		users := make([]*entity.User, 2)
		for j, p := range hist.Players {
			u, err := st.UserStore.GetByUUID(ctx, p.UserId)
			is.NoErr(err)
			users[j] = u
		}
		g := &entity.Game{GameReq: &entity.GameRequest{GameRequest: req}, Quickdata: &entity.Quickdata{}}
		g.SetHistory(proto.Clone(hist).(*macondopb.GameHistory))
		g.GameEndReason = step.reason
		ratings, err := gameplay.Rate(ctx, scores, g, winner, users, st.UserStore, step.now)
		is.NoErr(err)

		evt := &pb.GameEndedEvent{
			Scores:     scores,
			NewRatings: map[string]int32{nicks[0]: ratings[nicks[0]][1], nicks[1]: ratings[nicks[1]][1]},
			EndReason:  step.reason,
			Winner:     winner,
			Loser:      loser,
		}
		gs, err := gameplay.ComputeGameStats(ctx, proto.Clone(hist).(*macondopb.GameHistory), gameReq, variant, evt, st)
		is.NoErr(err)
		out.Games = append(out.Games, goldenGame{
			Game:            step.game + "#" + string(rune('1'+i)),
			EndReason:       step.reason.String(),
			Ratings:         ratings,
			OriginalRatings: g.Quickdata.OriginalRatings,
			NewRatings:      g.Quickdata.NewRatings,
			GameStats:       gs,
		})
	}

	for _, name := range []string{"cesar4", "Mina"} {
		u, err := st.UserStore.Get(ctx, name)
		is.NoErr(err)
		ps, err := profileStats(ctx, st, u.UUID)
		is.NoErr(err)
		out.Players[name] = goldenPlayer{Ratings: u.Profile.Ratings, Stats: ps}
	}
	return out
}

// Rating games and adding their stats gives exactly what it did before
// profile stats left the user lookups (#2355) and game end began locking
// the players' profiles (#2373): testdata/game_end_golden.json was written
// by the code before both (commit 3d00f9ebd), with this same scenario and
// its own ways of seeding a rating and reading profile stats. Don't
// regenerate it from the current code: that would only check the code
// against itself.
func TestGameEndMatchesGolden(t *testing.T) {
	is := is.New(t)
	_, st, _ := recreateDB()
	defer st.Disconnect()

	got, err := json.MarshalIndent(playGoldenGames(t, st), "", "  ")
	is.NoErr(err)
	want, err := os.ReadFile("./testdata/game_end_golden.json")
	is.NoErr(err)
	gotLines := strings.Split(string(got), "\n")
	wantLines := strings.Split(strings.TrimSuffix(string(want), "\n"), "\n")
	for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
		var g, w string
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if g != w {
			t.Fatalf("differs from testdata/game_end_golden.json at line %d:\n got: %s\nwant: %s", i+1, g, w)
		}
	}
}
