package cop_lambda

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/woogles-io/liwords/pkg/pair/cop"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/proto"
)

const TimeLimit = 15

type LambdaInvokeIO struct {
	Bytes []byte
}

func HandleRequest(ctx context.Context, evt LambdaInvokeIO) (string, error) {
	var pairRequest pb.PairRequest
	err := proto.Unmarshal(evt.Bytes, &pairRequest)
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

	lambdaInvokeIOJSON, err := json.Marshal(&LambdaInvokeIO{
		Bytes: pairResponseBytes,
	})
	if err != nil {
		return "", err
	}

	return string(lambdaInvokeIOJSON), nil
}

func main() {
	lambda.Start(HandleRequest)
}
