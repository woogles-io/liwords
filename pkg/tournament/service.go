package tournament

import (
	"context"

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

func (ts *TournamentService) SetTournamentControls(ctx context.Context, req *pb.TournamentControlsRequest) (*pb.TournamentResponse, error) {
	time, err := ptypes.Timestamp(req.StartTime)
	if err != nil {
		return nil, err
	}
	err = ts.tournamentStore.SetTournamentControls(ctx,
		req.TournamentId,
		req.TournamentName,
		req.Lexicon,
		req.Variant,
		req.TimeControlName,
		req.InitialTimeSeconds,
		req.ChallengeRule,
		req.RatingMode,
		req.MaxOvertimeMinutes,
		req.IncrementSeconds,
		time)
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) AddDirectors(ctx context.Context, req *pb.TournamentPersons) (*pb.TournamentResponse, error) {
	err := ts.tournamentStore.AddDirectors(ctx, req.TournamentId, convertPersonsToStringArray(req))
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) RemoveDirectors(ctx context.Context, req *pb.TournamentPersons) (*pb.TournamentResponse, error) {
	err := ts.tournamentStore.RemoveDirectors(ctx, req.TournamentId, convertPersonsToStringArray(req))
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) AddPlayers(ctx context.Context, req *pb.TournamentPersons) (*pb.TournamentResponse, error) {
	err := ts.tournamentStore.AddPlayers(ctx, req.TournamentId, convertPersonsToStringArray(req))
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) RemovePlayers(ctx context.Context, req *pb.TournamentPersons) (*pb.TournamentResponse, error) {
	err := ts.tournamentStore.RemovePlayers(ctx, req.TournamentId, convertPersonsToStringArray(req))
	if err != nil {
		return nil, err
	}
	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) SetPairing(ctx context.Context, req *pb.TournamentPairingRequest) (*pb.TournamentResponse, error) {
	err := ts.tournamentStore.SetPairing(ctx, req.TournamentId, req.PlayerOneId, req.PlayerTwoId, int(req.Round))
	if err != nil {
		return nil, err
	}

	TournamentSetPairingsEvent(ctx, ts.tournamentStore)

	return &pb.TournamentResponse{}, nil
}
func (ts *TournamentService) SetResult(ctx context.Context, req *pb.TournamentResultRequest) (*pb.TournamentResponse, error) {
	err := ts.tournamentStore.SetResult(ctx,
		req.TournamentId,
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

	TournamentGameEndedEvent(ctx, ts.tournamentStore, req.TournamentId, int(req.Round))

	return &pb.TournamentResponse{}, nil
}

// What this does is not yet clear. Need more designs.
func (ts *TournamentService) StartRound(ctx context.Context, req *pb.TournamentStartRoundRequest) (*pb.TournamentResponse, error) {
	return &pb.TournamentResponse{}, nil
}

func convertPersonsToStringArray(req *pb.TournamentPersons) []string {
	persons := []string{}
	for _, person := range req.Persons {
		persons = append(persons, person.PersonId)
	}
	return persons
}
