package tournament

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

type listStore struct {
	TournamentStore
	limit         int32
	count, offset int
}

func (s *listStore) GetPastTournaments(ctx context.Context, limit int32) ([]*entity.Tournament, error) {
	s.limit = limit
	return nil, nil
}

func (s *listStore) GetRecentClubSessions(ctx context.Context, clubID string, numSessions int, offset int) (*pb.ClubSessionsResponse, error) {
	s.count, s.offset = numSessions, offset
	return &pb.ClubSessionsResponse{}, nil
}

func TestPastTournamentsLimit(t *testing.T) {
	store := &listStore{}
	ts := &TournamentService{tournamentStore: store}
	for _, c := range []struct{ asked, want int32 }{
		{0, maxPastTournaments},
		{-1, maxPastTournaments}, // the store treats <= 0 as no limit
		{50, 50},
		{maxPastTournaments + 1, maxPastTournaments},
		{1 << 30, maxPastTournaments},
	} {
		_, err := ts.GetPastTournaments(context.Background(), connect.NewRequest(&pb.GetPastTournamentsRequest{Limit: c.asked}))
		if err != nil {
			t.Fatal(err)
		}
		if store.limit != c.want {
			t.Errorf("limit %d: store got %d, want %d", c.asked, store.limit, c.want)
		}
	}
}

func TestRecentClubSessionsLimit(t *testing.T) {
	store := &listStore{}
	ts := &TournamentService{tournamentStore: store}
	for _, c := range []struct {
		count, offset         int32
		wantCount, wantOffset int
	}{
		{0, 0, maxClubSessions, 0},
		{20, 40, 20, 40},
		{maxClubSessions + 1, 0, maxClubSessions, 0},
		{-5, -5, maxClubSessions, 0}, // Postgres refuses a negative offset
	} {
		_, err := ts.GetRecentClubSessions(context.Background(), connect.NewRequest(&pb.RecentClubSessionsRequest{
			Id: "club", Count: c.count, Offset: c.offset,
		}))
		if err != nil {
			t.Fatal(err)
		}
		if store.count != c.wantCount || store.offset != c.wantOffset {
			t.Errorf("count %d offset %d: store got %d, %d", c.count, c.offset, store.count, store.offset)
		}
	}
}
