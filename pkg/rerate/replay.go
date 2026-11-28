package rerate

import (
	"errors"
	"regexp"
	"sort"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
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

// The early bots MacondoBot1, MacondoBot2, ... were HastyBot under other
// names.
var hastyBotAlias = regexp.MustCompile(`^MacondoBot\d*$`)

// RatedAs is the name a player is rated under: their username, except that
// the early names of fixed-rating bots map to the bot's current name.
func RatedAs(username string) string {
	if hastyBotAlias.MatchString(username) {
		return "HastyBot"
	}
	return username
}

// Result returns a game's result as a replay rates it: its end time, scores
// and winner, with a league penalty for cheating applied. For a game that
// ended normally the winner follows the scores; the recorded winner only
// decides games that ended by a quit, whose scores don't.
func (d *Data) Result(g *Game) (end int64, scores [2]int32, winner int8) {
	end, scores, winner = g.End, g.Scores, g.Winner
	if p, ok := d.Penalties[g.ID]; ok {
		scores, winner = p.Scores, 1-p.Loser
		// A game forfeited by the penalty ended then; any other ended
		// before it.
		if p.End > 0 && p.End < end && g.Reason != pb.GameEndReason_FORCE_FORFEIT {
			end = p.End
		}
	}
	if !quit(g.Reason) {
		switch {
		case scores[0] > scores[1]:
			winner = 0
		case scores[0] < scores[1]:
			winner = 1
		default:
			winner = -1
		}
	}
	return end, scores, winner
}

func quit(r pb.GameEndReason) bool {
	return r == pb.GameEndReason_RESIGNED || r == pb.GameEndReason_TIME ||
		r == pb.GameEndReason_FORCE_FORFEIT || r == pb.GameEndReason_TRIPLE_CHALLENGE
}

type replayGame struct {
	*Game
	end    int64
	scores [2]int32
	winner int8
}

// Replay rates the games in end-time order under sys, exactly as games are
// rated when they end (gameplay.RatingSpread and RatingSystem.RateGame).
func Replay(sys gameplay.RatingSystem, d *Data, opts ReplayOptions) *Result {
	games := make([]replayGame, 0, len(d.Games))
	for i := range d.Games {
		g := replayGame{Game: &d.Games[i]}
		g.end, g.scores, g.winner = d.Result(g.Game)
		if (opts.Since == 0 || g.end >= opts.Since) && (opts.Until == 0 || g.end < opts.Until) {
			games = append(games, g)
		}
	}
	sort.Slice(games, func(i, j int) bool {
		if games[i].end != games[j].end {
			return games[i].end < games[j].end
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
		for len(resets) > 0 && resets[0].Time <= g.end {
			if resets[0].Time >= opts.Since {
				delete(res.Ratings, resets[0].UserID)
			}
			resets = resets[1:]
		}
		key := d.Keys[g.Key]
		spread, err := gameplay.RatingSpread(g.Reason, int(g.winner), g.scores[0], g.scores[1])
		if err != nil {
			res.Skipped["quit without a winner"]++
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
				ID: uint(id), UUID: u.UUID, Username: RatedAs(u.Username),
				IsBot:   u.Bot || g.QuickdataBot[i],
				Profile: &entity.Profile{Ratings: entity.Ratings{Data: ratings}},
			}
			if r, ok := ratings[key]; ok {
				cur[i] = &r
			}
		}
		before, after, err := sys.RateGame(cur, users[:], key, spread, g.end)
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
			opts.OnRated(g.Game, before, after)
		}
		res.Rated++
		res.Last, res.LastID = g.end, g.ID
	}
	for id, r := range res.Ratings {
		if len(r) == 0 {
			delete(res.Ratings, id)
		}
	}
	return res
}
