package puzzles

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/smithy-go"
	"github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/puzzle_service"
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
	ecsCluster         string
	puzzleGenTaskDef   string
}

func NewPuzzleService(ps PuzzleStore, us user.Store, k, c, td string) *PuzzleService {
	return &PuzzleService{
		puzzleStore:        ps,
		userStore:          us,
		puzzleGenSecretKey: k,
		ecsCluster:         c,
		puzzleGenTaskDef:   td,
	}
}

func (ps *PuzzleService) GetStartPuzzleId(ctx context.Context, req *connect.Request[pb.StartPuzzleIdRequest]) (*connect.Response[pb.StartPuzzleIdResponse], error) {
	puzzleId, pqr, err := GetStartPuzzleId(ctx, ps.puzzleStore, sessionUserUUIDOption(ctx, ps), req.Msg.Lexicon)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.StartPuzzleIdResponse{PuzzleId: puzzleId, QueryResult: pqr}), nil
}

func (ps *PuzzleService) GetNextPuzzleId(ctx context.Context, req *connect.Request[pb.NextPuzzleIdRequest]) (*connect.Response[pb.NextPuzzleIdResponse], error) {
	puzzleId, pqr, err := GetNextPuzzleId(ctx, ps.puzzleStore, sessionUserUUIDOption(ctx, ps), req.Msg.Lexicon)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.NextPuzzleIdResponse{PuzzleId: puzzleId, QueryResult: pqr}), nil
}

func (ps *PuzzleService) GetNextClosestRatingPuzzleId(ctx context.Context, req *connect.Request[pb.NextClosestRatingPuzzleIdRequest]) (*connect.Response[pb.NextClosestRatingPuzzleIdResponse], error) {
	puzzleId, pqr, err := GetNextClosestRatingPuzzleId(ctx, ps.puzzleStore, sessionUserUUIDOption(ctx, ps), req.Msg.Lexicon)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.NextClosestRatingPuzzleIdResponse{PuzzleId: puzzleId, QueryResult: pqr}), nil
}

func (ps *PuzzleService) GetPuzzle(ctx context.Context, req *connect.Request[pb.PuzzleRequest]) (*connect.Response[pb.PuzzleResponse], error) {
	gameHist, beforeText, attempts, status, firstAttemptTime, lastAttemptTime, newPuzzleRating, newUserRating, err := GetPuzzle(ctx, ps.puzzleStore, sessionUserUUIDOption(ctx, ps), req.Msg.PuzzleId)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	var correctAnswer *macondo.GameEvent
	var gameId string
	var turnNumber int32
	var afterText string

	if status != nil {
		correctAnswer, gameId, turnNumber, afterText, _, _, err = GetAnswer(ctx, ps.puzzleStore, req.Msg.PuzzleId)
		if err != nil {
			return nil, apiserver.InvalidArg(err.Error())
		}
	}

	npr := int32(0)
	nur := int32(0)

	if newPuzzleRating != nil {
		npr = int32(newPuzzleRating.Rating + 0.5)
	}
	if newUserRating != nil {
		nur = int32(newUserRating.Rating + 0.5)
	}

	return connect.NewResponse(&pb.PuzzleResponse{
		History:    gameHist,
		BeforeText: beforeText,
		Answer: &pb.AnswerResponse{
			Status:           boolPtrToPuzzleStatus(status),
			CorrectAnswer:    correctAnswer,
			GameId:           gameId,
			TurnNumber:       turnNumber,
			AfterText:        afterText,
			Attempts:         attempts,
			NewUserRating:    nur,
			NewPuzzleRating:  npr,
			FirstAttemptTime: timestamppb.New(firstAttemptTime),
			LastAttemptTime:  timestamppb.New(lastAttemptTime),
		},
	}), nil
}

func (ps *PuzzleService) GetPreviousPuzzleId(ctx context.Context, req *connect.Request[pb.PreviousPuzzleRequest]) (*connect.Response[pb.PreviousPuzzleResponse], error) {
	user, err := sessionUser(ctx, ps)
	if err != nil {
		return nil, err
	}
	puzzleId, err := GetPreviousPuzzleId(ctx, ps.puzzleStore, user.UUID, req.Msg.PuzzleId)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.PreviousPuzzleResponse{PuzzleId: puzzleId}), nil
}

