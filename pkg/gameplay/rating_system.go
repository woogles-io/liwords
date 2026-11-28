package gameplay

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/glicko"
)

// RatingSystem is the set of rules that rates finished games. See
// docs/ratings_plan.md.
type RatingSystem struct {
	Version int
	Glicko  glicko.Params
	// FixedBotRatings, if set, gives every bot a rating that never changes,
	// by username. Humans are rated against it as if its RD were 0. Games
	// between two bots aren't rated.
	FixedBotRatings map[string]float64
	// SeedMaxRD, if set, starts a player's first game in a rating pool from
	// their rating in another pool of the same variant with an RD at most
	// this; see newRating.
	SeedMaxRD float64
	// SeedMinRD is the lowest RD a seeded rating starts with.
	SeedMinRD float64
	// Paused means finished games aren't rated. The switch to a new system
	// pauses ratings while it writes everyone's new ratings; it rates the
	// games that ended meanwhile from their saved results.
	Paused bool
}

var RatingSystemV1 = RatingSystem{Version: 1, Glicko: glicko.V1}

var RatingSystemV2 = RatingSystem{
	Version: 2,
	Glicko:  glicko.V2,
	FixedBotRatings: map[string]float64{
		"BeginnerBot": 600,
		"BasicBot":    1000,
		"BetterBot":   1400,
		"STEEBot":     1800,
		"HastyBot":    2050,
		"BestBot":     2200,
	},
	SeedMaxRD: 300,
	SeedMinRD: 250,
}

// RatingSystemSettings reads the live rating-system setting: "1", "2" or
// "paused". The config store implements it.
type RatingSystemSettings interface {
	RatingSystemSetting(ctx context.Context) (string, error)
}

// liveSetting caches the setting for displayed ratings and seek checks, which
// run often. Rating a finished game always reads it fresh.
var liveSetting struct {
	sync.Mutex
	src     RatingSystemSettings
	value   string
	fetched time.Time
}

const liveSettingTTL = 5 * time.Second

// SetRatingSystemSettings sets where the live rating system is read from.
// Without it (in tests and tools), the live system is V1.
func SetRatingSystemSettings(src RatingSystemSettings) {
	liveSetting.Lock()
	defer liveSetting.Unlock()
	liveSetting.src, liveSetting.value, liveSetting.fetched = src, "", time.Time{}
}

// LiveRatingSystem returns the live rating system, read from the setting at
// most every few seconds. While ratings are paused it returns V1 with Paused
// set: profiles still hold V1 ratings until the switch writes V2 ones.
func LiveRatingSystem(ctx context.Context) RatingSystem {
	liveSetting.Lock()
	defer liveSetting.Unlock()
	if liveSetting.src == nil {
		return RatingSystemV1
	}
	if time.Since(liveSetting.fetched) >= liveSettingTTL {
		v, err := liveSetting.src.RatingSystemSetting(ctx)
		if err != nil {
			// Keep the last value rather than guess.
			log.Err(err).Msg("read-rating-system-setting")
		} else {
			liveSetting.value, liveSetting.fetched = v, time.Now()
		}
	}
	return ratingSystemFromSetting(liveSetting.value)
}

// ratingSystemForGameEnd reads the setting fresh, so a game that ends right
// after a change is rated under the new setting.
func ratingSystemForGameEnd(ctx context.Context) (RatingSystem, error) {
	liveSetting.Lock()
	src := liveSetting.src
	liveSetting.Unlock()
	if src == nil {
		return RatingSystemV1, nil
	}
	v, err := src.RatingSystemSetting(ctx)
	if err != nil {
		return RatingSystem{}, err
	}
	return ratingSystemFromSetting(v), nil
}

// ratingSystemFromSetting maps a setting value to a rating system. An unknown
// value pauses ratings rather than rate games under the wrong system.
func ratingSystemFromSetting(v string) RatingSystem {
	switch v {
	case "", "1":
		return RatingSystemV1
	case "2":
		return RatingSystemV2
	case "paused":
	default:
		log.Error().Str("value", v).Msg("unknown-rating-system-setting")
	}
	s := RatingSystemV1
	s.Paused = true
	return s
}

// Established reports whether a rating is settled enough to show without a
// "?" and to meet a seek's established-rating requirement.
func (s RatingSystem) Established(r entity.SingleRating) bool {
	return r.RatingDeviation <= s.Glicko.EstablishedRatingDeviation
}

// DisplayRating is the rating shown for a user in a rating pool, with a "?"
// if it isn't established. A bot with a fixed rating shows that rating. A user
// with no rating in the pool shows the rating they would start it at.
func (s RatingSystem) DisplayRating(u *entity.User, key entity.VariantKey) string {
	if r, ok := s.FixedBotRatings[u.Username]; ok && u.IsBot {
		return strconv.Itoa(int(math.Round(r)))
	}
	if u.Profile == nil {
		return "UnratedAnon"
	}
	r, ok := u.Profile.Ratings.Data[key]
	if !ok {
		r = *s.newRating(u.Profile.Ratings.Data, key, 0)
	}
	q := "?"
	if ok && s.Established(r) {
		q = ""
	}
	return strconv.Itoa(int(math.Round(r.Rating))) + q
}

