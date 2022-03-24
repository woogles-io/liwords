package puzzles

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/glicko"
	"github.com/domino14/liwords/pkg/stores/common"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/lithammer/shortuuid"
)

var InitialPuzzleRating = 1500
var InitialPuzzleRatingDeviation = 400
var InitialPuzzleVolatility = 0.06

type DBStore struct {
	db *sql.DB
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

func NewDBStore(db *sql.DB) (*DBStore, error) {
	return &DBStore{db: db}, nil
}

func (s *DBStore) CreatePuzzle(ctx context.Context, gameUUID string, turnNumber int32, answer *macondopb.GameEvent, authorUUID string,
	beforeText string, afterText string, tags []macondopb.PuzzleTag) error {

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

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

	err = func() error {
		var id int
		err = tx.QueryRowContext(ctx, `INSERT INTO puzzles (uuid, game_id, turn_number, author_id, answer, before_text, after_text, rating, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()) RETURNING id`,
			uuid, gameID, turnNumber, authorId, gameEventToAnswer(answer), beforeText, afterText, newRating).Scan(&id)
		if err != nil {
			return err
		}

		for _, tag := range tags {
			_, err := tx.ExecContext(ctx, `INSERT INTO puzzle_tags(tag_id, puzzle_id) VALUES ($1, $2)`, tag+1, id)
			if err != nil {
				return err
			}
		}
		return nil
	}()

	if err := tx.Commit(); err != nil {
		return err
	}

	return err
}

func (s *DBStore) GetRandomUnansweredPuzzleIdForUser(ctx context.Context, userId string) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	userDBID, err := common.GetUserDBIDFromUUID(ctx, tx, userId)
	if err != nil {
		return "", err
	}

	randomId, err := getRandomPuzzleId(ctx, tx)
	if err != nil {
		return "", err
	}
	if !randomId.Valid {
		return "", fmt.Errorf("no puzzle row id found for user: %s", userId)
	}

	var puzzleId string
	err = tx.QueryRowContext(ctx, `SELECT uuid FROM puzzles WHERE id NOT IN (SELECT puzzle_id FROM puzzle_attempts WHERE user_id = $1) AND id > $2 ORDER BY id LIMIT 1`, userDBID, randomId.Int64).Scan(&puzzleId)
	if err == sql.ErrNoRows {
		// Try again, but looking before the id instead
		err = tx.QueryRowContext(ctx, `SELECT uuid FROM puzzles WHERE id NOT IN (SELECT puzzle_id FROM puzzle_attempts WHERE user_id = $1) AND id <= $2 ORDER BY id DESC LIMIT 1`, userDBID, randomId.Int64).Scan(&puzzleId)
	}
	if err == sql.ErrNoRows {
		// The user has answered all available puzzles.
		// Return any random puzzle
		err = tx.QueryRowContext(ctx, `SELECT uuid FROM puzzles WHERE id > $1 ORDER BY id LIMIT 1`, randomId.Int64).Scan(&puzzleId)
	}
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("no puzzles found for user: %s", userId)
	} else if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return puzzleId, nil
}

func (s *DBStore) GetPuzzle(ctx context.Context, puzzleId string) (string, *macondopb.GameHistory, string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", nil, "", err
	}
	defer tx.Rollback()

	var gameId int
	var turnNumber int
	var beforeText string

	err = tx.QueryRowContext(ctx, `SELECT game_id, turn_number, before_text FROM puzzles WHERE uuid = $1`, puzzleId).Scan(&gameId, &turnNumber, &beforeText)
	if err == sql.ErrNoRows {
		return "", nil, "", fmt.Errorf("puzzle not found: %s", puzzleId)
	}
	if err != nil {
		return "", nil, "", err
	}
	hist, _, gameUUID, err := common.GetGameInfo(ctx, tx, gameId)
	if err != nil {
		return "", nil, "", err
	}

	hist.Events = hist.Events[:turnNumber]
	if err := tx.Commit(); err != nil {
		return "", nil, "", err
	}

	return gameUUID, hist, beforeText, nil
}

