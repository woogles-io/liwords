package cop_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	scenarioControlLossSims       = 20000
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
		ControlLossThreshold:       0.30,
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

// Scenarios 9-16: the hopeful-to-cash contender-group parity fixes, exercised
// as single-round tests in the penultimate round (round 7 of 8) of an 8-round,
// 17-player, PlacePrizes=3 tournament. All eight combine:
//   - contender group odd or even (does GetPrecompData's parity promotion fire?)
//   - with or without TopDownByes
//   - 1st Gibsonized or not
//
// Shared setup: P0-P3 form a "strong" quartet (contenders/leader), P4-P7 are
// their weak-quartet opponents, and P8-P16 (9 players) are pure filler who
// self-bye (lose) every one of the 6 completed rounds - always well below
// cash contention and, crucially, already fully "used up" for repeat-bye
// purposes. On top of that, exactly 3 of the strong quartet - whichever 3
// end up ranked 1st/2nd/3rd in the final standings - each get one of their
// 6 rounds converted into a bye too (with the same win/loss outcome they'd
// already have had that round, and that round's opponent also byes). So by
// round 7, the top 3 in the standings each already have exactly one prior
// bye, while everyone else in the strong/weak quartets has zero. That means
// TopDownByes' fewest-byes/top-rank tiebreak must skip the top 3 (who no
// longer have the fewest byes) and land on the next-fewest-byes contender
// instead - exactly the scenario the parity fixes need to handle correctly.
// Without TopDownByes, PC's penalty against byeing a hopeful-to-cash
// contender still generally pushes the bye onto a clearly-eliminated player.
func contenderParityBaseRequest(topDownByes bool) *pb.PairRequest {
	names := make([]string, 17)
	classes := make([]int32, 17)
	for i := range names {
		names[i] = fmt.Sprintf("P%d", i)
	}
	return &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                names,
		PlayerClasses:              classes,
		ClassPrizes:                []int32{2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 17,
		ValidPlayers:               17,
		Rounds:                     8,
		PlacePrizes:                3,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
		TopDownByes:                topDownByes,
		Seed:                       1,
	}
}

// addContenderParityRoundsNotGibsonEven: P0,P2,P3 finish 6-0 (tied, so
// nobody is Gibsonized) and are the top 3, each with a prior bye (rounds
// 0,2,1 respectively). P1 and P5 both finish 3-3, still hopeful-to-cash - 4
// total contenders (even), no promotion needed.
func addContenderParityRoundsNotGibsonEven(req *pb.PairRequest) {
	pairtestutils.AddRoundPairingsStr(req, "0 5 6 7 4 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "100 500 500 420 -50 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 3 0 1 2 7 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 500 500 100 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 390 380 420 400 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 2 7 0 1 6 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 390 -50 420 400 400 100 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 390 380 420 400 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 390 380 420 400 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
}

// Scenario 9: even contender group, not Gibsonized, without TopDownByes.
// Without TDB, round 7's bye lands on eliminated P1, outside the group.
func TestScenario9_ContenderParityEvenNotGibsonizedNoTDB(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := contenderParityBaseRequest(false)
	addContenderParityRoundsNotGibsonEven(req)

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_9_contender_parity_even_notgibson_notdb.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(!strings.Contains(resp.Log, "Odd number of non-Gibsonized hopeful-to-cash players"))
	is.True(!strings.Contains(resp.Log, "Contender group parity"))
}

// Scenario 10: even contender group, not Gibsonized, with TopDownByes.
// With TDB, P0,P2,P3 (top 3) already have a bye, so the fewest-byes
// tiebreak instead lands on P5 - still a genuine contender - flipping the
// 4-player even group odd and requiring the boundary to extend by one
// (promoting P1).
func TestScenario10_ContenderParityEvenNotGibsonizedWithTDB(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := contenderParityBaseRequest(true)
	addContenderParityRoundsNotGibsonEven(req)

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_10_contender_parity_even_notgibson_tdb.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(!strings.Contains(resp.Log, "Odd number of non-Gibsonized hopeful-to-cash players"))
	is.True(strings.Contains(resp.Log, "Contender group parity"))
	is.True(strings.Contains(resp.Log, "extending the boundary"))
}

// addContenderParityRoundsNotGibsonOdd: P0,P2,P3 finish 6-0 (tied, not
// Gibsonized) and are the top 3, each with a prior bye (rounds 0,1,2
// respectively). P1 finishes 3-3, still hopeful-to-cash - 3 total
// contenders (odd), so GetPrecompData promotes P5 to even the group at 4.
func addContenderParityRoundsNotGibsonOdd(req *pb.PairRequest) {
	pairtestutils.AddRoundPairingsStr(req, "0 5 6 7 4 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "100 500 500 420 -50 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 2 7 0 1 6 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 500 100 420 400 400 -50 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 3 0 1 2 7 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 500 500 100 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 380 500 420 400 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 380 500 420 400 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 380 500 420 400 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
}

// Scenario 11: odd contender group, not Gibsonized, without TopDownByes.
// Without TDB, round 7's bye lands on P5 - outside the (now 4-player)
// group - so no further adjustment beyond the precomp promotion.
func TestScenario11_ContenderParityOddNotGibsonizedNoTDB(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := contenderParityBaseRequest(false)
	addContenderParityRoundsNotGibsonOdd(req)

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_11_contender_parity_odd_notgibson_notdb.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "Odd number of non-Gibsonized hopeful-to-cash players (3)"))
	is.True(!strings.Contains(resp.Log, "Contender group parity"))
}

// Scenario 12: odd contender group, not Gibsonized, with TopDownByes.
// With TDB, P0,P2,P3 (top 3) already have a bye, so the fewest-byes
// tiebreak lands on P1 - which is itself the player GetPrecompData just
// promoted to fix parity. Removing P1 from the pairing pool undoes the need
// for that promotion, but since P1 *is* the promoted player (not a genuine
// contender ranked above it), the fix extends the boundary further
// (promoting P5) rather than retracting.
func TestScenario12_ContenderParityOddNotGibsonizedWithTDB(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := contenderParityBaseRequest(true)
	addContenderParityRoundsNotGibsonOdd(req)

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_12_contender_parity_odd_notgibson_tdb.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "Odd number of non-Gibsonized hopeful-to-cash players (3)"))
	is.True(strings.Contains(resp.Log, "Contender group parity"))
	is.True(strings.Contains(resp.Log, "extending the boundary"))
}

