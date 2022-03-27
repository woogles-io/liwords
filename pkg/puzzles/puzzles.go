package puzzles

import (
	"context"
	"fmt"
	"time"

	"github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/glicko"
	gamestore "github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/pkg/utilities"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/domino14/macondo/alphabet"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	macondopuzzles "github.com/domino14/macondo/puzzles"
	"github.com/rs/zerolog/log"
)

type PuzzleStore interface {
	CreatePuzzle(ctx context.Context, gameID string, turnNumber int32, answer *macondopb.GameEvent, authorID string,
		beforeText string, afterText string, tags []macondopb.PuzzleTag) error
	GetRandomUnansweredPuzzleIdForUser(context.Context, string) (string, error)
	GetPuzzle(ctx context.Context, userId string, puzzleId string) (*macondopb.GameHistory, string, int32, error)
	GetAnswer(ctx context.Context, puzzleId string) (*macondopb.GameEvent, string, string, *ipc.GameRequest, *entity.SingleRating, error)
	SubmitAnswer(ctx context.Context, userId string, ratingKey entity.VariantKey, newUserRating *entity.SingleRating,
		puzzleId string, newPuzzleRating *entity.SingleRating, userIsCorrect bool, userGaveUp bool) error
	GetAttempts(ctx context.Context, userId string, puzzleId string) (int32, error)
	GetUserRating(ctx context.Context, userId string, ratingKey entity.VariantKey) (*entity.SingleRating, error)
	SetPuzzleVote(ctx context.Context, userId string, puzzleId string, vote int) error
}

func CreatePuzzlesFromGame(ctx context.Context, gs *gamestore.DBStore, ps PuzzleStore,
	g *entity.Game, authorId string, gt ipc.GameType) ([]*macondopb.PuzzleCreationResponse, error) {

	pzls, err := macondopuzzles.CreatePuzzlesFromGame(g.Config(), &g.Game)
	if err != nil {
		return nil, err
	}
	log.Info().Msgf("created %d puzzles from game %s", len(pzls), g.GameID())

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

func GetPuzzle(ctx context.Context, ps PuzzleStore, userId string, puzzleId string) (*macondopb.GameHistory, string, int32, error) {
	return ps.GetPuzzle(ctx, userId, puzzleId)
}

func SubmitAnswer(ctx context.Context, ps PuzzleStore, puzzleId string, userId string, userAnswer *macondopb.GameEvent, showSolution bool) (bool, *macondopb.GameEvent, string, string, int32, error) {
	correctAnswer, gameId, afterText, req, puzzleRating, err := ps.GetAnswer(ctx, puzzleId)
	if err != nil {
		return false, nil, "", "", -1, err
	}

	userIsCorrect := answersAreEqual(userAnswer, correctAnswer)

	// Check if user has already seen this puzzle
	attempts, err := ps.GetAttempts(ctx, userId, puzzleId)
	if err != nil {
		return false, nil, "", "", -1, err
	}

	var newPuzzleSingleRating *entity.SingleRating
	var newUserSingleRating *entity.SingleRating
	rk := ratingKey(req)

	if attempts == 0 {
		// Get the user ratings
		userRating, err := ps.GetUserRating(ctx, userId, rk)
		if err != nil {
			return false, nil, "", "", -1, err
		}

		spread := glicko.SpreadScaling + 1

		if !userIsCorrect {
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

	err = ps.SubmitAnswer(ctx, userId, rk, newUserSingleRating, puzzleId, newPuzzleSingleRating, userIsCorrect, showSolution)
	if err != nil {
		return false, nil, "", "", -1, err
	}

	attempts, err = ps.GetAttempts(ctx, userId, puzzleId)
	if err != nil {
		return false, nil, "", "", -1, err
	}

	if !showSolution && !userIsCorrect {
		correctAnswer = nil
		gameId = ""
	}

	return userIsCorrect, correctAnswer, gameId, afterText, attempts, nil
}

func SetPuzzleVote(ctx context.Context, ps PuzzleStore, userId string, puzzleId string, vote int) error {
	if !(vote == -1 || vote == 0 || vote == 1) {
		return fmt.Errorf("puzzle vote must have a value of -1, 0, or 1 but got %d instead", vote)
	}
	return ps.SetPuzzleVote(ctx, userId, puzzleId, vote)
}

func answersAreEqual(userAnswer *macondopb.GameEvent, correctAnswer *macondopb.GameEvent) bool {
	if userAnswer == nil {
		// The user answer is nil when they have given up
		// and just want the answer without making an attempt
		return false
	}
	if correctAnswer == nil {
		log.Info().Msg("puzzle answer nil")
		return false
	}

	if userAnswer.Type == macondopb.GameEvent_TILE_PLACEMENT_MOVE &&
		correctAnswer.Type == macondopb.GameEvent_TILE_PLACEMENT_MOVE &&
		countPlayedTiles(userAnswer) == 1 && countPlayedTiles(correctAnswer) == 1 {
		return uniqueSingleTileKey(userAnswer) == uniqueSingleTileKey(correctAnswer)
	}

	return userAnswer.Type == correctAnswer.Type &&
		userAnswer.Row == correctAnswer.Row &&
		userAnswer.Column == correctAnswer.Column &&
		userAnswer.Direction == correctAnswer.Direction &&
		userAnswer.PlayedTiles == correctAnswer.PlayedTiles &&
		utilities.SortString(userAnswer.Exchanged) == utilities.SortString(correctAnswer.Exchanged)
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
