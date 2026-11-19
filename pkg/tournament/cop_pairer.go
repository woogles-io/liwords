package tournament

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/pair"
	"github.com/woogles-io/liwords/pkg/pair/cop"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// A COPPairer runs COP on a pairing request. COP is a pure function of the
// request, so running it twice on the same request gives the same answer.
type COPPairer interface {
	PairCOP(ctx context.Context, req *pb.PairRequest) (*pb.PairResponse, error)
}

// LocalCOPPairer runs COP in this process.
type LocalCOPPairer struct{}

// PairCOP runs COP in a goroutine and waits for it, or for ctx. COP takes
// no context, so a pairing abandoned with ctx keeps using CPU until it ends.
func (LocalCOPPairer) PairCOP(ctx context.Context, req *pb.PairRequest) (*pb.PairResponse, error) {
	done := make(chan *pb.PairResponse, 1)
	go func() { done <- cop.COPPair(req) }()
	select {
	case resp := <-done:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// LambdaCOPPairer runs COP on a Lambda function running cop-lambda
// (services/cop-lambda).
type LambdaCOPPairer struct {
	Client       *lambda.Client
	FunctionName string
}

func (l LambdaCOPPairer) PairCOP(ctx context.Context, req *pb.PairRequest) (*pb.PairResponse, error) {
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(pair.LambdaInvokeIO{Bytes: reqBytes})
	if err != nil {
		return nil, err
	}
	out, err := l.Client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(l.FunctionName),
		InvocationType: types.InvocationTypeRequestResponse,
		Payload:        payload,
	})
	if err != nil {
		return nil, err
	}
	if out.FunctionError != nil {
		return nil, fmt.Errorf("cop lambda: %s: %s", aws.ToString(out.FunctionError), out.Payload)
	}
	lo := &pair.LambdaInvokeIO{}
	if err := json.Unmarshal(out.Payload, lo); err != nil {
		return nil, err
	}
	resp := &pb.PairResponse{}
	if err := proto.Unmarshal(lo.Bytes, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
