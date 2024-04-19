package services

import (
	"context"

	"connectrpc.com/connect"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

type AutocompleteService struct {
	userStore user.Store
}

func NewAutocompleteService(u user.Store) *AutocompleteService {
	return &AutocompleteService{userStore: u}
}

func (as *AutocompleteService) GetCompletion(ctx context.Context, req *connect.Request[pb.UsernameSearchRequest],
) (*connect.Response[pb.UsernameSearchResponse], error) {
	users, err := as.userStore.UsersByPrefix(ctx, req.Msg.Prefix)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(&pb.UsernameSearchResponse{
		Users: users,
	}), nil
}
