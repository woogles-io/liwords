package puzzles

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	commondb "github.com/domino14/liwords/pkg/stores/common"
	gamestore "github.com/domino14/liwords/pkg/stores/game"
	puzzlesstore "github.com/domino14/liwords/pkg/stores/puzzles"
	"github.com/domino14/liwords/pkg/stores/user"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/macondo/board"
	"github.com/domino14/macondo/game"
	"github.com/domino14/macondo/gcgio"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/macondo/move"
)

const PuzzlerUUID = "puzzler"
const PuzzleCreatorUUID = "kenji"

func gameEventToClientGameplayEvent(evt *macondopb.GameEvent) *ipc.ClientGameplayEvent {
	cge := &ipc.ClientGameplayEvent{}

	switch evt.Type {
	case macondopb.GameEvent_TILE_PLACEMENT_MOVE:
		cge.Type = ipc.ClientGameplayEvent_TILE_PLACEMENT
		cge.Tiles = evt.PlayedTiles
		cge.PositionCoords = move.ToBoardGameCoords(int(evt.Row), int(evt.Column),
			evt.Direction == macondopb.GameEvent_VERTICAL)

	case macondopb.GameEvent_EXCHANGE:
		cge.Type = ipc.ClientGameplayEvent_EXCHANGE
		cge.Tiles = evt.Exchanged
	case macondopb.GameEvent_PASS:
		cge.Type = ipc.ClientGameplayEvent_PASS
	}

	return cge
}

