package tournament

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes"
	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"

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
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}
	err = SetTournamentMetadata(ctx, ts.tournamentStore, req.Id, req.Name, req.Description)
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
	newControls := convertSingleRoundControls(req.RoundControls)

	err = SetSingleRoundControls(ctx, ts.tournamentStore, req.Id, req.Division, int(req.Round), newControls)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) SetTournamentControls(ctx context.Context, req *pb.TournamentControlsRequest) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}
	time, err := ptypes.Timestamp(req.StartTime)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	newControls := &entity.TournamentControls{GameRequest: req.GameRequest,
		RoundControls: convertRoundControls(req.RoundControls),
		AutoStart:     req.AutoStart,
		StartTime:     time}

	err = SetTournamentControls(ctx, ts.tournamentStore, req.Id, req.Division, newControls)

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
	persons := map[string]int{}
	for idx := range req.DirectorUsernames {
		username := req.DirectorUsernames[idx]
		u, err := ts.userStore.Get(ctx, username)
		if err != nil {
			return nil, err
		}
		personID := u.UUID + ":" + u.Username
		// 1st director is the executive director.
		persons[personID] = idx
	}

	directors := &entity.TournamentPersons{
		Persons: persons,
	}

	log.Debug().Interface("directors", directors).Msg("directors")

	var tt entity.CompetitionType
	switch req.Type {
	case pb.TType_CLUB:
		tt = entity.TypeClub
		if !strings.HasPrefix(req.Slug, "/club/") {
			return nil, twirp.NewError(twirp.InvalidArgument, "club slug must start with /club/")
		}
	case pb.TType_STANDARD:
		tt = entity.TypeStandard
		if !strings.HasPrefix(req.Slug, "/tournament/") {
			return nil, twirp.NewError(twirp.InvalidArgument, "tournament slug must start with /tournament/")
		}
	case pb.TType_CLUB_SESSION:
		tt = entity.TypeClubSession
		if !strings.HasPrefix(req.Slug, "/club/") {
			return nil, twirp.NewError(twirp.InvalidArgument, "club-session slug must start with /club/")
		}
	default:
		return nil, twirp.NewError(twirp.InvalidArgument, "invalid tournament type")
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

	for uid, n := range t.Directors.Persons {
		// Legacy "persons" are stored as just their UUIDs.
		// We later on store them as uuid:username to speed up lookups.
		splitid := strings.Split(uid, ":")
		var uuid, username string
		if len(splitid) == 2 {
			username = splitid[1]
			uuid = splitid[0]
		} else if len(splitid) == 1 {
			uuid = splitid[0]
		} else {
			return nil, twirp.NewError(twirp.InvalidArgument, "bad userID: "+uid)
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
	case entity.TypeClubSession:
		tt = pb.TType_CLUB_SESSION
	case entity.TypeClub:
		tt = pb.TType_CLUB
	default:
		return nil, fmt.Errorf("unrecognized tournament type: %v", t.Type)
	}
	divNames := make([]string, len(t.Divisions))
	idx := 0
	for d := range t.Divisions {
		divNames[idx] = d
		idx++
	}

	return &pb.TournamentMetadataResponse{
		Name:        t.Name,
		Description: t.Description,
		Directors:   directors,
		Slug:        t.Slug,
		Id:          t.UUID,
		Type:        tt,
		Divisions:   divNames,
	}, nil

}

func (ts *TournamentService) AddDirectors(ctx context.Context, req *pb.TournamentPersons) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, true)
	if err != nil {
		return nil, err
	}
	m, err := convertPersonsToStringMap(ctx, req, ts.userStore)
	if err != nil {
		return nil, err
	}

	err = AddDirectors(ctx, ts.tournamentStore, req.Id, m)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) RemoveDirectors(ctx context.Context, req *pb.TournamentPersons) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, true)
	if err != nil {
		return nil, err
	}
	m, err := convertPersonsToStringMap(ctx, req, ts.userStore)
	if err != nil {
		return nil, err
	}

	err = RemoveDirectors(ctx, ts.tournamentStore, req.Id, m)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) AddPlayers(ctx context.Context, req *pb.TournamentPersons) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}
	m, err := convertPersonsToStringMap(ctx, req, ts.userStore)
	if err != nil {
		return nil, err
	}

	err = AddPlayers(ctx, ts.tournamentStore, req.Id, req.Division, m)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) RemovePlayers(ctx context.Context, req *pb.TournamentPersons) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, false)
	if err != nil {
		return nil, err
	}
	m, err := convertPersonsToStringMap(ctx, req, ts.userStore)
	if err != nil {
		return nil, err
	}

	err = RemovePlayers(ctx, ts.tournamentStore, req.Id, req.Division, m)
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
	err = PairRound(ctx, ts.tournamentStore, req.Id, req.Division, int(req.Round))
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
	return response, nil
}

