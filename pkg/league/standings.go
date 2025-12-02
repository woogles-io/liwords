package league

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	// MaxLeagueGamesPerPlayer is the maximum number of games each player should play in a league season
	MaxLeagueGamesPerPlayer = 14
)

// CalculatePromotionCount returns the number of players to promote/relegate
// based on the division size and the formula
func CalculatePromotionCount(divSize int, formula pb.PromotionFormula) int {
	if divSize == 0 {
		return 0
	}
	switch formula {
	case pb.PromotionFormula_PROMO_N_PLUS_1_DIV_5:
		// ceil((N+1)/5): 13->3, 15->4, 17->4, 20->5
		return int(math.Ceil(float64(divSize+1) / 5.0))
	case pb.PromotionFormula_PROMO_N_DIV_5:
		// ceil(N/5): 13->3, 15->3, 17->4, 20->4
		return int(math.Ceil(float64(divSize) / 5.0))
	default:
		// PROMO_N_DIV_6 (default): ceil(N/6): 13->3, 15->3, 17->3, 20->4
		return int(math.Ceil(float64(divSize) / 6.0))
	}
}

// CalculateExpectedGamesPerPlayer returns the expected number of games per player
// based on the number of players in the division
func CalculateExpectedGamesPerPlayer(numPlayers int) int {
	expectedGames := numPlayers - 1
	if expectedGames > MaxLeagueGamesPerPlayer {
		return MaxLeagueGamesPerPlayer
	}
	return expectedGames
}

// StandingsManager handles calculating and marking player standings
type StandingsManager struct {
	store league.Store
}

// NewStandingsManager creates a new standings manager
func NewStandingsManager(store league.Store) *StandingsManager {
	return &StandingsManager{
		store: store,
	}
}

// PlayerStanding represents a player's standing within a division
type PlayerStanding struct {
	UserID      int32 // Database ID, not UUID
	DivisionID  uuid.UUID
	Username    string
	Wins        int
	Losses      int
	Draws       int
	Spread      int
	GamesPlayed int
	Rank        int // Deprecated: calculated from position in sorted array, not stored in DB
	Outcome     pb.StandingResult
	// Extended stats
	TotalScore               int
	TotalOpponentScore       int
	TotalBingos              int
	TotalOpponentBingos      int
	TotalTurns               int
	HighTurn                 int
	HighGame                 int
	Timeouts                 int
	BlanksPlayed             int
	TotalTilesPlayed         int
	TotalOpponentTilesPlayed int
}

// CalculateAndSaveStandings calculates final standings for all divisions in a season
// and marks players with their outcomes (promoted/relegated/stayed)
func (sm *StandingsManager) CalculateAndSaveStandings(
	ctx context.Context,
	seasonID uuid.UUID,
) error {
	// Get the season to get its promotion formula
	season, err := sm.store.GetSeason(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("failed to get season: %w", err)
	}

	// Get all divisions for this season
	divisions, err := sm.store.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("failed to get divisions: %w", err)
	}

	// Sort divisions by division number to know which is highest/lowest
	sort.Slice(divisions, func(i, j int) bool {
		return divisions[i].DivisionNumber < divisions[j].DivisionNumber
	})

	// Find the highest division number
	highestRegularDivision := int32(0)
	for _, div := range divisions {
		if div.DivisionNumber > highestRegularDivision {
			highestRegularDivision = div.DivisionNumber
		}
	}

	// Convert DB int32 to proto enum
	promotionFormula := pb.PromotionFormula(season.PromotionFormula)

	// Calculate standings for each division
	for _, division := range divisions {
		err := sm.calculateDivisionStandings(ctx, division, highestRegularDivision, promotionFormula)
		if err != nil {
			return fmt.Errorf("failed to calculate standings for division %d: %w", division.DivisionNumber, err)
		}
	}

	return nil
}

