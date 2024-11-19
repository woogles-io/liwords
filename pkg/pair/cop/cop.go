package cop

import (
	"fmt"
	"strings"
	"time"

	"github.com/woogles-io/liwords/pkg/pair/copdata"
	"github.com/woogles-io/liwords/pkg/pair/verifyreq"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/encoding/protojson"
)

func COPPair(req *pb.PairRequest) *pb.PairResponse {
	// TODO: implement COP
	// Required data:
	// standings
	// sim results
	// number of times played - pairingCounts
	// total number of repeats - repeats
	// previous pairing

	// Weights:
	// repeats
	// rank diff
	// casher-noncasher
	// control loss

	// Constraints:
	// prepaired
	// koth
	// gibson
	// repeated bye

	resp := verifyreq.Verify(req)
	if resp != nil {
		return resp
	}

	marshaler := protojson.MarshalOptions{
		Multiline: true, // Enables pretty printing
		Indent:    "  ", // Sets the indentation level
	}
	jsonData, err := marshaler.Marshal(req)
	if err != nil {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_REQUEST_TO_JSON_FAILED,
			Message:   err.Error(),
		}
	}

	var logsb strings.Builder

	logsb.WriteString("Pairings Request:\n\n" + string(jsonData))

	_ = copdata.GetPrecompData(req, &logsb)

	// FIXME: set the random seed

	// FIXME: implement pairings with the cop data
	pairings := make([]int32, req.ValidPlayers)

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logsb.WriteString(fmt.Sprintf("COP successfully finished at %s\n", timestamp))

	return &pb.PairResponse{
		ErrorCode: pb.PairError_SUCCESS,
		Message:   logsb.String(),
		Pairings:  pairings,
	}
}
