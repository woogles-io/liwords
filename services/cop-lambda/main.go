package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/pair"
	"github.com/woogles-io/liwords/pkg/pair/cop"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const TimeLimit = 15

func HandleRequest(ctx context.Context, evt pair.LambdaInvokeIO) (*pair.LambdaInvokeIO, error) {
	log.Info().Int("msg-length", len(evt.Bytes)).Msg("got-pair-request")
	var pairRequest pb.PairRequest
	err := proto.Unmarshal(evt.Bytes, &pairRequest)
	if err != nil {
		return nil, err
	}
	log.Info().Msg("unmarshalled-pair-request")

	marshaler := protojson.MarshalOptions{
		Multiline:    true, // Enables pretty printing
		Indent:       "  ", // Sets the indentation level
		AllowPartial: true,
	}
	requestJSONData, err := marshaler.Marshal(&pairRequest)
	if err != nil {
		return nil, err
	}
	log.Info().Str("request-json", string(requestJSONData)).Msg("calling-cop-pair")

	pairResponse := cop.COPPair(&pairRequest)
	log.Info().Msg("marshalling-pair-response")

	pairResponseBytes, err := proto.Marshal(pairResponse)
	if err != nil {
		return nil, err
	}

	response := &pair.LambdaInvokeIO{
		Bytes: pairResponseBytes,
	}
	log.Info().Interface("invokeio-response", response).Msg("returning-pair-response")

	return response, nil
}

func main() {
	lambda.Start(HandleRequest)
}
