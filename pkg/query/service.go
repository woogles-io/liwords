package query

import (
	"context"

	"github.com/domino14/liwords/pkg/gameplay"
	pb "github.com/domino14/liwords/rpc/api/proto/query_service"
)

type QueryService struct {
	GameStore gameplay.GameStore
}

func NewModService(gs gameplay.GameStore) *QueryService {
	return &QueryService{gs}
}

func (ms *QueryService) GetGames(ctx context.Context, req *pb.GetGamesRequest) (*pb.GetGamesResponse, error) {
	return &pb.GetGamesResponse{}, nil
}
