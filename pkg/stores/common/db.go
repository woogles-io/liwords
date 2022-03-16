package common

import (
	"context"
	"database/sql"
	"fmt"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/proto"
)

func GetUserDBIDFromUUID(ctx context.Context, tx *sql.Tx, uuid string) (uint, error) {
	var id uint
	err := tx.QueryRowContext(ctx, "SELECT id FROM users WHERE uuid = $1", uuid).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("cannot get id from uuid %s: no rows for table users", uuid)
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetGameDBIDFromUUID(ctx context.Context, tx *sql.Tx, uuid string) (uint, error) {
	var id uint
	err := tx.QueryRowContext(ctx, "SELECT id FROM games WHERE uuid = $1", uuid).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("cannot get id from uuid %s: no rows for table games", uuid)
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetPuzzleDBIDFromUUID(ctx context.Context, tx *sql.Tx, uuid string) (uint, error) {
	var id uint
	err := tx.QueryRowContext(ctx, "SELECT id FROM puzzles WHERE uuid = $1", uuid).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("cannot get id from uuid %s: no rows for table puzzles", uuid)
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetGameInfo(ctx context.Context, tx *sql.Tx, gameId int) (*macondopb.GameHistory, *ipc.GameRequest, string, error) {
	var uuid string
	var historyBytes []byte
	var requestBytes []byte
	err := tx.QueryRowContext(ctx, `SELECT uuid, history, request FROM games WHERE id = $1`, gameId).Scan(&uuid, &historyBytes, &requestBytes)
	if err == sql.ErrNoRows {
		return nil, nil, "", fmt.Errorf("no rows for games table: %d", gameId)
	}
	if err != nil {
		return nil, nil, "", err
	}

	hist := &macondopb.GameHistory{}
	err = proto.Unmarshal(historyBytes, hist)
	if err != nil {
		return nil, nil, "", err
	}

	req := &ipc.GameRequest{}
	err = proto.Unmarshal(requestBytes, req)
	if err != nil {
		return nil, nil, "", err
	}

	return hist, req, uuid, nil
}

func InitializeUserRating(ctx context.Context, tx *sql.Tx, userId uint) error {
	var ratings *entity.Ratings
	err := tx.QueryRowContext(ctx, `SELECT ratings FROM profiles WHERE user_id = $1`, userId).Scan(&ratings)
	if err == sql.ErrNoRows {
		return fmt.Errorf("profile not found for user_id: %d", userId)
	}
	if err != nil {
		return err
	}

	if ratings.Data == nil {
		_, err := tx.ExecContext(ctx, `UPDATE profiles SET ratings = jsonb_set(ratings, '{Data}', jsonb '{}') WHERE user_id = $1;`, userId)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetUserRating(ctx context.Context, tx *sql.Tx, userId uint, ratingKey entity.VariantKey, defaultRating *entity.SingleRating) (*entity.SingleRating, error) {
	err := InitializeUserRating(ctx, tx, userId)
	if err != nil {
		return nil, err
	}

	var sr *entity.SingleRating
	err = tx.QueryRowContext(ctx, "select ratings->'Data'->$1 from profiles where user_id = $2", ratingKey, userId).Scan(&sr)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("profile not found for user_id: %d", userId)
	}
	if err != nil {
		return nil, err
	}

	if sr == nil {
		sr = defaultRating
		err = UpdateUserRating(ctx, tx, userId, ratingKey, sr)
		if err != nil {
			return nil, err
		}
	}

	return sr, nil
}

func UpdateUserRating(ctx context.Context, tx *sql.Tx, userId uint, ratingKey entity.VariantKey, newRating *entity.SingleRating) error {
	err := InitializeUserRating(ctx, tx, userId)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "update profiles set ratings = jsonb_set(ratings, array['Data', $1], $2) where user_id = $3", ratingKey, newRating, userId)
	if err != nil {
		return err
	}
	return nil
}
