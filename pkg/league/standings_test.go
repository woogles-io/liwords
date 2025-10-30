package league

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestMarkOutcomes_Division15Players(t *testing.T) {
	is := is.New(t)

	sm := &StandingsManager{}

	// Create 15 players
	standings := make([]PlayerStanding, 15)
	for i := 0; i < 15; i++ {
		standings[i] = PlayerStanding{
			UserID: uuid.NewString(),
			Rank:   i + 1,
		}
	}

	// Mark outcomes for a middle division (not highest or lowest)
	// highestRegularDivision = 5 (so division 5 is lowest, division 2 is middle)
	sm.markOutcomes(standings, 2, 5)

	// With 15 players: ceil(15/6) = 3 promoted, 3 relegated, 9 stay
	promotedCount := 0
	relegatedCount := 0
	stayedCount := 0

	for _, s := range standings {
		switch s.Outcome {
		case pb.StandingResult_RESULT_PROMOTED:
			promotedCount++
		case pb.StandingResult_RESULT_RELEGATED:
			relegatedCount++
		case pb.StandingResult_RESULT_STAYED:
			stayedCount++
		}
	}

	is.Equal(promotedCount, 3)
	is.Equal(relegatedCount, 3)
	is.Equal(stayedCount, 9)

	// Check that top 3 are promoted
	is.Equal(standings[0].Outcome, pb.StandingResult_RESULT_PROMOTED)
	is.Equal(standings[1].Outcome, pb.StandingResult_RESULT_PROMOTED)
	is.Equal(standings[2].Outcome, pb.StandingResult_RESULT_PROMOTED)

	// Check that bottom 3 are relegated
	is.Equal(standings[12].Outcome, pb.StandingResult_RESULT_RELEGATED)
	is.Equal(standings[13].Outcome, pb.StandingResult_RESULT_RELEGATED)
	is.Equal(standings[14].Outcome, pb.StandingResult_RESULT_RELEGATED)
}

func TestMarkOutcomes_Division13Players(t *testing.T) {
	is := is.New(t)

	sm := &StandingsManager{}

	// Create 13 players (minimum regular division size)
	standings := make([]PlayerStanding, 13)
	for i := 0; i < 13; i++ {
		standings[i] = PlayerStanding{
			UserID: uuid.NewString(),
			Rank:   i + 1,
		}
	}

	sm.markOutcomes(standings, 2, 5)

	// With 13 players: ceil(13/6) = 3 promoted, 3 relegated, 7 stay
	promotedCount := 0
	relegatedCount := 0
	stayedCount := 0

	for _, s := range standings {
		switch s.Outcome {
		case pb.StandingResult_RESULT_PROMOTED:
			promotedCount++
		case pb.StandingResult_RESULT_RELEGATED:
			relegatedCount++
		case pb.StandingResult_RESULT_STAYED:
			stayedCount++
		}
	}

	is.Equal(promotedCount, 3)
	is.Equal(relegatedCount, 3)
	is.Equal(stayedCount, 7)
}

func TestMarkOutcomes_Division20Players(t *testing.T) {
	is := is.New(t)

	sm := &StandingsManager{}

	// Create 20 players (maximum division size)
	standings := make([]PlayerStanding, 20)
	for i := 0; i < 20; i++ {
		standings[i] = PlayerStanding{
			UserID: uuid.NewString(),
			Rank:   i + 1,
		}
	}

	sm.markOutcomes(standings, 2, 5)

	// With 20 players: ceil(20/6) = 4 promoted, 4 relegated, 12 stay
	promotedCount := 0
	relegatedCount := 0
	stayedCount := 0

	for _, s := range standings {
		switch s.Outcome {
		case pb.StandingResult_RESULT_PROMOTED:
			promotedCount++
		case pb.StandingResult_RESULT_RELEGATED:
			relegatedCount++
		case pb.StandingResult_RESULT_STAYED:
			stayedCount++
		}
	}

	is.Equal(promotedCount, 4)
	is.Equal(relegatedCount, 4)
	is.Equal(stayedCount, 12)
}

