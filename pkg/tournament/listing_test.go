package tournament_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/matryer/is"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/tournament"
	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

// The listings load tournaments with pairings and standings stripped from
// their divisions; the metadata they return must match a full load.
func TestPastTournamentsMetadataMatchesFullLoad(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	stores, cfg := recreateDB()
	defer stores.Disconnect()
	tstore, us := stores.TournamentStore, stores.UserStore

	start := time.Now().Add(-48 * time.Hour).Truncate(time.Microsecond)
	end := time.Now().Add(-24 * time.Hour).Truncate(time.Microsecond)
	ty, err := tournament.NewTournament(ctx, tstore, tournamentName, "desc",
		makeTournamentPersons(map[string]int32{"Kieran:Kieran": 0}),
		entity.TypeStandard, "", "/tournament/listing", &start, &end, 0, false)
	is.NoErr(err)

	is.NoErr(tournament.AddDivision(ctx, tstore, ty.UUID, divOneName))
	is.NoErr(tournament.AddPlayers(ctx, tstore, us, ty.UUID, divOneName,
		makeTournamentPersons(map[string]int32{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100})))
	is.NoErr(tournament.SetDivisionControls(ctx, tstore, ty.UUID, divOneName, makeControls()))
	is.NoErr(tournament.SetRoundControls(ctx, tstore, ty.UUID, divOneName, makeRoundControls()))
	is.NoErr(tournament.StartRoundCountdown(ctx, tstore, ty.UUID, divOneName, 0))

	svc := tournament.NewTournamentService(tstore, us, cfg, nil, stores.Queries)
	full, err := svc.GetTournamentMetadata(ctx, connect.NewRequest(&pb.GetTournamentMetadataRequest{Id: ty.UUID}))
	is.NoErr(err)
	is.Equal(full.Msg.Metadata.RegistrantCount, int32(4))

	past, err := svc.GetPastTournaments(ctx, connect.NewRequest(&pb.GetPastTournamentsRequest{}))
	is.NoErr(err)
	is.Equal(len(past.Msg.Tournaments), 1)
	if !proto.Equal(past.Msg.Tournaments[0], full.Msg.Metadata) {
		t.Fatalf("listing metadata differs from full load:\nlisting: %v\nfull:    %v", past.Msg.Tournaments[0], full.Msg.Metadata)
	}
}
