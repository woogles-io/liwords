package gameplay

import (
	"context"
	"errors"
	"math"

	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/glicko"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Rate rates the game and saves the new ratings. users[i] must be the user
// for g.History().Players[i]; their in-memory ratings are updated to match.
func Rate(ctx context.Context, scores map[string]int32, g *entity.Game,
	winner string, users []*entity.User, userStore user.Store, now int64) (map[string][2]int32, error) {

	usernames := []string{g.History().Players[0].Nickname, g.History().Players[1].Nickname}
	ratingKey, err := g.RatingKey()
	if err != nil {
		return nil, err
	}
	// We have two users. Rate them.
	// If the game ended because of the following, apply the maximum spread.
	maxPenalty := g.GameEndReason == pb.GameEndReason_RESIGNED ||
		g.GameEndReason == pb.GameEndReason_TIME ||
		g.GameEndReason == pb.GameEndReason_FORCE_FORFEIT ||
		g.GameEndReason == pb.GameEndReason_TRIPLE_CHALLENGE

	// What is the spread from the point of view of users[0]?
	var spread int
	if maxPenalty {
		if winner == usernames[0] {
			spread = glicko.SpreadScaling
		} else if winner == usernames[1] {
			spread = -glicko.SpreadScaling
		} else {
			return nil, errors.New("no winner, but maximum penalty?")
		}
		log.Debug().Str("p0", usernames[0]).Str("p1", usernames[1]).Int("spread", spread).Msg("rating-max-penalty")
	} else {
		// The winner is the person with the higher points. Calculate
		// from the point of view of users[0] again. We will negate this
		// when rating in the other direction.
		spread = int(scores[usernames[0]] - scores[usernames[1]])
		if spread > 0 && winner != usernames[0] || spread < 0 && winner != usernames[1] {
			return nil, errors.New("winner does not match spread")
		}
		log.Debug().Str("p0", usernames[0]).Str("p1", usernames[1]).Int("spread", spread).Msg("rating")
	}

	sys, err := ratingSystemForGameEnd(ctx)
	if err != nil {
		return nil, err
	}
	if sys.Paused {
		log.Info().Str("gid", g.GameID()).Msg("ratings-paused-not-rating")
		return map[string][2]int32{}, nil
	}
	// Rated from the ratings saved now, not the ones loaded with the users:
	// a game of theirs that ended since has changed them.
	var before, after [2]*entity.SingleRating
	err = userStore.UpdateRatings(ctx, users[0].UUID, users[1].UUID, ratingKey,
		func(r0, r1 *entity.SingleRating) (*entity.SingleRating, *entity.SingleRating, error) {
			var err error
			before, after, err = sys.rateGame([2]*entity.SingleRating{r0, r1}, users, ratingKey, spread, now)
			if err != nil {
				return nil, nil, err
			}
			return after[0], after[1], nil
		})
	if errors.Is(err, errBotsNotRated) {
		return map[string][2]int32{}, nil
	}
	if err != nil {
		return nil, err
	}
	// A player whose rating didn't change (a bot with a fixed rating) shows
	// the same rating before and after.
	newRat := [2]float64{}
	for i := range 2 {
		newRat[i] = before[i].Rating
		if after[i] != nil {
			newRat[i] = after[i].Rating
		}
	}

	g.Quickdata.OriginalRatings = []float64{before[0].Rating, before[1].Rating}
	g.Quickdata.NewRatings = []float64{newRat[0], newRat[1]}

	for i, r := range after {
		if r == nil {
			continue
		}
		if users[i].Profile.Ratings.Data == nil {
			users[i].Profile.Ratings.Data = map[entity.VariantKey]entity.SingleRating{}
		}
		users[i].Profile.Ratings.Data[ratingKey] = *r
	}

	return map[string][2]int32{
		usernames[0]: [2]int32{int32(math.Round(before[0].Rating)), int32(math.Round(newRat[0]))},
		usernames[1]: [2]int32{int32(math.Round(before[1].Rating)), int32(math.Round(newRat[1]))},
	}, nil
}