func TestMarkOutcomes_Division1_NoPromotions(t *testing.T) {
	is := is.New(t)

	sm := &StandingsManager{}

	// Create 15 players in Division 1 (highest division)
	standings := make([]PlayerStanding, 15)
	for i := 0; i < 15; i++ {
		standings[i] = PlayerStanding{
			UserID: uuid.NewString(),
			Rank:   i + 1,
		}
	}

	sm.markOutcomes(standings, 1, 5) // Division 1

	// Division 1 cannot promote, so top players should stay
	promotedCount := 0
	relegatedCount := 0
	stayedCount := 0

	for _, s := range standings {
		switch s.Outcome {
		case pb.StandingResult_RESULT_PROMOTED:
			promotedCount++
		case pb.StandingResult_RESULT_RELEGATED:
			relegatedCount++
		case pb.StandingResult_RESULT_STAYED:
			stayedCount++
		}
	}

	is.Equal(promotedCount, 0)
	is.Equal(relegatedCount, 3) // Bottom 3 still relegated
	is.Equal(stayedCount, 12)   // Top 3 + middle 9 = 12 stay
}

func TestMarkOutcomes_LowestDivision_NoRelegations(t *testing.T) {
	is := is.New(t)

	sm := &StandingsManager{}

	// Create 15 players in lowest division (Division 5 out of 5)
	standings := make([]PlayerStanding, 15)
	for i := 0; i < 15; i++ {
		standings[i] = PlayerStanding{
			UserID: uuid.NewString(),
			Rank:   i + 1,
		}
	}

	sm.markOutcomes(standings, 5, 5) // Division 5, total 5 divisions

	// Lowest division cannot relegate, so bottom players should stay
	promotedCount := 0
	relegatedCount := 0
	stayedCount := 0

	for _, s := range standings {
		switch s.Outcome {
		case pb.StandingResult_RESULT_PROMOTED:
			promotedCount++
		case pb.StandingResult_RESULT_RELEGATED:
			relegatedCount++
		case pb.StandingResult_RESULT_STAYED:
			stayedCount++
		}
	}

	is.Equal(promotedCount, 3)  // Top 3 promoted
	is.Equal(relegatedCount, 0) // No relegations
	is.Equal(stayedCount, 12)   // Middle 9 + bottom 3 = 12 stay
}

func TestMarkOutcomes_RookieDivision_NoRelegations(t *testing.T) {
	is := is.New(t)

	sm := &StandingsManager{}

	// Create 15 players in rookie division (100+)
	standings := make([]PlayerStanding, 15)
	for i := 0; i < 15; i++ {
		standings[i] = PlayerStanding{
			UserID: uuid.NewString(),
			Rank:   i + 1,
		}
	}

	sm.markOutcomes(standings, 100, 5) // Rookie division

	// Rookie divisions cannot relegate (they're already at the bottom)
	promotedCount := 0
	relegatedCount := 0
	stayedCount := 0

	for _, s := range standings {
		switch s.Outcome {
		case pb.StandingResult_RESULT_PROMOTED:
			promotedCount++
		case pb.StandingResult_RESULT_RELEGATED:
			relegatedCount++
		case pb.StandingResult_RESULT_STAYED:
			stayedCount++
		}
	}

	is.Equal(promotedCount, 3)  // Top 3 will "graduate"
	is.Equal(relegatedCount, 0) // No relegations from rookie division
	is.Equal(stayedCount, 12)
}

func TestSortStandings_ByWins(t *testing.T) {
	is := is.New(t)

	sm := &StandingsManager{}

	standings := []PlayerStanding{
		{UserID: "player1", Wins: 5, Spread: 100},
		{UserID: "player2", Wins: 10, Spread: 50},
		{UserID: "player3", Wins: 8, Spread: 200},
	}

	sm.sortStandings(standings)

	// Should be sorted by wins descending
	is.Equal(standings[0].UserID, "player2") // 10 wins
	is.Equal(standings[1].UserID, "player3") // 8 wins
	is.Equal(standings[2].UserID, "player1") // 5 wins
}

func TestSortStandings_BySpreadWhenWinsTied(t *testing.T) {
	is := is.New(t)

	sm := &StandingsManager{}

	standings := []PlayerStanding{
		{UserID: "player1", Wins: 10, Spread: 50},
		{UserID: "player2", Wins: 10, Spread: 200},
		{UserID: "player3", Wins: 10, Spread: 100},
	}

	sm.sortStandings(standings)

	// Should be sorted by spread descending when wins are equal
	is.Equal(standings[0].UserID, "player2") // 200 spread
	is.Equal(standings[1].UserID, "player3") // 100 spread
	is.Equal(standings[2].UserID, "player1") // 50 spread
}