// addContenderParityRoundsGibsonEven: P0 finishes 6-0 with a big spread
// lead, Gibsonized; P1,P3 (each capped at 3 wins, safely below P0's floor)
// are the next 2 highest-ranked and are the top 3 alongside P0, each with a
// prior bye (rounds 0,2,4 respectively). P2 and their P5-P7 mirrors are
// also still hopeful-to-cash - 6 total non-Gibsonized contenders (even), no
// promotion needed.
func addContenderParityRoundsGibsonEven(req *pb.PairRequest) {
	pairtestutils.AddRoundPairingsStr(req, "0 5 6 7 4 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "100 500 500 390 -50 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 500 500 390 400 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 1 6 7 0 5 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 100 500 390 400 -50 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 390 380 550 400 400 400 350 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 3 0 1 2 7 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 390 380 100 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 390 380 550 400 400 400 350 -50 -50 -50 -50 -50 -50 -50 -50 -50")
}

// Scenario 13: even contender group, 1st Gibsonized, without TopDownByes.
// Without TDB, the bye still lands on the Gibsonized P0 via the GB (Gibson
// bye) policy, which doesn't consider bye history the way TDB does. Since
// P0 was already excluded from the parity-relevant count, no adjustment
// fires.
func TestScenario13_ContenderParityEvenGibsonizedNoTDB(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := contenderParityBaseRequest(false)
	addContenderParityRoundsGibsonEven(req)

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_13_contender_parity_even_gibson_notdb.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "P0   | 6.0  | 600  | Yes"))
	is.True(!strings.Contains(resp.Log, "Odd number of non-Gibsonized hopeful-to-cash players"))
	is.True(!strings.Contains(resp.Log, "Contender group parity"))
}

// Scenario 14: even contender group, 1st Gibsonized, with TopDownByes.
// With TDB, P0,P1,P3 (top 3) already have a bye, so the fewest-byes
// tiebreak instead lands on P2 - a genuine, non-Gibsonized contender -
// flipping the 6-player even (Gibson-aware) group odd and requiring the
// boundary to extend by one.
func TestScenario14_ContenderParityEvenGibsonizedWithTDB(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := contenderParityBaseRequest(true)
	addContenderParityRoundsGibsonEven(req)

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_14_contender_parity_even_gibson_tdb.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "P0   | 6.0  | 600  | Yes"))
	is.True(!strings.Contains(resp.Log, "Odd number of non-Gibsonized hopeful-to-cash players"))
	is.True(strings.Contains(resp.Log, "Contender group parity"))
	is.True(strings.Contains(resp.Log, "extending the boundary"))
}

// addContenderParityRoundsGibsonOdd: P0 finishes 6-0, Gibsonized; P1,P2
// (each capped at 3 wins) are the next 2 highest-ranked and are the top 3
// alongside P0, each with a prior bye (rounds 0,1,2 respectively). P3 and
// their P4-P7 mirrors are also still hopeful-to-cash - 7 total
// non-Gibsonized contenders (odd), so GetPrecompData promotes P16 to even
// the group at 8.
func addContenderParityRoundsGibsonOdd(req *pb.PairRequest) {
	pairtestutils.AddRoundPairingsStr(req, "0 5 6 7 4 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "100 400 400 400 -50 405 405 425 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "5 1 7 4 3 0 6 2 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "460 100 440 430 400 400 -50 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "6 7 2 5 4 3 0 1 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "460 450 100 430 -50 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "7 4 5 6 1 2 3 0 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "460 400 400 400 405 405 425 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "460 450 440 430 400 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
	pairtestutils.AddRoundPairingsStr(req, "5 6 7 4 3 0 1 2 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "460 400 400 400 425 400 405 405 -50 -50 -50 -50 -50 -50 -50 -50 -50")
}

// Scenario 15: odd contender group, 1st Gibsonized, without TopDownByes.
// Without TDB, the bye still lands on the Gibsonized P0 via GB, which needs
// no further adjustment since P0 was already excluded from the count.
func TestScenario15_ContenderParityOddGibsonizedNoTDB(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := contenderParityBaseRequest(false)
	addContenderParityRoundsGibsonOdd(req)

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_15_contender_parity_odd_gibson_notdb.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "P0   | 6.0  | 400  | Yes"))
	is.True(strings.Contains(resp.Log, "Odd number of non-Gibsonized hopeful-to-cash players (7)"))
	is.True(!strings.Contains(resp.Log, "Contender group parity"))
}

// Scenario 16: odd contender group, 1st Gibsonized, with TopDownByes.
// With TDB, P0,P1,P2 (top 3) already have a bye, so the fewest-byes
// tiebreak instead lands on P3 - a genuine, non-Gibsonized contender ranked
// above the precomp-promoted P16 - which already restores parity on the
// raw (pre-promotion) count. This is the double-promotion bug: the fix
// retracts P16's promotion instead of extending past it.
func TestScenario16_ContenderParityOddGibsonizedWithTDB(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := contenderParityBaseRequest(true)
	addContenderParityRoundsGibsonOdd(req)

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_16_contender_parity_odd_gibson_tdb.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "P0   | 6.0  | 400  | Yes"))
	is.True(strings.Contains(resp.Log, "Odd number of non-Gibsonized hopeful-to-cash players (7)"))
	is.True(strings.Contains(resp.Log, "Contender group parity"))
	is.True(strings.Contains(resp.Log, "retracting the earlier promotion"))
}

// addContenderParityRoundsGibsonEvenFinalRound extends
// addContenderParityRoundsGibsonEven with one more round (round 7 of 8, same
// pattern as its round 4/6: the strong quartet keeps beating the weak
// quartet, filler keeps self-byeing), leaving exactly 1 round remaining -
// the final round - for Scenario 17.
func addContenderParityRoundsGibsonEvenFinalRound(req *pb.PairRequest) {
	addContenderParityRoundsGibsonEven(req)
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 390 380 550 400 400 400 350 -50 -50 -50 -50 -50 -50 -50 -50 -50")
}

// Scenario 17: 1st is Gibsonized going into the final round, and
// TopDownByes assigns the bye to a genuine hopeful-to-cash contender rather
// than to the Gibsonized leader (whom GB would otherwise force it onto) or
// to an eliminated player. Builds on the Scenario 14 setup, extended by one
// more round - P0 is Gibsonized for 1st (7-0), and by the final round P0,
// P1, and P3 all have a prior bye, so TDB's fewest-byes/top-rank tiebreak
// skips past all three of them and lands on P6, who is still hopeful to
// cash. "Gibson Gets Bye: true" in the log confirms GB would otherwise have
// forced the bye onto the Gibsonized P0; TB takes precedence instead.
func TestScenario17_GibsonFinalRoundByeGoesToContender(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := contenderParityBaseRequest(true)
	addContenderParityRoundsGibsonEvenFinalRound(req)

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_17_gibson_final_round_bye_to_contender.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "Rounds Remaining: 1"))
	// P0 is Gibsonized for 1st, and GB would otherwise force the bye onto them.
	is.True(strings.Contains(resp.Log, "P0   | 7.0  | 700  | Yes"))
	is.True(strings.Contains(resp.Log, "Gibson Gets Bye: true"))
	// P6 - not Gibsonized, still hopeful to cash - receives the bye instead of P0.
	is.Equal(resp.Pairings[6], int32(6))
}

