package puzzles

import (
	"context"
	"errors"
	"fmt"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/puzzle_service"
	"github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	errNotAuthorized = errors.New("this user is not authorized to perform this action")
)

type PuzzleService struct {
	puzzleStore        PuzzleStore
	userStore          user.Store
	puzzleGenSecretKey string
}

func NewPuzzleService(ps PuzzleStore, us user.Store, k string) *PuzzleService {
	return &PuzzleService{puzzleStore: ps, userStore: us, puzzleGenSecretKey: k}
}

func (ps *PuzzleService) GetStartPuzzleId(ctx context.Context, req *pb.StartPuzzleIdRequest) (*pb.StartPuzzleIdResponse, error) {
	puzzleId, err := GetStartPuzzleId(ctx, ps.puzzleStore, sessionUserUUIDOption(ctx, ps), req.Lexicon)
	if err != nil {
		return nil, err
	}
	return &pb.StartPuzzleIdResponse{PuzzleId: puzzleId}, nil
}

func (ps *PuzzleService) GetNextPuzzleId(ctx context.Context, req *pb.NextPuzzleIdRequest) (*pb.NextPuzzleIdResponse, error) {
	puzzleId, err := GetNextPuzzleId(ctx, ps.puzzleStore, sessionUserUUIDOption(ctx, ps), req.Lexicon)
	if err != nil {
		return nil, err
	}
	return &pb.NextPuzzleIdResponse{PuzzleId: puzzleId}, nil
}

func (ps *PuzzleService) GetPuzzle(ctx context.Context, req *pb.PuzzleRequest) (*pb.PuzzleResponse, error) {
	gameHist, beforeText, attempts, status, firstAttemptTime, lastAttemptTime, err := GetPuzzle(ctx, ps.puzzleStore, sessionUserUUIDOption(ctx, ps), req.PuzzleId)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	var correctAnswer *macondo.GameEvent
	var gameId string
	var turnNumber int32
	var afterText string

	if status != nil {
		correctAnswer, gameId, turnNumber, afterText, _, _, err = GetAnswer(ctx, ps.puzzleStore, req.PuzzleId)
		if err != nil {
			return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
		}
	}

	return &pb.PuzzleResponse{History: gameHist, BeforeText: beforeText, Answer: &pb.AnswerResponse{
		Status:           boolPtrToPuzzleStatus(status),
		CorrectAnswer:    correctAnswer,
		GameId:           gameId,
		TurnNumber:       turnNumber,
		AfterText:        afterText,
		Attempts:         attempts,
		NewUserRating:    0,
		NewPuzzleRating:  0,
		FirstAttemptTime: timestamppb.New(firstAttemptTime),
		LastAttemptTime:  timestamppb.New(lastAttemptTime)}}, nil
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
	userIsCorrect, status, correctAnswer, gameId, turnNumber, afterText, attempts, newUserRating, newPuzzleRating, firstAttemptTime, lastAttemptTime, err := SubmitAnswer(ctx, ps.puzzleStore, req.PuzzleId, user.UUID, req.Answer, req.ShowSolution)
	if err != nil {
		return nil, err
	}
	return &pb.SubmissionResponse{UserIsCorrect: userIsCorrect,
		Answer: &pb.AnswerResponse{
			Status:           boolPtrToPuzzleStatus(status),
			CorrectAnswer:    correctAnswer,
			GameId:           gameId,
			TurnNumber:       turnNumber,
			AfterText:        afterText,
			Attempts:         attempts,
			NewUserRating:    newUserRating,
			NewPuzzleRating:  newPuzzleRating,
			FirstAttemptTime: timestamppb.New(firstAttemptTime),
			LastAttemptTime:  timestamppb.New(lastAttemptTime)}}, nil
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

func (ps *PuzzleService) StartPuzzleGenJob(ctx context.Context, req *pb.APIPuzzleGenerationJobRequest) (*pb.APIPuzzleGenerationJobResponse, error) {
	user, err := sessionUser(ctx, ps)
	if err != nil {
		return nil, err
	}
	if !user.IsAdmin {
		return nil, twirp.NewError(twirp.Unauthenticated, errNotAuthorized.Error())
	}
	log.Debug().Msgf("keys %s %s", req.SecretKey, ps.puzzleGenSecretKey)
	if req.SecretKey != ps.puzzleGenSecretKey {
		return nil, twirp.NewError(twirp.PermissionDenied, "must include puzzle generation secret key")
	}
	// for logs
	req.SecretKey = ""
	bts, err := protojson.Marshal(req.Request)
	if err != nil {
		return nil, err
	}
	fmt.Printf("docker-compose run --rm -w /opt/program/cmd/puzzlegen app go run . '%s'\n", string(bts))

	return &pb.APIPuzzleGenerationJobResponse{}, nil
}

// Returns the UUID of the user if they are logged in
// or an empty string if the user is not logged in
func sessionUserUUIDOption(ctx context.Context, ps *PuzzleService) string {
	userUUID := ""
	user, err := sessionUser(ctx, ps)
	if err == nil {
		userUUID = user.UUID
	}
	return userUUID
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
