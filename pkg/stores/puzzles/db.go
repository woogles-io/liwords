package puzzles

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	commontest "github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/glicko"
	"github.com/domino14/liwords/pkg/stores/common"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/domino14/liwords/rpc/api/proto/puzzle_service"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DBStore struct {
	dbPool *pgxpool.Pool
}

type answer struct {
	EventType   int32
	Row         int32
	Column      int32
	Direction   int32
	Position    string
	PlayedTiles string
	Exchanged   string
	Score       int32
	IsBingo     bool
}

func NewDBStore(p *pgxpool.Pool) (*DBStore, error) {
	return &DBStore{dbPool: p}, nil
}

func (s *DBStore) Disconnect() {
	s.dbPool.Close()
}

func (s *DBStore) CreateGenerationLog(ctx context.Context, req *puzzle_service.PuzzleGenerationJobRequest) (int, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return -1, err
	}
	defer tx.Rollback(ctx)

	data, err := json.Marshal(req)
	if err != nil {
		return -1, err
	}

	var id int
	err = tx.QueryRow(ctx, `INSERT INTO puzzle_generation_logs (request, created_at) VALUES ($1, NOW()) RETURNING id`, data).Scan(&id)
	if err != nil {
		return -1, err
	}

	if err := tx.Commit(ctx); err != nil {
		return -1, err
	}

	return id, nil
}

func (s *DBStore) UpdateGenerationLogStatus(ctx context.Context, id int, fulfilled bool, procErr error) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	errorStatus := &sql.NullString{}
	if procErr != nil {
		errorStatus.Valid = true
		errorStatus.String = procErr.Error()
	}
	result, err := tx.Exec(ctx, `UPDATE puzzle_generation_logs SET completed_at = NOW(), error_status = $1, fulfilled = $2 WHERE id = $3`, errorStatus, fulfilled, id)
	if result.RowsAffected() != 1 {
		return fmt.Errorf("no rows affecting when updating log %d", id)
	}
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) GetJobLogs(ctx context.Context, limit, offset int) ([]*puzzle_service.PuzzleJobLog, error) {
	var createdAt time.Time
	var completedAt time.Time
	rows, err := s.dbPool.Query(ctx,
		`SELECT id, request, fulfilled, error_status, created_at, completed_at
	 FROM puzzle_generation_logs
	 ORDER by created_at DESC
	 LIMIT $1
	 OFFSET $2
	 `, limit, offset)

	jobLogs := []*puzzle_service.PuzzleJobLog{}
	for rows.Next() {
		jobLog := puzzle_service.PuzzleJobLog{}
		fulfilled := &sql.NullBool{}
		errorStatus := &sql.NullString{}

		err = rows.Scan(&jobLog.Id, &jobLog.Request, fulfilled, errorStatus, &createdAt, &completedAt)
		if err != nil {
			rows.Close()
			return nil, err
		}
		if fulfilled.Valid {
			jobLog.Fulfilled = fulfilled.Bool
		} else {
			jobLog.Fulfilled = false
		}
		if errorStatus.Valid {
			jobLog.ErrorStatus = errorStatus.String
		} else {
			jobLog.ErrorStatus = ""
		}
		jobLog.CreatedAt = timestamppb.New(createdAt)
		jobLog.CompletedAt = timestamppb.New(completedAt)
		jobLogs = append(jobLogs, &jobLog)
	}
	rows.Close()
	if err != nil {
		return nil, err
	}
	return jobLogs, nil
}

