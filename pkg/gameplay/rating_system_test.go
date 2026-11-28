package gameplay

import (
	"context"
	"errors"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/glicko"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func human(name string, ratings map[entity.VariantKey]entity.SingleRating) *entity.User {
	return &entity.User{Username: name, Profile: &entity.Profile{Ratings: entity.Ratings{Data: ratings}}}
}

func bot(name string) *entity.User {
	u := human(name, nil)
	u.IsBot = true
	return u
}

const nwlRegular = entity.VariantKey("NWL18.classic.regular")

// V1 rates both players, bots included, exactly as glicko.Rate does.
func TestRateGameV1(t *testing.T) {
	is := is.New(t)
	r0 := &entity.SingleRating{Rating: 1700, RatingDeviation: 80, Volatility: 0.06, LastGameTimestamp: 1000}
	users := []*entity.User{human("a", nil), bot("HastyBot")}
	before, after, err := RatingSystemV1.RateGame([2]*entity.SingleRating{r0, nil}, users, nwlRegular, 40, 5000)
	is.NoErr(err)

	// A player with no rating starts at 1500 / 350, last played now.
	is.Equal(*before[1], entity.SingleRating{Rating: 1500, RatingDeviation: 350, Volatility: 0.06, LastGameTimestamp: 5000})
	r, rd, v := glicko.Rate(1700, 80, 0.06, 1500, 350, 40, 4000)
	is.Equal(*after[0], entity.SingleRating{Rating: r, RatingDeviation: rd, Volatility: v, LastGameTimestamp: 5000})
	r, rd, v = glicko.Rate(1500, 350, 0.06, 1700, 80, -40, 0)
	is.Equal(*after[1], entity.SingleRating{Rating: r, RatingDeviation: rd, Volatility: v, LastGameTimestamp: 5000})
	// The saved rating isn't changed in place.
	is.Equal(r0.Rating, 1700.0)
}

// V2 rates the human against the bot's fixed rating with RD 0, and leaves the
// bot alone.
func TestRateGameV2Bot(t *testing.T) {
	is := is.New(t)
	r1 := &entity.SingleRating{Rating: 1650, RatingDeviation: 70, Volatility: 0.06, LastGameTimestamp: 1000}
	saved := &entity.SingleRating{Rating: 2400, RatingDeviation: 60, Volatility: 0.06}
	users := []*entity.User{bot("HastyBot"), human("a", nil)}
	before, after, err := RatingSystemV2.RateGame([2]*entity.SingleRating{saved, r1}, users, nwlRegular, 125, 5000)
	is.NoErr(err)
	is.Equal(before[0].Rating, 2050.0) // the fixed rating, not the saved one
	is.True(after[0] == nil)
	r, rd, v := glicko.V2.Rate(1650, 70, 0.06, 2050, 0, -125, 4000)
	is.Equal(*after[1], entity.SingleRating{Rating: r, RatingDeviation: rd, Volatility: v, LastGameTimestamp: 5000})
	is.True(r < 1650)
}

func TestRateGameV2BotErrors(t *testing.T) {
	is := is.New(t)
	_, _, err := RatingSystemV2.RateGame([2]*entity.SingleRating{}, []*entity.User{bot("HastyBot"), bot("BestBot")}, nwlRegular, 10, 5000)
	is.True(errors.Is(err, ErrBotsNotRated))
	_, _, err = RatingSystemV2.RateGame([2]*entity.SingleRating{}, []*entity.User{human("a", nil), bot("NewBot")}, nwlRegular, 10, 5000)
	is.True(err != nil)
}

