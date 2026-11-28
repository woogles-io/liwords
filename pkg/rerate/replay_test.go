package rerate

import (
	"path/filepath"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	alice, bob, carol = 1, 2, 3
	hasty, oldBot     = 10, 11
	nwl               = entity.VariantKey("NWL18.classic.regular")
	nwlCorres         = entity.VariantKey("NWL18.classic.corres")
)

func testData() *Data {
	return &Data{
		Keys: []entity.VariantKey{nwl, nwlCorres},
		Users: map[int64]User{
			alice:  {UUID: "a", Username: "alice"},
			bob:    {UUID: "b", Username: "bob"},
			carol:  {UUID: "c", Username: "carol"},
			hasty:  {UUID: "h", Username: "HastyBot", Bot: true},
			oldBot: {UUID: "o", Username: "MacondoBot1"},
		},
		Skipped: map[string]int{},
	}
}

func game(id, end int64, key uint16, p0, p1 int64, reason pb.GameEndReason, winner int8, s0, s1 int32) Game {
	return Game{ID: id, End: end, Key: key, Players: [2]int64{p0, p1}, Reason: reason,
		Winner: winner, Scores: [2]int32{s0, s1}}
}

// A replay rates games in end-time order, not id order, with the same code
// that rates games when they end.
func TestReplayMatchesGameEnd(t *testing.T) {
	is := is.New(t)
	d := testData()
	d.Games = []Game{
		game(3, 3000, 0, alice, hasty, pb.GameEndReason_RESIGNED, 1, 50, 200),
		game(1, 2000, 0, alice, bob, pb.GameEndReason_STANDARD, 0, 420, 380),
		game(2, 1000, 0, bob, alice, pb.GameEndReason_STANDARD, 0, 400, 300), // ended first
		game(4, 4000, 1, alice, carol, pb.GameEndReason_STANDARD, 1, 300, 310),
	}
	res := Replay(gameplay.RatingSystemV2, d, ReplayOptions{})
	is.Equal(res.Rated, 4)

	// The same games by hand, in end order.
	sys := gameplay.RatingSystemV2
	state := map[int64]map[entity.VariantKey]entity.SingleRating{alice: {}, bob: {}, carol: {}, hasty: {}}
	rate := func(p0, p1 int64, key entity.VariantKey, spread int, now int64) {
		users := []*entity.User{
			{Username: d.Users[p0].Username, IsBot: d.Users[p0].Bot, Profile: &entity.Profile{Ratings: entity.Ratings{Data: state[p0]}}},
			{Username: d.Users[p1].Username, IsBot: d.Users[p1].Bot, Profile: &entity.Profile{Ratings: entity.Ratings{Data: state[p1]}}},
		}
		var cur [2]*entity.SingleRating
		for i, p := range []int64{p0, p1} {
			if r, ok := state[p][key]; ok {
				cur[i] = &r
			}
		}
		_, after, err := sys.RateGame(cur, users, key, spread, now)
		is.NoErr(err)
		for i, p := range []int64{p0, p1} {
			if after[i] != nil {
				state[p][key] = *after[i]
			}
		}
	}
	rate(bob, alice, nwl, 100, 1000)
	rate(alice, bob, nwl, 40, 2000)
	rate(alice, hasty, nwl, -125, 3000)
	rate(alice, carol, nwlCorres, -10, 4000) // seeded from alice's NWL regular

	is.Equal(res.Ratings[alice], state[alice])
	is.Equal(res.Ratings[bob], state[bob])
	is.Equal(res.Ratings[carol], state[carol])
	is.Equal(len(res.Ratings[hasty]), 0) // a fixed bot gets no rating
	is.Equal(res.Games[alice][nwl], 3)
	is.Equal(res.Last, int64(4000))
}

func TestReplaySkips(t *testing.T) {
	is := is.New(t)
	d := testData()
	d.Games = []Game{
		game(1, 1000, 0, hasty, hasty, pb.GameEndReason_STANDARD, 0, 400, 300),
		// Flagged a bot only in the game's record, and not on the ladder.
		{ID: 2, End: 2000, Players: [2]int64{alice, oldBot}, QuickdataBot: [2]bool{false, true},
			Reason: pb.GameEndReason_STANDARD, Winner: 1, Scores: [2]int32{300, 400}},
		game(3, 3000, 0, alice, bob, pb.GameEndReason_STANDARD, 1, 400, 300), // winner disagrees
	}
	res := Replay(gameplay.RatingSystemV2, d, ReplayOptions{})
	is.Equal(res.Rated, 0)
	is.Equal(res.Skipped, map[string]int{"bot vs bot": 1, "bot without a fixed rating": 1,
		"winner does not match result": 1})
	is.Equal(len(res.Ratings), 0)
}

// A moderator reset clears the player's ratings from then on.
func TestReplayResets(t *testing.T) {
	is := is.New(t)
	d := testData()
	d.Games = []Game{
		game(1, 1000, 0, alice, bob, pb.GameEndReason_STANDARD, 0, 450, 300),
		game(2, 3000, 0, alice, bob, pb.GameEndReason_STANDARD, 0, 450, 300),
	}
	d.Resets = []Reset{{UserID: alice, Time: 2000}}
	var before []entity.SingleRating
	res := Replay(gameplay.RatingSystemV2, d, ReplayOptions{OnRated: func(g *Game, b, a [2]*entity.SingleRating) {
		before = append(before, *b[0])
	}})
	is.Equal(res.Rated, 2)
	// Alice starts both games new; bob's second game starts from his first.
	is.Equal(before[0].Rating, 1200.0)
	is.Equal(before[1].Rating, 1200.0)
	is.Equal(before[1].RatingDeviation, 600.0)
	is.Equal(res.Games[bob][nwl], 2)
}

// Since starts everyone from scratch at that time.
func TestReplaySince(t *testing.T) {
	is := is.New(t)
	d := testData()
	d.Games = []Game{
		game(1, 1000, 0, alice, bob, pb.GameEndReason_STANDARD, 0, 450, 300),
		game(2, 3000, 0, alice, bob, pb.GameEndReason_STANDARD, 0, 450, 300),
	}
	all := Replay(gameplay.RatingSystemV2, d, ReplayOptions{})
	late := Replay(gameplay.RatingSystemV2, d, ReplayOptions{Since: 2000})
	is.Equal(late.Rated, 1)
	is.True(late.Ratings[alice][nwl] != all.Ratings[alice][nwl])
	is.Equal(late.Games[alice][nwl], 1)
}

func TestSaveLoad(t *testing.T) {
	is := is.New(t)
	d := testData()
	d.Games = []Game{game(1, 1000, 0, alice, bob, pb.GameEndReason_STANDARD, 0, 450, 300)}
	d.Unfinished = []int64{7, 9}
	d.Current = map[int64]map[entity.VariantKey]entity.SingleRating{alice: {nwl: {Rating: 1500, RatingDeviation: 80}}}
	path := filepath.Join(t.TempDir(), "d.gob.gz")
	is.NoErr(d.Save(path))
	got, err := Load(path)
	is.NoErr(err)
	is.Equal(got, d)
}
