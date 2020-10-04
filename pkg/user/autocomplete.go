package user

import (
	"context"

	pb "github.com/domino14/liwords/rpc/api/proto/user_service"
)

type AutocompleteService struct {
	userStore Store
}

func NewAutocompleteService(u Store) *AutocompleteService {
	return &AutocompleteService{userStore: u}
}

func (as *AutocompleteService) GetCompletion(ctx context.Context, req *pb.UsernameSearchRequest) (*pb.UsernameSearchResponse, error) {
	usernames, err := as.userStore.UsernamesByPrefix(ctx, req.Prefix)
	if err != nil {
		return nil, err
	}
	return &pb.UsernameSearchResponse{
		Usernames: usernames,
	}, nil
}
