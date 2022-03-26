package common

import (
	"database/sql"
	"fmt"
	"os"
)

const TestDBName = "liwords_test"

var TestDBHost = os.Getenv("TEST_DB_HOST")
var TestingConnStr = "host=" + TestDBHost + " port=5432 user=postgres password=pass sslmode=disable"
var TestingDBConnStr = fmt.Sprintf("%s database=%s", TestingConnStr, TestDBName)
var MigrationFile = "file://../../db/migrations"
var MigrationConnString = fmt.Sprintf("postgres://postgres:pass@localhost:5432/%s?sslmode=disable", TestDBName)

func RecreateDB() error {
	db, err := sql.Open("pgx", TestingConnStr)
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

func OpenDB() (*sql.DB, error) {
	db, err := sql.Open("pgx", TestingDBConnStr)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}
