package tournament

import (
	"context"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"

	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

// The most tournaments GetEnteredTournaments lists of those played in, and
// as many of those registered in.
const enteredTournamentsLimit = 100

// GetEnteredTournaments lists the tournaments and clubs a user played games
// in, the last played first, then the unfinished ones they are registered in
// without a game yet: the caller's own or, for a caller who can see private
// user data (the admin user lookup), those of the user named in the request.
func (ts *TournamentService) GetEnteredTournaments(ctx context.Context, req *connect.Request[pb.GetEnteredTournamentsRequest]) (*connect.Response[pb.GetEnteredTournamentsResponse], error) {
	viewer, err := apiserver.AuthUser(ctx, ts.userStore)
	if err != nil {
		return nil, err
	}
	player, err := listedUser(ctx, ts.userStore, viewer, req.Msg.Username, func() (bool, error) {
		return rbac.HasPermission(ctx, ts.queries, viewer.ID, rbac.CanSeePrivateUserData)
	})
	if err != nil {
		return nil, err
	}

	played, err := ts.queries.ListTournamentsPlayedByUser(ctx, models.ListTournamentsPlayedByUserParams{
		UserID: pgtype.Int4{Int32: int32(player.ID), Valid: true},
		Lim:    enteredTournamentsLimit,
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	registered, err := ts.queries.ListUnfinishedTournamentsWithPlayer(ctx, models.ListUnfinishedTournamentsWithPlayerParams{
		UserUuid: player.UUID,
		Lim:      enteredTournamentsLimit,
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	var entered []*pb.EnteredTournament
	byID := map[string]*pb.EnteredTournament{}
	for _, p := range played {
		e := &pb.EnteredTournament{Id: p.TournamentID, Games: p.Games}
		if p.LastPlayed.Valid {
			e.LastPlayed = timestamppb.New(p.LastPlayed.Time)
		}
		byID[e.Id] = e
		entered = append(entered, e)
	}
	for _, id := range registered {
		if !id.Valid {
			continue
		}
		if e, ok := byID[id.String]; ok {
			e.Registered = true
			continue
		}
		e := &pb.EnteredTournament{Id: id.String, Registered: true}
		byID[e.Id] = e
		entered = append(entered, e)
	}

	ids := make([]string, 0, len(entered))
	for _, e := range entered {
		ids = append(ids, e.Id)
	}
	listings, err := ts.queries.GetTournamentListings(ctx, ids)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	for _, l := range listings {
		e := byID[l.Uuid.String]
		if e == nil {
			continue
		}
		e.Name = l.Name.String
		e.Slug = l.Slug.String
		e.IrlMode = l.IrlMode
		if tt, err := competitionTType(entity.CompetitionType(l.Type.String)); err == nil {
			e.Type = tt
		}
	}
	return connect.NewResponse(&pb.GetEnteredTournamentsResponse{Tournaments: entered}), nil
}
