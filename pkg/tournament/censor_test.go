package tournament

import (
	"context"
	"testing"

	"github.com/matryer/is"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/user"
	"github.com/woogles-io/liwords/pkg/utilities"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

type countingUserStore struct {
	user.Store
	gets         map[string]int
	batchCalls   int
	batchedUUIDs []string
	actions      map[string]map[string]*ms.ModAction
}

func (s *countingUserStore) Get(ctx context.Context, username string) (*entity.User, error) {
	s.gets[username]++
	return &entity.User{Username: username, UUID: "uuid-" + username}, nil
}

func (s *countingUserStore) GetActionsBatch(ctx context.Context, uuids []string) (map[string]map[string]*ms.ModAction, error) {
	s.batchCalls++
	s.batchedUUIDs = uuids
	return s.actions, nil
}

func TestCensorRecentGamesBatchesModActions(t *testing.T) {
	is := is.New(t)
	suspend := ms.ModActionType_SUSPEND_ACCOUNT.String()
	us := &countingUserStore{gets: map[string]int{}, actions: map[string]map[string]*ms.ModAction{
		"uuid-bad1": {suspend: {}},
		"uuid-bad2": {suspend: {}},
		// Temporary suspensions are not censored.
		"uuid-temp": {suspend: {EndTime: timestamppb.Now()}},
	}}

	player := func(name string) *ipc.TournamentGameEndedEvent_Player {
		return &ipc.TournamentGameEndedEvent_Player{Username: name, UserId: "uuid-" + name}
	}
	game := func(p1, p2 *ipc.TournamentGameEndedEvent_Player) *ipc.TournamentGameEndedEvent {
		return &ipc.TournamentGameEndedEvent{Players: []*ipc.TournamentGameEndedEvent_Player{p1, p2}}
	}
	rgr := &pb.RecentGamesResponse{Games: []*ipc.TournamentGameEndedEvent{
		game(player("alice"), player("bob")),
		game(player("bob"), player("alice")),
		game(player("alice"), player("bad1")),
		game(player("bad1"), player("bob")),
		game(player("bad1"), player("bad2")),
		game(player("temp"), player("alice")),
		// No stored user ID: falls back to a lookup by username.
		game(&ipc.TournamentGameEndedEvent_Player{Username: "carol"}, player("bob")),
	}}

	is.NoErr(censorRecentGamesResponse(context.Background(), us, rgr))

	is.Equal(us.batchCalls, 1)
	is.Equal(us.batchedUUIDs, []string{"uuid-alice", "uuid-bob", "uuid-bad1", "uuid-bad2", "uuid-temp", "uuid-carol"})
	is.Equal(us.gets, map[string]int{"carol": 1})

	names := func(g *ipc.TournamentGameEndedEvent) [2]string {
		return [2]string{g.Players[0].Username, g.Players[1].Username}
	}
	is.Equal(names(rgr.Games[0]), [2]string{"alice", "bob"})
	is.Equal(names(rgr.Games[2]), [2]string{"alice", utilities.CensoredUsername})
	is.Equal(names(rgr.Games[3]), [2]string{utilities.CensoredUsername, "bob"})
	is.Equal(names(rgr.Games[4]), [2]string{utilities.CensoredUsername, utilities.AnotherCensoredUsername})
	is.Equal(names(rgr.Games[5]), [2]string{"temp", "alice"})
	// Censored players' IDs are hidden too.
	is.Equal(rgr.Games[4].Players[0].UserId, utilities.CensoredUsername)
	is.Equal(rgr.Games[4].Players[1].UserId, utilities.AnotherCensoredUsername)
}
