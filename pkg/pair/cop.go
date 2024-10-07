package pair

import (
	"fmt"
	"strings"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/encoding/protojson"
)

type PrecompData struct {
	standings         *Standings
	pairingCounts     map[string]int
	repeatCounts      []int
	gibsonizedPlayers []bool
}

const (
	ByePlayerIndex uint64 = (uint64(1) << PlayerSpreadOffset) - 1
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

	var logsb strings.Builder

	_, resp := getPrecompData(req, &logsb)

	if resp.ErrorCode != pb.PairError_SUCCESS {
		return resp
	}

	resp.Message = logsb.String()
	return resp
}

func getPrecompData(req *pb.PairRequest, logsb *strings.Builder) (*PrecompData, *pb.PairResponse) {
	resp := verifyPairRequest(req)
	if resp != nil {
		return nil, resp
	}

	reqJSONstr := getReqAsJSONString(req)

	if reqJSONstr == "" {
		return nil, &pb.PairResponse{
			ErrorCode: pb.PairError_REQUEST_TO_JSON_FAILED,
			Message:   "unable to parse request",
		}
	}

	resp = &pb.PairResponse{
		ErrorCode: pb.PairError_SUCCESS,
		Pairings:  make([]int32, req.Players),
	}

	logsb.WriteString("Pairings Request:\n\n" + reqJSONstr)

	standings := createInitialStandings(req)

	logsb.WriteString("\n\nInitial Standings:\n\n" + getStandingsString(req, standings))

	pairingCounts, repeatCounts := getPairingFrequency(req)

	gibsonizedPlayers := getGibsonizedPlayers(req, standings)

	return &PrecompData{
		standings:         standings,
		pairingCounts:     pairingCounts,
		repeatCounts:      repeatCounts,
		gibsonizedPlayers: gibsonizedPlayers,
	}, resp
}