func TestNewRatingV2(t *testing.T) {
	is := is.New(t)
	sr := func(r, rd float64) entity.SingleRating {
		return entity.SingleRating{Rating: r, RatingDeviation: rd, Volatility: 0.07, LastGameTimestamp: 1}
	}
	corres := entity.VariantKey("NWL18.classic.corres")
	for _, tc := range []struct {
		name   string
		others map[entity.VariantKey]entity.SingleRating
		r, rd  float64
	}{
		{"no other ratings", nil, 1200, 600},
		{"only uncertain ratings", map[entity.VariantKey]entity.SingleRating{
			"NWL18.classic.regular": sr(1800, 301)}, 1200, 600},
		{"other variants don't count", map[entity.VariantKey]entity.SingleRating{
			"NWL18.wordsmog.regular": sr(1800, 60), "NWL18.puzzle.corres": sr(1900, 60)}, 1200, 600},
		{"lowest RD, kept above the minimum", map[entity.VariantKey]entity.SingleRating{
			"NWL18.classic.regular": sr(1700, 290), "NWL18.classic.rapid": sr(1600, 270)}, 1600, 270},
		{"RD raised to the minimum", map[entity.VariantKey]entity.SingleRating{
			"NWL18.classic.regular": sr(1700, 60)}, 1700, 250},
		{"same lexicon family first", map[entity.VariantKey]entity.SingleRating{
			"CSW19.classic.regular": sr(2000, 60), "NWL18.classic.blitz": sr(1500, 200)}, 1500, 250},
		{"another lexicon if needed", map[entity.VariantKey]entity.SingleRating{
			"CSW19.classic.regular": sr(2000, 60)}, 2000, 250},
	} {
		got := RatingSystemV2.newRating(tc.others, corres, 99)
		is.Equal(*got, entity.SingleRating{Rating: tc.r, RatingDeviation: tc.rd, Volatility: glicko.InitialVolatility, LastGameTimestamp: 99}) // tc.name
	}
	// V1 never seeds.
	got := RatingSystemV1.newRating(map[entity.VariantKey]entity.SingleRating{"NWL18.classic.regular": sr(1700, 60)}, corres, 99)
	is.Equal(got.Rating, 1500.0)
}

// V1 shows ratings as entity.RelevantRating did before the rating systems
// were split: 1500? for no rating, "?" above RD 90, the bot's own rating.
func TestDisplayRatingV1(t *testing.T) {
	is := is.New(t)
	s := RatingSystemV1
	is.Equal(s.DisplayRating(&entity.User{Username: "anon"}, nwlRegular), "UnratedAnon")
	is.Equal(s.DisplayRating(human("a", nil), nwlRegular), "1500?")
	u := human("a", map[entity.VariantKey]entity.SingleRating{
		nwlRegular:              {Rating: 1712.6, RatingDeviation: 90},
		"NWL18.classic.blitz":   {Rating: 1600.4, RatingDeviation: 90.1},
		"CSW19.classic.regular": {Rating: 1800, RatingDeviation: 60},
	})
	is.Equal(s.DisplayRating(u, nwlRegular), "1713")
	is.Equal(s.DisplayRating(u, "NWL18.classic.blitz"), "1600?")
	is.Equal(s.DisplayRating(u, "NWL18.classic.rapid"), "1500?")
	b := bot("HastyBot")
	b.Profile.Ratings.Data = map[entity.VariantKey]entity.SingleRating{nwlRegular: {Rating: 1987, RatingDeviation: 60}}
	is.Equal(s.DisplayRating(b, nwlRegular), "1987")
	is.True(s.Established(entity.SingleRating{RatingDeviation: 90}))
	is.True(!s.Established(entity.SingleRating{RatingDeviation: 90.1}))
}

// V2 shows a bot's fixed rating, the seeded start for a pool the player hasn't
// played, and drops the "?" at RD 160.
func TestDisplayRatingV2(t *testing.T) {
	is := is.New(t)
	s := RatingSystemV2
	is.Equal(s.DisplayRating(human("a", nil), nwlRegular), "1200?")
	u := human("a", map[entity.VariantKey]entity.SingleRating{
		nwlRegular:            {Rating: 1650, RatingDeviation: 160},
		"NWL18.classic.blitz": {Rating: 1600, RatingDeviation: 161},
	})
	is.Equal(s.DisplayRating(u, nwlRegular), "1650")
	is.Equal(s.DisplayRating(u, "NWL18.classic.blitz"), "1600?")
	// Seeded from NWL regular, the lowest RD in the variant.
	is.Equal(s.DisplayRating(u, "NWL18.classic.corres"), "1650?")
	b := bot("BeginnerBot")
	b.Profile.Ratings.Data = map[entity.VariantKey]entity.SingleRating{nwlRegular: {Rating: 1361, RatingDeviation: 60}}
	is.Equal(s.DisplayRating(b, nwlRegular), "600")
	is.True(s.Established(entity.SingleRating{RatingDeviation: 160}))
	is.True(!s.Established(entity.SingleRating{RatingDeviation: 160.5}))
}

