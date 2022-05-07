package puzzles

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"math"
	"testing"
	"time"

	"github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/glicko"
	commondb "github.com/domino14/liwords/pkg/stores/common"
	gamestore "github.com/domino14/liwords/pkg/stores/game"
	puzzlesstore "github.com/domino14/liwords/pkg/stores/puzzles"
	"github.com/domino14/liwords/pkg/stores/user"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/domino14/liwords/rpc/api/proto/puzzle_service"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/macondo/board"
	"github.com/domino14/macondo/game"
	"github.com/domino14/macondo/gcgio"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/macondo/move"
)

const PuzzlerUUID = "puzzler"
const PuzzleCreatorUUID = "kenji"
const OtherLexicon = "CSW19"

func gameEventToClientGameplayEvent(evt *pb.GameEvent) *ipc.ClientGameplayEvent {
	cge := &ipc.ClientGameplayEvent{}

	switch evt.Type {
	case pb.GameEvent_TILE_PLACEMENT_MOVE:
		cge.Type = ipc.ClientGameplayEvent_TILE_PLACEMENT
		cge.Tiles = evt.PlayedTiles
		cge.PositionCoords = move.ToBoardGameCoords(int(evt.Row), int(evt.Column),
			evt.Direction == pb.GameEvent_VERTICAL)

	case pb.GameEvent_EXCHANGE:
		cge.Type = ipc.ClientGameplayEvent_EXCHANGE
		cge.Tiles = evt.Exchanged
	case pb.GameEvent_PASS:
		cge.Type = ipc.ClientGameplayEvent_PASS
	}

	return cge
}

type DBController struct {
	pool *pgxpool.Pool
	ps   *puzzlesstore.DBStore
	us   *user.DBStore
	gs   gameplay.GameStore
}

func (dbc *DBController) cleanup() {
	dbc.us.Disconnect()
	dbc.gs.(*gamestore.Cache).Disconnect()
	dbc.ps.Disconnect()
	dbc.pool.Close()
}

