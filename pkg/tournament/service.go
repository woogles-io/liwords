package tournament

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/lithammer/shortuuid/v4"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/notify"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	"github.com/woogles-io/liwords/pkg/utilities"

	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

// TournamentService is a service that contains functions that
// allow directors to interact with their tournaments
type TournamentService struct {
	tournamentStore TournamentStore
	userStore       user.Store
	eventChannel    chan *entity.EventWrapper
	cfg             *config.Config
	lambdaClient    *lambda.Client
	queries         *models.Queries
}

// NewTournamentService creates a TournamentService
func NewTournamentService(ts TournamentStore, us user.Store, cfg *config.Config, lc *lambda.Client, q *models.Queries) *TournamentService {
	return &TournamentService{ts, us, nil, cfg, lc, q}
}

func (ts *TournamentService) SetEventChannel(c chan *entity.EventWrapper) {
	ts.eventChannel = c
}

func (ts *TournamentService) AddDivision(ctx context.Context, req *connect.Request[pb.TournamentDivisionRequest],
) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}
	err = AddDivision(ctx, ts.tournamentStore, req.Msg.Id, req.Msg.Division)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) RemoveDivision(ctx context.Context, req *connect.Request[pb.TournamentDivisionRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}
	err = RemoveDivision(ctx, ts.tournamentStore, req.Msg.Id, req.Msg.Division)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) RenameDivision(ctx context.Context, req *connect.Request[pb.DivisionRenameRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}
	err = RenameDivision(ctx, ts.tournamentStore, req.Msg.Id, req.Msg.Division, req.Msg.NewName)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) SetTournamentMetadata(ctx context.Context, req *connect.Request[pb.SetTournamentMetadataRequest]) (*connect.Response[pb.TournamentResponse], error) {
	if req.Msg.Metadata == nil {
		return nil, apiserver.InvalidArg("tournament metadata was empty")
	}
	err := authenticateDirector(ctx, ts, req.Msg.Metadata.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}

	err = SetTournamentMetadata(ctx, ts.tournamentStore, req.Msg.Metadata, req.Msg.SetOnlySpecified)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) SetSingleRoundControls(ctx context.Context, req *connect.Request[pb.SingleRoundControlsRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}
	err = SetSingleRoundControls(ctx, ts.tournamentStore, req.Msg.Id, req.Msg.Division, int(req.Msg.RoundControls.Round), req.Msg.RoundControls)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) SetRoundControls(ctx context.Context, req *connect.Request[ipc.DivisionRoundControls]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}
	err = SetRoundControls(ctx, ts.tournamentStore, req.Msg.Id, req.Msg.Division, req.Msg.RoundControls)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) SetDivisionControls(ctx context.Context, req *connect.Request[ipc.DivisionControls]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}

	err = SetDivisionControls(ctx, ts.tournamentStore, req.Msg.Id, req.Msg.Division, req.Msg)

	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) NewTournament(ctx context.Context, req *connect.Request[pb.NewTournamentRequest]) (*connect.Response[pb.NewTournamentResponse], error) {
	user, err := apiserver.AuthUser(ctx, ts.userStore)
	if err != nil {
		return nil, err
	}

	allowed, err := rbac.HasPermission(ctx, ts.queries, user.ID, rbac.CanCreateTournaments)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apiserver.PermissionDenied("not permitted to create tournaments")
	}

	if len(req.Msg.DirectorUsernames) < 1 {
		return nil, apiserver.InvalidArg("need at least one director id")
	}

	startTimeExists := req.Msg.ScheduledStartTime != nil && req.Msg.ScheduledStartTime.Seconds != 0
	endTimeExists := req.Msg.ScheduledEndTime != nil && req.Msg.ScheduledEndTime.Seconds != 0

	if req.Msg.ScheduledStartTime != nil {
		if req.Msg.ScheduledStartTime.AsTime().Before(time.Now()) {
			return nil, apiserver.InvalidArg("scheduled start time cannot be in the past")
		}
	}
	if endTimeExists {
		if !startTimeExists {
			return nil, apiserver.InvalidArg("scheduled start time is required when scheduled end time is provided")
		}
		if req.Msg.ScheduledEndTime.AsTime().Before(req.Msg.ScheduledStartTime.AsTime()) {
			return nil, apiserver.InvalidArg("scheduled end time cannot be before the start time")
		}
	}

	directors := &ipc.TournamentPersons{
		Persons: []*ipc.TournamentPerson{},
	}
	for idx, username := range req.Msg.DirectorUsernames {
		u, err := ts.userStore.Get(ctx, username)
		if err != nil {
			return nil, err
		}
		directors.Persons = append(directors.Persons, &ipc.TournamentPerson{Id: u.TournamentID(), Rating: int32(idx)})
	}

	log.Debug().Interface("directors", directors).Msg("directors")

	tt, err := validateTournamentTypeMatchesSlug(req.Msg.Type, req.Msg.Slug)
	if err != nil {
		return nil, err
	}

	var scheduledStartTime *time.Time
	var scheduledEndTime *time.Time

	if startTimeExists {
		startTime := req.Msg.ScheduledStartTime.AsTime()
		scheduledStartTime = &startTime
	}
	if endTimeExists {
		endTime := req.Msg.ScheduledEndTime.AsTime()
		scheduledEndTime = &endTime
	}

	t, err := NewTournament(ctx, ts.tournamentStore, req.Msg.Name, req.Msg.Description, directors,
		tt, "", req.Msg.Slug, scheduledStartTime, scheduledEndTime, user.ID)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	config, err := config.Ctx(ctx)
	if err != nil {
		log.Err(err).Str("userID", user.UUID).Msg("notification-nil-config")
		return nil, err
	}
	if config.DiscordToken != "" {
		notify.Post(fmt.Sprintf("A new tournament has been created by %s: %s (%s)",
			user.Username, t.Name, t.Slug), config.DiscordToken)
	}

	return connect.NewResponse(&pb.NewTournamentResponse{
		Id:   t.UUID,
		Slug: t.Slug,
	}), nil
}

