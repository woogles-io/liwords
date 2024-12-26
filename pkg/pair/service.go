package pair

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/config"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type PairService struct {
	cfg          *config.Config
	lambdaClient *lambda.Client
}

type LambdaInvokeIO struct {
	Bytes []byte
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

	lambdaInvokeInputJSON, err := json.Marshal(LambdaInvokeIO{Bytes: pairRequestBytes})
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

	lo := &LambdaInvokeIO{}
	err = json.Unmarshal(out.Payload, lo)
	if err != nil {
		return nil, err
	}
	pairResponse := &pb.PairResponse{}
	err = proto.Unmarshal(lo.Bytes, pairResponse)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(pairResponse), nil
}
