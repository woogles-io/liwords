package main

import (
	"regexp"
	"strings"
)

// sqlcName matches the comment sqlc puts at the top of every generated query:
// "-- name: ListActiveGames :many".
var sqlcName = regexp.MustCompile(`^--\s*name:\s*(\w+)`)

// maxSQLSpanName bounds the name of a span for SQL that sqlc did not generate.
// Jaeger's OpenSearch index ignores operation names over 256 characters, so a
// longer one can be stored but never searched or aggregated.
const maxSQLSpanName = 100

// pgxSpanName names a query's span after its sqlc query name, or after the
// start of the statement for SQL that sqlc did not generate. otelpgx would
// otherwise name the span after the whole statement; the full SQL is still in
// the span's db.query.text attribute.
func pgxSpanName(stmt string) string {
	stmt = strings.TrimSpace(stmt)
	if m := sqlcName.FindStringSubmatch(stmt); m != nil {
		return m[1]
	}
	name := strings.Join(strings.Fields(stmt), " ")
	if len(name) > maxSQLSpanName {
		name = name[:maxSQLSpanName]
	}
	if name == "" {
		return "UNKNOWN"
	}
	return name
}
