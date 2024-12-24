package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/pair"
	"github.com/woogles-io/liwords/pkg/pair/cop"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const TimeLimit = 15

func HandleRequest(ctx context.Context, evt pair.LambdaInvokeIO) (string, error) {
	log.Info().Int("msg-length", len(evt.Bytes)).Msg("got-pair-request")
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

	lambdaInvokeIOJSON, err := json.Marshal(&pair.LambdaInvokeIO{
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
