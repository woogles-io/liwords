package league

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// TestCalculateAndSaveStandings_TiesAtTheLines checks the outcomes a season's
// close saves from the stored records: players level for first in Division 1
// are joint champions, and in Division 2 of 3 (15 players, 5 up and 5 down)
// the players level across the promotion line all go up and the players level
// across the relegation line all stay.
func TestCalculateAndSaveStandings_TiesAtTheLines(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, cleanup := setupIntegrationTest(t)
	defer cleanup()

	store := allStores.LeagueStore
	_, seasonID := createLeagueAndSeason(t, ctx, allStores)
	err := store.UpdateSeasonPromotionFormula(ctx, models.UpdateSeasonPromotionFormulaParams{
		Uuid: seasonID, PromotionFormula: int32(ipc.PromotionFormula_PROMO_N_DIV_3),
	})
	is.NoErr(err)

	divs := make([]uuid.UUID, 3)
	for d := range divs {
		divs[d] = uuid.New()
		_, err := store.CreateDivision(ctx, models.CreateDivisionParams{
			Uuid: divs[d], SeasonID: seasonID, DivisionNumber: int32(d + 1),
			DivisionName: pgtype.Text{String: "Division", Valid: true},
		})
		is.NoErr(err)
	}

	type record struct{ user, wins, losses, spread int32 }
	// Division 1: testuser16 and testuser17 level on points and spread.
	div1 := []record{{16, 3, 1, 50}, {17, 3, 1, 50}, {18, 1, 3, -20}, {19, 1, 3, -80}}
	// Division 2: testuser1 to testuser15 in that order, testuser6 level with
	// testuser5 at the promotion line and testuser11 level with testuser10 at
	// the relegation line.
	var div2 []record
	for i := int32(1); i <= 15; i++ {
		pos := i
		if i == 6 || i == 11 {
			pos = i - 1
		}
		div2 = append(div2, record{i, 15 - pos, pos - 1, 100 * (8 - pos)})
	}
	for d, records := range [][]record{div1, div2} {
		for _, r := range records {
			err := store.UpsertStanding(ctx, models.UpsertStandingParams{
				DivisionID: divs[d], UserID: r.user,
				Wins:        pgtype.Int4{Int32: r.wins, Valid: true},
				Losses:      pgtype.Int4{Int32: r.losses, Valid: true},
				Draws:       pgtype.Int4{Int32: 0, Valid: true},
				Spread:      pgtype.Int4{Int32: r.spread, Valid: true},
				GamesPlayed: pgtype.Int4{Int32: r.wins + r.losses, Valid: true},
			})
			is.NoErr(err)
		}
	}

	err = NewStandingsManager(store).CalculateAndSaveStandings(ctx, seasonID)
	is.NoErr(err)

	want := map[int32]ipc.StandingResult{
		16: ipc.StandingResult_RESULT_CHAMPION,
		17: ipc.StandingResult_RESULT_CHAMPION,
		18: ipc.StandingResult_RESULT_RELEGATED,
		19: ipc.StandingResult_RESULT_RELEGATED,
	}
	for i := int32(1); i <= 15; i++ {
		switch {
		case i <= 6:
			want[i] = ipc.StandingResult_RESULT_PROMOTED
		case i <= 11:
			want[i] = ipc.StandingResult_RESULT_STAYED
		default:
			want[i] = ipc.StandingResult_RESULT_RELEGATED
		}
	}
	for d := 0; d < 2; d++ {
		standings, err := store.GetStandings(ctx, divs[d])
		is.NoErr(err)
		for _, s := range standings {
			if got := ipc.StandingResult(s.Result.Int32); got != want[s.UserID] {
				t.Errorf("%s: got %v, want %v", s.Username, got, want[s.UserID])
			}
		}
	}
}