func (s *DBStore) CreatePuzzle(ctx context.Context, gameUUID string, turnNumber int32, answer *macondopb.GameEvent, authorUUID string,
	lexicon string, beforeText string, afterText string, tags []macondopb.PuzzleTag, generationId int, bucketIndex int32) error {

	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	gameID, err := common.GetGameDBIDFromUUID(ctx, tx, gameUUID)
	if err != nil {
		return err
	}

	authorId := sql.NullInt64{Valid: false}
	if authorUUID != "" {
		aid, err := common.GetUserDBIDFromUUID(ctx, tx, authorUUID)
		if err != nil {
			return err
		}
		authorId.Int64 = int64(aid)
		authorId.Valid = true
	}

	newRating := &entity.SingleRating{
		Rating:            float64(glicko.InitialRating),
		RatingDeviation:   float64(glicko.InitialRatingDeviation),
		Volatility:        glicko.InitialVolatility,
		LastGameTimestamp: time.Now().Unix()}

	uuid := shortuuid.New()

	var id int
	err = tx.QueryRow(ctx, `INSERT INTO puzzles (uuid, game_id, turn_number, author_id, answer, lexicon, before_text, after_text, rating, generation_id, bucket_index, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()) RETURNING id`,
		uuid, gameID, turnNumber, authorId, gameEventToAnswer(answer), lexicon, beforeText, afterText, newRating, generationId, bucketIndex).Scan(&id)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		_, err := tx.Exec(ctx, `INSERT INTO puzzle_tags(tag_id, puzzle_id) VALUES ((SELECT id FROM puzzle_tag_titles WHERE tag_title = $1), $2)`, tag.String(), id)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return err
}

