package services

import (
	"context"

	"github.com/twitchtv/twirp"

	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

type AutocompleteService struct {
	userStore user.Store
}

func NewAutocompleteService(u user.Store) *AutocompleteService {
	return &AutocompleteService{userStore: u}
}

func (as *AutocompleteService) GetCompletion(ctx context.Context, req *pb.UsernameSearchRequest) (*pb.UsernameSearchResponse, error) {
	users, err := as.userStore.UsersByPrefix(ctx, req.Prefix)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &pb.UsernameSearchResponse{
		Users: users,
	}, nil
}
