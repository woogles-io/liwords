package pair

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/pair/cop_lambda"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/proto"

	"connectrpc.com/connect"
)

type PairService struct {
	cfg          *config.Config
	lambdaClient *lambda.Client
}

func NewPairService(cfg *config.Config, lc *lambda.Client) *PairService {
	return &PairService{
		cfg:          cfg,
		lambdaClient: lc,
	}
}

func (ps *PairService) HandlePairRequest(ctx context.Context, req *connect.Request[pb.PairRequest]) (*connect.Response[pb.PairResponse], error) {
	pairRequestBytes, err := proto.Marshal(req.Msg)
	if err != nil {
		return nil, err
	}

	lambdaInvokeInputJSON, err := json.Marshal(cop_lambda.LambdaInvokeInput{PairRequestBytes: pairRequestBytes})
	if err != nil {
		return nil, err
	}

	out, err := ps.lambdaClient.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(ps.cfg.COPPairLambdaFunctionName),
		InvocationType: types.InvocationTypeRequestResponse,
		Payload:        lambdaInvokeInputJSON,
	})
	if err != nil {
		return nil, err
	}

	type lambdaOutput struct {
		StatusCode int    `json:"statusCode"`
		Body       string `json:"body"`
		Payload    string `json:"payload"`
	}

	lo := &lambdaOutput{}
	err = json.Unmarshal(out.Payload, lo)
	if err != nil {
		return nil, err
	}
	if lo.StatusCode != 200 {
		return nil, apiserver.InternalErr(errors.New(lo.Body))
	}
	bts, err := base64.StdEncoding.DecodeString(lo.Payload)
	if err != nil {
		return nil, err
	}
	pairResponse := &pb.PairResponse{}
	err = proto.Unmarshal(bts, pairResponse)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(pairResponse), nil
}
