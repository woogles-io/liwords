package common

import (
	"context"
	"fmt"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/proto"
)

var DefaultTxOptions = pgx.TxOptions{
	IsoLevel:       pgx.ReadCommitted,
	AccessMode:     pgx.ReadWrite,
	DeferrableMode: pgx.Deferrable, // not used for this isolevel/access mode
}

var RepeatableReadTxOptions = pgx.TxOptions{
	IsoLevel:       pgx.RepeatableRead,
	AccessMode:     pgx.ReadWrite,
	DeferrableMode: pgx.Deferrable, // not used for this isolevel/access mode
}

type RowIterator interface {
	Close()
	Next() bool
	Scan(dest ...interface{}) error
}

func GetUserDBIDFromUUID(ctx context.Context, tx pgx.Tx, uuid string) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, "SELECT id FROM users WHERE uuid = $1", uuid).Scan(&id)
	if err == pgx.ErrNoRows {
		return 0, fmt.Errorf("cannot get id from uuid %s: no rows for table users", uuid)
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetGameDBIDFromUUID(ctx context.Context, tx pgx.Tx, uuid string) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, "SELECT id FROM games WHERE uuid = $1", uuid).Scan(&id)
	if err == pgx.ErrNoRows {
		return 0, fmt.Errorf("cannot get id from uuid %s: no rows for table games", uuid)
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetPuzzleDBIDFromUUID(ctx context.Context, tx pgx.Tx, uuid string) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, "SELECT id FROM puzzles WHERE uuid = $1", uuid).Scan(&id)
	if err == pgx.ErrNoRows {
		return 0, fmt.Errorf("cannot get id from uuid %s: no rows for table puzzles", uuid)
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetGameInfo(ctx context.Context, tx pgx.Tx, gameId int) (*macondopb.GameHistory, *ipc.GameRequest, string, error) {
	var uuid string
	var historyBytes []byte
	var requestBytes []byte
	err := tx.QueryRow(ctx, `SELECT uuid, history, request FROM games WHERE id = $1`, gameId).Scan(&uuid, &historyBytes, &requestBytes)
	if err == pgx.ErrNoRows {
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

func InitializeUserRating(ctx context.Context, tx pgx.Tx, userId int64) error {
	var ratings *entity.Ratings
	err := tx.QueryRow(ctx, `SELECT ratings FROM profiles WHERE user_id = $1`, userId).Scan(&ratings)
	if err == pgx.ErrNoRows {
		return fmt.Errorf("profile not found for user_id: %d", userId)
	}
	if err != nil {
		return err
	}

	if ratings.Data == nil {
		result, err := tx.Exec(ctx, `UPDATE profiles SET ratings = jsonb_set(ratings, '{Data}', jsonb '{}') WHERE user_id = $1 AND NULLIF(ratings->'Data', 'null') IS NULL`, userId)
		if err != nil {
			return err
		}
		rowsAffected := result.RowsAffected()
		if rowsAffected != 1 {
			return fmt.Errorf("not exactly one row affected for initializing user rating: %d, %d", userId, rowsAffected)
		}
	}
	return nil
}

func GetUserRating(ctx context.Context, tx pgx.Tx, userId int64, ratingKey entity.VariantKey, defaultRating *entity.SingleRating) (*entity.SingleRating, error) {
	err := InitializeUserRating(ctx, tx, userId)
	if err != nil {
		return nil, err
	}

	var sr *entity.SingleRating
	err = tx.QueryRow(ctx, "select ratings->'Data'->$1 from profiles where user_id = $2", ratingKey, userId).Scan(&sr)
	if err == pgx.ErrNoRows {
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

func UpdateUserRating(ctx context.Context, tx pgx.Tx, userId int64, ratingKey entity.VariantKey, newRating *entity.SingleRating) error {
	err := InitializeUserRating(ctx, tx, userId)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "update profiles set ratings = jsonb_set(ratings, array['Data', $1], $2) where user_id = $3", ratingKey, newRating, userId)
	if err != nil {
		return err
	}
	return nil
}

func OpenDB(host, port, name, user, password, sslmode string) (*pgxpool.Pool, error) {
	connStr := PostgresConnUri(host, port, name, user, password, sslmode)
	ctx := context.Background()

	dbPool, err := pgxpool.Connect(context.Background(), connStr)
	if err != nil {
		return nil, err
	}

	err = dbPool.Ping(ctx)
	if err != nil {
		return nil, err
	}
	return dbPool, nil
}

func PostgresConnUri(host, port, name, user, password, sslmode string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, name, sslmode)
}

// PostgresConnDSN is obsolete and only for Gorm. Remove once we get rid of gorm.
func PostgresConnDSN(host, port, name, user, password, sslmode string) string {
	return fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		host, port, name, user, password, sslmode)
}
