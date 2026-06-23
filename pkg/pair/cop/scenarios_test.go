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
	scenarioHopefulness           = 0.02
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

// Scenario 7: Factor-3 expansion in the 2nd-to-last round.
// 1st has N wins, 2nd and 3rd have N-1 wins, 4th/5th/6th have N-2 wins, all with
// close spreads so the factor-3 condition triggers and COP tries to force 1v4, 2v5, 3v6.
// 12 players, 8 rounds, 6 completed, PlacePrizes=3.
// P0=5-1 +100, P2=4-2 +50, P4=4-2 +30, P6=3-3 +10, P8=3-3 +5, P10=3-3 +0.
func TestScenario7_Factor3ExpansionSecondToLastRound(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9", "P10", "P11"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 12,
		ValidPlayers:               12,
		Rounds:                     8,
		PlacePrizes:                3,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
		Seed:                       1,
	}
	// AddNDummyRounds pairs: P0vP1, P2vP3, P4vP5, P6vP7, P8vP9, P10vP11
	// Round 1-5: P0,P2,P4,P6,P8,P10 win (even players win)
	//   P0 wins by 25: R1=425/400, R2=425/400, R3=425/400, R4=425/400, R5=425/400
	//   P2 wins by 15: R1=415/400, ... (same pattern)
	//   P4 wins by 10: R1=410/400, ...
	//   P6 wins by  5: R1=405/400, ...
	//   P8 wins by  3: R1=403/400, ...
	//   P10 wins by 1: R1=401/400, ...
	// Round 6: odd players win — P0,P2,P4 each lose to P1,P3,P5; P6,P8,P10 also lose
	//   P0 loses: 400/425 → -25; P2 loses: 400/415; P4 loses: 400/410
	//   P6 loses: 400/405; P8 loses: 400/403; P10 loses: 400/401
	// After 6 rounds:
	//   P0: 5-1, spread = 5*25 - 25 = 100
	//   P2: 5-1 would need more losses... let's use explicit results.
	// Instead use explicit score strings:
	//   R1-R5 (even wins):
	//     P0+25, P2+15, P4+10, P6+5, P8+3, P10+1 per round
	//   R6 (odd wins):
	//     all even players lose by same amount
	// Spreads after 6 rounds: P0=5*25-25=100, P2=5*15-15=60... let me adjust
	// R1-R5: P0 wins by 20, P2 wins by 12, P4 wins by 8, P6 wins by 4, P8 wins by 2, P10 wins by 1
	// R6: P0 loses by 20, P2 loses by 12, P4 loses by 8, P6 loses by 4, P8 loses by 2, P10 loses by 1
	// Spreads: P0=4*20=80, P2=4*12=48, P4=4*8=32, P6=4*4=16, P8=4*2=8, P10=4*1=4
	// All within 500 of P0 → factor-3 check triggers.
	pairtestutils.AddNDummyRounds(req, 6)
	// R1
	pairtestutils.AddRoundResultsStr(req, "420 400 412 400 408 400 404 400 402 400 401 400")
	// R2
	pairtestutils.AddRoundResultsStr(req, "420 400 412 400 408 400 404 400 402 400 401 400")
	// R3
	pairtestutils.AddRoundResultsStr(req, "420 400 412 400 408 400 404 400 402 400 401 400")
	// R4
	pairtestutils.AddRoundResultsStr(req, "420 400 412 400 408 400 404 400 402 400 401 400")
	// R5
	pairtestutils.AddRoundResultsStr(req, "420 400 412 400 408 400 404 400 402 400 401 400")
	// R6: odd players win
	pairtestutils.AddRoundResultsStr(req, "400 420 400 412 400 408 400 404 400 402 400 401")

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_7_factor3_expansion.log", resp.Log)
	fmt.Println(resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// Scenario 8: Factor-3 control loss for 2nd or 3rd in the 2nd-to-last round.
// 1st has N wins, 2nd and 3rd have N-1 wins, 4th/5th/6th have N-2 wins.
// 2nd and 3rd are very close in spread to each other and to 1st (tiny margins).
// 4th has the most spread of any player in the tournament, which would make
// factor-3 (1v4) dangerous for 1st. The new control loss check evaluates whether
// 2nd or 3rd loses control under factor-3 pairings vs playing 1st directly.
// 12 players, 8 rounds, 6 completed, PlacePrizes=3.
// P0=5-1 +5, P2=4-2 +4, P4=4-2 +3, P6=3-3 +300, P8=3-3 +15, P10=3-3 ~-6.
func TestScenario8_Factor3ControlLoss2ndOr3rd(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9", "P10", "P11"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 12,
		ValidPlayers:               12,
		Rounds:                     8,
		PlacePrizes:                3,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
		Seed:                       1,
	}
	// AddNDummyRounds pairs: P0vP1, P2vP3, P4vP5, P6vP7, P8vP9, P10vP11
	// After 6 rounds:
	//   P0: 5-1, spread = 5*20 - 95 = +5
	//   P2: 4-2, spread = 4*15 - 26 - 30 = +4
	//   P4: 4-2, spread = 4*14 - 27 - 26 = +3
	//   P6: 3-3, spread = 3*110 - 3*10 = +300 (most spread in tournament)
	//   P8: 3-3, spread = 3*22 - 3*17 = +15
	//   P10: 3-3, spread = 3*20 - 3*22 = -6
	pairtestutils.AddNDummyRounds(req, 6)
	// R1: all even players win
	pairtestutils.AddRoundResultsStr(req, "420 400 415 400 414 400 510 400 422 400 420 400")
	// R2: P0,P2,P4 win; P6,P8,P10 lose
	pairtestutils.AddRoundResultsStr(req, "420 400 415 400 414 400 400 410 400 417 400 422")
	// R3: same as R1
	pairtestutils.AddRoundResultsStr(req, "420 400 415 400 414 400 510 400 422 400 420 400")
	// R4: same as R2
	pairtestutils.AddRoundResultsStr(req, "420 400 415 400 414 400 400 410 400 417 400 422")
	// R5: P0 wins; P2,P4 lose (first losses); P6,P8,P10 win
	pairtestutils.AddRoundResultsStr(req, "420 400 400 426 400 427 510 400 422 400 420 400")
	// R6: all even players lose
	pairtestutils.AddRoundResultsStr(req, "400 495 400 430 400 426 400 410 400 417 400 422")

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_8_factor3_control_loss.log", resp.Log)
	fmt.Println(resp.Log)
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

// july4thOneBehindRounds lists the 1-indexed round numbers where pairings are based on
// results from 1 game behind (i.e. all results available). All other rounds use 2 games
// behind — the director pairs the next round before the current round finishes.
var july4thOneBehindRounds = map[int]bool{
	1: true, 5: true, 9: true, 13: true, 17: true, 21: true, 25: true, 26: true, 27: true, 28: true,
}

// makeRandomResults generates a random result for every game in pairings.
func makeRandomResults(pairings []int32, numPlayers int, rng *rand.Rand, spreadsDist []uint64) *pb.RoundResults {
	results := make([]int32, numPlayers)
	spreadsDistSize := len(spreadsDist)
	for i := 0; i < numPlayers; i++ {
		if pairings[i] == int32(i) {
			results[i] = 50 // bye
			continue
		}
		opp := int(pairings[i])
		if i < opp {
			spread := int32(spreadsDist[rng.Intn(spreadsDistSize)])
			base := int32(400)
			if rng.Intn(2) == 0 {
				results[i] = base + spread/2
				results[opp] = base - spread/2
			} else {
				results[i] = base - spread/2
				results[opp] = base + spread/2
			}
		}
	}
	return &pb.RoundResults{Results: results}
}

// generateFontesPairings returns deterministic Fontes-style pairings for round r (0-indexed).
func generateFontesPairings(r int, numPlayers int) []int32 {
	pairings := make([]int32, numPlayers)
	paired := make([]bool, numPlayers)
	step := numPlayers/2 + r
	for i := 0; i < numPlayers; i++ {
		if paired[i] {
			continue
		}
		j := (i + step) % numPlayers
		startJ := j
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
	return pairings
}

// numResultsForRound returns how many past results to include when pairing the given
// 1-indexed round, honouring the 1-behind / 2-behind timing rule.
func numResultsForRound(roundNum int, available int) int {
	behind := 2
	if july4thOneBehindRounds[roundNum] {
		behind = 1
	}
	n := roundNum - behind
	if n < 0 {
		n = 0
	}
	if n > available {
		n = available
	}
	return n
}

// July 4th 2026 28-game 53-player event: 3 rounds of fontes-style pairings, then 25 rounds of COP.
// Uses the Division 1 player list from wordgameplayers.org/tournaments/1162.
// Pairings simulate real-tournament timing: most rounds are paired 2 games behind
// (before previous round finishes); rounds 1,5,9,13,17,21,25-28 use 1-game-behind results.
// Run with: COP_SCENARIOS=1 go test -run TestScenarioMultiRound_July4th2026
func TestScenarioMultiRound_July4th2026(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping July 4th 2026 scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	spreadsDist := standings.GetScoreDifferences()

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

		allPairings := []*pb.RoundPairings{}
		allResults := []*pb.RoundResults{}

		// Fontes-style pairings for the first 3 rounds.
		for r := 0; r < fontesRounds; r++ {
			pairings := generateFontesPairings(r, numPlayers)
			allPairings = append(allPairings, &pb.RoundPairings{Pairings: pairings})
			allResults = append(allResults, makeRandomResults(pairings, numPlayers, rng, spreadsDist))
		}

		// COP rounds for rounds 4–28, with real-tournament timing.
		for round := fontesRounds + 1; round <= totalRounds; round++ {
			numRes := numResultsForRound(round, len(allResults))

			req.DivisionPairings = allPairings
			req.DivisionResults = allResults[:numRes]

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
			allPairings = append(allPairings, &pb.RoundPairings{Pairings: pairings})
			allResults = append(allResults, makeRandomResults(pairings, numPlayers, rng, spreadsDist))
		}
	}
}

// July 4th 2026 WOW division: top half of the 35-player WOW field (18 players) from
// wordgameplayers.org/tournaments/1161, using the same 28-round structure and real-tournament
// timing as the main event (see TestScenarioMultiRound_July4th2026).
// Run with: COP_SCENARIOS=1 go test -run TestScenarioMultiRound_July4th2026WOW
func TestScenarioMultiRound_July4th2026WOW(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping July 4th 2026 WOW scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	spreadsDist := standings.GetScoreDifferences()

	const numRuns = 10

	// Top 18 of 35 players (even number = top half rounded up) from tournament 1161.
	numPlayers := 18
	totalRounds := 28
	fontesRounds := 3

	names := []string{
		"Orry Swift", "Karl Higby", "Michael Thelen", "Evan Chester",
		"Sam Towne", "Michael Fagen", "Jeffrey Nelson", "John Scalzo",
		"Brian McCarthy", "Joel Horn", "Beth Mix", "Samuel Heiman",
		"Joshua Pepper", "Judy Horn", "Letitia Sears", "Nick Purifoy",
		"Mary Krizan", "Sally Scalzo",
	}
	classes := make([]int32, numPlayers)

	for run := 0; run < numRuns; run++ {
		seed := time.Now().UnixNano()
		rng := rand.New(rand.NewSource(uint64(seed)))
		runDir := fmt.Sprintf("july4th2026wow_run_%02d", run+1)

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
			PlacePrizes:                4,
			DivisionSims:               scenarioDivisionSims,
			ControlLossSims:            scenarioControlLossSims,
			ControlLossActivationRound: 22,
			AllowRepeatByes:            false,
			Seed:                       seed,
		}

		allPairings := []*pb.RoundPairings{}
		allResults := []*pb.RoundResults{}

		for r := 0; r < fontesRounds; r++ {
			pairings := generateFontesPairings(r, numPlayers)
			allPairings = append(allPairings, &pb.RoundPairings{Pairings: pairings})
			allResults = append(allResults, makeRandomResults(pairings, numPlayers, rng, spreadsDist))
		}

		for round := fontesRounds + 1; round <= totalRounds; round++ {
			numRes := numResultsForRound(round, len(allResults))

			req.DivisionPairings = allPairings
			req.DivisionResults = allResults[:numRes]

			if round == totalRounds {
				req.GibsonSpread = scenarioLastRoundGibsonSpread
			} else {
				req.GibsonSpread = scenarioGibsonSpread
			}

			resp := cop.COPPair(req)
			is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
			fmt.Printf("July 4th 2026 WOW run %d round %d pairings: %v\n", run+1, round, resp.Pairings)
			writeScenarioLog(t, fmt.Sprintf("%s/round_%02d.log", runDir, round), resp.Log)

			pairings := make([]int32, numPlayers)
			copy(pairings, resp.Pairings)
			allPairings = append(allPairings, &pb.RoundPairings{Pairings: pairings})
			allResults = append(allResults, makeRandomResults(pairings, numPlayers, rng, spreadsDist))
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