// addContenderParityRoundsGibsonEvenFinalRoundKeepingParity extends
// addContenderParityRoundsGibsonEven with a different round 7 than
// addContenderParityRoundsGibsonEvenFinalRound (Scenario 17's): P7, who
// would otherwise fall out of contention with only 1 round left (see
// Scenario 17's log), instead wins its round 7 game, keeping all 6
// non-Gibsonized players from addContenderParityRoundsGibsonEven's raw
// contender group naturally hopeful to cash with the final round still
// ahead - reproducing Scenario 14's "even, no promotion needed" shape one
// round later, for Scenario 18.
func addContenderParityRoundsGibsonEvenFinalRoundKeepingParity(req *pb.PairRequest) {
	addContenderParityRoundsGibsonEven(req)
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "500 390 380 370 400 400 400 430 -50 -50 -50 -50 -50 -50 -50 -50 -50")
}

// addContenderParityRoundsGibsonOddFinalRound extends
// addContenderParityRoundsGibsonOdd with one more round (round 7 of 8, same
// pattern as round 5: the strong quartet keeps beating the weak quartet,
// filler keeps self-byeing), leaving exactly 1 round remaining - the final
// round - for Scenario 19.
func addContenderParityRoundsGibsonOddFinalRound(req *pb.PairRequest) {
	addContenderParityRoundsGibsonOdd(req)
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3 8 9 10 11 12 13 14 15 16")
	pairtestutils.AddRoundResultsStr(req, "460 450 440 430 400 400 400 400 -50 -50 -50 -50 -50 -50 -50 -50 -50")
}

// Scenario 18: same setup as Scenario 14 (even contender group, 1st
// Gibsonized, with TopDownByes), but with only the final round remaining
// instead of 2 - via addContenderParityRoundsGibsonEvenFinalRoundKeepingParity
// rather than Scenario 17's addContenderParityRoundsGibsonEvenFinalRound, so
// the raw contender group stays a naturally-hopeful, no-promotion-needed 6
// (Scenario 14's shape) instead of shrinking to Scenario 17's 5-plus-promotion.
// adjustLowestPossibleHopeCasherForBye is read unconditionally regardless of
// round count (see its doc comment), so the same boundary-extension fix from
// Scenario 14 must still fire here, one round later, even though CC's
// fourth-quarter hopeful-grouping rule itself is disabled in the final round.
func TestScenario18_ContenderParityEvenGibsonizedWithTDBFinalRound(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := contenderParityBaseRequest(true)
	addContenderParityRoundsGibsonEvenFinalRoundKeepingParity(req)

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_18_contender_parity_even_gibson_tdb_final_round.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "Rounds Remaining: 1"))
	is.True(strings.Contains(resp.Log, "P0   | 7.0  | 700  | Yes"))
	is.True(!strings.Contains(resp.Log, "Odd number of non-Gibsonized hopeful-to-cash players"))
	is.True(strings.Contains(resp.Log, "Contender group parity"))
	is.True(strings.Contains(resp.Log, "extending the boundary"))
}

// Scenario 19: same setup as Scenario 16 (odd contender group, 1st
// Gibsonized, with TopDownByes - the double-promotion bug's retraction
// fix), extended by one more round so only the final round remains instead
// of 2. With only 1 round left, fewer players remain naturally hopeful to
// cash than in Scenario 16 (3 vs. 7 - a real, expected effect of having one
// fewer round to close the gap), but the raw count is still odd, so the same
// promotion-then-retraction shape from Scenario 16 still fires here too.
func TestScenario19_ContenderParityOddGibsonizedWithTDBFinalRound(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := contenderParityBaseRequest(true)
	addContenderParityRoundsGibsonOddFinalRound(req)

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_19_contender_parity_odd_gibson_tdb_final_round.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "Rounds Remaining: 1"))
	is.True(strings.Contains(resp.Log, "P0   | 7.0  | 460  | Yes"))
	// With only 1 round left (vs. 2 in Scenario 16), fewer players remain
	// naturally hopeful to cash - here 3, down from Scenario 16's 7 - but the
	// group is still odd, so the same promotion/retraction shape still fires.
	is.True(strings.Contains(resp.Log, "Odd number of non-Gibsonized hopeful-to-cash players (3)"))
	is.True(strings.Contains(resp.Log, "Contender group parity"))
	is.True(strings.Contains(resp.Log, "retracting the earlier promotion"))
}

// Scenario 20: control loss forces 1st vs 3rd, while the bye lands on 4th -
// a clearly non-contending player well behind 3rd - via TopDownByes' fewest-
// byes tiebreak. 1st, 2nd, and 3rd (P0, P1, P2) are close in wins/spread and
// each already have one prior bye; 4th (P3) has none, so TDB picks P3 for
// the bye this round even though it's not the CL destiny-child. Checks that
// CL's forced-pairing-with-1st logic and TB's bye assignment coexist without
// conflict when the two land on different players.
func TestScenario20_ControlLossWithByeOnFarBehindFourth(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6"},
		PlayerClasses:              make([]int32, 7),
		ClassPrizes:                []int32{2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.05,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 7,
		ValidPlayers:               7,
		Rounds:                     8,
		PlacePrizes:                2,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 4,
		AllowRepeatByes:            false,
		TopDownByes:                true,
		Seed:                       1,
	}
	// Round 1: P0 byes (win). P1 beats P4, P2 beats P5, P6 beats P3.
	// (P0, P1, P2 each get exactly one prior bye across rounds 1-3, so
	// TopDownByes' fewest-byes tiebreak skips them this round; P3 never
	// byes, so it's the pick.)
	pairtestutils.AddRoundPairingsStr(req, "0 4 5 6 1 2 3")
	pairtestutils.AddRoundResultsStr(req, "50 455 420 400 400 400 415")
	// Round 2: P1 byes (win). P0 beats P4, P2 beats P6, P5 beats P3.
	pairtestutils.AddRoundPairingsStr(req, "4 1 6 5 0 3 2")
	pairtestutils.AddRoundResultsStr(req, "460 50 430 400 400 415 400")
	// Round 3: P2 byes (win). P0 beats P5, P1 beats P6, P4 beats P3.
	pairtestutils.AddRoundPairingsStr(req, "5 6 2 4 3 0 1")
	pairtestutils.AddRoundResultsStr(req, "460 420 50 400 406 400 400")
	// Round 4: P4 byes (loss). P0 beats P6, P1 beats P5, P3 beats P2.
	pairtestutils.AddRoundPairingsStr(req, "6 5 3 2 4 1 0")
	pairtestutils.AddRoundResultsStr(req, "460 420 400 420 -50 400 400")

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_20_control_loss_bye_on_far_behind_fourth.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// P0, P1, P2 are 1st-3rd, close together; P3 is a clear step behind them
	// (2 fewer wins, ~100pt worse spread than P2).
	is.True(strings.Contains(resp.Log, "P0   | 4.0  | 230"))
	is.True(strings.Contains(resp.Log, "P1   | 4.0  | 145"))
	is.True(strings.Contains(resp.Log, "P2   | 3.0  | 80"))
	is.True(strings.Contains(resp.Log, "P3   | 1.0  | -16"))
	// Control loss flags P2 (0-indexed rank 2, "3rd") - not P1 ("2nd") - as
	// the destiny child, so CL forces 1st (P0) vs 3rd (P2) instead of 1st vs 2nd.
	is.True(strings.Contains(resp.Log, "Destinys Child: P2"))
	is.Equal(resp.Pairings[0], int32(2))
	is.Equal(resp.Pairings[2], int32(0))
	// P3 (4th, well behind 3rd) receives the bye via TopDownByes, even though
	// it isn't the destiny child CL is protecting.
	is.Equal(resp.Pairings[3], int32(3))
}