func (ts *TournamentService) GetTournament(ctx context.Context, req *connect.Request[pb.GetTournamentRequest],
) (*connect.Response[ipc.FullTournamentDivisions], error) {
	response, err := GetXHRResponse(ctx, ts.tournamentStore, req.Msg.Id)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(response), nil
}

func (ts *TournamentService) GetTournamentMetadata(ctx context.Context, req *connect.Request[pb.GetTournamentMetadataRequest]) (*connect.Response[pb.TournamentMetadataResponse], error) {
	if req.Msg.Id != "" && req.Msg.Slug != "" {
		return nil, apiserver.InvalidArg("you must provide tournament ID or slug, but not both")
	}
	if req.Msg.Id == "" && req.Msg.Slug == "" {
		return nil, apiserver.InvalidArg("you must provide either a tournament ID, or a slug")
	}

	var t *entity.Tournament
	var err error

	if req.Msg.Id != "" {
		t, err = ts.tournamentStore.Get(ctx, req.Msg.Id)
		if err != nil {
			return nil, apiserver.InvalidArg(err.Error())
		}
	} else if req.Msg.Slug != "" {
		t, err = ts.tournamentStore.GetBySlug(ctx, req.Msg.Slug)
		if err != nil {
			return nil, apiserver.InvalidArg(err.Error())
		}
	}

	if t == nil {
		return nil, apiserver.InvalidArg("tournament not found")
	}
	metadata, err := dbTournamentToTournamentMetadataResponse(ctx, t)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	directors := []string{}
	for n, director := range t.Directors.Persons {
		// Legacy "persons" are stored as just their UUIDs.
		// We later on store them as uuid:username to speed up lookups.
		splitid := strings.Split(director.Id, ":")
		var uuid, username string
		if len(splitid) == 2 {
			username = splitid[1]
			uuid = splitid[0]
		} else if len(splitid) == 1 {
			uuid = splitid[0]
		} else {
			return nil, apiserver.InvalidArg("bad userID: " + director.Id)
		}

		if username == "" {
			u, err := ts.userStore.GetByUUID(ctx, uuid)
			if err != nil {
				return nil, apiserver.InvalidArg(err.Error())
			}
			username = u.Username
		}

		// HACK: Append :readonly suffix for read-only directors (Rating=1)
		// TODO: Replace with proper permissions field when backend schema is updated
		if director.Rating == 1 {
			username = username + ":readonly"
		}

		if n == 0 {
			directors = append([]string{username}, directors...)
		} else {
			directors = append(directors, username)
		}
	}

	return connect.NewResponse(&pb.TournamentMetadataResponse{
		Metadata:  metadata,
		Directors: directors,
	}), nil

}

