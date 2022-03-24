package puzzles

import (
	"context"
	"fmt"
	"time"

	"github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/glicko"
	gamestore "github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/domino14/macondo/alphabet"
	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	macondopuzzles "github.com/domino14/macondo/puzzles"
	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"
)

type PuzzleStore interface {
	CreatePuzzle(ctx context.Context, gameID string, turnNumber int32, answer *macondopb.GameEvent, authorID string,
		beforeText string, afterText string, tags []macondopb.PuzzleTag) error
	GetRandomUnansweredPuzzleIdForUser(context.Context, string) (string, error)
	GetPuzzle(ctx context.Context, puzzleId string) (string, *macondopb.GameHistory, string, error)
	GetAnswer(ctx context.Context, puzzleId string) (*macondopb.GameEvent, string, *ipc.GameRequest, *entity.SingleRating, error)
	AnswerPuzzle(ctx context.Context, userId string, ratingKey entity.VariantKey, newUserRating *entity.SingleRating, puzzleId string, newPuzzleRating *entity.SingleRating, correct bool) error
	HasUserAttemptedPuzzle(ctx context.Context, userId string, puzzleId string) (bool, error)
	GetUserRating(ctx context.Context, userId string, ratingKey entity.VariantKey) (*entity.SingleRating, error)
	SetPuzzleVote(ctx context.Context, userId string, puzzleId string, vote int) error
}

func CreatePuzzlesFromGame(ctx context.Context, gs *gamestore.DBStore, ps PuzzleStore, mcg *macondogame.Game, authorId string, gt ipc.GameType) ([]*macondopb.PuzzleCreationResponse, error) {
	g := newPuzzleGame(mcg)
	pzls, err := macondopuzzles.CreatePuzzlesFromGame(g.Config(), mcg)
	if err != nil {
		return nil, err
	}

	// Only create if there were puzzles
	if len(pzls) > 0 {
		// If the mcg game is not from a game that already
		// exists in the database, then create the game
		if gt != ipc.GameType_NATIVE {
			err = gs.CreateRaw(ctx, g, gt)
			if err != nil {
				return nil, err
			}
		}

		for _, pzl := range pzls {
			err := ps.CreatePuzzle(ctx, pzl.GameId, pzl.TurnNumber, pzl.Answer, authorId, "", "", pzl.Tags)
			if err != nil {
				return nil, err
			}
		}
	}
	return pzls, nil
}

func GetRandomUnansweredPuzzleIdForUser(ctx context.Context, ps PuzzleStore, userId string) (string, error) {
	return ps.GetRandomUnansweredPuzzleIdForUser(ctx, userId)
}

func GetPuzzle(ctx context.Context, ps PuzzleStore, puzzleId string) (string, *macondopb.GameHistory, string, error) {
	return ps.GetPuzzle(ctx, puzzleId)
}

func GetAnswer(ctx context.Context, ps PuzzleStore, puzzleId string, userId string, userAnswer *macondopb.GameEvent) (bool, *macondopb.GameEvent, string, error) {
	answer, afterText, req, puzzleRating, err := ps.GetAnswer(ctx, puzzleId)
	if err != nil {
		return false, nil, "", err
	}

	correct := answersAreEqual(userAnswer, answer)

	// Check if user has already seen this puzzle
	attempted, err := ps.HasUserAttemptedPuzzle(ctx, userId, puzzleId)
	if err != nil {
		return false, nil, "", err
	}

	var newPuzzleSingleRating *entity.SingleRating
	var newUserSingleRating *entity.SingleRating
	rk := ratingKey(req)

	if !attempted {
		// Get the user ratings
		userRating, err := ps.GetUserRating(ctx, userId, rk)
		if err != nil {
			return false, nil, "", err
		}

		spread := glicko.SpreadScaling + 1

		if !correct {
			spread *= -1
		}

		var now = time.Now().Unix()
		newUserRating, newUserRatingDeviation, newUserVolatility := glicko.Rate(
			userRating.Rating, userRating.RatingDeviation, userRating.Volatility,
			puzzleRating.Rating, puzzleRating.RatingDeviation,
			spread, int(now-userRating.LastGameTimestamp),
		)
		newPuzzleRating, newPuzzleRatingDeviation, newPuzzleVolatility := glicko.Rate(
			puzzleRating.Rating, puzzleRating.RatingDeviation, puzzleRating.Volatility,
			userRating.Rating, userRating.RatingDeviation,
			-spread, int(now-puzzleRating.LastGameTimestamp),
		)

		newUserSingleRating = &entity.SingleRating{
			Rating:            newUserRating,
			RatingDeviation:   newUserRatingDeviation,
			Volatility:        newUserVolatility,
			LastGameTimestamp: now,
		}

		newPuzzleSingleRating = &entity.SingleRating{
			Rating:            newPuzzleRating,
			RatingDeviation:   newPuzzleRatingDeviation,
			Volatility:        newPuzzleVolatility,
			LastGameTimestamp: now,
		}
	}

	err = ps.AnswerPuzzle(ctx, userId, rk, newUserSingleRating, puzzleId, newPuzzleSingleRating, correct)
	if err != nil {
		return false, nil, "", err
	}
	return correct, answer, afterText, nil
}