// Scenario 21: class prize KOTH with no bye involved. 30 players in 3
// classes (P0-P1 unclassed cash leaders, P2-P5 class B, P6-P7 class C,
// P8-P29 unclassed filler), only 2 overall cash prizes, and 2 class prizes
// each for classes B and C, with 1 round remaining. P0/P1 each have 3 wins
// (they beat a fresh filler every round); P2, P3, and P4 each have exactly
// 1 win (P2 over P7, P3 over P5, P4 over P5 again) and otherwise only ties,
// putting all 3 in a tied "1 win" tier that's a full win behind P0/P1 (so
// they can't threaten the 2 cash places) but clearly ahead of every filler
// (who nets 0 or -1). The class KOTH rule force-pairs P2 and P3 - class B's
// top 2, tied on wins - for the last class B prize; P4 (also tied on wins,
// but 3rd in class rank via spread) and both class C players are far enough
// behind their neighbors that nobody else gets pulled in.
func TestScenario21_ClassPrizeKOTH(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod: pb.PairMethod_COP,
		PlayerNames: []string{
			"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9",
			"P10", "P11", "P12", "P13", "P14", "P15", "P16", "P17", "P18", "P19",
			"P20", "P21", "P22", "P23", "P24", "P25", "P26", "P27", "P28", "P29",
		},
		PlayerClasses: []int32{
			0, 0, 1, 1, 1, 1, 2, 2, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		},
		ClassPrizes:                []int32{2, 2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 30,
		ValidPlayers:               30,
		Rounds:                     4,
		PlacePrizes:                2,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 2,
		AllowRepeatByes:            false,
		Seed:                       1,
	}
	// Round 1: P0 beats P8, P1 beats P9 (both +600); P2 beats P7 (+100), P3
	// beats P5 (+90) - these double as P7's and P5's 1st losses; P4 and P6
	// tie fillers; everyone else ties.
	pairtestutils.AddRoundPairingsStr(req,
		"8 9 7 5 16 3 17 2 0 1 11 10 13 12 15 14 4 6 19 18 21 20 23 22 25 24 27 26 29 28")
	pairtestutils.AddRoundResultsStr(req,
		"1000 1000 500 490 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400")
	// Round 2: P0 beats P10, P1 beats P11 (both +600); P4 beats P5 (+80,
	// P5's 2nd loss) and P6 beats P7 (+70, P7's 2nd loss) - putting P5 and
	// P7 3 wins behind their class neighbors (P4 and P6), well past the
	// 1-round-remaining catch-up threshold regardless of spread; P2 and P3
	// tie fillers; everyone else ties.
	pairtestutils.AddRoundPairingsStr(req,
		"10 11 14 15 5 4 7 6 9 8 0 1 13 12 2 3 17 16 19 18 21 20 23 22 25 24 27 26 29 28")
	pairtestutils.AddRoundResultsStr(req,
		"1000 1000 400 400 480 400 470 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400")
	// Round 3: P0 beats P12, P1 beats P13 (both +600, reaching 3 wins - a
	// full win ahead of P2/P3/P4's 1 win, so none of them can catch the 2
	// cash places); everyone else ties.
	pairtestutils.AddRoundPairingsStr(req,
		"12 13 3 2 5 4 7 6 9 8 11 10 0 1 15 14 17 16 19 18 21 20 23 22 25 24 27 26 29 28")
	pairtestutils.AddRoundResultsStr(req,
		"1000 1000 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400")

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_21_class_prize_koth.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Cash prize KOTH: P0 and P1 (the only 2 hopeful for the 2 cash places)
	// play each other.
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	// Class B prize KOTH: P2 and P3 - class B's top 2, tied on wins - are
	// force-paired for the last of class B's 2 prizes.
	is.Equal(resp.Pairings[2], int32(3))
	is.Equal(resp.Pairings[3], int32(2))
}

// Scenario 22: same setup as Scenario 21, except P2 (class B's top player)
// receives a top-down bye this round - engineered by giving P0 and P1 a
// prior bye each (round 1) while P2 (and everyone else still playing) has
// none. The class prize KOTH rule must now work around P2's bye: instead of
// dropping the class B forced pairing entirely (the pre-fix behavior), it
// should skip P2 and force P3 vs P4 - the top 2 and 3 remaining class B
// players - together instead.
func TestScenario22_ClassPrizeKOTHWorksAroundTopDownBye(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod: pb.PairMethod_COP,
		PlayerNames: []string{
			"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9",
			"P10", "P11", "P12", "P13", "P14", "P15", "P16", "P17", "P18", "P19",
			"P20", "P21", "P22", "P23", "P24", "P25", "P26", "P27", "P28", "P29",
		},
		PlayerClasses: []int32{
			0, 0, 1, 1, 1, 1, 2, 2, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		},
		ClassPrizes:                []int32{2, 2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 30,
		ValidPlayers:               29,
		RemovedPlayers:             []int32{29},
		TopDownByes:                true,
		Rounds:                     4,
		PlacePrizes:                2,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 2,
		AllowRepeatByes:            false,
		Seed:                       1,
	}
	// Round 1: identical to Scenario 21's round 1, except P0 and P1 take
	// byes (worth the same +600 as their round-1 win there) instead of
	// playing P8/P9, who tie each other instead. This gives P0 and P1 each
	// one prior bye, while P2 (and everyone else still playing) has none;
	// TopDownByes' fewest-byes scan will pick P2 for this round's bye over
	// P0/P1 as a result.
	pairtestutils.AddRoundPairingsStr(req,
		"0 1 7 5 16 3 17 2 9 8 11 10 13 12 15 14 4 6 19 18 21 20 23 22 25 24 27 26 29 28")
	pairtestutils.AddRoundResultsStr(req,
		"600 600 500 490 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400")
	// Rounds 2-3: identical to Scenario 21's rounds 2-3.
	pairtestutils.AddRoundPairingsStr(req,
		"10 11 14 15 5 4 7 6 9 8 0 1 13 12 2 3 17 16 19 18 21 20 23 22 25 24 27 26 29 28")
	pairtestutils.AddRoundResultsStr(req,
		"1000 1000 400 400 480 400 470 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400")
	pairtestutils.AddRoundPairingsStr(req,
		"12 13 3 2 5 4 7 6 9 8 11 10 0 1 15 14 17 16 19 18 21 20 23 22 25 24 27 26 29 28")
	pairtestutils.AddRoundResultsStr(req,
		"1000 1000 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400")

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_22_class_prize_koth_with_bye.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Cash prize KOTH: P0 and P1 still play each other.
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	// P2 receives this round's top-down bye instead of playing for the
	// class B prize.
	is.Equal(resp.Pairings[2], int32(2))
	// Class B prize KOTH works around P2's bye: P3 and P4 - the top 2 and 3
	// remaining class B players - are force-paired instead of P2 and P3.
	is.Equal(resp.Pairings[3], int32(4))
	is.Equal(resp.Pairings[4], int32(3))
}

