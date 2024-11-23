package cop

import (
	"fmt"
	"strings"
	"time"

	"github.com/woogles-io/liwords/pkg/matching"
	copdatapkg "github.com/woogles-io/liwords/pkg/pair/copdata"
	pkgstnd "github.com/woogles-io/liwords/pkg/pair/standings"

	"github.com/woogles-io/liwords/pkg/pair/verifyreq"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/encoding/protojson"
)

type policy struct {
	name    string
	handler func(*pb.PairRequest, *copdatapkg.PrecompData) (int64, bool)
}

var constraintPolicies = []policy{
	{
		name: "PolicyOne",
		handler: func(req *pb.PairRequest, copdata *copdatapkg.PrecompData) (int64, bool) {
			return 42, true
		},
	},
	{
		name: "PolicyTwo",
		handler: func(req *pb.PairRequest, copdata *copdatapkg.PrecompData) (int64, bool) {
			return 0, false
		},
	},
	{
		name: "PolicyThree",
		handler: func(req *pb.PairRequest, copdata *copdatapkg.PrecompData) (int64, bool) {
			return 100, true
		},
	},
}

var weightPolicies = []policy{
	{
		name: "PolicyOne",
		handler: func(req *pb.PairRequest, copdata *copdatapkg.PrecompData) (int64, bool) {
			return 42, true
		},
	},
	{
		name: "PolicyTwo",
		handler: func(req *pb.PairRequest, copdata *copdatapkg.PrecompData) (int64, bool) {
			return 0, false
		},
	},
	{
		name: "PolicyThree",
		handler: func(req *pb.PairRequest, copdata *copdatapkg.PrecompData) (int64, bool) {
			return 100, true
		},
	},
}

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
	req.Seed = int32(time.Now().Unix())
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

	copdata := copdatapkg.GetPrecompData(req, &logsb)

	numPlayers := copdata.Standings.GetNumPlayers()
	numPlayerNodes := numPlayers
	if numPlayers%2 == 1 {
		numPlayerNodes++
	}
	playerNodes := make([]int, numPlayerNodes)

	for i := 0; i < numPlayers; i++ {
		playerNodes[i] = copdata.Standings.GetPlayerIndex(i)
	}
	if numPlayers%2 == 1 {
		playerNodes[numPlayers] = pkgstnd.ByePlayerIndex
	}

	edges := []*matching.Edge{}

	for i := 0; i < numPlayerNodes; i++ {
		for j := i + 1; j < numPlayerNodes; j++ {
			for _, constraintPolicy := range constraintPolicies {
				_, valid := constraintPolicy.handler(req, copdata)
				if !valid {
					continue
				}
			}
			weightSum := int64(0)
			for _, weightPolicy := range weightPolicies {
				weight, _ := weightPolicy.handler(req, copdata)
				weightSum += weight
			}
			edges = append(edges, matching.NewEdge(i, j, weightSum))
		}
	}

	pairings, totalWeight, err := matching.MinWeightMatching(edges, true)

	// if err != nil {
	// 	log.Debug().Msgf("matching failed: %v", edges)
	// 	return nil, err
	// }

	// if len(pairings) != numberOfMembers {
	// 	log.Debug().Msgf("matching incomplete: %v, %v", pairings, edges)
	// 	return nil, errors.New("pairings and members are not the same length")
	// }

	// if weight >= entity.ProhibitiveWeight {
	// 	return nil, errors.New("prohibitive weight reached, pairings are not possible with these settings")
	// }

	// FIXME: implement pairings with the cop data
	pairings := make([]int32, req.ValidPlayers)

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	// FIXME: record start time and total time too
	logsb.WriteString(fmt.Sprintf("COP successfully finished at %s\n", timestamp))

	return &pb.PairResponse{
		ErrorCode: pb.PairError_SUCCESS,
		Message:   logsb.String(),
		Pairings:  pairings,
	}
}
