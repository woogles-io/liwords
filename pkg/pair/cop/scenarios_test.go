package cop_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/pair/cop"
	"github.com/woogles-io/liwords/pkg/pair/standings"
	pairtestutils "github.com/woogles-io/liwords/pkg/pair/testutils"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"golang.org/x/exp/rand"
)

// Pairing pattern from AddNDummyRounds for N players: (0,1),(2,3),(4,5),...
// Results determine who wins each match.
//
// All tests require COP_SCENARIOS=1 to run (on-demand only).

func writeScenarioLog(t *testing.T, filename string, log string) {
	t.Helper()
	if err := os.WriteFile(filename, []byte(log), 0644); err != nil {
		t.Logf("failed to write log file %s: %v", filename, err)
	}
}

// Scenario 1: Tight race at the top; two players nearly tied for 1st with 2 rounds left.
// After 6 rounds: P0=5-1 (big spread), P2=5-1 (small spread), PlacePrizes=2.
func TestControlLoss_TightRaceAtTop(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping tight-race-at-top scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"Alice", "Bob", "Charlie", "Dave", "Eric", "Frank", "Grace", "Holly"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               200,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.02,
		AllPlayers:                 8,
		ValidPlayers:               8,
		Rounds:                     8,
		PlacePrizes:                2,
		DivisionSims:               1000,
		ControlLossSims:            1000,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
	}
	// 6 rounds with same pairing pattern: (0v1),(2v3),(4v5),(6v7)
	// P0 wins 5/6, P2 wins 5/6 — tight race for 1st by spread.
	// P0 wins by large margins; P2 wins by small margins.
	pairtestutils.AddNDummyRounds(req, 6)
	pairtestutils.AddRoundResultsStr(req, "480 340 420 390 400 370 390 380") // R1: P0 big, P2 small
	pairtestutils.AddRoundResultsStr(req, "470 350 415 395 395 375 385 385") // R2
	pairtestutils.AddRoundResultsStr(req, "460 360 410 400 390 380 380 390") // R3: P6 wins
	pairtestutils.AddRoundResultsStr(req, "450 370 405 405 385 385 390 380") // R4: tie→pick higher
	pairtestutils.AddRoundResultsStr(req, "440 380 400 410 380 390 395 375") // R5: P0 wins, P2 LOSES
	pairtestutils.AddRoundResultsStr(req, "430 390 395 415 375 395 400 370") // R6: P0 wins, P2 LOSES

	resp := cop.COPPair(req)
	writeScenarioLog(t, "tight_race_at_top.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// Scenario 2: Dominant leader with huge spread; first place could be in control loss territory.
// After 6 rounds: P0=6-0 with +700 spread, P2=4-2, PlacePrizes=2.
func TestControlLoss_DominantLeader(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping dominant-leader scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"Alice", "Bob", "Charlie", "Dave", "Eric", "Frank", "Grace", "Holly"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               200,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.02,
		AllPlayers:                 8,
		ValidPlayers:               8,
		Rounds:                     8,
		PlacePrizes:                2,
		DivisionSims:               1000,
		ControlLossSims:            1000,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
	}
	// P0 wins all 6 by 100+; P2 wins 4/6, P4 wins 3/6, P6 wins 3/6.
	pairtestutils.AddNDummyRounds(req, 6)
	pairtestutils.AddRoundResultsStr(req, "560 340 430 390 400 370 400 380") // R1
	pairtestutils.AddRoundResultsStr(req, "570 350 420 400 390 380 410 370") // R2
	pairtestutils.AddRoundResultsStr(req, "580 360 410 410 380 390 420 360") // R3: P2 loses
	pairtestutils.AddRoundResultsStr(req, "590 370 400 420 370 400 430 350") // R4: P2 loses again
	pairtestutils.AddRoundResultsStr(req, "600 380 440 430 360 410 400 360") // R5: P2 wins
	pairtestutils.AddRoundResultsStr(req, "610 390 450 440 350 420 390 370") // R6: P2 wins

	resp := cop.COPPair(req)
	writeScenarioLog(t, "dominant_leader.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// Scenario 3: Crowded second place; three players tied for 2nd with 2 rounds left.
// 12 players, PlacePrizes=2: P0=6-0, P2=P4=P6=4-2, all chasing the single 2nd-place prize.
func TestControlLoss_CrowdedSecond(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping crowded-second scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9", "P10", "P11"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               200,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.02,
		AllPlayers:                 12,
		ValidPlayers:               12,
		Rounds:                     8,
		PlacePrizes:                2,
		DivisionSims:               1000,
		ControlLossSims:            1000,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
	}
	// Pairings: (0v1),(2v3),(4v5),(6v7),(8v9),(10v11)
	// P0=6-0, P2=P4=P6=4-2, P8=3-3, P10=2-4
	pairtestutils.AddNDummyRounds(req, 6)
	pairtestutils.AddRoundResultsStr(req, "550 350 420 390 410 380 400 390 390 400 380 390") // R1: P0,P2,P4,P6 win
	pairtestutils.AddRoundResultsStr(req, "540 360 425 385 415 375 405 385 395 395 385 385") // R2: same
	pairtestutils.AddRoundResultsStr(req, "530 370 430 380 420 370 410 380 400 390 390 380") // R3: same
	pairtestutils.AddRoundResultsStr(req, "520 380 435 375 425 365 415 375 405 385 395 375") // R4: same
	pairtestutils.AddRoundResultsStr(req, "510 390 380 440 360 430 360 420 400 380 380 390") // R5: P2,P4,P6 LOSE
	pairtestutils.AddRoundResultsStr(req, "500 400 375 445 355 435 355 425 395 375 375 395") // R6: P2,P4,P6 LOSE

	resp := cop.COPPair(req)
	writeScenarioLog(t, "crowded_second.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// Scenario 4: Spread is the only differentiator; multiple players tied on wins with 2 rounds left.
// After 6 rounds: P0=5-1 (+400 spread), P2=5-1 (+50 spread), P4=5-1 (+20 spread), PlacePrizes=1.
func TestControlLoss_SpreadTiebreaker(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping spread-tiebreaker scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"Alice", "Bob", "Charlie", "Dave", "Eric", "Frank", "Grace", "Holly"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               200,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.02,
		AllPlayers:                 8,
		ValidPlayers:               8,
		Rounds:                     8,
		PlacePrizes:                1,
		DivisionSims:               1000,
		ControlLossSims:            1000,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
	}
	// Pairings: (0v1),(2v3),(4v5),(6v7)
	// P0 wins by 80 each round (5/6), P2 wins by 10 each round (5/6), P4 wins by 5 each round (5/6)
	pairtestutils.AddNDummyRounds(req, 6)
	pairtestutils.AddRoundResultsStr(req, "480 400 410 400 405 400 390 400") // R1: P0+80,P2+10,P4+5,P6-10
	pairtestutils.AddRoundResultsStr(req, "480 400 410 400 405 400 395 395") // R2: P6 ties→higher
	pairtestutils.AddRoundResultsStr(req, "480 400 410 400 405 400 390 400") // R3
	pairtestutils.AddRoundResultsStr(req, "480 400 410 400 405 400 390 400") // R4
	pairtestutils.AddRoundResultsStr(req, "480 400 410 400 405 400 390 400") // R5
	pairtestutils.AddRoundResultsStr(req, "390 480 399 410 399 406 400 380") // R6: P0,P2,P4 LOSE

	resp := cop.COPPair(req)
	writeScenarioLog(t, "spread_tiebreaker.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// Scenario 5: Multiple cash prizes with many contenders, 2 rounds left.
// 16 players, PlacePrizes=3: P0=6-0, P2=P4=5-1, P6=P8=P10=4-2.
func TestControlLoss_MultiplePrizes(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping multiple-prizes scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9", "P10", "P11", "P12", "P13", "P14", "P15"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               200,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.02,
		AllPlayers:                 16,
		ValidPlayers:               16,
		Rounds:                     8,
		PlacePrizes:                3,
		DivisionSims:               1000,
		ControlLossSims:            1000,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
	}
	// Pairings: (0v1),(2v3),(4v5),(6v7),(8v9),(10v11),(12v13),(14v15)
	// P0=6-0, P2=P4=5-1, P6=P8=P10=4-2, P12=3-3, P14=2-4
	pairtestutils.AddNDummyRounds(req, 6)
	pairtestutils.AddRoundResultsStr(req, "540 350 470 390 460 380 440 390 420 390 410 390 400 390 390 400") // R1
	pairtestutils.AddRoundResultsStr(req, "530 360 465 395 455 385 435 395 415 395 405 395 395 395 385 405") // R2
	pairtestutils.AddRoundResultsStr(req, "520 370 460 400 450 390 430 400 410 400 400 400 390 400 380 410") // R3
	pairtestutils.AddRoundResultsStr(req, "510 380 455 405 445 395 425 405 405 405 395 405 385 405 375 415") // R4
	pairtestutils.AddRoundResultsStr(req, "500 390 440 420 430 410 400 420 390 420 380 420 380 400 370 400") // R5: P2,P4 lose
	pairtestutils.AddRoundResultsStr(req, "490 400 430 430 420 420 390 430 380 430 370 430 375 395 365 395") // R6: P6,P8,P10 lose

	resp := cop.COPPair(req)
	writeScenarioLog(t, "multiple_prizes.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// Scenario 6: Near-Gibson situation; leader has spread just under the threshold.
// After 6 rounds: P0=6-0 with +350 spread (near Gibson with GibsonSpread=200, 2 rounds left → max catchable=400).
func TestControlLoss_NearGibson(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping near-gibson scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"Alice", "Bob", "Charlie", "Dave", "Eric", "Frank", "Grace", "Holly"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               200,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.02,
		AllPlayers:                 8,
		ValidPlayers:               8,
		Rounds:                     8,
		PlacePrizes:                2,
		DivisionSims:               1000,
		ControlLossSims:            1000,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
	}
	// P0 wins all 6 by ~58 each (total +350). P2=5-1, P4=4-2.
	pairtestutils.AddNDummyRounds(req, 6)
	pairtestutils.AddRoundResultsStr(req, "458 400 430 390 400 390 395 385") // R1: P0+58, P2+40
	pairtestutils.AddRoundResultsStr(req, "456 400 425 395 395 395 390 390") // R2
	pairtestutils.AddRoundResultsStr(req, "454 400 420 400 390 400 385 395") // R3: P2 wins barely, P4 loses
	pairtestutils.AddRoundResultsStr(req, "452 400 415 405 385 405 380 400") // R4: P2 loses
	pairtestutils.AddRoundResultsStr(req, "450 400 410 410 380 410 395 385") // R5: P2 wins, P4 loses
	pairtestutils.AddRoundResultsStr(req, "448 400 440 380 375 415 390 390") // R6: P2 wins big

	resp := cop.COPPair(req)
	writeScenarioLog(t, "near_gibson.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// Albany CSW ME 2025: simulate the last 16 rounds using COP, starting from after round 16.
// Uses CreateBLSRound32PairRequest which is the same tournament data.
// Run with: COP_SCENARIOS=1 go test -run TestAlbanyCSW2025ME_Last16Rounds
func TestAlbanyCSW2025ME_Last16Rounds(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping Albany CSW 2025 ME scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	rng := rand.New(rand.NewSource(42))
	spreadsDist := standings.GetScoreDifferences()
	spreadsDistSize := len(spreadsDist)

	// Build base request from BLS data (same tournament), trimmed to 16 rounds.
	base := pairtestutils.CreateBLSRound32PairRequest()
	numPlayers := int(base.AllPlayers)

	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                base.PlayerNames,
		PlayerClasses:              base.PlayerClasses,
		ClassPrizes:                base.ClassPrizes,
		GibsonSpread:               base.GibsonSpread,
		ControlLossThreshold:       base.ControlLossThreshold,
		HopefulnessThreshold:       base.HopefulnessThreshold,
		AllPlayers:                 base.AllPlayers,
		ValidPlayers:               base.ValidPlayers,
		Rounds:                     base.Rounds,
		PlacePrizes:                base.PlacePrizes,
		DivisionSims:               1000,
		ControlLossSims:            1000,
		ControlLossActivationRound: base.ControlLossActivationRound,
		AllowRepeatByes:            base.AllowRepeatByes,
		RemovedPlayers:             base.RemovedPlayers,
		Seed:                       0,
		DivisionPairings:           base.DivisionPairings[:16],
		DivisionResults:            base.DivisionResults[:16],
	}

	// Simulate rounds 17-32 with COP pairing and random results.
	for round := 17; round <= 32; round++ {
		resp := cop.COPPair(req)
		is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
		fmt.Printf("Albany CSW 2025 ME round %d pairings: %v\n", round, resp.Pairings)
		writeScenarioLog(t, fmt.Sprintf("albany_csw_2025_me_round_%02d.log", round), resp.Log)

		pairings := make([]int32, numPlayers)
		copy(pairings, resp.Pairings)
		req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{Pairings: pairings})

		// Generate random results for each player.
		results := make([]int32, numPlayers)
		byePlayer := -1
		for i := 0; i < numPlayers; i++ {
			if pairings[i] == int32(i) {
				byePlayer = i
			}
		}
		for i := 0; i < numPlayers; i++ {
			if i == byePlayer {
				results[i] = 50
				continue
			}
			opp := int(pairings[i])
			if i < opp {
				spread := int32(spreadsDist[rng.Intn(spreadsDistSize)])
				base := int32(400)
				results[i] = base + spread/2
				results[opp] = base - spread/2
			}
		}
		req.DivisionResults = append(req.DivisionResults, &pb.RoundResults{Results: results})
	}
}

// Fake 28-game 51-player event: 3 rounds of fontes-style pairings, then 25 rounds of COP.
// Run with: COP_SCENARIOS=1 go test -run TestFake28Game51Players
func TestFake28Game51Players(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping fake 51-player scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	rng := rand.New(rand.NewSource(99))
	spreadsDist := standings.GetScoreDifferences()
	spreadsDistSize := len(spreadsDist)

	numPlayers := 51
	totalRounds := 28
	fontesRounds := 3

	names := make([]string, numPlayers)
	classes := make([]int32, numPlayers)
	for i := 0; i < numPlayers; i++ {
		names[i] = fmt.Sprintf("Player%02d", i)
	}

	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                names,
		PlayerClasses:              classes,
		ClassPrizes:                []int32{2},
		GibsonSpread:               200,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.02,
		AllPlayers:                 int32(numPlayers),
		ValidPlayers:               int32(numPlayers),
		Rounds:                     int32(totalRounds),
		PlacePrizes:                3,
		DivisionSims:               1000,
		ControlLossSims:            1000,
		ControlLossActivationRound: 22,
		AllowRepeatByes:            false,
	}

	// Generate random results for a given pairing.
	addRandomResults := func(pairings []int32) {
		results := make([]int32, numPlayers)
		byePlayer := -1
		for i := 0; i < numPlayers; i++ {
			if pairings[i] == int32(i) {
				byePlayer = i
			}
		}
		for i := 0; i < numPlayers; i++ {
			if i == byePlayer {
				results[i] = 50
				continue
			}
			opp := int(pairings[i])
			if i < opp {
				spread := int32(spreadsDist[rng.Intn(spreadsDistSize)])
				baseScore := int32(400)
				results[i] = baseScore + spread/2
				results[opp] = baseScore - spread/2
			}
		}
		req.DivisionResults = append(req.DivisionResults, &pb.RoundResults{Results: results})
	}

	// Fontes-style pairings for first 3 rounds: group players into N/F tiers,
	// pair top-half vs bottom-half within each group. Use simple rotation for variety.
	for r := 0; r < fontesRounds; r++ {
		pairings := make([]int32, numPlayers)
		paired := make([]bool, numPlayers)

		// Each fontes round pairs player i with player (i + numPlayers/2 + r) % numPlayers.
		step := numPlayers/2 + r
		for i := 0; i < numPlayers; i++ {
			if paired[i] {
				continue
			}
			j := (i + step) % numPlayers
			startJ := j
			// Avoid pairing already-paired or self.
			// Break if we've wrapped all the way around (no valid partner — will get bye).
			for paired[j] || j == i {
				j = (j + 1) % numPlayers
				if j == startJ {
					j = -1
					break
				}
			}
			if j < 0 {
				continue
			}
			pairings[i] = int32(j)
			pairings[j] = int32(i)
			paired[i] = true
			paired[j] = true
		}
		// The one remaining unpaired player (51 is odd) gets a bye.
		for i := 0; i < numPlayers; i++ {
			if !paired[i] {
				pairings[i] = int32(i)
				paired[i] = true
			}
		}

		req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{Pairings: pairings})
		addRandomResults(pairings)
	}

	// COP rounds for the remaining 25 rounds.
	for round := fontesRounds + 1; round <= totalRounds; round++ {
		resp := cop.COPPair(req)
		is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
		fmt.Printf("Fake 51-player round %d pairings: %v\n", round, resp.Pairings)
		writeScenarioLog(t, fmt.Sprintf("fake_28game_51players_round_%02d.log", round), resp.Log)

		pairings := make([]int32, numPlayers)
		copy(pairings, resp.Pairings)
		req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{Pairings: pairings})
		addRandomResults(pairings)
	}
}
