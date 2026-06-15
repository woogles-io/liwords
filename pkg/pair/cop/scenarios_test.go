package cop_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/pair/cop"
	"github.com/woogles-io/liwords/pkg/pair/standings"
	pairtestutils "github.com/woogles-io/liwords/pkg/pair/testutils"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"golang.org/x/exp/rand"
)

// All tests require COP_SCENARIOS=1 to run (on-demand only).
// All scenarios use AddNDummyRounds which fixes pairings as (0v1),(2v3),(4v5),...
// so each pair always plays each other. Results strings determine who wins each round.
// All scenarios have 2 rounds left (6 rounds completed, Rounds=8).

const (
	scenarioDivisionSims          = 100000
	scenarioControlLossSims       = 10000
	scenarioHopefulness           = 0.01
	scenarioGibsonSpread          = 200
	scenarioLastRoundGibsonSpread = 250
)

func writeScenarioLog(t *testing.T, filename string, log string) {
	t.Helper()
	path := filepath.Join("logs", filename)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Logf("failed to create logs directory: %v", err)
		return
	}
	if err := os.WriteFile(path, []byte(log), 0644); err != nil {
		t.Logf("failed to write log file %s: %v", path, err)
	}
}

// Scenario 1: Five players tied at N wins with PlacePrizes=4.
// Top 4 are secure on control; COP should pair 1st vs 5th.
// 10 players, P0=P2=P4=P6=P8=5-1 with decreasing spreads (+300,+200,+150,+100,+50).
func TestScenario1_FiveAtTopFourPrizes(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 10,
		ValidPlayers:               10,
		Rounds:                     8,
		PlacePrizes:                4,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
		Seed:                       1,
	}
	// R1-R5: P0+64, P2+44, P4+34, P6+24, P8+14 (even players win)
	// R6: all odd players win by 20
	// Result: P0=5-1 +300, P2=5-1 +200, P4=5-1 +150, P6=5-1 +100, P8=5-1 +50
	pairtestutils.AddNDummyRounds(req, 6)
	pairtestutils.AddRoundResultsStr(req, "432 368 422 378 417 383 412 388 407 393")
	pairtestutils.AddRoundResultsStr(req, "432 368 422 378 417 383 412 388 407 393")
	pairtestutils.AddRoundResultsStr(req, "432 368 422 378 417 383 412 388 407 393")
	pairtestutils.AddRoundResultsStr(req, "432 368 422 378 417 383 412 388 407 393")
	pairtestutils.AddRoundResultsStr(req, "432 368 422 378 417 383 412 388 407 393")
	pairtestutils.AddRoundResultsStr(req, "390 410 390 410 390 410 390 410 390 410")

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_1_five_at_top_four_prizes.log", resp.Log)
	fmt.Println(resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// Scenario 2: Two players at N wins, two at N-1 wins, close spreads.
// COP should not force any pairing because 3rd/4th can win by winning out.
// P0=5-1 +100, P2=5-1 +80, P4=4-2 +50, P6=4-2 +40, PlacePrizes=2.
func TestScenario2_TwoNTwoNMinus1CloseSpread(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 8,
		ValidPlayers:               8,
		Rounds:                     8,
		PlacePrizes:                2,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
	}
	// R1-R4: all even players win (P0+30, P2+26, P4+25, P6+20)
	// R5: P0,P2 win; P5,P7 beat P4,P6 (P4,P6 first losses)
	// R6: P1,P3 beat P0,P2; P5,P7 beat P4,P6 (P0,P2 first losses, P4,P6 second losses)
	// Result: P0=5-1 +100, P2=5-1 +80, P4=4-2 +50, P6=4-2 +40
	pairtestutils.AddNDummyRounds(req, 6)
	pairtestutils.AddRoundResultsStr(req, "430 400 426 400 425 400 420 400")
	pairtestutils.AddRoundResultsStr(req, "430 400 426 400 425 400 420 400")
	pairtestutils.AddRoundResultsStr(req, "430 400 426 400 425 400 420 400")
	pairtestutils.AddRoundResultsStr(req, "430 400 426 400 425 400 420 400")
	pairtestutils.AddRoundResultsStr(req, "430 400 426 400 400 425 400 420")
	pairtestutils.AddRoundResultsStr(req, "400 450 400 450 400 425 400 420")

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_2_two_n_two_n_minus_1_close_spread.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// Scenario 3: Two players at N wins, two at N-1 wins, far inferior spreads.
// The N-1 players need both more wins AND a spread comeback; 1% hopefulness threshold
// determines whether they are locked out. 2nd's destiny control is not threatened,
// so 1v2 is only forced if everyone else is locked out.
// P0=5-1 +500, P2=5-1 +400, P4=4-2 -100, P6=4-2 -150, PlacePrizes=2.
func TestScenario3_TwoNTwoNMinus1FarSpread(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 8,
		ValidPlayers:               8,
		Rounds:                     8,
		PlacePrizes:                2,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
	}
	// R1-R4: P0+110, P2+90, P4+25, P6+25
	// R5: P0,P2 still win; P5 beats P4 by 100, P7 beats P6 by 125
	// R6: P1,P3 beat P0,P2; P5 beats P4 by 100, P7 beats P6 by 125
	// Result: P0=5-1 +500, P2=5-1 +400, P4=4-2 -100, P6=4-2 -150
	pairtestutils.AddNDummyRounds(req, 6)
	pairtestutils.AddRoundResultsStr(req, "510 400 490 400 425 400 425 400")
	pairtestutils.AddRoundResultsStr(req, "510 400 490 400 425 400 425 400")
	pairtestutils.AddRoundResultsStr(req, "510 400 490 400 425 400 425 400")
	pairtestutils.AddRoundResultsStr(req, "510 400 490 400 425 400 425 400")
	pairtestutils.AddRoundResultsStr(req, "510 400 490 400 300 400 275 400")
	pairtestutils.AddRoundResultsStr(req, "400 450 400 450 300 400 275 400")

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_3_two_n_two_n_minus_1_far_spread.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// Scenario 4: One player at N, one at N-1, one at N-2; 2nd's spread is close to 1st.
// Because 2nd can still win the tournament by winning out, COP should default to 1v3.
// P0=5-1 +100, P2=4-2 +90, P4=3-3 ~0, PlacePrizes=1.
func TestScenario4_OneNOneNMinus1OneNMinus2CloseSpread(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 8,
		ValidPlayers:               8,
		Rounds:                     8,
		PlacePrizes:                1,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
	}
	// P0: wins R1-R5 by 25, loses R6 by 25 → 5-1 +100
	// P2: wins R1-R4 by 30, loses R5-R6 by 15 → 4-2 +90
	// P4: wins R1,R3,R5 by 20, loses R2,R4,R6 by 20 → 3-3 ±0
	// P6: wins R1,R3,R5 by 15, loses R2,R4,R6 by 15 → 3-3 ±0 (avoids filler players at 4-2)
	pairtestutils.AddNDummyRounds(req, 6)
	pairtestutils.AddRoundResultsStr(req, "425 400 430 400 420 400 415 400")
	pairtestutils.AddRoundResultsStr(req, "425 400 430 400 380 400 385 400")
	pairtestutils.AddRoundResultsStr(req, "425 400 430 400 420 400 415 400")
	pairtestutils.AddRoundResultsStr(req, "425 400 430 400 380 400 385 400")
	pairtestutils.AddRoundResultsStr(req, "425 400 385 400 420 400 415 400")
	pairtestutils.AddRoundResultsStr(req, "400 425 385 400 380 400 385 400")

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_4_one_n_one_n_minus_1_one_n_minus_2_close_spread.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// Scenario 5: Two players at N wins, two at N-2 wins.
// With KOTH in the last round, 3rd and 4th cannot win; COP should recognize this.
// P0=5-1 +300, P2=5-1 +200, P4=3-3 ~0, P6=3-3 ~-48, PlacePrizes=2.
func TestScenario5_TwoNTwoNMinus2KOTH(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 8,
		ValidPlayers:               8,
		Rounds:                     8,
		PlacePrizes:                2,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
	}
	// P0: wins R1-R5 by 70, loses R6 by 50 → 5-1 +300
	// P2: wins R1-R5 by 50, loses R6 by 50 → 5-1 +200
	// P4: alternates win(+20)/loss(-20) → 3-3 ±0
	// P6: alternates win(+10)/loss(-26) → 3-3 ~-48
	pairtestutils.AddNDummyRounds(req, 6)
	pairtestutils.AddRoundResultsStr(req, "470 400 450 400 420 400 410 400")
	pairtestutils.AddRoundResultsStr(req, "470 400 450 400 380 400 374 400")
	pairtestutils.AddRoundResultsStr(req, "470 400 450 400 420 400 410 400")
	pairtestutils.AddRoundResultsStr(req, "470 400 450 400 380 400 374 400")
	pairtestutils.AddRoundResultsStr(req, "470 400 450 400 420 400 410 400")
	pairtestutils.AddRoundResultsStr(req, "400 450 400 450 380 400 374 400")

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_5_two_n_two_n_minus_2_koth.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// Scenario 6: One player at N wins, two at N-2 wins.
// COP should find control loss for both N-2 players and pair 1st vs 2nd.
// P0=5-1 +200, P2=3-3 +51, P4=3-3 ~0, PlacePrizes=2.
func TestScenario6_OneNTwoNMinus2BothControlLoss(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 8,
		ValidPlayers:               8,
		Rounds:                     8,
		PlacePrizes:                2,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
	}
	// P0: wins R1-R5 by 50, loses R6 by 50 → 5-1 +200
	// P2: wins R1-R3 by 40, loses R4-R6 by 23 → 3-3 +51
	// P4: wins R1,R3,R5 by 20, loses R2,R4,R6 by 20 → 3-3 ±0
	// P6: wins R1,R3,R5 by 20, loses R2,R4,R6 by 20 → 3-3 ±0 (avoids filler player at 4-2)
	pairtestutils.AddNDummyRounds(req, 6)
	pairtestutils.AddRoundResultsStr(req, "450 400 440 400 420 400 420 400")
	pairtestutils.AddRoundResultsStr(req, "450 400 440 400 380 400 380 400")
	pairtestutils.AddRoundResultsStr(req, "450 400 440 400 420 400 420 400")
	pairtestutils.AddRoundResultsStr(req, "450 400 400 423 380 400 380 400")
	pairtestutils.AddRoundResultsStr(req, "450 400 400 423 420 400 420 400")
	pairtestutils.AddRoundResultsStr(req, "400 450 400 423 380 400 380 400")

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_6_one_n_two_n_minus_2_both_control_loss.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// Albany CSW ME 2025: show what COP would have paired for rounds 17-32, given the actual
// historical results for all prior rounds. Each round uses only real data.
// Run with: COP_SCENARIOS=1 go test -run TestAlbanyCSW2025ME_Last16Rounds
func TestScenarioMultiRound_AlbanyCSW2025ME_Last16Rounds(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping Albany CSW 2025 ME scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)

	base := pairtestutils.CreateBLSRound32PairRequest()

	for round := 17; round <= 32; round++ {
		gibsonSpread := int32(scenarioGibsonSpread)
		if round == int(base.Rounds) {
			gibsonSpread = scenarioLastRoundGibsonSpread
		}
		req := &pb.PairRequest{
			PairMethod:                 pb.PairMethod_COP,
			PlayerNames:                base.PlayerNames,
			PlayerClasses:              base.PlayerClasses,
			ClassPrizes:                base.ClassPrizes,
			GibsonSpread:               gibsonSpread,
			ControlLossThreshold:       base.ControlLossThreshold,
			HopefulnessThreshold:       scenarioHopefulness,
			AllPlayers:                 base.AllPlayers,
			ValidPlayers:               base.ValidPlayers,
			Rounds:                     base.Rounds,
			PlacePrizes:                base.PlacePrizes,
			DivisionSims:               scenarioDivisionSims,
			ControlLossSims:            scenarioControlLossSims,
			ControlLossActivationRound: base.ControlLossActivationRound,
			AllowRepeatByes:            base.AllowRepeatByes,
			RemovedPlayers:             base.RemovedPlayers,
			Seed:                       0,
			DivisionPairings:           base.DivisionPairings[:round-1],
			DivisionResults:            base.DivisionResults[:round-1],
		}

		resp := cop.COPPair(req)
		is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
		fmt.Printf("Albany CSW 2025 ME round %d pairings: %v\n", round, resp.Pairings)
		writeScenarioLog(t, fmt.Sprintf("albany_csw_2025_me_round_%02d.log", round), resp.Log)
	}
}

