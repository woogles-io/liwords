package puzzles

import (
	"context"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/puzzle_service"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"
)

type PuzzleService struct {
	puzzleStore PuzzleStore
	userStore   user.Store
}

func NewPuzzleService(ps PuzzleStore, us user.Store) *PuzzleService {
	return &PuzzleService{puzzleStore: ps, userStore: us}
}

func (ps *PuzzleService) GetRandomUnansweredPuzzleIdForUser(ctx context.Context, req *pb.RandomUnansweredPuzzleIdRequest) (*pb.RandomUnansweredPuzzleIdResponse, error) {
	user, err := sessionUser(ctx, ps)
	if err != nil {
		return nil, err
	}
	puzzleId, err := GetRandomUnansweredPuzzleIdForUser(ctx, ps.puzzleStore, user.UUID)
	if err != nil {
		return nil, err
	}
	return &pb.RandomUnansweredPuzzleIdResponse{PuzzleId: puzzleId}, nil
}

func (ps *PuzzleService) GetPuzzle(ctx context.Context, req *pb.PuzzleRequest) (*pb.PuzzleResponse, error) {
	gameUUID, gameHist, beforeText, err := GetPuzzle(ctx, ps.puzzleStore, req.PuzzleId)
	if err != nil {
		return nil, err
	}
	return &pb.PuzzleResponse{GameId: gameUUID, History: gameHist, BeforeText: beforeText}, nil
}

func (ps *PuzzleService) GetAnswer(ctx context.Context, req *pb.AnswerRequest) (*pb.AnswerResponse, error) {
	user, err := sessionUser(ctx, ps)
	if err != nil {
		return nil, err
	}
	correct, correctAnswer, afterText, err := GetAnswer(ctx, ps.puzzleStore, req.PuzzleId, user.UUID, req.Answer)
	if err != nil {
		return nil, err
	}
	return &pb.AnswerResponse{Correct: correct, Answer: correctAnswer, AfterText: afterText}, nil
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
