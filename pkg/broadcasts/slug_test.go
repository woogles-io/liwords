package broadcasts

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

// Slugs are found whatever their case, and can't differ only in case.
func TestSlugIgnoresCase(t *testing.T) {
	pool, q := setupFeedDB(t)
	ctx := context.Background()
	b := createTestBroadcast(t, pool, q, "SATB-2026", "http://unused.invalid/")

	for _, slug := range []string{"SATB-2026", "satb-2026", "Satb-2026"} {
		row, err := q.GetBroadcastBySlug(ctx, slug)
		if err != nil || row.ID != b.id {
			t.Fatalf("GetBroadcastBySlug(%q): row=%+v err=%v", slug, row, err)
		}
	}
	if _, err := q.GetBroadcastVersion(ctx, "satb-2026"); err != nil {
		t.Fatalf("GetBroadcastVersion: %v", err)
	}

	_, err := q.CreateBroadcast(ctx, models.CreateBroadcastParams{
		Uuid:                uuid.New(),
		Slug:                "satb-2026",
		Name:                "duplicate",
		BroadcastUrl:        "http://unused.invalid/",
		BroadcastUrlFormat:  "tsh_newt_json",
		PollIntervalSeconds: 60,
		Lexicon:             "CSW24",
		BoardLayout:         "CrosswordGame",
		LetterDistribution:  "english",
		CreatorID:           b.creatorID,
	})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("slug differing only in case: err=%v, want a unique violation", err)
	}
}