func TestCalculateAndSaveStandings_Integration(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	store, cleanup := setupTest(t)
	defer cleanup()

	// Create league and season
	leagueID := uuid.New()
	_, err := store.CreateLeague(ctx, models.CreateLeagueParams{
		Uuid:        leagueID,
		Name:        "Test League",
		Description: pgtype.Text{String: "Test", Valid: true},
		Slug:        "test",
		Settings:    []byte(`{}`),
		IsActive:    pgtype.Bool{Bool: true, Valid: true},
		CreatedBy:   pgtype.Int8{Int64: 1, Valid: true},
	})
	is.NoErr(err)

	seasonID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         seasonID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Valid: true},
		EndDate:      pgtype.Timestamptz{Valid: true},
		Status:       "COMPLETED",
	})
	is.NoErr(err)

	// Create 3 divisions with different sizes
	div1, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       seasonID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 15, Valid: true},
	})
	is.NoErr(err)

	div2, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       seasonID,
		DivisionNumber: 2,
		DivisionName:   pgtype.Text{String: "Division 2", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 13, Valid: true},
	})
	is.NoErr(err)

	div3, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       seasonID,
		DivisionNumber: 3,
		DivisionName:   pgtype.Text{String: "Division 3", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 20, Valid: true},
	})
	is.NoErr(err)

	// Register players in each division
	rm := NewRegistrationManager(store)

	// Division 1: 15 players
	for i := 0; i < 15; i++ {
		userID := uuid.NewString()
		err = rm.RegisterPlayer(ctx, userID, seasonID, 1500)
		is.NoErr(err)
		err = store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      userID,
			SeasonID:    seasonID,
			DivisionID:  pgtype.UUID{Bytes: div1.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	// Division 2: 13 players
	for i := 0; i < 13; i++ {
		userID := uuid.NewString()
		err = rm.RegisterPlayer(ctx, userID, seasonID, 1400)
		is.NoErr(err)
		err = store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      userID,
			SeasonID:    seasonID,
			DivisionID:  pgtype.UUID{Bytes: div2.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	// Division 3: 20 players
	for i := 0; i < 20; i++ {
		userID := uuid.NewString()
		err = rm.RegisterPlayer(ctx, userID, seasonID, 1300)
		is.NoErr(err)
		err = store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      userID,
			SeasonID:    seasonID,
			DivisionID:  pgtype.UUID{Bytes: div3.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	// Calculate and save standings
	sm := NewStandingsManager(store)
	err = sm.CalculateAndSaveStandings(ctx, seasonID)
	is.NoErr(err)

	// Verify standings were saved
	div1Standings, err := store.GetStandings(ctx, div1.Uuid)
	is.NoErr(err)
	is.Equal(len(div1Standings), 15)

	// Division 1: No promotions (it's the highest), 3 relegated, 12 stay
	div1Promoted := 0
	div1Relegated := 0
	div1Stayed := 0
	for _, s := range div1Standings {
		if s.Result.Valid {
			switch s.Result.String {
			case "PROMOTED":
				div1Promoted++
			case "RELEGATED":
				div1Relegated++
			case "STAYED":
				div1Stayed++
			}
		}
	}
	is.Equal(div1Promoted, 0)
	is.Equal(div1Relegated, 3)
	is.Equal(div1Stayed, 12)

	// Division 2: 3 promoted, 3 relegated, 7 stay
	div2Standings, err := store.GetStandings(ctx, div2.Uuid)
	is.NoErr(err)
	is.Equal(len(div2Standings), 13)

	div2Promoted := 0
	div2Relegated := 0
	div2Stayed := 0
	for _, s := range div2Standings {
		if s.Result.Valid {
			switch s.Result.String {
			case "PROMOTED":
				div2Promoted++
			case "RELEGATED":
				div2Relegated++
			case "STAYED":
				div2Stayed++
			}
		}
	}
	is.Equal(div2Promoted, 3)
	is.Equal(div2Relegated, 3)
	is.Equal(div2Stayed, 7)

	// Division 3 (lowest): 4 promoted, 0 relegated, 16 stay
	div3Standings, err := store.GetStandings(ctx, div3.Uuid)
	is.NoErr(err)
	is.Equal(len(div3Standings), 20)

	div3Promoted := 0
	div3Relegated := 0
	div3Stayed := 0
	for _, s := range div3Standings {
		if s.Result.Valid {
			switch s.Result.String {
			case "PROMOTED":
				div3Promoted++
			case "RELEGATED":
				div3Relegated++
			case "STAYED":
				div3Stayed++
			}
		}
	}
	is.Equal(div3Promoted, 4)
	is.Equal(div3Relegated, 0) // Lowest division can't relegate
	is.Equal(div3Stayed, 16)
}
