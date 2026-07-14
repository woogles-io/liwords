package league

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/league_service"
)

// TestGetAllDivisionStandings_Integration exercises the batched, single-snapshot
// GetAllDivisionStandings against a real Postgres. Three divisions are created
// OUT of division_number order (2, 3, 1) to prove the endpoint orders by
// division_number rather than insertion or uuid order, and the middle division
// (2) has a registered player but NO standings row. That empty-standings case is
// exactly what the groupByDivision lockstep must handle: the batched
// GetStandingsForDivisions returns rows only for divisions 1 and 3, so division
// 2 must line up with an empty bucket instead of stealing division 3's rows.
func TestGetAllDivisionStandings_Integration(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, cleanup := setupIntegrationTest(t)
	defer cleanup()

	store := allStores.LeagueStore
	_, seasonID := createLeagueAndSeason(t, ctx, allStores)

	mkDivision := func(number int32) uuid.UUID {
		id := uuid.New()
		_, err := store.CreateDivision(ctx, models.CreateDivisionParams{
			Uuid:           id,
			SeasonID:       seasonID,
			DivisionNumber: number,
			DivisionName:   pgtype.Text{String: "Division", Valid: true},
		})
		is.NoErr(err)
		return id
	}

	// Insert out of order on purpose.
	div2 := mkDivision(2)
	div3 := mkDivision(3)
	div1 := mkDivision(1)

	register := func(userID int32, divID uuid.UUID, withStanding bool, wins, spread int32) {
		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:     userID,
			SeasonID:   seasonID,
			Status:     pgtype.Text{String: "REGISTERED", Valid: true},
			DivisionID: pgtype.UUID{Bytes: divID, Valid: true},
		})
		is.NoErr(err)
		if withStanding {
			err = store.UpsertStanding(ctx, models.UpsertStandingParams{
				UserID:     userID,
				DivisionID: divID,
				Wins:       pgtype.Int4{Int32: wins, Valid: true},
				Losses:     pgtype.Int4{Int32: 0, Valid: true},
				Draws:      pgtype.Int4{Int32: 0, Valid: true},
				Spread:     pgtype.Int4{Int32: spread, Valid: true},
			})
			is.NoErr(err)
		}
	}

	// Div 1: users 1,2 with standings. Div 2: user 3 registered, NO standings.
	// Div 3: users 4,5 with standings.
	register(1, div1, true, 5, 100)
	register(2, div1, true, 3, 50)
	register(3, div2, false, 0, 0)
	register(4, div3, true, 4, 80)
	register(5, div3, true, 2, 20)

	svc := &LeagueService{store: store}
	resp, err := svc.GetAllDivisionStandings(
		ctx,
		connect.NewRequest(&pb.SeasonRequest{SeasonId: seasonID.String()}),
	)
	is.NoErr(err)

	divs := resp.Msg.Divisions
	is.Equal(len(divs), 3)

	// Ordered by division_number, not insertion/uuid order.
	is.Equal(divs[0].DivisionNumber, int32(1))
	is.Equal(divs[1].DivisionNumber, int32(2))
	is.Equal(divs[2].DivisionNumber, int32(3))

	// Correct grouping, including the empty-middle-division case: division 2 has
	// exactly the one registered-but-unplayed player (zero-filled), and division
	// 3's players did not leak into division 2.
	is.Equal(len(divs[0].Standings), 2) // div 1
	is.Equal(len(divs[1].Standings), 1) // div 2 (zero-filled registration)
	is.Equal(len(divs[2].Standings), 2) // div 3

	is.Equal(divs[1].Standings[0].UserId, "test-uuid-3")

	div1Users := map[string]bool{}
	for _, s := range divs[0].Standings {
		div1Users[s.UserId] = true
	}
	is.True(div1Users["test-uuid-1"])
	is.True(div1Users["test-uuid-2"])
	is.True(!div1Users["test-uuid-4"]) // div 3 player must not appear in div 1

	// Ranks are 1..n per division and bounds are populated and sane.
	for _, d := range divs {
		n := int32(len(d.Standings))
		for i, s := range d.Standings {
			is.Equal(s.Rank, int32(i+1))
			is.True(s.BestRank >= 1 && s.BestRank <= n)
			is.True(s.WorstRank >= 1 && s.WorstRank <= n)
			is.True(s.BestRank <= s.WorstRank)
		}
	}
}