func (ps *PuzzleService) SubmitAnswer(ctx context.Context, req *connect.Request[pb.SubmissionRequest]) (*connect.Response[pb.SubmissionResponse], error) {
	user, err := sessionUser(ctx, ps)
	if err != nil {
		return nil, err
	}
	userIsCorrect, status, correctAnswer, gameId, turnNumber, afterText, attempts, firstAttemptTime, lastAttemptTime, newPuzzleRating, newUserRating, err := SubmitAnswer(ctx, ps.puzzleStore, user.UUID, req.Msg.PuzzleId, req.Msg.Answer, req.Msg.ShowSolution)
	if err != nil {
		return nil, err
	}

	npr := int32(0)
	nur := int32(0)

	if newPuzzleRating != nil {
		npr = int32(newPuzzleRating.Rating + 0.5)
	}
	if newUserRating != nil {
		nur = int32(newUserRating.Rating + 0.5)
	}

	return connect.NewResponse(&pb.SubmissionResponse{
		UserIsCorrect: userIsCorrect,
		Answer: &pb.AnswerResponse{
			Status:           boolPtrToPuzzleStatus(status),
			CorrectAnswer:    correctAnswer,
			GameId:           gameId,
			TurnNumber:       turnNumber,
			AfterText:        afterText,
			Attempts:         attempts,
			NewUserRating:    nur,
			NewPuzzleRating:  npr,
			FirstAttemptTime: timestamppb.New(firstAttemptTime),
			LastAttemptTime:  timestamppb.New(lastAttemptTime),
		},
	}), nil
}

func (ps *PuzzleService) GetPuzzleAnswer(ctx context.Context, req *connect.Request[pb.PuzzleRequest]) (*connect.Response[pb.AnswerResponse], error) {
	user, err := sessionUser(ctx, ps)
	if err != nil {
		return nil, err
	}
	answer, err := GetPuzzleAnswer(ctx, ps.puzzleStore, user.UUID, req.Msg.PuzzleId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.AnswerResponse{CorrectAnswer: answer}), nil
}

func (ps *PuzzleService) SetPuzzleVote(ctx context.Context, req *connect.Request[pb.PuzzleVoteRequest]) (*connect.Response[pb.PuzzleVoteResponse], error) {
	user, err := sessionUser(ctx, ps)
	if err != nil {
		return nil, err
	}
	err = SetPuzzleVote(ctx, ps.puzzleStore, user.UUID, req.Msg.PuzzleId, int(req.Msg.Vote))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.PuzzleVoteResponse{}), nil
}

func (ps *PuzzleService) StartPuzzleGenJob(ctx context.Context, req *connect.Request[pb.APIPuzzleGenerationJobRequest]) (*connect.Response[pb.APIPuzzleGenerationJobResponse], error) {
	user, err := sessionUser(ctx, ps)
	if err != nil {
		return nil, err
	}
	if !user.IsAdmin {
		return nil, apiserver.Unauthenticated(errNotAuthorized.Error())
	}
	log.Debug().Msgf("keys %s %s", req.Msg.SecretKey, ps.puzzleGenSecretKey)
	if req.Msg.SecretKey != ps.puzzleGenSecretKey {
		return nil, apiserver.PermissionDenied("must include puzzle generation secret key")
	}
	bts, err := protojson.Marshal(req.Msg.Request)
	if err != nil {
		return nil, err
	}
	// This message is meant to be copy-pasted from the terminal
	// when run locally. On production, though, it will try to execute
	// an ECS task.
	fmt.Printf("docker compose run --rm -w /opt/program/cmd/puzzlegen app go run . '%s'\n", string(bts))

	err = invokeECSPuzzleGen(ctx, string(bts), ps.ecsCluster, ps.puzzleGenTaskDef)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(&pb.APIPuzzleGenerationJobResponse{}), nil
}

func (ps *PuzzleService) GetPuzzleJobLogs(ctx context.Context, req *connect.Request[pb.PuzzleJobLogsRequest]) (*connect.Response[pb.PuzzleJobLogsResponse], error) {
	user, err := sessionUser(ctx, ps)
	if err != nil {
		return nil, err
	}
	if !user.IsAdmin {
		return nil, apiserver.Unauthenticated(errNotAuthorized.Error())
	}
	logs, err := GetPuzzleJobLogs(ctx, ps.puzzleStore, int(req.Msg.Limit), int(req.Msg.Offset))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.PuzzleJobLogsResponse{Logs: logs}), nil
}

func invokeECSPuzzleGen(ctx context.Context, arg, cluster, taskdef string) error {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithClientLogMode(aws.LogRetries|aws.LogRequestWithBody))
	if err != nil {
		return err
	}
	svc := ecs.NewFromConfig(cfg)
	input := &ecs.RunTaskInput{
		Cluster:        aws.String(cluster),
		TaskDefinition: aws.String(taskdef),
		Overrides: &types.TaskOverride{
			ContainerOverrides: []types.ContainerOverride{
				{
					Command: []string{
						"/opt/puzzle-generator",
						arg,
					},
					Name: aws.String("liwords-puzzlegen"),
				},
			},
		},
	}
	result, err := svc.RunTask(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			code := apiErr.ErrorCode()
			message := apiErr.ErrorMessage()
			return fmt.Errorf("aws error: code: %s, message: %s", code, message)
		} else {
			return err
		}
	}
	log.Info().Msgf("run-ecs-task-result: %v", result)
	return nil
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
		return nil, apiserver.InternalErr(err)
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