func (ts *TournamentService) AddDirectors(ctx context.Context, req *connect.Request[ipc.TournamentPersons]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}

	err = AddDirectors(ctx, ts.tournamentStore, ts.userStore, req.Msg.Id, req.Msg)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) RemoveDirectors(ctx context.Context, req *connect.Request[ipc.TournamentPersons]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}

	err = RemoveDirectors(ctx, ts.tournamentStore, ts.userStore, req.Msg.Id, req.Msg)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) AddPlayers(ctx context.Context, req *connect.Request[ipc.TournamentPersons]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}

	err = AddPlayers(ctx, ts.tournamentStore, ts.userStore, req.Msg.Id, req.Msg.Division, req.Msg)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) RemovePlayers(ctx context.Context, req *connect.Request[ipc.TournamentPersons]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}

	err = RemovePlayers(ctx, ts.tournamentStore, ts.userStore, req.Msg.Id, req.Msg.Division, req.Msg)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) SetPairing(ctx context.Context, req *connect.Request[pb.TournamentPairingsRequest],
) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}

	err = SetPairings(ctx, ts.tournamentStore, req.Msg.Id, req.Msg.Division, req.Msg.Pairings)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) SetResult(ctx context.Context, req *connect.Request[pb.TournamentResultOverrideRequest],
) (*connect.Response[pb.TournamentResponse], error) {

	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		t, err := ts.tournamentStore.Get(ctx, req.Msg.Id)
		if err != nil {
			return nil, apiserver.InternalErr(err)
		}
		if t.ExtraMeta.IRLMode && !req.Msg.Amendment {
			// For now, IRL mode will not require authentication to set results.
			// Of course, this isn't ideal, and people can mess around and set
			// results for ongoing tournaments. We can fix this later with a
			// temporary session or similar. But I doubt people will be messing
			// around too much seeing as how I can barely convince more than one
			// or two people to ever code on this app.
			log.Info().AnErr("auth-error", err).Msg("unauthenticated-but-irlmode")
		} else {
			return nil, err
		}
	}
	err = SetResult(ctx,
		ts.tournamentStore,
		ts.userStore,
		req.Msg.Id,
		req.Msg.Division,
		req.Msg.PlayerOneId,
		req.Msg.PlayerTwoId,
		int(req.Msg.PlayerOneScore),
		int(req.Msg.PlayerTwoScore),
		req.Msg.PlayerOneResult,
		req.Msg.PlayerTwoResult,
		req.Msg.GameEndReason,
		int(req.Msg.Round),
		int(req.Msg.GameIndex),
		req.Msg.Amendment,
		nil,
		ts.queries)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) PairRound(ctx context.Context, req *connect.Request[pb.PairRoundRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}
	if req.Msg.DeletePairings {
		err = DeletePairings(ctx, ts.tournamentStore, req.Msg.Id, req.Msg.Division, int(req.Msg.Round))
	} else {
		err = PairRound(ctx, ts.tournamentStore, req.Msg.Id, req.Msg.Division, int(req.Msg.Round), req.Msg.PreserveByes)
	}
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) RecentGames(ctx context.Context, req *connect.Request[pb.RecentGamesRequest]) (*connect.Response[pb.RecentGamesResponse], error) {
	response, err := ts.tournamentStore.GetRecentGames(ctx, req.Msg.Id, int(req.Msg.NumGames), int(req.Msg.Offset))
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	// Censors the response in-place
	err = censorRecentGamesResponse(ctx, ts.userStore, response)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(response), nil
}