func (s *DBStore) GetStartPuzzleId(ctx context.Context, userUUID string, lexicon string) (string, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.RepeatableReadTxOptions)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var startPuzzleUUID string
	if userUUID == "" {
		startPuzzleUUID, err = getRandomPuzzleUUID(ctx, tx, lexicon, nil)
		if err != nil {
			return "", err
		}
	} else {
		uid, err := common.GetUserDBIDFromUUID(ctx, tx, userUUID)
		if err != nil {
			log.Err(err).Msg("get-user-dbid")
			return "", err
		}

		getNext := false
		var pid int
		status := &sql.NullBool{}
		err = tx.QueryRow(ctx, `SELECT puzzle_id, correct FROM puzzle_attempts WHERE user_id = $1 AND (SELECT lexicon FROM puzzles WHERE id = puzzle_id) = $2 ORDER BY updated_at DESC LIMIT 1`, uid, lexicon).Scan(&pid, status)
		if err == pgx.ErrNoRows {
			// User has not seen any puzzles, just get a random puzzle
			getNext = true
		} else if err != nil {
			log.Err(err).Msg("error-init-query")
			return "", err
		}

		// If the user has not seen any puzzles
		// or they solved or gave up on the last puzzle,
		// give the user a new puzzle
		if getNext || status.Valid {
			startPuzzleUUID, err = getNextPuzzleId(ctx, tx, userUUID, lexicon)
			if err != nil {
				return "", err
			}
		} else {
			err = tx.QueryRow(ctx, `SELECT uuid FROM puzzles WHERE id = $1`, pid).Scan(&startPuzzleUUID)
			if err != nil {
				log.Err(err).Msg("error-scanning")
				return "", err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return startPuzzleUUID, nil
}

func (s *DBStore) GetNextPuzzleId(ctx context.Context, userUUID string, lexicon string) (string, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.RepeatableReadTxOptions)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var nextPuzzleUUID string

	if userUUID == "" {
		nextPuzzleUUID, err = getRandomPuzzleUUID(ctx, tx, lexicon, nil)
		if err != nil {
			return "", err
		}
	} else {
		nextPuzzleUUID, err = getNextPuzzleId(ctx, tx, userUUID, lexicon)
		if err != nil {
			return "", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return nextPuzzleUUID, nil
}

func (s *DBStore) GetPuzzle(ctx context.Context, userUUID string, puzzleUUID string) (*macondopb.GameHistory, string, int32, *bool, time.Time, time.Time, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, "", -1, nil, time.Time{}, time.Time{}, err
	}
	defer tx.Rollback(ctx)

	userLoggedIn := userUUID != ""
	attemptExists := false
	attempts := int32(0)
	var status *bool
	firstAttemptTime := time.Time{}
	lastAttemptTime := time.Time{}

	if userLoggedIn {
		_, attemptExists, attempts, status, firstAttemptTime, lastAttemptTime, err = getAttempts(ctx, tx, userUUID, puzzleUUID)
		if err != nil {
			return nil, "", -1, nil, time.Time{}, time.Time{}, err
		}
	}

	var gameId int
	var turnNumber int
	var beforeText string

	err = tx.QueryRow(ctx, `SELECT game_id, turn_number, before_text FROM puzzles WHERE uuid = $1`, puzzleUUID).Scan(&gameId, &turnNumber, &beforeText)
	if err == pgx.ErrNoRows {
		return nil, "", -1, nil, time.Time{}, time.Time{}, entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PUZZLE_UUID_NOT_FOUND, userUUID, puzzleUUID)
	}
	if err != nil {
		return nil, "", -1, nil, time.Time{}, time.Time{}, err
	}

	hist, _, _, err := common.GetGameInfo(ctx, tx, gameId)
	if err != nil {
		return nil, "", -1, nil, time.Time{}, time.Time{}, err
	}

	if len(hist.Players) < 2 {
		return nil, "", -1, nil, time.Time{}, time.Time{}, fmt.Errorf("history has less than two players for puzzle %s", puzzleUUID)
	}

	// Create the puzzle attempt if it does not exist
	// attemptExists will also be true if the user is not logged in
	if userLoggedIn {
		pid, err := common.GetPuzzleDBIDFromUUID(ctx, tx, puzzleUUID)
		if err != nil {
			return nil, "", -1, nil, time.Time{}, time.Time{}, err
		}

		uid, err := common.GetUserDBIDFromUUID(ctx, tx, userUUID)
		if err != nil {
			return nil, "", -1, nil, time.Time{}, time.Time{}, err
		}
		if attemptExists {
			result, err := tx.Exec(ctx, `UPDATE puzzle_attempts SET updated_at = NOW() WHERE puzzle_id = $1 AND user_id = $2`, pid, uid)
			rowsAffected := result.RowsAffected()
			if err != nil {
				return nil, "", -1, nil, time.Time{}, time.Time{}, err
			}
			if rowsAffected != 1 {
				return nil, "", -1, nil, time.Time{}, time.Time{}, entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PUZZLE_UPDATE_ATTEMPT, userUUID, puzzleUUID)
			}
		} else {
			_, err = tx.Exec(ctx, `INSERT INTO puzzle_attempts (puzzle_id, user_id, attempts, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`, pid, uid, 0)
			if err != nil {
				return nil, "", -1, nil, time.Time{}, time.Time{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, "", -1, nil, time.Time{}, time.Time{}, err
	}

	puzzleEvent := hist.Events[turnNumber]

	hist.Events = hist.Events[:turnNumber]
	// Set LastKnownRacks to make history valid.
	playerIndexes := map[string]int{}
	for idx := range hist.Players {
		playerIndexes[hist.Players[idx].Nickname] = idx
	}
	// XXX: Outdated, fix with PlayerIndex in the future
	hist.LastKnownRacks = []string{"", ""}
	idx := playerIndexes[puzzleEvent.Nickname]
	hist.LastKnownRacks[idx] = puzzleEvent.Rack
	hist.OriginalGcg = ""
	hist.IdAuth = ""
	hist.Uid = ""

	sanitizeHistory(hist)

	return hist, beforeText, attempts, status, firstAttemptTime, lastAttemptTime, nil
}

func (s *DBStore) GetPreviousPuzzleId(ctx context.Context, userUUID string, puzzleUUID string) (string, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	pid, err := common.GetPuzzleDBIDFromUUID(ctx, tx, puzzleUUID)
	if err != nil {
		return "", err
	}

	uid, err := common.GetUserDBIDFromUUID(ctx, tx, userUUID)
	if err != nil {
		return "", err
	}

	var currentPuzzleUpdatedTime time.Time
	err = tx.QueryRow(ctx, `SELECT updated_at FROM puzzle_attempts WHERE puzzle_id = $1 AND user_id = $2`, pid, uid).Scan(&currentPuzzleUpdatedTime)
	if err == pgx.ErrNoRows {
		return "", entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_NO_ATTEMPTS, userUUID, puzzleUUID)
	}
	if err != nil {
		return "", err
	}

	var previousPid int
	err = tx.QueryRow(ctx, `SELECT puzzle_id FROM puzzle_attempts WHERE user_id = $1 AND updated_at <= $2 AND puzzle_id != $3 ORDER BY updated_at DESC LIMIT 1`, uid, currentPuzzleUpdatedTime, pid).Scan(&previousPid)
	if err == pgx.ErrNoRows {
		return "", entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_ATTEMPT_NOT_FOUND, userUUID, puzzleUUID)
	}
	if err != nil {
		return "", err
	}

	var previousPuzzleUUID string
	err = tx.QueryRow(ctx, `SELECT uuid FROM puzzles WHERE id = $1`, previousPid).Scan(&previousPuzzleUUID)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return previousPuzzleUUID, nil
}

func (s *DBStore) GetAnswer(ctx context.Context, puzzleUUID string) (*macondopb.GameEvent, string, int32, string, *ipc.GameRequest, *entity.SingleRating, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, "", -1, "", nil, nil, err
	}
	defer tx.Rollback(ctx)
	var ans *answer
	var rat *entity.SingleRating
	var afterText string
	var gameId int
	var turnNumber int32

	err = tx.QueryRow(ctx, `SELECT answer, rating, after_text, game_id, turn_number FROM puzzles WHERE uuid = $1`, puzzleUUID).Scan(&ans, &rat, &afterText, &gameId, &turnNumber)
	if err == pgx.ErrNoRows {
		return nil, "", -1, "", nil, nil, entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_ANSWER_PUZZLE_UUID_NOT_FOUND, puzzleUUID)
	}
	if err != nil {
		return nil, "", -1, "", nil, nil, err
	}

	_, req, gameUUID, err := common.GetGameInfo(ctx, tx, gameId)
	if err != nil {
		return nil, "", -1, "", nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, "", -1, "", nil, nil, err
	}
	return answerToGameEvent(ans), gameUUID, turnNumber, afterText, req, rat, nil
}

