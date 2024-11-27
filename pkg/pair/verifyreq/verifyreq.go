package verifyreq

import (
	"fmt"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	MaxPlayerCount = 50000
)

// This function can be broken up and renamed as more pairing methods are added
func Verify(req *pb.PairRequest) *pb.PairResponse {
	// Verify number of players
	if req.ValidPlayers < 2 {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_PLAYER_COUNT_INSUFFICIENT,
			ErrorMessage: fmt.Sprintf("not enough players (%d)", req.ValidPlayers),
		}
	}
	if req.AllPlayers > MaxPlayerCount {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_PLAYER_COUNT_TOO_LARGE,
			ErrorMessage: fmt.Sprintf("too many players (%d)", req.AllPlayers),
		}
	}
	// Verify number of rounds
	if req.Rounds < 1 {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_ROUND_COUNT_INSUFFICIENT,
			ErrorMessage: fmt.Sprintf("not enough rounds (%d)", req.Rounds),
		}
	}
	// Verify player names
	if len(req.PlayerNames) != int(req.AllPlayers) {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_PLAYER_NAME_COUNT_INSUFFICIENT,
			ErrorMessage: fmt.Sprintf("player name count (%d) does not match number of players (%d)", len(req.PlayerNames), req.AllPlayers),
		}
	}
	for playerIdx, playerName := range req.PlayerNames {
		if playerName == "" {
			return &pb.PairResponse{
				ErrorCode:    pb.PairError_PLAYER_NAME_EMPTY,
				ErrorMessage: fmt.Sprintf("player name is empty for player %d", playerIdx+1),
			}
		}
	}

	// Verify division pairings
	numPairings := len(req.DivisionPairings)
	if numPairings > int(req.Rounds) {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_MORE_PAIRINGS_THAN_ROUNDS,
			ErrorMessage: fmt.Sprintf("more pairings (%d) than rounds (%d)", numPairings, req.Rounds),
		}
	}
	if numPairings == int(req.Rounds) {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_ALL_ROUNDS_PAIRED,
			ErrorMessage: fmt.Sprintf("equal pairings (%d) and rounds (%d)", numPairings, req.Rounds),
		}
	}
	for roundIdx, roundPairings := range req.DivisionPairings {
		if len(roundPairings.Pairings) != int(req.AllPlayers) {
			return &pb.PairResponse{
				ErrorCode:    pb.PairError_INVALID_ROUND_PAIRINGS_COUNT,
				ErrorMessage: fmt.Sprintf("round pairings length (%d) for round %d does not match number of players (%d)", len(roundPairings.Pairings), roundIdx+1, req.AllPlayers),
			}
		}
		seen := make(map[int]bool)
		for playerIdx, oppIdxInt32 := range roundPairings.Pairings {
			oppIdx := int(oppIdxInt32)
			if oppIdx < -1 || oppIdx >= int(req.AllPlayers) {
				return &pb.PairResponse{
					ErrorCode:    pb.PairError_PLAYER_INDEX_OUT_OF_BOUNDS,
					ErrorMessage: fmt.Sprintf("opponent (%d) for player %d in round %d is out of bounds", oppIdx+1, playerIdx+1, roundIdx+1),
				}
			}
			if oppIdx < 0 {
				if roundIdx != len(req.DivisionPairings)-1 {
					return &pb.PairResponse{
						ErrorCode:    pb.PairError_UNPAIRED_PLAYER,
						ErrorMessage: fmt.Sprintf("player (%d) not paired in round %d", playerIdx, roundIdx+1),
					}
				}
				continue
			}
			// Bye
			if playerIdx == oppIdx {
				_, playerIsAlreadyPaired := seen[playerIdx]
				if playerIsAlreadyPaired {
					return &pb.PairResponse{
						ErrorCode:    pb.PairError_INVALID_PAIRING,
						ErrorMessage: fmt.Sprintf("player %d is paired but also has a bye", playerIdx+1),
					}
				}
				seen[playerIdx] = true
			}
			if playerIdx < oppIdx {
				playerOppExists := seen[playerIdx]
				oppOppExists := seen[oppIdx]
				if playerOppExists || oppOppExists {
					return &pb.PairResponse{
						ErrorCode:    pb.PairError_INVALID_PAIRING,
						ErrorMessage: fmt.Sprintf("one of the players %d and %d is paired multiple times", playerIdx+1, oppIdx+1),
					}
				}
				playerOppIdx := int(roundPairings.Pairings[playerIdx])
				oppOppIdx := int(roundPairings.Pairings[oppIdx])
				if playerOppIdx != oppIdx || oppOppIdx != playerIdx {
					return &pb.PairResponse{
						ErrorCode:    pb.PairError_INVALID_PAIRING,
						ErrorMessage: fmt.Sprintf("opponents for players %d and %d are not the players themselves", playerIdx+1, oppIdx+1),
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
			ErrorCode:    pb.PairError_MORE_RESULTS_THAN_ROUNDS,
			ErrorMessage: fmt.Sprintf("more results (%d) than rounds (%d)", len(req.DivisionResults), req.Rounds),
		}
	}
	if len(req.DivisionResults) > len(req.DivisionPairings) {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_MORE_RESULTS_THAN_PAIRINGS,
			ErrorMessage: fmt.Sprintf("more results (%d) than pairings (%d)", len(req.DivisionResults), len(req.DivisionPairings)),
		}
	}
	for roundIdx, roundResults := range req.DivisionResults {
		if len(roundResults.Results) != int(req.AllPlayers) {
			return &pb.PairResponse{
				ErrorCode:    pb.PairError_INVALID_ROUND_RESULTS_COUNT,
				ErrorMessage: fmt.Sprintf("round results length (%d) for round %d does not match number of players (%d)", len(roundResults.Results), roundIdx+1, req.AllPlayers),
			}
		}
	}

	// Verify classes
	if len(req.Classes) > int(req.ValidPlayers) {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_MORE_CLASSES_THAN_PLAYERS,
			ErrorMessage: fmt.Sprintf("more classes (%d) than players (%d)", len(req.Classes), req.ValidPlayers),
		}
	}
	for classIdx, class := range req.Classes {
		if class >= req.ValidPlayers || class <= 0 {
			return &pb.PairResponse{
				ErrorCode:    pb.PairError_INVALID_CLASS,
				ErrorMessage: fmt.Sprintf("invalid class %d", class),
			}
		}
		if classIdx > 0 && class <= req.Classes[classIdx-1] {
			return &pb.PairResponse{
				ErrorCode:    pb.PairError_MISORDERED_CLASS,
				ErrorMessage: fmt.Sprintf("misordered class %d", class),
			}
		}
	}
	// Verify class prizes
	if len(req.ClassPrizes) > int(req.ValidPlayers) {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_INVALID_CLASS_PRIZES_COUNT,
			ErrorMessage: fmt.Sprintf("class prizes length (%d) does not match number of players (%d)", len(req.ClassPrizes), req.ValidPlayers),
		}
	}
	for _, classPrize := range req.ClassPrizes {
		if classPrize > req.ValidPlayers || classPrize < 1 {
			return &pb.PairResponse{
				ErrorCode:    pb.PairError_INVALID_CLASS_PRIZE,
				ErrorMessage: fmt.Sprintf("invalid class prize %d", classPrize),
			}
		}
	}

	// Verify gibsons
	if len(req.GibsonSpreads) > int(req.Rounds) {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_INVALID_GIBSON_SPREAD_COUNT,
			ErrorMessage: fmt.Sprintf("more gibson spreads (%d) than rounds (%d)", len(req.GibsonSpreads), req.Rounds),
		}
	}
	for _, gibsonSpread := range req.GibsonSpreads {
		if gibsonSpread < 0 {
			return &pb.PairResponse{
				ErrorCode:    pb.PairError_INVALID_GIBSON_SPREAD,
				ErrorMessage: fmt.Sprintf("invalid gibson spread %d", gibsonSpread),
			}
		}
	}

	// Verify control loss threshold
	if req.ControlLossThreshold < 0 || req.ControlLossThreshold > 1 {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_INVALID_CONTROL_LOSS_THRESHOLD,
			ErrorMessage: fmt.Sprintf("invalid control loss threshold %f", req.ControlLossThreshold),
		}
	}

	// Verify hopefulness threshold
	if req.HopefulnessThreshold < 0 || req.HopefulnessThreshold > 1 {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_INVALID_HOPEFULNESS_THRESHOLD,
			ErrorMessage: fmt.Sprintf("invalid hopefulness threshold %f", req.HopefulnessThreshold),
		}
	}

	// Verify division sims
	if req.DivisionSims < 1 {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_INVALID_DIVISION_SIMS,
			ErrorMessage: fmt.Sprintf("invalid division sims %d", req.DivisionSims),
		}
	}

	// Verify control loss sims
	if req.ControlLossSims < 1 {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_INVALID_CONTROL_LOSS_SIMS,
			ErrorMessage: fmt.Sprintf("invalid control loss sims %d", req.ControlLossSims),
		}
	}

	// Verify place prizes
	if req.PlacePrizes > req.ValidPlayers || req.PlacePrizes < 1 {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_INVALID_PLACE_PRIZES,
			ErrorMessage: fmt.Sprintf("invalid place prizes %d", req.PlacePrizes),
		}
	}

	// Verify removed players
	for _, removedPlayer := range req.RemovedPlayers {
		if removedPlayer < 0 || removedPlayer >= req.AllPlayers {
			return &pb.PairResponse{
				ErrorCode:    pb.PairError_INVALID_REMOVED_PLAYER,
				ErrorMessage: fmt.Sprintf("invalid removed player %d", removedPlayer),
			}
		}
	}

	if int(req.AllPlayers-req.ValidPlayers) != len(req.RemovedPlayers) {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_INVALID_VALID_PLAYER_COUNT,
			ErrorMessage: fmt.Sprintf("total players %d minus removed players %d does not equal valid players %d", req.AllPlayers, len(req.RemovedPlayers), req.ValidPlayers),
		}
	}

	return nil
}
