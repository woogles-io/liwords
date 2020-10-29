package tournament

import (
	"context"

	"github.com/domino14/liwords/pkg/entity"
	pb "github.com/domino14/liwords/rpc/api/proto/tournament_service"

	"github.com/golang/protobuf/ptypes"
)

// TournamentService is a Twirp service that contains functions that
// allow directors to interact with their tournaments
type TournamentService struct {
	tournamentStore TournamentStore
}

// NewTournamentService creates a Twirp TournamentService
func NewTournamentService(ts TournamentStore) *TournamentService {
	return &TournamentService{ts}
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

func (ts *TournamentService) SetTournamentControls(ctx context.Context, req *pb.TournamentControlsRequest) (*pb.TournamentResponse, error) {
	time, err := ptypes.Timestamp(req.StartTime)
	if err != nil {
		return nil, err
	}

	newControls := &entity.TournamentControls{GameRequest: req.GameRequest,
		PairingMethods: convertIntsToPairingMethods(req.PairingMethods),
		FirstMethods:   convertIntsToFirstMethods(req.FirstMethods),
		NumberOfRounds: int(req.NumberOfRounds),
		GamesPerRound:  int(req.GamesPerRound),
		StartTime:      time}

	err = SetTournamentControls(ctx, ts.tournamentStore, req.Id, req.Division, newControls)

	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
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
		req.Amendment)
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
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
