package puzzles

import (
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"testing"
	"time"

	"github.com/domino14/macondo/board"
	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/macondo/game"
	"github.com/domino14/macondo/gcgio"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/macondo/move"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lithammer/shortuuid/v4"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/common"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/glicko"
	commondb "github.com/woogles-io/liwords/pkg/stores/common"
	gamestore "github.com/woogles-io/liwords/pkg/stores/game"
	puzzlesstore "github.com/woogles-io/liwords/pkg/stores/puzzles"
	"github.com/woogles-io/liwords/pkg/stores/user"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"github.com/woogles-io/liwords/rpc/api/proto/puzzle_service"
)

var pkg = "puzzles"

const PuzzlerUUID = "puzzler"
const PuzzleCreatorUUID = "kenji"
const OtherLexicon = "CSW19"

var DefaultConfig = config.DefaultConfig()

func ctxForTests() context.Context {
	ctx := context.Background()
	ctx = log.Logger.WithContext(ctx)
	ctx = DefaultConfig.WithContext(ctx)
	return ctx
}

func gameEventToClientGameplayEvent(evt *pb.GameEvent) *ipc.ClientGameplayEvent {
	cge := &ipc.ClientGameplayEvent{}
	// use hard-coded english alphabet for this test
	eng, err := tilemapping.GetDistribution(DefaultConfig.WGLConfig(), "english")
	if err != nil {
		panic(err)
	}
	engTM := eng.TileMapping()
	switch evt.Type {
	case pb.GameEvent_TILE_PLACEMENT_MOVE:
		cge.Type = ipc.ClientGameplayEvent_TILE_PLACEMENT
		bts, err := tilemapping.ToMachineWord(evt.PlayedTiles, engTM)
		if err != nil {
			panic(err)
		}
		cge.MachineLetters = bts.ToByteArr()
		cge.PositionCoords = move.ToBoardGameCoords(int(evt.Row), int(evt.Column),
			evt.Direction == pb.GameEvent_VERTICAL)

	case pb.GameEvent_EXCHANGE:
		cge.Type = ipc.ClientGameplayEvent_EXCHANGE
		bts, err := tilemapping.ToMachineWord(evt.Exchanged, engTM)
		if err != nil {
			panic(err)
		}
		cge.MachineLetters = bts.ToByteArr()
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

	ctx := ctxForTests()
	engDist, err := tilemapping.GetDistribution(DefaultConfig.WGLConfig(), "english")
	is.NoErr(err)
	rk := entity.LexiconToPuzzleVariantKey(common.DefaultGameReq.Lexicon)

	pcid, err := commondb.GetDBIDFromUUID(ctx, pool, &commondb.CommonDBConfig{
		TableType: commondb.UsersTable,
		Value:     PuzzleCreatorUUID,
	})
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
	_, _, err = GetNextPuzzleId(ctx, ps, "", common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	// Path 1
	// Submit an incorrect answer
	puzzleUUID, _, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	hist, _, attempts, status, firstAttemptTime, lastAttemptTime, newPuzzleRating, newUserRating, err := GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(0))
	is.True(status == nil)
	is.Equal(hist.OriginalGcg, "")
	is.Equal(hist.IdAuth, "")
	is.Equal(hist.Uid, "")
	is.True(firstAttemptTime.Equal(time.Time{}))
	is.True(lastAttemptTime.Equal(time.Time{}))
	is.True(newPuzzleRating != nil)
	is.True(newUserRating != nil)
	is.True(common.WithinEpsilon(newPuzzleRating.Rating, commondb.InitialRating.Rating))
	is.True(common.WithinEpsilon(newUserRating.Rating, commondb.InitialRating.Rating))

	// Reloading should return the same values except for attempt times
	hist, _, attempts, status, firstAttemptTime, lastAttemptTime, newPuzzleRating, newUserRating, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(0))
	is.True(status == nil)
	is.Equal(hist.OriginalGcg, "")
	is.Equal(hist.IdAuth, "")
	is.Equal(hist.Uid, "")
	is.True(!firstAttemptTime.Equal(time.Time{}))
	is.True(!lastAttemptTime.Equal(time.Time{}))
	is.True(newPuzzleRating != nil)
	is.True(newUserRating != nil)
	is.True(common.WithinEpsilon(newPuzzleRating.Rating, commondb.InitialRating.Rating))
	is.True(common.WithinEpsilon(newUserRating.Rating, commondb.InitialRating.Rating))

	userIsCorrect, status, correctAnswer, gameId, _, _, attempts, _, _, newPuzzleRating, newUserRating, err := SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, &ipc.ClientGameplayEvent{}, false)
	is.NoErr(err)
	is.Equal(attempts, int32(1))
	is.True(!userIsCorrect)
	is.True(status == nil)
	is.True(correctAnswer == nil)
	is.True(newUserRating.Rating != 0)
	is.True(newPuzzleRating.Rating != 0)
	is.Equal(gameId, "")
	is.Equal(attempts, int32(1))

	_, _, attempts, status, firstAttemptTime, lastAttemptTime, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(1))
	is.True(status == nil)
	is.True(!firstAttemptTime.Equal(time.Time{}))
	is.True(!lastAttemptTime.Equal(time.Time{}))
	is.True(firstAttemptTime.Equal(lastAttemptTime))

	_, err = GetPreviousPuzzleId(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.Equal(err.Error(), entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_ATTEMPT_NOT_FOUND, PuzzlerUUID, puzzleUUID).Error())

	correctAnswer, _, _, _, _, newPuzzleRating, err = ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)
	is.True(correctAnswer != nil)

	newUserRating, err = getUserRating(ctx, pool, PuzzlerUUID, rk)
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
	userIsCorrect, status, correctAnswer, gameId, _, _, attempts, _, _, newPuzzleRating, newUserRating, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, correctCGE, false)
	is.NoErr(err)
	is.True(correctAnswer != nil)
	is.True(newUserRating.Rating != 0)
	is.True(newPuzzleRating.Rating != 0)
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
	userIsCorrect, status, correctAnswer, gameId, _, _, _, _, _, newPuzzleRating, newUserRating, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, correctCGE, false)
	is.NoErr(err)
	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, pool, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(userIsCorrect)
	is.True(*status)
	is.True(answersAreEqual(correctCGE, correctAnswer, engDist))
	is.True(gameId != "")
	is.Equal(attempts, int32(2))
	is.True(newUserRating.Rating != 0)
	is.True(newPuzzleRating.Rating != 0)
	is.True(recordedCorrect.Valid)
	is.True(recordedCorrect.Bool)

	oldPuzzleRating = newPuzzleRating
	oldUserRating = newUserRating

	// Path 3
	// Submit a correct answer
	puzzleUUID, _, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, _, attempts, status, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
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
	puzzleUUID, _, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
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

	rated, attemptExists, attempts, _, _, _, _, _, err := ps.GetAttempts(ctx, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(!attemptExists)
	is.Equal(attempts, int32(0))
	is.True(!rated)

	// This should create the attempt record
	_, _, attempts, status, _, lastAttemptTime, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(status == nil)
	is.Equal(attempts, int32(0))

	rated, attemptExists, attempts, _, _, _, _, _, err = ps.GetAttempts(ctx, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(attemptExists)
	is.Equal(attempts, int32(0))
	is.True(!rated)

	// Answer should be unavailable
	_, err = GetPuzzleAnswer(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.Equal(err.Error(), entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_ANSWER_NOT_YET_RATED, PuzzlerUUID, puzzleUUID).Error())

	// This should update the attempt record
	_, _, attempts, status, _, newLastAttemptTime, _, _, err := GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(status == nil)
	is.Equal(attempts, int32(0))
	is.True(newLastAttemptTime.After(lastAttemptTime))

	// If the user has already gotten the puzzle correct, subsequent
	// submissions should not affect the status or number of attempts.
	userIsCorrect, status, correctAnswer, gameId, _, _, attempts, _, _, newPuzzleRating, newUserRating, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, correctCGE, false)
	is.NoErr(err)
	is.True(userIsCorrect)
	is.True(*status)
	is.True(correctAnswer != nil)
	is.True(newUserRating.Rating != 0)
	is.True(newPuzzleRating.Rating != 0)
	is.True(gameId != "")
	is.Equal(attempts, int32(1))

	// Puzzle should be rated now
	rated, attemptExists, attempts, _, _, _, _, _, err = ps.GetAttempts(ctx, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(attemptExists)
	is.Equal(attempts, int32(1))
	is.True(rated)

	// Answer should be available
	correctCGE = gameEventToClientGameplayEvent(correctAnswer)
	answerFromGet, err := GetPuzzleAnswer(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(answerFromGet != nil)
	is.True(answersAreEqual(correctCGE, answerFromGet, engDist))

	// The status should be the same for an incorrect answer
	correctAnswer.Type = pb.GameEvent_EXCHANGE
	userIsCorrect, status, correctAnswer, gameId, _, _, attempts, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzleUUID, gameEventToClientGameplayEvent(correctAnswer), false)
	is.NoErr(err)
	is.True(!userIsCorrect)
	is.True(*status)
	is.True(correctAnswer == nil)
	is.True(gameId == "")
	is.Equal(attempts, int32(1))

	// Path 5 and 6
	// Submit incorrect answers and then give up
	puzzleUUID, _, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, _, attempts, status, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
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
	hist, _, attempts, status, firstAttemptTime, lastAttemptTime, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(hist != nil)
	is.Equal(attempts, int32(2))
	is.True(!*status)
	is.True(!firstAttemptTime.Equal(time.Time{}))
	is.True(!lastAttemptTime.Equal(time.Time{}))
	is.True(lastAttemptTime.After(firstAttemptTime))

	hist, _, attempts, status, firstAttemptTime, lastAttemptTime, _, _, err = GetPuzzle(ctx, ps, "", puzzleUUID)
	is.NoErr(err)
	is.True(hist != nil)
	is.Equal(attempts, int32(0))
	is.True(status == nil)
	is.True(firstAttemptTime.Equal(time.Time{}))
	is.True(lastAttemptTime.Equal(time.Time{}))

	// Path 2
	// Give up immediately without submitting any answers
	puzzleUUID, _, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, _, attempts, status, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
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
		puzzleUUID, _, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
		is.NoErr(err)

		puzzleLexicon, err := getPuzzleLexicon(ctx, pool, puzzleUUID)
		is.NoErr(err)
		is.Equal(puzzleLexicon, common.DefaultGameReq.Lexicon)

		hist, _, attempts, _, _, _, _, _, err := GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
		is.NoErr(err)
		is.Equal(attempts, int32(0))

		puzzleDBID, err := commondb.GetDBIDFromUUID(ctx, pool, &commondb.CommonDBConfig{
			TableType: commondb.PuzzlesTable,
			Value:     puzzleUUID,
		})
		is.NoErr(err)

		var turnNumber int
		err = pool.QueryRow(ctx, `SELECT turn_number FROM puzzles WHERE id = $1`, puzzleDBID).Scan(&turnNumber)
		is.NoErr(err)

		rated, attemptExists, attempts, _, _, _, _, _, err := ps.GetAttempts(ctx, PuzzlerUUID, puzzleUUID)
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

	for i := 0; i < totalPuzzles*3; i++ {
		puzzleUUID, _, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
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
	ctx := ctxForTests()

	// Ensure that getting the previous puzzle works
	// for attempted and unattempted puzzles

	puzzle1, _, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	_, err = GetPreviousPuzzleId(ctx, ps, PuzzlerUUID, puzzle1)
	is.Equal(err.Error(), entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_NO_ATTEMPTS, PuzzlerUUID, puzzle1).Error())
	_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle1)
	is.NoErr(err)
	_, err = GetPreviousPuzzleId(ctx, ps, PuzzlerUUID, puzzle1)
	is.Equal(err.Error(), entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_ATTEMPT_NOT_FOUND, PuzzlerUUID, puzzle1).Error())
	_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzle1, &ipc.ClientGameplayEvent{}, true)
	is.NoErr(err)

	puzzle2, _, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle2)
	is.NoErr(err)
	_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzle2, &ipc.ClientGameplayEvent{}, false)
	is.NoErr(err)

	puzzle3, _, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle3)
	is.NoErr(err)

	puzzle4, _, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle4)
	is.NoErr(err)
	actualPreviousPuzzle, err := GetPreviousPuzzleId(ctx, ps, PuzzlerUUID, puzzle4)
	is.NoErr(err)
	is.Equal(puzzle3, actualPreviousPuzzle)
	_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzle4, &ipc.ClientGameplayEvent{}, false)
	is.NoErr(err)

	puzzle5, _, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle5)
	is.NoErr(err)

	// Have another user do a bunch of puzzles
	// This should not affect the previous puzzle
	// of the original user
	_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzleCreatorUUID, puzzle5)
	is.NoErr(err)
	_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzleCreatorUUID, puzzle5, &ipc.ClientGameplayEvent{}, false)
	is.NoErr(err)
	for i := 0; i < 5; i++ {
		otherPuzzle, _, err := GetNextPuzzleId(ctx, ps, PuzzleCreatorUUID, common.DefaultGameReq.Lexicon)
		is.NoErr(err)
		_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzleCreatorUUID, otherPuzzle)
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
	ctx := ctxForTests()

	// This should work for users who are not logged in
	_, _, err := GetStartPuzzleId(ctx, ps, "", common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, _, err = GetStartPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	puzzle1, _, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle1)
	is.NoErr(err)

	actualStartPuzzle, _, err := GetStartPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	is.Equal(puzzle1, actualStartPuzzle)

	_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzleCreatorUUID, puzzle1)
	is.NoErr(err)

	// Other users doing puzzles should not affect the original user's start puzzle
	actualStartPuzzle, _, err = GetStartPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	is.Equal(puzzle1, actualStartPuzzle)

	// The user using a different lexicon should not affect
	// the start puzzle for that lexicon
	puzzle2, _, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, OtherLexicon)
	is.NoErr(err)

	_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle2)
	is.NoErr(err)

	actualStartPuzzle, _, err = GetStartPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	is.Equal(puzzle1, actualStartPuzzle)

	_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzle1, &ipc.ClientGameplayEvent{}, true)
	is.NoErr(err)

	// Since the most recent puzzle was completed, the
	// start puzzle for the user should just be a random puzzle
	actualStartPuzzle, _, err = GetStartPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	is.True(puzzle1 != actualStartPuzzle)

}

