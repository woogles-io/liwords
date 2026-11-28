package profile

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

type briefStore struct {
	user.Store
	asked int
}

func (s *briefStore) GetBriefProfiles(ctx context.Context, uuids []string) (map[string]*pb.BriefProfile, error) {
	s.asked = len(uuids)
	return map[string]*pb.BriefProfile{}, nil
}

func TestBriefProfilesCap(t *testing.T) {
	store := &briefStore{}
	ps := &ProfileService{userStore: store}
	ask := func(n int) error {
		ids := make([]string, n)
		_, err := ps.GetBriefProfiles(context.Background(), connect.NewRequest(&pb.BriefProfilesRequest{UserIds: ids}))
		return err
	}

	if err := ask(MaxBriefProfiles + 1); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("over the cap: got %v", err)
	}
	if store.asked != 0 {
		t.Fatal("looked users up despite the cap")
	}
	if err := ask(MaxBriefProfiles); err != nil {
		t.Fatalf("at the cap: %v", err)
	}
	if store.asked != MaxBriefProfiles {
		t.Fatalf("looked up %d users", store.asked)
	}
}
