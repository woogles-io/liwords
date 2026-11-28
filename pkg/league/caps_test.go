package league

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	leaguestore "github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/league_service"
)

type seasonsStore struct {
	leaguestore.Store
	limit int32
}

func (s *seasonsStore) GetRecentSeasons(ctx context.Context, arg models.GetRecentSeasonsParams) ([]models.LeagueSeason, error) {
	s.limit = arg.Limit
	return nil, nil
}

func TestRecentSeasonsLimit(t *testing.T) {
	store := &seasonsStore{}
	ls := &LeagueService{store: store}
	for _, c := range []struct{ asked, want int32 }{
		{0, 2},
		{2, 2},
		{maxRecentSeasons, maxRecentSeasons},
		{1 << 30, maxRecentSeasons},
	} {
		_, err := ls.GetRecentSeasons(context.Background(), connect.NewRequest(&pb.GetRecentSeasonsRequest{
			LeagueId: uuid.NewString(), Limit: c.asked,
		}))
		if err != nil {
			t.Fatal(err)
		}
		if store.limit != c.want {
			t.Errorf("limit %d: store got %d, want %d", c.asked, store.limit, c.want)
		}
	}
}