func TestPuzzlesMain(t *testing.T) {
	is := is.New(t)
	dbc, authoredPuzzles, totalPuzzles := RecreateDB()
	defer func() {
		dbc.cleanup()
	}()
	pool, ps := dbc.pool, dbc.ps

	ctx := context.Background()

	rk := ratingKey(common.DefaultGameReq.Lexicon)

	pcid, err := transactGetDBIDFromUUID(ctx, pool, "users", PuzzleCreatorUUID)
	is.NoErr(err)

	var curatedPuzzles int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM puzzles WHERE author_id = $1`, pcid).Scan(&curatedPuzzles)
	is.NoErr(err)
	is.Equal(curatedPuzzles, authoredPuzzles)

	// Paths
	//   - First attempt
	// 1   - Incorrect, don't show solution
	// 2   - Show solution
	// 3   - Correct
	//   - After first attempt
	// 4   - puzzle over
	//     - puzzle not over
	// 5     - Incorrect, don't show solution
	// 6     - Show solution
	// 7     - Correct

	// This should work for users who are not logged in
	_, err = GetNextPuzzleId(ctx, ps, "", common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	// Path 1
	// Submit an incorrect answer
	puzzleUUID, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	hist, _, attempts, status, firstAttemptTime, lastAttemptTime, err := GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(0))
	is.True(status == nil)
	is.Equal(hist.OriginalGcg, "")
	is.Equal(hist.IdAuth, "")
	is.Equal(hist.Uid, "")
	is.True(firstAttemptTime.Equal(time.Time{}))
	is.True(lastAttemptTime.Equal(time.Time{}))

	userIsCorrect, status, correctAnswer, gameId, _, _, attempts, newUserRatingInt, newPuzzleRatingInt, _, _, err := SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, &ipc.ClientGameplayEvent{}, false)
	is.NoErr(err)
	is.Equal(attempts, int32(1))
	is.True(!userIsCorrect)
	is.True(status == nil)
	is.True(correctAnswer == nil)
	is.True(newUserRatingInt != int32(0))
	is.True(newPuzzleRatingInt != int32(0))
	is.Equal(gameId, "")
	is.Equal(attempts, int32(1))

	_, _, attempts, status, firstAttemptTime, lastAttemptTime, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(1))
	is.True(status == nil)
	is.True(!firstAttemptTime.Equal(time.Time{}))
	is.True(!lastAttemptTime.Equal(time.Time{}))
	is.True(firstAttemptTime.Equal(lastAttemptTime))

	_, err = GetPreviousPuzzleId(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.Equal(err.Error(), entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_ATTEMPT_NOT_FOUND, PuzzlerUUID, puzzleUUID).Error())

	correctAnswer, _, _, _, _, newPuzzleRating, err := ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)
	is.True(correctAnswer != nil)

	newUserRating, err := getUserRating(ctx, pool, PuzzlerUUID, rk)
	is.NoErr(err)

	// User rating should go down, puzzle rating should go up
	is.True(float64(glicko.InitialRating) < newPuzzleRating.Rating)
	is.True(float64(glicko.InitialRatingDeviation) > newPuzzleRating.RatingDeviation)
	is.True(float64(glicko.InitialRating) > newUserRating.Rating)
	is.True(float64(glicko.InitialRatingDeviation) > newUserRating.RatingDeviation)

	attempts, recordedCorrect, err := getPuzzleAttempt(ctx, pool, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(1))
	is.True(!recordedCorrect.Valid)

	oldPuzzleRating := newPuzzleRating
	oldUserRating := newUserRating
	correctCGE := gameEventToClientGameplayEvent(correctAnswer)

	// Path 7
	// Submit the correct answer for the same puzzle,
	userIsCorrect, status, correctAnswer, gameId, _, _, attempts, newUserRatingInt, newPuzzleRatingInt, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, correctCGE, false)
	is.NoErr(err)
	is.True(correctAnswer != nil)
	is.Equal(newUserRatingInt, int32(0))
	is.Equal(newPuzzleRatingInt, int32(0))
	is.True(gameId != "")

	_, _, _, _, _, newPuzzleRating, err = ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)
	newUserRating, err = getUserRating(ctx, pool, PuzzlerUUID, rk)
	is.NoErr(err)

	// rating should remain unchanged and another attempt should be recorded
	is.True(userIsCorrect)
	is.True(*status)
	is.Equal(attempts, int32(2))
	is.True(common.WithinEpsilon(oldPuzzleRating.Rating, newPuzzleRating.Rating))
	is.True(common.WithinEpsilon(oldPuzzleRating.RatingDeviation, newPuzzleRating.RatingDeviation))
	is.True(common.WithinEpsilon(oldUserRating.Rating, newUserRating.Rating))
	is.True(common.WithinEpsilon(oldUserRating.RatingDeviation, newUserRating.RatingDeviation))
	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, pool, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(2))
	is.True(recordedCorrect.Valid)
	is.True(recordedCorrect.Bool)

	// Path 4
	// Submit another answer which should not change the puzzle attempt record
	userIsCorrect, status, correctAnswer, gameId, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, correctCGE, false)
	is.NoErr(err)
	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, pool, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(userIsCorrect)
	is.True(*status)
	is.True(answersAreEqual(correctCGE, correctAnswer))
	is.True(gameId != "")
	is.Equal(attempts, int32(2))
	is.True(recordedCorrect.Valid)
	is.True(recordedCorrect.Bool)

	oldPuzzleRating = newPuzzleRating
	oldUserRating = newUserRating

	// Path 3
	// Submit a correct answer
	puzzleUUID, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, _, attempts, status, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(status == nil)
	is.Equal(attempts, int32(0))

	correctAnswer, _, _, _, _, oldPuzzleRating, err = ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)

	oldUserRating, err = getUserRating(ctx, pool, PuzzlerUUID, rk)
	is.NoErr(err)

	correctCGE = gameEventToClientGameplayEvent(correctAnswer)

	userIsCorrect, status, _, _, _, _, attempts, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, correctCGE, false)
	is.NoErr(err)

	_, _, _, _, _, newPuzzleRating, err = ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)

	newUserRating, err = getUserRating(ctx, pool, PuzzlerUUID, rk)
	is.NoErr(err)

	// User rating should go up, puzzle rating should go down
	is.True(userIsCorrect)
	is.True(*status)
	is.Equal(attempts, int32(1))

	is.True(oldPuzzleRating.Rating > newPuzzleRating.Rating)
	is.True(oldPuzzleRating.RatingDeviation > newPuzzleRating.RatingDeviation)
	is.True(oldUserRating.Rating < newUserRating.Rating)
	is.True(oldUserRating.RatingDeviation > newUserRating.RatingDeviation)

	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, pool, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(1))
	is.True(recordedCorrect.Valid)
	is.True(recordedCorrect.Bool)

	// Check that the rating transaction rolls back correctly
	puzzleUUID, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	correctAnswer, _, _, _, _, oldPuzzleRating, err = ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)
	oldUserRating, err = getUserRating(ctx, pool, PuzzlerUUID, rk)
	is.NoErr(err)

	correctCGE = gameEventToClientGameplayEvent(correctAnswer)
	_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, "incorrect uuid", puzzleUUID, correctCGE, false)
	is.Equal(err.Error(), fmt.Sprintf("cannot get id from uuid %s: no rows for table %s", "incorrect uuid", "users"))

	_, _, _, _, _, newPuzzleRating, err = ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)
	newUserRating, err = getUserRating(ctx, pool, PuzzlerUUID, rk)
	is.NoErr(err)

	is.True(common.WithinEpsilon(oldPuzzleRating.Rating, newPuzzleRating.Rating))
	is.True(common.WithinEpsilon(oldPuzzleRating.RatingDeviation, newPuzzleRating.RatingDeviation))
	is.True(common.WithinEpsilon(oldUserRating.Rating, newUserRating.Rating))
	is.True(common.WithinEpsilon(oldUserRating.RatingDeviation, newUserRating.RatingDeviation))

	// Attempting to submit an answer to a puzzle for which the user has
	// not triggered the GetPuzzle endpoint should fail, since the
	// GetPuzzle endpoint will create the attempt in the db.
	_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, correctCGE, false)
	is.Equal(err.Error(), entity.NewWooglesError(ipc.WooglesError_PUZZLE_SUBMIT_ANSWER_PUZZLE_ATTEMPT_NOT_FOUND, PuzzlerUUID, puzzleUUID).Error())

	rated, attemptExists, attempts, _, _, _, err := ps.GetAttempts(ctx, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(!attemptExists)
	is.Equal(attempts, int32(0))
	is.True(!rated)

	// This should create the attempt record
	_, _, attempts, status, _, lastAttemptTime, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(status == nil)
	is.Equal(attempts, int32(0))

	rated, attemptExists, attempts, _, _, _, err = ps.GetAttempts(ctx, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(attemptExists)
	is.Equal(attempts, int32(0))
	is.True(!rated)

	// Answer should be unavailable
	_, err = GetPuzzleAnswer(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.Equal(err.Error(), entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_ANSWER_NOT_YET_RATED, PuzzlerUUID, puzzleUUID).Error())

	// This should update the attempt record
	_, _, attempts, status, _, newLastAttemptTime, err := GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(status == nil)
	is.Equal(attempts, int32(0))
	is.True(newLastAttemptTime.After(lastAttemptTime))

	// If the user has already gotten the puzzle correct, subsequent
	// submissions should not affect the status or number of attempts.
	userIsCorrect, status, correctAnswer, gameId, _, _, attempts, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, correctCGE, false)
	is.NoErr(err)
	is.True(userIsCorrect)
	is.True(*status)
	is.True(correctAnswer != nil)
	is.True(gameId != "")
	is.Equal(attempts, int32(1))

	// Puzzle should be rated now
	rated, attemptExists, attempts, _, _, _, err = ps.GetAttempts(ctx, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(attemptExists)
	is.Equal(attempts, int32(1))
	is.True(rated)

	// Answer should be available
	correctCGE = gameEventToClientGameplayEvent(correctAnswer)
	answerFromGet, err := GetPuzzleAnswer(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(answerFromGet != nil)
	is.True(answersAreEqual(correctCGE, answerFromGet))

	// The status should be the same for an incorrect answer
	correctAnswer.PlayedTiles += "Z"
	userIsCorrect, status, correctAnswer, gameId, _, _, attempts, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, gameEventToClientGameplayEvent(correctAnswer), false)
	is.NoErr(err)
	is.True(!userIsCorrect)
	is.True(*status)
	is.True(correctAnswer == nil)
	is.True(gameId == "")
	is.Equal(attempts, int32(1))

	// Path 5 and 6
	// Submit incorrect answers and then give up
	puzzleUUID, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, _, attempts, status, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(status == nil)
	is.Equal(attempts, int32(0))

	userIsCorrect, status, correctAnswer, gameId, _, _, attempts, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, &ipc.ClientGameplayEvent{}, false)
	is.NoErr(err)
	is.True(!userIsCorrect)
	is.True(status == nil)
	is.True(correctAnswer == nil)
	is.True(gameId == "")
	is.Equal(attempts, int32(1))

	userIsCorrect, status, correctAnswer, gameId, _, _, attempts, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, &ipc.ClientGameplayEvent{}, false)
	is.NoErr(err)
	is.True(!userIsCorrect)
	is.True(status == nil)
	is.True(correctAnswer == nil)
	is.True(gameId == "")
	is.Equal(attempts, int32(2))

	correctAnswer, _, _, _, _, _, err = ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)

	// If the user has given up, the answer sent should not be considered
	// for recording the correctness of the attempt
	userIsCorrect, status, correctAnswer, gameId, _, _, attempts, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, gameEventToClientGameplayEvent(correctAnswer), true)
	is.NoErr(err)
	is.True(userIsCorrect)
	is.True(!*status)
	is.True(correctAnswer != nil)
	is.True(gameId != "")
	is.Equal(attempts, int32(2))

	// The result should be the same for an incorrect answer
	correctAnswer.PlayedTiles += "Z"
	userIsCorrect, status, correctAnswer, gameId, _, _, attempts, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, gameEventToClientGameplayEvent(correctAnswer), true)
	is.NoErr(err)
	is.True(!userIsCorrect)
	is.True(!*status)
	is.True(correctAnswer != nil)
	is.True(gameId != "")
	is.Equal(attempts, int32(2))

	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, pool, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(2))
	is.True(recordedCorrect.Valid)
	is.True(!recordedCorrect.Bool)

	// The response for getting a puzzle should be
	// different if the user is logged out
	hist, _, attempts, status, firstAttemptTime, lastAttemptTime, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(hist != nil)
	is.Equal(attempts, int32(2))
	is.True(!*status)
	is.True(!firstAttemptTime.Equal(time.Time{}))
	is.True(!lastAttemptTime.Equal(time.Time{}))
	is.True(lastAttemptTime.After(firstAttemptTime))

	hist, _, attempts, status, firstAttemptTime, lastAttemptTime, err = GetPuzzle(ctx, ps, "", puzzleUUID)
	is.NoErr(err)
	is.True(hist != nil)
	is.Equal(attempts, int32(0))
	is.True(status == nil)
	is.True(firstAttemptTime.Equal(time.Time{}))
	is.True(lastAttemptTime.Equal(time.Time{}))

	// Path 2
	// Give up immediately without submitting any answers
	puzzleUUID, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, _, attempts, status, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(status == nil)
	is.Equal(attempts, int32(0))

	userIsCorrect, status, correctAnswer, gameId, _, _, attempts, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, nil, true)
	is.NoErr(err)
	is.True(!userIsCorrect)
	is.True(!*status)
	is.True(correctAnswer != nil)
	is.True(gameId != "")
	is.Equal(attempts, int32(0))

	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, pool, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(0))
	is.True(recordedCorrect.Valid)
	is.True(!recordedCorrect.Bool)

	// The user should not see repeat puzzles until they
	// have answered all of them
	unseenPuzzles, err := getNumUnattemptedPuzzlesInLexicon(ctx, pool, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	for i := 0; i < unseenPuzzles; i++ {
		puzzleUUID, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
		is.NoErr(err)

		puzzleLexicon, err := getPuzzleLexicon(ctx, pool, puzzleUUID)
		is.NoErr(err)
		is.Equal(puzzleLexicon, common.DefaultGameReq.Lexicon)

		hist, _, attempts, _, _, _, err := GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
		is.NoErr(err)
		is.Equal(attempts, int32(0))

		puzzleDBID, err := transactGetDBIDFromUUID(ctx, pool, "puzzles", puzzleUUID)
		is.NoErr(err)

		var turnNumber int
		err = pool.QueryRow(ctx, `SELECT turn_number FROM puzzles WHERE id = $1`, puzzleDBID).Scan(&turnNumber)
		is.NoErr(err)

		rated, attemptExists, attempts, _, _, _, err := ps.GetAttempts(ctx, PuzzlerUUID, puzzleUUID)
		is.NoErr(err)
		is.True(attemptExists)
		is.True(!rated)
		is.Equal(attempts, int32(0))
		is.Equal(len(hist.Events), turnNumber)

		userIsCorrect, status, _, _, _, _, attempts, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, &ipc.ClientGameplayEvent{}, false)
		is.NoErr(err)
		is.True(!userIsCorrect)
		is.True(status == nil)
		is.Equal(attempts, int32(1))
	}

	// The user should only see puzzles for their requested lexicon
	// regardless of how many puzzles they request
	attemptedPuzzles, err := getNumAttemptedPuzzles(ctx, pool, PuzzlerUUID)
	is.NoErr(err)

	unattemptedPuzzles, err := getNumUnattemptedPuzzles(ctx, pool, PuzzlerUUID)
	is.NoErr(err)
	is.Equal(totalPuzzles, attemptedPuzzles+unattemptedPuzzles)

	for i := 0; i < totalPuzzles*10; i++ {
		puzzleUUID, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
		is.NoErr(err)
		_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, &ipc.ClientGameplayEvent{}, false)
		is.NoErr(err)
	}

	newAttemptedPuzzles, err := getNumAttemptedPuzzles(ctx, pool, PuzzlerUUID)
	is.NoErr(err)
	is.Equal(newAttemptedPuzzles, attemptedPuzzles)

	// Test voting system

	err = SetPuzzleVote(ctx, ps, PuzzlerUUID, puzzleUUID, 1)
	is.NoErr(err)

	pop, err := getPuzzlePopularity(ctx, pool, puzzleUUID)
	is.NoErr(err)
	is.Equal(pop, 1)

	err = SetPuzzleVote(ctx, ps, PuzzleCreatorUUID, puzzleUUID, -1)
	is.NoErr(err)

	pop, err = getPuzzlePopularity(ctx, pool, puzzleUUID)
	is.NoErr(err)
	is.Equal(pop, 0)

	err = SetPuzzleVote(ctx, ps, PuzzlerUUID, puzzleUUID, 0)
	is.NoErr(err)

	pop, err = getPuzzlePopularity(ctx, pool, puzzleUUID)
	is.NoErr(err)
	is.Equal(pop, -1)

}

func TestPuzzlesPrevious(t *testing.T) {
	is := is.New(t)
	dbc, _, _ := RecreateDB()
	defer func() {
		dbc.cleanup()
	}()
	ps := dbc.ps
	ctx := context.Background()

	// Ensure that getting the previous puzzle works
	// for attempted and unattempted puzzles

	puzzle1, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	_, err = GetPreviousPuzzleId(ctx, ps, PuzzlerUUID, puzzle1)
	is.Equal(err.Error(), entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_NO_ATTEMPTS, PuzzlerUUID, puzzle1).Error())
	_, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle1)
	is.NoErr(err)
	_, err = GetPreviousPuzzleId(ctx, ps, PuzzlerUUID, puzzle1)
	is.Equal(err.Error(), entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_ATTEMPT_NOT_FOUND, PuzzlerUUID, puzzle1).Error())
	_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzle1, &ipc.ClientGameplayEvent{}, true)
	is.NoErr(err)

	puzzle2, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	_, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle2)
	is.NoErr(err)
	_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzle2, &ipc.ClientGameplayEvent{}, false)
	is.NoErr(err)

	puzzle3, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	_, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle3)
	is.NoErr(err)

	puzzle4, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	_, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle4)
	is.NoErr(err)
	actualPreviousPuzzle, err := GetPreviousPuzzleId(ctx, ps, PuzzlerUUID, puzzle4)
	is.NoErr(err)
	is.Equal(puzzle3, actualPreviousPuzzle)
	_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzle4, &ipc.ClientGameplayEvent{}, false)
	is.NoErr(err)

	puzzle5, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	_, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle5)
	is.NoErr(err)

	// Have another user do a bunch of puzzles
	// This should not affect the previous puzzle
	// of the original user
	_, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzleCreatorUUID, puzzle5)
	is.NoErr(err)
	_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzleCreatorUUID, puzzle5, &ipc.ClientGameplayEvent{}, false)
	is.NoErr(err)
	for i := 0; i < 5; i++ {
		otherPuzzle, err := GetNextPuzzleId(ctx, ps, PuzzleCreatorUUID, common.DefaultGameReq.Lexicon)
		is.NoErr(err)
		_, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzleCreatorUUID, otherPuzzle)
		is.NoErr(err)
		_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzleCreatorUUID, otherPuzzle, &ipc.ClientGameplayEvent{}, false)
		is.NoErr(err)
	}

	actualPreviousPuzzle, err = GetPreviousPuzzleId(ctx, ps, PuzzlerUUID, puzzle5)
	is.NoErr(err)
	is.Equal(puzzle4, actualPreviousPuzzle)
}

func TestPuzzlesStart(t *testing.T) {
	is := is.New(t)
	dbc, _, _ := RecreateDB()
	defer func() {
		dbc.cleanup()
	}()
	ps := dbc.ps
	ctx := context.Background()

	// This should work for users who are not logged in
	_, err := GetStartPuzzleId(ctx, ps, "", common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, err = GetStartPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	puzzle1, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle1)
	is.NoErr(err)

	actualStartPuzzle, err := GetStartPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	is.Equal(puzzle1, actualStartPuzzle)

	_, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzleCreatorUUID, puzzle1)
	is.NoErr(err)

	// Other users doing puzzles should not affect the original user's start puzzle
	actualStartPuzzle, err = GetStartPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	is.Equal(puzzle1, actualStartPuzzle)

	// The user using a different lexicon should not affect
	// the start puzzle for that lexicon
	puzzle2, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, OtherLexicon)
	is.NoErr(err)

	_, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle2)
	is.NoErr(err)

	actualStartPuzzle, err = GetStartPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	is.Equal(puzzle1, actualStartPuzzle)

	_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzle1, &ipc.ClientGameplayEvent{}, true)
	is.NoErr(err)

	// Since the most recent puzzle was completed, the
	// start puzzle for the user should just be a random puzzle
	actualStartPuzzle, err = GetStartPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	is.True(puzzle1 != actualStartPuzzle)

}

func TestPuzzlesNextClosestRating(t *testing.T) {
	is := is.New(t)
	dbc, _, _ := RecreateDB()
	defer func() {
		dbc.cleanup()
	}()
	ctx := context.Background()

	playerRating := 10000.0

	dbc.us.SetRatings(ctx, PuzzlerUUID, PuzzleCreatorUUID, "CSW19.puzzle.corres", entity.SingleRating{
		Volatility:      glicko.InitialVolatility,
		Rating:          playerRating,
		RatingDeviation: 40.0,
	}, entity.SingleRating{
		Volatility:      glicko.InitialVolatility,
		Rating:          playerRating,
		RatingDeviation: 40.0,
	})

	puzzleRatingDiffs := []int{
		50,
		-49,
		100,
		200,
		-300,
		-250,
		-400,
		150,
	}

	// affectedIds := []int{}

	for _, puzzleRatingDiff := range puzzleRatingDiffs {
		result, err := dbc.pool.Exec(ctx, `UPDATE puzzles SET rating = jsonb_set(jsonb_set(rating, array['rd'], $2), array['r'], $1) WHERE id = (SELECT id FROM puzzles WHERE lexicon = $3 LIMIT 1)`,
			playerRating+float64(puzzleRatingDiff), 40.0, common.DefaultGameReq.Lexicon)
		is.NoErr(err)
		is.Equal(result.RowsAffected(), int64(1))
	}

	currentDiff := 0.0

	for range puzzleRatingDiffs {
		puzzleUUID, err := GetNextClosestRatingPuzzleId(ctx, dbc.ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
		is.NoErr(err)

		_, _, _, _, _, _, err = GetPuzzle(ctx, dbc.ps, PuzzlerUUID, puzzleUUID)
		is.NoErr(err)

		var puzzleRating float64
		var puzzleId int
		err = dbc.pool.QueryRow(ctx, `SELECT id, rating->'r' FROM puzzles WHERE uuid = $1`, puzzleUUID).Scan(&puzzleId, &puzzleRating)
		is.NoErr(err)
		thisDiff := math.Abs(playerRating - puzzleRating)
		is.True(thisDiff > currentDiff)
		currentDiff = thisDiff
	}
}

func TestPuzzlesVerticalPlays(t *testing.T) {
	is := is.New(t)
	dbc, _, _ := RecreateDB()
	defer func() {
		dbc.cleanup()
	}()
	pool, ps := dbc.pool, dbc.ps
	ctx := context.Background()

	correct, err := transposedPlayByAnswerIsCorrect(ctx, pool, ps, "ZINNIA", PuzzlerUUID)
	is.NoErr(err)
	is.True(correct)

	correct, err = transposedPlayByAnswerIsCorrect(ctx, pool, ps, "QUANT", PuzzlerUUID)
	is.NoErr(err)
	is.True(correct)

	correct, err = transposedPlayByAnswerIsCorrect(ctx, pool, ps, "LINUX", PuzzlerUUID)
	is.NoErr(err)
	is.True(correct)

	correct, err = transposedPlayByAnswerIsCorrect(ctx, pool, ps, "ALI.....", PuzzlerUUID)
	is.NoErr(err)
	is.True(!correct)

	correct, err = transposedPlayByAnswerIsCorrect(ctx, pool, ps, "AD......", PuzzlerUUID)
	is.NoErr(err)
	is.True(!correct)

	correct, err = transposedPlayByAnswerIsCorrect(ctx, pool, ps, "ERI.OID", PuzzlerUUID)
	is.NoErr(err)
	is.True(!correct)
}

func TestUniqueSingleTileKey(t *testing.T) {
	is := is.New(t)
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "Q.", Direction: pb.GameEvent_HORIZONTAL}),
		uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "Q.", Direction: pb.GameEvent_VERTICAL}))
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: ".R", Direction: pb.GameEvent_HORIZONTAL}),
		uniqueSingleTileKey(&pb.GameEvent{Row: 7, Column: 11, PlayedTiles: ".R", Direction: pb.GameEvent_VERTICAL}))
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "B....", Direction: pb.GameEvent_HORIZONTAL}),
		uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "B....", Direction: pb.GameEvent_VERTICAL}))
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 9, Column: 3, PlayedTiles: "....X", Direction: pb.GameEvent_HORIZONTAL}),
		uniqueSingleTileKey(&pb.GameEvent{Row: 5, Column: 7, PlayedTiles: "....X", Direction: pb.GameEvent_VERTICAL}))
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 11, Column: 9, PlayedTiles: "..A...", Direction: pb.GameEvent_HORIZONTAL}),
		uniqueSingleTileKey(&pb.GameEvent{Row: 7, Column: 11, PlayedTiles: "....A..", Direction: pb.GameEvent_VERTICAL}))
	is.True(uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "A.", Direction: pb.GameEvent_HORIZONTAL}) !=
		uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "Q.", Direction: pb.GameEvent_VERTICAL}))
}

func RecreateDB() (*DBController, int, int) {
	cfg := &config.Config{}
	cfg.MacondoConfig = common.DefaultConfig
	cfg.DBConnUri = commondb.TestingPostgresConnUri()
	cfg.DBConnDSN = commondb.TestingPostgresConnDSN()
	cfg.MacondoConfig.DefaultLexicon = common.DefaultLexicon
	zerolog.SetGlobalLevel(zerolog.Disabled)
	ctx := context.Background()
	log.Info().Msg("here first")
	// Recreate the test database
	err := commondb.RecreateTestDB()
	if err != nil {
		panic(err)
	}

	// Reconnect to the new test database
	pool, err := commondb.OpenTestingDB()
	if err != nil {
		panic(err)
	}

	userStore, err := user.NewDBStore(commondb.TestingPostgresConnDSN())
	if err != nil {
		panic(err)
	}
	err = userStore.New(context.Background(), &entity.User{Username: "Puzzler", Email: "puzzler@woogles.io", UUID: PuzzlerUUID})
	if err != nil {
		panic(err)
	}

	err = userStore.New(context.Background(), &entity.User{Username: "Kenji", Email: "kenji@woogles.io", UUID: PuzzleCreatorUUID})
	if err != nil {
		panic(err)
	}

	tempGameStore, err := gamestore.NewDBStore(cfg, userStore)
	if err != nil {
		panic(err)
	}

	gameStore := gamestore.NewCache(tempGameStore)
	if err != nil {
		panic(err)
	}

	puzzlesStore, err := puzzlesstore.NewDBStore(pool)
	if err != nil {
		panic(err)
	}

	pgrjReq := proto.Clone(&puzzle_service.PuzzleGenerationJobRequest{
		BotVsBot:               true,
		Lexicon:                "CSW21",
		LetterDistribution:     "english",
		SqlOffset:              0,
		GameConsiderationLimit: 1000000,
		GameCreationLimit:      100000,
		Request: &pb.PuzzleGenerationRequest{
			Buckets: []*pb.PuzzleBucket{
				{
					Size:     50000,
					Includes: []pb.PuzzleTag{pb.PuzzleTag_EQUITY},
					Excludes: []pb.PuzzleTag{},
				},
			},
		},
	}).(*puzzle_service.PuzzleGenerationJobRequest)

	reqId, err := puzzlesStore.CreateGenerationLog(ctx, pgrjReq)
	if err != nil {
		panic(err)
	}

	files, err := ioutil.ReadDir("./testdata")
	if err != nil {
		panic(err)
	}

	rules, err := game.NewBasicGameRules(&common.DefaultConfig, common.DefaultLexicon, board.CrosswordGameLayout, "english", game.CrossScoreAndSet, game.VarClassic)
	if err != nil {
		panic(err)
	}

	authoredPuzzles := 0
	totalPuzzles := 0
	for idx, f := range files {
		gameHistory, err := gcgio.ParseGCG(&common.DefaultConfig, fmt.Sprintf("./testdata/%s", f.Name()))
		if err != nil {
			panic(err)
		}
		// Set the correct challenge rule to allow games with
		// lost challenges.
		gameHistory.ChallengeRule = pb.ChallengeRule_FIVE_POINT
		game, err := game.NewFromHistory(gameHistory, rules, 0)
		if err != nil {
			panic(err)
		}
		gameReq := proto.Clone(common.DefaultGameReq).(*ipc.GameRequest)

		pcUUID := ""
		if idx%2 == 1 {
			pcUUID = PuzzleCreatorUUID
			gameReq.Lexicon = OtherLexicon
		}
		entGame := entity.NewGame(game, gameReq)
		pzls, err := CreatePuzzlesFromGame(ctx, pgrjReq.Request, reqId, gameStore, puzzlesStore, entGame, pcUUID, ipc.GameType_ANNOTATED)
		if err != nil {
			panic(err)
		}
		if idx%2 == 1 {
			authoredPuzzles += len(pzls)
		}
		totalPuzzles += len(pzls)
	}

	return &DBController{
		pool: pool, ps: puzzlesStore, us: userStore, gs: gameStore,
	}, authoredPuzzles, totalPuzzles
}

func getUserRating(ctx context.Context, pool *pgxpool.Pool, userUUID string, rk entity.VariantKey) (*entity.SingleRating, error) {
	id, err := transactGetDBIDFromUUID(ctx, pool, "users", userUUID)
	if err != nil {
		return nil, err
	}
	tx, err := pool.BeginTx(ctx, commondb.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	initialRating := &entity.SingleRating{
		Rating:            float64(glicko.InitialRating),
		RatingDeviation:   float64(glicko.InitialRatingDeviation),
		Volatility:        glicko.InitialVolatility,
		LastGameTimestamp: time.Now().Unix()}

	userRating, err := commondb.GetUserRating(ctx, tx, id, rk, initialRating)

	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return userRating, nil
}

func transposedPlayByAnswerIsCorrect(ctx context.Context, pool *pgxpool.Pool, ps *puzzlesstore.DBStore, playedTiles string, userUUID string) (bool, error) {
	puzzleUUID, err := getPuzzleUUIDByAnswer(ctx, pool, playedTiles)
	if err != nil {
		return false, err
	}
	return transposedPlayIsCorrect(ctx, pool, ps, puzzleUUID, userUUID)
}

func transposedPlayIsCorrect(ctx context.Context, pool *pgxpool.Pool, ps *puzzlesstore.DBStore, puzzleUUID string, userUUID string) (bool, error) {
	correctAnswer, _, _, _, _, _, err := ps.GetAnswer(ctx, puzzleUUID)
	if err != nil {
		return false, err
	}

	correctCGE := gameEventToClientGameplayEvent(correctAnswer)

	// Flip the coordinates
	correctCGE.PositionCoords = move.ToBoardGameCoords(
		int(correctAnswer.Column),
		int(correctAnswer.Row),
		correctAnswer.Direction == pb.GameEvent_HORIZONTAL)

	_, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	if err != nil {
		return false, err
	}

	userIsCorrect, _, _, _, _, _, _, _, _, _, _, err := SubmitAnswer(ctx, ps, userUUID, puzzleUUID, correctCGE, false)
	if err != nil {
		return false, err
	}
	return userIsCorrect, nil
}

func getPuzzleUUIDByAnswer(ctx context.Context, pool *pgxpool.Pool, playedTiles string) (string, error) {
	var puzzleUUID string
	err := pool.QueryRow(ctx, `SELECT uuid FROM puzzles WHERE answer->>'PlayedTiles' = $1`, playedTiles).Scan(&puzzleUUID)
	return puzzleUUID, err
}

func getPuzzlePopularity(ctx context.Context, pool *pgxpool.Pool, puzzleUUID string) (int, error) {
	pid, err := transactGetDBIDFromUUID(ctx, pool, "puzzles", puzzleUUID)
	if err != nil {
		return 0, err
	}
	var popularity int
	err = pool.QueryRow(ctx, `SELECT SUM(vote) FROM puzzle_votes WHERE puzzle_id = $1`, pid).Scan(&popularity)
	return popularity, err
}

func getPuzzleAttempt(ctx context.Context, pool *pgxpool.Pool, userUUID string, puzzleUUID string) (int32, *sql.NullBool, error) {
	pid, err := transactGetDBIDFromUUID(ctx, pool, "puzzles", puzzleUUID)
	if err != nil {
		return 0, nil, err
	}

	uid, err := transactGetDBIDFromUUID(ctx, pool, "users", userUUID)
	if err != nil {
		return 0, nil, err
	}
	var attempts int32
	correct := &sql.NullBool{}
	err = pool.QueryRow(ctx, `SELECT attempts, correct FROM puzzle_attempts WHERE user_id = $1 AND puzzle_id = $2`, uid, pid).Scan(&attempts, correct)
	if err != nil {
		return 0, nil, err
	}
	return attempts, correct, nil
}

func getNumUnattemptedPuzzles(ctx context.Context, pool *pgxpool.Pool, userUUID string) (int, error) {
	uid, err := transactGetDBIDFromUUID(ctx, pool, "users", userUUID)
	if err != nil {
		return -1, err
	}
	var unseen int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM puzzles WHERE puzzles.id NOT IN (SELECT puzzle_id FROM puzzle_attempts WHERE user_id = $1)`, uid).Scan(&unseen)
	if err != nil {
		return -1, err
	}
	return unseen, nil
}