func TestPuzzlesNextClosestRating(t *testing.T) {
	is := is.New(t)
	dbc, _, _ := RecreateDB()
	defer func() {
		dbc.cleanup()
	}()
	ctx := ctxForTests()

	playerRating := 10000.0

	dbc.us.SetRatings(ctx, PuzzlerUUID, PuzzleCreatorUUID, "CSW19.puzzle.corres", &entity.SingleRating{
		Volatility:      glicko.InitialVolatility,
		Rating:          playerRating,
		RatingDeviation: 40.0,
	}, &entity.SingleRating{
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
		puzzleUUID, _, err := GetNextClosestRatingPuzzleId(ctx, dbc.ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
		is.NoErr(err)

		_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, dbc.ps, PuzzlerUUID, puzzleUUID)
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

func TestPuzzlesQueryResult(t *testing.T) {
	is := is.New(t)
	dbc, _, _ := RecreateDB()
	defer func() {
		dbc.cleanup()
	}()
	ps := dbc.ps
	ctx := ctxForTests()

	totalPuzzles, err := getNumUnattemptedPuzzlesInLexicon(ctx, dbc.pool, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, pqr, err := GetStartPuzzleId(ctx, ps, "", common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	is.Equal(pqr, puzzle_service.PuzzleQueryResult_RANDOM)

	_, pqr, err = GetStartPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	is.Equal(pqr, puzzle_service.PuzzleQueryResult_UNSEEN)

	puzzle, _, err := GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle)
	is.NoErr(err)

	actualStartPuzzle, pqr, err := GetStartPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	is.Equal(puzzle, actualStartPuzzle)
	is.Equal(pqr, puzzle_service.PuzzleQueryResult_START)

	_, pqr, err = GetNextPuzzleId(ctx, ps, "", common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	is.Equal(pqr, puzzle_service.PuzzleQueryResult_RANDOM)

	_, pqr, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)
	is.Equal(pqr, puzzle_service.PuzzleQueryResult_UNSEEN)

	// See all of the puzzles
	for i := 0; i < totalPuzzles-1; i++ {
		puzzle, pqr, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
		is.NoErr(err)
		is.Equal(pqr, puzzle_service.PuzzleQueryResult_UNSEEN)
		_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle)
		is.NoErr(err)
	}

	// All of these puzzles should now be UNRATED
	for i := 0; i < totalPuzzles*2; i++ {
		puzzle, pqr, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
		is.NoErr(err)
		is.Equal(pqr, puzzle_service.PuzzleQueryResult_UNRATED)
		_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle)
		is.NoErr(err)
	}

	// Rate all of the puzzles
	for i := 0; i < totalPuzzles; i++ {
		puzzle, pqr, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
		is.NoErr(err)
		is.Equal(pqr, puzzle_service.PuzzleQueryResult_UNRATED)
		_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzle, &ipc.ClientGameplayEvent{}, false)
		is.NoErr(err)
	}

	// Finish all of the puzzles
	for i := 0; i < totalPuzzles; i++ {
		puzzle, pqr, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
		is.NoErr(err)
		is.Equal(pqr, puzzle_service.PuzzleQueryResult_UNFINISHED)
		_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzle, &ipc.ClientGameplayEvent{}, true)
		is.NoErr(err)
	}

	// All puzzles should be exhausted
	for i := 0; i < totalPuzzles; i++ {
		_, pqr, err = GetNextPuzzleId(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
		is.NoErr(err)
		is.Equal(pqr, puzzle_service.PuzzleQueryResult_EXHAUSTED)
	}

	// Repeat with GetClosest
	totalPuzzles, err = getNumUnattemptedPuzzlesInLexicon(ctx, dbc.pool, PuzzlerUUID, OtherLexicon)
	is.NoErr(err)

	for i := 0; i < totalPuzzles; i++ {
		puzzle, pqr, err = GetNextClosestRatingPuzzleId(ctx, ps, PuzzlerUUID, OtherLexicon)
		is.NoErr(err)
		is.Equal(pqr, puzzle_service.PuzzleQueryResult_UNSEEN)
		_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle)
		is.NoErr(err)
	}

	for i := 0; i < totalPuzzles*2; i++ {
		puzzle, pqr, err = GetNextClosestRatingPuzzleId(ctx, ps, PuzzlerUUID, OtherLexicon)
		is.NoErr(err)
		is.Equal(pqr, puzzle_service.PuzzleQueryResult_UNRATED)
		_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzle)
		is.NoErr(err)
	}

	for i := 0; i < totalPuzzles; i++ {
		puzzle, pqr, err = GetNextClosestRatingPuzzleId(ctx, ps, PuzzlerUUID, OtherLexicon)
		is.NoErr(err)
		is.Equal(pqr, puzzle_service.PuzzleQueryResult_UNRATED)
		_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzle, &ipc.ClientGameplayEvent{}, false)
		is.NoErr(err)
	}

	for i := 0; i < totalPuzzles; i++ {
		puzzle, pqr, err = GetNextClosestRatingPuzzleId(ctx, ps, PuzzlerUUID, OtherLexicon)
		is.NoErr(err)
		is.Equal(pqr, puzzle_service.PuzzleQueryResult_UNFINISHED)
		_, _, _, _, _, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, PuzzlerUUID, puzzle, &ipc.ClientGameplayEvent{}, true)
		is.NoErr(err)
	}

	for i := 0; i < totalPuzzles; i++ {
		_, pqr, err = GetNextClosestRatingPuzzleId(ctx, ps, PuzzlerUUID, OtherLexicon)
		is.NoErr(err)
		is.Equal(pqr, puzzle_service.PuzzleQueryResult_EXHAUSTED)
	}
}

