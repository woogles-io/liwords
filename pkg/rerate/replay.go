package rerate

import (
	"errors"
	"sort"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
)

// ReplayOptions limit a replay to games that ended in [Since, Until); zero
// means no limit. A replay with Since starts everyone from scratch then.
type ReplayOptions struct {
	Since, Until int64
	// OnRated, if set, is called for each rated game with the ratings it was
	// played at and the new ones (nil for a bot's fixed rating).
	OnRated func(g *Game, before, after [2]*entity.SingleRating)
}

// Result is where a replay ends up.
type Result struct {
	Ratings map[int64]map[entity.VariantKey]entity.SingleRating
	// Games counts each player's rated games per pool.
	Games   map[int64]map[entity.VariantKey]int
	Rated   int
	Skipped map[string]int
	Last    int64 // end time of the last game replayed
	LastID  int64
}

// Replay rates the games in end-time order under sys, exactly as games are
// rated when they end (gameplay.RatingSpread and RatingSystem.RateGame).
func Replay(sys gameplay.RatingSystem, d *Data, opts ReplayOptions) *Result {
	games := make([]*Game, 0, len(d.Games))
	for i := range d.Games {
		g := &d.Games[i]
		if (opts.Since == 0 || g.End >= opts.Since) && (opts.Until == 0 || g.End < opts.Until) {
			games = append(games, g)
		}
	}
	sort.Slice(games, func(i, j int) bool {
		if games[i].End != games[j].End {
			return games[i].End < games[j].End
		}
		return games[i].ID < games[j].ID
	})

	res := &Result{
		Ratings: map[int64]map[entity.VariantKey]entity.SingleRating{},
		Games:   map[int64]map[entity.VariantKey]int{},
		Skipped: map[string]int{},
	}
	resets := d.Resets
	for _, g := range games {
		for len(resets) > 0 && resets[0].Time <= g.End {
			if resets[0].Time >= opts.Since {
				delete(res.Ratings, resets[0].UserID)
			}
			resets = resets[1:]
		}
		key := d.Keys[g.Key]
		spread, err := gameplay.RatingSpread(g.Reason, int(g.Winner), g.Scores[0], g.Scores[1])
		if err != nil {
			res.Skipped["winner does not match result"]++
			continue
		}
		var users [2]*entity.User
		var cur [2]*entity.SingleRating
		for i, id := range g.Players {
			u := d.Users[id]
			ratings := res.Ratings[id]
			if ratings == nil {
				ratings = map[entity.VariantKey]entity.SingleRating{}
				res.Ratings[id] = ratings
			}
			users[i] = &entity.User{
				ID: uint(id), UUID: u.UUID, Username: u.Username,
				IsBot:   u.Bot || g.QuickdataBot[i],
				Profile: &entity.Profile{Ratings: entity.Ratings{Data: ratings}},
			}
			if r, ok := ratings[key]; ok {
				cur[i] = &r
			}
		}
		before, after, err := sys.RateGame(cur, users[:], key, spread, g.End)
		switch {
		case errors.Is(err, gameplay.ErrBotsNotRated):
			res.Skipped["bot vs bot"]++
			continue
		case errors.Is(err, gameplay.ErrNoFixedRating):
			res.Skipped["bot without a fixed rating"]++
			continue
		case err != nil:
			res.Skipped[err.Error()]++
			continue
		}
		for i, id := range g.Players {
			if after[i] == nil {
				continue
			}
			res.Ratings[id][key] = *after[i]
			if res.Games[id] == nil {
				res.Games[id] = map[entity.VariantKey]int{}
			}
			res.Games[id][key]++
		}
		if opts.OnRated != nil {
			opts.OnRated(g, before, after)
		}
		res.Rated++
		res.Last, res.LastID = g.End, g.ID
	}
	for id, r := range res.Ratings {
		if len(r) == 0 {
			delete(res.Ratings, id)
		}
	}
	return res
}
