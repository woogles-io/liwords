package puzzles

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"testing"

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

	"github.com/domino14/macondo/board"
	"github.com/domino14/macondo/game"
	"github.com/domino14/macondo/gcgio"
	"github.com/domino14/macondo/gen/api/proto/macondo"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const PuzzlerUUID = "puzzler"
const PuzzleCreatorUUID = "kenji"

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

	// Submit an incorrect answer
	pid, err := GetRandomUnansweredPuzzleIdForUser(ctx, ps, PuzzlerUUID)
	is.NoErr(err)

	_, _, _, attempts, err := GetPuzzle(ctx, ps, PuzzlerUUID, pid)
	is.NoErr(err)
	is.Equal(attempts, int32(0))

	correct, correctAnswer, _, attempts, err := SubmitAnswer(ctx, ps, pid, PuzzlerUUID, &pb.GameEvent{})
	is.NoErr(err)
	is.Equal(attempts, int32(1))
	is.True(correctAnswer == nil)

	correctAnswer, _, _, newPuzzleRating, err := ps.GetAnswer(ctx, pid)
	is.NoErr(err)
	newUserRating, err := getUserRating(ctx, db, PuzzlerUUID, rk)
	is.NoErr(err)

	// User rating should go down, puzzle rating should go up
	is.True(!correct)
	is.True(float64(puzzlesstore.InitialPuzzleRating) < newPuzzleRating.Rating)
	is.True(float64(puzzlesstore.InitialPuzzleRatingDeviation) > newPuzzleRating.RatingDeviation)
	is.True(float64(puzzlesstore.InitialPuzzleRating) > newUserRating.Rating)
	is.True(float64(puzzlesstore.InitialPuzzleRatingDeviation) > newUserRating.RatingDeviation)
	attempts, recordedCorrect, err := getPuzzleAttempt(ctx, db, PuzzlerUUID, pid)
	is.NoErr(err)
	is.Equal(attempts, int32(1))
	is.True(!recordedCorrect.Valid)

	oldPuzzleRating := newPuzzleRating
	oldUserRating := newUserRating

	// Submit the correct answer for the same puzzle,
	correct, _, _, attempts, err = SubmitAnswer(ctx, ps, pid, PuzzlerUUID, correctAnswer)
	is.NoErr(err)

	_, _, _, newPuzzleRating, err = ps.GetAnswer(ctx, pid)
	is.NoErr(err)
	newUserRating, err = getUserRating(ctx, db, PuzzlerUUID, rk)
	is.NoErr(err)

	// rating should remain unchanged and another attempt should be recorded
	is.True(correct)
	is.Equal(attempts, int32(2))
	is.True(common.WithinEpsilon(oldPuzzleRating.Rating, newPuzzleRating.Rating))
	is.True(common.WithinEpsilon(oldPuzzleRating.RatingDeviation, newPuzzleRating.RatingDeviation))
	is.True(common.WithinEpsilon(oldUserRating.Rating, newUserRating.Rating))
	is.True(common.WithinEpsilon(oldUserRating.RatingDeviation, newUserRating.RatingDeviation))
	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, db, PuzzlerUUID, pid)
	is.NoErr(err)
	is.Equal(attempts, int32(2))
	is.True(recordedCorrect.Valid)
	is.True(recordedCorrect.Bool)

	// Submit another answer which should not change the puzzle attempt record
	_, _, _, _, err = SubmitAnswer(ctx, ps, pid, PuzzlerUUID, correctAnswer)
	is.NoErr(err)
	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, db, PuzzlerUUID, pid)
	is.NoErr(err)
	is.Equal(attempts, int32(2))
	is.True(recordedCorrect.Valid)
	is.True(recordedCorrect.Bool)

	oldPuzzleRating = newPuzzleRating
	oldUserRating = newUserRating

	// Submit a correct answer
	pid, err = GetRandomUnansweredPuzzleIdForUser(ctx, ps, PuzzlerUUID)
	is.NoErr(err)

	_, _, _, attempts, err = GetPuzzle(ctx, ps, PuzzlerUUID, pid)
	is.NoErr(err)
	is.Equal(attempts, int32(0))

	correctAnswer, _, _, oldPuzzleRating, err = ps.GetAnswer(ctx, pid)
	is.NoErr(err)

	oldUserRating, err = getUserRating(ctx, db, PuzzlerUUID, rk)
	is.NoErr(err)

	correct, _, _, attempts, err = SubmitAnswer(ctx, ps, pid, PuzzlerUUID, correctAnswer)
	is.NoErr(err)

	_, _, _, newPuzzleRating, err = ps.GetAnswer(ctx, pid)
	is.NoErr(err)

	newUserRating, err = getUserRating(ctx, db, PuzzlerUUID, rk)
	is.NoErr(err)

	// User rating should go up, puzzle rating should go down
	is.True(correct)
	is.Equal(attempts, int32(1))
	is.True(oldPuzzleRating.Rating > newPuzzleRating.Rating)
	is.True(oldPuzzleRating.RatingDeviation > newPuzzleRating.RatingDeviation)
	is.True(oldUserRating.Rating < newUserRating.Rating)
	is.True(oldUserRating.RatingDeviation > newUserRating.RatingDeviation)
	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, db, PuzzlerUUID, pid)
	is.NoErr(err)
	is.Equal(attempts, int32(1))
	is.True(recordedCorrect.Valid)
	is.True(recordedCorrect.Bool)

	// Check that the rating transaction rolls back correctly
	pid, err = GetRandomUnansweredPuzzleIdForUser(ctx, ps, PuzzlerUUID)
	is.NoErr(err)

	correctAnswer, _, _, oldPuzzleRating, err = ps.GetAnswer(ctx, pid)
	is.NoErr(err)
	oldUserRating, err = getUserRating(ctx, db, PuzzlerUUID, rk)
	is.NoErr(err)

	_, _, _, attempts, err = SubmitAnswer(ctx, ps, pid, "incorrect uuid", correctAnswer)
	is.Equal(err.Error(), fmt.Sprintf("cannot get id from uuid %s: no rows for table %s", "incorrect uuid", "users"))

	_, _, _, newPuzzleRating, err = ps.GetAnswer(ctx, pid)
	is.NoErr(err)
	newUserRating, err = getUserRating(ctx, db, PuzzlerUUID, rk)
	is.NoErr(err)

	is.Equal(attempts, int32(-1))
	is.True(common.WithinEpsilon(oldPuzzleRating.Rating, newPuzzleRating.Rating))
	is.True(common.WithinEpsilon(oldPuzzleRating.RatingDeviation, newPuzzleRating.RatingDeviation))
	is.True(common.WithinEpsilon(oldUserRating.Rating, newUserRating.Rating))
	is.True(common.WithinEpsilon(oldUserRating.RatingDeviation, newUserRating.RatingDeviation))

	// Submit an incorrect answer and then give up
	pid, err = GetRandomUnansweredPuzzleIdForUser(ctx, ps, PuzzlerUUID)
	is.NoErr(err)

	_, _, _, attempts, err = GetPuzzle(ctx, ps, PuzzlerUUID, pid)
	is.NoErr(err)
	is.Equal(attempts, int32(0))

	correct, correctAnswer, _, attempts, err = SubmitAnswer(ctx, ps, pid, PuzzlerUUID, &pb.GameEvent{})
	is.NoErr(err)
	is.True(!correct)
	is.True(correctAnswer == nil)
	is.Equal(attempts, int32(1))

	correct, correctAnswer, _, attempts, err = SubmitAnswer(ctx, ps, pid, PuzzlerUUID, nil)
	is.NoErr(err)
	is.True(!correct)
	is.True(correctAnswer != nil)
	is.Equal(attempts, int32(1))

	attempts, recordedCorrect, err = getPuzzleAttempt(ctx, db, PuzzlerUUID, pid)
	is.NoErr(err)
	is.Equal(attempts, int32(1))
	is.True(recordedCorrect.Valid)
	is.True(!recordedCorrect.Bool)
	// The user should not see repeat puzzles until they
	// have answered all of them

	for i := 0; i < totalPuzzles-3; i++ {
		pid, err = GetRandomUnansweredPuzzleIdForUser(ctx, ps, PuzzlerUUID)
		is.NoErr(err)

		_, hist, _, attempts, err := GetPuzzle(ctx, ps, PuzzlerUUID, pid)
		is.NoErr(err)
		is.Equal(attempts, int32(0))

		puzzleDBID, err := transactGetDBIDFromUUID(ctx, db, "puzzles", pid)
		is.NoErr(err)

		var turnNumber int
		err = db.QueryRowContext(ctx, `SELECT turn_number FROM puzzles WHERE id = $1`, puzzleDBID).Scan(&turnNumber)
		is.NoErr(err)

		attempts, err = ps.GetAttempts(ctx, PuzzlerUUID, pid)
		is.NoErr(err)

		is.Equal(attempts, int32(0))
		is.Equal(len(hist.Events), turnNumber)

		_, _, _, attempts, err = SubmitAnswer(ctx, ps, pid, PuzzlerUUID, &pb.GameEvent{})
		is.NoErr(err)
		is.Equal(attempts, int32(1))
	}

	pid, err = GetRandomUnansweredPuzzleIdForUser(ctx, ps, PuzzlerUUID)
	is.NoErr(err)

	attempts, err = ps.GetAttempts(ctx, PuzzlerUUID, pid)
	is.NoErr(err)
	is.Equal(attempts, int32(1))

	// Test voting system

	err = SetPuzzleVote(ctx, ps, PuzzlerUUID, pid, 1)
	is.NoErr(err)

	pop, err := getPuzzlePopularity(ctx, db, pid)
	is.NoErr(err)
	is.Equal(pop, 1)

	err = SetPuzzleVote(ctx, ps, PuzzleCreatorUUID, pid, -1)
	is.NoErr(err)

	pop, err = getPuzzlePopularity(ctx, db, pid)
	is.NoErr(err)
	is.Equal(pop, 0)

	err = SetPuzzleVote(ctx, ps, PuzzlerUUID, pid, 0)
	is.NoErr(err)

	pop, err = getPuzzlePopularity(ctx, db, pid)
	is.NoErr(err)
	is.Equal(pop, -1)

	db.Close()
}

