package user

import (
	"context"

	"github.com/twitchtv/twirp"

	pb "github.com/domino14/liwords/rpc/api/proto/user_service"
)

type AutocompleteService struct {
	userStore Store
}

func NewAutocompleteService(u Store) *AutocompleteService {
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
