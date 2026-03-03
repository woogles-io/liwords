package analysis_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/analysis"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const pkg = "analysis_test"

func setupTestDB(t *testing.T) (*pgxpool.Pool, *models.Queries) {
	err := common.RecreateTestDB(pkg)
	if err != nil {
		t.Fatal(err)
	}

	pool, err := common.OpenTestingDB(pkg)
	if err != nil {
		t.Fatal(err)
	}

	queries := models.New(pool)
	return pool, queries
}

func createTestUsers(t *testing.T, pool *pgxpool.Pool) {
	ustore, err := user.NewDBStore(pool)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	for i := 1; i <= 10; i++ {
		u := &entity.User{
			Username: fmt.Sprintf("testuser%d", i),
			Email:    fmt.Sprintf("testuser%d@test.com", i),
			UUID:     fmt.Sprintf("test-uuid-%d", i),
		}
		err = ustore.New(ctx, u)
		if err != nil {
			t.Fatalf("failed to create test user %d: %v", i, err)
		}
	}
}

// createMinimalLeague creates a league with 1 season, 1 division, and 2 registered players
func createMinimalLeague(t *testing.T, ctx context.Context, queries *models.Queries) (leagueID, seasonID, divisionID uuid.UUID) {
	is := is.New(t)

	leagueID = uuid.New()
	_, err := queries.CreateLeague(ctx, models.CreateLeagueParams{
		Uuid:        leagueID,
		Name:        "Test League",
		Description: pgtype.Text{String: "Test League for Analysis", Valid: true},
		Slug:        "test-league",
		Settings:    []byte(`{}`),
		IsActive:    pgtype.Bool{Bool: true, Valid: true},
		CreatedBy:   pgtype.Int8{Int64: 1, Valid: true},
	})
	is.NoErr(err)

	seasonID = uuid.New()
	_, err = queries.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         seasonID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       int32(ipc.SeasonStatus_SEASON_ACTIVE),
	})
	is.NoErr(err)

	divisionID = uuid.New()
	_, err = queries.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           divisionID,
		SeasonID:       seasonID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
	})
	is.NoErr(err)

	// Register 2 players
	for i := 1; i <= 2; i++ {
		_, err := queries.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:           int32(i),
			SeasonID:         seasonID,
			DivisionID:       pgtype.UUID{Bytes: divisionID, Valid: true},
			RegistrationDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			FirstsCount:      pgtype.Int4{Int32: 0, Valid: true},
			Status:           pgtype.Text{String: "ACTIVE", Valid: true},
			SeasonsAway:      pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	return leagueID, seasonID, divisionID
}

// TestEnqueueGameForAnalysis tests that a game can be enqueued for analysis
func TestEnqueueGameForAnalysis(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	pool, queries := setupTestDB(t)
	defer pool.Close()

	// Create test game ID
	gameID := uuid.New().String()

	// Enqueue the game
	err := analysis.EnqueueGameForAnalysis(ctx, queries, gameID, 0)
	is.NoErr(err)

	// Verify job was created
	job, err := queries.GetJobByGameID(ctx, gameID)
	is.NoErr(err)
	is.Equal(job.Status, "pending")
	is.Equal(job.GameID, gameID)

	// Verify config is valid JSON with expected fields
	var config map[string]interface{}
	err = json.Unmarshal(job.ConfigJson, &config)
	is.NoErr(err)

	// Check key config values
	is.Equal(config["sim_plays_early_mid"], float64(40))
	is.Equal(config["sim_plies_early_mid"], float64(5))
	is.Equal(config["peg_early_cutoff"], true)
	is.Equal(config["threads"], float64(0))
}

// TestEnqueueWithPriority tests that priority is correctly set
func TestEnqueueWithPriority(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	pool, queries := setupTestDB(t)
	defer pool.Close()

	// Enqueue 3 games with different priorities
	game1 := uuid.New().String()
	game2 := uuid.New().String()
	game3 := uuid.New().String()

	err := analysis.EnqueueGameForAnalysis(ctx, queries, game1, 0) // Low priority
	is.NoErr(err)

	err = analysis.EnqueueGameForAnalysis(ctx, queries, game2, 10) // High priority
	is.NoErr(err)

	err = analysis.EnqueueGameForAnalysis(ctx, queries, game3, 5) // Medium priority
	is.NoErr(err)

	// Claim jobs and verify they come out in priority order
	testUserUUID := pgtype.Text{String: "test-uuid-1", Valid: true}

	job1, err := queries.ClaimNextJob(ctx, testUserUUID)
	is.NoErr(err)
	is.Equal(job1.GameID, game2) // Highest priority first

	job2, err := queries.ClaimNextJob(ctx, testUserUUID)
	is.NoErr(err)
	is.Equal(job2.GameID, game3) // Medium priority second

	job3, err := queries.ClaimNextJob(ctx, testUserUUID)
	is.NoErr(err)
	is.Equal(job3.GameID, game1) // Lowest priority last
}

