package pair

import (
	"fmt"
	"sort"
	"strings"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	PlayerWinsOffset   int    = 48
	PlayerSpreadOffset int    = 16
	InitialSpreadValue int    = 1 << (PlayerWinsOffset - PlayerSpreadOffset - 1)
	PlayerIndexMask    uint64 = 0xFFFF
)

func COPPair(req *pb.PairRequest) *pb.PairResponse {
	resp := verifyPairRequest(req)
	if resp != nil {
		return resp
	}

	reqJSONstr := getReqAsJSONString(req)

	if reqJSONstr == "" {
		return &pb.PairResponse{
			ErrorCode: pb.PairError_REQUEST_TO_JSON_FAILED,
			Message:   "unable to parse request",
		}
	}

	resp = &pb.PairResponse{
		ErrorCode: pb.PairError_SUCCESS,
		Message:   "",
		Pairings:  make([]int32, req.Players),
	}

	var logsb strings.Builder

	logsb.WriteString("Pairings Request:\n\n" + reqJSONstr)

	standings := createStandings(req)

	logsb.WriteString("\n\nInitial Standings:\n\n" + getStandingsString(req, standings))

	// TODO: implement COP
	// Required data:
	// standings
	// sim results
	// number of times played
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

	resp.Message = logsb.String()
	return resp
}

// To keep everything as ints and avoid int floor division for ties:
// A win counts as 2
// A tie counts as 1
// A loss counts as 0
func incPlayerWins(standings []uint64, playerIdx int, winScore int) {
	standings[playerIdx] += uint64(winScore) << PlayerWinsOffset
}

func incPlayerSpread(standings []uint64, playerIdx int, spread int) {
	if spread < 0 {
		standings[playerIdx] -= uint64((-spread)) << PlayerSpreadOffset
	} else {
		standings[playerIdx] += uint64(spread) << PlayerSpreadOffset
	}
}

// Standings are implemented as an array of 64-bit integers
// for tournament simulation performance.
func createStandings(req *pb.PairRequest) []uint64 {
	standings := make([]uint64, req.Players)

	for playerIdx := 0; playerIdx < int(req.Players); playerIdx++ {
		standings[playerIdx] = uint64(playerIdx)
		incPlayerSpread(standings, playerIdx, InitialSpreadValue)
	}

	for roundIdx, roundResults := range req.DivisionResults {
		for playerIdx, playerScore := range roundResults.Results {
			oppIdx := int(req.DivisionPairings[roundIdx].Pairings[playerIdx])
			if playerIdx == int(oppIdx) {
				// Bye
				if playerScore >= 0 {
					incPlayerWins(standings, playerIdx, 2)
				}
				incPlayerSpread(standings, playerIdx, int(playerScore))
			} else if playerIdx < oppIdx {
				oppScore := roundResults.Results[oppIdx]
				playerSpread := playerScore - oppScore
				oppSpread := oppScore - playerScore
				if playerSpread > 0 {
					incPlayerWins(standings, playerIdx, 2)
				} else if playerSpread < 0 {
					incPlayerWins(standings, oppIdx, 2)
				} else {
					incPlayerWins(standings, playerIdx, 1)
					incPlayerWins(standings, oppIdx, 1)
				}
				incPlayerSpread(standings, playerIdx, int(playerSpread))
				incPlayerSpread(standings, oppIdx, int(oppSpread))
			}
		}
	}

	sort.Slice(standings, func(i, j int) bool {
		return standings[i] > standings[j]
	})

	return standings
}

func getStandingsString(req *pb.PairRequest, standings []uint64) string {
	var sb strings.Builder
	header := fmt.Sprintf("%-30s | %-4s | %-6s\n", "Player Name", "Wins", "Spread")
	sb.WriteString(header)
	sb.WriteString(strings.Repeat("-", len(header)) + "\n")

	for _, playerData := range standings {
		playerIdx := playerData & uint64(PlayerIndexMask)
		wins := (playerData >> PlayerWinsOffset) & 0xFFFF
		spread := ((playerData >> PlayerSpreadOffset) & 0xFFFFFFFF)

		humanReadableWins := float64(wins) / 2

		humanReadableSpread := 0
		if spread > uint64(InitialSpreadValue) {
			humanReadableSpread = int(spread - uint64(InitialSpreadValue))
		} else {
			humanReadableSpread = -int(uint64(InitialSpreadValue) - spread)
		}

		playerName := req.PlayerNames[playerIdx]
		sb.WriteString(fmt.Sprintf("%-30s | %-4.1f | %-6d\n", playerName, humanReadableWins, humanReadableSpread))
	}

	return sb.String()
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
		if class >= req.Players || class < 0 {
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
