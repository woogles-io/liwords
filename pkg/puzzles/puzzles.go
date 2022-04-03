package puzzles

import (
	"context"
	"strconv"
	"time"

	"github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/glicko"
	gamestore "github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/pkg/utilities"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/domino14/macondo/alphabet"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/macondo/move"
	macondopuzzles "github.com/domino14/macondo/puzzles"
	"github.com/rs/zerolog/log"
)

type PuzzleStore interface {
	CreatePuzzle(ctx context.Context, gameID string, turnNumber int32, answer *macondopb.GameEvent, authorID string,
		lexicon string, beforeText string, afterText string, tags []macondopb.PuzzleTag) error
	GetStartPuzzleId(ctx context.Context, userId string, lexicon string) (string, error)
	GetNextPuzzleId(ctx context.Context, userId string, lexicon string) (string, error)
	GetPuzzle(ctx context.Context, userId string, puzzleUUID string) (*macondopb.GameHistory, string, int32, *bool, time.Time, time.Time, error)
	GetPreviousPuzzleId(ctx context.Context, userId string, puzzleUUID string) (string, error)
	GetAnswer(ctx context.Context, puzzleUUID string) (*macondopb.GameEvent, string, int32, string, *ipc.GameRequest, *entity.SingleRating, error)
	SubmitAnswer(ctx context.Context, userId string, ratingKey entity.VariantKey, newUserRating *entity.SingleRating,
		puzzleUUID string, newPuzzleRating *entity.SingleRating, userIsCorrect bool, userGaveUp bool) error
	GetAttempts(ctx context.Context, userId string, puzzleUUID string) (bool, int32, *bool, time.Time, time.Time, error)
	GetUserRating(ctx context.Context, userId string, ratingKey entity.VariantKey) (*entity.SingleRating, error)
	SetPuzzleVote(ctx context.Context, userId string, puzzleUUID string, vote int) error
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
			err := ps.CreatePuzzle(ctx, pzl.GameId, pzl.TurnNumber, pzl.Answer, authorId, g.GameReq.Lexicon, "", "", pzl.Tags)
			if err != nil {
				return nil, err
			}
		}
	}
	return pzls, nil
}

func GetStartPuzzleId(ctx context.Context, ps PuzzleStore, userId string, lexicon string) (string, error) {
	return ps.GetStartPuzzleId(ctx, userId, lexicon)
}

func GetNextPuzzleId(ctx context.Context, ps PuzzleStore, userId string, lexicon string) (string, error) {
	return ps.GetNextPuzzleId(ctx, userId, lexicon)
}

func GetPuzzle(ctx context.Context, ps PuzzleStore, userId string, puzzleUUID string) (*macondopb.GameHistory, string, int32, *bool, time.Time, time.Time, error) {
	return ps.GetPuzzle(ctx, userId, puzzleUUID)
}

func GetPreviousPuzzleId(ctx context.Context, ps PuzzleStore, userId string, puzzleUUID string) (string, error) {
	return ps.GetPreviousPuzzleId(ctx, userId, puzzleUUID)
}