func (ts *TournamentService) FinishTournament(ctx context.Context, req *connect.Request[pb.FinishTournamentRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}
	err = SetFinished(ctx, ts.tournamentStore, req.Msg.Id)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) UnfinishTournament(ctx context.Context, req *connect.Request[pb.UnfinishTournamentRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}
	err = SetUnfinished(ctx, ts.tournamentStore, req.Msg.Id)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) StartRoundCountdown(ctx context.Context, req *connect.Request[pb.TournamentStartRoundCountdownRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}

	if req.Msg.StartAllRounds {
		err = StartAllRoundCountdowns(ctx, ts.tournamentStore, req.Msg.Id, int(req.Msg.Round))
	} else {
		err = StartRoundCountdown(ctx, ts.tournamentStore, req.Msg.Id, req.Msg.Division, int(req.Msg.Round))
	}

	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) CreateClubSession(ctx context.Context, req *connect.Request[pb.NewClubSessionRequest],
) (*connect.Response[pb.ClubSessionResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.ClubId, req.Msg, true)
	if err != nil {
		return nil, err
	}
	// Fetch the club
	club, err := ts.tournamentStore.Get(ctx, req.Msg.ClubId)
	if err != nil {
		return nil, err
	}
	if club.Type != entity.TypeClub {
		return nil, errors.New("club sessions can only be created for clubs")
	}
	// /club/madison/
	slugPrefix := club.Slug + "/"
	slug := slugPrefix + req.Msg.Date.AsTime().Format("2006-01-02-") + shortuuid.New()[2:5]

	sessionDate := req.Msg.Date.AsTime().Format("Mon Jan 2, 2006")

	name := club.Name + " - " + sessionDate

	// TODO: ScheduledStartTime should be required after the front-end
	// is updated to send it.
	if req.Msg.Date != nil {
		if req.Msg.Date.AsTime().Before(time.Now()) {
			return nil, apiserver.InvalidArg("Date cannot be in the past")
		}
	}

	var scheduledStartTime *time.Time
	if req.Msg.Date != nil && req.Msg.Date.Seconds != 0 {
		startTime := req.Msg.Date.AsTime()
		scheduledStartTime = &startTime
	}
	// Create a tournament / club session.
	t, err := NewTournament(ctx, ts.tournamentStore, name, club.Description, club.Directors,
		entity.TypeChild, club.UUID, slug, scheduledStartTime, nil, 0 /*fix me when we ever have club sessions*/)

	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.ClubSessionResponse{
		TournamentId: t.UUID,
		Slug:         t.Slug,
	}), nil

}

func (ts *TournamentService) GetRecentClubSessions(ctx context.Context, req *connect.Request[pb.RecentClubSessionsRequest]) (*connect.Response[pb.ClubSessionsResponse], error) {
	response, err := ts.tournamentStore.GetRecentClubSessions(ctx, req.Msg.Id, int(req.Msg.Count), int(req.Msg.Offset))
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(response), nil
}

func (ts *TournamentService) GetRecentAndUpcomingTournaments(ctx context.Context, req *connect.Request[pb.GetRecentAndUpcomingTournamentsRequest]) (*connect.Response[pb.GetRecentAndUpcomingTournamentsResponse], error) {
	tournaments, err := ts.tournamentStore.GetRecentAndUpcomingTournaments(ctx)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	response := &pb.GetRecentAndUpcomingTournamentsResponse{
		Tournaments: make([]*pb.TournamentMetadata, len(tournaments)),
	}
	for i, t := range tournaments {
		if t == nil {
			return nil, apiserver.InternalErr(errors.New("tournament is nil"))
		}
		tMeta, err := dbTournamentToTournamentMetadataResponse(ctx, t)
		if err != nil {
			return nil, apiserver.InternalErr(err)
		}
		response.Tournaments[i] = tMeta
	}
	return connect.NewResponse(response), nil
}

