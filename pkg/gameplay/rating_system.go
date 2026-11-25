package gameplay

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/woogles-io/liwords/pkg/config"
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

// ratingSystemFor returns the rating system the config picks, or V1.
func ratingSystemFor(ctx context.Context) RatingSystem {
	if cfg, err := config.Ctx(ctx); err == nil && cfg.RatingSystem == 2 {
		return RatingSystemV2
	}
	return RatingSystemV1
}

// newRating is the rating a player starts a pool with.
//
// With seeding on, it's their rating from another pool of the same variant
// whose RD is at most SeedMaxRD: the same lexicon family if they have one,
// then the lowest RD. Its RD is at least SeedMinRD. Pools that are mostly
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
				(r.RatingDeviation == best.RatingDeviation && k < bestKey)))
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

var errBotsNotRated = errors.New("games between bots with fixed ratings are not rated")

// rateGame rates one game. cur holds the players' saved ratings in the pool
// (nil if they have none yet), spread is from player 0's point of view, and
// users[i] is player i. It returns the ratings the game was played at and the
// new ratings; a nil new rating means that player's rating doesn't change
// (a bot with a fixed rating).
func (s RatingSystem) rateGame(cur [2]*entity.SingleRating, users []*entity.User,
	key entity.VariantKey, spread int, now int64) (before, after [2]*entity.SingleRating, err error) {

	fixed := [2]bool{}
	for i := range 2 {
		if s.FixedBotRatings == nil || !users[i].IsBot {
			continue
		}
		r, ok := s.FixedBotRatings[users[i].Username]
		if !ok {
			return before, after, fmt.Errorf("bot %s has no fixed rating", users[i].Username)
		}
		fixed[i] = true
		before[i] = &entity.SingleRating{Rating: r, RatingDeviation: 0,
			Volatility: s.Glicko.InitialVolatility, LastGameTimestamp: now}
	}
	if fixed[0] && fixed[1] {
		return before, after, errBotsNotRated
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
