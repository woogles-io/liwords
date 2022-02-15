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

func GetDBIDFromUUID(ctx context.Context, db *sql.DB, table string, uuid string) (uint, error) {
	var id uint
	// XXX: This sprintf is slightly less preferred, imo. I could not figure out how to
	// get the driver to accept something like 'FROM $1 WHERE'
	err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT id FROM %s WHERE uuid = $1", table), uuid).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("cannot get id from uuid %s: no rows for table %s", uuid, table)
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetGameInfo(ctx context.Context, db *sql.DB, gameId int) (*macondopb.GameHistory, *ipc.GameRequest, string, error) {
	var uuid string
	var historyBytes []byte
	var requestBytes []byte
	err := db.QueryRowContext(ctx, `SELECT uuid, history, request FROM games WHERE id = $1`, gameId).Scan(&uuid, &historyBytes, &requestBytes)
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

func InitializeUserRating(ctx context.Context, db *sql.DB, tx *sql.Tx, userId uint) error {
	var ratings *entity.Ratings
	err := db.QueryRowContext(ctx, `SELECT jsonb_extract_path(ratings, 'Data') FROM profiles WHERE user_id = $1`, userId).Scan(&ratings)
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

func GetUserRating(ctx context.Context, db *sql.DB, tx *sql.Tx, userId uint, ratingKey entity.VariantKey, defaultRating *entity.SingleRating) (*entity.SingleRating, error) {
	err := InitializeUserRating(ctx, db, tx, userId)
	if err != nil {
		return nil, err
	}

	var sr *entity.SingleRating
	// XXX: Somewhat ugly, see comment above
	queryStmt := fmt.Sprintf(`SELECT jsonb_extract_path(ratings, 'Data', '%s') FROM profiles WHERE user_id = $1`, ratingKey)
	err = db.QueryRowContext(ctx, queryStmt, userId).Scan(&sr)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("profile not found for user_id: %d", userId)
	}
	if err != nil {
		return nil, err
	}

	if sr == nil {
		sr = defaultRating
		err = UpdateUserRating(ctx, db, tx, userId, ratingKey, sr)
		if err != nil {
			return nil, err
		}
	}

	return sr, nil
}

func UpdateUserRating(ctx context.Context, db *sql.DB, tx *sql.Tx, userId uint, ratingKey entity.VariantKey, newRating *entity.SingleRating) error {
	err := InitializeUserRating(ctx, db, tx, userId)
	if err != nil {
		return err
	}

	// XXX: Somewhat ugly, see comment above
	stmt := fmt.Sprintf(`UPDATE profiles SET ratings = jsonb_set(ratings, '{Data, %s}', $1) WHERE user_id = $2`, ratingKey)
	_, err = tx.ExecContext(ctx, stmt, newRating, userId)
	if err != nil {
		return err
	}
	return nil
}
