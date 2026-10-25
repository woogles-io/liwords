package tournament

import (
	"context"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/user"
	"github.com/woogles-io/liwords/pkg/utilities"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

type countingUserStore struct {
	user.Store
	gets    map[string]int
	actions map[string]int
	banned  map[string]bool
}

func (s *countingUserStore) Get(ctx context.Context, username string) (*entity.User, error) {
	s.gets[username]++
	return &entity.User{Username: username, UUID: "uuid-" + username}, nil
}

func (s *countingUserStore) GetActions(ctx context.Context, uuid string) (map[string]*ms.ModAction, error) {
	s.actions[uuid]++
	if s.banned[uuid] {
		t := ms.ModActionType_SUSPEND_ACCOUNT
		return map[string]*ms.ModAction{t.String(): {Type: t}}, nil
	}
	return map[string]*ms.ModAction{}, nil
}

func TestCensorRecentGamesLoadsEachPlayerOnce(t *testing.T) {
	is := is.New(t)
	us := &countingUserStore{gets: map[string]int{}, actions: map[string]int{},
		banned: map[string]bool{"uuid-bad1": true, "uuid-bad2": true}}

	game := func(p1, p2 string) *ipc.TournamentGameEndedEvent {
		return &ipc.TournamentGameEndedEvent{Players: []*ipc.TournamentGameEndedEvent_Player{
			{Username: p1}, {Username: p2}}}
	}
	rgr := &pb.RecentGamesResponse{Games: []*ipc.TournamentGameEndedEvent{
		game("alice", "bob"), game("bob", "alice"), game("alice", "bad1"),
		game("bad1", "bob"), game("bad1", "bad2"), game("alice", "bob"),
	}}

	is.NoErr(censorRecentGamesResponse(context.Background(), us, rgr))

	for _, name := range []string{"alice", "bob", "bad1", "bad2"} {
		is.Equal(us.gets[name], 1)
		is.Equal(us.actions["uuid-"+name], 1)
	}
	names := func(g *ipc.TournamentGameEndedEvent) [2]string {
		return [2]string{g.Players[0].Username, g.Players[1].Username}
	}
	is.Equal(names(rgr.Games[0]), [2]string{"alice", "bob"})
	is.Equal(names(rgr.Games[2]), [2]string{"alice", utilities.CensoredUsername})
	is.Equal(names(rgr.Games[3]), [2]string{utilities.CensoredUsername, "bob"})
	is.Equal(names(rgr.Games[4]), [2]string{utilities.CensoredUsername, utilities.AnotherCensoredUsername})
}
