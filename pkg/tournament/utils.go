package tournament

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type COPIntermediateConfig struct {
	GibsonSpread         []int     `yaml:"gibson_spread"`
	ControlLossThreshold float64   `yaml:"control_loss_threshold"`
	HopefulnessThreshold []float64 `yaml:"hopefulness_threshold"`
	DivisionSims         int       `yaml:"division_sims"`
	ControlLossSims      int       `yaml:"control_loss_sims"`
	// ControlLossActivationRound is 0-indexed
	ControlLossActivationRound int    `yaml:"control_loss_activation_round"`
	TournamentID               string `yaml:"tournament_id"`
	DivisionName               string `yaml:"division_name"`
	PlacePrizes                int    `yaml:"place_prizes"`
}

// TournamentDivisionToCOPRequest converts a TournamentDivisionDataResponse to a COPRequest.
// The round that is passed in must be 0-indexed.
func TournamentDivisionToCOPRequest(
	division *ipc.TournamentDivisionDataResponse,
	round int64,
	cfg *COPIntermediateConfig,
) (*ipc.PairRequest, error) {

	// Build DivisionPairings for each round up to the current round
	divisionPairings := make([]*ipc.RoundPairings, int(round))
	for r := 0; r < int(round); r++ {
		// Initialize pairings for this round with -1 (no opponent)
		pairings := make([]int32, len(division.Players.Persons))
		for i := range pairings {
			pairings[i] = -1
		}
		// Fill in pairings from the division's pairingMap
		for _, pairing := range division.PairingMap {
			if int(pairing.Round) == r {
				if len(pairing.Players) == 2 {
					a, b := pairing.Players[0], pairing.Players[1]
					pairings[a] = b
					pairings[b] = a
				}
			}
		}
		divisionPairings[r] = &ipc.RoundPairings{Pairings: pairings}
	}

	divisionResults := make([]*ipc.RoundResults, int(round))
	for r := 0; r < int(round); r++ {
		// Initialize results for this round with -1 (no result)
		results := make([]int32, len(division.Players.Persons))
		for i := range results {
			results[i] = -1
		}
		// Fill in results from the division's resultMap
		for _, pairing := range division.PairingMap {
			if int(pairing.Round) == r {
				if len(pairing.Players) == 2 && len(pairing.Games) == 1 {
					a, b := pairing.Players[0], pairing.Players[1]
					results[a] = int32(pairing.Games[0].Scores[0])
					results[b] = int32(pairing.Games[0].Scores[1])
				}
			}
		}
		divisionResults[r] = &ipc.RoundResults{Results: results}
	}

	nrounds := int32(len(division.RoundControls))
	var gibsonSpread int32
	idx := int(nrounds - int32(round+1))
	if idx < 0 {
		return nil, fmt.Errorf("round %d is out of bounds for the number of rounds %d", round+1, nrounds)
	}
	if idx >= len(cfg.GibsonSpread) {
		gibsonSpread = int32(cfg.GibsonSpread[len(cfg.GibsonSpread)-1])
	} else {
		gibsonSpread = int32(cfg.GibsonSpread[idx])
	}

	var hopefulnessThreshold float64
	if idx >= len(cfg.HopefulnessThreshold) {
		hopefulnessThreshold = cfg.HopefulnessThreshold[len(cfg.HopefulnessThreshold)-1]
	} else {
		hopefulnessThreshold = cfg.HopefulnessThreshold[idx]
	}
	removedPlayers := []int32{}
	for i, p := range division.Players.Persons {
		if p.Suspended {
			removedPlayers = append(removedPlayers, int32(i))
		}
	}

	// build up pair request
	pairRequest := &ipc.PairRequest{
		PairMethod: ipc.PairMethod_COP,
		PlayerNames: lo.Map(
			division.Players.Persons,
			func(p *ipc.TournamentPerson, _ int) string {
				if strings.Contains(p.Id, ":") {
					return strings.Split(p.Id, ":")[1] // Extract the player name from the ID
				}
				return p.Id
			}),
		DivisionPairings:           divisionPairings,
		DivisionResults:            divisionResults,
		AllPlayers:                 int32(len(division.Players.Persons)),
		RemovedPlayers:             removedPlayers,
		PlayerClasses:              make([]int32, len(division.Players.Persons)),
		ValidPlayers:               int32(len(division.Players.Persons) - len(removedPlayers)),
		Rounds:                     int32(nrounds),
		PlacePrizes:                int32(cfg.PlacePrizes),
		DivisionSims:               int32(cfg.DivisionSims),
		ControlLossSims:            int32(cfg.ControlLossSims),
		ControlLossActivationRound: int32(cfg.ControlLossActivationRound),
		GibsonSpread:               gibsonSpread,
		ControlLossThreshold:       cfg.ControlLossThreshold,
		HopefulnessThreshold:       hopefulnessThreshold,
	}
	return pairRequest, nil
}