// Scenario 23: same setup as Scenario 21/22, except the top-down bye goes
// to P3 - class B's *second*-highest player, not its top player - engineered
// by giving P0, P1, and P2 each a prior bye (round 1) while P3 has none.
// (A 4th, neutral-scored bye for P29 keeps round 1's player count even,
// since 3 real byes alone would leave an odd number of players to pair up.)
// P7's round-1 loss, previously delivered by P2, is rerouted through a
// low-spread filler (P18) so it doesn't depend on P2 being available to
// play. The class prize KOTH rule must skip P3 (not P2 this time) and force
// P2 vs P4 - class B's top and third-ranked players - together instead.
func TestScenario23_ClassPrizeKOTHWorksAroundBothTopDownByeAndSecondRankedByePlayer(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod: pb.PairMethod_COP,
		PlayerNames: []string{
			"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9",
			"P10", "P11", "P12", "P13", "P14", "P15", "P16", "P17", "P18", "P19",
			"P20", "P21", "P22", "P23", "P24", "P25", "P26", "P27", "P28", "P29",
		},
		PlayerClasses: []int32{
			0, 0, 1, 1, 1, 1, 2, 2, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		},
		ClassPrizes:                []int32{2, 2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 30,
		ValidPlayers:               29,
		RemovedPlayers:             []int32{29},
		TopDownByes:                true,
		Rounds:                     4,
		PlacePrizes:                2,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 2,
		AllowRepeatByes:            false,
		Seed:                       1,
	}
	// Round 1: P0, P1, and P2 take byes (P2's worth +100, matching its
	// round-1 win in Scenario 21/22); P29 also takes a neutral (0-score, so
	// no-op) bye purely to keep the round's player count even. P3 still
	// beats P5 (+90), unaffected. P18 beats P7 (+30) in place of P2's usual
	// win over P7 - still enough of a gap (P7 nets 2 full losses across
	// rounds 1-2, same as Scenario 21/22) to keep P7 well out of catching
	// P6, its class C neighbor. Everyone else ties.
	pairtestutils.AddRoundPairingsStr(req,
		"0 1 2 5 16 3 17 18 9 8 11 10 13 12 15 14 4 6 7 28 21 20 23 22 25 24 27 26 19 29")
	pairtestutils.AddRoundResultsStr(req,
		"600 600 100 490 400 400 400 400 400 400 400 400 400 400 400 400 400 400 430 400 400 400 400 400 400 400 400 400 400 0")
	// Rounds 2-3: identical to Scenario 21/22's rounds 2-3.
	pairtestutils.AddRoundPairingsStr(req,
		"10 11 14 15 5 4 7 6 9 8 0 1 13 12 2 3 17 16 19 18 21 20 23 22 25 24 27 26 29 28")
	pairtestutils.AddRoundResultsStr(req,
		"1000 1000 400 400 480 400 470 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400")
	pairtestutils.AddRoundPairingsStr(req,
		"12 13 3 2 5 4 7 6 9 8 11 10 0 1 15 14 17 16 19 18 21 20 23 22 25 24 27 26 29 28")
	pairtestutils.AddRoundResultsStr(req,
		"1000 1000 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400 400")

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_23_class_prize_koth_second_ranked_bye.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Cash prize KOTH: P0 and P1 still play each other.
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	// P3 - class B's 2nd-highest player - receives this round's top-down
	// bye, ahead of P4 and every filler, since P0, P1, and P2 already have
	// one each.
	is.Equal(resp.Pairings[3], int32(3))
	// Class B prize KOTH works around P3's bye: P2 and P4 - class B's top
	// and 3rd-ranked players - are force-paired instead of P2 and P3.
	is.Equal(resp.Pairings[2], int32(4))
	is.Equal(resp.Pairings[4], int32(2))
}

// Scenario 24: control loss's destiny child would be the exact same player
// TopDownByes assigns this round's bye to - the real-world conflict fixed by
// ComputeTopDownByeRankIdx (see cop.go's computeTopDownByePlayer and
// copdata.go's control-loss search). Identical standings to Scenario 20 (P0,
// P1, P2 close together, P3 a clear step behind), except the bye history is
// rearranged so P2 - not P3 - has zero prior byes, making P2 the TopDownByes
// pick this round. Without skipping P2 in the destiny-child search, CL would
// bar the leader from everyone except P2, while TB simultaneously bars the
// leader from playing P2 (who's on bye instead) - leaving the leader with no
// legal opponent at all (OVERCONSTRAINED, as in the July 4th 2026 random-start
// run 11 round 26 log). With the fix, the search skips P2 and finds no other
// candidate close enough to flag (P3 is too far behind to be hopeful for
// 1st), so control loss simply doesn't fire this round.
func TestScenario24_ControlLossDestinyChildIsTopDownByeRecipient(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6"},
		PlayerClasses:              make([]int32, 7),
		ClassPrizes:                []int32{2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.05,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 7,
		ValidPlayers:               7,
		Rounds:                     8,
		PlacePrizes:                2,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 4,
		AllowRepeatByes:            false,
		TopDownByes:                true,
		Seed:                       1,
	}
	// Round 1: P0 byes (win). P1 beats P4, P2 beats P5, P6 beats P3.
	pairtestutils.AddRoundPairingsStr(req, "0 4 5 6 1 2 3")
	pairtestutils.AddRoundResultsStr(req, "50 455 420 400 400 400 415")
	// Round 2: P1 byes (win). P0 beats P4, P2 beats P6, P5 beats P3.
	pairtestutils.AddRoundPairingsStr(req, "4 1 6 5 0 3 2")
	pairtestutils.AddRoundResultsStr(req, "460 50 430 400 400 415 400")
	// Round 3: P3 byes (loss, -6) instead of P2 (who plays P4 for the same
	// +50 margin P2's Scenario-20 bye would have given it) - the only change
	// from Scenario 20's setup, so P2 ends up with zero prior byes and P3
	// has one, while every player's final win/spread record is unchanged.
	pairtestutils.AddRoundPairingsStr(req, "5 6 4 3 2 0 1")
	pairtestutils.AddRoundResultsStr(req, "460 420 450 -6 400 400 400")
	// Round 4: P4 byes (loss). P0 beats P6, P1 beats P5, P3 beats P2.
	pairtestutils.AddRoundPairingsStr(req, "6 5 3 2 4 1 0")
	pairtestutils.AddRoundResultsStr(req, "460 420 400 420 -50 400 400")

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_24_control_loss_destiny_child_is_tdb_recipient.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Same final standings as Scenario 20.
	is.True(strings.Contains(resp.Log, "P0   | 4.0  | 230"))
	is.True(strings.Contains(resp.Log, "P1   | 4.0  | 145"))
	is.True(strings.Contains(resp.Log, "P2   | 3.0  | 80"))
	is.True(strings.Contains(resp.Log, "P3   | 1.0  | -16"))
	// The destiny-child search excludes P2 (it's getting this round's bye).
	is.True(strings.Contains(resp.Log, "Control loss: excluding rank 3 (P2)"))
	// P3 isn't a genuine contender, so with P2 excluded, control loss finds
	// nobody to flag.
	is.True(strings.Contains(resp.Log, "Destinys Child: (none)"))
	// P2 (not P3, unlike Scenario 20) receives the bye via TopDownByes.
	is.Equal(resp.Pairings[2], int32(2))
}

