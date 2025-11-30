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

	div1Promoted := 0
	div1Relegated := 0
	div1Stayed := 0
	for _, s := range div1Standings {
		switch s.Outcome {
		case pb.StandingResult_RESULT_PROMOTED:
			div1Promoted++
		case pb.StandingResult_RESULT_RELEGATED:
			div1Relegated++
		case pb.StandingResult_RESULT_STAYED:
			div1Stayed++
		}
	}
	// Division 1: Can't promote, can relegate
	is.Equal(div1Promoted, 0)
	is.Equal(div1Relegated, 3)
	is.Equal(div1Stayed, 12)

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
