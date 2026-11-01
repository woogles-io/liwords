package league

import (
	"testing"

	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// TestMarkOutcomes_ThreeDivisions tests the specific scenario from the integration test
// where we have 3 divisions (1, 2, 3) and need to ensure Division 2 can still relegate
func TestMarkOutcomes_ThreeDivisions(t *testing.T) {
	is := is.New(t)
	sm := &StandingsManager{}

	// Division 1: 15 players, highest division
	div1Standings := make([]PlayerStanding, 15)
	for i := 0; i < 15; i++ {
		div1Standings[i] = PlayerStanding{
			UserID: int32(i + 1),
			Rank:   i + 1,
		}
	}
	sm.markOutcomes(div1Standings, 1, 3, pb.PromotionFormula_PROMO_N_DIV_6) // Division 1, highest regular = 3

	div1Champion := 0
	div1Promoted := 0
	div1Relegated := 0
	div1Stayed := 0
	for _, s := range div1Standings {
		switch s.Outcome {
		case pb.StandingResult_RESULT_CHAMPION:
			div1Champion++
		case pb.StandingResult_RESULT_PROMOTED:
			div1Promoted++
		case pb.StandingResult_RESULT_RELEGATED:
			div1Relegated++
		case pb.StandingResult_RESULT_STAYED:
			div1Stayed++
		}
	}
	// Division 1: Rank 1 is champion, can't promote, can relegate
	is.Equal(div1Champion, 1)
	is.Equal(div1Promoted, 0)
	is.Equal(div1Relegated, 3)
	is.Equal(div1Stayed, 11) // 15 - 1 champion - 3 relegated = 11

	// Division 2: 13 players, middle division
	div2Standings := make([]PlayerStanding, 13)
	for i := 0; i < 13; i++ {
		div2Standings[i] = PlayerStanding{
			UserID: int32(16 + i), // IDs 16-28
			Rank:   i + 1,
		}
	}
	sm.markOutcomes(div2Standings, 2, 3, pb.PromotionFormula_PROMO_N_DIV_6) // Division 2, highest regular = 3

	div2Promoted := 0
	div2Relegated := 0
	div2Stayed := 0
	for _, s := range div2Standings {
		switch s.Outcome {
		case pb.StandingResult_RESULT_PROMOTED:
			div2Promoted++
		case pb.StandingResult_RESULT_RELEGATED:
			div2Relegated++
		case pb.StandingResult_RESULT_STAYED:
			div2Stayed++
		}
	}
	// Division 2: Can promote and relegate
	is.Equal(div2Promoted, 3)
	is.Equal(div2Relegated, 3)
	is.Equal(div2Stayed, 7)

	// Division 3: 20 players, lowest division
	div3Standings := make([]PlayerStanding, 20)
	for i := 0; i < 20; i++ {
		div3Standings[i] = PlayerStanding{
			UserID: int32(29 + i), // IDs 29-48
			Rank:   i + 1,
		}
	}
	sm.markOutcomes(div3Standings, 3, 3, pb.PromotionFormula_PROMO_N_DIV_6) // Division 3, highest regular = 3

	div3Promoted := 0
	div3Relegated := 0
	div3Stayed := 0
	for _, s := range div3Standings {
		switch s.Outcome {
		case pb.StandingResult_RESULT_PROMOTED:
			div3Promoted++
		case pb.StandingResult_RESULT_RELEGATED:
			div3Relegated++
		case pb.StandingResult_RESULT_STAYED:
			div3Stayed++
		}
	}
	// Division 3: Can promote, can't relegate (it's the lowest)
	is.Equal(div3Promoted, 4)
	is.Equal(div3Relegated, 0)
	is.Equal(div3Stayed, 16)
}

// TestMarkOutcomes_TiesAtTheLines checks that players level on points and
// spread share the better outcome at the promotion and relegation lines.
func TestMarkOutcomes_TiesAtTheLines(t *testing.T) {
	is := is.New(t)
	sm := &StandingsManager{}

	// 15 players in a middle division with PROMO_N_DIV_3: 5 up and 5 down,
	// sorted as the standings are, by points and then spread.
	records := []struct{ wins, losses, spread int }{
		{11, 3, 605},
		{9, 5, 694},
		{9, 5, 347},
		{9, 5, 319},
		{8, 6, 186},
		{8, 6, 186}, // level with 5th, across the promotion line
		{7, 7, -131},
		{7, 7, -200},
		{6, 8, 165},
		{6, 8, -18},
		{6, 8, -18}, // level with 10th, across the relegation line
		{5, 9, -192},
		{4, 10, -729},
		{4, 10, -729}, // level with 13th, both inside the relegation zone
		{2, 12, -618},
	}
	standings := make([]PlayerStanding, len(records))
	for i, r := range records {
		standings[i] = PlayerStanding{
			UserID: int32(i + 1),
			Wins:   r.wins,
			Losses: r.losses,
			Spread: r.spread,
		}
	}
	sm.markOutcomes(standings, 3, 5, pb.PromotionFormula_PROMO_N_DIV_3)

	const (
		up   = pb.StandingResult_RESULT_PROMOTED
		stay = pb.StandingResult_RESULT_STAYED
		down = pb.StandingResult_RESULT_RELEGATED
	)
	want := []pb.StandingResult{
		up, up, up, up, up, up,
		stay, stay, stay, stay, stay,
		down, down, down, down,
	}
	for i, s := range standings {
		is.Equal(s.Outcome, want[i]) // outcome at this position
	}
}

// TestMarkOutcomes_ThreeWayTies checks that a tie of more than two shares the
// better outcome whole: three level across the promotion line all go up, and
// three level across the relegation line all stay.
func TestMarkOutcomes_ThreeWayTies(t *testing.T) {
	is := is.New(t)
	sm := &StandingsManager{}

	records := []struct{ wins, losses, spread int }{
		{11, 3, 600},
		{10, 4, 400},
		{9, 5, 300},
		{9, 5, 200},
		{8, 6, 100},
		{8, 6, 100},
		{8, 6, 100}, // 5th to 7th level, across the promotion line
		{7, 7, 50},
		{7, 7, -10},
		{6, 8, -20},
		{6, 8, -20},
		{6, 8, -20}, // 10th to 12th level, across the relegation line
		{5, 9, -100},
		{4, 10, -200},
		{2, 12, -400},
	}
	standings := make([]PlayerStanding, len(records))
	for i, r := range records {
		standings[i] = PlayerStanding{
			UserID: int32(i + 1),
			Wins:   r.wins,
			Losses: r.losses,
			Spread: r.spread,
		}
	}
	sm.markOutcomes(standings, 3, 5, pb.PromotionFormula_PROMO_N_DIV_3)

	const (
		up   = pb.StandingResult_RESULT_PROMOTED
		stay = pb.StandingResult_RESULT_STAYED
		down = pb.StandingResult_RESULT_RELEGATED
	)
	want := []pb.StandingResult{
		up, up, up, up, up, up, up,
		stay, stay, stay, stay, stay,
		down, down, down,
	}
	for i, s := range standings {
		is.Equal(s.Outcome, want[i]) // outcome at this position
	}
}
