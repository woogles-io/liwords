package league

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/league_service"
)

// An anonymous caller must be turned away before the league is looked up.
// The service has no stores, so reaching one would panic.
func TestBootstrapSeasonNeedsAuth(t *testing.T) {
	is := is.New(t)
	ls := &LeagueService{}
	_, err := ls.BootstrapSeason(context.Background(),
		connect.NewRequest(&pb.BootstrapSeasonRequest{LeagueId: "some-league"}))
	is.True(err != nil)
	is.Equal(connect.CodeOf(err), connect.CodeUnauthenticated)
}
