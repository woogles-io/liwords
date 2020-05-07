package server

import (
	"context"

	pb "github.com/domino14/crosswords/rpc/api/proto"
)

type GameService struct{}

func (s *GameService) NewGame(ctx context.Context, req *pb.NewGameRequest) (*pb.NewGameResponse, error) {
	return nil, nil
}
