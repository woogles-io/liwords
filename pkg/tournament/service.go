package tournament

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/user"
	"github.com/domino14/liwords/pkg/utilities"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	pb "github.com/domino14/liwords/rpc/api/proto/tournament_service"
)

// TournamentService is a Twirp service that contains functions that
// allow directors to interact with their tournaments
type TournamentService struct {
	tournamentStore TournamentStore
	userStore       user.Store
	eventChannel    chan *entity.EventWrapper
}

// NewTournamentService creates a Twirp TournamentService
func NewTournamentService(ts TournamentStore, us user.Store) *TournamentService {
	return &TournamentService{ts, us, nil}
}

func (ts *TournamentService) SetEventChannel(c chan *entity.EventWrapper) {
	ts.eventChannel = c
}

func (ts *TournamentService) AddDivision(ctx context.Context, req *pb.TournamentDivisionRequest) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}
	err = AddDivision(ctx, ts.tournamentStore, req.Id, req.Division)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) RemoveDivision(ctx context.Context, req *pb.TournamentDivisionRequest) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}
	err = RemoveDivision(ctx, ts.tournamentStore, req.Id, req.Division)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) SetTournamentMetadata(ctx context.Context, req *pb.SetTournamentMetadataRequest) (*pb.TournamentResponse, error) {
	if req.Metadata == nil {
		return nil, twirp.NewError(twirp.InvalidArgument, "tournament metadata was empty")
	}
	err := authenticateDirector(ctx, ts, req.Metadata.Id, false)
	if err != nil {
		return nil, err
	}

	err = SetTournamentMetadata(ctx, ts.tournamentStore, req.Metadata)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) SetSingleRoundControls(ctx context.Context, req *pb.SingleRoundControlsRequest) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}
	err = SetSingleRoundControls(ctx, ts.tournamentStore, req.Id, req.Division, int(req.Round), req.RoundControls)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) SetRoundControls(ctx context.Context, req *realtime.DivisionRoundControls) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}
	err = SetRoundControls(ctx, ts.tournamentStore, req.Id, req.Division, req.RoundControls)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) SetDivisionControls(ctx context.Context, req *realtime.DivisionControls) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}

	err = SetDivisionControls(ctx, ts.tournamentStore, req.Id, req.Division, req)

	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) NewTournament(ctx context.Context, req *pb.NewTournamentRequest) (*pb.NewTournamentResponse, error) {
	_, err := directorOrAdmin(ctx, ts)
	if err != nil {
		return nil, err
	}

	if len(req.DirectorUsernames) < 1 {
		return nil, twirp.NewError(twirp.InvalidArgument, "need at least one director id")
	}
	directors := &realtime.TournamentPersons{
		Persons: []*realtime.TournamentPerson{},
	}
	for idx := range req.DirectorUsernames {
		username := req.DirectorUsernames[idx]
		u, err := ts.userStore.Get(ctx, username)
		if err != nil {
			return nil, err
		}
		directors.Persons = append(directors.Persons, &realtime.TournamentPerson{Id: u.TournamentID(), Rating: int32(idx)})
	}

	log.Debug().Interface("directors", directors).Msg("directors")

	tt, err := validateTournamentMeta(req.Type, req.Slug)
	if err != nil {
		return nil, err
	}
	t, err := NewTournament(ctx, ts.tournamentStore, req.Name, req.Description, directors,
		tt, "", req.Slug)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.NewTournamentResponse{
		Id:   t.UUID,
		Slug: t.Slug,
	}, nil
}

func (ts *TournamentService) GetTournament(ctx context.Context, req *pb.GetTournamentRequest) (*realtime.FullTournamentDivisions, error) {
	response, err := GetXHRResponse(ctx, ts.tournamentStore, req.Id)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return response, nil
}

