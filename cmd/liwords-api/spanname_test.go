package main

import (
	"strings"
	"testing"
)

func TestPgxSpanName(t *testing.T) {
	for _, tc := range []struct {
		stmt, want string
	}{
		{"-- name: ListActiveGamesFromOngoing :many\nSELECT g.uuid\nFROM ongoing_games g", "ListActiveGamesFromOngoing"},
		{"  -- name: GetGame :one\nSELECT 1", "GetGame"},
		{"SELECT expires_at, data\n  FROM db_sessions WHERE uuid = $1", "SELECT expires_at, data FROM db_sessions WHERE uuid = $1"},
		{"   ", "UNKNOWN"},
	} {
		if got := pgxSpanName(tc.stmt); got != tc.want {
			t.Errorf("pgxSpanName(%q) = %q, want %q", tc.stmt, got, tc.want)
		}
	}
	long := "SELECT " + strings.Repeat("a, ", 200) + "b FROM t"
	if got := pgxSpanName(long); len(got) != maxSQLSpanName {
		t.Errorf("long statement gave a %d-character name, want %d", len(got), maxSQLSpanName)
	}
}