func (s *DBStore) SubmitAnswer(ctx context.Context, userUUID string, ratingKey entity.VariantKey,
	newUserRating *entity.SingleRating, puzzleUUID string, newPuzzleRating *entity.SingleRating, userIsCorrect bool, showSolution bool) error {

	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	pid, err := common.GetPuzzleDBIDFromUUID(ctx, tx, puzzleUUID)
	if err != nil {
		return err
	}

	uid, err := common.GetUserDBIDFromUUID(ctx, tx, userUUID)
	if err != nil {
		return err
	}

	newCorrectOption := &sql.NullBool{}
	// Consider the puzzle completed if the user
	// has gotten the answer correct or has given up
	// by requesting the solution
	if userIsCorrect || showSolution {
		newCorrectOption.Valid = true
		// Only consider the user correct if they did
		// not request the solution
		newCorrectOption.Bool = userIsCorrect && !showSolution
	}

	if newUserRating != nil && newPuzzleRating != nil {
		result, err := tx.Exec(ctx, `UPDATE puzzles SET rating = $1 WHERE id = $2`, newPuzzleRating, pid)
		if err != nil {
			return err
		}

		rowsAffected := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected != 1 {
			return entity.NewWooglesError(ipc.WooglesError_PUZZLE_SUBMIT_ANSWER_PUZZLE_ID_NOT_FOUND, userUUID, puzzleUUID)
		}

		err = common.UpdateUserRating(ctx, tx, uid, ratingKey, newUserRating)
		if err != nil {
			return err
		}

		attempts := 1

		if showSolution {
			attempts = 0
		}

		result, err = tx.Exec(ctx, `UPDATE puzzle_attempts SET correct = $1, attempts = $2, new_user_rating = $3, new_puzzle_rating = $4, created_at = NOW(), updated_at = NOW() WHERE puzzle_id = $5 AND user_id = $6`,
			newCorrectOption, attempts, newUserRating, newPuzzleRating, pid, uid)

		rowsAffected = result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected != 1 {
			return entity.NewWooglesError(ipc.WooglesError_PUZZLE_SUBMIT_ANSWER_PUZZLE_ATTEMPT_NOT_FOUND, userUUID, puzzleUUID)
		}
	} else {
		// Update the attempt if the puzzle is not complete
		oldCorrectOption := &sql.NullBool{}
		err = tx.QueryRow(ctx, `SELECT correct FROM puzzle_attempts WHERE puzzle_id = $1 AND user_id = $2 FOR UPDATE`, pid, uid).Scan(oldCorrectOption)
		if err != nil {
			return err
		}

		if !oldCorrectOption.Valid {
			result, err := tx.Exec(ctx, `UPDATE puzzle_attempts SET correct = $1 WHERE puzzle_id = $2 AND user_id = $3`, newCorrectOption, pid, uid)
			if err != nil {
				return err
			}

			rowsAffected := result.RowsAffected()
			if err != nil {
				return err
			}
			if rowsAffected != 1 {
				return entity.NewWooglesError(ipc.WooglesError_PUZZLE_SUBMIT_ANSWER_SET_CORRECT, userUUID, puzzleUUID)
			}

			if !showSolution {
				result, err := tx.Exec(ctx, `UPDATE puzzle_attempts SET attempts = attempts + 1, updated_at = NOW() WHERE puzzle_id = $1 AND user_id = $2`, pid, uid)
				if err != nil {
					return err
				}
				rowsAffected := result.RowsAffected()
				if err != nil {
					return err
				}
				if rowsAffected != 1 {
					entity.NewWooglesError(ipc.WooglesError_PUZZLE_SUBMIT_ANSWER_SET_ATTEMPTS, userUUID, puzzleUUID)
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) GetUserRating(ctx context.Context, userID string, ratingKey entity.VariantKey) (*entity.SingleRating, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	uid, err := common.GetUserDBIDFromUUID(ctx, tx, userID)
	if err != nil {
		return nil, err
	}

	initialRating := &entity.SingleRating{
		Rating:            float64(glicko.InitialRating),
		RatingDeviation:   float64(glicko.InitialRatingDeviation),
		Volatility:        glicko.InitialVolatility,
		LastGameTimestamp: time.Now().Unix()}

	sr, err := common.GetUserRating(ctx, tx, uid, ratingKey, initialRating)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return sr, nil
}

func (s *DBStore) SetPuzzleVote(ctx context.Context, userID string, puzzleID string, vote int) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	pid, err := common.GetPuzzleDBIDFromUUID(ctx, tx, puzzleID)
	if err != nil {
		return err
	}

	uid, err := common.GetUserDBIDFromUUID(ctx, tx, userID)
	if err != nil {
		return err
	}

	result, err := tx.Exec(ctx, `INSERT INTO puzzle_votes (puzzle_id, user_id, vote) VALUES ($1, $2, $3) ON CONFLICT (puzzle_id, user_id) DO UPDATE SET vote = $3`, pid, uid, vote)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return entity.NewWooglesError(ipc.WooglesError_PUZZLE_SET_PUZZLE_VOTE_ID_NOT_FOUND, userID, puzzleID)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return err
}

func (s *DBStore) GetAttempts(ctx context.Context, userUUID string, puzzleUUID string) (bool, bool, int32, *bool, time.Time, time.Time, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return false, false, -1, nil, time.Time{}, time.Time{}, err
	}
	defer tx.Rollback(ctx)

	rated, attemptExists, attempts, status, firstAttemptTime, lastAttemptTime, err := getAttempts(ctx, tx, userUUID, puzzleUUID)
	if err != nil {
		return false, false, -1, nil, time.Time{}, time.Time{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, false, -1, nil, time.Time{}, time.Time{}, err
	}

	return rated, attemptExists, attempts, status, firstAttemptTime, lastAttemptTime, nil
}

func (s *DBStore) GetJobInfo(ctx context.Context, genId int) (time.Time, time.Time, time.Duration, *bool, *string, int, int, [][]int, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return time.Time{}, time.Time{}, -1, nil, nil, -1, -1, nil, err
	}
	defer tx.Rollback(ctx)

	createdAtTime := time.Time{}
	completedAtTime := time.Time{}
	fulfilled := &sql.NullBool{}
	errorStatus := &sql.NullString{}
	err = tx.QueryRow(ctx, `SELECT created_at, completed_at, fulfilled, error_status FROM puzzle_generation_logs WHERE id = $1`, genId).Scan(&createdAtTime, &completedAtTime, fulfilled, errorStatus)
	if err == pgx.ErrNoRows {
		return time.Time{}, time.Time{}, -1, nil, nil, -1, -1, nil, fmt.Errorf("row not found while calculating job duration: %d", genId)
	}
	if err != nil {
		return time.Time{}, time.Time{}, -1, nil, nil, -1, -1, nil, err
	}

	fo := false
	fulfilledOption := &fo
	if fulfilled.Valid {
		*fulfilledOption = fulfilled.Bool
	} else {
		fulfilledOption = nil
	}
	eso := ""
	errorStatusOption := &eso
	if errorStatus.Valid {
		*errorStatusOption = errorStatus.String
	} else {
		errorStatusOption = nil
	}

	rows, err := tx.Query(ctx, `SELECT bucket_index, COUNT(*) FROM puzzles WHERE generation_id = $1 GROUP BY bucket_index ORDER BY bucket_index ASC`, genId)
	if err == pgx.ErrNoRows {
		return time.Time{}, time.Time{}, -1, nil, nil, -1, -1, nil, fmt.Errorf("no rows found for generation_id: %d", genId)
	}
	if err != nil {
		return time.Time{}, time.Time{}, -1, nil, nil, -1, -1, nil, err
	}
	bucketResults := [][]int{}
	for rows.Next() {
		var bucketIndex int
		var numPuzzles int
		if err := rows.Scan(&bucketIndex, &numPuzzles); err != nil {
			rows.Close()
			return time.Time{}, time.Time{}, -1, nil, nil, -1, -1, nil, err
		}
		bucketResults = append(bucketResults, []int{bucketIndex, numPuzzles})
	}

	rows.Close()

	numTotalPuzzles := 0
	breakdowns := [][]int{}
	for i := 0; i < len(bucketResults); i++ {
		bucketIndex := bucketResults[i][0]
		numPuzzles := bucketResults[i][1]
		var numGames int
		err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM (SELECT DISTINCT game_id FROM puzzles WHERE generation_id = $1 AND bucket_index = $2) as unique_games`, genId, bucketIndex).Scan(&numGames)
		if err != nil {
			return time.Time{}, time.Time{}, -1, nil, nil, -1, -1, nil, err
		}
		breakdowns = append(breakdowns, []int{bucketIndex, numPuzzles, numGames})
		numTotalPuzzles += numPuzzles
	}

	var numTotalGames int
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM (SELECT DISTINCT game_id FROM puzzles WHERE generation_id = $1) as unique_games`, genId).Scan(&numTotalGames)
	if err != nil {
		return time.Time{}, time.Time{}, -1, nil, nil, -1, -1, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return time.Time{}, time.Time{}, -1, nil, nil, -1, -1, nil, err
	}

	return createdAtTime, completedAtTime, completedAtTime.Sub(createdAtTime), fulfilledOption, errorStatusOption, numTotalPuzzles, numTotalGames, breakdowns, nil
}

func (s *DBStore) GetPotentialPuzzleGames(ctx context.Context, limit int, offset int) (common.RowIterator, error) {
	// Only look at the last 20K games played when this query is run.
	rows, err := s.dbPool.Query(ctx,
		`SELECT games.uuid FROM (select * from games order by id desc limit 20000) games
		left join puzzles on puzzles.game_id = games.id
		where puzzles.id is null AND (stats->'d1'->'Challenged Phonies'->'t' = '0')
		AND (stats->'d2'->'Challenged Phonies'->'t' = '0')
		AND (stats->'d1'->'Unchallenged Phonies'->'t' = '0')
		AND (stats->'d2'->'Unchallenged Phonies'->'t' = '0')
		AND game_end_reason not in ($1, $2, $3)
		ORDER BY games.id DESC LIMIT $4 OFFSET $5`,
		ipc.GameEndReason_NONE, ipc.GameEndReason_ABORTED, ipc.GameEndReason_CANCELLED, limit, offset)

	if err != nil {
		return nil, err
	}
	return rows, nil
}

func getAttempts(ctx context.Context, tx pgx.Tx, userUUID string, puzzleUUID string) (bool, bool, int32, *bool, time.Time, time.Time, error) {
	pid, err := common.GetPuzzleDBIDFromUUID(ctx, tx, puzzleUUID)
	if err != nil {
		return false, false, -1, nil, time.Time{}, time.Time{}, err
	}

	uid, err := common.GetUserDBIDFromUUID(ctx, tx, userUUID)
	if err != nil {
		return false, false, -1, nil, time.Time{}, time.Time{}, err
	}

	var attempts int32
	correct := &sql.NullBool{}
	var firstAttemptTime time.Time
	var lastAttemptTime time.Time

	err = tx.QueryRow(ctx, `SELECT attempts, correct, created_at, updated_at FROM puzzle_attempts WHERE user_id = $1 AND puzzle_id = $2`, uid, pid).Scan(&attempts, correct, &firstAttemptTime, &lastAttemptTime)
	if err == pgx.ErrNoRows {
		return false, false, 0, nil, time.Time{}, time.Time{}, nil
	}
	if err != nil {
		return false, false, -1, nil, time.Time{}, time.Time{}, err
	}

	var userWasCorrect bool
	status := &userWasCorrect

	if correct.Valid {
		*status = correct.Bool
	} else {
		status = nil
	}

	return attempts != 0 || status != nil, true, attempts, status, firstAttemptTime, lastAttemptTime, nil
}

func getNextPuzzleId(ctx context.Context, tx pgx.Tx, userUUID string, lexicon string) (string, error) {
	uid, err := common.GetUserDBIDFromUUID(ctx, tx, userUUID)
	if err != nil {
		return "", err
	}

	randomId, err := getRandomPuzzleDBID(ctx, tx)
	if err != nil {
		return "", err
	}
	if !randomId.Valid {
		return "", entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_RANDOM_PUZZLE_ID_NOT_FOUND, userUUID, lexicon)
	}

	var puzzleUUID string
	err = tx.QueryRow(ctx, `SELECT uuid FROM puzzles WHERE lexicon = $1 AND id NOT IN (SELECT puzzle_id FROM puzzle_attempts WHERE user_id = $2) AND id > $3 ORDER BY id LIMIT 1`, lexicon, uid, randomId.Int64).Scan(&puzzleUUID)
	if err == pgx.ErrNoRows {
		// Try again, but looking before the id instead
		err = tx.QueryRow(ctx, `SELECT uuid FROM puzzles WHERE lexicon = $1 AND id NOT IN (SELECT puzzle_id FROM puzzle_attempts WHERE user_id = $2) AND id <= $3 ORDER BY id DESC LIMIT 1`, lexicon, uid, randomId.Int64).Scan(&puzzleUUID)
	}
	// The user has answered all available puzzles.
	// Return any random puzzle

	if err == pgx.ErrNoRows {
		puzzleUUID, err = getRandomPuzzleUUID(ctx, tx, lexicon, randomId)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	return puzzleUUID, nil
}

func getRandomPuzzleUUID(ctx context.Context, tx pgx.Tx, lexicon string, randomId *sql.NullInt64) (string, error) {
	var err error
	if randomId == nil {
		randomId, err = getRandomPuzzleDBID(ctx, tx)
		if err != nil {
			return "", err
		}
	}
	var puzzleUUID string
	err = tx.QueryRow(ctx, `SELECT uuid FROM puzzles WHERE lexicon = $1 AND id > $2 ORDER BY id LIMIT 1`, lexicon, randomId.Int64).Scan(&puzzleUUID)
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `SELECT uuid FROM puzzles WHERE lexicon = $1 AND id <= $2 ORDER BY id DESC LIMIT 1`, lexicon, randomId.Int64).Scan(&puzzleUUID)
	}
	if err == pgx.ErrNoRows {
		return "", entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_RANDOM_PUZZLE_NOT_FOUND, "", lexicon)
	} else if err != nil {
		return "", err
	}
	return puzzleUUID, nil
}

