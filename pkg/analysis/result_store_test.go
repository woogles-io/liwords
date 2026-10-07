package analysis_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"
	"google.golang.org/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/analysis"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/analysis_service"
)

// s3Fixture is an AnalysisService backed by a MemoryResultStore, with one
// league game queued for analysis and two workers that can submit results.
type s3Fixture struct {
	pool       *pgxpool.Pool
	queries    *models.Queries
	svc        *analysis.AnalysisService
	store      *analysis.MemoryResultStore
	divisionID uuid.UUID
	gameID     string
	// worker contexts carry each worker's API key, as the middleware would.
	worker1, worker2 context.Context
}

func newS3Fixture(t *testing.T) *s3Fixture {
	is := is.New(t)
	ctx := context.Background()

	pool, queries := setupTestDB(t)
	t.Cleanup(pool.Close)
	createTestUsers(t, pool)

	ustore, err := user.NewDBStore(pool)
	is.NoErr(err)
	workerCtx := func(userUUID string) context.Context {
		key, err := ustore.ResetAPIKey(ctx, userUUID)
		is.NoErr(err)
		return apiserver.StoreAPIKeyInContext(ctx, key)
	}

	store := analysis.NewMemoryResultStore()
	// Zero-turn submissions (which read the game store) aren't exercised here.
	svc := analysis.NewAnalysisService(ustore, nil, queries, pool)
	svc.SetResultStore(store)

	leagueID, seasonID, divisionID := createMinimalLeague(t, ctx, queries)
	seedStanding(t, ctx, queries, divisionID, 1, 0, 0)
	seedStanding(t, ctx, queries, divisionID, 2, 0, 0)
	gameID := insertLeagueGame(t, ctx, pool, leagueID, seasonID, divisionID, time.Now(), 400, 380)
	is.NoErr(analysis.EnqueueGameForAnalysis(ctx, queries, gameID, 0))

	return &s3Fixture{
		pool: pool, queries: queries, svc: svc, store: store,
		divisionID: divisionID, gameID: gameID,
		worker1: workerCtx("test-uuid-3"), worker2: workerCtx("test-uuid-4"),
	}
}

// fullResult is a result with turns, so it gets stored as an object.
func fullResult(mi0, mi1 float64) *macondopb.GameAnalysisResult {
	r := analysisResult(mi0, mi1)
	r.Turns = []*macondopb.TurnAnalysis{
		{TurnNumber: 1, PlayerName: "testuser1", PlayedMove: "8D QUIXOTIC", PlayedScore: 365},
	}
	return r
}

func pgtextOf(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }

func (f *s3Fixture) claim(t *testing.T, worker string) uuid.UUID {
	t.Helper()
	job, err := f.queries.ClaimNextJob(context.Background(), pgtextOf(worker))
	is.New(t).NoErr(err)
	return job.ID
}

func (f *s3Fixture) submit(t *testing.T, ctx context.Context, jobID uuid.UUID, r *macondopb.GameAnalysisResult) *pb.SubmitResultResponse {
	t.Helper()
	resp, err := f.svc.SubmitResult(ctx, connect.NewRequest(&pb.SubmitResultRequest{
		JobId: jobID.String(), Result: r,
	}))
	is.New(t).NoErr(err)
	return resp.Msg
}

func (f *s3Fixture) job(t *testing.T) models.GetJobByGameIDRow {
	t.Helper()
	job, err := f.queries.GetJobByGameID(context.Background(), f.gameID)
	is.New(t).NoErr(err)
	return job
}

func (f *s3Fixture) served(t *testing.T) *pb.GetAnalysisResultResponse {
	t.Helper()
	resp, err := f.svc.GetAnalysisResult(context.Background(),
		connect.NewRequest(&pb.GetAnalysisResultRequest{GameId: f.gameID}))
	is.New(t).NoErr(err)
	return resp.Msg
}