func TestUniqueSingleTileKey(t *testing.T) {
	is := is.New(t)
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "Q.", Direction: macondo.GameEvent_HORIZONTAL}),
		uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "Q.", Direction: macondo.GameEvent_VERTICAL}))
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: ".R", Direction: macondo.GameEvent_HORIZONTAL}),
		uniqueSingleTileKey(&pb.GameEvent{Row: 7, Column: 11, PlayedTiles: ".R", Direction: macondo.GameEvent_VERTICAL}))
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "B....", Direction: macondo.GameEvent_HORIZONTAL}),
		uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "B....", Direction: macondo.GameEvent_VERTICAL}))
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 9, Column: 3, PlayedTiles: "....X", Direction: macondo.GameEvent_HORIZONTAL}),
		uniqueSingleTileKey(&pb.GameEvent{Row: 5, Column: 7, PlayedTiles: "....X", Direction: macondo.GameEvent_VERTICAL}))
	is.Equal(uniqueSingleTileKey(&pb.GameEvent{Row: 11, Column: 9, PlayedTiles: "..A...", Direction: macondo.GameEvent_HORIZONTAL}),
		uniqueSingleTileKey(&pb.GameEvent{Row: 7, Column: 11, PlayedTiles: "....A..", Direction: macondo.GameEvent_VERTICAL}))
	is.True(uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "A.", Direction: macondo.GameEvent_HORIZONTAL}) !=
		uniqueSingleTileKey(&pb.GameEvent{Row: 8, Column: 10, PlayedTiles: "Q.", Direction: macondo.GameEvent_VERTICAL}))
}

func RecreateDB() (*sql.DB, *puzzlesstore.DBStore, int, int, error) {
	cfg := &config.Config{}
	cfg.MacondoConfig = common.DefaultConfig
	cfg.DBConnString = commondb.TestingDBConnStr
	cfg.MacondoConfig.DefaultLexicon = common.DefaultLexicon
	zerolog.SetGlobalLevel(zerolog.Disabled)
	ctx := context.Background()

	// Recreate the test database
	commondb.RecreateDB()

	// Reconnect to the new test database
	db, err := commondb.OpenDB()
	if err != nil {
		return nil, nil, 0, 0, err
	}

	userStore, err := user.NewDBStore(commondb.TestingDBConnStr)
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

	m, err := migrate.New(commondb.MigrationFile, commondb.MigrationConnString)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	if err := m.Up(); err != nil {
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
		pcUUID := ""
		if idx%2 == 1 {
			pcUUID = PuzzleCreatorUUID
		}
		pzls, err := CreatePuzzlesFromGame(ctx, gameStore, puzzlesStore, game, pcUUID, ipc.GameType_ANNOTATED)
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