func TestPuzzles(t *testing.T) {
	is := is.New(t)
	db, ps, authoredPuzzles, totalPuzzles, err := RecreateDB()
	is.NoErr(err)
	ctx := context.Background()

	rk := ratingKey(common.DefaultGameReq)

	pcid, err := transactGetDBIDFromUUID(ctx, db, "users", PuzzleCreatorUUID)
	is.NoErr(err)

	var curatedPuzzles int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM puzzles WHERE author_id = $1`, pcid).Scan(&curatedPuzzles)
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

	// Path 1
	// Submit an incorrect answer
	puzzleUUID, err := GetRandomUnansweredPuzzleIdForUser(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, err = GetPreviousPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.Equal(err.Error(), entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_NO_ATTEMPTS, PuzzlerUUID, puzzleUUID).Error())

	hist, _, attempts, userPreviousCorrect, firstAttemptTime, lastAttemptTime, err := GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(0))
	is.True(userPreviousCorrect == nil)
	is.Equal(hist.OriginalGcg, "")
	is.Equal(hist.IdAuth, "")
	is.Equal(hist.Uid, "")
	is.True(firstAttemptTime.Equal(time.Time{}))
	is.True(lastAttemptTime.Equal(time.Time{}))

	status, correctAnswer, gameId, _, attempts, _, _, err := SubmitAnswer(ctx, ps, puzzleUUID, PuzzlerUUID, &ipc.ClientGameplayEvent{}, false)
	is.NoErr(err)
	is.Equal(attempts, int32(1))
	is.True(status == nil)
	is.True(correctAnswer == nil)
	is.Equal(gameId, "")
	is.Equal(attempts, int32(1))

	_, _, attempts, userPreviousCorrect, firstAttemptTime, lastAttemptTime, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(1))
	is.True(userPreviousCorrect == nil)
	is.True(!firstAttemptTime.Equal(time.Time{}))
	is.True(!lastAttemptTime.Equal(time.Time{}))
	is.True(firstAttemptTime.Equal(lastAttemptTime))

	_, err = GetPreviousPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.Equal(err.Error(), entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_ATTEMPT_NOT_FOUND, PuzzlerUUID, puzzleUUID).Error())

	correctAnswer, _, _, _, newPuzzleRating, err := ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)
	is.True(correctAnswer != nil)

	newUserRating, err := getUserRating(ctx, db, PuzzlerUUID, rk)
	is.NoErr(err)

	// User rating should go down, puzzle rating should go up
	is.True(float64(puzzlesstore.InitialPuzzleRating) < newPuzzleRating.Rating)
	is.True(float64(puzzlesstore.InitialPuzzleRatingDeviation) > newPuzzleRating.RatingDeviation)
	is.True(float64(puzzlesstore.InitialPuzzleRating) > newUserRating.Rating)
	is.True(float64(puzzlesstore.InitialPuzzleRatingDeviation) > newUserRating.RatingDeviation)
	attempts, recordedCorrect, err := getPuzzleAttempt(ctx, db, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(1))
	is.True(!recordedCorrect.Valid)

	oldPuzzleRating := newPuzzleRating
	oldUserRating := newUserRating
	correctCGE := gameEventToClientGameplayEvent(correctAnswer)

	// Path 7
	// Submit the correct answer for the same puzzle,
	status, correctAnswer, gameId, _, attempts, _, _, err = SubmitAnswer(ctx, ps, puzzleUUID, PuzzlerUUID, correctCGE, false)
	is.NoErr(err)
	is.True(correctAnswer != nil)
	is.True(gameId != "")

	_, _, _, _, newPuzzleRating, err = ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)
	newUserRating, err = getUserRating(ctx, db, PuzzlerUUID, rk)
	is.NoErr(err)

	// rating should remain unchanged and another attempt should be recorded
	is.True(*status)
	is.Equal(attempts, int32(2))
	is.True(common.WithinEpsilon(oldPuzzleRating.Rating, newPuzzleRating.Rating))
	is.True(common.WithinEpsilon(oldPuzzleRating.RatingDeviation, newPuzzleRating.RatingDeviation))
	is.True(common.WithinEpsilon(oldUserRating.Rating, newUserRating.Rating))
	is.True(common.WithinEpsilon(oldUserRating.RatingDeviation, newUserRating.RatingDeviation))
	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, db, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(2))
	is.True(recordedCorrect.Valid)
	is.True(recordedCorrect.Bool)

	// Path 4
	// Submit another answer which should not change the puzzle attempt record
	status, correctAnswer, gameId, _, _, _, _, err = SubmitAnswer(ctx, ps, puzzleUUID, PuzzlerUUID, correctCGE, false)
	is.NoErr(err)
	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, db, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(*status)
	is.True(answersAreEqual(correctCGE, correctAnswer))
	is.True(gameId != "")
	is.Equal(attempts, int32(2))
	is.True(recordedCorrect.Valid)
	is.True(recordedCorrect.Bool)

	oldPuzzleRating = newPuzzleRating
	oldUserRating = newUserRating
	expectedPreviousPuzzleUUID := puzzleUUID

	// Path 3
	// Submit a correct answer
	puzzleUUID, err = GetRandomUnansweredPuzzleIdForUser(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	// Get the previous puzzle while the current puzzle is unattempted
	actualPreviousPuzzleUUID, err := GetPreviousPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(expectedPreviousPuzzleUUID, actualPreviousPuzzleUUID)

	// Getting previous again should fail since that was the first puzzle
	// the user attempted
	_, err = GetPreviousPuzzle(ctx, ps, PuzzlerUUID, actualPreviousPuzzleUUID)
	is.Equal(err.Error(), entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_ATTEMPT_NOT_FOUND, PuzzlerUUID, actualPreviousPuzzleUUID).Error())

	_, _, attempts, userPreviousCorrect, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(userPreviousCorrect == nil)
	is.Equal(attempts, int32(0))

	correctAnswer, _, _, _, oldPuzzleRating, err = ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)

	oldUserRating, err = getUserRating(ctx, db, PuzzlerUUID, rk)
	is.NoErr(err)

	correctCGE = gameEventToClientGameplayEvent(correctAnswer)

	status, _, _, _, attempts, _, _, err = SubmitAnswer(ctx, ps, puzzleUUID, PuzzlerUUID, correctCGE, false)
	is.NoErr(err)

	// Get the previous puzzle while the current puzzle is attempted
	actualPreviousPuzzleUUID, err = GetPreviousPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(expectedPreviousPuzzleUUID, actualPreviousPuzzleUUID)

	_, _, _, _, newPuzzleRating, err = ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)

	newUserRating, err = getUserRating(ctx, db, PuzzlerUUID, rk)
	is.NoErr(err)

	// User rating should go up, puzzle rating should go down
	is.True(*status)
	is.Equal(attempts, int32(1))
	is.True(oldPuzzleRating.Rating > newPuzzleRating.Rating)
	is.True(oldPuzzleRating.RatingDeviation > newPuzzleRating.RatingDeviation)
	is.True(oldUserRating.Rating < newUserRating.Rating)
	is.True(oldUserRating.RatingDeviation > newUserRating.RatingDeviation)
	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, db, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(1))
	is.True(recordedCorrect.Valid)
	is.True(recordedCorrect.Bool)

	// Check that the rating transaction rolls back correctly
	puzzleUUID, err = GetRandomUnansweredPuzzleIdForUser(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	correctAnswer, _, _, _, oldPuzzleRating, err = ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)
	oldUserRating, err = getUserRating(ctx, db, PuzzlerUUID, rk)
	is.NoErr(err)

	correctCGE = gameEventToClientGameplayEvent(correctAnswer)
	_, _, _, _, attempts, _, _, err = SubmitAnswer(ctx, ps, puzzleUUID, "incorrect uuid", correctCGE, false)
	is.Equal(err.Error(), fmt.Sprintf("cannot get id from uuid %s: no rows for table %s", "incorrect uuid", "users"))

	_, _, _, _, newPuzzleRating, err = ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)
	newUserRating, err = getUserRating(ctx, db, PuzzlerUUID, rk)
	is.NoErr(err)

	is.Equal(attempts, int32(-1))
	is.True(common.WithinEpsilon(oldPuzzleRating.Rating, newPuzzleRating.Rating))
	is.True(common.WithinEpsilon(oldPuzzleRating.RatingDeviation, newPuzzleRating.RatingDeviation))
	is.True(common.WithinEpsilon(oldUserRating.Rating, newUserRating.Rating))
	is.True(common.WithinEpsilon(oldUserRating.RatingDeviation, newUserRating.RatingDeviation))

	// Path 5 and 6
	// Submit incorrect answers and then give up
	puzzleUUID, err = GetRandomUnansweredPuzzleIdForUser(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, _, attempts, userPreviousCorrect, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(userPreviousCorrect == nil)
	is.Equal(attempts, int32(0))

	status, correctAnswer, gameId, _, attempts, _, _, err = SubmitAnswer(ctx, ps, puzzleUUID, PuzzlerUUID, &ipc.ClientGameplayEvent{}, false)
	is.NoErr(err)
	is.True(status == nil)
	is.True(correctAnswer == nil)
	is.True(gameId == "")
	is.Equal(attempts, int32(1))

	status, correctAnswer, gameId, _, attempts, _, _, err = SubmitAnswer(ctx, ps, puzzleUUID, PuzzlerUUID, &ipc.ClientGameplayEvent{}, false)
	is.NoErr(err)
	is.True(status == nil)
	is.True(correctAnswer == nil)
	is.True(gameId == "")
	is.Equal(attempts, int32(2))

	correctAnswer, _, _, _, _, err = ps.GetAnswer(ctx, puzzleUUID)
	is.NoErr(err)

	// If the user has given up, the answer sent should not be considered
	// for recording the correctness of the attempt
	status, correctAnswer, gameId, _, attempts, _, _, err = SubmitAnswer(ctx, ps, puzzleUUID, PuzzlerUUID, gameEventToClientGameplayEvent(correctAnswer), true)
	is.NoErr(err)
	is.True(!*status)
	is.True(correctAnswer != nil)
	is.True(gameId != "")
	is.Equal(attempts, int32(2))

	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, db, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(2))
	is.True(recordedCorrect.Valid)
	is.True(!recordedCorrect.Bool)

	// The response for getting a puzzle should be
	// different if the user is logged out
	hist, _, attempts, userPreviousCorrect, firstAttemptTime, lastAttemptTime, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(hist != nil)
	is.Equal(attempts, int32(2))
	is.True(!*userPreviousCorrect)
	is.True(!firstAttemptTime.Equal(time.Time{}))
	is.True(!lastAttemptTime.Equal(time.Time{}))
	is.True(lastAttemptTime.After(firstAttemptTime))

	hist, _, attempts, userPreviousCorrect, firstAttemptTime, lastAttemptTime, err = GetPuzzle(ctx, ps, "", puzzleUUID)
	is.NoErr(err)
	is.True(hist != nil)
	is.Equal(attempts, int32(0))
	is.True(userPreviousCorrect == nil)
	is.True(firstAttemptTime.Equal(time.Time{}))
	is.True(lastAttemptTime.Equal(time.Time{}))

	// Path 2
	// Give up immediately without submitting any answers
	puzzleUUID, err = GetRandomUnansweredPuzzleIdForUser(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	_, _, attempts, userPreviousCorrect, _, _, err = GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.True(userPreviousCorrect == nil)
	is.Equal(attempts, int32(0))

	status, correctAnswer, gameId, _, attempts, _, _, err = SubmitAnswer(ctx, ps, puzzleUUID, PuzzlerUUID, nil, true)
	is.NoErr(err)
	is.True(!*status)
	is.True(correctAnswer != nil)
	is.True(gameId != "")
	is.Equal(attempts, int32(0))

	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, db, PuzzlerUUID, puzzleUUID)
	is.NoErr(err)
	is.Equal(attempts, int32(0))
	is.True(recordedCorrect.Valid)
	is.True(!recordedCorrect.Bool)

	// The user should not see repeat puzzles until they
	// have answered all of them
	unseenPuzzles, err := getNumUnattemptedPuzzlesInLexicon(ctx, db, PuzzlerUUID, common.DefaultGameReq.Lexicon)
	is.NoErr(err)

	for i := 0; i < unseenPuzzles; i++ {
		expectedPreviousPuzzleUUID = puzzleUUID
		puzzleUUID, err = GetRandomUnansweredPuzzleIdForUser(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
		is.NoErr(err)

		actualPreviousPuzzleUUID, err = GetPreviousPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
		is.NoErr(err)
		is.Equal(expectedPreviousPuzzleUUID, actualPreviousPuzzleUUID)

		puzzleLexicon, err := getPuzzleLexicon(ctx, db, puzzleUUID)
		is.NoErr(err)
		is.Equal(puzzleLexicon, common.DefaultGameReq.Lexicon)

		hist, _, attempts, _, _, _, err := GetPuzzle(ctx, ps, PuzzlerUUID, puzzleUUID)
		is.NoErr(err)
		is.Equal(attempts, int32(0))

		puzzleDBID, err := transactGetDBIDFromUUID(ctx, db, "puzzles", puzzleUUID)
		is.NoErr(err)

		var turnNumber int
		err = db.QueryRowContext(ctx, `SELECT turn_number FROM puzzles WHERE id = $1`, puzzleDBID).Scan(&turnNumber)
		is.NoErr(err)

		attempts, _, _, _, err = ps.GetAttempts(ctx, PuzzlerUUID, puzzleUUID)
		is.NoErr(err)

		is.Equal(attempts, int32(0))
		is.Equal(len(hist.Events), turnNumber)

		_, _, _, _, attempts, _, _, err = SubmitAnswer(ctx, ps, puzzleUUID, PuzzlerUUID, &ipc.ClientGameplayEvent{}, false)
		is.NoErr(err)
		is.Equal(attempts, int32(1))
	}

	// The user should only see puzzles for their requested lexicon
	// regardless of how many puzzles they request
	attemptedPuzzles, err := getNumAttemptedPuzzles(ctx, db, PuzzlerUUID)
	is.NoErr(err)

	unattemptedPuzzles, err := getNumUnattemptedPuzzles(ctx, db, PuzzlerUUID)
	is.NoErr(err)
	is.Equal(totalPuzzles, attemptedPuzzles+unattemptedPuzzles)

	for i := 0; i < totalPuzzles*10; i++ {
		puzzleUUID, err = GetRandomUnansweredPuzzleIdForUser(ctx, ps, PuzzlerUUID, common.DefaultGameReq.Lexicon)
		is.NoErr(err)
		_, _, _, _, _, _, _, err = SubmitAnswer(ctx, ps, puzzleUUID, PuzzlerUUID, &ipc.ClientGameplayEvent{}, false)
		is.NoErr(err)
	}

	newAttemptedPuzzles, err := getNumAttemptedPuzzles(ctx, db, PuzzlerUUID)
	is.NoErr(err)
	is.Equal(newAttemptedPuzzles, attemptedPuzzles)

	// Test voting system

	err = SetPuzzleVote(ctx, ps, PuzzlerUUID, puzzleUUID, 1)
	is.NoErr(err)

	pop, err := getPuzzlePopularity(ctx, db, puzzleUUID)
	is.NoErr(err)
	is.Equal(pop, 1)

	err = SetPuzzleVote(ctx, ps, PuzzleCreatorUUID, puzzleUUID, -1)
	is.NoErr(err)

	pop, err = getPuzzlePopularity(ctx, db, puzzleUUID)
	is.NoErr(err)
	is.Equal(pop, 0)

	err = SetPuzzleVote(ctx, ps, PuzzlerUUID, puzzleUUID, 0)
	is.NoErr(err)

	pop, err = getPuzzlePopularity(ctx, db, puzzleUUID)
	is.NoErr(err)
	is.Equal(pop, -1)

	db.Close()
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

func RecreateDB() (*sql.DB, *puzzlesstore.DBStore, int, int, error) {
	cfg := &config.Config{}
	cfg.MacondoConfig = common.DefaultConfig
	cfg.DBConnUri = commondb.TestingPostgresConnUri()
	cfg.DBConnDSN = commondb.TestingPostgresConnDSN()
	cfg.MacondoConfig.DefaultLexicon = common.DefaultLexicon
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	ctx := context.Background()
	log.Info().Msg("here first")
	// Recreate the test database
	err := commondb.RecreateTestDB()
	if err != nil {
		return nil, nil, 0, 0, err
	}

	// Reconnect to the new test database
	db, err := commondb.OpenTestingDB()
	if err != nil {
		return nil, nil, 0, 0, err
	}

	userStore, err := user.NewDBStore(commondb.TestingPostgresConnDSN())
	if err != nil {
		return nil, nil, 0, 0, err
	}
	err = userStore.New(context.Background(), &entity.User{Username: "Puzzler", Email: "puzzler@woogles.io", UUID: PuzzlerUUID})
	if err != nil {
		return nil, nil, 0, 0, err
	}

	err = userStore.New(context.Background(), &entity.User{Username: "Kenji", Email: "kenji@woogles.io", UUID: PuzzleCreatorUUID})
	if err != nil {
		return nil, nil, 0, 0, err
	}

	gameStore, err := gamestore.NewDBStore(cfg, userStore)
	if err != nil {
		return nil, nil, 0, 0, err
	}

	puzzlesStore, err := puzzlesstore.NewDBStore(db)
	if err != nil {
		return nil, nil, 0, 0, err
	}

	files, err := ioutil.ReadDir("./testdata")
	if err != nil {
		return nil, nil, 0, 0, err
	}

	rules, err := game.NewBasicGameRules(&common.DefaultConfig, common.DefaultLexicon, board.CrosswordGameLayout, "english", game.CrossScoreAndSet, game.VarClassic)
	if err != nil {
		return nil, nil, 0, 0, err
	}

	authoredPuzzles := 0
	totalPuzzles := 0
	for idx, f := range files {
		gameHistory, err := gcgio.ParseGCG(&common.DefaultConfig, fmt.Sprintf("./testdata/%s", f.Name()))
		if err != nil {
			return nil, nil, 0, 0, err
		}
		// Set the correct challenge rule to allow games with
		// lost challenges.
		gameHistory.ChallengeRule = pb.ChallengeRule_FIVE_POINT
		game, err := game.NewFromHistory(gameHistory, rules, 0)
		if err != nil {
			return nil, nil, 0, 0, err
		}
		gameReq := proto.Clone(common.DefaultGameReq).(*ipc.GameRequest)

		pcUUID := ""
		if idx%2 == 1 {
			pcUUID = PuzzleCreatorUUID
			gameReq.Lexicon = "CSW19"
		}
		entGame := entity.NewGame(game, gameReq)
		pzls, err := CreatePuzzlesFromGame(ctx, gameStore, puzzlesStore, entGame, pcUUID, ipc.GameType_ANNOTATED)
		if err != nil {
			return nil, nil, 0, 0, err
		}
		if idx%2 == 1 {
			authoredPuzzles += len(pzls)
		}
		totalPuzzles += len(pzls)
	}

	return db, puzzlesStore, authoredPuzzles, totalPuzzles, nil
}

func getUserRating(ctx context.Context, db *sql.DB, userUUID string, rk entity.VariantKey) (*entity.SingleRating, error) {
	id, err := transactGetDBIDFromUUID(ctx, db, "users", userUUID)
	if err != nil {
		return nil, err
	}

	var ratings *entity.Ratings
	err = db.QueryRowContext(ctx, `SELECT ratings FROM profiles WHERE user_id = $1`, id).Scan(&ratings)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("profile not found for user_id: %s", userUUID)
	}
	if err != nil {
		return nil, err
	}

	sr, exists := ratings.Data[rk]
	if !exists {
		return nil, fmt.Errorf("rating does not exist for rating key %s", rk)
	}

	return &sr, nil
}

func getPuzzlePopularity(ctx context.Context, db *sql.DB, puzzleUUID string) (int, error) {
	pid, err := transactGetDBIDFromUUID(ctx, db, "puzzles", puzzleUUID)
	if err != nil {
		return 0, err
	}
	var popularity int
	err = db.QueryRowContext(ctx, `SELECT SUM(vote) FROM puzzle_votes WHERE puzzle_id = $1`, pid).Scan(&popularity)
	return popularity, err
}

func getPuzzleAttempt(ctx context.Context, db *sql.DB, userUUID string, puzzleUUID string) (int32, *sql.NullBool, error) {
	pid, err := transactGetDBIDFromUUID(ctx, db, "puzzles", puzzleUUID)
	if err != nil {
		return 0, nil, err
	}

	uid, err := transactGetDBIDFromUUID(ctx, db, "users", userUUID)
	if err != nil {
		return 0, nil, err
	}
	var attempts int32
	correct := &sql.NullBool{}
	err = db.QueryRowContext(ctx, `SELECT attempts, correct FROM puzzle_attempts WHERE user_id = $1 AND puzzle_id = $2`, uid, pid).Scan(&attempts, correct)
	if err != nil {
		return 0, nil, err
	}
	return attempts, correct, nil
}

func getNumUnattemptedPuzzles(ctx context.Context, db *sql.DB, userUUID string) (int, error) {
	uid, err := transactGetDBIDFromUUID(ctx, db, "users", userUUID)
	if err != nil {
		return -1, err
	}
	var unseen int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM puzzles WHERE puzzles.id NOT IN (SELECT puzzle_id FROM puzzle_attempts WHERE user_id = $1)`, uid).Scan(&unseen)
	if err != nil {
		return -1, err
	}
	return unseen, nil
}

