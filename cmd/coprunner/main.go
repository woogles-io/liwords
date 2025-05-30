package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"github.com/woogles-io/liwords/rpc/api/proto/pair_service/pair_serviceconnect"
	"github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
	"github.com/woogles-io/liwords/rpc/api/proto/tournament_service/tournament_serviceconnect"
	"gopkg.in/yaml.v3"
)

// All rounds are 1-indexed for the CLI API.
type Config struct {
	GibsonSpread               []int     `yaml:"gibson_spread"`
	ControlLossThreshold       float64   `yaml:"control_loss_threshold"`
	HopefulnessThreshold       []float64 `yaml:"hopefulness_threshold"`
	DivisionSims               int       `yaml:"division_sims"`
	ControlLossSims            int       `yaml:"control_loss_sims"`
	ControlLossActivationRound int       `yaml:"control_loss_activation_round"`
	TournamentID               string    `yaml:"tournament_id"`
	DivisionName               string    `yaml:"division_name"`
	PlacePrizes                int       `yaml:"place_prizes"`
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: coprunner <1-indexed-round> [config_file]")
	}
	wooglesURL := os.Getenv("WOOGLES_URL")
	if wooglesURL == "" {
		wooglesURL = "https://woogles.io"
	}

	round := os.Args[1]
	rdnum, err := strconv.ParseInt(round, 10, 64) // Validate that round is a number
	if err != nil {
		log.Fatal("Round must be a 1-indexed number")
	}
	if rdnum < 1 {
		log.Fatal("Round must be a positive number")
	}
	rdnum-- // Convert to 0-indexed for internal use

	cfgFile := "config.yml"
	if len(os.Args) > 2 {
		cfgFile = os.Args[2]
	}

	configData, err := os.ReadFile(cfgFile)
	if err != nil {
		log.Fatalf("Failed to read config.yml: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		log.Fatalf("Failed to parse config.yml: %v", err)
	}

	fmt.Printf("Config: %+v\n", cfg)
	ctx := context.Background()

	tourneyclient := tournament_serviceconnect.NewTournamentServiceClient(http.DefaultClient,
		wooglesURL+"/api")

	divs, err := tourneyclient.GetTournament(ctx, connect.NewRequest(&tournament_service.GetTournamentRequest{
		Id: cfg.TournamentID,
	}))
	if err != nil {
		log.Fatalf("Failed to get tournament: %v", err)
	}
	division := divs.Msg.Divisions[cfg.DivisionName]
	if division == nil {
		log.Fatalf("Division %s not found in tournament %s", cfg.DivisionName, cfg.TournamentID)
	}

	// Build DivisionPairings for each round up to the current round
	divisionPairings := make([]*ipc.RoundPairings, int(rdnum))
	for r := 0; r < int(rdnum); r++ {
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

	divisionResults := make([]*ipc.RoundResults, int(rdnum))
	for r := 0; r < int(rdnum); r++ {
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
	idx := int(nrounds - int32(rdnum+1))
	if idx < 0 {
		log.Fatalf("Round %d is out of bounds for the number of rounds %d", rdnum+1, nrounds)
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

	pairclient := pair_serviceconnect.NewPairServiceClient(http.DefaultClient,
		wooglesURL+"/api")
	res, err := pairclient.HandlePairRequest(ctx, connect.NewRequest(pairRequest))
	if err != nil {
		log.Fatalf("Failed to handle pair request: %v", err)
	}

	fmt.Printf("Pair log:\n")
	fmt.Println(res.Msg.Log)
}