// July 4th 2026 28-game 53-player event: 3 rounds of fontes-style pairings, then 25 rounds of COP.
// Uses the Division 1 player list from wordgameplayers.org/tournaments/1162.
// Run with: COP_SCENARIOS=1 go test -run TestScenarioMultiRound_July4th2026
func TestScenarioMultiRound_July4th2026(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping July 4th 2026 scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	spreadsDist := standings.GetScoreDifferences()
	spreadsDistSize := len(spreadsDist)

	const numRuns = 10

	numPlayers := 53
	totalRounds := 28
	fontesRounds := 3

	names := []string{
		"Wellington Jighere", "Nigel Richards", "Will Anderson", "Dave Wiegand",
		"Adam Logan", "Josh Sokol", "Eta Karo", "Matthew Tunnicliffe",
		"Joshua Castellano", "Enoch Nwali", "Matthew O'Connor", "Rob Robinsky",
		"Austin Shin", "Kevin Fraley", "Noah Slatkoff", "Jason Keller",
		"Thomas Reinke", "Edgar Odongkara", "Sammy Okosagah", "Charles Reinke",
		"Lukeman Owolabi", "Brian Po", "Olawale Fashina", "Chukwudi Ehibudu",
		"Rasheed Balogun", "Samuel Anikoh", "Chris Lipe", "Robert Linn",
		"Jason Carney", "Joel Wapnick", "Jared Robinson", "Laurie Cohen",
		"Anthony Ikolo", "Oshevire Avwenagha", "Amit Chakrabarti", "Mohammad Sulaiman",
		"Scott Jackson", "Marlon Hill", "Akeem Adekunle", "Dipo Akanbi",
		"Jason Ubeika", "Niel Gan", "Bharath Balakrishnan", "Femi Awowade",
		"Mark Francillon", "Daniel Blake", "Osikhena Ojior", "Greg Harper",
		"Zachary Dang", "Collins Okafor", "Tijan Jeng", "Ayotunde Adeyeri",
		"Fidelis Olotu",
	}
	classes := make([]int32, numPlayers)

	for run := 0; run < numRuns; run++ {
		seed := time.Now().UnixNano()
		rng := rand.New(rand.NewSource(uint64(seed)))
		runDir := fmt.Sprintf("july4th2026_run_%02d", run+1)

		req := &pb.PairRequest{
			PairMethod:                 pb.PairMethod_COP,
			PlayerNames:                names,
			PlayerClasses:              classes,
			ClassPrizes:                []int32{2},
			GibsonSpread:               scenarioGibsonSpread,
			ControlLossThreshold:       0.30,
			HopefulnessThreshold:       scenarioHopefulness,
			AllPlayers:                 int32(numPlayers),
			ValidPlayers:               int32(numPlayers),
			Rounds:                     int32(totalRounds),
			PlacePrizes:                10,
			DivisionSims:               scenarioDivisionSims,
			ControlLossSims:            scenarioControlLossSims,
			ControlLossActivationRound: 22,
			AllowRepeatByes:            false,
			Seed:                       seed,
		}

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
					if rng.Intn(2) == 0 {
						results[i] = baseScore + spread/2
						results[opp] = baseScore - spread/2
					} else {
						results[i] = baseScore - spread/2
						results[opp] = baseScore + spread/2
					}
				}
			}
			req.DivisionResults = append(req.DivisionResults, &pb.RoundResults{Results: results})
		}

		// Fontes-style pairings for first 3 rounds.
		for r := 0; r < fontesRounds; r++ {
			pairings := make([]int32, numPlayers)
			paired := make([]bool, numPlayers)

			step := numPlayers/2 + r
			for i := 0; i < numPlayers; i++ {
				if paired[i] {
					continue
				}
				j := (i + step) % numPlayers
				startJ := j
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
			if round == totalRounds {
				req.GibsonSpread = scenarioLastRoundGibsonSpread
			} else {
				req.GibsonSpread = scenarioGibsonSpread
			}

			resp := cop.COPPair(req)
			is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
			fmt.Printf("July 4th 2026 run %d round %d pairings: %v\n", run+1, round, resp.Pairings)
			writeScenarioLog(t, fmt.Sprintf("%s/round_%02d.log", runDir, round), resp.Log)

			pairings := make([]int32, numPlayers)
			copy(pairings, resp.Pairings)
			req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{Pairings: pairings})
			addRandomResults(pairings)
		}
	}
}

// Manhattan Open 2024 (or similar): 18 players, 16 rounds, PlacePrizes=2.
// Pairs the final round (round 16) using historical data through round 15.
func TestScenario_Manhattan(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := pairtestutils.CreateManhattanAfterRound14PairRequest()
	req.DivisionSims = scenarioDivisionSims
	req.ControlLossSims = scenarioControlLossSims
	req.HopefulnessThreshold = scenarioHopefulness
	req.GibsonSpread = scenarioGibsonSpread
	req.ControlLossActivationRound = 10
	req.Seed = 1

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_manhattan.log", resp.Log)
	fmt.Println(resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}
