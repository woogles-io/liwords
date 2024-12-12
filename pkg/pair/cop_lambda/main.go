package cop_lambda

import (
	"context"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/woogles-io/liwords/pkg/pair/cop"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/encoding/protojson"
)

const TimeLimit = 15

func HandleRequest(ctx context.Context, evt pb.PairRequest) (string, error) {
	var cancel context.CancelFunc
	_, cancel = context.WithTimeout(ctx, time.Duration(TimeLimit)*time.Second)
	pairResponse := cop.COPPair(&evt)
	cancel()
	pairResponseJSON, err := protojson.Marshal(pairResponse)
	if err != nil {
		return "", err
	}
	return string(pairResponseJSON), nil
}

func main() {
	lambda.Start(HandleRequest)
}