func getRandomPuzzleDBID(ctx context.Context, tx pgx.Tx) (*sql.NullInt64, error) {
	var id sql.NullInt64
	err := tx.QueryRow(ctx, "SELECT FLOOR(RANDOM() * MAX(id)) FROM puzzles").Scan(&id)
	return &id, err
}

func sanitizeHistory(hist *macondopb.GameHistory) {
	playerSanitizationMap := map[string]string{
		hist.Players[0].Nickname: commontest.DefaultPlayerOneInfo.Nickname,
		hist.Players[1].Nickname: commontest.DefaultPlayerTwoInfo.Nickname,
	}
	for _, evt := range hist.Events {
		evt.Nickname = playerSanitizationMap[evt.Nickname]
	}

	hist.Players[0].Nickname = commontest.DefaultPlayerOneInfo.Nickname
	hist.Players[0].RealName = commontest.DefaultPlayerOneInfo.FullName
	hist.Players[0].UserId = commontest.DefaultPlayerOneInfo.UserId
	hist.Players[1].Nickname = commontest.DefaultPlayerTwoInfo.Nickname
	hist.Players[1].RealName = commontest.DefaultPlayerTwoInfo.FullName
	hist.Players[1].UserId = commontest.DefaultPlayerTwoInfo.UserId
}

func (a *answer) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *answer) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

func gameEventToAnswer(evt *macondopb.GameEvent) *answer {
	return &answer{
		EventType:   int32(evt.Type),
		Row:         evt.Row,
		Column:      evt.Column,
		Direction:   int32(evt.Direction),
		Position:    evt.Position,
		PlayedTiles: evt.PlayedTiles,
		Exchanged:   evt.Exchanged,
		Score:       evt.Score,
		IsBingo:     evt.IsBingo,
	}
}

func answerToGameEvent(a *answer) *macondopb.GameEvent {
	return &macondopb.GameEvent{
		Type:        macondopb.GameEvent_Type(a.EventType),
		Row:         a.Row,
		Column:      a.Column,
		Direction:   macondopb.GameEvent_Direction(a.Direction),
		Position:    a.Position,
		PlayedTiles: a.PlayedTiles,
		Exchanged:   a.Exchanged,
		Score:       a.Score,
		IsBingo:     a.IsBingo,
	}
}
