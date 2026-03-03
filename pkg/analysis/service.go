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
	"google.golang.org/protobuf/encoding/protojson"

	macondo "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/analysis_service"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type GameStore interface {
	Get(ctx context.Context, id string) (*entity.Game, error)
}

type AnalysisService struct {
	userStore user.Store
	gameStore GameStore
	queries   *models.Queries
}

func NewAnalysisService(userStore user.Store, gameStore GameStore, queries *models.Queries) *AnalysisService {
	return &AnalysisService{
		userStore: userStore,
		gameStore: gameStore,
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
	if err := protojson.Unmarshal(resultProto, &result); err != nil {
		return connect.NewResponse(&pb.SubmitResultResponse{
			Accepted: false,
			Error:    "invalid protojson",
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
	completedJob, err := s.queries.CompleteJob(ctx, models.CompleteJobParams{
		Result:            resultProto,
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
		Str("game_id", completedJob.GameID).
		Str("user", user.Username).
		Int32("duration_ms", completedJob.DurationMs).
		Msg("result accepted")

	// Update league standings with mistake index if this is a league game
	go s.updateLeagueMistakeIndex(ctx, completedJob.GameID, &result)

	return connect.NewResponse(&pb.SubmitResultResponse{
		Accepted: true,
	}), nil
}

// updateLeagueMistakeIndex updates league standings with mistake index for a completed analysis.
// Runs asynchronously (best-effort) so failures don't affect the SubmitResult response.
func (s *AnalysisService) updateLeagueMistakeIndex(ctx context.Context, gameID string, result *macondo.GameAnalysisResult) {
	gameInfo, err := s.queries.GetGameLeagueInfo(ctx, pgtype.Text{String: gameID, Valid: true})
	if err != nil {
		log.Debug().Str("game_id", gameID).Msg("game not found or not a league game for mistake index update")
		return
	}
	if !gameInfo.LeagueDivisionID.Valid {
		return // not a league game
	}

	divisionID, err := uuid.FromBytes(gameInfo.LeagueDivisionID.Bytes[:])
	if err != nil {
		log.Error().Err(err).Str("game_id", gameID).Msg("failed to parse division UUID for mistake index")
		return
	}

	players := []struct {
		playerID     pgtype.Int4
		mistakeIndex float64
	}{
		{gameInfo.Player0ID, result.PlayerSummaries[0].GetMistakeIndex()},
		{gameInfo.Player1ID, result.PlayerSummaries[1].GetMistakeIndex()},
	}

	for _, p := range players {
		if !p.playerID.Valid {
			continue
		}
		err := s.queries.IncrementStandingMistakeIndex(ctx, models.IncrementStandingMistakeIndexParams{
			DivisionID:        divisionID,
			UserID:            p.playerID.Int32,
			TotalMistakeIndex: pgtype.Float8{Float64: p.mistakeIndex, Valid: true},
		})
		if err != nil {
			log.Error().Err(err).
				Str("game_id", gameID).
				Int32("user_id", p.playerID.Int32).
				Msg("failed to update league mistake index")
		}
	}

	log.Info().
		Str("game_id", gameID).
		Str("division_id", divisionID.String()).
		Msg("updated league standings with mistake index")
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

// RequestAnalysis handles user request to analyze their game
func (s *AnalysisService) RequestAnalysis(
	ctx context.Context,
	req *connect.Request[pb.RequestAnalysisRequest],
) (*connect.Response[pb.RequestAnalysisResponse], error) {

	// Get authenticated user
	user, err := apiserver.AuthUser(ctx, s.userStore)
	if err != nil {
		return nil, err
	}

	gameID := req.Msg.GameId
	if gameID == "" {
		return nil, apiserver.InvalidArg("game_id is required")
	}

	// Fetch the game
	g, err := s.gameStore.Get(ctx, gameID)
	if err != nil {
		return nil, apiserver.InvalidArg("game not found")
	}

	// Check if game has ended
	if g.Playing() != macondo.PlayState_GAME_OVER {
		return connect.NewResponse(&pb.RequestAnalysisResponse{
			Status:  pb.RequestAnalysisResponse_GAME_NOT_ENDED,
			Message: "Game must be completed before requesting analysis",
		}), nil
	}

	// Check authorization:
	// - For regular games: user must be a player
	// - For annotated games: user must be the creator
	isAuthorized := false

	if g.Type == ipc.GameType_ANNOTATED {
		// For annotated games, only the creator can request analysis
		// Creator is stored in annotated_game_metadata table
		owner, err := s.queries.GetGameOwner(ctx, gameID)
		if err == nil && owner.CreatorUuid == user.UUID {
			isAuthorized = true
		}
	} else {
		// For regular games, user must be a player
		for _, playerID := range g.PlayerDBIDs {
			if playerID == user.ID {
				isAuthorized = true
				break
			}
		}
	}

	if !isAuthorized {
		return connect.NewResponse(&pb.RequestAnalysisResponse{
			Status:  pb.RequestAnalysisResponse_NOT_A_PLAYER,
			Message: "You must be a player in this game (or creator for annotated games) to request analysis",
		}), nil
	}

	// Check if variant is supported (only classic for now)
	variantName := g.GameReq.Rules.VariantName
	if variantName != "classic" && variantName != "" {
		return connect.NewResponse(&pb.RequestAnalysisResponse{
			Status:  pb.RequestAnalysisResponse_INVALID_VARIANT,
			Message: "Analysis is only available for classic games",
		}), nil
	}

	// Check for existing analysis job
	existingJob, err := s.queries.GetJobByGameID(ctx, gameID)
	if err == nil {
		// Job exists
		queuePos := int32(0)
		if existingJob.Status == "pending" {
			// Get queue position
			pos, err := s.queries.GetQueuePosition(ctx, existingJob.ID)
			if err == nil {
				queuePos = int32(pos)
			}
		}

		return connect.NewResponse(&pb.RequestAnalysisResponse{
			Status:        pb.RequestAnalysisResponse_ALREADY_REQUESTED,
			Message:       "Analysis has already been requested for this game",
			JobId:         existingJob.ID.String(),
			QueuePosition: queuePos,
		}), nil
	}

	// Check rate limit (5 per day)
	requestCount, err := s.queries.GetUserRequestCountToday(ctx, user.UUID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to check rate limit: %w", err))
	}

	if requestCount >= 5 {
		return connect.NewResponse(&pb.RequestAnalysisResponse{
			Status:  pb.RequestAnalysisResponse_RATE_LIMITED,
			Message: "You have reached the daily limit of 5 analysis requests. Please try again tomorrow.",
		}), nil
	}

	// Create analysis job with higher priority for user requests
	config := map[string]interface{}{
		"sim_plays_early_mid":        40,
		"sim_plies_early_mid":        5,
		"sim_stop_early_mid":         99,
		"sim_plays_early_preendgame": 80,
		"sim_plies_early_preendgame": 10,
		"sim_stop_early_preendgame":  99,
		"peg_early_cutoff":           true,
		"threads":                    0,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to marshal config: %w", err))
	}

	// User requests get priority 5 (higher than automatic league analysis at 0)
	priority := pgtype.Int4{Int32: 5, Valid: true}
	requestedBy := pgtype.Text{String: user.UUID, Valid: true}

	jobID, err := s.queries.CreateUserRequestedJob(ctx, models.CreateUserRequestedJobParams{
		GameID:              gameID,
		ConfigJson:          configJSON,
		Priority:            priority,
		RequestedByUserUuid: requestedBy,
	})

	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to create analysis job: %w", err))
	}

	// Record the request for rate limiting
	err = s.queries.RecordUserAnalysisRequest(ctx, models.RecordUserAnalysisRequestParams{
		UserUuid: user.UUID,
		GameID:   gameID,
		JobID:    pgtype.UUID{Bytes: jobID, Valid: true},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to record user analysis request")
		// Don't fail the request, job was created successfully
	}

	// Get queue position
	queuePos, err := s.queries.GetQueuePosition(ctx, jobID)
	if err != nil {
		queuePos = 1 // fallback
	}

	log.Info().
		Str("job_id", jobID.String()).
		Str("game_id", gameID).
		Str("user", user.Username).
		Int32("queue_position", queuePos).
		Msg("user requested analysis")

	return connect.NewResponse(&pb.RequestAnalysisResponse{
		Status:        pb.RequestAnalysisResponse_SUCCESS,
		Message:       fmt.Sprintf("Analysis queued successfully! You are #%d in the queue.", queuePos),
		JobId:         jobID.String(),
		QueuePosition: queuePos,
	}), nil
}

// GetAnalysisStatus returns the status of an analysis job for a game
func (s *AnalysisService) GetAnalysisStatus(
	ctx context.Context,
	req *connect.Request[pb.GetAnalysisStatusRequest],
) (*connect.Response[pb.GetAnalysisStatusResponse], error) {

	gameID := req.Msg.GameId
	if gameID == "" {
		return nil, apiserver.InvalidArg("game_id is required")
	}

	job, err := s.queries.GetJobByGameID(ctx, gameID)
	if err != nil {
		return connect.NewResponse(&pb.GetAnalysisStatusResponse{
			Status: pb.GetAnalysisStatusResponse_NOT_FOUND,
		}), nil
	}

	var status pb.GetAnalysisStatusResponse_JobStatus
	switch job.Status {
	case "pending":
		status = pb.GetAnalysisStatusResponse_PENDING
	case "claimed", "processing":
		status = pb.GetAnalysisStatusResponse_PROCESSING
	case "completed":
		status = pb.GetAnalysisStatusResponse_COMPLETED
	case "failed":
		status = pb.GetAnalysisStatusResponse_FAILED
	default:
		status = pb.GetAnalysisStatusResponse_NOT_FOUND
	}

	queuePos := int32(0)
	if job.Status == "pending" {
		pos, err := s.queries.GetQueuePosition(ctx, job.ID)
		if err == nil {
			queuePos = int32(pos)
		}
	}

	errorMsg := ""
	if job.ErrorMessage.Valid {
		errorMsg = job.ErrorMessage.String
	}

	return connect.NewResponse(&pb.GetAnalysisStatusResponse{
		Status:        status,
		JobId:         job.ID.String(),
		QueuePosition: queuePos,
		ErrorMessage:  errorMsg,
	}), nil
}

// GetAnalysisResult returns the completed analysis result
func (s *AnalysisService) GetAnalysisResult(
	ctx context.Context,
	req *connect.Request[pb.GetAnalysisResultRequest],
) (*connect.Response[pb.GetAnalysisResultResponse], error) {

	gameID := req.Msg.GameId
	if gameID == "" {
		return nil, apiserver.InvalidArg("game_id is required")
	}

	job, err := s.queries.GetJobByGameID(ctx, gameID)
	if err != nil || job.Status != "completed" {
		return connect.NewResponse(&pb.GetAnalysisResultResponse{
			Found: false,
		}), nil
	}

	if len(job.Result) == 0 {
		return connect.NewResponse(&pb.GetAnalysisResultResponse{
			Found: false,
		}), nil
	}

	return connect.NewResponse(&pb.GetAnalysisResultResponse{
		Found:       true,
		ResultProto: job.Result,
	}), nil
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
