package league

import "github.com/google/uuid"

// groupByDivision splits a flat, division_number-ordered result set into one
// contiguous bucket per division, aligned to divIDs.
//
// Precondition: rows are globally ordered so that all rows sharing a division
// id form a single contiguous run, and those runs appear in the same order as
// divIDs. Both hold when the query and GetDivisionsBySeason are ORDER BY
// division_number. Matching is by division id, not "the next run", so a
// division with no rows yields an empty bucket without stealing a later
// division's run.
func groupByDivision[T any](divIDs []uuid.UUID, rows []T, divOf func(T) uuid.UUID) [][]T {
	buckets := make([][]T, len(divIDs))
	ri := 0
	for di, id := range divIDs {
		start := ri
		for ri < len(rows) && divOf(rows[ri]) == id {
			ri++
		}
		buckets[di] = rows[start:ri]
	}
	return buckets
}