func (ts *TournamentService) GetPastTournaments(ctx context.Context, req *connect.Request[pb.GetPastTournamentsRequest]) (*connect.Response[pb.GetPastTournamentsResponse], error) {
	limit := req.Msg.Limit
	if limit == 0 {
		limit = 100 // default limit
	}
	tournaments, err := ts.tournamentStore.GetPastTournaments(ctx, limit)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	response := &pb.GetPastTournamentsResponse{
		Tournaments: make([]*pb.TournamentMetadata, len(tournaments)),
	}
	for i, t := range tournaments {
		if t == nil {
			return nil, apiserver.InternalErr(errors.New("tournament is nil"))
		}
		tMeta, err := dbTournamentToTournamentMetadataResponse(ctx, t)
		if err != nil {
			return nil, apiserver.InternalErr(err)
		}
		response.Tournaments[i] = tMeta
	}
	return connect.NewResponse(response), nil
}

// HACK: The requireFullDirector parameter checks director permissions using the Rating field.
// Rating field is temporarily repurposed: 0=Full Director, 1=Read-only Director
// TODO: Replace with proper permissions field when backend schema is updated
func authenticateDirector(ctx context.Context, ts *TournamentService, id string, req proto.Message, requireFullDirector bool) error {
	user, err := apiserver.AuthUser(ctx, ts.userStore)
	if err != nil {
		return err
	}
	// If the user has the manage_tournaments permission, they can manage any
	// tournaments. This permission should not be given out willy-nilly.
	allowed, err := rbac.HasPermission(ctx, ts.queries, user.ID, rbac.CanManageTournaments)
	if err != nil {
		return err
	}

	fullID := user.TournamentID()
	log.Info().
		Str("requester", fullID).
		Str("tournament-id", id).
		Interface("req", req).
		Str("req-name", string(req.ProtoReflect().Type().Descriptor().FullName())).
		Bool("rbac-allowed", allowed).
		Bool("require-full-director", requireFullDirector).
		Msg("attempting-to-authenticate-tournament-request")

	if allowed {
		return nil
	}
	// Otherwise check if the director is one of the listed tournament directors.

	t, err := ts.tournamentStore.Get(ctx, id)
	if err != nil {
		return apiserver.InternalErr(err)
	}

	log.Debug().Str("fullID", fullID).Interface("persons", t.Directors.Persons).Msg("authenticating-director")

	authorized := false
	isReadOnly := false
	for _, director := range t.Directors.Persons {
		if director.Id == fullID {
			authorized = true
			// HACK: Rating field repurposed: 0=Full Director, 1=Read-only Director
			if director.Rating == 1 {
				isReadOnly = true
			}
			break
		}
	}
	if !authorized {
		return apiserver.Unauthenticated("this user is not an authorized director for this event")
	}

	// Check if operation requires full director permissions
	if requireFullDirector && isReadOnly {
		return apiserver.PermissionDenied("this operation requires full director permissions")
	}

	return nil
}

func censorRecentGamesResponse(ctx context.Context, us user.Store, rgr *pb.RecentGamesResponse) error {
	knownUsers := make(map[string]bool)

	for _, game := range rgr.Games {
		playerOneUserEntity, err := us.Get(ctx, game.Players[0].Username)
		if err != nil {
			return err
		}
		playerTwoUserEntity, err := us.Get(ctx, game.Players[1].Username)
		if err != nil {
			return err
		}
		playerOne := playerOneUserEntity.UUID
		playerTwo := playerTwoUserEntity.UUID

		_, known := knownUsers[playerOne]
		if !known {
			knownUsers[playerOne] = mod.IsCensorable(ctx, us, playerOne)
		}
		if knownUsers[playerOne] {
			game.Players[0].Username = utilities.CensoredUsername
		}

		_, known = knownUsers[playerTwo]
		if !known {
			knownUsers[playerTwo] = mod.IsCensorable(ctx, us, playerTwo)
		}
		if knownUsers[playerTwo] {
			game.Players[1].Username = utilities.CensoredUsername
			if knownUsers[playerOne] {
				game.Players[1].Username = utilities.AnotherCensoredUsername
			}
		}
	}
	return nil
}

