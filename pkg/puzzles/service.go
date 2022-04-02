package puzzles

import (
	"context"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/puzzle_service"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PuzzleService struct {
	puzzleStore PuzzleStore
	userStore   user.Store
}

func NewPuzzleService(ps PuzzleStore, us user.Store) *PuzzleService {
	return &PuzzleService{puzzleStore: ps, userStore: us}
}

func (ps *PuzzleService) GetStartPuzzleId(ctx context.Context, req *pb.StartPuzzleIdRequest) (*pb.StartPuzzleIdResponse, error) {
	user, err := sessionUser(ctx, ps)
	if err != nil {
		return nil, err
	}
	puzzleId, err := GetStartPuzzleId(ctx, ps.puzzleStore, user.UUID, req.Lexicon)
	if err != nil {
		return nil, err
	}
	return &pb.StartPuzzleIdResponse{PuzzleId: puzzleId}, nil
}

func (ps *PuzzleService) GetNextPuzzleId(ctx context.Context, req *pb.NextPuzzleIdRequest) (*pb.NextPuzzleIdResponse, error) {
	user, err := sessionUser(ctx, ps)
	if err != nil {
		return nil, err
	}
	puzzleId, err := GetNextPuzzleId(ctx, ps.puzzleStore, user.UUID, req.Lexicon)
	if err != nil {
		return nil, err
	}
	return &pb.NextPuzzleIdResponse{PuzzleId: puzzleId}, nil
}

func (ps *PuzzleService) GetPuzzle(ctx context.Context, req *pb.PuzzleRequest) (*pb.PuzzleResponse, error) {
	// Since we want to allow people to see puzzles without
	// logging in, continue even if there is an error.
	// Assume an error means the request is unauthenticated.
	userUUID := ""
	user, err := sessionUser(ctx, ps)
	if err == nil {
		userUUID = user.UUID
	}
	gameHist, beforeText, attempts, status, firstAttemptTime, lastAttemptTime, err := GetPuzzle(ctx, ps.puzzleStore, userUUID, req.PuzzleId)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	return &pb.PuzzleResponse{History: gameHist, BeforeText: beforeText, Attempts: attempts, Status: boolPtrToPuzzleStatus(status), FirstAttemptTime: timestamppb.New(firstAttemptTime), LastAttemptTime: timestamppb.New(lastAttemptTime)}, nil
}

func (ps *PuzzleService) GetPreviousPuzzleId(ctx context.Context, req *pb.PreviousPuzzleRequest) (*pb.PreviousPuzzleResponse, error) {
	user, err := sessionUser(ctx, ps)
	if err != nil {
		return nil, err
	}
	puzzleId, err := GetPreviousPuzzleId(ctx, ps.puzzleStore, user.UUID, req.PuzzleId)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.PreviousPuzzleResponse{PuzzleId: puzzleId}, nil
}

func (ps *PuzzleService) SubmitAnswer(ctx context.Context, req *pb.SubmissionRequest) (*pb.SubmissionResponse, error) {
	user, err := sessionUser(ctx, ps)
	if err != nil {
		return nil, err
	}
	userIsCorrect, status, correctAnswer, gameId, afterText, attempts, firstAttemptTime, lastAttemptTime, err := SubmitAnswer(ctx, ps.puzzleStore, req.PuzzleId, user.UUID, req.Answer, req.ShowSolution)
	if err != nil {
		return nil, err
	}
	return &pb.SubmissionResponse{UserIsCorrect: userIsCorrect, Status: boolPtrToPuzzleStatus(status), CorrectAnswer: correctAnswer, GameId: gameId, AfterText: afterText, Attempts: attempts, FirstAttemptTime: timestamppb.New(firstAttemptTime), LastAttemptTime: timestamppb.New(lastAttemptTime)}, nil
}

func (ps *PuzzleService) SetPuzzleVote(ctx context.Context, req *pb.PuzzleVoteRequest) (*pb.PuzzleVoteResponse, error) {
	user, err := sessionUser(ctx, ps)
	if err != nil {
		return nil, err
	}
	err = SetPuzzleVote(ctx, ps.puzzleStore, user.UUID, req.PuzzleId, int(req.Vote))
	if err != nil {
		return nil, err
	}
	return &pb.PuzzleVoteResponse{}, nil
}

func sessionUser(ctx context.Context, ps *PuzzleService) (*entity.User, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ps.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, twirp.InternalErrorWith(err)
	}
	return user, nil
}

func boolPtrToPuzzleStatus(b *bool) pb.PuzzleStatus {
	status := pb.PuzzleStatus_UNANSWERED
	if b != nil {
		if *b {
			status = pb.PuzzleStatus_CORRECT
		} else {
			status = pb.PuzzleStatus_INCORRECT
		}
	}
	return status
}