func (ts *TournamentService) GetTournamentMetadata(ctx context.Context, req *pb.GetTournamentMetadataRequest) (*pb.TournamentMetadataResponse, error) {
	if req.Id != "" && req.Slug != "" {
		return nil, twirp.NewError(twirp.InvalidArgument, "you must provide tournament ID or slug, but not both")
	}
	if req.Id == "" && req.Slug == "" {
		return nil, twirp.NewError(twirp.InvalidArgument, "you must provide either a tournament ID, or a slug")
	}
	var t *entity.Tournament
	var err error
	if req.Id != "" {
		t, err = ts.tournamentStore.Get(ctx, req.Id)
		if err != nil {
			return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
		}
	} else if req.Slug != "" {
		t, err = ts.tournamentStore.GetBySlug(ctx, req.Slug)
		if err != nil {
			return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
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
			return nil, twirp.NewError(twirp.InvalidArgument, "bad userID: "+director.Id)
		}

		if username == "" {
			u, err := ts.userStore.GetByUUID(ctx, uuid)
			if err != nil {
				return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
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
	}

	return &pb.TournamentMetadataResponse{
		Metadata:  metadata,
		Directors: directors,
	}, nil

}

func (ts *TournamentService) AddDirectors(ctx context.Context, req *realtime.TournamentPersons) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, true)
	if err != nil {
		return nil, err
	}

	err = AddDirectors(ctx, ts.tournamentStore, ts.userStore, req.Id, req)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) RemoveDirectors(ctx context.Context, req *realtime.TournamentPersons) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, true)
	if err != nil {
		return nil, err
	}

	err = RemoveDirectors(ctx, ts.tournamentStore, ts.userStore, req.Id, req)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) AddPlayers(ctx context.Context, req *realtime.TournamentPersons) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}

	err = AddPlayers(ctx, ts.tournamentStore, ts.userStore, req.Id, req.Division, req)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) RemovePlayers(ctx context.Context, req *realtime.TournamentPersons) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}

	err = RemovePlayers(ctx, ts.tournamentStore, ts.userStore, req.Id, req.Division, req)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) SetPairing(ctx context.Context, req *pb.TournamentPairingsRequest) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}

	err = SetPairings(ctx, ts.tournamentStore, req.Id, req.Division, req.Pairings)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) SetResult(ctx context.Context, req *pb.TournamentResultOverrideRequest) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}
	err = SetResult(ctx,
		ts.tournamentStore,
		ts.userStore,
		req.Id,
		req.Division,
		req.PlayerOneId,
		req.PlayerTwoId,
		int(req.PlayerOneScore),
		int(req.PlayerTwoScore),
		req.PlayerOneResult,
		req.PlayerTwoResult,
		req.GameEndReason,
		int(req.Round),
		int(req.GameIndex),
		req.Amendment,
		nil)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) PairRound(ctx context.Context, req *pb.PairRoundRequest) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}
	if req.DeletePairings {
		err = DeletePairings(ctx, ts.tournamentStore, req.Id, req.Division, int(req.Round))
	} else {
		err = PairRound(ctx, ts.tournamentStore, req.Id, req.Division, int(req.Round), req.PreserveByes)
	}
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) RecentGames(ctx context.Context, req *pb.RecentGamesRequest) (*pb.RecentGamesResponse, error) {
	response, err := ts.tournamentStore.GetRecentGames(ctx, req.Id, int(req.NumGames), int(req.Offset))
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	// Censors the response in-place
	err = censorRecentGamesResponse(ctx, ts.userStore, response)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return response, nil
}