func (ts *TournamentService) OpenRegistration(ctx context.Context, req *connect.Request[pb.OpenRegistrationRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}
	err = OpenRegistration(ctx, ts.tournamentStore, req.Msg.Id)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) CloseRegistration(ctx context.Context, req *connect.Request[pb.CloseRegistrationRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}
	err = CloseRegistration(ctx, ts.tournamentStore, req.Msg.Id)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) Register(ctx context.Context, req *connect.Request[pb.RegisterRequest]) (*connect.Response[pb.TournamentResponse], error) {
	user, err := apiserver.AuthUser(ctx, ts.userStore)
	if err != nil {
		return nil, err
	}
	err = Register(ctx, ts.tournamentStore, req.Msg.Id, req.Msg.Division, user.TournamentID())
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

// CheckIn does not require director permission.
func (ts *TournamentService) CheckIn(ctx context.Context, req *connect.Request[pb.CheckinRequest]) (*connect.Response[pb.TournamentResponse], error) {
	user, err := apiserver.AuthUser(ctx, ts.userStore)
	if err != nil {
		return nil, err
	}
	if req.Msg.Checkin {
		err = CheckIn(ctx, ts.tournamentStore, req.Msg.Id, user.TournamentID())
	} else {
		err = UncheckIn(ctx, ts.tournamentStore, req.Msg.Id, user.TournamentID())
	}
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) UncheckAllIn(ctx context.Context, req *connect.Request[pb.UncheckAllInRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}
	err = UncheckAllIn(ctx, ts.tournamentStore, req.Msg.Id)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) OpenCheckins(ctx context.Context, req *connect.Request[pb.OpenCheckinsRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}
	err = OpenCheckins(ctx, ts.tournamentStore, req.Msg.Id)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) CloseCheckins(ctx context.Context, req *connect.Request[pb.CloseCheckinsRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}
	err = CloseCheckins(ctx, ts.tournamentStore, req.Msg.Id)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) RemoveAllPlayersNotCheckedIn(ctx context.Context, req *connect.Request[pb.RemoveAllPlayersNotCheckedInRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}
	err = RemoveAllPlayersNotCheckedIn(ctx, ts.tournamentStore, req.Msg.Id)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) UnstartTournament(ctx context.Context, req *connect.Request[pb.UnstartTournamentRequest]) (*connect.Response[pb.TournamentResponse], error) {
	// Unstarting a tournament rolls the round back to zero, and deletes all game info,
	// but does not delete the players or divisions.
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}

	t, err := ts.tournamentStore.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, err
	}

	for division := range t.Divisions {
		dm := t.Divisions[division].DivisionManager
		if dm == nil {
			return nil, fmt.Errorf("cannot reset division %s because it has a nil division manager", division)
		}
		err = dm.ResetToBeginning()
		if err != nil {
			return nil, err
		}
	}
	t.IsStarted = false
	t.IsFinished = false
	t.ExtraMeta.CheckinsOpen = false
	t.ExtraMeta.RegistrationOpen = false

	err = ts.tournamentStore.Set(ctx, t)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) ExportTournament(ctx context.Context, req *connect.Request[pb.ExportTournamentRequest]) (*connect.Response[pb.ExportTournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}

	t, err := ts.tournamentStore.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if req.Msg.Format == "" {
		return nil, apiserver.InvalidArg("must provide a format")
	}
	ret, err := ExportTournament(ctx, t, ts.userStore, req.Msg.Format)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.ExportTournamentResponse{Exported: ret}), nil
}