func verifyPairRequest(req *pb.PairRequest) *pb.PairResponse {
	// Verify number of players
	if req.Players < 2 {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_PLAYER_COUNT_INSUFFICIENT,
			Message:   fmt.Sprintf("not enough players (%d)", req.Players),
		}
	}
	if req.Players >= 1<<PlayerSpreadOffset {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_PLAYER_COUNT_TOO_LARGE,
			Message:   fmt.Sprintf("too many players (%d)", req.Players),
		}
	}
	// Verify number of rounds
	if req.Rounds < 1 {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_ROUND_COUNT_INSUFFICIENT,
			Message:   fmt.Sprintf("not enough rounds (%d)", req.Rounds),
		}
	}
	// Verify player names
	if len(req.PlayerNames) != int(req.Players) {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_PLAYER_NAME_COUNT_INSUFFICIENT,
			Message:   fmt.Sprintf("player name count (%d) does not match number of players (%d)", len(req.PlayerNames), req.Players),
		}
	}
	for playerIdx, playerName := range req.PlayerNames {
		if playerName == "" {
			return &pb.PairResponse{
				ErrorCode: pb.PairError_PLAYER_NAME_EMPTY,
				Message:   fmt.Sprintf("player name is empty for player %d", playerIdx+1),
			}
		}
	}

	// Verify division pairings
	numResults := len(req.DivisionResults)
	numPairings := len(req.DivisionPairings)
	if numPairings > int(req.Rounds) {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_MORE_PAIRINGS_THAN_ROUNDS,
			Message:   fmt.Sprintf("more pairings (%d) than rounds (%d)", numPairings, req.Rounds),
		}
	}
	if numPairings == int(req.Rounds) {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_ALL_ROUNDS_PAIRED,
			Message:   fmt.Sprintf("equal pairings (%d) and rounds (%d)", numPairings, req.Rounds),
		}
	}
	for roundIdx, roundPairings := range req.DivisionPairings {
		if len(roundPairings.Pairings) != int(req.Players) {
			return &pb.PairResponse{
				ErrorCode: pb.PairError_INVALID_ROUND_PAIRINGS_COUNT,
				Message:   fmt.Sprintf("round pairings length (%d) for round %d does not match number of players (%d)", len(roundPairings.Pairings), roundIdx+1, req.Players),
			}
		}
		seen := make(map[int]bool)
		for playerIdx, oppIdxInt32 := range roundPairings.Pairings {
			oppIdx := int(oppIdxInt32)
			if oppIdx < -1 || oppIdx >= int(req.Players) {
				return &pb.PairResponse{
					ErrorCode: pb.PairError_PLAYER_INDEX_OUT_OF_BOUNDS,
					Message:   fmt.Sprintf("opponent (%d) for player %d in round %d is out of bounds", oppIdx+1, playerIdx+1, roundIdx+1),
				}
			}
			if oppIdx < 0 {
				if roundIdx < numResults {
					return &pb.PairResponse{
						ErrorCode: pb.PairError_UNPAIRED_PLAYER,
						Message:   fmt.Sprintf("player (%d) not paired in round %d", playerIdx, roundIdx+1),
					}
				}
				continue
			}
			// Bye
			if playerIdx == oppIdx {
				_, playerIsAlreadyPaired := seen[playerIdx]
				if playerIsAlreadyPaired {
					return &pb.PairResponse{
						ErrorCode: pb.PairError_INVALID_PAIRING,
						Message:   fmt.Sprintf("player %d is paired but also has a bye", playerIdx+1),
					}
				}
				seen[playerIdx] = true
			}
			if playerIdx < oppIdx {
				playerOppExists := seen[playerIdx]
				oppOppExists := seen[oppIdx]
				if playerOppExists || oppOppExists {
					return &pb.PairResponse{
						ErrorCode: pb.PairError_INVALID_PAIRING,
						Message:   fmt.Sprintf("one of the players %d and %d is paired multiple times", playerIdx+1, oppIdx+1),
					}
				}
				playerOppIdx := int(roundPairings.Pairings[playerIdx])
				oppOppIdx := int(roundPairings.Pairings[oppIdx])
				if playerOppIdx != oppIdx || oppOppIdx != playerIdx {
					return &pb.PairResponse{
						ErrorCode: pb.PairError_INVALID_PAIRING,
						Message:   fmt.Sprintf("opponents for players %d and %d are not the players themselves", playerIdx+1, oppIdx+1),
					}
				}
				seen[playerIdx] = true
				seen[oppIdx] = true
			}
		}
	}

	// Verify division results
	if len(req.DivisionResults) > int(req.Rounds) {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_MORE_RESULTS_THAN_ROUNDS,
			Message:   fmt.Sprintf("more results (%d) than rounds (%d)", len(req.DivisionResults), req.Rounds),
		}
	}
	if len(req.DivisionResults) > len(req.DivisionPairings) {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_MORE_RESULTS_THAN_PAIRINGS,
			Message:   fmt.Sprintf("more results (%d) than pairings (%d)", len(req.DivisionResults), len(req.DivisionPairings)),
		}
	}
	for roundIdx, roundResults := range req.DivisionResults {
		if len(roundResults.Results) != int(req.Players) {
			return &pb.PairResponse{
				ErrorCode: pb.PairError_INVALID_ROUND_RESULTS_COUNT,
				Message:   fmt.Sprintf("round results length (%d) for round %d does not match number of players (%d)", len(roundResults.Results), roundIdx+1, req.Players),
			}
		}
	}

	// Verify classes
	if len(req.Classes) > int(req.Players) {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_MORE_CLASSES_THAN_PLAYERS,
			Message:   fmt.Sprintf("more classes (%d) than players (%d)", len(req.Classes), req.Players),
		}
	}
	for classIdx, class := range req.Classes {
		if class >= req.Players || class <= 0 {
			return &pb.PairResponse{
				ErrorCode: pb.PairError_INVALID_CLASS,
				Message:   fmt.Sprintf("invalid class %d", class),
			}
		}
		if classIdx > 0 && class <= req.Classes[classIdx-1] {
			return &pb.PairResponse{
				ErrorCode: pb.PairError_MISORDERED_CLASS,
				Message:   fmt.Sprintf("misordered class %d", class),
			}
		}
	}
	// Verify class prizes
	if len(req.ClassPrizes) > int(req.Players) {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_INVALID_CLASS_PRIZES_COUNT,
			Message:   fmt.Sprintf("class prizes length (%d) does not match number of players (%d)", len(req.ClassPrizes), req.Players),
		}
	}
	for _, classPrize := range req.ClassPrizes {
		if classPrize > req.Players || classPrize < 1 {
			return &pb.PairResponse{
				ErrorCode: pb.PairError_INVALID_CLASS_PRIZE,
				Message:   fmt.Sprintf("invalid class prize %d", classPrize),
			}
		}
	}

	// Verify gibsons
	if len(req.GibsonSpreads) > int(req.Players) {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_INVALID_GIBSON_SPREAD_COUNT,
			Message:   fmt.Sprintf("more gibson spreads (%d) than players (%d)", len(req.GibsonSpreads), req.Players),
		}
	}
	for _, gibsonSpread := range req.GibsonSpreads {
		if gibsonSpread < 0 {
			return &pb.PairResponse{
				ErrorCode: pb.PairError_INVALID_GIBSON_SPREAD,
				Message:   fmt.Sprintf("invalid gibson spread %d", gibsonSpread),
			}
		}
	}

	// Verify control loss threshold
	if req.ControlLossThreshold < 0 || req.ControlLossThreshold > 1 {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_INVALID_CONTROL_LOSS_THRESHOLD,
			Message:   fmt.Sprintf("invalid control loss threshold %f", req.ControlLossThreshold),
		}
	}

	// Verify hopefulness threshold
	if req.HopefulnessThreshold < 0 || req.HopefulnessThreshold > 1 {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_INVALID_HOPEFULNESS_THRESHOLD,
			Message:   fmt.Sprintf("invalid hopefulness threshold %f", req.HopefulnessThreshold),
		}
	}

	// Verify division sims
	if req.DivisionSims < 1 {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_INVALID_DIVISION_SIMS,
			Message:   fmt.Sprintf("invalid division sims %d", req.DivisionSims),
		}
	}

	// Verify control loss sims
	if req.ControlLossSims < 1 {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_INVALID_CONTROL_LOSS_SIMS,
			Message:   fmt.Sprintf("invalid control loss sims %d", req.ControlLossSims),
		}
	}

	// Verify place prizes
	if req.PlacePrizes > req.Players || req.PlacePrizes < 1 {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_INVALID_PLACE_PRIZES,
			Message:   fmt.Sprintf("invalid place prizes %d", req.PlacePrizes),
		}
	}

	return nil
}

