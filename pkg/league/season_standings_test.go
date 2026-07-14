package league

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// The batched GetAllDivisionStandings fetches every division's rows in one
// query ordered by division_number, then splits the flat result into one
// contiguous bucket per division. groupByDivision does that split. The rows
// are globally ordered by division_number and each carries its division id, so
// a division's rows form a contiguous run in the same order as the divisions
// slice. A division with no rows must yield an empty bucket WITHOUT stealing a
// later division's run -- that misalignment is the bug this helper must avoid.

type gbdRow struct {
	d uuid.UUID
	v int
}

func gbdKey(r gbdRow) uuid.UUID { return r.d }

func TestGroupByDivision_Contiguous(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	rows := []gbdRow{{a, 1}, {a, 2}, {b, 3}, {c, 4}, {c, 5}}
	got := groupByDivision([]uuid.UUID{a, b, c}, rows, gbdKey)
	require.Len(t, got, 3)
	require.Equal(t, []gbdRow{{a, 1}, {a, 2}}, got[0])
	require.Equal(t, []gbdRow{{b, 3}}, got[1])
	require.Equal(t, []gbdRow{{c, 4}, {c, 5}}, got[2])
}

func TestGroupByDivision_EmptyMiddleDivisionKeepsAlignment(t *testing.T) {
	// B has no rows. Its bucket must be empty and C must still line up -- a
	// naive "next run belongs to next division" walk would give C's rows to B.
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	rows := []gbdRow{{a, 1}, {c, 2}}
	got := groupByDivision([]uuid.UUID{a, b, c}, rows, gbdKey)
	require.Len(t, got, 3)
	require.Equal(t, []gbdRow{{a, 1}}, got[0])
	require.Empty(t, got[1])
	require.Equal(t, []gbdRow{{c, 2}}, got[2])
}

func TestGroupByDivision_LeadingEmptyDivision(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	rows := []gbdRow{{b, 1}, {c, 2}}
	got := groupByDivision([]uuid.UUID{a, b, c}, rows, gbdKey)
	require.Len(t, got, 3)
	require.Empty(t, got[0])
	require.Equal(t, []gbdRow{{b, 1}}, got[1])
	require.Equal(t, []gbdRow{{c, 2}}, got[2])
}

func TestGroupByDivision_TrailingEmptyDivision(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	rows := []gbdRow{{a, 1}, {b, 2}}
	got := groupByDivision([]uuid.UUID{a, b, c}, rows, gbdKey)
	require.Len(t, got, 3)
	require.Equal(t, []gbdRow{{a, 1}}, got[0])
	require.Equal(t, []gbdRow{{b, 2}}, got[1])
	require.Empty(t, got[2])
}

func TestGroupByDivision_AllEmpty(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	got := groupByDivision([]uuid.UUID{a, b}, []gbdRow{}, gbdKey)
	require.Len(t, got, 2)
	require.Empty(t, got[0])
	require.Empty(t, got[1])
}

func TestGroupByDivision_NoDivisions(t *testing.T) {
	got := groupByDivision([]uuid.UUID{}, []gbdRow{{uuid.New(), 1}}, gbdKey)
	require.Empty(t, got)
}
