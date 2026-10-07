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
	"google.golang.org/protobuf/encoding/protojson"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
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

// insertLeagueGame adds a finished league game between users 1 and 2, with the
// game_players rows the season queries read, and returns its id.
func insertLeagueGame(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	leagueID, seasonID, divisionID uuid.UUID, playedAt time.Time,
	player0Score, player1Score int) string {
	is := is.New(t)

	// Game ids are at most 24 characters.
	gameID := "leaguegame" + uuid.New().String()[:10]
	_, err := pool.Exec(ctx, `
		INSERT INTO games(uuid, created_at, updated_at, player0_id, player1_id, started,
		                  game_end_reason, type, game_request, history, quickdata, timers,
		                  league_id, season_id, league_division_id)
		VALUES ($1, $2, $2, 1, 2, true, $3, 0, '{}', '', '{}', '{}', $4, $5, $6)`,
		gameID, playedAt, int(ipc.GameEndReason_STANDARD), leagueID, seasonID, divisionID)
	is.NoErr(err)

	for _, p := range []struct {
		playerID, index, score, opponentID, opponentScore int
	}{
		{1, 0, player0Score, 2, player1Score},
		{2, 1, player1Score, 1, player0Score},
	} {
		_, err = pool.Exec(ctx, `
			INSERT INTO game_players(game_uuid, player_id, player_index, score, won,
			                         game_end_reason, created_at, game_type, opponent_id,
			                         opponent_score, league_season_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 0, $8, $9, $10)`,
			gameID, p.playerID, p.index, p.score, p.score > p.opponentScore,
			int(ipc.GameEndReason_STANDARD), playedAt, p.opponentID, p.opponentScore, seasonID)
		is.NoErr(err)
	}

	return gameID
}

// completeAnalysis stores result against gameID the way the worker path does,
// through protojson.
func completeAnalysis(t *testing.T, ctx context.Context, queries *models.Queries,
	gameID string, result *macondopb.GameAnalysisResult) []byte {
	is := is.New(t)

	err := analysis.EnqueueGameForAnalysis(ctx, queries, gameID, 0)
	is.NoErr(err)

	return claimAndComplete(t, ctx, queries, result)
}

// claimAndComplete claims the next pending job and completes it with result.
func claimAndComplete(t *testing.T, ctx context.Context, queries *models.Queries,
	result *macondopb.GameAnalysisResult) []byte {
	is := is.New(t)

	workerUUID := pgtype.Text{String: "test-uuid-3", Valid: true}
	job, err := queries.ClaimNextJob(ctx, workerUUID)
	is.NoErr(err)

	resultJSON, err := protojson.Marshal(result)
	is.NoErr(err)

	_, err = queries.CompleteJob(ctx, models.CompleteJobParams{
		Result:            resultJSON,
		ID:                job.ID,
		ClaimedByUserUuid: workerUUID,
	})
	is.NoErr(err)

	return resultJSON
}

func analysisResult(mi0, mi1 float64) *macondopb.GameAnalysisResult {
	return &macondopb.GameAnalysisResult{
		AnalysisVersion: 2,
		PlayerSummaries: []*macondopb.PlayerSummary{
			{PlayerName: "testuser1", MistakeIndex: mi0},
			{PlayerName: "testuser2", MistakeIndex: mi1},
		},
	}
}

// seedStanding writes a standing with the given MI columns, as a drifted
// counter would have left them.
func seedStanding(t *testing.T, ctx context.Context, queries *models.Queries,
	divisionID uuid.UUID, userID int32, totalMI float64, analyzed int32) {
	is := is.New(t)
	err := queries.UpsertStanding(ctx, models.UpsertStandingParams{
		DivisionID:        divisionID,
		UserID:            userID,
		TotalMistakeIndex: pgtype.Float8{Float64: totalMI, Valid: true},
		GamesAnalyzed:     pgtype.Int4{Int32: analyzed, Valid: true},
	})
	is.NoErr(err)
}

