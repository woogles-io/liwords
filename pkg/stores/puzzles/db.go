package puzzles

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/glicko"
	"github.com/domino14/liwords/pkg/stores/common"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lithammer/shortuuid"
)

var InitialPuzzleRating = 1500
var InitialPuzzleRatingDeviation = 400
var InitialPuzzleVolatility = 0.06

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

func (s *DBStore) CreatePuzzle(ctx context.Context, gameUUID string, turnNumber int32, answer *macondopb.GameEvent, authorUUID string,
	lexicon string, beforeText string, afterText string, tags []macondopb.PuzzleTag) error {

	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	gameID, err := common.GetGameDBIDFromUUID(ctx, tx, gameUUID)
	if err != nil {
		return err
	}

	// XXX: This is a bit hacky and can probably be improved, but
	// the insert value for author_id needs to be either a valid
	// author id or a nil value

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
		Rating:            float64(InitialPuzzleRating),
		RatingDeviation:   float64(InitialPuzzleRatingDeviation),
		Volatility:        InitialPuzzleVolatility,
		LastGameTimestamp: time.Now().Unix()}

	uuid := shortuuid.New()

	var id int
	err = tx.QueryRow(ctx, `INSERT INTO puzzles (uuid, game_id, turn_number, author_id, answer, lexicon, before_text, after_text, rating, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW()) RETURNING id`,
		uuid, gameID, turnNumber, authorId, gameEventToAnswer(answer), lexicon, beforeText, afterText, newRating).Scan(&id)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		_, err := tx.Exec(ctx, `INSERT INTO puzzle_tags(tag_id, puzzle_id) VALUES ($1, $2)`, tag+1, id)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return err
}

func (s *DBStore) GetRandomUnansweredPuzzleIdForUser(ctx context.Context, userUUID string, lexicon string) (string, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	uid, err := common.GetUserDBIDFromUUID(ctx, tx, userUUID)
	if err != nil {
		return "", err
	}

	randomId, err := getRandomPuzzleId(ctx, tx)
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
		err = tx.QueryRow(ctx, `SELECT uuid FROM puzzles WHERE lexicon = $1 AND id > $2 ORDER BY id LIMIT 1`, lexicon, randomId.Int64).Scan(&puzzleUUID)
	}
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `SELECT uuid FROM puzzles WHERE lexicon = $1 AND id <= $2 ORDER BY id DESC LIMIT 1`, lexicon, randomId.Int64).Scan(&puzzleUUID)
	}
	if err == pgx.ErrNoRows {
		return "", entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_RANDOM_PUZZLE_NOT_FOUND, userUUID, lexicon)
	} else if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return puzzleUUID, nil
}

func (s *DBStore) GetPuzzle(ctx context.Context, userUUID string, puzzleUUID string) (*macondopb.GameHistory, string, int32, *bool, time.Time, time.Time, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, "", -1, nil, time.Time{}, time.Time{}, err
	}
	defer tx.Rollback(ctx)

	attempts := int32(0)
	var userIsCorrect *bool
	firstAttemptTime := time.Time{}
	lastAttemptTime := time.Time{}

	if userUUID != "" {
		attempts, userIsCorrect, firstAttemptTime, lastAttemptTime, err = s.GetAttempts(ctx, userUUID, puzzleUUID)
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

	return hist, beforeText, attempts, userIsCorrect, firstAttemptTime, lastAttemptTime, nil
}

func (s *DBStore) GetPreviousPuzzle(ctx context.Context, userUUID string, puzzleUUID string) (string, error) {
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

	var previousPuzzleUUID string
	var currentPuzzleUpdatedTime time.Time
	err = tx.QueryRow(ctx, `SELECT updated_at FROM puzzle_attempts WHERE puzzle_id = $1`, pid).Scan(&currentPuzzleUpdatedTime)
	if err == pgx.ErrNoRows {
		// Current puzzle has not been attempted, just select the most recently attempted puzzle
		var mostRecentPid int
		err = tx.QueryRow(ctx, `SELECT puzzle_id FROM puzzle_attempts WHERE user_id = $1 ORDER BY updated_at DESC LIMIT 1`, uid).Scan(&mostRecentPid)
		if err == pgx.ErrNoRows {
			return "", entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_NO_ATTEMPTS, userUUID, puzzleUUID)
		}
		if err != nil {
			return "", err
		}

		err = tx.QueryRow(ctx, `SELECT uuid FROM puzzles WHERE id = $1`, mostRecentPid).Scan(&previousPuzzleUUID)
		if err == pgx.ErrNoRows {
			return "", entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_NO_ATTEMPTS_ID_NOT_FOUND, strconv.Itoa(mostRecentPid), userUUID, puzzleUUID)
		}
		if err != nil {
			return "", err
		}
	}
	if err != nil {
		return "", err
	}

	if previousPuzzleUUID == "" {
		var previousPid int
		err = tx.QueryRow(ctx, `SELECT puzzle_id FROM puzzle_attempts WHERE updated_at <= $1 AND puzzle_id != $2 ORDER BY updated_at DESC LIMIT 1`, currentPuzzleUpdatedTime, pid).Scan(&previousPid)
		if err == pgx.ErrNoRows {
			return "", entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_ATTEMPT_NOT_FOUND, userUUID, puzzleUUID)
		}
		if err != nil {
			return "", err
		}

		err = tx.QueryRow(ctx, `SELECT uuid FROM puzzles WHERE id = $1`, previousPid).Scan(&previousPuzzleUUID)
		if err == pgx.ErrNoRows {
			return "", entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_ID_NOT_FOUND, strconv.Itoa(previousPid), userUUID, puzzleUUID)
		}
		if err != nil {
			return "", err
		}
	}

	return previousPuzzleUUID, nil
}