// TestClaimNextJob tests claiming jobs from the queue
func TestClaimNextJob(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	pool, queries := setupTestDB(t)
	defer pool.Close()

	createTestUsers(t, pool)

	// Create a job
	gameID := uuid.New().String()
	err := analysis.EnqueueGameForAnalysis(ctx, queries, gameID, 0)
	is.NoErr(err)

	// Claim the job
	testUserUUID := pgtype.Text{String: "test-uuid-1", Valid: true}
	job, err := queries.ClaimNextJob(ctx, testUserUUID)
	is.NoErr(err)
	is.Equal(job.GameID, gameID)

	// Verify job is claimed
	jobStatus, err := queries.GetJobByGameID(ctx, gameID)
	is.NoErr(err)
	is.Equal(jobStatus.Status, "claimed")

	// Try to claim another job - should get error (no jobs available)
	_, err = queries.ClaimNextJob(ctx, testUserUUID)
	is.True(err != nil) // Should be no jobs available
}

// TestHeartbeat tests updating job heartbeat
func TestHeartbeat(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	pool, queries := setupTestDB(t)
	defer pool.Close()

	createTestUsers(t, pool)

	// Create and claim a job
	gameID := uuid.New().String()
	err := analysis.EnqueueGameForAnalysis(ctx, queries, gameID, 0)
	is.NoErr(err)

	testUserUUID := pgtype.Text{String: "test-uuid-1", Valid: true}
	job, err := queries.ClaimNextJob(ctx, testUserUUID)
	is.NoErr(err)

	// Update heartbeat
	err = queries.UpdateHeartbeat(ctx, models.UpdateHeartbeatParams{
		ID:                job.ID,
		ClaimedByUserUuid: testUserUUID,
	})
	is.NoErr(err)

	// Verify status changed to "processing"
	jobStatus, err := queries.GetJobByGameID(ctx, gameID)
	is.NoErr(err)
	is.Equal(jobStatus.Status, "processing")

	// Update heartbeat again - should stay as processing
	err = queries.UpdateHeartbeat(ctx, models.UpdateHeartbeatParams{
		ID:                job.ID,
		ClaimedByUserUuid: testUserUUID,
	})
	is.NoErr(err)

	jobStatus, err = queries.GetJobByGameID(ctx, gameID)
	is.NoErr(err)
	is.Equal(jobStatus.Status, "processing")
}

// TestCompleteJob tests completing a job with results
func TestCompleteJob(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	pool, queries := setupTestDB(t)
	defer pool.Close()

	createTestUsers(t, pool)

	// Create and claim a job
	gameID := uuid.New().String()
	err := analysis.EnqueueGameForAnalysis(ctx, queries, gameID, 0)
	is.NoErr(err)

	testUserUUID := pgtype.Text{String: "test-uuid-1", Valid: true}
	job, err := queries.ClaimNextJob(ctx, testUserUUID)
	is.NoErr(err)

	// Complete the job with mock result
	mockResult := []byte(`{"turns": [{"equity": 0.5}], "player_summaries": [{}, {}]}`)

	completedJob, err := queries.CompleteJob(ctx, models.CompleteJobParams{
		Result:            mockResult,
		ID:                job.ID,
		ClaimedByUserUuid: testUserUUID,
	})
	is.NoErr(err)
	is.True(completedJob.DurationMs >= 0) // Should have a duration

	// Verify job is completed
	jobStatus, err := queries.GetJobByGameID(ctx, gameID)
	is.NoErr(err)
	is.Equal(jobStatus.Status, "completed")
	is.True(len(jobStatus.Result) > 0)
}