func getNumAttemptedPuzzles(ctx context.Context, pool *pgxpool.Pool, userUUID string) (int, error) {
	uid, err := transactGetDBIDFromUUID(ctx, pool, "users", userUUID)
	if err != nil {
		return -1, err
	}
	var attempted int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM puzzle_attempts WHERE user_id = $1`, uid).Scan(&attempted)
	if err != nil {
		return -1, err
	}
	return attempted, nil
}

func getNumUnattemptedPuzzlesInLexicon(ctx context.Context, pool *pgxpool.Pool, userUUID string, lexicon string) (int, error) {
	uid, err := transactGetDBIDFromUUID(ctx, pool, "users", userUUID)
	if err != nil {
		return -1, err
	}
	var unseen int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM puzzles WHERE lexicon = $1 AND puzzles.id NOT IN (SELECT puzzle_id FROM puzzle_attempts WHERE user_id = $2)`, lexicon, uid).Scan(&unseen)
	if err != nil {
		return -1, err
	}
	return unseen, nil
}

func getPuzzleLexicon(ctx context.Context, pool *pgxpool.Pool, puzzleUUID string) (string, error) {
	var lexicon string
	err := pool.QueryRow(ctx, `SELECT lexicon FROM puzzles WHERE uuid = $1`, puzzleUUID).Scan(&lexicon)
	if err != nil {
		return "", err
	}
	return lexicon, nil
}

func transactGetDBIDFromUUID(ctx context.Context, pool *pgxpool.Pool, table string, uuid string) (int64, error) {
	tx, err := pool.BeginTx(ctx, commondb.DefaultTxOptions)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var id int64
	if table == "users" {
		id, err = commondb.GetUserDBIDFromUUID(ctx, tx, uuid)
	} else if table == "games" {
		id, err = commondb.GetGameDBIDFromUUID(ctx, tx, uuid)
	} else if table == "puzzles" {
		id, err = commondb.GetPuzzleDBIDFromUUID(ctx, tx, uuid)
	} else {
		return 0, fmt.Errorf("unknown table: %s", table)
	}
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return id, nil
}