func (ts *TournamentService) FinishTournament(ctx context.Context, req *pb.FinishTournamentRequest) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, true)
	if err != nil {
		return nil, err
	}
	err = SetFinished(ctx, ts.tournamentStore, req.Id)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) StartRoundCountdown(ctx context.Context, req *pb.TournamentStartRoundCountdownRequest) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}

	if req.StartAllRounds {
		err = StartAllRoundCountdowns(ctx, ts.tournamentStore, req.Id, int(req.Round))
	} else {
		err = StartRoundCountdown(ctx, ts.tournamentStore, req.Id, req.Division, int(req.Round))
	}

	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) CreateClubSession(ctx context.Context, req *pb.NewClubSessionRequest) (*pb.ClubSessionResponse, error) {
	err := authenticateDirector(ctx, ts, req.ClubId, false)
	if err != nil {
		return nil, err
	}
	// Fetch the club
	club, err := ts.tournamentStore.Get(ctx, req.ClubId)
	if err != nil {
		return nil, err
	}
	if club.Type != entity.TypeClub {
		return nil, errors.New("club sessions can only be created for clubs")
	}
	// /club/madison/
	slugPrefix := club.Slug + "/"
	slug := slugPrefix + req.Date.AsTime().Format("2006-01-02-") + shortuuid.New()[2:5]

	sessionDate := req.Date.AsTime().Format("Mon Jan 2, 2006")

	name := club.Name + " - " + sessionDate
	// Create a tournament / club session.
	t, err := NewTournament(ctx, ts.tournamentStore, name, club.Description, club.Directors,
		entity.TypeChild, club.UUID, slug)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.ClubSessionResponse{
		TournamentId: t.UUID,
		Slug:         t.Slug,
	}, nil

}

func (ts *TournamentService) GetRecentClubSessions(ctx context.Context, req *pb.RecentClubSessionsRequest) (*pb.ClubSessionsResponse, error) {
	return ts.tournamentStore.GetRecentClubSessions(ctx, req.Id, int(req.Count), int(req.Offset))
}

func sessionUser(ctx context.Context, ts *TournamentService) (*entity.User, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ts.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, twirp.InternalErrorWith(err)
	}
	return user, nil
}

func directorOrAdmin(ctx context.Context, ts *TournamentService) (*entity.User, error) {

	user, err := sessionUser(ctx, ts)
	if err != nil {
		return nil, err
	}

	if !user.IsDirector && !user.IsAdmin {
		return nil, twirp.NewError(twirp.Unauthenticated, "this user is not an authorized director")
	}
	return user, nil
}

func authenticateDirector(ctx context.Context, ts *TournamentService, id string, authenticateExecutive bool) error {
	user, err := sessionUser(ctx, ts)
	if err != nil {
		return err
	}
	// Site admins are always allowed to modify any tournaments. (There should only be a small number of these)
	if user.IsAdmin {
		return nil
	}
	err = AuthorizedDirector(ctx, user, ts.tournamentStore, id, authenticateExecutive)
	if err != nil {
		return err
	}

	return nil
}

func AuthorizedDirector(ctx context.Context, u *entity.User, s TournamentStore, id string, authenticateExecutive bool) error {
	t, err := s.Get(ctx, id)
	if err != nil {
		return twirp.InternalErrorWith(err)
	}
	fullID := u.TournamentID()
	log.Debug().Str("fullID", fullID).Interface("persons", t.Directors.Persons).Msg("authenticating-director")

	if authenticateExecutive && fullID != t.ExecutiveDirector {
		return twirp.NewError(twirp.Unauthenticated, "this user is not the authorized executive director for this event")
	}
	authorized := false
	for _, director := range t.Directors.Persons {
		if director.Id == fullID {
			authorized = true
			break
		}
	}
	if !authorized {
		return twirp.NewError(twirp.Unauthenticated, "this user is not an authorized director for this event")
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
func (ts *TournamentService) CheckIn(ctx context.Context, req *pb.CheckinRequest) (*pb.TournamentResponse, error) {
	user, err := sessionUser(ctx, ts)
	if err != nil {
		return nil, err
	}

	err = CheckIn(ctx, ts.tournamentStore, req.Id, user.TournamentID())
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) UncheckIn(ctx context.Context, req *pb.UncheckInRequest) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}
	err = UncheckIn(ctx, ts.tournamentStore, req.Id)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) UnstartTournament(ctx context.Context, req *pb.UnstartTournamentRequest) (*pb.TournamentResponse, error) {
	// Unstarting a tournament rolls the round back to zero, and deletes all game info,
	// but does not delete the players or divisions.
	// Obviously this is only meant to be used for testing purposes.
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}

	t, err := ts.tournamentStore.Get(ctx, req.Id)
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

	err = ts.tournamentStore.Set(ctx, t)
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}
