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

const (
	timeFormat = "2006-01-02T15:04:05.000Z"
)

type policy struct {
	name    string
	handler func(*pb.PairRequest, *copdatapkg.PrecompData, []int, int, int) (int64, bool)
}

// TODO: implement COP
// Constraints:
// prepaired
// koth

// Weights:
// gibson
// repeats
// rank diff
// casher-noncasher
// control loss
// repeated bye

var constraintPolicies = []policy{
	{
		name: "PolicyOne",
		handler: func(req *pb.PairRequest, copdata *copdatapkg.PrecompData, playerNodes []int, ri int, rj int) (int64, bool) {
			return 42, true
		},
	},
}

var weightPolicies = []policy{}

func COPPair(req *pb.PairRequest) *pb.PairResponse {
	logsb := &strings.Builder{}
	starttime := time.Now()
	resp := copPairWithLog(req, logsb)
	endtime := time.Now()
	duration := endtime.Sub(starttime)
	if resp.ErrorCode != pb.PairError_SUCCESS {
		logsb.WriteString("COP finished with error:\n" + resp.ErrorMessage + "\n")
	}
	logsb.WriteString(fmt.Sprintf("Started:  %s\nFinished: %s\nDuration: %s",
		starttime.Format(timeFormat), endtime.Format(timeFormat), duration))
	resp.Log = logsb.String()
	return resp
}

func copPairWithLog(req *pb.PairRequest, logsb *strings.Builder) *pb.PairResponse {
	req.Seed = int32(time.Now().Unix())
	marshaler := protojson.MarshalOptions{
		Multiline: true, // Enables pretty printing
		Indent:    "  ", // Sets the indentation level
	}
	jsonData, err := marshaler.Marshal(req)
	if err != nil {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_REQUEST_TO_JSON_FAILED,
			ErrorMessage: err.Error(),
		}
	}

	logsb.WriteString("Pairings Request:\n\n" + string(jsonData))

	resp := verifyreq.Verify(req)
	if resp != nil {
		return resp
	}

	copdata := copdatapkg.GetPrecompData(req, logsb)

	pairings, resp := copMinWeightMatching(req, copdata, logsb)

	if resp != nil {
		return resp
	}

	pairingsProto := make([]int32, len(pairings))
	for i, pairing := range pairings {
		pairingsProto[i] = int32(pairing)
	}

	return &pb.PairResponse{
		ErrorCode: pb.PairError_SUCCESS,
		Pairings:  pairingsProto,
	}
}