func (ts *TournamentService) StartTournament(ctx context.Context, req *pb.StartTournamentRequest) (*pb.TournamentResponse, error) {
	err := authenticateDirector(ctx, ts, req.Id, true)
	if err != nil {
		return nil, err
	}
	err = StartTournament(ctx, ts.tournamentStore, req.Id, true)
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
	err = StartRoundCountdown(ctx, ts.tournamentStore, req.Id, req.Division, int(req.Round), true, true)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.TournamentResponse{}, nil
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
	user, err := directorOrAdmin(ctx, ts)
	if err != nil {
		return err
	}
	// Site admins are always allowed to modify any tournaments. (There should only be a small number of these)
	if user.IsAdmin {
		return nil
	}

	t, err := ts.tournamentStore.Get(ctx, id)
	if err != nil {
		return twirp.InternalErrorWith(err)
	}
	fullID := user.UUID + ":" + user.Username

	if authenticateExecutive && fullID != t.ExecutiveDirector {
		return twirp.NewError(twirp.Unauthenticated, "this user is not the authorized executive director for this event")
	}
	_, authorized := t.Directors.Persons[fullID]
	if !authorized {
		return twirp.NewError(twirp.Unauthenticated, "this user is not an authorized director for this event")
	}

	return nil
}

func convertPersonsToStringMap(ctx context.Context, req *pb.TournamentPersons, us user.Store) (*entity.TournamentPersons, error) {
	personsMap := map[string]int{}
	for _, person := range req.Persons {

		u, err := us.Get(ctx, person.PersonId)
		if err != nil {
			return nil, err
		}

		personsMap[u.UUID+":"+person.PersonId] = int(person.PersonInt)
	}
	return &entity.TournamentPersons{Persons: personsMap}, nil
}

func convertSingleRoundControls(reqRC *realtime.RoundControl) *entity.RoundControls {
	return &entity.RoundControls{FirstMethod: reqRC.FirstMethod,
		PairingMethod:               reqRC.PairingMethod,
		GamesPerRound:               int(reqRC.GamesPerRound),
		Round:                       int(reqRC.Round),
		Factor:                      int(reqRC.Factor),
		InitialFontes:               int(reqRC.InitialFontes),
		MaxRepeats:                  int(reqRC.MaxRepeats),
		AllowOverMaxRepeats:         reqRC.AllowOverMaxRepeats,
		RepeatRelativeWeight:        int(reqRC.RepeatRelativeWeight),
		WinDifferenceRelativeWeight: int(reqRC.WinDifferenceRelativeWeight)}
}

func convertRoundControls(reqRoundControls []*realtime.RoundControl) []*entity.RoundControls {
	rcs := []*entity.RoundControls{}
	for i := 0; i < len(reqRoundControls); i++ {
		rcs = append(rcs, convertSingleRoundControls(reqRoundControls[i]))
	}
	return rcs
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
		entity.TypeClubSession, club.UUID, slug)
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