// standingMI returns a standing's (total mistake index, games analyzed).
func standingMI(t *testing.T, ctx context.Context, queries *models.Queries,
	divisionID uuid.UUID, userID int32) (float64, int32) {
	is := is.New(t)
	st, err := queries.GetPlayerStanding(ctx, models.GetPlayerStandingParams{
		DivisionID: divisionID,
		UserID:     userID,
	})
	is.NoErr(err)
	is.True(st.TotalMistakeIndex.Valid)
	is.True(st.GamesAnalyzed.Valid)
	return st.TotalMistakeIndex.Float64, st.GamesAnalyzed.Int32
}

func assertMI(t *testing.T, ctx context.Context, queries *models.Queries,
	divisionID uuid.UUID, userID int32, wantTotal float64, wantAnalyzed int32) {
	t.Helper()
	total, analyzed := standingMI(t, ctx, queries, divisionID, userID)
	if analyzed != wantAnalyzed || total < wantTotal-1e-9 || total > wantTotal+1e-9 {
		t.Fatalf("user %d: got MI total %v over %d games, want %v over %d",
			userID, total, analyzed, wantTotal, wantAnalyzed)
	}
}

// TestRefreshDivisionMistakeIndex covers the rebuild of standings' MI columns
// from stored analysis results. The old running counters drifted whenever an
// increment and its matching decrement didn't both land: a rejected late
// submission decremented without re-adding, a requeue re-added without
// subtracting. A rebuild has no such pairing to get wrong.
func TestRefreshDivisionMistakeIndex(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	pool, queries := setupTestDB(t)
	defer pool.Close()

	createTestUsers(t, pool)
	leagueID, seasonID, divisionID := createMinimalLeague(t, ctx, queries)

	// Drifted counters, plus a player with no analyzed games.
	seedStanding(t, ctx, queries, divisionID, 1, 99, 19)
	seedStanding(t, ctx, queries, divisionID, 2, 1, 11)
	seedStanding(t, ctx, queries, divisionID, 4, 5, 5)

	now := time.Now()
	gameA := insertLeagueGame(t, ctx, pool, leagueID, seasonID, divisionID, now.Add(-3*time.Hour), 400, 380)
	gameB := insertLeagueGame(t, ctx, pool, leagueID, seasonID, divisionID, now.Add(-2*time.Hour), 450, 300)
	// Never analyzed: must not count.
	insertLeagueGame(t, ctx, pool, leagueID, seasonID, divisionID, now.Add(-time.Hour), 350, 390)

	completeAnalysis(t, ctx, queries, gameA, analysisResult(3.0, 4.0))
	completeAnalysis(t, ctx, queries, gameB, analysisResult(1.5, 2.5))

	is.NoErr(queries.RefreshDivisionMistakeIndex(ctx, divisionID))
	assertMI(t, ctx, queries, divisionID, 1, 4.5, 2)
	assertMI(t, ctx, queries, divisionID, 2, 6.5, 2)
	assertMI(t, ctx, queries, divisionID, 4, 0, 0)

	// Idempotent: a repeat, as from a duplicate or late submission, changes nothing.
	is.NoErr(queries.RefreshDivisionMistakeIndex(ctx, divisionID))
	assertMI(t, ctx, queries, divisionID, 1, 4.5, 2)
	assertMI(t, ctx, queries, divisionID, 2, 6.5, 2)

	// A whole-row upsert (standings recalculation) leaves MI alone.
	seedStanding(t, ctx, queries, divisionID, 1, 0, 0)
	assertMI(t, ctx, queries, divisionID, 1, 4.5, 2)

	// Requeued for reanalysis: the job is pending again but keeps its old
	// result, so the game stays counted at its old value meanwhile.
	is.NoErr(analysis.RequeueJobByGameID(ctx, queries, gameA, 0))
	is.NoErr(queries.RefreshDivisionMistakeIndex(ctx, divisionID))
	assertMI(t, ctx, queries, divisionID, 1, 4.5, 2)
	assertMI(t, ctx, queries, divisionID, 2, 6.5, 2)

	// The reanalysis replaces the old result rather than adding to it.
	claimAndComplete(t, ctx, queries, analysisResult(5.0, 1.0))
	is.NoErr(queries.RefreshDivisionMistakeIndex(ctx, divisionID))
	assertMI(t, ctx, queries, divisionID, 1, 6.5, 2)
	assertMI(t, ctx, queries, divisionID, 2, 3.5, 2)

	// A second job for the same game: only the newest counts, once.
	completeAnalysis(t, ctx, queries, gameB, analysisResult(2.0, 2.0))
	is.NoErr(queries.RefreshDivisionMistakeIndex(ctx, divisionID))
	assertMI(t, ctx, queries, divisionID, 1, 7.0, 2)
	assertMI(t, ctx, queries, divisionID, 2, 3.0, 2)
}