// Scenario 25: top-down byes take precedence over Factor 3 the same way they
// take precedence over control loss (Scenario 24) - the real-world conflict
// fixed in computeFactor3ForcedPairings. Same setup as Scenario 7 (12 players
// close enough in wins/spread to trigger Factor 3's 1v4/2v5/3v6 expansion in
// the 2nd-to-last round), plus a 13th filler player who absorbs every
// historical bye, leaving P0-P11 tied at zero prior byes - so TopDownByes'
// tiebreak (ties go to the higher-ranked player) picks P0, the leader and
// top of the Factor 3 group itself. Without the fix, Factor 3 would force P0
// vs P3 while TopDownByes simultaneously forces P0 vs BYE - a direct
// conflict leaving P0 (and, via the same unpaired-parity cascade as
// Scenario 24's real-world case, others) with no legal pairing at all
// (OVERCONSTRAINED, as in the July 4th 2026 random-start run 13 round 27
// log). With the fix, Factor 3 is canceled outright this round instead.
func TestScenario25_TopDownByeTakesPrecedenceOverFactor3(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9", "P10", "P11", "P12"},
		PlayerClasses:              make([]int32, 13),
		ClassPrizes:                []int32{2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 13,
		ValidPlayers:               13,
		Rounds:                     8,
		PlacePrizes:                3,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
		TopDownByes:                true,
		Seed:                       1,
	}
	// Same 6 rounds as Scenario 7 (AddNDummyRounds pairs P0vP1, P2vP3, P4vP5,
	// P6vP7, P8vP9, P10vP11; P12, the 13th/odd player, self-byes every round
	// with a losing score, so it never threatens the standings and absorbs
	// all 6 rounds' worth of byes, leaving P0-P11 with none).
	pairtestutils.AddNDummyRounds(req, 6)
	pairtestutils.AddRoundResultsStr(req, "420 400 412 400 408 400 404 400 402 400 401 400 -50")
	pairtestutils.AddRoundResultsStr(req, "420 400 412 400 408 400 404 400 402 400 401 400 -50")
	pairtestutils.AddRoundResultsStr(req, "420 400 412 400 408 400 404 400 402 400 401 400 -50")
	pairtestutils.AddRoundResultsStr(req, "420 400 412 400 408 400 404 400 402 400 401 400 -50")
	pairtestutils.AddRoundResultsStr(req, "420 400 412 400 408 400 404 400 402 400 401 400 -50")
	// R6: odd players win
	pairtestutils.AddRoundResultsStr(req, "400 420 400 412 400 408 400 404 400 402 400 401 -50")

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_25_topdownbye_precedence_over_factor3.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "Factor 3 skipped: rank 1 (P0) is in the top 6"))
	is.True(strings.Contains(resp.Log, "Destinys Child: (none)"))
	// P0 receives the bye via TopDownByes rather than being forced into a
	// Factor 3 pairing.
	is.Equal(resp.Pairings[0], int32(0))
}

// Scenario 26: the leader themselves (rank 0) is this round's top-down bye
// recipient - the real-world conflict fixed by skipping control loss
// entirely in that case. P0, P2, and P4 all finish 4-0 (P0 with the best
// spread) and never bye across 4 rounds (AddNDummyRounds always pairs them
// against P1/P3/P5), while P6 - the odd 7th player - self-byes every round
// instead, so going into round 5, P0 is tied with everyone but P6 at zero
// prior byes; TopDownByes' tiebreak (ties go to the higher-ranked player)
// picks P0. Without the fix, control loss would still try to force P0 to
// play a specific opponent (P2 or P4) while TopDownByes simultaneously
// forces P0 vs BYE - leaving P0 with no legal pairing at all
// (OVERCONSTRAINED, as in the July 4th 2026 random-start run 175 round 25
// log, where the leader Edgar Odongkara was the bye recipient). With the
// fix, control loss is skipped outright since the leader isn't playing
// anyone this round.
func TestScenario26_TopDownByeRecipientIsTheLeader(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6"},
		PlayerClasses:              make([]int32, 7),
		ClassPrizes:                []int32{2},
		GibsonSpread:               scenarioGibsonSpread,
		ControlLossThreshold:       0.05,
		HopefulnessThreshold:       scenarioHopefulness,
		AllPlayers:                 7,
		ValidPlayers:               7,
		Rounds:                     8,
		PlacePrizes:                2,
		DivisionSims:               scenarioDivisionSims,
		ControlLossSims:            scenarioControlLossSims,
		ControlLossActivationRound: 4,
		AllowRepeatByes:            false,
		TopDownByes:                true,
		Seed:                       1,
	}
	// AddNDummyRounds pairs P0vP1, P2vP3, P4vP5 every round; P6 (the odd 7th
	// player) self-byes every round instead, so only P6 ever accumulates a
	// bye across rounds 1-4 - P0, P2, and P4 (the eventual top 3) never do.
	pairtestutils.AddNDummyRounds(req, 4)
	for range 4 {
		pairtestutils.AddRoundResultsStr(req, "420 400 410 400 405 400 -50")
	}

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_26_topdownbye_recipient_is_leader.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "Control loss skipped: rank 1 (P0)"))
	is.True(strings.Contains(resp.Log, "Destinys Child: (none)"))
	// P0 receives the bye via TopDownByes rather than being forced into a
	// control-loss pairing.
	is.Equal(resp.Pairings[0], int32(0))
}

