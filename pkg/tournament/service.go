package tournament

import (
	"context"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/tournament_service"

	"github.com/golang/protobuf/ptypes"
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
	err := AddDivision(ctx, ts.tournamentStore, req.Id, req.Division)
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) RemoveDivision(ctx context.Context, req *pb.TournamentDivisionRequest) (*pb.TournamentResponse, error) {
	err := RemoveDivision(ctx, ts.tournamentStore, req.Id, req.Division)
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) SetTournamentMetadata(ctx context.Context, req *pb.TournamentMetadataRequest) (*pb.TournamentResponse, error) {
	err := SetTournamentMetadata(ctx, ts.tournamentStore, req.Id, req.Name, req.Description)
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) SetTournamentControls(ctx context.Context, req *pb.TournamentControlsRequest) (*pb.TournamentResponse, error) {
	time, err := ptypes.Timestamp(req.StartTime)
	if err != nil {
		return nil, err
	}

	newControls := &entity.TournamentControls{GameRequest: req.GameRequest,
		PairingMethods: convertIntsToPairingMethods(req.PairingMethods),
		FirstMethods:   convertIntsToFirstMethods(req.FirstMethods),
		NumberOfRounds: int(req.NumberOfRounds),
		GamesPerRound:  convertIntsToGamesPerRound(req.GamesPerRound),
		StartTime:      time}

	err = SetTournamentControls(ctx, ts.tournamentStore, req.Id, req.Division, newControls)

	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) NewTournament(ctx context.Context, req *pb.NewTournamentRequest) (*pb.NewTournamentResponse, error) {

	directors := &entity.TournamentPersons{
		Persons: map[string]int{req.DirectorId: 0},
	}

	t, err := NewTournament(ctx, ts.tournamentStore, req.Slug, req.Name, req.Description, directors)
	if err != nil {
		return nil, err
	}
	return &pb.NewTournamentResponse{
		Id: t.UUID,
	}, nil
}

func (ts *TournamentService) GetTournamentMetadata(ctx context.Context, req *pb.GetTournamentMetadataRequest) (*pb.TournamentMetadataResponse, error) {
	t, err := ts.tournamentStore.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	execdir, err := ts.userStore.GetByUUID(ctx, t.ExecutiveDirector)
	if err != nil {
		return nil, err
	}

	return &pb.TournamentMetadataResponse{
		Name:             t.Name,
		Description:      t.Description,
		DirectorUsername: execdir.Username,
	}, nil

}

func (ts *TournamentService) AddDirectors(ctx context.Context, req *pb.TournamentPersons) (*pb.TournamentResponse, error) {
	err := AddDirectors(ctx, ts.tournamentStore, req.Id, convertPersonsToStringMap(req))
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) RemoveDirectors(ctx context.Context, req *pb.TournamentPersons) (*pb.TournamentResponse, error) {
	err := RemoveDirectors(ctx, ts.tournamentStore, req.Id, convertPersonsToStringMap(req))
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) AddPlayers(ctx context.Context, req *pb.TournamentPersons) (*pb.TournamentResponse, error) {
	err := AddPlayers(ctx, ts.tournamentStore, req.Id, req.Division, convertPersonsToStringMap(req))
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) RemovePlayers(ctx context.Context, req *pb.TournamentPersons) (*pb.TournamentResponse, error) {
	err := RemovePlayers(ctx, ts.tournamentStore, req.Id, req.Division, convertPersonsToStringMap(req))
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) SetPairing(ctx context.Context, req *pb.TournamentPairingRequest) (*pb.TournamentResponse, error) {
	err := SetPairing(ctx, ts.tournamentStore, req.Id, req.Division, req.PlayerOneId, req.PlayerTwoId, int(req.Round))
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) SetResult(ctx context.Context, req *pb.TournamentResultOverrideRequest) (*pb.TournamentResponse, error) {
	err := SetResult(ctx,
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
		nil,
		ts.eventChannel)
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}

func (ts *TournamentService) RecentGames(ctx context.Context, req *pb.RecentGamesRequest) (*pb.RecentGamesResponse, error) {
	return ts.tournamentStore.GetRecentGames(ctx, req.Id, int(req.NumGames), int(req.Offset))
}

// What this does is not yet clear. Need more designs.
func (ts *TournamentService) StartRound(ctx context.Context, req *pb.TournamentStartRoundRequest) (*pb.TournamentResponse, error) {
	err := StartRound(ctx, ts.tournamentStore, req.Id, req.Division, int(req.Round))
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}

func convertPersonsToStringMap(req *pb.TournamentPersons) *entity.TournamentPersons {
	personsMap := map[string]int{}
	for _, person := range req.Persons {
		personsMap[person.PersonId] = int(person.PersonInt)
	}
	return &entity.TournamentPersons{Persons: personsMap}
}

func convertIntsToPairingMethods(methods []int32) []entity.PairingMethod {
	pairingMethods := []entity.PairingMethod{}
	for i := 0; i < len(methods)-1; i++ {
		pairingMethods = append(pairingMethods, entity.PairingMethod(methods[i]))
	}
	return pairingMethods
}

func convertIntsToFirstMethods(methods []int32) []entity.FirstMethod {
	firstMethods := []entity.FirstMethod{}
	for i := 0; i < len(methods)-1; i++ {
		firstMethods = append(firstMethods, entity.FirstMethod(methods[i]))
	}
	return firstMethods
}

func convertIntsToGamesPerRound(gpr []int32) []int {
	ints := []int{}
	for i := 0; i < len(gpr)-1; i++ {
		ints = append(ints, int(gpr[i]))
	}
	return ints
}
