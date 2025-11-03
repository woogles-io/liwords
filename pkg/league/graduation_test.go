package league

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

func TestCalculateGraduationGroups_Standard(t *testing.T) {
	is := is.New(t)

	gm := &GraduationManager{}

	// Create 19 rookie standings
	rookies := make([]models.LeagueStanding, 19)
	for i := 0; i < 19; i++ {
		rookies[i] = models.LeagueStanding{
			UserID: uuid.NewString(),
			Rank:   pgtype.Int4{Int32: int32(i + 1), Valid: true},
		}
	}

	// 19 rookies, 12 divisions -> [4,4,4,4,3] into [8,9,10,11,12]
	groups := gm.calculateGraduationGroups(rookies, 12)

	is.Equal(len(groups), 5)

	// Check group sizes
	is.Equal(len(groups[0].Rookies), 4)
	is.Equal(len(groups[1].Rookies), 4)
	is.Equal(len(groups[2].Rookies), 4)
	is.Equal(len(groups[3].Rookies), 4)
	is.Equal(len(groups[4].Rookies), 3)

	// Check target divisions
	is.Equal(groups[0].TargetDivision, int32(8))
	is.Equal(groups[1].TargetDivision, int32(9))
	is.Equal(groups[2].TargetDivision, int32(10))
	is.Equal(groups[3].TargetDivision, int32(11))
	is.Equal(groups[4].TargetDivision, int32(12))

	// Check ranks
	is.Equal(groups[0].StartRank, 1)
	is.Equal(groups[0].EndRank, 4)
	is.Equal(groups[4].StartRank, 17)
	is.Equal(groups[4].EndRank, 19)
}

func TestCalculateGraduationGroups_SkipDivision1(t *testing.T) {
	is := is.New(t)

	gm := &GraduationManager{}

	// Create 15 rookie standings
	rookies := make([]models.LeagueStanding, 15)
	for i := 0; i < 15; i++ {
		rookies[i] = models.LeagueStanding{
			UserID: uuid.NewString(),
			Rank:   pgtype.Int4{Int32: int32(i + 1), Valid: true},
		}
	}

	// 15 rookies, 5 divisions -> [3,3,3,3,3] into [2,3,4,5] (skip Div 1)
	groups := gm.calculateGraduationGroups(rookies, 5)

	is.Equal(len(groups), 5)

	// Check that we skip Division 1
	is.Equal(groups[0].TargetDivision, int32(2))
	is.Equal(groups[1].TargetDivision, int32(3))
	is.Equal(groups[2].TargetDivision, int32(4))
	is.Equal(groups[3].TargetDivision, int32(5))
	is.Equal(groups[4].TargetDivision, int32(5)) // Overflow to last division

	// Check group sizes (all should be 3)
	for _, group := range groups {
		is.Equal(len(group.Rookies), 3)
	}
}

func TestCalculateGraduationGroups_SingleDivision(t *testing.T) {
	is := is.New(t)

	gm := &GraduationManager{}

	// Create 20 rookie standings
	rookies := make([]models.LeagueStanding, 20)
	for i := 0; i < 20; i++ {
		rookies[i] = models.LeagueStanding{
			UserID: uuid.NewString(),
			Rank:   pgtype.Int4{Int32: int32(i + 1), Valid: true},
		}
	}

	// 20 rookies, 1 division -> [20] into [1]
	groups := gm.calculateGraduationGroups(rookies, 1)

	is.Equal(len(groups), 1)
	is.Equal(len(groups[0].Rookies), 20)
	is.Equal(groups[0].TargetDivision, int32(1))
	is.Equal(groups[0].StartRank, 1)
	is.Equal(groups[0].EndRank, 20)
}

func TestCalculateGraduationGroups_TwoDivisions(t *testing.T) {
	is := is.New(t)

	gm := &GraduationManager{}

	// Create 12 rookie standings
	rookies := make([]models.LeagueStanding, 12)
	for i := 0; i < 12; i++ {
		rookies[i] = models.LeagueStanding{
			UserID: uuid.NewString(),
			Rank:   pgtype.Int4{Int32: int32(i + 1), Valid: true},
		}
	}

	// 12 rookies, 2 divisions -> all to Division 2
	groups := gm.calculateGraduationGroups(rookies, 2)

	// groupSize = ceil(12/6) = 2, numGroups = ceil(12/2) = 6 groups
	// But only 2 divisions exist, so all overflow to Div 2
	is.Equal(len(groups), 6)
	for _, group := range groups {
		is.Equal(group.TargetDivision, int32(2)) // All go to Div 2
	}
}

func TestCalculateGraduationGroups_ExactGroups(t *testing.T) {
	is := is.New(t)

	gm := &GraduationManager{}

	// Create 6 rookie standings
	rookies := make([]models.LeagueStanding, 6)
	for i := 0; i < 6; i++ {
		rookies[i] = models.LeagueStanding{
			UserID: uuid.NewString(),
			Rank:   pgtype.Int4{Int32: int32(i + 1), Valid: true},
		}
	}

	// 6 rookies, 5 divisions
	// groupSize = ceil(6/6) = 1, numGroups = ceil(6/1) = 6 groups
	// Starting div = max(2, 5 - 6 + 1) = max(2, 0) = 2
	groups := gm.calculateGraduationGroups(rookies, 5)

	is.Equal(len(groups), 6)
	// Groups should be distributed [2,3,4,5,5,5] (overflow to div 5)
	is.Equal(groups[0].TargetDivision, int32(2))
	is.Equal(groups[1].TargetDivision, int32(3))
	is.Equal(groups[2].TargetDivision, int32(4))
	is.Equal(groups[3].TargetDivision, int32(5))
	is.Equal(groups[4].TargetDivision, int32(5))
	is.Equal(groups[5].TargetDivision, int32(5))
}

func TestCalculateGraduationGroups_ManyRookies(t *testing.T) {
	is := is.New(t)

	gm := &GraduationManager{}

	// Create 100 rookie standings
	rookies := make([]models.LeagueStanding, 100)
	for i := 0; i < 100; i++ {
		rookies[i] = models.LeagueStanding{
			UserID: uuid.NewString(),
			Rank:   pgtype.Int4{Int32: int32(i + 1), Valid: true},
		}
	}

	// 100 rookies, 3 divisions -> overflow case
	// ceil(100/6) = 17 per group, ceil(100/17) = 6 groups
	// Starting div = max(2, 3 - 6 + 1) = max(2, -2) = 2
	groups := gm.calculateGraduationGroups(rookies, 3)

	is.Equal(len(groups), 6)

	// All should target divisions 2 or 3 (capped at highest)
	for _, group := range groups {
		is.True(group.TargetDivision >= 2 && group.TargetDivision <= 3)
	}

	// Check total rookies
	totalRookies := 0
	for _, group := range groups {
		totalRookies += len(group.Rookies)
	}
	is.Equal(totalRookies, 100)
}