func TestPuzzlesVerticalPlays(t *testing.T) {
	is := is.New(t)
	dbc, _, _ := RecreateDB()
	defer func() {
		dbc.cleanup()
	}()
	pool, ps := dbc.pool, dbc.ps
	ctx := ctxForTests()

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
	ld, err := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	is.NoErr(err)
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "Q.", Direction: pb.GameEvent_HORIZONTAL}, ld),
		uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "Q.", Direction: pb.GameEvent_VERTICAL}, ld))
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: ".R", Direction: pb.GameEvent_HORIZONTAL}, ld),
		uniqueSingleTileKey(&pb.GameEvent{Row: 7, Column: 11, PlayedTiles: ".R", Direction: pb.GameEvent_VERTICAL}, ld))
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "B....", Direction: pb.GameEvent_HORIZONTAL}, ld),
		uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "B....", Direction: pb.GameEvent_VERTICAL}, ld))
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 9, Column: 3, PlayedTiles: "....X", Direction: pb.GameEvent_HORIZONTAL}, ld),
		uniqueSingleTileKey(&pb.GameEvent{Row: 5, Column: 7, PlayedTiles: "....X", Direction: pb.GameEvent_VERTICAL}, ld))
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 11, Column: 9, PlayedTiles: "..A...", Direction: pb.GameEvent_HORIZONTAL}, ld),
		uniqueSingleTileKey(&pb.GameEvent{Row: 7, Column: 11, PlayedTiles: "....A..", Direction: pb.GameEvent_VERTICAL}, ld))
	is.True(uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "A.", Direction: pb.GameEvent_HORIZONTAL}, ld) !=
		uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "Q.", Direction: pb.GameEvent_VERTICAL}, ld))
}

