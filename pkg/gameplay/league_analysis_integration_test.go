package gameplay_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"
	"google.golang.org/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// TestLeagueGameEnqueuedForAnalysis verifies that when a league game ends,
// it gets automatically enqueued for analysis.
func TestLeagueGameEnqueuedForAnalysis(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	// Setup: create database and stores
	pool, stores, _ := recreateDB()
	defer stores.Disconnect()

	// Create a minimal league setup
	leagueID := uuid.New()
	_, err := stores.LeagueStore.CreateLeague(ctx, models.CreateLeagueParams{
		Uuid:        leagueID,
		Name:        "Test League",
		Description: pgtype.Text{String: "Test League for Analysis Integration", Valid: true},
		Slug:        "test-league-analysis",
		Settings:    []byte(`{}`),
		IsActive:    pgtype.Bool{Bool: true, Valid: true},
		CreatedBy:   pgtype.Int8{Int64: 1, Valid: true},
	})
	is.NoErr(err)

	seasonID := uuid.New()
	_, err = stores.LeagueStore.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         seasonID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       int32(pb.SeasonStatus_SEASON_ACTIVE),
	})
	is.NoErr(err)

	divisionID := uuid.New()
	_, err = stores.LeagueStore.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           divisionID,
		SeasonID:       seasonID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
	})
	is.NoErr(err)

	// Create a minimal game history with players
	minimalHistory := &macondopb.GameHistory{
		Players: []*macondopb.PlayerInfo{
			{Nickname: "cesar4", RealName: "Cesar", UserId: "xjCWug7EZtDxDHX5fRZTLo"},
			{Nickname: "Mina", RealName: "Mina", UserId: "qUQkST8CendYA3baHNoPjk"},
		},
		FinalScores: []int32{400, 350},
		Lexicon:     "CSW21",
		IdAuth:      "xjCWug7EZtDxDHX5fRZTLo",
	}
	histBytes, err := proto.Marshal(minimalHistory)
	is.NoErr(err)

	// Create a game in the database with league metadata
	// Game IDs must be 24 chars or less
	gameID := "leaguegame" + uuid.New().String()[:10]
	_, err = pool.Exec(ctx, `
		INSERT INTO games(uuid, player0_id, player1_id, started, game_end_reason, type, game_request,
			              history, quickdata, timers, league_id, league_division_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		gameID,
		1, // player0_id (cesar4)
		2, // player1_id (Mina)
		true,
		int(pb.GameEndReason_STANDARD), // Game ended normally
		0,                              // type
		`{"lexicon": "CSW21", "rules": {"boardLayoutName": "CrosswordGame", "letterDistributionName": "english"}, "initialTimeSeconds": 1500, "maxOvertimeMinutes": 0}`,
		histBytes,
		`{}`,
		`{"mo": 0, "tr": [1500000, 1500000], "ts": 1700000000000}`, // minimal timers - no overtime, 1500s each
		leagueID,
		divisionID,
	)
	is.NoErr(err)

	// Load the game from the store
	g, err := stores.GameStore.Get(ctx, gameID)
	is.NoErr(err)

	// Verify it's a league game
	is.True(g.LeagueDivisionID != nil)
	is.Equal(*g.LeagueDivisionID, divisionID)

	// Call PerformEndgameDuties (this is what happens when a game ends)
	err = gameplay.PerformEndgameDuties(ctx, g, stores)
	is.NoErr(err)

	// Verify: Check that an analysis job was created
	// Use the game's actual ID (which might differ from the database uuid)
	actualGameID := g.GameID()
	job, err := stores.Queries.GetJobByGameID(ctx, actualGameID)
	is.NoErr(err)
	is.Equal(job.Status, "pending")
	is.Equal(job.GameID, actualGameID)
	is.True(len(job.ConfigJson) > 0)
}

// TestNonLeagueGameNotEnqueued verifies that regular (non-league) games
// do NOT get enqueued for analysis.
func TestNonLeagueGameNotEnqueued(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	// Setup: create database and stores
	pool, stores, _ := recreateDB()
	defer stores.Disconnect()

	// Create a minimal history for non-league game too
	minimalHistory := &macondopb.GameHistory{
		Players: []*macondopb.PlayerInfo{
			{Nickname: "cesar4", RealName: "Cesar", UserId: "xjCWug7EZtDxDHX5fRZTLo"},
			{Nickname: "Mina", RealName: "Mina", UserId: "qUQkST8CendYA3baHNoPjk"},
		},
		FinalScores: []int32{300, 250},
		Lexicon:     "CSW21",
		IdAuth:      "xjCWug7EZtDxDHX5fRZTLo",
	}
	histBytes, err := proto.Marshal(minimalHistory)
	is.NoErr(err)

	// Create a regular (non-league) game in the database
	// Game IDs must be 24 chars or less
	gameID := "regulargame" + uuid.New().String()[:10]
	_, err = pool.Exec(ctx, `
		INSERT INTO games(uuid, player0_id, player1_id, started, game_end_reason, type, game_request,
			              history, quickdata, timers)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		gameID,
		1, // player0_id
		2, // player1_id
		true,
		int(pb.GameEndReason_STANDARD),
		0, // type
		`{"lexicon": "CSW21", "rules": {"boardLayoutName": "CrosswordGame", "letterDistributionName": "english"}, "initialTimeSeconds": 1500, "maxOvertimeMinutes": 0}`,
		histBytes,
		`{}`,
		`{"mo": 0, "tr": [1500000, 1500000], "ts": 1700000000000}`,
	)
	is.NoErr(err)

	// Load the game
	g, err := stores.GameStore.Get(ctx, gameID)
	is.NoErr(err)

	// Verify it's NOT a league game
	is.True(g.LeagueDivisionID == nil)

	// Call PerformEndgameDuties
	err = gameplay.PerformEndgameDuties(ctx, g, stores)
	is.NoErr(err)

	// Verify: No analysis job should be created
	_, err = stores.Queries.GetJobByGameID(ctx, gameID)
	is.True(err != nil) // Should get an error because no job exists
}
