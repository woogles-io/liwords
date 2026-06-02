package common

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woogles-io/liwords/pkg/entity"

	"github.com/rs/zerolog/log"
)

var TestDBHost = os.Getenv("TEST_DB_HOST")
var MigrationsPath = os.Getenv("DB_MIGRATIONS_PATH")
var TestDBPrefix = os.Getenv("TEST_DB_PREFIX")
var TestDBPort = os.Getenv("DB_PORT")
var TestDBUser = os.Getenv("DB_USER")
var TestDBPassword = os.Getenv("DB_PASSWORD")
var TestDBSSLMode = os.Getenv("DB_SSL_MODE")

func TestDBName(pkg string) string {
	return TestDBPrefix + "_" + pkg
}

func RecreateTestDB(pkg string) error {
	ctx := context.Background()
	db, err := pgx.Connect(ctx, PostgresConnUri(TestDBHost, TestDBPort,
		"", TestDBUser, TestDBPassword, TestDBSSLMode))
	if err != nil {
		return err
	}
	defer db.Close(ctx)
	log.Info().Msg("dropping db")
	_, err = db.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", TestDBName(pkg)))
	if err != nil {
		return err
	}
	log.Info().Msg("creating db")
	_, err = db.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", TestDBName(pkg)))
	if err != nil {
		return err
	}
	log.Info().Msg("running migrations")
	// And create all tables/sequences/etc.
	m, err := migrate.New(MigrationsPath, TestingPostgresConnUri(pkg))
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil {
		return err
	}
	e1, e2 := m.Close()
	log.Err(e1).Msg("close-source")
	log.Err(e2).Msg("close-database")
	log.Info().Msg("created test db")
	return nil
}

func TeardownTestDB(pkg string) error {
	ctx := context.Background()
	db, err := pgx.Connect(ctx, PostgresConnUri(TestDBHost, TestDBPort,
		"", TestDBUser, TestDBPassword, TestDBSSLMode))
	if err != nil {
		return err
	}
	defer db.Close(ctx)

	_, err = db.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", TestDBName(pkg)))
	if err != nil {
		return err
	}
	return nil
}

func OpenTestingDB(pkg string) (*pgxpool.Pool, error) {
	return OpenDB(TestDBHost, TestDBPort, TestDBName(pkg), TestDBUser, TestDBPassword, TestDBSSLMode)
}

func TestingPostgresConnUri(pkg string) string {
	return PostgresConnUri(TestDBHost, TestDBPort, TestDBName(pkg), TestDBUser, TestDBPassword, TestDBSSLMode)
}

// XXX: Delete me after removing Gorm
func TestingPostgresConnDSN(pkg string) string {
	return PostgresConnDSN(TestDBHost, TestDBPort, TestDBName(pkg), TestDBUser, TestDBPassword, TestDBSSLMode)
}

// UpdateTableByUUID updates columns in a table row identified by its uuid column.
func UpdateTableByUUID(ctx context.Context, pool *pgxpool.Pool, table, uuid string, columns []string, args []interface{}) error {
	tx, err := pool.BeginTx(ctx, DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var setClauses strings.Builder
	allArgs := make([]interface{}, len(args)+1)
	for i, col := range columns {
		if i > 0 {
			setClauses.WriteString(", ")
		}
		fmt.Fprintf(&setClauses, "%s = $%d", col, i+1)
		allArgs[i] = args[i]
	}
	allArgs[len(args)] = uuid
	query := fmt.Sprintf("UPDATE %s SET %s WHERE uuid = $%d", table, setClauses.String(), len(args)+1)

	if _, err = tx.Exec(ctx, query, allArgs...); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func GetUserRatingWithPool(ctx context.Context, pool *pgxpool.Pool, userId int64, ratingKey entity.VariantKey) (*entity.SingleRating, error) {
	tx, err := pool.BeginTx(ctx, DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rating, err := GetUserRating(ctx, tx, userId, ratingKey)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return rating, nil
}

func GetUserStatsWithPool(ctx context.Context, pool *pgxpool.Pool, userId int64, ratingKey entity.VariantKey) (*entity.Stats, error) {
	tx, err := pool.BeginTx(ctx, DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	stats, err := GetUserStats(ctx, tx, userId, ratingKey)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return stats, nil
}

func GetUserDBIDByUUID(ctx context.Context, pool *pgxpool.Pool, uuid string) (int64, error) {
	var id int32
	err := pool.QueryRow(ctx, `SELECT id FROM users WHERE uuid = $1`, uuid).Scan(&id)
	return int64(id), err
}

func GetPuzzleDBIDByUUID(ctx context.Context, pool *pgxpool.Pool, uuid string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `SELECT id FROM puzzles WHERE uuid = $1`, uuid).Scan(&id)
	return id, err
}
