package cop_lambda

import (
	"context"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/woogles-io/liwords/pkg/pair/cop"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/proto"
)

const TimeLimit = 15

type LambdaInvokeInput struct {
	PairRequestBytes []byte
}

func HandleRequest(ctx context.Context, evt LambdaInvokeInput) (string, error) {
	var pairRequest pb.PairRequest
	err := proto.Unmarshal(evt.PairRequestBytes, &pairRequest)
	if err != nil {
		return "", err
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(TimeLimit)*time.Second)
	defer cancel()

	if err := ctxWithTimeout.Err(); err != nil {
		return "", err
	}

	pairResponse := cop.COPPair(ctxWithTimeout, &pairRequest)

	pairResponseBytes, err := proto.Marshal(pairResponse)
	if err != nil {
		return "", err
	}
	return string(pairResponseBytes), nil
}

func main() {
	lambda.Start(HandleRequest)
}