func getNumAttemptedPuzzles(ctx context.Context, db *sql.DB, userUUID string) (int, error) {
	uid, err := transactGetDBIDFromUUID(ctx, db, "users", userUUID)
	if err != nil {
		return -1, err
	}
	var attempted int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM puzzle_attempts WHERE user_id = $1`, uid).Scan(&attempted)
	if err != nil {
		return -1, err
	}
	return attempted, nil
}

func getNumUnattemptedPuzzlesInLexicon(ctx context.Context, db *sql.DB, userUUID string, lexicon string) (int, error) {
	uid, err := transactGetDBIDFromUUID(ctx, db, "users", userUUID)
	if err != nil {
		return -1, err
	}
	var unseen int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM puzzles WHERE lexicon = $1 AND puzzles.id NOT IN (SELECT puzzle_id FROM puzzle_attempts WHERE user_id = $2)`, lexicon, uid).Scan(&unseen)
	if err != nil {
		return -1, err
	}
	return unseen, nil
}

func getPuzzleLexicon(ctx context.Context, db *sql.DB, puzzleUUID string) (string, error) {
	var lexicon string
	err := db.QueryRowContext(ctx, `SELECT lexicon FROM puzzles WHERE uuid = $1`, puzzleUUID).Scan(&lexicon)
	if err != nil {
		return "", err
	}
	return lexicon, nil
}

func transactGetDBIDFromUUID(ctx context.Context, db *sql.DB, table string, uuid string) (int64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

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

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}