type fakeSetting struct {
	value string
	reads int
}

func (f *fakeSetting) RatingSystemSetting(context.Context) (string, error) {
	f.reads++
	return f.value, nil
}

func TestRatingSystemFromSetting(t *testing.T) {
	is := is.New(t)
	is.Equal(ratingSystemFromSetting("").Version, 1)
	is.Equal(ratingSystemFromSetting("1").Version, 1)
	is.True(!ratingSystemFromSetting("1").Paused)
	is.Equal(ratingSystemFromSetting("2").Version, 2)
	is.True(!ratingSystemFromSetting("2").Paused)
	is.True(ratingSystemFromSetting("paused").Paused)
	// An unknown value pauses rather than guess.
	is.True(ratingSystemFromSetting("3").Paused)
}

// Displayed ratings read the setting at most every few seconds; game end
// reads it every time.
func TestLiveRatingSystemReads(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	is.Equal(LiveRatingSystem(ctx).Version, 1)

	f := &fakeSetting{value: "2"}
	SetRatingSystemSettings(f)
	defer SetRatingSystemSettings(nil)
	is.Equal(LiveRatingSystem(ctx).Version, 2)
	f.value = "1"
	is.Equal(LiveRatingSystem(ctx).Version, 2) // cached
	is.Equal(f.reads, 1)
	s, err := ratingSystemForGameEnd(ctx)
	is.NoErr(err)
	is.Equal(s.Version, 1)
	is.Equal(f.reads, 2)
}

func TestRatingSpread(t *testing.T) {
	is := is.New(t)
	for _, c := range []struct {
		reason ipc.GameEndReason
		winner int
		s0, s1 int32
		want   int
		err    bool
	}{
		{ipc.GameEndReason_STANDARD, 0, 400, 350, 50, false},
		{ipc.GameEndReason_STANDARD, 1, 300, 450, -150, false},
		{ipc.GameEndReason_STANDARD, -1, 400, 400, 0, false},
		{ipc.GameEndReason_STANDARD, 1, 400, 350, 0, true}, // winner disagrees with the score
		{ipc.GameEndReason_CONSECUTIVE_ZEROES, 0, 300, 290, 10, false},
		{ipc.GameEndReason_ADJUDICATED, 0, 300, 290, 10, false},
		{ipc.GameEndReason_RESIGNED, 0, 10, 300, 125, false},
		{ipc.GameEndReason_TIME, 1, 400, 100, -125, false},
		{ipc.GameEndReason_FORCE_FORFEIT, 1, 0, 0, -125, false},
		{ipc.GameEndReason_TRIPLE_CHALLENGE, 0, 0, 30, 125, false},
		{ipc.GameEndReason_RESIGNED, -1, 0, 0, 0, true},
	} {
		got, err := RatingSpread(c.reason, c.winner, c.s0, c.s1)
		is.Equal(err != nil, c.err)
		is.Equal(got, c.want)
	}
}

// Pools tied on RD seed from the one played most recently.
func TestNewRatingV2TieBreak(t *testing.T) {
	is := is.New(t)
	others := map[entity.VariantKey]entity.SingleRating{
		"NWL18.classic.blitz":   {Rating: 1500, RatingDeviation: 60, LastGameTimestamp: 100},
		"NWL18.classic.regular": {Rating: 1700, RatingDeviation: 60, LastGameTimestamp: 200},
		"NWL18.classic.rapid":   {Rating: 1600, RatingDeviation: 60, LastGameTimestamp: 150},
	}
	r := RatingSystemV2.newRating(others, "NWL18.classic.corres", 1000)
	is.Equal(r.Rating, 1700.0)
	is.Equal(r.RatingDeviation, 250.0)
}
