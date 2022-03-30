package common

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	// See:
	// https://github.com/jackc/pgx/wiki/Getting-started-with-pgx-through-database-sql
	// XXX: Why not use pgx interface directly?
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/rs/zerolog/log"
)

var TestDBHost = os.Getenv("TEST_DB_HOST")
var MigrationsPath = os.Getenv("DB_MIGRATIONS_PATH")

const (
	TestDBName     = "liwords_test"
	TestDBPort     = "5432"
	TestDBUser     = "postgres"
	TestDBPassword = "pass"
	TestDBSSLMode  = "disable"
)

func RecreateTestDB() error {
	db, err := sql.Open("pgx", PostgresConnUri(TestDBHost, TestDBPort,
		"", TestDBUser, TestDBPassword, TestDBSSLMode))
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", TestDBName))
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", TestDBName))
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
	db, err := sql.Open("pgx", PostgresConnUri(TestDBHost, TestDBPort,
		"", TestDBUser, TestDBPassword, TestDBSSLMode))
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", TestDBName))
	if err != nil {
		return err
	}
	return nil
}

func OpenTestingDB() (*sql.DB, error) {
	return OpenDB(TestDBHost, TestDBPort, TestDBName, TestDBUser, TestDBPassword, TestDBSSLMode)
}

func TestingPostgresConnUri() string {
	return PostgresConnUri(TestDBHost, TestDBPort, TestDBName, TestDBUser, TestDBPassword, TestDBSSLMode)
}

// XXX: Delete me after removing Gorm
func TestingPostgresConnDSN() string {
	return PostgresConnDSN(TestDBHost, TestDBPort, TestDBName, TestDBUser, TestDBPassword, TestDBSSLMode)
}
