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
	winnerIdx := -1
	if winner == usernames[0] {
		winnerIdx = 0
	} else if winner == usernames[1] {
		winnerIdx = 1
	}
	spread, err := RatingSpread(g.GameEndReason, winnerIdx,
		scores[usernames[0]], scores[usernames[1]])
	if err != nil {
		return nil, err
	}
	log.Debug().Str("p0", usernames[0]).Str("p1", usernames[1]).Int("spread", spread).Msg("rating")

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
			before, after, err = sys.RateGame([2]*entity.SingleRating{r0, r1}, users, ratingKey, spread, now)
			if err != nil {
				return nil, nil, err
			}
			return after[0], after[1], nil
		})
	if errors.Is(err, ErrBotsNotRated) {
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

// RatingSpread is the spread a game is rated with, from player 0's point of
// view. winnerIdx is the winner's index, or -1 if there is none. Games that
// end by resignation, time, forfeit or triple challenge count as the maximum
// spread; otherwise it's the score difference, which must agree with the
// winner.
func RatingSpread(reason pb.GameEndReason, winnerIdx int, score0, score1 int32) (int, error) {
	switch reason {
	case pb.GameEndReason_RESIGNED, pb.GameEndReason_TIME,
		pb.GameEndReason_FORCE_FORFEIT, pb.GameEndReason_TRIPLE_CHALLENGE:
		switch winnerIdx {
		case 0:
			return glicko.SpreadScaling, nil
		case 1:
			return -glicko.SpreadScaling, nil
		}
		return 0, errors.New("no winner, but maximum penalty?")
	}
	spread := int(score0 - score1)
	if spread > 0 && winnerIdx != 0 || spread < 0 && winnerIdx != 1 {
		return 0, errors.New("winner does not match spread")
	}
	return spread, nil
}
