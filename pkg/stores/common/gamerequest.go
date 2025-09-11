package common

import (
	"fmt"

	"github.com/rs/zerolog/log"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// UnmarshalGameRequest unmarshals GameRequest data with preference for jsonb over bytea.
// This supports the migration from protobuf bytea to JSON jsonb storage.
// gameRequestBytes: data from the game_request JSONB column
// requestBytes: data from the request bytea column
func UnmarshalGameRequest(gameRequestBytes []byte, requestBytes []byte) (*pb.GameRequest, error) {
	// First try to use the jsonb data if it exists
	if len(gameRequestBytes) > 0 {
		log.Debug().Bytes("grbts", gameRequestBytes).Msg("Unmarshaling GameRequest from jsonb column")
		gamereq := &pb.GameRequest{}
		// Try protojson unmarshal (for JSONB data)
		err := protojson.Unmarshal(gameRequestBytes, gamereq)
		if err == nil && gamereq.Rules != nil {
			// Successfully unmarshaled and has valid data
			return gamereq, nil
		}
		log.Debug().Err(err).Msg("Failed to unmarshal full GameRequest from jsonb column, falling back to bytea")
	}
	// Fall back to bytea protobuf data
	if len(requestBytes) == 0 {
		return nil, fmt.Errorf("no request data available in either jsonb or bytea format")
	}

	gamereq := &pb.GameRequest{}
	err := proto.Unmarshal(requestBytes, gamereq)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal request data: %w", err)
	}

	return gamereq, nil
}