func SubmitAnswer(ctx context.Context, ps PuzzleStore, puzzleUUID string, userId string,
	userAnswer *ipc.ClientGameplayEvent, showSolution bool) (bool, *bool, *macondopb.GameEvent, string, int32, string, int32, int32, int32, time.Time, time.Time, error) {
	correctAnswer, gameId, turnNumber, afterText, req, puzzleRating, err := ps.GetAnswer(ctx, puzzleUUID)
	if err != nil {
		return false, nil, nil, "", -1, "", -1, -1, -1, time.Time{}, time.Time{}, err
	}
	userIsCorrect := answersAreEqual(userAnswer, correctAnswer)
	// Check if user has already seen this puzzle
	_, attempts, status, _, _, err := ps.GetAttempts(ctx, userId, puzzleUUID)
	if err != nil {
		return false, nil, nil, "", -1, "", -1, -1, -1, time.Time{}, time.Time{}, err
	}
	log.Debug().Interface("status", status).
		Int32("attempts", attempts).Msg("equal")
	var newPuzzleSingleRating *entity.SingleRating
	var newUserSingleRating *entity.SingleRating
	rk := ratingKey(req)

	if attempts == 0 && status == nil {
		// Get the user ratings
		userRating, err := ps.GetUserRating(ctx, userId, rk)
		if err != nil {
			return false, nil, nil, "", -1, "", -1, -1, -1, time.Time{}, time.Time{}, err
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

	err = ps.SubmitAnswer(ctx, userId, rk, newUserSingleRating, puzzleUUID, newPuzzleSingleRating, userIsCorrect, showSolution)
	if err != nil {
		return false, nil, nil, "", -1, "", -1, -1, -1, time.Time{}, time.Time{}, err
	}

	_, attempts, status, firstAttemptTime, lastAttemptTime, err := ps.GetAttempts(ctx, userId, puzzleUUID)
	if err != nil {
		return false, nil, nil, "", -1, "", -1, -1, -1, time.Time{}, time.Time{}, err
	}

	if !showSolution && !userIsCorrect {
		correctAnswer = nil
		gameId = ""
	}

	nur := int32(0)
	npr := int32(0)
	if newUserSingleRating != nil {
		nur = int32(newUserSingleRating.Rating)
	}
	if newPuzzleSingleRating != nil {
		npr = int32(newPuzzleSingleRating.Rating)
	}

	return userIsCorrect, status, correctAnswer, gameId, turnNumber, afterText, attempts, nur, npr, firstAttemptTime, lastAttemptTime, nil
}

func SetPuzzleVote(ctx context.Context, ps PuzzleStore, userId string, puzzleUUID string, vote int) error {
	if !(vote == -1 || vote == 0 || vote == 1) {
		return entity.NewWooglesError(ipc.WooglesError_PUZZLE_VOTE_INVALID, userId, puzzleUUID, strconv.Itoa(vote))
	}
	return ps.SetPuzzleVote(ctx, userId, puzzleUUID, vote)
}

func answersAreEqual(userAnswer *ipc.ClientGameplayEvent, correctAnswer *macondopb.GameEvent) bool {
	if userAnswer == nil {
		// The user answer is nil when they have given up
		// and just want the answer without making an attempt
		return false
	}
	// Convert the ClientGameplayEvent to a macondo GameEvent:
	converted := &macondopb.GameEvent{}

	switch userAnswer.Type {
	case ipc.ClientGameplayEvent_TILE_PLACEMENT:
		converted.Type = macondopb.GameEvent_TILE_PLACEMENT_MOVE
		converted.PlayedTiles = userAnswer.Tiles
		row, col, vertical := move.FromBoardGameCoords(userAnswer.PositionCoords)

		converted.Row = int32(row)
		converted.Column = int32(col)
		if vertical {
			converted.Direction = macondopb.GameEvent_VERTICAL
		} else {
			converted.Direction = macondopb.GameEvent_HORIZONTAL
		}
	case ipc.ClientGameplayEvent_EXCHANGE:
		converted.Type = macondopb.GameEvent_EXCHANGE
		converted.Exchanged = userAnswer.Tiles
	case ipc.ClientGameplayEvent_PASS:
		converted.Type = macondopb.GameEvent_PASS
	}

	if correctAnswer == nil {
		log.Info().Msg("puzzle answer nil")
		return false
	}
	log.Debug().Interface("converted", converted).Msg("converted-event")

	if converted.Type == macondopb.GameEvent_TILE_PLACEMENT_MOVE &&
		correctAnswer.Type == macondopb.GameEvent_TILE_PLACEMENT_MOVE &&
		countPlayedTiles(converted) == 1 && countPlayedTiles(correctAnswer) == 1 {
		return uniqueSingleTileKey(converted) == uniqueSingleTileKey(correctAnswer)
	}

	return converted.Type == correctAnswer.Type &&
		converted.Row == correctAnswer.Row &&
		converted.Column == correctAnswer.Column &&
		converted.Direction == correctAnswer.Direction &&
		converted.PlayedTiles == correctAnswer.PlayedTiles &&
		utilities.SortString(converted.Exchanged) == utilities.SortString(correctAnswer.Exchanged)
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