func (ts *TournamentService) GetTournamentScorecards(ctx context.Context, req *connect.Request[pb.TournamentScorecardRequest],
) (*connect.Response[pb.TournamentScorecardResponse], error) {

	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}

	request, err := protojson.Marshal(req.Msg)
	if err != nil {
		return nil, err
	}

	out, err := ts.lambdaClient.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(ts.cfg.TourneyPDFLambdaFunctionName),
		InvocationType: types.InvocationTypeRequestResponse,
		Payload:        request,
	})
	if err != nil {
		return nil, err
	}

	type lambdaOutput struct {
		StatusCode int    `json:"statusCode"`
		Body       string `json:"body"`
		Payload    string `json:"payload"`
	}

	lo := &lambdaOutput{}
	err = json.Unmarshal(out.Payload, lo)
	if err != nil {
		return nil, err
	}
	if lo.StatusCode != 200 {
		return nil, apiserver.InternalErr(errors.New(lo.Body))
	}
	bts, err := base64.StdEncoding.DecodeString(lo.Payload)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.TournamentScorecardResponse{
		PdfZip: bts,
	}), nil
}

func (ts *TournamentService) RunCOP(ctx context.Context, req *connect.Request[pb.RunCopRequest],
) (*connect.Response[ipc.PairResponse], error) {

	err := authenticateDirector(ctx, ts, req.Msg.Id, req.Msg, true)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&ipc.PairResponse{}), nil
}

func (ts *TournamentService) InitializeMonitoringKeys(ctx context.Context, req *connect.Request[pb.InitializeMonitoringKeysRequest],
) (*connect.Response[pb.TournamentResponse], error) {

	user, err := apiserver.AuthUser(ctx, ts.userStore)
	if err != nil {
		return nil, err
	}

	// Check if user is registered in this tournament
	t, err := ts.tournamentStore.Get(ctx, req.Msg.TournamentId)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	userIsParticipant := false
	userID := user.TournamentID()

	for _, division := range t.Divisions {
		if division.DivisionManager != nil {
			players := division.DivisionManager.GetPlayers()
			for _, p := range players.Persons {
				if p.Id == userID {
					userIsParticipant = true
					break
				}
			}
			if userIsParticipant {
				break
			}
		}
	}

	if !userIsParticipant {
		return nil, apiserver.PermissionDenied("you are not registered in this tournament")
	}

	err = InitializeMonitoringKeys(ctx, ts.tournamentStore, req.Msg.TournamentId, user.TournamentID())
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) RequestMonitoringStream(ctx context.Context, req *connect.Request[pb.RequestMonitoringStreamRequest],
) (*connect.Response[pb.TournamentResponse], error) {

	user, err := apiserver.AuthUser(ctx, ts.userStore)
	if err != nil {
		return nil, err
	}

	// Check if user is registered in this tournament
	t, err := ts.tournamentStore.Get(ctx, req.Msg.TournamentId)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	userIsParticipant := false
	userID := user.TournamentID()

	for _, division := range t.Divisions {
		if division.DivisionManager != nil {
			players := division.DivisionManager.GetPlayers()
			for _, p := range players.Persons {
				if p.Id == userID {
					userIsParticipant = true
					break
				}
			}
			if userIsParticipant {
				break
			}
		}
	}

	if !userIsParticipant {
		return nil, apiserver.PermissionDenied("you are not registered in this tournament")
	}

	err = RequestMonitoringStream(ctx, ts.tournamentStore, req.Msg.TournamentId, user.TournamentID(), req.Msg.StreamType)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) ResetMonitoringStream(ctx context.Context, req *connect.Request[pb.ResetMonitoringStreamRequest],
) (*connect.Response[pb.TournamentResponse], error) {

	// Only directors can reset streams (read-only directors are allowed)
	err := authenticateDirector(ctx, ts, req.Msg.TournamentId, req.Msg, false)
	if err != nil {
		return nil, err
	}

	err = ResetMonitoringStream(ctx, ts.tournamentStore, req.Msg.TournamentId, req.Msg.UserId, req.Msg.StreamType)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) GetTournamentMonitoring(ctx context.Context, req *connect.Request[pb.GetTournamentMonitoringRequest],
) (*connect.Response[pb.GetTournamentMonitoringResponse], error) {

	user, err := apiserver.AuthUser(ctx, ts.userStore)
	if err != nil {
		return nil, err
	}

	// Check if user is a director (no error means they are - including read-only)
	err = authenticateDirector(ctx, ts, req.Msg.TournamentId, req.Msg, false)
	isDirector := (err == nil)

	participants, err := GetTournamentMonitoring(ctx, ts.tournamentStore, req.Msg.TournamentId)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	// If not a director, filter to only the current user's data
	if !isDirector {
		filteredParticipants := []*ipc.MonitoringData{}
		userID := user.UUID // Use UUID only, not TournamentID() which includes username
		for _, p := range participants {
			if p.UserId == userID {
				filteredParticipants = append(filteredParticipants, p)
				break
			}
		}
		participants = filteredParticipants
	}

	return connect.NewResponse(&pb.GetTournamentMonitoringResponse{
		Participants: participants,
	}), nil
}

