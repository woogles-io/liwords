package league

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// TestRebalance_TiesAtTheLines checks next season's placement when ties at
// the lines change how many move: players tied across a promotion line all go
// up, and players tied across a relegation line all stay.
//
// Scenario: 45 players in 3 divisions of 15, 5 up and 5 down without ties.
//   - Div 1: 11 STAYED, 4 RELEGATED (a tie at its relegation line keeps one).
//   - Div 2: 6 PROMOTED (a tie at its promotion line), 5 STAYED, 4 RELEGATED.
//   - Div 3: 6 PROMOTED (a tie at its promotion line), 9 STAYED.
//
// Every PROMOTED player must land at least a division up, every STAYED
// player outside the bottom division no lower than before, and nobody may be
// lost.
func TestRebalance_TiesAtTheLines(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, cleanup := setupIntegrationTest(t)
	defer cleanup()

	store := allStores.LeagueStore
	leagueID, season1ID := createLeagueAndSeason(t, ctx, allStores)

	divs := make([]uuid.UUID, 3)
	for d := range divs {
		divs[d] = uuid.New()
		_, err := store.CreateDivision(ctx, models.CreateDivisionParams{
			Uuid: divs[d], SeasonID: season1ID, DivisionNumber: int32(d + 1),
			DivisionName: pgtype.Text{String: "Division", Valid: true},
		})
		is.NoErr(err)
	}

	// outcome of player i (1-based), in season 1 division (i-1)/15 + 1
	outcome := func(i int) ipc.StandingResult {
		div, pos := (i-1)/15+1, (i-1)%15+1
		switch {
		case div == 1 && pos > 11:
			return ipc.StandingResult_RESULT_RELEGATED
		case div == 2 && pos <= 6:
			return ipc.StandingResult_RESULT_PROMOTED
		case div == 2 && pos > 11:
			return ipc.StandingResult_RESULT_RELEGATED
		case div == 3 && pos <= 6:
			return ipc.StandingResult_RESULT_PROMOTED
		}
		return ipc.StandingResult_RESULT_STAYED
	}

	for i := 1; i <= 45; i++ {
		divID := divs[(i-1)/15]
		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID: int32(i), SeasonID: season1ID,
			Status:     pgtype.Text{String: "REGISTERED", Valid: true},
			DivisionID: pgtype.UUID{Bytes: divID, Valid: true},
		})
		is.NoErr(err)
		err = store.UpsertStanding(ctx, models.UpsertStandingParams{
			UserID: int32(i), DivisionID: divID,
			Wins: pgtype.Int4{Int32: 7, Valid: true}, Losses: pgtype.Int4{Int32: 7, Valid: true},
		})
		is.NoErr(err)
		err = store.UpdateStandingResult(ctx, models.UpdateStandingResultParams{
			DivisionID: divID, UserID: int32(i),
			Result: pgtype.Int4{Int32: int32(outcome(i)), Valid: true},
		})
		is.NoErr(err)
	}

	season2ID := uuid.New()
	_, err := store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid: season2ID, LeagueID: leagueID, SeasonNumber: 2,
		StartDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:   pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:    int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	is.NoErr(err)
	for i := 1; i <= 45; i++ {
		_, err = store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID: int32(i), SeasonID: season2ID,
			Status: pgtype.Text{String: "REGISTERED", Valid: true},
		})
		is.NoErr(err)
	}
	regs, err := store.GetSeasonRegistrations(ctx, season2ID)
	is.NoErr(err)
	is.Equal(len(regs), 45)
	categorized := make([]CategorizedPlayer, len(regs))
	for i, r := range regs {
		categorized[i] = CategorizedPlayer{Registration: r, Category: PlayerCategoryReturning}
	}

	rm := NewRebalanceManager(allStores)
	result, err := rm.RebalanceDivisions(ctx, leagueID, season1ID, season2ID, 2, categorized, 15)
	is.NoErr(err)
	is.Equal(result.DivisionsCreated, 3)

	s2Divs, err := store.GetDivisionsBySeason(ctx, season2ID)
	is.NoErr(err)
	numberOf := map[uuid.UUID]int{}
	for _, d := range s2Divs {
		numberOf[d.Uuid] = int(d.DivisionNumber)
	}

	sizes := map[int]int{}
	for i := 1; i <= 45; i++ {
		reg, err := store.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
			SeasonID: season2ID, UserID: int32(i),
		})
		is.NoErr(err)
		is.True(reg.DivisionID.Valid) // everyone placed
		now := numberOf[uuid.UUID(reg.DivisionID.Bytes)]
		sizes[now]++
		before := (i-1)/15 + 1
		switch outcome(i) {
		case ipc.StandingResult_RESULT_PROMOTED:
			is.True(now <= before-1) // a promoted player moves up
		case ipc.StandingResult_RESULT_STAYED:
			if before < 3 {
				is.True(now <= before) // a stayed player is not moved down
			}
		}
	}
	is.Equal(sizes[1]+sizes[2]+sizes[3], 45)
	t.Logf("season 2 division sizes: %d, %d, %d", sizes[1], sizes[2], sizes[3])
}