func (s *DBStore) GetAnswer(ctx context.Context, puzzleId string) (*macondopb.GameEvent, string, *ipc.GameRequest, *entity.SingleRating, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", nil, nil, err
	}
	defer tx.Rollback()
	var ans *answer
	var rat *entity.SingleRating
	var afterText string
	var gameId int

	err = tx.QueryRowContext(ctx, `SELECT answer, rating, after_text, game_id FROM puzzles WHERE uuid = $1`, puzzleId).Scan(&ans, &rat, &afterText, &gameId)
	if err == sql.ErrNoRows {
		return nil, "", nil, nil, fmt.Errorf("puzzle not found: %s", puzzleId)
	}
	if err != nil {
		return nil, "", nil, nil, err
	}

	_, req, _, err := common.GetGameInfo(ctx, tx, gameId)
	if err != nil {
		return nil, "", nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, "", nil, nil, err
	}
	return answerToGameEvent(ans), afterText, req, rat, nil
}

func (s *DBStore) AnswerPuzzle(ctx context.Context, userId string, ratingKey entity.VariantKey,
	newUserRating *entity.SingleRating, puzzleId string, newPuzzleRating *entity.SingleRating, correct bool) error {

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	pid, err := common.GetPuzzleDBIDFromUUID(ctx, tx, puzzleId)
	if err != nil {
		return err
	}

	uid, err := common.GetUserDBIDFromUUID(ctx, tx, userId)
	if err != nil {
		return err
	}

	if newUserRating != nil && newPuzzleRating != nil {
		result, err := tx.ExecContext(ctx, `UPDATE puzzles SET rating = $1 WHERE id = $2`, newPuzzleRating, pid)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected != 1 {
			return fmt.Errorf("not exactly one row affected when setting puzzle rating: %d, %d", pid, rowsAffected)
		}

		err = common.UpdateUserRating(ctx, tx, uid, ratingKey, newUserRating)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `INSERT INTO puzzle_attempts (puzzle_id, user_id, attempts, correct, new_user_rating, new_puzzle_rating, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
			pid, uid, 1, correct, newUserRating, newPuzzleRating)
		if err != nil {
			return err
		}
	} else {
		// Update the attempt if another incorrect answer was given
		var alreadyCorrect bool
		err = tx.QueryRowContext(ctx, `SELECT correct FROM puzzle_attempts WHERE puzzle_id = $1 AND user_id = $2 FOR UPDATE`, pid, uid).Scan(&alreadyCorrect)
		if err != nil {
			return err
		}

		if !alreadyCorrect {
			result, err := tx.ExecContext(ctx, `UPDATE puzzle_attempts SET attempts = attempts + 1, correct = $1 WHERE puzzle_id = $2 AND user_id = $3`, correct, pid, uid)
			if err != nil {
				return err
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if rowsAffected != 1 {
				return fmt.Errorf("not exactly one row affected when updating puzzle attempt: %d, %d, %d", pid, uid, rowsAffected)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) HasUserAttemptedPuzzle(ctx context.Context, userID string, puzzleID string) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	pid, err := common.GetPuzzleDBIDFromUUID(ctx, tx, puzzleID)
	if err != nil {
		return false, err
	}

	uid, err := common.GetUserDBIDFromUUID(ctx, tx, userID)
	if err != nil {
		return false, err
	}

	var seen bool
	err = s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM puzzle_attempts WHERE puzzle_id = $1 AND user_id = $2)`, pid, uid).Scan(&seen)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}

	return seen, nil
}

func (s *DBStore) GetUserRating(ctx context.Context, userID string, ratingKey entity.VariantKey) (*entity.SingleRating, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

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

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return sr, nil
}

func (s *DBStore) SetPuzzleVote(ctx context.Context, userID string, puzzleID string, vote int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	pid, err := common.GetPuzzleDBIDFromUUID(ctx, tx, puzzleID)
	if err != nil {
		return err
	}

	uid, err := common.GetUserDBIDFromUUID(ctx, tx, userID)
	if err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `INSERT INTO puzzle_votes (puzzle_id, user_id, vote) VALUES ($1, $2, $3) ON CONFLICT (puzzle_id, user_id) DO UPDATE SET vote = $3`, pid, uid, vote)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return fmt.Errorf("not exactly one row affected when setting puzzle vote: %d, %d, %d", pid, uid, rowsAffected)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return err
}

func getRandomPuzzleId(ctx context.Context, tx *sql.Tx) (sql.NullInt64, error) {
	var id sql.NullInt64
	err := tx.QueryRowContext(ctx, "SELECT FLOOR(RANDOM() * MAX(id)) FROM puzzles").Scan(&id)
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
