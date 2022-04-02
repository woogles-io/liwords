package common

import (
	"context"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/rs/zerolog/log"
)

var TestDBHost = os.Getenv("TEST_DB_HOST")
var MigrationsPath = os.Getenv("DB_MIGRATIONS_PATH")
var TestDBName = os.Getenv("TEST_DB_NAME")
var TestDBPort = os.Getenv("DB_PORT")
var TestDBUser = os.Getenv("DB_USER")
var TestDBPassword = os.Getenv("DB_PASSWORD")
var TestDBSSLMode = os.Getenv("DB_SSL_MODE")

func RecreateTestDB() error {
	ctx := context.Background()
	db, err := pgx.Connect(ctx, PostgresConnUri(TestDBHost, TestDBPort,
		"", TestDBUser, TestDBPassword, TestDBSSLMode))
	if err != nil {
		return err
	}
	defer db.Close(ctx)
	_, err = db.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", TestDBName))
	if err != nil {
		return err
	}

	_, err = db.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", TestDBName))
	if err != nil {
		return err
	}
	log.Info().Msg("running migrations")
	// And create all tables/sequences/etc.
	m, err := migrate.New(MigrationsPath, TestingPostgresConnUri())
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

func TeardownTestDB() error {
	ctx := context.Background()
	db, err := pgx.Connect(ctx, PostgresConnUri(TestDBHost, TestDBPort,
		"", TestDBUser, TestDBPassword, TestDBSSLMode))
	if err != nil {
		return err
	}
	defer db.Close(ctx)

	_, err = db.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", TestDBName))
	if err != nil {
		return err
	}
	return nil
}

func OpenTestingDB() (*pgxpool.Pool, error) {
	return OpenDB(TestDBHost, TestDBPort, TestDBName, TestDBUser, TestDBPassword, TestDBSSLMode)
}

func TestingPostgresConnUri() string {
	return PostgresConnUri(TestDBHost, TestDBPort, TestDBName, TestDBUser, TestDBPassword, TestDBSSLMode)
}

// XXX: Delete me after removing Gorm
func TestingPostgresConnDSN() string {
	return PostgresConnDSN(TestDBHost, TestDBPort, TestDBName, TestDBUser, TestDBPassword, TestDBSSLMode)
}
