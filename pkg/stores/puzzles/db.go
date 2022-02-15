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

	gameID, err := common.GetDBIDFromUUID(ctx, s.db, "games", gameUUID)
	if err != nil {
		return err
	}

	// XXX: This is a bit hacky and can probably be improved, but
	// the insert value for author_id needs to be either a valid
	// author id or a nil value

	authorId := uint(0)
	authorIdptr := &authorId
	if authorUUID != "" {
		aid, err := common.GetDBIDFromUUID(ctx, s.db, "users", authorUUID)
		if err != nil {
			return err
		}
		authorId = aid
	} else {
		authorIdptr = nil
	}

	newRating := &entity.SingleRating{
		Rating:            float64(InitialPuzzleRating),
		RatingDeviation:   float64(InitialPuzzleRatingDeviation),
		Volatility:        InitialPuzzleVolatility,
		LastGameTimestamp: time.Now().Unix()}

	uuid := shortuuid.New()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	err = func() error {
		var id int
		err = s.db.QueryRowContext(ctx, `INSERT INTO puzzles (uuid, game_id, turn_number, author_id, answer, before_text, after_text, rating, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()) RETURNING id`,
			uuid, gameID, turnNumber, authorIdptr, gameEventToAnswer(answer), beforeText, afterText, newRating).Scan(&id)
		if err != nil {
			return err
		}

		for _, tag := range tags {
			_, err := s.db.ExecContext(ctx, `INSERT INTO puzzle_tags(tag_id, puzzle_id) VALUES ($1, $2)`, tag+1, id)
			if err != nil {
				return err
			}
		}
		return nil
	}()

	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return err
}

func (s *DBStore) GetRandomUnansweredPuzzleIdForUser(ctx context.Context, userId string) (string, error) {
	userDBID, err := common.GetDBIDFromUUID(ctx, s.db, "users", userId)
	if err != nil {
		return "", err
	}
	var puzzleId string
	err = s.db.QueryRowContext(ctx, `SELECT uuid FROM puzzles WHERE id NOT IN (SELECT puzzle_id FROM puzzle_attempts WHERE user_id = $1) ORDER BY RANDOM()`, userDBID).Scan(&puzzleId)
	if err == sql.ErrNoRows {
		// User has answered all available puzzles, return a random seen puzzle
		err = s.db.QueryRowContext(ctx, `SELECT uuid FROM puzzles ORDER BY RANDOM()`).Scan(&puzzleId)
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no puzzles found for user: %s", userId)
		}
		if err != nil {
			return "", err
		}
	}
	if err != nil {
		return "", err
	}
	return puzzleId, nil
}

func (s *DBStore) GetPuzzle(ctx context.Context, puzzleId string) (string, *macondopb.GameHistory, string, error) {
	var gameId int
	var turnNumber int
	var beforeText string

	err := s.db.QueryRowContext(ctx, `SELECT game_id, turn_number, before_text FROM puzzles WHERE uuid = $1`, puzzleId).Scan(&gameId, &turnNumber, &beforeText)
	if err == sql.ErrNoRows {
		return "", nil, "", fmt.Errorf("puzzle not found: %s", puzzleId)
	}
	if err != nil {
		return "", nil, "", err
	}
	hist, _, gameUUID, err := common.GetGameInfo(ctx, s.db, gameId)
	if err != nil {
		return "", nil, "", err
	}

	hist.Events = hist.Events[:turnNumber]

	return gameUUID, hist, beforeText, nil
}

func (s *DBStore) GetAnswer(ctx context.Context, puzzleId string) (*macondopb.GameEvent, string, *ipc.GameRequest, *entity.SingleRating, error) {
	var ans *answer
	var rat *entity.SingleRating
	var afterText string
	var gameId int

	err := s.db.QueryRowContext(ctx, `SELECT answer, rating, after_text, game_id FROM puzzles WHERE uuid = $1`, puzzleId).Scan(&ans, &rat, &afterText, &gameId)
	if err == sql.ErrNoRows {
		return nil, "", nil, nil, fmt.Errorf("puzzle not found: %s", puzzleId)
	}
	if err != nil {
		return nil, "", nil, nil, err
	}

	_, req, _, err := common.GetGameInfo(ctx, s.db, gameId)
	if err != nil {
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

	err = func() error {
		pid, err := common.GetDBIDFromUUID(ctx, s.db, "puzzles", puzzleId)
		if err != nil {
			return err
		}

		uid, err := common.GetDBIDFromUUID(ctx, s.db, "users", userId)
		if err != nil {
			return err
		}

		if newUserRating != nil && newPuzzleRating != nil {
			_, err = tx.ExecContext(ctx, `UPDATE puzzles SET rating = $1 WHERE id = $2`, newPuzzleRating, pid)
			if err != nil {
				return err
			}

			err = common.UpdateUserRating(ctx, s.db, tx, uid, ratingKey, newUserRating)
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
			var attempts int
			err = s.db.QueryRowContext(ctx, `SELECT correct, attempts FROM puzzle_attempts WHERE puzzle_id = $1 AND user_id = $2`, pid, uid).Scan(&alreadyCorrect, &attempts)
			if err != nil {
				return err
			}

			if !alreadyCorrect {
				_, err = tx.ExecContext(ctx, `UPDATE puzzle_attempts SET attempts = $1, correct = $2 WHERE puzzle_id = $3 AND user_id = $4`, attempts+1, correct, pid, uid)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}()

	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) HasUserAttemptedPuzzle(ctx context.Context, userID string, puzzleID string) (bool, error) {
	pid, err := common.GetDBIDFromUUID(ctx, s.db, "puzzles", puzzleID)
	if err != nil {
		return false, err
	}

	uid, err := common.GetDBIDFromUUID(ctx, s.db, "users", userID)
	if err != nil {
		return false, err
	}

	var seen bool
	err = s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM puzzle_attempts WHERE puzzle_id = $1 AND user_id = $2)`, pid, uid).Scan(&seen)
	if err != nil {
		return false, err
	}
	return seen, nil
}

// XXX: Pass through function until we figure out the stores solution
func (s *DBStore) GetUserRating(ctx context.Context, userID string, ratingKey entity.VariantKey) (*entity.SingleRating, error) {
	uid, err := common.GetDBIDFromUUID(ctx, s.db, "users", userID)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	initialRating := &entity.SingleRating{
		Rating:            float64(glicko.InitialRating),
		RatingDeviation:   float64(glicko.InitialRatingDeviation),
		Volatility:        glicko.InitialVolatility,
		LastGameTimestamp: time.Now().Unix()}

	sr, err := common.GetUserRating(ctx, s.db, tx, uid, ratingKey, initialRating)

	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return sr, nil
}

func (s *DBStore) SetPuzzleVote(ctx context.Context, userID string, puzzleID string, vote int) error {
	pid, err := common.GetDBIDFromUUID(ctx, s.db, "puzzles", puzzleID)
	if err != nil {
		return err
	}

	uid, err := common.GetDBIDFromUUID(ctx, s.db, "users", userID)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `INSERT INTO puzzle_votes (puzzle_id, user_id, vote) VALUES ($1, $2, $3) ON CONFLICT (puzzle_id, user_id) DO UPDATE SET vote = $3`, pid, uid, vote)
	return err
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
