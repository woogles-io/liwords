package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"connectrpc.com/connect"
	"github.com/woogles-io/liwords/pkg/tournament"
	"github.com/woogles-io/liwords/rpc/api/proto/pair_service/pair_serviceconnect"
	"github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
	"github.com/woogles-io/liwords/rpc/api/proto/tournament_service/tournament_serviceconnect"
	"gopkg.in/yaml.v3"
)

// All rounds are 1-indexed for the CLI API.

func main() {
	if len(os.Args) < 2 {
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

	cfg := &tournament.COPIntermediateConfig{}
	if err := yaml.Unmarshal(configData, cfg); err != nil {
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
	pairRequest, err := tournament.TournamentDivisionToCOPRequest(division, rdnum, cfg)
	if err != nil {
		log.Fatalf("Failed to convert division to COP request: %v", err)
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