// TestReclaimStaleJobs tests that stale jobs are reclaimed
func TestReclaimStaleJobs(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	pool, queries := setupTestDB(t)
	defer pool.Close()

	createTestUsers(t, pool)

	// Create and claim a job
	gameID := uuid.New().String()
	err := analysis.EnqueueGameForAnalysis(ctx, queries, gameID, 0)
	is.NoErr(err)

	testUserUUID := pgtype.Text{String: "test-uuid-1", Valid: true}
	job, err := queries.ClaimNextJob(ctx, testUserUUID)
	is.NoErr(err)

	// Manually set heartbeat to 3 minutes ago (past the 2 minute timeout)
	_, err = pool.Exec(ctx, `
		UPDATE analysis_jobs
		SET heartbeat_at = NOW() - INTERVAL '3 minutes'
		WHERE id = $1
	`, job.ID)
	is.NoErr(err)

	// Reclaim stale jobs
	err = queries.ReclaimStaleJobs(ctx)
	is.NoErr(err)

	// Verify job is back to pending
	jobStatus, err := queries.GetJobByGameID(ctx, gameID)
	is.NoErr(err)
	is.Equal(jobStatus.Status, "pending")

	// Verify retry count was incremented
	var retryCount int
	err = pool.QueryRow(ctx, `SELECT retry_count FROM analysis_jobs WHERE id = $1`, job.ID).Scan(&retryCount)
	is.NoErr(err)
	is.Equal(retryCount, 1)
}

// TestMaxRetries tests that jobs fail after max retries
func TestMaxRetries(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	pool, queries := setupTestDB(t)
	defer pool.Close()

	createTestUsers(t, pool)

	// Create and claim a job
	gameID := uuid.New().String()
	err := analysis.EnqueueGameForAnalysis(ctx, queries, gameID, 0)
	is.NoErr(err)

	testUserUUID := pgtype.Text{String: "test-uuid-1", Valid: true}
	job, err := queries.ClaimNextJob(ctx, testUserUUID)
	is.NoErr(err)

	// Set retry count to max (3) and make it stale
	_, err = pool.Exec(ctx, `
		UPDATE analysis_jobs
		SET retry_count = 3,
		    heartbeat_at = NOW() - INTERVAL '3 minutes'
		WHERE id = $1
	`, job.ID)
	is.NoErr(err)

	// Reclaim stale jobs
	err = queries.ReclaimStaleJobs(ctx)
	is.NoErr(err)

	// Verify job is marked as failed
	jobStatus, err := queries.GetJobByGameID(ctx, gameID)
	is.NoErr(err)
	is.Equal(jobStatus.Status, "failed")
	is.True(jobStatus.ErrorMessage.Valid)
	is.True(len(jobStatus.ErrorMessage.String) > 0)
}

// TestQueuePosition tests getting position in queue
func TestQueuePosition(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	pool, queries := setupTestDB(t)
	defer pool.Close()

	// Create 5 jobs with different priorities and times
	game1 := uuid.New().String()
	game2 := uuid.New().String()
	game3 := uuid.New().String()

	jobID1, err := queries.CreateAnalysisJob(ctx, models.CreateAnalysisJobParams{
		GameID:     game1,
		ConfigJson: []byte(`{}`),
		Priority:   pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	time.Sleep(10 * time.Millisecond) // Ensure different created_at times

	jobID2, err := queries.CreateAnalysisJob(ctx, models.CreateAnalysisJobParams{
		GameID:     game2,
		ConfigJson: []byte(`{}`),
		Priority:   pgtype.Int4{Int32: 5, Valid: true},
	})
	is.NoErr(err)

	time.Sleep(10 * time.Millisecond)

	jobID3, err := queries.CreateAnalysisJob(ctx, models.CreateAnalysisJobParams{
		GameID:     game3,
		ConfigJson: []byte(`{}`),
		Priority:   pgtype.Int4{Int32: 5, Valid: true},
	})
	is.NoErr(err)

	// Check queue positions
	// jobID2 should be position 1 (priority 5, oldest)
	// jobID3 should be position 2 (priority 5, newer)
	// jobID1 should be position 3 (priority 0)

	pos1, err := queries.GetQueuePosition(ctx, jobID1)
	is.NoErr(err)
	is.Equal(pos1, int32(3))

	pos2, err := queries.GetQueuePosition(ctx, jobID2)
	is.NoErr(err)
	is.Equal(pos2, int32(1))

	pos3, err := queries.GetQueuePosition(ctx, jobID3)
	is.NoErr(err)
	is.Equal(pos3, int32(2))
}

// NOTE: Integration test for "league game finishes -> gets enqueued"
//
// The above tests verify the analysis queue mechanics work correctly.
// The actual integration of "league game ends -> EnqueueGameForAnalysis is called"
// happens in pkg/gameplay/end.go:254-262:
//
//     if g.LeagueDivisionID != nil {
//         const priority = 0
//         err = analysis.EnqueueGameForAnalysis(ctx, stores.Queries, g.GameID(), priority)
//         ...
//     }
//
// This integration is tested manually and in production. A full automated integration
// test would require setting up: UserStore, GameStore (cache), game entities, and the
// complete game-ending flow, which is beyond the scope of these focused queue tests.
