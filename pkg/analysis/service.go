package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	macondo "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/analysis_service"
)

type AnalysisService struct {
	userStore user.Store
	queries   *models.Queries
}

func NewAnalysisService(userStore user.Store, queries *models.Queries) *AnalysisService {
	return &AnalysisService{
		userStore: userStore,
		queries:   queries,
	}
}

func (s *AnalysisService) ClaimJob(
	ctx context.Context,
	req *connect.Request[pb.ClaimJobRequest],
) (*connect.Response[pb.ClaimJobResponse], error) {

	// Authenticate via API key (from middleware)
	apiKey, err := apiserver.GetAPIKey(ctx)
	if err != nil {
		return nil, apiserver.Unauthenticated("API key required")
	}

	user, err := s.userStore.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, apiserver.Unauthenticated("invalid API key")
	}

	// Claim next job
	userUUID := pgtype.Text{String: user.UUID, Valid: true}
	job, err := s.queries.ClaimNextJob(ctx, userUUID)
	if err != nil {
		// No jobs available (or other error - treat as no jobs)
		return connect.NewResponse(&pb.ClaimJobResponse{
			NoJobs: true,
		}), nil
	}

	// Parse config JSON
	var config pb.AnalysisConfig
	if err := json.Unmarshal(job.ConfigJson, &config); err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("invalid config: %w", err))
	}

	log.Info().
		Str("job_id", job.ID.String()).
		Str("game_id", job.GameID).
		Str("user", user.Username).
		Msg("job claimed")

	return connect.NewResponse(&pb.ClaimJobResponse{
		NoJobs: false,
		JobId:  job.ID.String(),
		GameId: job.GameID,
		Config: &config,
	}), nil
}

func (s *AnalysisService) Heartbeat(
	ctx context.Context,
	req *connect.Request[pb.HeartbeatRequest],
) (*connect.Response[pb.HeartbeatResponse], error) {

	apiKey, err := apiserver.GetAPIKey(ctx)
	if err != nil {
		return nil, apiserver.Unauthenticated("API key required")
	}

	user, err := s.userStore.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, apiserver.Unauthenticated("invalid API key")
	}

	jobID, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid job_id")
	}

	userUUID := pgtype.Text{String: user.UUID, Valid: true}
	err = s.queries.UpdateHeartbeat(ctx, models.UpdateHeartbeatParams{
		ID:                jobID,
		ClaimedByUserUuid: userUUID,
	})

	if err != nil {
		// Job was reclaimed or doesn't exist
		return connect.NewResponse(&pb.HeartbeatResponse{
			Continue: false,
		}), nil
	}

	return connect.NewResponse(&pb.HeartbeatResponse{
		Continue: true,
	}), nil
}

func (s *AnalysisService) SubmitResult(
	ctx context.Context,
	req *connect.Request[pb.SubmitResultRequest],
) (*connect.Response[pb.SubmitResultResponse], error) {

	apiKey, err := apiserver.GetAPIKey(ctx)
	if err != nil {
		return nil, apiserver.Unauthenticated("API key required")
	}

	user, err := s.userStore.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, apiserver.Unauthenticated("invalid API key")
	}

	jobID, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid job_id")
	}

	resultProto := req.Msg.ResultProto

	// Validate protobuf can be unmarshaled
	var result macondo.GameAnalysisResult
	if err := proto.Unmarshal(resultProto, &result); err != nil {
		return connect.NewResponse(&pb.SubmitResultResponse{
			Accepted: false,
			Error:    "invalid protobuf",
		}), nil
	}

	// Basic validation
	if len(result.Turns) == 0 {
		return connect.NewResponse(&pb.SubmitResultResponse{
			Accepted: false,
			Error:    "result has no turns",
		}), nil
	}

	if len(result.PlayerSummaries) != 2 {
		return connect.NewResponse(&pb.SubmitResultResponse{
			Accepted: false,
			Error:    "result must have 2 player summaries",
		}), nil
	}

	// Store result
	userUUID := pgtype.Text{String: user.UUID, Valid: true}
	durationMS, err := s.queries.CompleteJob(ctx, models.CompleteJobParams{
		ResultProto:       resultProto,
		ID:                jobID,
		ClaimedByUserUuid: userUUID,
	})

	if err != nil {
		return connect.NewResponse(&pb.SubmitResultResponse{
			Accepted: false,
			Error:    "job not found or already completed",
		}), nil
	}

	log.Info().
		Str("job_id", jobID.String()).
		Str("user", user.Username).
		Int32("duration_ms", durationMS).
		Msg("result accepted")

	return connect.NewResponse(&pb.SubmitResultResponse{
		Accepted: true,
	}), nil
}

// StartReclaimWorker reclaims stale jobs in background
func (s *AnalysisService) StartReclaimWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.queries.ReclaimStaleJobs(ctx); err != nil {
				log.Error().Err(err).Msg("failed to reclaim stale jobs")
			}
		}
	}
}

// EnqueueGameForAnalysis creates an analysis job for a completed game
func EnqueueGameForAnalysis(ctx context.Context, queries *models.Queries, gameID string, priority int) error {
	// Default analysis configuration
	config := map[string]interface{}{
		"sim_plays_early_mid":        40,
		"sim_plies_early_mid":        5,
		"sim_stop_early_mid":         99,
		"sim_plays_early_preendgame": 80,
		"sim_plies_early_preendgame": 10,
		"sim_stop_early_preendgame":  99,
		"peg_early_cutoff":           true,
		"threads":                    0, // worker chooses
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	priorityPG := pgtype.Int4{Int32: int32(priority), Valid: true}
	jobID, err := queries.CreateAnalysisJob(ctx, models.CreateAnalysisJobParams{
		GameID:     gameID,
		ConfigJson: configJSON,
		Priority:   priorityPG,
	})

	if err != nil {
		return fmt.Errorf("failed to create analysis job: %w", err)
	}

	log.Info().
		Str("job_id", jobID.String()).
		Str("game_id", gameID).
		Int("priority", priority).
		Msg("enqueued game for analysis")

	return nil
}
