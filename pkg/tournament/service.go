package tournament

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/lithammer/shortuuid"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/mod"
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
}

// NewTournamentService creates a TournamentService
func NewTournamentService(ts TournamentStore, us user.Store, cfg *config.Config) *TournamentService {
	return &TournamentService{ts, us, nil, cfg}
}

func (ts *TournamentService) SetEventChannel(c chan *entity.EventWrapper) {
	ts.eventChannel = c
}

func (ts *TournamentService) AddDivision(ctx context.Context, req *connect.Request[pb.TournamentDivisionRequest],
) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
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
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
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
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
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
	err := authenticateDirector(ctx, ts, req.Msg.Metadata.Id, false, req.Msg)
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
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
	if err != nil {
		return nil, err
	}
	err = SetSingleRoundControls(ctx, ts.tournamentStore, req.Msg.Id, req.Msg.Division, int(req.Msg.Round), req.Msg.RoundControls)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) SetRoundControls(ctx context.Context, req *connect.Request[ipc.DivisionRoundControls]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
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
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
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
	_, err := directorOrAdmin(ctx, ts)
	if err != nil {
		return nil, err
	}

	if len(req.Msg.DirectorUsernames) < 1 {
		return nil, apiserver.InvalidArg("need at least one director id")
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

	tt, err := validateTournamentMeta(req.Msg.Type, req.Msg.Slug)
	if err != nil {
		return nil, err
	}
	t, err := NewTournament(ctx, ts.tournamentStore, req.Msg.Name, req.Msg.Description, directors,
		tt, "", req.Msg.Slug)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
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
		if n == 0 {
			directors = append([]string{username}, directors...)
		} else {
			directors = append(directors, username)
		}
	}

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
	}

	return connect.NewResponse(&pb.TournamentMetadataResponse{
		Metadata:  metadata,
		Directors: directors,
	}), nil

}

func (ts *TournamentService) AddDirectors(ctx context.Context, req *connect.Request[ipc.TournamentPersons]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, true, req.Msg)
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
	err := authenticateDirector(ctx, ts, req.Msg.Id, true, req.Msg)
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
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
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
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
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
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
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
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
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
		nil)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) PairRound(ctx context.Context, req *connect.Request[pb.PairRoundRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
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
	err := authenticateDirector(ctx, ts, req.Msg.Id, true, req.Msg)
	if err != nil {
		return nil, err
	}
	err = SetFinished(ctx, ts.tournamentStore, req.Msg.Id)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) StartRoundCountdown(ctx context.Context, req *connect.Request[pb.TournamentStartRoundCountdownRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
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
	err := authenticateDirector(ctx, ts, req.Msg.ClubId, false, req.Msg)
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
	// Create a tournament / club session.
	t, err := NewTournament(ctx, ts.tournamentStore, name, club.Description, club.Directors,
		entity.TypeChild, club.UUID, slug)
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

func sessionUser(ctx context.Context, ts *TournamentService) (*entity.User, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ts.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, apiserver.InternalErr(err)
	}
	return user, nil
}

func directorOrAdmin(ctx context.Context, ts *TournamentService) (*entity.User, error) {

	user, err := sessionUser(ctx, ts)
	if err != nil {
		return nil, err
	}

	if !user.IsDirector && !user.IsAdmin {
		return nil, apiserver.Unauthenticated("this user is not an authorized director")
	}
	return user, nil
}

func authenticateDirector(ctx context.Context, ts *TournamentService, id string, authenticateExecutive bool, req proto.Message) error {
	user, err := sessionUser(ctx, ts)
	if err != nil {
		return err
	}
	fullID := user.TournamentID()
	log.Info().
		Str("requester", fullID).
		Interface("req", req).
		Str("req-name", string(req.ProtoReflect().Type().Descriptor().FullName())).
		Msg("authenticated-tournament-request")

	// Site admins are always allowed to modify any tournaments. (There should only be a small number of these)
	if user.IsAdmin {
		return nil
	}
	t, err := ts.tournamentStore.Get(ctx, id)
	if err != nil {
		return apiserver.InternalErr(err)
	}

	log.Debug().Str("fullID", fullID).Interface("persons", t.Directors.Persons).Msg("authenticating-director")

	if authenticateExecutive && fullID != t.ExecutiveDirector {
		return apiserver.Unauthenticated("this user is not the authorized executive director for this event")
	}
	authorized := false
	for _, director := range t.Directors.Persons {
		if director.Id == fullID {
			authorized = true
			break
		}
	}
	if !authorized {
		return apiserver.Unauthenticated("this user is not an authorized director for this event")
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

// CheckIn does not require director permission.
func (ts *TournamentService) CheckIn(ctx context.Context, req *connect.Request[pb.CheckinRequest]) (*connect.Response[pb.TournamentResponse], error) {
	user, err := sessionUser(ctx, ts)
	if err != nil {
		return nil, err
	}

	err = CheckIn(ctx, ts.tournamentStore, req.Msg.Id, user.TournamentID())
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) UncheckIn(ctx context.Context, req *connect.Request[pb.UncheckInRequest]) (*connect.Response[pb.TournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
	if err != nil {
		return nil, err
	}
	err = UncheckIn(ctx, ts.tournamentStore, req.Msg.Id)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) UnstartTournament(ctx context.Context, req *connect.Request[pb.UnstartTournamentRequest]) (*connect.Response[pb.TournamentResponse], error) {
	// Unstarting a tournament rolls the round back to zero, and deletes all game info,
	// but does not delete the players or divisions.
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
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

	err = ts.tournamentStore.Set(ctx, t)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.TournamentResponse{}), nil
}

func (ts *TournamentService) ExportTournament(ctx context.Context, req *connect.Request[pb.ExportTournamentRequest]) (*connect.Response[pb.ExportTournamentResponse], error) {
	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
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

	err := authenticateDirector(ctx, ts, req.Msg.Id, false, req.Msg)
	if err != nil {
		return nil, err
	}
	natsconn, err := nats.Connect(ts.cfg.NatsURL)
	if err != nil {
		return nil, err
	}
	request, err := protojson.Marshal(req.Msg)
	if err != nil {
		return nil, err
	}

	resp, err := natsconn.Request(ts.cfg.TourneyPDFSubject, request, time.Second*5)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.TournamentScorecardResponse{
		PdfZip: resp.Data,
	}), nil
}
