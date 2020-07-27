package gameplay

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/glicko"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/rs/zerolog/log"
)

func rate(ctx context.Context, scores map[string]int32, g *entity.Game,
	winner string, userStore user.Store) (map[string]int32, error) {

	// Fetch the users from the store.
	users := []*entity.User{}
	usernames := []string{}
	for username := range scores {
		p, err := userStore.Get(ctx, username)
		if err != nil {
			return nil, err
		}
		users = append(users, p)
		usernames = append(usernames, username)
	}
	ratingKey, err := g.RatingKey()
	if err != nil {
		return nil, err
	}
	// We have two users. Rate them.
	// If the game ended because of the following, apply the maximum spread.
	maxPenalty := g.GameEndReason == pb.GameEndReason_RESIGNED ||
		g.GameEndReason == pb.GameEndReason_ABANDONED ||
		g.GameEndReason == pb.GameEndReason_TIME
	// Get the user ratings
	rat0, err := users[0].GetRating(ratingKey)
	if err != nil {
		return nil, err
	}
	rat1, err := users[1].GetRating(ratingKey)
	if err != nil {
		return nil, err
	}

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

	var now = time.Now().Unix()
	if rat0.LastGameTimestamp == 0 {
		rat0.LastGameTimestamp = now
	}
	if rat1.LastGameTimestamp == 0 {
		rat1.LastGameTimestamp = now
	}
	// Rate for each player separately.
	p0rat, p0rd, p0v := glicko.Rate(
		rat0.Rating, rat0.RatingDeviation, rat0.Volatility,
		rat1.Rating, rat1.RatingDeviation,
		spread, int(now-rat0.LastGameTimestamp),
	)
	p1rat, p1rd, p1v := glicko.Rate(
		rat1.Rating, rat1.RatingDeviation, rat1.Volatility,
		rat0.Rating, rat0.RatingDeviation,
		-spread, int(now-rat1.LastGameTimestamp),
	)

	// Save the new ratings. This should probably be in some sort of
	// transaction, but ..
	err = userStore.SetRating(ctx, users[0].UUID, ratingKey, entity.SingleRating{
		Rating:            p0rat,
		RatingDeviation:   p0rd,
		Volatility:        p0v,
		LastGameTimestamp: now,
	})
	if err != nil {
		return nil, err
	}
	err = userStore.SetRating(ctx, users[1].UUID, ratingKey, entity.SingleRating{
		Rating:            p1rat,
		RatingDeviation:   p1rd,
		Volatility:        p1v,
		LastGameTimestamp: now,
	})
	if err != nil {
		return nil, err
	}

	return map[string]int32{
		usernames[0]: int32(math.Round(p0rat)),
		usernames[1]: int32(math.Round(p1rat)),
	}, nil
}