// calculateDivisionStandings calculates standings for a single division
func (sm *StandingsManager) calculateDivisionStandings(
	ctx context.Context,
	division models.LeagueDivision,
	highestRegularDivision int32,
	promotionFormula pb.PromotionFormula,
) error {
	// Get all registrations for this division
	registrations, err := sm.store.GetDivisionRegistrations(ctx, division.Uuid)
	if err != nil {
		return fmt.Errorf("failed to get division registrations: %w", err)
	}

	if len(registrations) == 0 {
		return nil // Nothing to do
	}

	// Get game results for this division
	gameResults, err := sm.store.GetDivisionGameResults(ctx, division.Uuid)
	if err != nil {
		return fmt.Errorf("failed to get game results: %w", err)
	}

	// Create a map to track player stats
	playerStats := make(map[int32]*PlayerStanding)
	for _, reg := range registrations {
		username := ""
		if reg.Username.Valid {
			username = reg.Username.String
		}
		playerStats[reg.UserID] = &PlayerStanding{
			UserID:      reg.UserID,
			DivisionID:  division.Uuid,
			Username:    username,
			Wins:        0,
			Losses:      0,
			Draws:       0,
			Spread:      0,
			GamesPlayed: 0,
		}
	}

	// Process each game result
	for _, game := range gameResults {
		// Get player IDs
		player0ID := game.Player0ID.Int32
		player1ID := game.Player1ID.Int32

		// Skip if players not in this division (shouldn't happen, but be safe)
		p0Stats, p0Exists := playerStats[player0ID]
		p1Stats, p1Exists := playerStats[player1ID]
		if !p0Exists || !p1Exists {
			continue
		}

		// Increment games played
		p0Stats.GamesPlayed++
		p1Stats.GamesPlayed++

		// Calculate spread
		p0Score := int(game.Player0Score)
		p1Score := int(game.Player1Score)
		p0Stats.Spread += (p0Score - p1Score)
		p1Stats.Spread += (p1Score - p0Score)

		// Determine outcome based on won column from game_players
		// won = true means won, false means lost, null means tie
		if game.Player0Won.Valid {
			if game.Player0Won.Bool {
				// Player 0 won
				p0Stats.Wins++
				p1Stats.Losses++
			} else {
				// Player 0 lost (player 1 won)
				p1Stats.Wins++
				p0Stats.Losses++
			}
		} else {
			// Draw (won is null for both players)
			p0Stats.Draws++
			p1Stats.Draws++
		}
	}

	// Convert map to slice
	standings := make([]PlayerStanding, 0, len(playerStats))
	for _, stats := range playerStats {
		standings = append(standings, *stats)
	}

	// Sort standings by points, spread, username
	sm.sortStandings(standings)

	// Assign ranks based on position (for marking promotion/relegation outcomes)
	for i := range standings {
		standings[i].Rank = i + 1
	}

	// Mark outcomes based on rank
	sm.markOutcomes(standings, division.DivisionNumber, highestRegularDivision, promotionFormula)

	// Calculate expected games per player based on division size
	expectedGames := CalculateExpectedGamesPerPlayer(len(registrations))

	// Save to database (rank is not saved - it's calculated on-demand when fetching)
	for _, standing := range standings {
		gamesRemaining := expectedGames - standing.GamesPlayed
		if gamesRemaining < 0 {
			gamesRemaining = 0 // Safety check
		}

		err := sm.store.UpsertStanding(ctx, models.UpsertStandingParams{
			DivisionID:     division.Uuid,
			UserID:         standing.UserID,
			Wins:           pgtype.Int4{Int32: int32(standing.Wins), Valid: true},
			Losses:         pgtype.Int4{Int32: int32(standing.Losses), Valid: true},
			Draws:          pgtype.Int4{Int32: int32(standing.Draws), Valid: true},
			Spread:         pgtype.Int4{Int32: int32(standing.Spread), Valid: true},
			GamesPlayed:    pgtype.Int4{Int32: int32(standing.GamesPlayed), Valid: true},
			GamesRemaining: pgtype.Int4{Int32: int32(gamesRemaining), Valid: true},
			Result:         pgtype.Int4{Int32: int32(standing.Outcome), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to save standing: %w", err)
		}
	}

	return nil
}

// SortStandingsByRank sorts standings by points (wins*2+draws) desc, spread desc, then username asc
// This is the canonical sorting function used everywhere for consistency
func SortStandingsByRank(standings []models.GetStandingsRow) {
	sort.Slice(standings, func(i, j int) bool {
		// First by points (descending) where points = wins*2 + draws
		pointsI := int(standings[i].Wins.Int32)*2 + int(standings[i].Draws.Int32)
		pointsJ := int(standings[j].Wins.Int32)*2 + int(standings[j].Draws.Int32)
		if pointsI != pointsJ {
			return pointsI > pointsJ
		}
		// Then by spread (descending)
		if standings[i].Spread.Int32 != standings[j].Spread.Int32 {
			return standings[i].Spread.Int32 > standings[j].Spread.Int32
		}
		// Finally by username (ascending) for deterministic tiebreaker
		usernameI := ""
		usernameJ := ""
		if standings[i].Username.Valid {
			usernameI = strings.ToLower(standings[i].Username.String)
		}
		if standings[j].Username.Valid {
			usernameJ = strings.ToLower(standings[j].Username.String)
		}
		return usernameI < usernameJ
	})
}

// sortStandings sorts PlayerStanding slices using the same logic as SortStandingsByRank
func (sm *StandingsManager) sortStandings(standings []PlayerStanding) {
	sort.Slice(standings, func(i, j int) bool {
		// First by points (descending) where points = wins*2 + draws
		pointsI := standings[i].Wins*2 + standings[i].Draws
		pointsJ := standings[j].Wins*2 + standings[j].Draws
		if pointsI != pointsJ {
			return pointsI > pointsJ
		}
		// Then by spread (descending)
		if standings[i].Spread != standings[j].Spread {
			return standings[i].Spread > standings[j].Spread
		}
		// Finally by username (ascending) for deterministic tiebreaker
		return strings.ToLower(standings[i].Username) < strings.ToLower(standings[j].Username)
	})
}

// markOutcomes assigns promotion/relegation/stayed outcomes based on rank
func (sm *StandingsManager) markOutcomes(
	standings []PlayerStanding,
	divisionNumber int32,
	highestRegularDivision int32,
	promotionFormula pb.PromotionFormula,
) {
	divSize := len(standings)
	if divSize == 0 {
		return
	}

	// Calculate number of promoted and relegated based on the season's formula
	promotionCount := CalculatePromotionCount(divSize, promotionFormula)
	relegationCount := promotionCount

	isHighestDivision := divisionNumber == 1
	isLowestDivision := divisionNumber >= highestRegularDivision

	for i := range standings {
		rank := i + 1

		if rank <= promotionCount && !isHighestDivision {
			// Top performers get promoted (unless already in Division 1)
			standings[i].Outcome = pb.StandingResult_RESULT_PROMOTED
		} else if rank > divSize-relegationCount && !isLowestDivision {
			// Bottom performers get relegated (unless in lowest division)
			standings[i].Outcome = pb.StandingResult_RESULT_RELEGATED
		} else {
			// Everyone else stays
			standings[i].Outcome = pb.StandingResult_RESULT_STAYED
		}
	}
}

// UpdateStandingsIncremental updates standings for a single completed game
// This is called immediately when a league game completes to provide real-time standings
func (sm *StandingsManager) UpdateStandingsIncremental(
	ctx context.Context,
	divisionID uuid.UUID,
	player0ID int32,
	player1ID int32,
	winnerIdx int32, // 0, 1, or -1 for tie
	p0Stats GameStats,
	p1Stats GameStats,
) error {
	// Use atomic operations to prevent race conditions when multiple games finish simultaneously
	// Note: player IDs are database IDs (int32), not UUID strings

	// Get division registrations to calculate expected games
	registrations, err := sm.store.GetDivisionRegistrations(ctx, divisionID)
	if err != nil {
		return fmt.Errorf("failed to get division registrations: %w", err)
	}

	// Calculate expected games per player based on division size
	expectedGames := CalculateExpectedGamesPerPlayer(len(registrations))
	// When a player's first game completes, games_played will be 1, so games_remaining should be expectedGames - 1
	initialGamesRemaining := expectedGames - 1

	// Calculate deltas for player 0
	var p0Wins, p0Losses, p0Draws int32
	if winnerIdx == 0 {
		p0Wins = 1
	} else if winnerIdx == 1 {
		p0Losses = 1
	} else {
		p0Draws = 1
	}
	p0Spread := p0Stats.Score - p1Stats.Score

	// Calculate deltas for player 1
	var p1Wins, p1Losses, p1Draws int32
	if winnerIdx == 1 {
		p1Wins = 1
	} else if winnerIdx == 0 {
		p1Losses = 1
	} else {
		p1Draws = 1
	}
	p1Spread := p1Stats.Score - p0Stats.Score

	// Convert timeout bool to int
	var p0Timeouts, p1Timeouts int32
	if p0Stats.TimedOut {
		p0Timeouts = 1
	}
	if p1Stats.TimedOut {
		p1Timeouts = 1
	}

	// Atomically increment player 0's standings
	err = sm.store.IncrementStandingsAtomic(ctx, models.IncrementStandingsAtomicParams{
		DivisionID:               divisionID,
		UserID:                   player0ID,
		Wins:                     pgtype.Int4{Int32: p0Wins, Valid: true},
		Losses:                   pgtype.Int4{Int32: p0Losses, Valid: true},
		Draws:                    pgtype.Int4{Int32: p0Draws, Valid: true},
		Spread:                   pgtype.Int4{Int32: p0Spread, Valid: true},
		GamesRemaining:           pgtype.Int4{Int32: int32(initialGamesRemaining), Valid: true},
		TotalScore:               pgtype.Int4{Int32: p0Stats.Score, Valid: true},
		TotalOpponentScore:       pgtype.Int4{Int32: p1Stats.Score, Valid: true},
		TotalBingos:              pgtype.Int4{Int32: p0Stats.Bingos, Valid: true},
		TotalOpponentBingos:      pgtype.Int4{Int32: p1Stats.Bingos, Valid: true},
		TotalTurns:               pgtype.Int4{Int32: p0Stats.Turns, Valid: true},
		HighTurn:                 pgtype.Int4{Int32: p0Stats.HighTurn, Valid: true},
		HighGame:                 pgtype.Int4{Int32: p0Stats.HighGame, Valid: true},
		Timeouts:                 pgtype.Int4{Int32: p0Timeouts, Valid: true},
		BlanksPlayed:             pgtype.Int4{Int32: p0Stats.BlanksPlayed, Valid: true},
		TotalTilesPlayed:         pgtype.Int4{Int32: p0Stats.TilesPlayed, Valid: true},
		TotalOpponentTilesPlayed: pgtype.Int4{Int32: p1Stats.TilesPlayed, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to increment standings for player %d: %w", player0ID, err)
	}

	// Atomically increment player 1's standings
	err = sm.store.IncrementStandingsAtomic(ctx, models.IncrementStandingsAtomicParams{
		DivisionID:               divisionID,
		UserID:                   player1ID,
		Wins:                     pgtype.Int4{Int32: p1Wins, Valid: true},
		Losses:                   pgtype.Int4{Int32: p1Losses, Valid: true},
		Draws:                    pgtype.Int4{Int32: p1Draws, Valid: true},
		Spread:                   pgtype.Int4{Int32: p1Spread, Valid: true},
		GamesRemaining:           pgtype.Int4{Int32: int32(initialGamesRemaining), Valid: true},
		TotalScore:               pgtype.Int4{Int32: p1Stats.Score, Valid: true},
		TotalOpponentScore:       pgtype.Int4{Int32: p0Stats.Score, Valid: true},
		TotalBingos:              pgtype.Int4{Int32: p1Stats.Bingos, Valid: true},
		TotalOpponentBingos:      pgtype.Int4{Int32: p0Stats.Bingos, Valid: true},
		TotalTurns:               pgtype.Int4{Int32: p1Stats.Turns, Valid: true},
		HighTurn:                 pgtype.Int4{Int32: p1Stats.HighTurn, Valid: true},
		HighGame:                 pgtype.Int4{Int32: p1Stats.HighGame, Valid: true},
		Timeouts:                 pgtype.Int4{Int32: p1Timeouts, Valid: true},
		BlanksPlayed:             pgtype.Int4{Int32: p1Stats.BlanksPlayed, Valid: true},
		TotalTilesPlayed:         pgtype.Int4{Int32: p1Stats.TilesPlayed, Valid: true},
		TotalOpponentTilesPlayed: pgtype.Int4{Int32: p0Stats.TilesPlayed, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to increment standings for player %d: %w", player1ID, err)
	}

	// Note: Rank is no longer stored in DB - it's calculated on-demand when fetching standings
	// by sorting by (wins*2 + draws) DESC, spread DESC, username ASC

	return nil
}

// RecalculateSeasonExtendedStats recalculates all extended stats (bingos, turns, blanks, etc.)
// for all divisions in a season. This is used to backfill stats for existing games.
func (sm *StandingsManager) RecalculateSeasonExtendedStats(
	ctx context.Context,
	seasonID uuid.UUID,
) error {
	// Get all divisions for this season
	divisions, err := sm.store.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("failed to get divisions: %w", err)
	}

	// Calculate stats for each division
	for _, division := range divisions {
		err := sm.recalculateDivisionExtendedStats(ctx, division.Uuid)
		if err != nil {
			return fmt.Errorf("failed to recalculate stats for division %d: %w", division.DivisionNumber, err)
		}
	}

	return nil
}

// recalculateDivisionExtendedStats recalculates extended stats for a single division
func (sm *StandingsManager) recalculateDivisionExtendedStats(
	ctx context.Context,
	divisionID uuid.UUID,
) error {
	// Get all registrations for this division
	registrations, err := sm.store.GetDivisionRegistrations(ctx, divisionID)
	if err != nil {
		return fmt.Errorf("failed to get division registrations: %w", err)
	}

	if len(registrations) == 0 {
		return nil // Nothing to do
	}

	// Get all games with stats for this division
	games, err := sm.store.GetDivisionGamesWithStats(ctx, divisionID)
	if err != nil {
		return fmt.Errorf("failed to get games with stats: %w", err)
	}

	// Create a map to track player extended stats
	playerStats := make(map[int32]*PlayerStanding)
	for _, reg := range registrations {
		playerStats[reg.UserID] = &PlayerStanding{
			UserID:     reg.UserID,
			DivisionID: divisionID,
		}
	}

	// Process each game
	for _, game := range games {
		player0ID := game.Player0ID.Int32
		player1ID := game.Player1ID.Int32

		p0Standing, p0Exists := playerStats[player0ID]
		p1Standing, p1Exists := playerStats[player1ID]
		if !p0Exists || !p1Exists {
			continue
		}

		// Extract scores
		p0Score := int(game.Player0Score)
		p1Score := int(game.Player1Score)

		// Accumulate scores
		p0Standing.TotalScore += p0Score
		p0Standing.TotalOpponentScore += p1Score
		p1Standing.TotalScore += p1Score
		p1Standing.TotalOpponentScore += p0Score

		// Track high game
		if p0Score > p0Standing.HighGame {
			p0Standing.HighGame = p0Score
		}
		if p1Score > p1Standing.HighGame {
			p1Standing.HighGame = p1Score
		}

		// Track timeouts (game_end_reason 1 = TIME)
		if game.GameEndReason == 1 {
			// The loser timed out - determine who lost
			if game.Player0Won.Valid {
				if game.Player0Won.Bool {
					// Player 0 won, so player 1 timed out
					p1Standing.Timeouts++
				} else {
					// Player 1 won, so player 0 timed out
					p0Standing.Timeouts++
				}
			}
		}

		// Extract stats from the game stats blob
		// PlayerOneData = player who went first = player index 0
		// PlayerTwoData = player who went second = player index 1
		stats := &game.Stats

		// Player 0 stats
		if bingoStat, ok := stats.PlayerOneData[entity.BINGOS_STAT]; ok {
			p0Standing.TotalBingos += bingoStat.Total
		}
		if bingoStat, ok := stats.PlayerTwoData[entity.BINGOS_STAT]; ok {
			p0Standing.TotalOpponentBingos += bingoStat.Total
		}
		if turnsStat, ok := stats.PlayerOneData[entity.TURNS_STAT]; ok {
			p0Standing.TotalTurns += turnsStat.Total
		}
		if highTurnStat, ok := stats.PlayerOneData[entity.HIGH_TURN_STAT]; ok {
			if highTurnStat.Total > p0Standing.HighTurn {
				p0Standing.HighTurn = highTurnStat.Total
			}
		}
		if tilesStat, ok := stats.PlayerOneData[entity.TILES_PLAYED_STAT]; ok {
			p0Standing.TotalTilesPlayed += tilesStat.Total
			if tilesStat.Subitems != nil {
				p0Standing.BlanksPlayed += tilesStat.Subitems["?"]
			}
		}
		// Opponent tiles played for p0
		if tilesStat, ok := stats.PlayerTwoData[entity.TILES_PLAYED_STAT]; ok {
			p0Standing.TotalOpponentTilesPlayed += tilesStat.Total
		}

		// Player 1 stats
		if bingoStat, ok := stats.PlayerTwoData[entity.BINGOS_STAT]; ok {
			p1Standing.TotalBingos += bingoStat.Total
		}
		if bingoStat, ok := stats.PlayerOneData[entity.BINGOS_STAT]; ok {
			p1Standing.TotalOpponentBingos += bingoStat.Total
		}
		if turnsStat, ok := stats.PlayerTwoData[entity.TURNS_STAT]; ok {
			p1Standing.TotalTurns += turnsStat.Total
		}
		if highTurnStat, ok := stats.PlayerTwoData[entity.HIGH_TURN_STAT]; ok {
			if highTurnStat.Total > p1Standing.HighTurn {
				p1Standing.HighTurn = highTurnStat.Total
			}
		}
		if tilesStat, ok := stats.PlayerTwoData[entity.TILES_PLAYED_STAT]; ok {
			p1Standing.TotalTilesPlayed += tilesStat.Total
			if tilesStat.Subitems != nil {
				p1Standing.BlanksPlayed += tilesStat.Subitems["?"]
			}
		}
		// Opponent tiles played for p1
		if tilesStat, ok := stats.PlayerOneData[entity.TILES_PLAYED_STAT]; ok {
			p1Standing.TotalOpponentTilesPlayed += tilesStat.Total
		}
	}

	// Update each player's standings with the extended stats
	for _, standing := range playerStats {
		// Get the existing standing to preserve wins/losses/draws/spread/games_played
		existingStanding, err := sm.store.GetPlayerStanding(ctx, models.GetPlayerStandingParams{
			DivisionID: divisionID,
			UserID:     standing.UserID,
		})
		if err != nil {
			// Player might not have any standings yet, skip
			continue
		}

		// Update with full data including extended stats
		err = sm.store.UpsertStanding(ctx, models.UpsertStandingParams{
			DivisionID:               divisionID,
			UserID:                   standing.UserID,
			Wins:                     existingStanding.Wins,
			Losses:                   existingStanding.Losses,
			Draws:                    existingStanding.Draws,
			Spread:                   existingStanding.Spread,
			GamesPlayed:              existingStanding.GamesPlayed,
			GamesRemaining:           existingStanding.GamesRemaining,
			Result:                   existingStanding.Result,
			TotalScore:               pgtype.Int4{Int32: int32(standing.TotalScore), Valid: true},
			TotalOpponentScore:       pgtype.Int4{Int32: int32(standing.TotalOpponentScore), Valid: true},
			TotalBingos:              pgtype.Int4{Int32: int32(standing.TotalBingos), Valid: true},
			TotalOpponentBingos:      pgtype.Int4{Int32: int32(standing.TotalOpponentBingos), Valid: true},
			TotalTurns:               pgtype.Int4{Int32: int32(standing.TotalTurns), Valid: true},
			HighTurn:                 pgtype.Int4{Int32: int32(standing.HighTurn), Valid: true},
			HighGame:                 pgtype.Int4{Int32: int32(standing.HighGame), Valid: true},
			Timeouts:                 pgtype.Int4{Int32: int32(standing.Timeouts), Valid: true},
			BlanksPlayed:             pgtype.Int4{Int32: int32(standing.BlanksPlayed), Valid: true},
			TotalTilesPlayed:         pgtype.Int4{Int32: int32(standing.TotalTilesPlayed), Valid: true},
			TotalOpponentTilesPlayed: pgtype.Int4{Int32: int32(standing.TotalOpponentTilesPlayed), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update standing for player %d: %w", standing.UserID, err)
		}
	}

	return nil
}