// newRating is the rating a player starts a pool with.
//
// With seeding on, it's their rating from another pool of the same variant
// whose RD is at most SeedMaxRD: the same lexicon family if they have one,
// then the lowest RD, then the most recently played. Its RD is at least
// SeedMinRD. Pools that are mostly
// human-vs-human (correspondence, for example) are anchored by where newcomers
// start, so starting a strong player at the initial rating drags the pool down.
// Otherwise it's the initial rating.
func (s RatingSystem) newRating(others map[entity.VariantKey]entity.SingleRating, key entity.VariantKey, now int64) *entity.SingleRating {
	start := &entity.SingleRating{
		Rating:            s.Glicko.InitialRating,
		RatingDeviation:   s.Glicko.InitialRatingDeviation,
		Volatility:        s.Glicko.InitialVolatility,
		LastGameTimestamp: now,
	}
	if s.SeedMaxRD == 0 {
		return start
	}
	lex, variant, ok := splitKey(key)
	if !ok {
		return start
	}
	var best *entity.SingleRating
	var bestKey entity.VariantKey
	bestSameLex := false
	for k, r := range others {
		klex, kvariant, ok := splitKey(k)
		if !ok || k == key || kvariant != variant || r.RatingDeviation > s.SeedMaxRD {
			continue
		}
		sameLex := klex == lex
		better := best == nil ||
			(sameLex && !bestSameLex) ||
			(sameLex == bestSameLex && (r.RatingDeviation < best.RatingDeviation ||
				(r.RatingDeviation == best.RatingDeviation && (r.LastGameTimestamp > best.LastGameTimestamp ||
					(r.LastGameTimestamp == best.LastGameTimestamp && k < bestKey)))))
		if better {
			r := r
			best, bestKey, bestSameLex = &r, k, sameLex
		}
	}
	if best == nil {
		return start
	}
	start.Rating = best.Rating
	start.RatingDeviation = max(best.RatingDeviation, s.SeedMinRD)
	return start
}

// splitKey splits a rating key such as "NWL18.classic.regular" into its
// lexicon family, variant and time control.
func splitKey(key entity.VariantKey) (lex, variant string, ok bool) {
	parts := strings.Split(string(key), ".")
	if len(parts) != 3 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// ErrBotsNotRated: games between two bots with fixed ratings aren't rated.
var ErrBotsNotRated = errors.New("games between bots with fixed ratings are not rated")

// ErrNoFixedRating: a bot without a fixed rating can't be rated against.
var ErrNoFixedRating = errors.New("bot has no fixed rating")

// RateGame rates one game. cur holds the players' saved ratings in the pool
// (nil if they have none yet), spread is from player 0's point of view, and
// users[i] is player i. It returns the ratings the game was played at and the
// new ratings; a nil new rating means that player's rating doesn't change
// (a bot with a fixed rating).
func (s RatingSystem) RateGame(cur [2]*entity.SingleRating, users []*entity.User,
	key entity.VariantKey, spread int, now int64) (before, after [2]*entity.SingleRating, err error) {

	fixed := [2]bool{}
	for i := range 2 {
		if s.FixedBotRatings == nil || !users[i].IsBot {
			continue
		}
		r, ok := s.FixedBotRatings[users[i].Username]
		if !ok {
			return before, after, fmt.Errorf("%w: %s", ErrNoFixedRating, users[i].Username)
		}
		fixed[i] = true
		before[i] = &entity.SingleRating{Rating: r, RatingDeviation: 0,
			Volatility: s.Glicko.InitialVolatility, LastGameTimestamp: now}
	}
	if fixed[0] && fixed[1] {
		return before, after, ErrBotsNotRated
	}
	for i := range 2 {
		if fixed[i] {
			continue
		}
		if cur[i] == nil {
			var others map[entity.VariantKey]entity.SingleRating
			if users[i].Profile != nil {
				others = users[i].Profile.Ratings.Data
			}
			before[i] = s.newRating(others, key, now)
		} else {
			r := *cur[i]
			if r.LastGameTimestamp == 0 {
				r.LastGameTimestamp = now
			}
			before[i] = &r
		}
	}
	for i := range 2 {
		if fixed[i] {
			continue
		}
		me, opp, sp := before[i], before[1-i], spread
		if i == 1 {
			sp = -spread
		}
		r, rd, v := s.Glicko.Rate(me.Rating, me.RatingDeviation, me.Volatility,
			opp.Rating, opp.RatingDeviation, sp, int(now-me.LastGameTimestamp))
		after[i] = &entity.SingleRating{Rating: r, RatingDeviation: rd, Volatility: v, LastGameTimestamp: now}
	}
	return before, after, nil
}