// Scenario 27: same real-world round as
// CreateJuly4th2026RandomStartRun305AfterRound24PairRequest (July 4th 2026
// random-start run 305, round 25 - where the control-loss destiny-child
// search correctly excluded Lukeman Owolabi, this round's top-down bye
// recipient, and paired successfully), except Lukeman Owolabi has one fewer
// win: round 24's Lukeman-vs-Daniel-Blake result (a 477-323 Lukeman win) is
// flipped into a 323-477 loss, dropping Lukeman from 16 to 15 wins.
func TestScenario27_July4th2026Run305Round25LukemanOneFewerWin(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	req := pairtestutils.CreateJuly4th2026RandomStartRun305AfterRound24PairRequest()

	lukemanIdx := 20
	danielBlakeIdx := 45
	lastRound := req.DivisionResults[len(req.DivisionResults)-1]
	is.Equal(lastRound.Results[lukemanIdx], int32(477))
	is.Equal(lastRound.Results[danielBlakeIdx], int32(323))
	lastRound.Results[lukemanIdx], lastRound.Results[danielBlakeIdx] = lastRound.Results[danielBlakeIdx], lastRound.Results[lukemanIdx]

	resp := cop.COPPair(req)
	writeScenarioLog(t, "scenario_27_july4th2026_run305_round25_lukeman_one_fewer_win.log", resp.Log)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// The extra loss drops Lukeman Owolabi from rank 4 (16 wins) to rank 6
	// (15 wins) - Robert Linn (also 15 wins, better spread) takes rank 4 and
	// becomes the new top-down bye recipient instead.
	is.True(strings.Contains(resp.Log, "Lukeman Owolabi      | 15.0 | -178"))
	is.True(strings.Contains(resp.Log, "Control loss: excluding rank 4 (Robert Linn)"))
	is.True(strings.Contains(resp.Log, "Destinys Child: Eta Karo"))
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

// Albany CSW ME July 2026 (wordgameplayers.org/directors/AA003954/2026-07-02-Albany-CSW-ME):
// show what COP would have paired for the last quarter (rounds 22-28 of 28), given the actual
// historical results for all prior rounds. Each round uses only real data - no simulated
// results, unlike TestScenarioMultiRound_July4th2026.
// Run with: COP_SCENARIOS=1 go test -run TestScenarioMultiRound_AlbanyCSWJuly2026_LastQuarter
func TestScenarioMultiRound_AlbanyCSWJuly2026_LastQuarter(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping Albany CSW July 2026 scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)

	base := pairtestutils.CreateAlbanyCSWJuly2026Round28PairRequest()

	// Last quarter: rounds 22-28 of 28 (roundPairingsRemaining*4 <= Rounds starting at round 22).
	for round := int(base.Rounds)*3/4 + 1; round <= int(base.Rounds); round++ {
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
		fmt.Printf("Albany CSW July 2026 round %d pairings: %v\n", round, resp.Pairings)
		writeScenarioLog(t, fmt.Sprintf("albany_csw_july_2026_round_%02d.log", round), resp.Log)
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

// July 4th 2026 28-game 53-player event: 3 rounds of Initial Fontes, then 25 rounds of COP.
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
			TopDownByes:                true,
			ControlLossActivationRound: 24,
			AllowRepeatByes:            false,
			InitialNonperfRounds:       int32(fontesRounds),
			Seed:                       seed,
		}

		allPairings := []*pb.RoundPairings{}
		allResults := []*pb.RoundResults{}

		// Rounds 1-3 use Initial Fontes; rounds 4-28 use COP, with real-tournament timing.
		for round := 1; round <= totalRounds; round++ {
			numRes := numResultsForRound(round, len(allResults))

			if round <= fontesRounds {
				req.PairMethod = pb.PairMethod_PAIR_INITIAL_FONTES
			} else {
				req.PairMethod = pb.PairMethod_COP
			}
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

// TestScenarioMultiRound_July4th2026RandomStart is identical to
// TestScenarioMultiRound_July4th2026 except that rounds 1-22 are paired
// randomly (PAIR_RANDOM) instead of Initial Fontes + COP - simulating a
// standings-blind start - with real COP pairing only kicking in at round 23.
// Run with: COP_SCENARIOS=1 go test -run TestScenarioMultiRound_July4th2026RandomStart
func TestScenarioMultiRound_July4th2026RandomStart(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping July 4th 2026 random-start scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	spreadsDist := standings.GetScoreDifferences()

	const numRuns = 1000000000

	numPlayers := 53
	totalRounds := 28
	randomRounds := 22

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
		runDir := fmt.Sprintf("july4th2026_random_start_run_%02d", run+1)

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
			TopDownByes:                true,
			ControlLossActivationRound: 24,
			AllowRepeatByes:            false,
			Seed:                       seed,
		}

		allPairings := []*pb.RoundPairings{}
		allResults := []*pb.RoundResults{}

		// Rounds 1-22 are paired randomly; rounds 23-28 use COP, with real-tournament timing.
		for round := 1; round <= totalRounds; round++ {
			numRes := numResultsForRound(round, len(allResults))

			if round <= randomRounds {
				req.PairMethod = pb.PairMethod_PAIR_RANDOM
			} else {
				req.PairMethod = pb.PairMethod_COP
			}
			req.DivisionPairings = allPairings
			req.DivisionResults = allResults[:numRes]

			if round == totalRounds {
				req.GibsonSpread = scenarioLastRoundGibsonSpread
			} else {
				req.GibsonSpread = scenarioGibsonSpread
			}

			resp := cop.COPPair(req)
			fmt.Printf("July 4th 2026 random-start run %d round %d pairings: %v\n", run+1, round, resp.Pairings)
			writeScenarioLog(t, fmt.Sprintf("%s/round_%02d.log", runDir, round), resp.Log)
			is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)

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
			InitialNonperfRounds:       int32(fontesRounds),
			Seed:                       seed,
		}

		allPairings := []*pb.RoundPairings{}
		allResults := []*pb.RoundResults{}

		// Rounds 1-3 use Initial Fontes; rounds 4-28 use COP, with real-tournament timing.
		for round := 1; round <= totalRounds; round++ {
			numRes := numResultsForRound(round, len(allResults))

			if round <= fontesRounds {
				req.PairMethod = pb.PairMethod_PAIR_INITIAL_FONTES
			} else {
				req.PairMethod = pb.PairMethod_COP
			}
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

// July 4th 2026 Division 2: 33 players from wordgameplayers.org/tournaments/1162,
// using the same 28-round structure and real-tournament timing as the main event
// (see TestScenarioMultiRound_July4th2026).
// Run with: COP_SCENARIOS=1 go test -run TestScenarioMultiRound_July4th2026Div2
func TestScenarioMultiRound_July4th2026Div2(t *testing.T) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skip("Skipping July 4th 2026 Div2 scenario test. Set COP_SCENARIOS=1 to run.")
	}
	is := is.New(t)
	spreadsDist := standings.GetScoreDifferences()

	const numRuns = 10

	numPlayers := 33
	totalRounds := 28
	fontesRounds := 3

	names := []string{
		"Terry Kang", "K Kaia", "Nitya Chagti", "Justin Morris",
		"David Nwabor", "Benjamin Bloom", "Heidi Robertson", "Rebecca Soble",
		"Ayodele Odekunle", "Elise Bickford", "Yvonne Lobo", "Priya Fernando",
		"Yaacov Barak", "Roger Cullman", "Lindsay Shin", "Joe Roberdeau",
		"Ather Sharif", "Sharmaine Farini", "Jane Geary", "Thomas Stumpf",
		"Jean-Louis Guill", "Samuel Fomum", "Kaveri Warriar", "Nancy Bowen",
		"Caroline Polak Scowcroft", "Joan Kavanaugh", "Iliana Filby", "Cheryl Melvin",
		"Absar Mustajab", "Hema Shah", "Ida Ann Shapiro", "Ben Tu'itahi",
		"Donald Ituah",
	}
	classes := make([]int32, numPlayers)

	for run := 0; run < numRuns; run++ {
		seed := time.Now().UnixNano()
		rng := rand.New(rand.NewSource(uint64(seed)))
		runDir := fmt.Sprintf("july4th2026div2_run_%02d", run+1)

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
			PlacePrizes:                6,
			DivisionSims:               scenarioDivisionSims,
			ControlLossSims:            scenarioControlLossSims,
			ControlLossActivationRound: 22,
			AllowRepeatByes:            false,
			TopDownByes:                true,
			InitialNonperfRounds:       int32(fontesRounds),
			Seed:                       seed,
		}

		allPairings := []*pb.RoundPairings{}
		allResults := []*pb.RoundResults{}

		// Rounds 1-3 use Initial Fontes; rounds 4-28 use COP, with real-tournament timing.
		for round := 1; round <= totalRounds; round++ {
			numRes := numResultsForRound(round, len(allResults))

			if round <= fontesRounds {
				req.PairMethod = pb.PairMethod_PAIR_INITIAL_FONTES
			} else {
				req.PairMethod = pb.PairMethod_COP
			}
			req.DivisionPairings = allPairings
			req.DivisionResults = allResults[:numRes]

			if round == totalRounds {
				req.GibsonSpread = scenarioLastRoundGibsonSpread
			} else {
				req.GibsonSpread = scenarioGibsonSpread
			}

			resp := cop.COPPair(req)
			writeScenarioLog(t, fmt.Sprintf("%s/round_%02d.log", runDir, round), resp.Log)
			is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
			fmt.Printf("July 4th 2026 Div2 run %d round %d pairings: %v\n", run+1, round, resp.Pairings)

			pairings := make([]int32, numPlayers)
			copy(pairings, resp.Pairings)
			allPairings = append(allPairings, &pb.RoundPairings{Pairings: pairings})
			allResults = append(allResults, makeRandomResults(pairings, numPlayers, rng, spreadsDist))
		}
	}
}