func RecreateDB() (*DBController, int, int) {
	cfg := DefaultConfig
	cfg.DBConnUri = commondb.TestingPostgresConnUri(pkg)
	cfg.DBConnDSN = commondb.TestingPostgresConnDSN(pkg)
	cfg.MacondoConfig().Set(macondoconfig.ConfigDefaultLexicon, common.DefaultLexicon)
	zerolog.SetGlobalLevel(zerolog.Disabled)
	ctx := context.Background()
	log.Info().Msg("here first")
	// Recreate the test database
	err := commondb.RecreateTestDB(pkg)
	if err != nil {
		panic(err)
	}

	// Reconnect to the new test database
	pool, err := commondb.OpenTestingDB(pkg)
	if err != nil {
		panic(err)
	}

	userStore, err := user.NewDBStore(pool)
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

	rules, err := game.NewBasicGameRules(DefaultConfig.MacondoConfig(), common.DefaultLexicon, board.CrosswordGameLayout, "english", game.CrossScoreAndSet, game.VarClassic)
	if err != nil {
		panic(err)
	}

	authoredPuzzles := 0
	totalPuzzles := 0
	for idx, f := range files {
		gameHistory, err := gcgio.ParseGCG(DefaultConfig.MacondoConfig(), fmt.Sprintf("./testdata/%s", f.Name()))
		if err != nil {
			panic(err)
		}
		// Set the correct challenge rule to allow games with
		// lost challenges.
		gameHistory.ChallengeRule = pb.ChallengeRule_FIVE_POINT
		// Overwrite the UUID that macondo generates for this game so that
		// it fits in the database.
		gameHistory.Uid = shortuuid.New()
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
		pzls, err := CreatePuzzlesFromGame(ctx, 1000, pgrjReq.Request, reqId, gameStore, puzzlesStore, entGame, pcUUID, ipc.GameType_ANNOTATED, true)
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
	id, err := commondb.GetDBIDFromUUID(ctx, pool, &commondb.CommonDBConfig{
		TableType: commondb.UsersTable,
		Value:     userUUID,
	})
	if err != nil {
		return nil, err
	}
	tx, err := pool.BeginTx(ctx, commondb.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	userRating, err := commondb.GetUserRating(ctx, tx, id, rk)

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

	_, _, _, _, _, _, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
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
	pid, err := commondb.GetDBIDFromUUID(ctx, pool, &commondb.CommonDBConfig{
		TableType: commondb.PuzzlesTable,
		Value:     puzzleUUID,
	})
	if err != nil {
		return 0, err
	}
	var popularity int
	err = pool.QueryRow(ctx, `SELECT SUM(vote) FROM puzzle_votes WHERE puzzle_id = $1`, pid).Scan(&popularity)
	return popularity, err
}

func getPuzzleAttempt(ctx context.Context, pool *pgxpool.Pool, userUUID string, puzzleUUID string) (int32, *pgtype.Bool, error) {
	pid, err := commondb.GetDBIDFromUUID(ctx, pool, &commondb.CommonDBConfig{
		TableType: commondb.PuzzlesTable,
		Value:     puzzleUUID,
	})
	if err != nil {
		return 0, nil, err
	}

	uid, err := commondb.GetDBIDFromUUID(ctx, pool, &commondb.CommonDBConfig{
		TableType: commondb.UsersTable,
		Value:     userUUID,
	})
	if err != nil {
		return 0, nil, err
	}
	var attempts int32
	correct := &pgtype.Bool{}
	err = pool.QueryRow(ctx, `SELECT attempts, correct FROM puzzle_attempts WHERE user_id = $1 AND puzzle_id = $2`, uid, pid).Scan(&attempts, correct)
	if err != nil {
		return 0, nil, err
	}
	return attempts, correct, nil
}

func getNumUnattemptedPuzzles(ctx context.Context, pool *pgxpool.Pool, userUUID string) (int, error) {
	uid, err := commondb.GetDBIDFromUUID(ctx, pool, &commondb.CommonDBConfig{
		TableType: commondb.UsersTable,
		Value:     userUUID,
	})
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
	uid, err := commondb.GetDBIDFromUUID(ctx, pool, &commondb.CommonDBConfig{
		TableType: commondb.UsersTable,
		Value:     userUUID,
	})
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
	uid, err := commondb.GetDBIDFromUUID(ctx, pool, &commondb.CommonDBConfig{
		TableType: commondb.UsersTable,
		Value:     userUUID,
	})
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
