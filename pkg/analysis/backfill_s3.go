package analysis

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

// BackfillOptions controls UploadResults and ClearUploadedResults.
type BackfillOptions struct {
	Workers   int
	BatchSize int
	// StartAfter resumes after this job id (exclusive).
	StartAfter uuid.UUID
	// MaxJobs stops after this many jobs; 0 means no limit.
	MaxJobs int
	DryRun  bool
	// Progress, when set, is called after each batch with the last id done.
	Progress func(BackfillStats)
}

// BackfillStats counts what a backfill pass did.
type BackfillStats struct {
	Processed, Done, Skipped, Failed int64
	Cursor                           uuid.UUID
}

type backfillCounters struct {
	processed, done, skipped, failed atomic.Int64
}

func (c *backfillCounters) stats(cursor uuid.UUID) BackfillStats {
	return BackfillStats{
		Processed: c.processed.Load(),
		Done:      c.done.Load(),
		Skipped:   c.skipped.Load(),
		Failed:    c.failed.Load(),
		Cursor:    cursor,
	}
}

// runBatches pages through rows with list and handles each with handle, using
// up to opts.Workers goroutines. Each batch finishes before the cursor moves,
// so a stopped run resumes from the last reported cursor without gaps.
func runBatches[R any](ctx context.Context, opts BackfillOptions,
	list func(ctx context.Context, after uuid.UUID, limit int32) ([]R, error),
	id func(R) uuid.UUID,
	handle func(ctx context.Context, row R, c *backfillCounters)) (BackfillStats, error) {

	workers := max(opts.Workers, 1)
	var c backfillCounters
	cursor := opts.StartAfter
	for {
		if ctx.Err() != nil {
			return c.stats(cursor), ctx.Err()
		}
		limit := opts.BatchSize
		if opts.MaxJobs > 0 {
			limit = min(limit, opts.MaxJobs-int(c.processed.Load()))
			if limit <= 0 {
				return c.stats(cursor), nil
			}
		}
		rows, err := list(ctx, cursor, int32(limit))
		if err != nil {
			return c.stats(cursor), err
		}
		if len(rows) == 0 {
			return c.stats(cursor), nil
		}

		sem := make(chan struct{}, workers)
		var wg sync.WaitGroup
		for _, row := range rows {
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				handle(ctx, row, &c)
				c.processed.Add(1)
			}()
		}
		wg.Wait()

		cursor = id(rows[len(rows)-1])
		if opts.Progress != nil {
			opts.Progress(c.stats(cursor))
		}
	}
}

// UploadResults copies results still held only in the result column into the
// store and records their keys. The result column is left alone; see
// ClearUploadedResults. Safe to stop and re-run.
func UploadResults(ctx context.Context, queries *models.Queries, store ResultStore, opts BackfillOptions) (BackfillStats, error) {
	return runBatches(ctx, opts,
		func(ctx context.Context, after uuid.UUID, limit int32) ([]models.ListAnalysisResultsToUploadRow, error) {
			return queries.ListAnalysisResultsToUpload(ctx, models.ListAnalysisResultsToUploadParams{
				After: after, BatchSize: limit,
			})
		},
		func(r models.ListAnalysisResultsToUploadRow) uuid.UUID { return r.ID },
		func(ctx context.Context, row models.ListAnalysisResultsToUploadRow, c *backfillCounters) {
			l := log.With().Str("job_id", row.ID.String()).Str("game_id", row.GameID).Logger()
			result, err := unmarshalResult(row.Result)
			if err != nil {
				l.Error().Err(err).Msg("backfill-upload: stored result unreadable")
				c.failed.Add(1)
				return
			}
			if opts.DryRun {
				c.done.Add(1)
				return
			}
			key := ResultKey(row.GameID, row.ID)
			if err := store.Put(ctx, key, result); err != nil {
				l.Error().Err(err).Msg("backfill-upload: put failed")
				c.failed.Add(1)
				return
			}
			n, err := queries.SetAnalysisResultS3Key(ctx, models.SetAnalysisResultS3KeyParams{
				ResultS3Key: pgtype.Text{String: key, Valid: true},
				ID:          row.ID,
			})
			if err != nil || n == 0 {
				// A reanalysis stored its own result meanwhile (or the write
				// failed); nothing points at this object.
				if delErr := store.Delete(ctx, key); delErr != nil {
					l.Warn().Err(delErr).Str("s3_key", key).Msg("backfill-upload: orphan delete failed")
				}
				if err != nil {
					l.Error().Err(err).Msg("backfill-upload: set key failed")
					c.failed.Add(1)
				} else {
					c.skipped.Add(1)
				}
				return
			}
			c.done.Add(1)
		})
}

// errResultMismatch means the stored object doesn't match the result column.
var errResultMismatch = errors.New("stored object differs from result column")

// ClearUploadedResults drops the result column's copy for jobs that no longer
// need it: those whose object matches it, and zero-turn ones, which have no
// object. Jobs whose summary columns aren't filled yet are left alone, since
// the MI queries still fall back to the result column for them.
func ClearUploadedResults(ctx context.Context, queries *models.Queries, store ResultStore, opts BackfillOptions) (BackfillStats, error) {
	return runBatches(ctx, opts,
		func(ctx context.Context, after uuid.UUID, limit int32) ([]models.ListAnalysisResultsToClearRow, error) {
			return queries.ListAnalysisResultsToClear(ctx, models.ListAnalysisResultsToClearParams{
				After: after, BatchSize: limit,
			})
		},
		func(r models.ListAnalysisResultsToClearRow) uuid.UUID { return r.ID },
		func(ctx context.Context, row models.ListAnalysisResultsToClearRow, c *backfillCounters) {
			l := log.With().Str("job_id", row.ID.String()).Str("game_id", row.GameID).Logger()
			dbResult, err := unmarshalResult(row.Result)
			if err != nil {
				l.Error().Err(err).Msg("backfill-clear: stored result unreadable")
				c.failed.Add(1)
				return
			}
			if !row.ResultS3Key.Valid {
				// Not uploaded: only a zero-turn result can go, as it never
				// gets an object.
				if len(dbResult.GetTurns()) > 0 {
					c.skipped.Add(1)
					return
				}
			} else {
				stored, err := store.Get(ctx, row.ResultS3Key.String)
				if err == nil && !proto.Equal(stored, dbResult) {
					err = errResultMismatch
				}
				if err != nil {
					l.Error().Err(err).Str("s3_key", row.ResultS3Key.String).Msg("backfill-clear: object check failed")
					c.failed.Add(1)
					return
				}
			}
			if opts.DryRun {
				c.done.Add(1)
				return
			}
			n, err := queries.ClearAnalysisResult(ctx, models.ClearAnalysisResultParams{
				ID:          row.ID,
				ResultS3Key: row.ResultS3Key,
			})
			if err != nil {
				l.Error().Err(err).Msg("backfill-clear: clear failed")
				c.failed.Add(1)
				return
			}
			if n == 0 {
				c.skipped.Add(1) // the job changed since it was listed
				return
			}
			c.done.Add(1)
		})
}