// runHypotheticalScenario runs a hypothetical tournament of the given size: 3 rounds of
// Initial Fontes, then COP for the remaining rounds, with simulated (random) results
// throughout, standard 1-game-behind timing, and control loss activating for the last 4 rounds.
// PlacePrizes scales with field size (4 for fields up to 20, 6 beyond that).
func runHypotheticalScenario(t *testing.T, numPlayers, totalRounds int) {
	if os.Getenv("COP_SCENARIOS") == "" {
		t.Skipf("Skipping hypothetical %dp/%dr scenario test. Set COP_SCENARIOS=1 to run.", numPlayers, totalRounds)
	}
	is := is.New(t)
	spreadsDist := standings.GetScoreDifferences()

	const numRuns = 10
	const fontesRounds = 3

	placePrizes := int32(4)
	if numPlayers > 20 {
		placePrizes = 6
	}

	names := make([]string, numPlayers)
	for i := range names {
		names[i] = fmt.Sprintf("P%d", i)
	}
	classes := make([]int32, numPlayers)

	for run := 0; run < numRuns; run++ {
		seed := time.Now().UnixNano()
		rng := rand.New(rand.NewSource(uint64(seed)))
		runDir := fmt.Sprintf("hypothetical%dp%dr_run_%02d", numPlayers, totalRounds, run+1)

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
			PlacePrizes:                placePrizes,
			DivisionSims:               scenarioDivisionSims,
			ControlLossSims:            scenarioControlLossSims,
			ControlLossActivationRound: int32(totalRounds - 4),
			AllowRepeatByes:            false,
			InitialNonperfRounds:       int32(fontesRounds),
			Seed:                       seed,
		}

		allPairings := []*pb.RoundPairings{}
		allResults := []*pb.RoundResults{}

		// Rounds 1-3 use Initial Fontes; rounds 4 onward use COP.
		for round := 1; round <= totalRounds; round++ {
			if round <= fontesRounds {
				req.PairMethod = pb.PairMethod_PAIR_INITIAL_FONTES
			} else {
				req.PairMethod = pb.PairMethod_COP
			}
			req.DivisionPairings = allPairings
			req.DivisionResults = allResults

			if round == totalRounds {
				req.GibsonSpread = scenarioLastRoundGibsonSpread
			} else {
				req.GibsonSpread = scenarioGibsonSpread
			}

			resp := cop.COPPair(req)
			is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
			fmt.Printf("Hypothetical %dp/%dr run %d round %d pairings: %v\n", numPlayers, totalRounds, run+1, round, resp.Pairings)
			writeScenarioLog(t, fmt.Sprintf("%s/round_%02d.log", runDir, round), resp.Log)

			pairings := make([]int32, numPlayers)
			copy(pairings, resp.Pairings)
			allPairings = append(allPairings, &pb.RoundPairings{Pairings: pairings})
			allResults = append(allResults, makeRandomResults(pairings, numPlayers, rng, spreadsDist))
		}
	}
}

// Run with: COP_SCENARIOS=1 go test -run TestScenarioHypo12p7r
func TestScenarioHypo12p7r(t *testing.T) {
	runHypotheticalScenario(t, 12, 7)
}

// Run with: COP_SCENARIOS=1 go test -run TestScenarioHypo18p15r
func TestScenarioHypo18p15r(t *testing.T) {
	runHypotheticalScenario(t, 18, 15)
}

// Run with: COP_SCENARIOS=1 go test -run TestScenarioHypo26p15r
func TestScenarioHypo26p15r(t *testing.T) {
	runHypotheticalScenario(t, 26, 15)
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
