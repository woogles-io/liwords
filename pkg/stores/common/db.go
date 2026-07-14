package common

import (
	"context"
	"fmt"
	"strings"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/glicko"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/encoding/protojson"
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

// ReadOnlyTxOptions is for transactions that only issue SELECTs.
var ReadOnlyTxOptions = pgx.TxOptions{
	IsoLevel:   pgx.ReadCommitted,
	AccessMode: pgx.ReadOnly,
}

// RepeatableReadReadOnlyTxOptions is for read-only transactions that need a
// consistent snapshot across multiple SELECTs.
var RepeatableReadReadOnlyTxOptions = pgx.TxOptions{
	IsoLevel:   pgx.RepeatableRead,
	AccessMode: pgx.ReadOnly,
}

var InitialRating = &entity.SingleRating{
	Rating:          float64(glicko.InitialRating),
	RatingDeviation: float64(glicko.InitialRatingDeviation),
	Volatility:      glicko.InitialVolatility,
}

func GetGameInfo(ctx context.Context, tx pgx.Tx, gameId int) (*macondopb.GameHistory, *ipc.GameRequest, string, error) {
	var uuid string
	var historyBytes []byte
	var gameRequestBytes []byte
	err := tx.QueryRow(ctx, `SELECT uuid, history, game_request FROM games WHERE id = $1`, gameId).Scan(&uuid, &historyBytes, &gameRequestBytes)
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
	gr := &ipc.GameRequest{}
	err = protojson.Unmarshal(gameRequestBytes, gr) // ignore error, may be empty or invalid
	if err != nil {
		return nil, nil, "", err
	}

	return hist, gr, uuid, nil
}

func InitializeUserRating(ctx context.Context, tx pgx.Tx, userId int64) error {
	_, err := tx.Exec(ctx, `UPDATE profiles SET ratings = jsonb_set(ratings, '{Data}', jsonb '{}') WHERE user_id = $1 AND NULLIF(ratings->'Data', 'null') IS NULL`, userId)
	return err
}

func InitializeUserStats(ctx context.Context, tx pgx.Tx, userId int64) error {
	_, err := tx.Exec(ctx, `UPDATE profiles SET stats = jsonb_set(stats, '{Data}', jsonb '{}') WHERE user_id = $1 AND NULLIF(stats->'Data', 'null') IS NULL`, userId)
	return err
}

func GetUserRating(ctx context.Context, tx pgx.Tx, userId int64, ratingKey entity.VariantKey) (*entity.SingleRating, error) {
	err := InitializeUserRating(ctx, tx, userId)
	if err != nil {
		return nil, err
	}

	var playerRating *entity.SingleRating
	err = tx.QueryRow(ctx, "SELECT ratings->'Data'->$1 FROM profiles WHERE user_id = $2", ratingKey, userId).Scan(&playerRating)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("ratings not found for user_id: %d", userId)
	}
	if err != nil {
		return nil, err
	}

	if playerRating == nil {
		playerRating = entity.NewDefaultRating(true)
		err = UpdateUserRating(ctx, tx, userId, ratingKey, playerRating)
		if err != nil {
			return nil, err
		}
	}

	return playerRating, nil
}

func GetUserStats(ctx context.Context, tx pgx.Tx, userId int64, ratingKey entity.VariantKey) (*entity.Stats, error) {
	err := InitializeUserStats(ctx, tx, userId)
	if err != nil {
		return nil, err
	}

	var stats *entity.Stats
	err = tx.QueryRow(ctx, "SELECT stats->'Data'->$1 FROM profiles WHERE user_id = $2", ratingKey, userId).Scan(&stats)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("stats not found for user_id: %d", userId)
	}
	if err != nil {
		return nil, err
	}

	if stats == nil {
		stats = &entity.Stats{}
		err = UpdateUserStats(ctx, tx, userId, ratingKey, stats)
		if err != nil {
			return nil, err
		}
	}

	return stats, nil
}

func UpdateUserRating(ctx context.Context, tx pgx.Tx, userId int64, ratingKey entity.VariantKey, newRating *entity.SingleRating) error {
	err := InitializeUserRating(ctx, tx, userId)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "UPDATE profiles SET ratings = jsonb_set(ratings, array['Data', $1], $2) WHERE user_id = $3", ratingKey, newRating, userId)
	if err != nil {
		return err
	}
	return nil
}

func UpdateUserStats(ctx context.Context, tx pgx.Tx, userId int64, ratingKey entity.VariantKey, newStats *entity.Stats) error {
	err := InitializeUserStats(ctx, tx, userId)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "UPDATE profiles SET stats = jsonb_set(stats, array['Data', $1], $2) WHERE user_id = $3", ratingKey, newStats, userId)
	if err != nil {
		return err
	}
	return nil
}

func OpenDB(host, port, name, user, password, sslmode string) (*pgxpool.Pool, error) {
	connStr := PostgresConnUri(host, port, name, user, password, sslmode)
	ctx := context.Background()

	dbPool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, err
	}

	err = dbPool.Ping(ctx)
	if err != nil {
		return nil, err
	}
	return dbPool, nil
}

func BuildIn(num int, start int) string {
	var stmt strings.Builder
	fmt.Fprintf(&stmt, "$%d", start)
	for i := start + 1; i < start+num; i++ {
		fmt.Fprintf(&stmt, ", $%d", i)
	}
	return stmt.String()
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

func ToPGTypeText(str string) pgtype.Text {
	return pgtype.Text{Valid: true, String: str}
}