func SetPuzzleVote(ctx context.Context, ps PuzzleStore, userId string, puzzleId string, vote int) error {
	if !(vote == -1 || vote == 0 || vote == 1) {
		return fmt.Errorf("puzzle vote must have a value of -1, 0, or 1 but got %d instead", vote)
	}
	return ps.SetPuzzleVote(ctx, userId, puzzleId, vote)
}

func newPuzzleGame(mcg *macondogame.Game) *entity.Game {
	g := entity.NewGame(mcg, common.DefaultGameReq)
	g.Started = true
	uuid := shortuuid.New()
	g.GameEndReason = ipc.GameEndReason_STANDARD
	g.Quickdata.FinalScores = []int32{int32(g.Game.PointsFor(0)), int32(g.Game.PointsFor(1))}
	g.Quickdata.PlayerInfo = []*ipc.PlayerInfo{&common.DefaultPlayerOneInfo, &common.DefaultPlayerTwoInfo}
	g.Game.History().Uid = uuid
	g.Game.History().PlayState = macondopb.PlayState_GAME_OVER
	g.Timers = entity.Timers{
		TimeRemaining: []int{0, 0},
		MaxOvertime:   0,
	}

	return g
}

func answersAreEqual(ans1 *macondopb.GameEvent, ans2 *macondopb.GameEvent) bool {
	if ans1 == nil {
		log.Info().Msg("user answer nil")
		return false
	}
	if ans2 == nil {
		log.Info().Msg("puzzle answer nil")
		return false
	}

	if ans1.Type == macondopb.GameEvent_TILE_PLACEMENT_MOVE &&
		ans2.Type == macondopb.GameEvent_TILE_PLACEMENT_MOVE &&
		countPlayedTiles(ans1) == 1 && countPlayedTiles(ans2) == 1 {
		return uniqueSingleTileKey(ans1) == uniqueSingleTileKey(ans2)
	}

	return ans1.Type == ans2.Type &&
		ans1.Row == ans2.Row &&
		ans1.Column == ans2.Column &&
		ans1.Direction == ans2.Direction &&
		ans1.PlayedTiles == ans2.PlayedTiles &&
		ans1.Exchanged == ans2.Exchanged
}

func countPlayedTiles(ge *macondopb.GameEvent) int {
	sum := 0
	for _, tile := range ge.PlayedTiles {
		if tile != alphabet.ASCIIPlayedThrough {
			sum++
		}
	}
	return sum
}

func uniqueSingleTileKey(ge *macondopb.GameEvent) int {
	// Find the tile.
	var idx int
	var tile rune
	for idx, tile = range ge.PlayedTiles {
		if tile != alphabet.ASCIIPlayedThrough {
			break
		}
	}

	var row, col int
	row = int(ge.Row)
	col = int(ge.Column)
	// We want to get the coordinate of the tile that is on the board itself.
	if ge.GetDirection() == macondopb.GameEvent_VERTICAL {
		row += idx
	} else {
		col += idx
	}
	// A unique, fast to compute key for this play.
	return row + alphabet.MaxAlphabetSize*col +
		alphabet.MaxAlphabetSize*alphabet.MaxAlphabetSize*int(tile)
}

func ratingKey(gameRequest *ipc.GameRequest) entity.VariantKey {
	return entity.ToVariantKey(gameRequest.Lexicon, common.PuzzleVariant, entity.TCCorres)
}
