package common

import (
	"database/sql"
	"fmt"
	"os"
)

var TestDBHost = os.Getenv("TEST_DB_HOST")

const (
	TestDBName     = "liwords_test"
	TestDBPort     = "5432"
	TestDBUser     = "postgres"
	TestDBPassword = "pass"
	TestDBSSLMode  = "disable"
)

var MigrationFile = "file://../../db/migrations"

func RecreateTestDB() error {
	db, err := sql.Open("pgx", PostgresConnString(TestDBHost, TestDBPort,
		"", TestDBUser, TestDBPassword, TestDBSSLMode))
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", TestDBName))
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", TestDBName))
	if err != nil {
		return err
	}

	db.Close()
	return nil
}

func OpenTestingDB() (*sql.DB, error) {
	return OpenDB(TestDBHost, TestDBPort, TestDBName, TestDBUser, TestDBPassword, TestDBSSLMode)
}

func TestingPostgresConnString() string {
	return PostgresConnString(TestDBHost, TestDBPort, TestDBName, TestDBUser, TestDBPassword, TestDBSSLMode)
}

func TestingMigrationConnString() string {
	return MigrationConnString(TestDBHost, TestDBPort, TestDBName, TestDBUser, TestDBPassword, TestDBSSLMode)
}
