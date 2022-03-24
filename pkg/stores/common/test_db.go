package common

import (
	"database/sql"
	"fmt"
	"os"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

const TestDBName = "liwords_test"

var TestDBHost = os.Getenv("TEST_DB_HOST")
var TestingConnStr = "host=" + TestDBHost + " port=5432 user=postgres password=pass sslmode=disable"
var TestingDBConnStr = fmt.Sprintf("%s database=%s", TestingConnStr, TestDBName)

func RecreatePuzzleTables(db *sql.DB) error {
	createPuzzlesStmt := `CREATE TABLE puzzles (
		id BIGSERIAL PRIMARY KEY,
		uuid text UNIQUE NOT NULL,
		game_id bigint NOT NULL,
		turn_number integer NOT NULL,
		answer jsonb NOT NULL,
		author_id integer,
		before_text text,
		after_text text,
		rating jsonb NOT NULL,
		created_at timestamptz NOT NULL DEFAULT NOW(),
		updated_at timestamptz NOT NULL DEFAULT NOW(),
		FOREIGN KEY (game_id) REFERENCES games (id),
		FOREIGN KEY (author_id) REFERENCES users (id))`

	createPuzzleTagTitlesStmt := `CREATE TABLE puzzle_tag_titles (
		id BIGSERIAL PRIMARY KEY,
	   	tag_title text NOT NULL)`

	createPuzzleTagsStmt := `CREATE TABLE puzzle_tags (
	   	puzzle_id bigint NOT NULL,
	   	tag_id bigint NOT NULL,
		UNIQUE(puzzle_id, tag_id),
	   	FOREIGN KEY (puzzle_id) REFERENCES puzzles(id),
	   	FOREIGN KEY (tag_id) REFERENCES puzzle_tag_titles(id))`

	createPuzzleAttemptsStmt := `CREATE TABLE puzzle_attempts (
		puzzle_id bigint NOT NULL,
		user_id bigint NOT NULL,
		correct bool,
		attempts integer,
		new_user_rating jsonb,
		new_puzzle_rating jsonb,
		created_at timestamptz NOT NULL DEFAULT NOW(),
		UNIQUE(puzzle_id, user_id),
		FOREIGN KEY (puzzle_id) REFERENCES puzzles (id),
		FOREIGN KEY (user_id) REFERENCES users (id))`

	createPuzzleVotesStmt := `CREATE TABLE puzzle_votes (
		puzzle_id bigint NOT NULL,
		user_id bigint NOT NULL,
		vote integer,
		created_at timestamptz NOT NULL DEFAULT NOW(),
		UNIQUE(puzzle_id, user_id),
		FOREIGN KEY (puzzle_id) REFERENCES puzzles (id),
		FOREIGN KEY (user_id) REFERENCES users (id))`

	stmts := []string{
		"DROP TABLE IF EXISTS puzzles CASCADE",
		"DROP TABLE IF EXISTS puzzle_tag_titles CASCADE",
		"DROP TABLE IF EXISTS puzzle_tags CASCADE",
		"DROP TABLE IF EXISTS puzzle_attempts CASCADE",
		createPuzzlesStmt,
		createPuzzleTagTitlesStmt,
		createPuzzleTagsStmt,
		createPuzzleAttemptsStmt,
		createPuzzleVotesStmt,
	}

	for _, stmt := range stmts {
		_, err := db.Exec(stmt)
		if err != nil {
			return err
		}
	}

	for name := range macondopb.PuzzleTag_value {
		_, err := db.Exec("INSERT INTO puzzle_tag_titles (tag_title) VALUES ($1)", name)
		if err != nil {
			return err
		}
	}
	return nil
}

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