// waitForKeys waits for the store to hold exactly want; deletes of replaced
// or orphaned objects happen in the background.
func (f *s3Fixture) waitForKeys(t *testing.T, want ...string) {
	t.Helper()
	slices.Sort(want)
	var got []string
	for range 100 {
		got = f.store.Keys()
		slices.Sort(got)
		if slices.Equal(got, want) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("store keys = %v, want %v", got, want)
}

func TestSubmitResultStoresObject(t *testing.T) {
	is := is.New(t)
	f := newS3Fixture(t)

	jobID := f.claim(t, "test-uuid-3")
	want := fullResult(3.0, 4.0)
	is.True(f.submit(t, f.worker1, jobID, want).Accepted)

	job := f.job(t)
	is.True(job.ResultS3Key.Valid)
	is.Equal(len(job.Result), 0) // the column no longer holds results
	is.Equal(job.AnalysisVersion.Int32, int32(2))
	f.waitForKeys(t, job.ResultS3Key.String)

	served := f.served(t)
	is.True(served.Found)
	is.True(proto.Equal(served.Result, want))

	// Standings read the summary columns, not the object.
	is.NoErr(f.queries.RefreshDivisionMistakeIndex(context.Background(), f.divisionID))
	assertMI(t, context.Background(), f.queries, f.divisionID, 1, 3.0, 1)
	assertMI(t, context.Background(), f.queries, f.divisionID, 2, 4.0, 1)
}

// TestLateSubmissionCantReplaceResult: a worker whose job was reclaimed and
// finished by another worker submits late. It must not touch the accepted
// result, before or after this change's ownership check.
func TestLateSubmissionCantReplaceResult(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	f := newS3Fixture(t)

	jobID := f.claim(t, "test-uuid-3")
	// Worker 1 goes quiet; the job is reclaimed and worker 2 finishes it.
	_, err := f.pool.Exec(ctx, `UPDATE analysis_jobs SET heartbeat_at = NOW() - INTERVAL '1 hour' WHERE id = $1`, jobID)
	is.NoErr(err)
	is.NoErr(f.queries.ReclaimStaleJobs(ctx))
	is.Equal(f.claim(t, "test-uuid-4"), jobID)
	accepted := fullResult(3.0, 4.0)
	is.True(f.submit(t, f.worker2, jobID, accepted).Accepted)
	acceptedKey := f.job(t).ResultS3Key.String

	// Worker 1's late submission is turned away without storing anything.
	is.True(!f.submit(t, f.worker1, jobID, fullResult(9.0, 9.0)).Accepted)
	f.waitForKeys(t, acceptedKey)
	is.Equal(f.job(t).ResultS3Key.String, acceptedKey)
	is.True(proto.Equal(f.served(t).Result, accepted))
}

// TestJobLostDuringUpload: the job changes hands while the result uploads, so
// CompleteJob rejects it. The uploaded object must be removed, not served.
func TestJobLostDuringUpload(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	f := newS3Fixture(t)

	jobID := f.claim(t, "test-uuid-3")
	f.store.BeforePut = func() {
		_, err := f.pool.Exec(ctx, `UPDATE analysis_jobs SET claimed_by_user_uuid = NULL, status = 'pending' WHERE id = $1`, jobID)
		is.NoErr(err)
	}
	is.True(!f.submit(t, f.worker1, jobID, fullResult(3.0, 4.0)).Accepted)
	f.waitForKeys(t)
	is.True(!f.job(t).ResultS3Key.Valid)
}

func TestFailedUploadIsNotAccepted(t *testing.T) {
	is := is.New(t)
	f := newS3Fixture(t)

	jobID := f.claim(t, "test-uuid-3")
	f.store.FailPut = true
	is.True(!f.submit(t, f.worker1, jobID, fullResult(3.0, 4.0)).Accepted)
	// The job is still the worker's, so it can retry.
	f.store.FailPut = false
	is.True(f.submit(t, f.worker1, jobID, fullResult(3.0, 4.0)).Accepted)
}

func TestReanalysisReplacesObject(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	f := newS3Fixture(t)

	jobID := f.claim(t, "test-uuid-3")
	is.True(f.submit(t, f.worker1, jobID, fullResult(3.0, 4.0)).Accepted)
	oldKey := f.job(t).ResultS3Key.String

	// Requeued: the job keeps its old object (and MI) until the new one lands.
	is.NoErr(analysis.RequeueJobByGameID(ctx, f.queries, f.gameID, 0))
	is.Equal(f.job(t).ResultS3Key.String, oldKey)
	is.NoErr(f.queries.RefreshDivisionMistakeIndex(ctx, f.divisionID))
	assertMI(t, ctx, f.queries, f.divisionID, 1, 3.0, 1)

	is.Equal(f.claim(t, "test-uuid-3"), jobID)
	newer := fullResult(5.0, 1.0)
	is.True(f.submit(t, f.worker1, jobID, newer).Accepted)
	newKey := f.job(t).ResultS3Key.String
	is.True(newKey != oldKey)
	f.waitForKeys(t, newKey) // the old object is deleted
	is.True(proto.Equal(f.served(t).Result, newer))
}

func TestGetAnalysisResultSources(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	f := newS3Fixture(t)

	// A job completed before results moved to S3: served from the column.
	legacy := fullResult(3.0, 4.0)
	claimAndComplete(t, ctx, f.queries, legacy)
	served := f.served(t)
	is.True(served.Found)
	is.True(proto.Equal(served.Result, legacy))

	// Uploaded but the fetch fails: the column's copy still serves it.
	is.NoErr(f.store.Put(ctx, "k", legacy))
	_, err := f.pool.Exec(ctx, `UPDATE analysis_jobs SET result_s3_key = 'k'`)
	is.NoErr(err)
	f.store.FailGet = true
	is.True(proto.Equal(f.served(t).Result, legacy))

	// ...and with no copy left, the failure is reported rather than hidden.
	_, err = f.pool.Exec(ctx, `UPDATE analysis_jobs SET result = NULL`)
	is.NoErr(err)
	_, err = f.svc.GetAnalysisResult(ctx, connect.NewRequest(&pb.GetAnalysisResultRequest{GameId: f.gameID}))
	is.True(err != nil)

	// A zero-turn game: completed with neither an object nor a column copy.
	_, err = f.pool.Exec(ctx, `UPDATE analysis_jobs SET result_s3_key = NULL`)
	is.NoErr(err)
	served = f.served(t)
	is.True(served.Found)
	is.Equal(len(served.Result.GetTurns()), 0)
}

// TestUploadAndClearResults covers cmd/backfill-analysis-s3's two passes over
// jobs completed before results moved to S3.
func TestUploadAndClearResults(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	pool, queries := setupTestDB(t)
	defer pool.Close()
	createTestUsers(t, pool)
	leagueID, seasonID, divisionID := createMinimalLeague(t, ctx, queries)

	now := time.Now()
	type legacyJob struct {
		result  *macondopb.GameAnalysisResult
		columns bool // summary columns already filled
	}
	jobs := []legacyJob{
		{fullResult(3.0, 4.0), true},
		{fullResult(1.0, 2.0), true},
		{&macondopb.GameAnalysisResult{AnalysisVersion: 2}, true}, // zero-turn
		{fullResult(5.0, 6.0), false},                             // columns not backfilled yet
	}
	games := make([]string, len(jobs))
	for i, j := range jobs {
		games[i] = insertLeagueGame(t, ctx, pool, leagueID, seasonID, divisionID, now.Add(time.Duration(i)*time.Minute), 400, 380)
		is.NoErr(analysis.EnqueueGameForAnalysis(ctx, queries, games[i], 0))
		claimAndCompleteWith(t, ctx, queries, j.result, j.columns)
	}
	store := analysis.NewMemoryResultStore()
	opts := analysis.BackfillOptions{Workers: 3, BatchSize: 2}

	// Upload: everything with turns gets an object; the column is kept.
	stats, err := analysis.UploadResults(ctx, queries, store, opts)
	is.NoErr(err)
	is.Equal(stats.Done, int64(3))
	is.Equal(stats.Failed, int64(0))
	is.Equal(len(store.Keys()), 3)
	for i, g := range games {
		job, err := queries.GetJobByGameID(ctx, g)
		is.NoErr(err)
		is.True(len(job.Result) > 0)
		is.Equal(job.ResultS3Key.Valid, i != 2)
	}

	// Re-running finds nothing left to upload.
	stats, err = analysis.UploadResults(ctx, queries, store, opts)
	is.NoErr(err)
	is.Equal(stats.Processed, int64(0))

	// A corrupted object must not let its job's column copy go.
	job1, err := queries.GetJobByGameID(ctx, games[1])
	is.NoErr(err)
	is.NoErr(store.Put(ctx, job1.ResultS3Key.String, fullResult(9.0, 9.0)))

	// Clear: verified uploads and the zero-turn job; not the corrupted one,
	// nor the job whose summary columns are still missing.
	stats, err = analysis.ClearUploadedResults(ctx, queries, store, opts)
	is.NoErr(err)
	is.Equal(stats.Done, int64(2))
	is.Equal(stats.Failed, int64(1))
	for i, g := range games {
		job, err := queries.GetJobByGameID(ctx, g)
		is.NoErr(err)
		cleared := len(job.Result) == 0
		is.Equal(cleared, i == 0 || i == 2)
	}

	// Standings are unchanged by all of this: they read the summary columns,
	// falling back to the column copy only where those are missing.
	seedStanding(t, ctx, queries, divisionID, 1, 0, 0)
	seedStanding(t, ctx, queries, divisionID, 2, 0, 0)
	is.NoErr(queries.RefreshDivisionMistakeIndex(ctx, divisionID))
	assertMI(t, ctx, queries, divisionID, 1, 9.0, 3)
	assertMI(t, ctx, queries, divisionID, 2, 12.0, 3)
}
