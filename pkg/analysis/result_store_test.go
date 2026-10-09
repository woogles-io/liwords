package analysis_test

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
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
	"github.com/woogles-io/liwords/rpc/api/proto/analysis_service/analysis_serviceconnect"
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

// fullResult is a result with ten turns, so it gets stored as an object and
// mistake indexes up to 10 are plausible.
func fullResult(mi0, mi1 float64) *macondopb.GameAnalysisResult {
	r := analysisResult(mi0, mi1)
	for i := range 10 {
		r.Turns = append(r.Turns, &macondopb.TurnAnalysis{
			TurnNumber: int32(i + 1), PlayerIndex: int32(i % 2),
			PlayerName: fmt.Sprintf("testuser%d", i%2+1), PlayedMove: "8D QUIXOTIC", PlayedScore: 365,
		})
	}
	return r
}

func pgtextOf(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }

func (f *s3Fixture) claim(t *testing.T, worker string) uuid.UUID {
	t.Helper()
	job, err := f.queries.ClaimNextJob(context.Background(), anyClaims(pgtextOf(worker)))
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

	// Served from the object.
	jobID := f.claim(t, "test-uuid-3")
	want := fullResult(3.0, 4.0)
	is.True(f.submit(t, f.worker1, jobID, want).Accepted)
	is.True(proto.Equal(f.served(t).Result, want))

	// A failed fetch is reported, not served as an empty analysis.
	f.store.FailGet = true
	_, err := f.svc.GetAnalysisResult(ctx, connect.NewRequest(&pb.GetAnalysisResultRequest{GameId: f.gameID}))
	is.True(err != nil)
	f.store.FailGet = false

	// A zero-turn game: completed without an object.
	_, err = f.pool.Exec(ctx, `UPDATE analysis_jobs SET result_s3_key = NULL`)
	is.NoErr(err)
	served := f.served(t)
	is.True(served.Found)
	is.Equal(len(served.Result.GetTurns()), 0)
}

// TestSubmitResultWithoutStore: a server with no result store turns results
// away (the worker retries later) rather than accepting and losing them.
func TestSubmitResultWithoutStore(t *testing.T) {
	is := is.New(t)
	f := newS3Fixture(t)
	f.svc.SetResultStore(nil)

	jobID := f.claim(t, "test-uuid-3")
	is.True(!f.submit(t, f.worker1, jobID, fullResult(3.0, 4.0)).Accepted)
	is.Equal(f.job(t).Status, "claimed")
}

func (f *s3Fixture) claimJob(t *testing.T, ctx context.Context) *pb.ClaimJobResponse {
	t.Helper()
	resp, err := f.svc.ClaimJob(ctx, connect.NewRequest(&pb.ClaimJobRequest{
		MacondoVersion: analysis.MinMacondoVersion,
	}))
	is.New(t).NoErr(err)
	return resp.Msg
}

// TestClaimJobCapsActiveClaims: one account can't claim the whole queue and
// let it time out into failure.
func TestClaimJobCapsActiveClaims(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	f := newS3Fixture(t)

	// Three more queued games, four in all.
	leagueID, seasonID := uuid.Nil, uuid.Nil
	is.NoErr(f.pool.QueryRow(ctx, `SELECT league_id, season_id FROM games WHERE uuid = $1`, f.gameID).Scan(&leagueID, &seasonID))
	for i := range 3 {
		g := insertLeagueGame(t, ctx, f.pool, leagueID, seasonID, f.divisionID, time.Now().Add(time.Duration(i+1)*time.Minute), 400, 380)
		is.NoErr(analysis.EnqueueGameForAnalysis(ctx, f.queries, g, 0))
	}

	first := f.claimJob(t, f.worker1)
	second := f.claimJob(t, f.worker1)
	is.True(!first.NoJobs && !second.NoJobs)
	is.True(f.claimJob(t, f.worker1).NoJobs) // holding two already

	// Another worker is unaffected.
	is.True(!f.claimJob(t, f.worker2).NoJobs)

	// Finishing one frees a slot.
	jobID, err := uuid.Parse(first.JobId)
	is.NoErr(err)
	is.True(f.submit(t, f.worker1, jobID, fullResult(1.0, 2.0)).Accepted)
	is.True(!f.claimJob(t, f.worker1).NoJobs)
}

// TestSubmitResultRejectsImpossibleResults: values no real analysis produces
// are turned away, and the job stays with the worker to resubmit.
func TestSubmitResultRejectsImpossibleResults(t *testing.T) {
	is := is.New(t)
	f := newS3Fixture(t)
	jobID := f.claim(t, "test-uuid-3")

	tooManyTurns := fullResult(1.0, 1.0)
	for i := range 501 {
		tooManyTurns.Turns = append(tooManyTurns.Turns, &macondopb.TurnAnalysis{TurnNumber: int32(i + 11)})
	}
	badPlayer := fullResult(0, 0)
	badPlayer.Turns[0].PlayerIndex = 2

	for name, r := range map[string]*macondopb.GameAnalysisResult{
		"MI above turn count": fullResult(11, 0), // ten turns, so at most 10
		"negative MI":         fullResult(-0.2, 0),
		"NaN MI":              fullResult(math.NaN(), 0),
		"too many turns":      tooManyTurns,
		"bad player index":    badPlayer,
	} {
		resp := f.submit(t, f.worker1, jobID, r)
		if resp.Accepted {
			t.Errorf("%s: accepted", name)
		}
	}
	f.waitForKeys(t) // nothing stored

	is.True(f.submit(t, f.worker1, jobID, fullResult(1.0, 0.5)).Accepted)
}

// TestSubmitResultSizeCap: the queue service handler, configured as in
// cmd/liwords-api, rejects oversized requests before reading them in.
func TestSubmitResultSizeCap(t *testing.T) {
	is := is.New(t)
	f := newS3Fixture(t)

	mux := http.NewServeMux()
	mux.Handle(analysis_serviceconnect.NewAnalysisQueueServiceHandler(f.svc,
		connect.WithReadMaxBytes(analysis.MaxSubmitResultBytes)))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := analysis_serviceconnect.NewAnalysisQueueServiceClient(srv.Client(), srv.URL)

	huge := fullResult(1.0, 1.0)
	huge.Turns[0].PlayedMove = strings.Repeat("A", analysis.MaxSubmitResultBytes)
	_, err := client.SubmitResult(context.Background(), connect.NewRequest(&pb.SubmitResultRequest{
		JobId: uuid.NewString(), Result: huge,
	}))
	is.Equal(connect.CodeOf(err), connect.CodeResourceExhausted)
}