func getReqAsJSONString(req *pb.PairRequest) string {
	marshaler := protojson.MarshalOptions{
		Multiline: true, // Enables pretty printing
		Indent:    "  ", // Sets the indentation level
	}
	jsonData, err := marshaler.Marshal(req)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func createInitialStandings(req *pb.PairRequest) *Standings {
	standings := CreateEmptyStandings(int(req.Players))

	for roundIdx, roundResults := range req.DivisionResults {
		for playerIdx, playerScore := range roundResults.Results {
			oppIdx := int(req.DivisionPairings[roundIdx].Pairings[playerIdx])
			if playerIdx == int(oppIdx) {
				// Bye
				if playerScore >= 0 {
					standings.IncrementPlayerWins(playerIdx)
				}
				standings.IncrementPlayerSpread(playerIdx, int(playerScore))
			} else if playerIdx < oppIdx {
				oppScore := roundResults.Results[oppIdx]
				playerSpread := playerScore - oppScore
				oppSpread := oppScore - playerScore
				if playerSpread > 0 {
					standings.IncrementPlayerWins(playerIdx)
				} else if playerSpread < 0 {
					standings.IncrementPlayerWins(oppIdx)
				} else {
					standings.IncrementPlayerTies(playerIdx)
					standings.IncrementPlayerTies(oppIdx)
				}
				standings.IncrementPlayerSpread(playerIdx, int(playerSpread))
				standings.IncrementPlayerSpread(oppIdx, int(oppSpread))
			}
		}
	}

	standings.Sort()

	return standings
}

func getStandingsString(req *pb.PairRequest, standings *Standings) string {
	maxNameLength := 0
	for _, playerName := range req.PlayerNames {
		if len(playerName) > maxNameLength {
			if len(playerName) > 30 {
				maxNameLength = 30
			} else {
				maxNameLength = len(playerName)
			}
		}
	}

	playerNameColWidth := maxNameLength
	if playerNameColWidth > 30 {
		playerNameColWidth = 30
	}

	headerFormat := fmt.Sprintf("%%-4s | %%-%ds | %%-4s | %%-6s\n", playerNameColWidth)
	rowFormat := fmt.Sprintf("%%-4d | %%-%ds | %%-4.1f | %%-6d\n", playerNameColWidth)

	var sb strings.Builder
	header := fmt.Sprintf(headerFormat, "Rank", "Name", "Wins", "Spread")
	sb.WriteString(header)
	sb.WriteString(strings.Repeat("-", len(header)) + "\n")

	for rankIdx := 0; rankIdx < int(req.Players); rankIdx++ {
		playerIdx := standings.GetPlayerIndex(rankIdx)
		wins := standings.GetPlayerWins(rankIdx)
		spread := standings.GetPlayerSpread(rankIdx)
		playerName := req.PlayerNames[playerIdx]
		if len(playerName) > 30 {
			playerName = playerName[:30]
		}
		sb.WriteString(fmt.Sprintf(rowFormat, rankIdx+1, playerName, wins, spread))
	}

	return sb.String()
}

func getPairingFrequency(req *pb.PairRequest) (map[string]int, []int) {
	pairingCounts := make(map[string]int)
	totalRepeats := make([]int, req.Players)
	for _, roundPairings := range req.DivisionPairings {
		for playerIdx := 0; playerIdx < len(roundPairings.Pairings); playerIdx++ {
			oppIdx := int(roundPairings.Pairings[playerIdx])
			var pairingKey string
			if playerIdx == oppIdx {
				pairingKey = fmt.Sprintf("%d:BYE", playerIdx)
			} else if playerIdx < oppIdx {
				pairingKey = fmt.Sprintf("%d:%d", playerIdx, oppIdx)
			}
			if pairingCounts[pairingKey] > 0 {
				totalRepeats[playerIdx]++
				if playerIdx != oppIdx {
					totalRepeats[oppIdx]++
				}
			}
			pairingCounts[pairingKey]++
		}
	}
	return pairingCounts, totalRepeats
}

// Assumes the standings are already sorted
func getGibsonizedPlayers(req *pb.PairRequest, standings *Standings) []bool {
	gibsonizedPlayers := make([]bool, req.Players)
	roundsRemaining := int(req.Rounds) - len(req.DivisionResults)
	numInputGibonsSpreads := len(req.GibsonSpreads)
	cumeGibsonSpread := 0
	for round := roundsRemaining - 1; round >= 0; round-- {
		if round >= numInputGibonsSpreads {
			cumeGibsonSpread += int(req.GibsonSpreads[numInputGibonsSpreads-1])
		} else {
			cumeGibsonSpread += int(req.GibsonSpreads[round])
		}
	}

	for playerIdx := 0; playerIdx < int(req.Players); playerIdx++ {
		gibsonizedPlayers[playerIdx] = true
		if playerIdx > 0 && standings.CanCatch(roundsRemaining, cumeGibsonSpread, playerIdx-1, playerIdx) {
			gibsonizedPlayers[playerIdx] = false
			continue
		}
		if playerIdx < int(req.Players)-1 && standings.CanCatch(roundsRemaining, cumeGibsonSpread, playerIdx, playerIdx+1) {
			gibsonizedPlayers[playerIdx] = false
			continue
		}
	}
	return gibsonizedPlayers
}