func copMinWeightMatching(req *pb.PairRequest, copdata *copdatapkg.PrecompData, logsb *strings.Builder) ([]int, *pb.PairResponse) {
	numPlayers := copdata.Standings.GetNumPlayers()
	playerNodes := []int{}
	divisionPlayerData := [][]string{}
	for i := 0; i < numPlayers; i++ {
		playerNodes[i] = copdata.Standings.GetPlayerIndex(i)
		divisionPlayerData = append(divisionPlayerData, copdata.Standings.StringDataForPlayer(req, i))
	}

	addBye := numPlayers%2 == 1
	if addBye {
		playerNodes = append(playerNodes, pkgstnd.ByePlayerIndex)
		divisionPlayerData = append(divisionPlayerData, []string{"", "", "BYE", "", ""})
	}

	numPlayerNodes := len(playerNodes)

	edges := []*matching.Edge{}

	// pairing, invalid reason, num repeats, total weight, individual weights
	pairingDetails := [][]string{}
	pairingDetailsheader := []string{"Pairing", "Invalid Reason", "Repeats", "Total"}
	for _, weightPolicy := range weightPolicies {
		pairingDetailsheader = append(pairingDetailsheader, weightPolicy.name)
	}
	numColums := len(pairingDetailsheader)
	for rankIdxI := 0; rankIdxI < numPlayerNodes; rankIdxI++ {
		for rankIdxJ := rankIdxI + 1; rankIdxJ < numPlayerNodes; rankIdxJ++ {
			pairingDataRow := []string{getMatchupString(divisionPlayerData, rankIdxI, rankIdxJ)}
			violatedConstraint := ""
			for _, constraintPolicy := range constraintPolicies {
				_, valid := constraintPolicy.handler(req, copdata, playerNodes, rankIdxI, rankIdxJ)
				if !valid {
					violatedConstraint = constraintPolicy.name
					break
				}
			}
			pairingDataRow = append(pairingDataRow, violatedConstraint)
			if violatedConstraint == "" {
				pairingDataRow = append(pairingDataRow, fmt.Sprintf("%d", copdata.PairingCounts[copdatapkg.GetPairingKey(rankIdxI, rankIdxJ)]))
				weightSum := int64(0)
				for _, weightPolicy := range weightPolicies {
					weight, _ := weightPolicy.handler(req, copdata, playerNodes, rankIdxI, rankIdxJ)
					weightSum += weight
					pairingDataRow = append(pairingDataRow, fmt.Sprintf("%d", weight))
				}
				pairingDataRow[3] = fmt.Sprintf("%d", weightSum)
				edges = append(edges, matching.NewEdge(playerNodes[rankIdxI], playerNodes[rankIdxJ], weightSum))
			} else {
				emptyColsToAdd := numColums - len(pairingDataRow)
				for i := 0; i < emptyColsToAdd; i++ {
					pairingDataRow = append(pairingDataRow, "")
				}
			}
			pairingDetails = append(pairingDetails, pairingDataRow)
		}
	}

	copdatapkg.WriteStringDataToLog("Pairing Weights", pairingDetailsheader, pairingDetails, logsb)

	pairings, totalWeight, err := matching.MinWeightMatching(edges, true)

	if err != nil {
		return nil, &pb.PairResponse{
			ErrorCode:    pb.PairError_MIN_WEIGHT_MATCHING,
			ErrorMessage: fmt.Sprintf("min weight matching error: %s\n", err.Error()),
		}
	}

	if addBye {
		pairings = pairings[:len(pairings)-1]
	}

	if len(pairings) != numPlayers {
		return nil, &pb.PairResponse{
			ErrorCode:    pb.PairError_INVALID_PAIRINGS_LENGTH,
			ErrorMessage: fmt.Sprintf("invalid pairings length %d for %d players", len(pairings), numPlayers),
		}
	}

	finalPairings := [][]string{}
	for i := 0; i < len(pairings); i++ {
		playerIdx := i
		oppIdx := pairings[i]
		if oppIdx > playerIdx {
			continue
		}
		finalPairingsRow := []string{getMatchupString(divisionPlayerData, playerIdx, oppIdx)}
		finalPairingsRow = append(finalPairingsRow, fmt.Sprintf("%d", copdata.PairingCounts[copdatapkg.GetPairingKey(playerIdx, oppIdx)]))
		finalPairings = append(finalPairings, finalPairingsRow)
	}

	copdatapkg.WriteStringDataToLog("Final Pairings", []string{"Pairing", "Repeats"}, finalPairings, logsb)

	logsb.WriteString(fmt.Sprintf("Total Weight: %d\n", totalWeight))

	// Convert the bye format from ByePlayerIndex to player index
	for i := 0; i < len(pairings); i++ {
		if pairings[i] == pkgstnd.ByePlayerIndex {
			pairings[i] = i
		}
	}

	return pairings, nil
}

func getCompatPlayerRecord(playerData []string) string {
	return fmt.Sprintf("%s (#%s) %s %s %s", playerData[0], playerData[1], playerData[2], playerData[3], playerData[4])
}

func getMatchupString(divisionPlayerData [][]string, i int, j int) string {
	return fmt.Sprintf("%s vs %s", getCompatPlayerRecord(divisionPlayerData[i]), getCompatPlayerRecord(divisionPlayerData[j]))
}
