package analysis

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/analysis_service"
)

type AnalysisAdminService struct {
	userStore user.Store
	queries   *models.Queries
}

func NewAnalysisAdminService(userStore user.Store, queries *models.Queries) *AnalysisAdminService {
	return &AnalysisAdminService{
		userStore: userStore,
		queries:   queries,
	}
}

func (s *AnalysisAdminService) GetAdminStats(
	ctx context.Context,
	req *connect.Request[pb.GetAdminStatsRequest],
) (*connect.Response[pb.GetAdminStatsResponse], error) {

	_, err := apiserver.AuthenticateWithPermission(ctx, s.userStore, s.queries, rbac.AdminAllAccess)
	if err != nil {
		return nil, err
	}

	stats, err := s.queries.GetAdminAnalysisStats(ctx)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get analysis stats: %w", err))
	}

	leaderboardRows, err := s.queries.GetAnalysisLeaderboard(ctx, 20)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get leaderboard: %w", err))
	}

	leaderboard := make([]*pb.LeaderboardEntry, 0, len(leaderboardRows))
	for _, row := range leaderboardRows {
		leaderboard = append(leaderboard, &pb.LeaderboardEntry{
			Username:      row.Username.String,
			AnalysisCount: int32(row.AnalysisCount),
		})
	}

	return connect.NewResponse(&pb.GetAdminStatsResponse{
		TotalCompleted:  int32(stats.TotalCompleted),
		PendingCount:    int32(stats.PendingCount),
		ProcessingCount: int32(stats.ProcessingCount),
		Leaderboard:     leaderboard,
	}), nil
}

func (s *AnalysisAdminService) ListAnalyzedGames(
	ctx context.Context,
	req *connect.Request[pb.ListAnalyzedGamesRequest],
) (*connect.Response[pb.ListAnalyzedGamesResponse], error) {

	_, err := apiserver.AuthenticateWithPermission(ctx, s.userStore, s.queries, rbac.AdminAllAccess)
	if err != nil {
		return nil, err
	}

	pageSize := req.Msg.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50
	}
	page := req.Msg.Page
	if page < 0 {
		page = 0
	}

	jobs, err := s.queries.GetCompletedJobsList(ctx, models.GetCompletedJobsListParams{
		Limit:  pageSize,
		Offset: page * pageSize,
	})
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to list analyzed games: %w", err))
	}

	total, err := s.queries.GetTotalCompletedCount(ctx)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get total count: %w", err))
	}

	games := make([]*pb.AnalyzedGameSummary, 0, len(jobs))
	for _, job := range jobs {
		var createdAtMs int64
		if job.CreatedAt.Valid {
			createdAtMs = job.CreatedAt.Time.UnixMilli()
		}
		var completedAtMs int64
		if job.CompletedAt.Valid {
			completedAtMs = job.CompletedAt.Time.UnixMilli()
		}

		games = append(games, &pb.AnalyzedGameSummary{
			JobId:               job.JobID.String(),
			GameId:              job.GameID,
			CreatedAtMs:         createdAtMs,
			CompletedAtMs:       completedAtMs,
			RequestType:         job.RequestType,
			RequestedByUsername: job.RequestedByUsername,
		})
	}

	return connect.NewResponse(&pb.ListAnalyzedGamesResponse{
		Games: games,
		Total: int32(total),
	}), nil
}