func dbTournamentToTournamentMetadataResponse(ctx context.Context, t *entity.Tournament) (*pb.TournamentMetadata, error) {
	var tt pb.TType
	switch t.Type {
	case entity.TypeStandard:
		tt = pb.TType_STANDARD
	case entity.TypeChild:
		tt = pb.TType_CHILD
	case entity.TypeClub:
		tt = pb.TType_CLUB
	case entity.TypeLegacy:
		tt = pb.TType_LEGACY
	default:
		return nil, fmt.Errorf("unrecognized tournament type: %v", t.Type)
	}

	var scheduledStartTime *timestamppb.Timestamp
	if t.ScheduledStartTime != nil {
		scheduledStartTime = timestamppb.New(*t.ScheduledStartTime)
	}
	var scheduledEndTime *timestamppb.Timestamp
	if t.ScheduledEndTime != nil {
		scheduledEndTime = timestamppb.New(*t.ScheduledEndTime)
	}

	// Get first director username
	var firstDirector string
	if t.Directors != nil && len(t.Directors.Persons) > 0 {
		// Director ID format is "uuid:username", extract username
		firstDirectorID := t.Directors.Persons[0].Id
		parts := strings.Split(firstDirectorID, ":")
		if len(parts) == 2 {
			firstDirector = parts[1]
		} else {
			firstDirector = firstDirectorID
		}
	}

	// Calculate total registrant count across all divisions
	var registrantCount int32
	for _, division := range t.Divisions {
		if division.DivisionManager != nil {
			players := division.DivisionManager.GetPlayers()
			if players != nil {
				registrantCount += int32(len(players.Persons))
			}
		}
	}

	metadata := &pb.TournamentMetadata{
		Id:                        t.UUID,
		Name:                      t.Name,
		Description:               t.Description,
		Slug:                      t.Slug,
		Type:                      tt,
		Disclaimer:                t.ExtraMeta.Disclaimer,
		TileStyle:                 t.ExtraMeta.TileStyle,
		BoardStyle:                t.ExtraMeta.BoardStyle,
		DefaultClubSettings:       t.ExtraMeta.DefaultClubSettings,
		FreeformClubSettingFields: t.ExtraMeta.FreeformClubSettingFields,
		Password:                  t.ExtraMeta.Password,
		Logo:                      t.ExtraMeta.Logo,
		Color:                     t.ExtraMeta.Color,
		PrivateAnalysis:           t.ExtraMeta.PrivateAnalysis,
		IrlMode:                   t.ExtraMeta.IRLMode,
		Monitored:                 t.ExtraMeta.Monitored,
		ScheduledStartTime:        scheduledStartTime,
		ScheduledEndTime:          scheduledEndTime,
		CheckinsOpen:              t.ExtraMeta.CheckinsOpen,
		RegistrationOpen:          t.ExtraMeta.RegistrationOpen,
		FirstDirector:             firstDirector,
		RegistrantCount:           registrantCount,
	}

	return metadata, nil
}