func (s *DBStore) GetAnswer(ctx context.Context, puzzleUUID string) (*macondopb.GameEvent, string, string, *ipc.GameRequest, *entity.SingleRating, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, "", "", nil, nil, err
	}
	defer tx.Rollback(ctx)
	var ans *answer
	var rat *entity.SingleRating
	var afterText string
	var gameId int

	err = tx.QueryRow(ctx, `SELECT answer, rating, after_text, game_id FROM puzzles WHERE uuid = $1`, puzzleUUID).Scan(&ans, &rat, &afterText, &gameId)
	if err == pgx.ErrNoRows {
		return nil, "", "", nil, nil, entity.NewWooglesError(ipc.WooglesError_PUZZLE_GET_ANSWER_PUZZLE_UUID_NOT_FOUND, puzzleUUID)
	}
	if err != nil {
		return nil, "", "", nil, nil, err
	}

	_, req, gameUUID, err := common.GetGameInfo(ctx, tx, gameId)
	if err != nil {
		return nil, "", "", nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, "", "", nil, nil, err
	}
	return answerToGameEvent(ans), gameUUID, afterText, req, rat, nil
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

		_, err = tx.Exec(ctx, `INSERT INTO puzzle_attempts (puzzle_id, user_id, correct, attempts, new_user_rating, new_puzzle_rating, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`,
			pid, uid, newCorrectOption, attempts, newUserRating, newPuzzleRating)
		if err != nil {
			return err
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
				entity.NewWooglesError(ipc.WooglesError_PUZZLE_SUBMIT_ANSWER_SET_CORRECT, userUUID, puzzleUUID)
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

func (s *DBStore) GetAttempts(ctx context.Context, userUUID string, puzzleUUID string) (int32, *bool, time.Time, time.Time, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return -1, nil, time.Time{}, time.Time{}, err
	}
	defer tx.Rollback(ctx)

	pid, err := common.GetPuzzleDBIDFromUUID(ctx, tx, puzzleUUID)
	if err != nil {
		return -1, nil, time.Time{}, time.Time{}, err
	}

	uid, err := common.GetUserDBIDFromUUID(ctx, tx, userUUID)
	if err != nil {
		return -1, nil, time.Time{}, time.Time{}, err
	}

	var attempts int32
	correct := &sql.NullBool{}
	var firstAttemptTime time.Time
	var lastAttemptTime time.Time

	err = tx.QueryRow(ctx, `SELECT attempts, correct, created_at, updated_at FROM puzzle_attempts WHERE user_id = $1 AND puzzle_id = $2`, uid, pid).Scan(&attempts, correct, &firstAttemptTime, &lastAttemptTime)
	if err == pgx.ErrNoRows {
		return 0, nil, time.Time{}, time.Time{}, nil
	}
	if err != nil {
		return -1, nil, time.Time{}, time.Time{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return -1, nil, time.Time{}, time.Time{}, err
	}

	var userWasCorrect bool
	userWasCorrectPtr := &userWasCorrect

	if correct.Valid {
		*userWasCorrectPtr = correct.Bool
	} else {
		userWasCorrectPtr = nil
	}

	return attempts, userWasCorrectPtr, firstAttemptTime, lastAttemptTime, nil
}

func getRandomPuzzleId(ctx context.Context, tx pgx.Tx) (sql.NullInt64, error) {
	var id sql.NullInt64
	err := tx.QueryRow(ctx, "SELECT FLOOR(RANDOM() * MAX(id)) FROM puzzles").Scan(&id)
	return id, err
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
