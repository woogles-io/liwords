package collections

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsDuplicateGame(t *testing.T) {
	dupGame := &pgconn.PgError{Code: "23505", ConstraintName: "collection_games_collection_id_game_id_key"}
	dupChapter := &pgconn.PgError{Code: "23505", ConstraintName: "collection_games_collection_id_chapter_number_key"}

	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"duplicate game", dupGame, true},
		{"wrapped duplicate game", fmt.Errorf("add: %w", dupGame), true},
		{"duplicate chapter", dupChapter, false},
		{"other error", errors.New("boom"), false},
	} {
		if got := isDuplicateGame(tc.err); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