// TestPerfectGameReadsBackAsAnalyzed covers a mistake index of 0 -- a perfect
// game. protojson drops a field holding its zero value, so the stored result
// has no mistakeIndex key at all, and reading presence off that key reported
// the game as unanalyzed: the game history modal showed "-" and the season
// recalculation dropped the game from both players' analyzed counts.
func TestPerfectGameReadsBackAsAnalyzed(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	pool, queries := setupTestDB(t)
	defer pool.Close()

	createTestUsers(t, pool)
	leagueID, seasonID, divisionID := createMinimalLeague(t, ctx, queries)

	now := time.Now()
	analyzed := insertLeagueGame(t, ctx, pool, leagueID, seasonID, divisionID,
		now.Add(-time.Hour), 500, 288)
	insertLeagueGame(t, ctx, pool, leagueID, seasonID, divisionID,
		now.Add(-2*time.Hour), 400, 380)

	resultJSON := completeAnalysis(t, ctx, queries, analyzed, &macondopb.GameAnalysisResult{
		AnalysisVersion: 2,
		PlayerSummaries: []*macondopb.PlayerSummary{
			{PlayerName: "testuser1", MistakeIndex: 0},
			{PlayerName: "testuser2", MistakeIndex: 2.2},
		},
	})

	// The premise: the perfect player's summary carries no mistakeIndex key.
	var stored struct {
		PlayerSummaries []map[string]any `json:"playerSummaries"`
	}
	is.NoErr(json.Unmarshal(resultJSON, &stored))
	_, present := stored.PlayerSummaries[0]["mistakeIndex"]
	is.True(!present)

	seasonUUID := pgtype.UUID{Bytes: seasonID, Valid: true}
	// The perfect game is analyzed, and its index is 0 rather than missing.
	// Games are newest first, so the unanalyzed one is second.
	perfect, err := queries.GetPlayerSeasonGames(ctx, models.GetPlayerSeasonGamesParams{
		UserUuid: "test-uuid-1",
		SeasonID: seasonUUID,
	})
	is.NoErr(err)
	is.Equal(len(perfect), 2)
	is.Equal(perfect[0].GameUuid, analyzed)
	is.Equal(perfect[0].HasMistakeIndex, true)
	is.Equal(perfect[0].PlayerMistakeIndex, float64(0))
	is.Equal(perfect[1].HasMistakeIndex, false)

	// The opponent's own index still comes from their own summary.
	opponent, err := queries.GetPlayerSeasonGames(ctx, models.GetPlayerSeasonGamesParams{
		UserUuid: "test-uuid-2",
		SeasonID: seasonUUID,
	})
	is.NoErr(err)
	is.Equal(len(opponent), 2)
	is.Equal(opponent[0].HasMistakeIndex, true)
	is.Equal(opponent[0].PlayerMistakeIndex, 2.2)

	// The standings rebuild counts the game for both players.
	seedStanding(t, ctx, queries, divisionID, 1, 0, 0)
	seedStanding(t, ctx, queries, divisionID, 2, 0, 0)
	is.NoErr(queries.RefreshDivisionMistakeIndex(ctx, divisionID))
	assertMI(t, ctx, queries, divisionID, 1, 0, 1)
	assertMI(t, ctx, queries, divisionID, 2, 2.2, 1)
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
