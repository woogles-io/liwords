package league

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	// Priority score constants
	DivisionMultiplier = 100_000

	PriorityBonusRelegated       = 50_000
	PriorityBonusStayed          = 40_000
	PriorityBonusPromoted        = 30_000
	PriorityBonusHiatusReturning = 10_000
	PriorityBonusNew             = 5_000 // Lowest priority - new players placed naturally via rebalancing
	// Hiatus weight: 0.933^N (halves every ~10 seasons)
	HiatusWeightBase = 0.933

	// Division sizing
	MinimumFinalDivSize = 12
)

// RebalanceManager handles the rebalancing of divisions for a new season
type RebalanceManager struct {
	stores *stores.Stores
}

// NewRebalanceManager creates a new rebalance manager
func NewRebalanceManager(allStores *stores.Stores) *RebalanceManager {
	return &RebalanceManager{
		stores: allStores,
	}
}

// PlayerWithVirtualDiv represents a player with their assigned virtual division
type PlayerWithVirtualDiv struct {
	UserID               string // UUID string for external use
	UserDBID             int32  // Database ID for internal queries
	Username             string // Username for logging
	VirtualDivision      int32  // 1-indexed division
	PlacementStatus      ipc.PlacementStatus
	PreviousDivisionSize int
	PreviousRank         int32
	HiatusSeasons        int32
	Rating               int
	RegistrationRow      models.LeagueRegistration
}

// PlayerWithPriority extends PlayerWithVirtualDiv with priority score
type PlayerWithPriority struct {
	PlayerWithVirtualDiv
	PriorityScore float64
}

// RebalanceResult tracks the outcome of rebalancing
type RebalanceResult struct {
	DivisionsCreated int
	PlayersAssigned  int
	FinalDivMerged   bool
	VirtualDivisions map[string]int32 // UserID -> virtual division
	FinalDivisions   map[string]int32 // UserID -> real division
}

// RebalanceDivisions orchestrates the complete rebalancing process
func (rm *RebalanceManager) RebalanceDivisions(
	ctx context.Context,
	leagueID uuid.UUID,
	previousSeasonID uuid.UUID,
	newSeasonID uuid.UUID,
	newSeasonNumber int32,
	categorizedPlayers []CategorizedPlayer,
	idealDivisionSize int32,
) (*RebalanceResult, error) {
	result := &RebalanceResult{
		VirtualDivisions: make(map[string]int32),
		FinalDivisions:   make(map[string]int32),
	}

	if len(categorizedPlayers) == 0 {
		return result, nil
	}

	// Step 1: Update placement statuses (NEW/GRADUATED/HIATUS/etc.)
	err := rm.UpdatePlacementStatuses(ctx, leagueID, previousSeasonID, newSeasonID, newSeasonNumber, categorizedPlayers)
	if err != nil {
		return nil, fmt.Errorf("failed to update placement statuses: %w", err)
	}

	// Step 2: Assign virtual divisions to all players
	playersWithVirtualDivs, err := rm.AssignVirtualDivisions(ctx, leagueID, previousSeasonID, newSeasonID, categorizedPlayers)
	if err != nil {
		return nil, fmt.Errorf("failed to assign virtual divisions: %w", err)
	}

	// Store virtual divisions for result
	for _, p := range playersWithVirtualDivs {
		result.VirtualDivisions[p.UserID] = p.VirtualDivision
	}

	// Calculate number of virtual divisions
	numVirtualDivs := int32(0)
	for _, p := range playersWithVirtualDivs {
		if p.VirtualDivision > numVirtualDivs {
			numVirtualDivs = p.VirtualDivision
		}
	}

	// Step 3: Calculate priority scores
	playersWithPriority := rm.CalculatePriorityScores(playersWithVirtualDivs, numVirtualDivs, newSeasonNumber)

	// Step 4: Determine number of divisions to create
	numDivisions := int(math.Round(float64(len(playersWithPriority)) / float64(idealDivisionSize)))
	if numDivisions < 1 {
		numDivisions = 1
	}

	// Step 5: Create divisions and assign players
	err = rm.CreateDivisionsAndAssign(ctx, newSeasonID, playersWithPriority, numDivisions, idealDivisionSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create divisions and assign players: %w", err)
	}

	result.DivisionsCreated = numDivisions
	result.PlayersAssigned = len(playersWithPriority)

	// Step 6: Merge undersized final division if needed
	merged, err := rm.MergeUndersizedFinalDivision(ctx, newSeasonID, numDivisions)
	if err != nil {
		return nil, fmt.Errorf("failed to merge undersized final division: %w", err)
	}
	result.FinalDivMerged = merged
	if merged {
		result.DivisionsCreated--
	}

	// Get final division assignments for result and correct placement statuses
	for _, p := range playersWithPriority {
		reg, err := rm.stores.LeagueStore.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
			SeasonID: newSeasonID,
			UserID:   p.UserDBID,
		})
		if err == nil && reg.DivisionID.Valid {
			// Get division number
			div, err := rm.stores.LeagueStore.GetDivision(ctx, reg.DivisionID.Bytes)
			if err == nil {
				result.FinalDivisions[p.UserID] = div.DivisionNumber

				// Step 7: Correct placement status if actual movement differs from expected
				// VirtualDivision is where they "should" go based on outcome
				// FinalDivision is where they actually ended up
				correctedStatus := rm.correctPlacementStatus(p.PlacementStatus, p.VirtualDivision, div.DivisionNumber)
				if correctedStatus != p.PlacementStatus {
					err := rm.stores.LeagueStore.UpdatePlacementStatus(ctx, models.UpdatePlacementStatusParams{
						UserID:               p.UserDBID,
						PlacementStatus:      pgtype.Int4{Int32: int32(correctedStatus), Valid: true},
						PreviousDivisionRank: reg.PreviousDivisionRank,
						SeasonID:             newSeasonID,
					})
					if err != nil {
						log.Warn().Err(err).
							Int32("user_id", p.UserDBID).
							Str("old_status", p.PlacementStatus.String()).
							Str("new_status", correctedStatus.String()).
							Msg("failed to correct placement status")
					} else {
						log.Info().
							Str("username", p.Username).
							Int32("virtual_div", p.VirtualDivision).
							Int32("final_div", div.DivisionNumber).
							Str("old_status", p.PlacementStatus.String()).
							Str("new_status", correctedStatus.String()).
							Msg("corrected placement status based on actual division")
					}
				}
			}
		}
	}

	return result, nil
}

// correctPlacementStatus adjusts placement status if actual movement differs from expected
// For example, if a player was marked RELEGATED but ended up in the same division
// (because others didn't return), they should be marked as STAYED instead.
//
// VirtualDivision represents where they were *expected* to end up:
// - RELEGATED from Div 1 -> virtualDivision = 2
// - PROMOTED from Div 2 -> virtualDivision = 1
// - STAYED in Div 1 -> virtualDivision = 1
func (rm *RebalanceManager) correctPlacementStatus(
	originalStatus ipc.PlacementStatus,
	virtualDivision int32,
	finalDivision int32,
) ipc.PlacementStatus {
	// Correct placement statuses when final division differs from virtual division
	switch originalStatus {
	case ipc.PlacementStatus_PLACEMENT_RELEGATED:
		// Player was supposed to be relegated (virtualDivision is their target, which is lower than before)
		// If they ended up in a better (lower number) division than expected, mark as STAYED
		if finalDivision < virtualDivision {
			return ipc.PlacementStatus_PLACEMENT_STAYED
		}
	case ipc.PlacementStatus_PLACEMENT_PROMOTED:
		// Player was supposed to be promoted (virtualDivision is their target, which is higher than before)
		// If they ended up in a worse (higher number) division than expected, mark as RELEGATED
		if finalDivision > virtualDivision {
			return ipc.PlacementStatus_PLACEMENT_RELEGATED
		}
	case ipc.PlacementStatus_PLACEMENT_STAYED:
		// Player was supposed to stay in same division
		// If they ended up in a better (lower number) division, they got promoted to fill spots
		if finalDivision < virtualDivision {
			return ipc.PlacementStatus_PLACEMENT_PROMOTED
		}
		// If they ended up in a worse (higher number) division, they got relegated
		// This can happen when a league grows and a new division is created below them
		if finalDivision > virtualDivision {
			return ipc.PlacementStatus_PLACEMENT_RELEGATED
		}
	}
	return originalStatus
}

// UpdatePlacementStatuses sets placement_status for all players before rebalancing
// This includes NEW, GRADUATED, SHORT_HIATUS_RETURNING, LONG_HIATUS_RETURNING
// (PROMOTED/RELEGATED/STAYED are already set by MarkSeasonOutcomes at end of previous season)
// placementCache holds pre-fetched data to avoid N+1 queries
type playerStandingKey struct {
	divisionID uuid.UUID
	userID     int32
}

type placementCache struct {
	divisionStandings    map[uuid.UUID][]models.GetStandingsRow            // divisionID -> standings
	playerStandings      map[playerStandingKey]models.LeagueStanding       // (divisionID, userID) -> standing
	divisions            map[uuid.UUID]models.LeagueDivision               // divisionID -> division
	seasons              map[uuid.UUID]models.LeagueSeason                 // seasonID -> season
	allDivisionsBySeason map[uuid.UUID][]models.LeagueDivision             // seasonID -> divisions
	playerHistory        map[int32][]models.GetPlayerHistoriesForUsersRow // userID -> season history
}

// ComputePlayersWithPlacementStatus calculates placement status for all players WITHOUT writing to database
// This is used by both the real rebalancing flow and read-only testing
func (rm *RebalanceManager) ComputePlayersWithPlacementStatus(
	ctx context.Context,
	leagueID uuid.UUID,
	previousSeasonID uuid.UUID,
	newSeasonID uuid.UUID,
	newSeasonNumber int32,
	categorizedPlayers []CategorizedPlayer,
) ([]PlayerWithVirtualDiv, error) {
	// Pre-fetch all data we'll need to avoid N+1 queries
	cache := &placementCache{
		divisionStandings:    make(map[uuid.UUID][]models.GetStandingsRow),
		playerStandings:      make(map[playerStandingKey]models.LeagueStanding),
		divisions:            make(map[uuid.UUID]models.LeagueDivision),
		seasons:              make(map[uuid.UUID]models.LeagueSeason),
		allDivisionsBySeason: make(map[uuid.UUID][]models.LeagueDivision),
		playerHistory:        make(map[int32][]models.GetPlayerHistoriesForUsersRow),
	}

	// Collect user IDs for batch history fetch
	userIDs := make([]int32, 0, len(categorizedPlayers))
	for _, cp := range categorizedPlayers {
		userIDs = append(userIDs, cp.Registration.UserID)
	}

	// Pre-fetch player histories for registered players only (not entire league history)
	if len(userIDs) > 0 {
		histories, err := rm.stores.LeagueStore.GetPlayerHistoriesForUsers(ctx, models.GetPlayerHistoriesForUsersParams{
			UserIds:  userIDs,
			LeagueID: leagueID,
		})
		if err == nil {
			for _, h := range histories {
				cache.playerHistory[h.UserID] = append(cache.playerHistory[h.UserID], h)
			}
		}
	}

	// Pre-fetch all divisions and standings from previous season
	if previousSeasonID != uuid.Nil {
		prevDivisions, err := rm.stores.LeagueStore.GetDivisionsBySeason(ctx, previousSeasonID)
		if err == nil {
			cache.allDivisionsBySeason[previousSeasonID] = prevDivisions
			for _, div := range prevDivisions {
				cache.divisions[div.Uuid] = div
				// Pre-fetch standings for each division
				standings, err := rm.stores.LeagueStore.GetStandings(ctx, div.Uuid)
				if err == nil {
					cache.divisionStandings[div.Uuid] = standings
					// Also populate individual player standings map
					for _, s := range standings {
						key := playerStandingKey{divisionID: div.Uuid, userID: s.UserID}
						cache.playerStandings[key] = models.LeagueStanding{
							ID:                       s.ID,
							DivisionID:               s.DivisionID,
							UserID:                   s.UserID,
							Rank:                     pgtype.Int4{}, // Not included in GetStandingsRow
							Wins:                     s.Wins,
							Losses:                   s.Losses,
							Draws:                    s.Draws,
							Spread:                   s.Spread,
							GamesPlayed:              s.GamesPlayed,
							GamesRemaining:           s.GamesRemaining,
							Result:                   s.Result,
							UpdatedAt:                s.UpdatedAt,
							TotalScore:               s.TotalScore,
							TotalOpponentScore:       s.TotalOpponentScore,
							TotalBingos:              s.TotalBingos,
							TotalOpponentBingos:      s.TotalOpponentBingos,
							TotalTurns:               s.TotalTurns,
							HighTurn:                 s.HighTurn,
							HighGame:                 s.HighGame,
							Timeouts:                 s.Timeouts,
							BlanksPlayed:             s.BlanksPlayed,
							TotalTilesPlayed:         s.TotalTilesPlayed,
							TotalOpponentTilesPlayed: s.TotalOpponentTilesPlayed,
						}
					}
				}
			}
		}

		// Pre-fetch season info
		season, err := rm.stores.LeagueStore.GetSeason(ctx, previousSeasonID)
		if err == nil {
			cache.seasons[previousSeasonID] = season
		}
	}

	players := make([]PlayerWithVirtualDiv, 0, len(categorizedPlayers))

	for _, catPlayer := range categorizedPlayers {
		status, previousRank, seasonsAway, err := rm.computePlacementStatusForPlayer(ctx, leagueID, newSeasonID, newSeasonNumber, catPlayer, cache)
		if err != nil {
			return nil, err
		}

		reg := models.LeagueRegistration{
			ID:               catPlayer.Registration.ID,
			UserID:           catPlayer.Registration.UserID,
			SeasonID:         catPlayer.Registration.SeasonID,
			DivisionID:       catPlayer.Registration.DivisionID,
			RegistrationDate: catPlayer.Registration.RegistrationDate,
			FirstsCount:      catPlayer.Registration.FirstsCount,
			Status:           catPlayer.Registration.Status,
			CreatedAt:        catPlayer.Registration.CreatedAt,
			UpdatedAt:        catPlayer.Registration.UpdatedAt,
		}

		player := PlayerWithVirtualDiv{
			UserID:               catPlayer.Registration.UserUuid.String,
			UserDBID:             catPlayer.Registration.UserID,
			Username:             catPlayer.Registration.Username.String,
			PlacementStatus:      status,
			PreviousRank:         previousRank,
			HiatusSeasons:        seasonsAway,
			PreviousDivisionSize: 0,
			Rating:               int(catPlayer.Rating),
			RegistrationRow:      reg,
		}

		// Get previous division size for returning players from cache
		if status != ipc.PlacementStatus_PLACEMENT_NEW {
			if history, ok := cache.playerHistory[catPlayer.Registration.UserID]; ok && len(history) > 0 {
				for _, h := range history {
					if h.SeasonID != newSeasonID && h.DivisionID.Valid {
						// Get standings from cache
						if standings, ok := cache.divisionStandings[h.DivisionID.Bytes]; ok {
							player.PreviousDivisionSize = len(standings)
						}
						break
					}
				}
			}
		}

		players = append(players, player)
	}

	// Now calculate virtual divisions
	return rm.calculateVirtualDivisions(ctx, leagueID, previousSeasonID, players, cache)
}

// computePlacementStatusForPlayer determines placement status for a single player (pure computation)
func (rm *RebalanceManager) computePlacementStatusForPlayer(
	ctx context.Context,
	leagueID uuid.UUID,
	newSeasonID uuid.UUID,
	newSeasonNumber int32,
	player CategorizedPlayer,
	cache *placementCache,
) (status ipc.PlacementStatus, previousRank int32, seasonsAway int32, err error) {
	if player.Category == PlayerCategoryNew {
		return ipc.PlacementStatus_PLACEMENT_NEW, 0, 0, nil
	}

	// RETURNING player - get their history from cache
	history, ok := cache.playerHistory[player.Registration.UserID]
	if !ok || len(history) == 0 {
		return ipc.PlacementStatus_PLACEMENT_NEW, 0, 0, nil
	}

	// Find most recent season (not the new one)
	var lastSeason models.GetPlayerHistoriesForUsersRow
	found := false
	for _, h := range history {
		if h.SeasonID != newSeasonID {
			lastSeason = h
			found = true
			break
		}
	}

	if !found {
		return ipc.PlacementStatus_PLACEMENT_NEW, 0, 0, nil
	}

	// Calculate hiatus
	seasonsAway = newSeasonNumber - lastSeason.SeasonNumber - 1
	if lastSeason.PreviousDivisionRank.Valid {
		previousRank = lastSeason.PreviousDivisionRank.Int32
	}

	if seasonsAway > 0 {
		// Returning from hiatus
		if seasonsAway >= 1 && seasonsAway <= 3 {
			return ipc.PlacementStatus_PLACEMENT_SHORT_HIATUS_RETURNING, previousRank, seasonsAway, nil
		}
		return ipc.PlacementStatus_PLACEMENT_LONG_HIATUS_RETURNING, previousRank, seasonsAway, nil
	}

	// Consecutive play - check their standing result
	if lastSeason.DivisionID.Valid {
		// Get standing from cache instead of making a DB query
		key := playerStandingKey{divisionID: lastSeason.DivisionID.Bytes, userID: player.Registration.UserID}
		standing, ok := cache.playerStandings[key]
		if ok && standing.Result.Valid {
			// Use pre-marked result if available (season was closed)
			switch ipc.StandingResult(standing.Result.Int32) {
			case ipc.StandingResult_RESULT_PROMOTED:
				return ipc.PlacementStatus_PLACEMENT_PROMOTED, previousRank, 0, nil
			case ipc.StandingResult_RESULT_CHAMPION:
				return ipc.PlacementStatus_PLACEMENT_STAYED, previousRank, 0, nil
			case ipc.StandingResult_RESULT_RELEGATED:
				return ipc.PlacementStatus_PLACEMENT_RELEGATED, previousRank, 0, nil
			default:
				return ipc.PlacementStatus_PLACEMENT_STAYED, previousRank, 0, nil
			}
		} else if ok {
			// Result not set yet (season in progress) - compute outcome on-the-fly
			outcome, computedRank, computeErr := rm.computeOutcomeFromRank(ctx, lastSeason.DivisionID.Bytes, standing, lastSeason.SeasonID, cache)
			if computeErr != nil {
				return ipc.PlacementStatus_PLACEMENT_STAYED, previousRank, 0, fmt.Errorf("failed to compute outcome: %w", computeErr)
			}
			switch outcome {
			case ipc.StandingResult_RESULT_PROMOTED:
				return ipc.PlacementStatus_PLACEMENT_PROMOTED, computedRank, 0, nil
			case ipc.StandingResult_RESULT_CHAMPION:
				return ipc.PlacementStatus_PLACEMENT_STAYED, computedRank, 0, nil
			case ipc.StandingResult_RESULT_RELEGATED:
				return ipc.PlacementStatus_PLACEMENT_RELEGATED, computedRank, 0, nil
			default:
				return ipc.PlacementStatus_PLACEMENT_STAYED, computedRank, 0, nil
			}
		}
	}

	return ipc.PlacementStatus_PLACEMENT_STAYED, previousRank, 0, nil
}

// computeOutcomeFromRank computes a player's outcome (PROMOTED/RELEGATED/STAYED) and rank based on current standings
// This is used when testing with in-progress seasons where Result hasn't been marked yet
// Returns (outcome, rank, error)
func (rm *RebalanceManager) computeOutcomeFromRank(
	ctx context.Context,
	divisionID uuid.UUID,
	playerStanding models.LeagueStanding,
	seasonID uuid.UUID,
	cache *placementCache,
) (ipc.StandingResult, int32, error) {
	// Get all standings for this division from cache
	allStandings, ok := cache.divisionStandings[divisionID]
	if !ok {
		return ipc.StandingResult_RESULT_STAYED, 0, fmt.Errorf("division standings not found in cache")
	}

	// Sort to determine ranks
	SortStandingsByRank(allStandings)

	// Find this player's rank
	rank := int32(0)
	for i, s := range allStandings {
		if s.UserID == playerStanding.UserID {
			rank = int32(i + 1) // 1-indexed
			break
		}
	}

	if rank == 0 {
		return ipc.StandingResult_RESULT_STAYED, 0, fmt.Errorf("player not found in standings")
	}

	divSize := len(allStandings)

	// Get division from cache
	division, ok := cache.divisions[divisionID]
	if !ok {
		return ipc.StandingResult_RESULT_STAYED, 0, fmt.Errorf("division not found in cache")
	}

	// Get season from cache
	season, ok := cache.seasons[seasonID]
	if !ok {
		return ipc.StandingResult_RESULT_STAYED, 0, fmt.Errorf("season not found in cache")
	}

	// Get all divisions from cache
	allDivisions, ok := cache.allDivisionsBySeason[seasonID]
	if !ok {
		return ipc.StandingResult_RESULT_STAYED, 0, fmt.Errorf("divisions not found in cache")
	}

	highestDivisionNumber := int32(1)
	for _, div := range allDivisions {
		if div.DivisionNumber > highestDivisionNumber {
			highestDivisionNumber = div.DivisionNumber
		}
	}

	// Calculate promotion/relegation counts (same logic as in standings.go)
	promotionFormula := ipc.PromotionFormula(season.PromotionFormula)
	promotionCount := CalculatePromotionCount(divSize, promotionFormula)
	relegationCount := promotionCount

	isHighestDivision := division.DivisionNumber == 1
	isLowestDivision := division.DivisionNumber >= highestDivisionNumber

	// Apply same logic as markOutcomes in standings.go
	if rank == 1 && isHighestDivision {
		return ipc.StandingResult_RESULT_CHAMPION, rank, nil
	} else if int(rank) <= promotionCount && !isHighestDivision {
		return ipc.StandingResult_RESULT_PROMOTED, rank, nil
	} else if int(rank) > divSize-relegationCount && !isLowestDivision {
		return ipc.StandingResult_RESULT_RELEGATED, rank, nil
	}

	return ipc.StandingResult_RESULT_STAYED, rank, nil
}

func (rm *RebalanceManager) UpdatePlacementStatuses(
	ctx context.Context,
	leagueID uuid.UUID,
	previousSeasonID uuid.UUID,
	newSeasonID uuid.UUID,
	newSeasonNumber int32,
	categorizedPlayers []CategorizedPlayer,
) error {
	for _, player := range categorizedPlayers {
		if player.Category == PlayerCategoryNew {
			// Brand new player
			err := rm.stores.LeagueStore.UpdatePlacementStatusWithSeasonsAway(ctx, models.UpdatePlacementStatusWithSeasonsAwayParams{
				UserID:               player.Registration.UserID,
				PlacementStatus:      pgtype.Int4{Int32: int32(ipc.PlacementStatus_PLACEMENT_NEW), Valid: true},
				PreviousDivisionRank: pgtype.Int4{Int32: 0, Valid: false},
				SeasonsAway:          pgtype.Int4{Int32: 0, Valid: true},
				SeasonID:             newSeasonID,
			})
			if err != nil {
				return fmt.Errorf("failed to set NEW status for %d: %w", player.Registration.UserID, err)
			}
			continue
		}

		// RETURNING player - get their history
		history, err := rm.stores.LeagueStore.GetPlayerSeasonHistory(ctx, models.GetPlayerSeasonHistoryParams{
			UserID:   player.Registration.UserID,
			LeagueID: leagueID,
		})
		if err != nil {
			return fmt.Errorf("failed to get history for %d: %w", player.Registration.UserID, err)
		}

		// Find most recent season (not the new one)
		var lastSeason models.GetPlayerSeasonHistoryRow
		found := false
		for _, h := range history {
			if h.SeasonID != newSeasonID {
				lastSeason = h
				found = true
				break
			}
		}

		if !found {
			// No history found, treat as new
			err := rm.stores.LeagueStore.UpdatePlacementStatusWithSeasonsAway(ctx, models.UpdatePlacementStatusWithSeasonsAwayParams{
				UserID:               player.Registration.UserID,
				PlacementStatus:      pgtype.Int4{Int32: int32(ipc.PlacementStatus_PLACEMENT_NEW), Valid: true},
				PreviousDivisionRank: pgtype.Int4{Int32: 0, Valid: false},
				SeasonsAway:          pgtype.Int4{Int32: 0, Valid: true},
				SeasonID:             newSeasonID,
			})
			if err != nil {
				return fmt.Errorf("failed to set NEW status for %d: %w", player.Registration.UserID, err)
			}
			continue
		}

		// Calculate hiatus: newSeasonNumber - lastPlayedSeasonNumber - 1
		seasonsAway := newSeasonNumber - lastSeason.SeasonNumber - 1

		if seasonsAway > 0 {
			// Returning from hiatus
			var status ipc.PlacementStatus
			if seasonsAway >= 1 && seasonsAway <= 3 {
				status = ipc.PlacementStatus_PLACEMENT_SHORT_HIATUS_RETURNING
			} else {
				status = ipc.PlacementStatus_PLACEMENT_LONG_HIATUS_RETURNING
			}

			err := rm.stores.LeagueStore.UpdatePlacementStatusWithSeasonsAway(ctx, models.UpdatePlacementStatusWithSeasonsAwayParams{
				UserID:               player.Registration.UserID,
				PlacementStatus:      pgtype.Int4{Int32: int32(status), Valid: true},
				PreviousDivisionRank: lastSeason.PreviousDivisionRank,
				SeasonsAway:          pgtype.Int4{Int32: seasonsAway, Valid: true},
				SeasonID:             newSeasonID,
			})
			if err != nil {
				return fmt.Errorf("failed to set %s status for %d: %w", status, player.Registration.UserID, err)
			}
			continue
		}

		// Consecutive play - get outcome from standings (PROMOTED/RELEGATED/STAYED)
		// and convert to placement status for the new season
		var placementStatus ipc.PlacementStatus
		if lastSeason.DivisionID.Valid {
			// Get the player's standing from their previous division
			standing, err := rm.stores.LeagueStore.GetPlayerStanding(ctx, models.GetPlayerStandingParams{
				DivisionID: lastSeason.DivisionID.Bytes,
				UserID:     player.Registration.UserID,
			})
			if err != nil {
				// If no standing found (shouldn't happen), default to STAYED
				log.Warn().Err(err).Int32("user_id", player.Registration.UserID).Msg("no standing found, defaulting to STAYED")
				placementStatus = ipc.PlacementStatus_PLACEMENT_STAYED
			} else if standing.Result.Valid {
				// Convert StandingResult to PlacementStatus
				switch ipc.StandingResult(standing.Result.Int32) {
				case ipc.StandingResult_RESULT_PROMOTED:
					placementStatus = ipc.PlacementStatus_PLACEMENT_PROMOTED
				case ipc.StandingResult_RESULT_CHAMPION:
					// Champions stay in Division 1 - they can't be promoted any higher
					placementStatus = ipc.PlacementStatus_PLACEMENT_STAYED
				case ipc.StandingResult_RESULT_RELEGATED:
					placementStatus = ipc.PlacementStatus_PLACEMENT_RELEGATED
				default:
					placementStatus = ipc.PlacementStatus_PLACEMENT_STAYED
				}
			} else {
				placementStatus = ipc.PlacementStatus_PLACEMENT_STAYED
			}
		} else {
			placementStatus = ipc.PlacementStatus_PLACEMENT_STAYED
		}

		err = rm.stores.LeagueStore.UpdatePlacementStatusWithSeasonsAway(ctx, models.UpdatePlacementStatusWithSeasonsAwayParams{
			UserID:               player.Registration.UserID,
			PlacementStatus:      pgtype.Int4{Int32: int32(placementStatus), Valid: true},
			PreviousDivisionRank: lastSeason.PreviousDivisionRank,
			SeasonsAway:          pgtype.Int4{Int32: 0, Valid: true},
			SeasonID:             newSeasonID,
		})
		if err != nil {
			return fmt.Errorf("failed to set placement status for %d: %w", player.Registration.UserID, err)
		}
	}

	return nil
}

// AssignVirtualDivisions assigns virtual divisions to all players based on their outcomes
func (rm *RebalanceManager) AssignVirtualDivisions(
	ctx context.Context,
	leagueID uuid.UUID,
	previousSeasonID uuid.UUID,
	newSeasonID uuid.UUID,
	categorizedPlayers []CategorizedPlayer,
) ([]PlayerWithVirtualDiv, error) {
	players := []PlayerWithVirtualDiv{}

	// Fetch all registrations for the season at once to avoid N+1 queries
	allRegistrations, err := rm.stores.LeagueStore.GetSeasonRegistrations(ctx, newSeasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get season registrations: %w", err)
	}

	// Pre-fetch player histories and division standings
	cache := &placementCache{
		playerHistory:     make(map[int32][]models.GetPlayerHistoriesForUsersRow),
		divisionStandings: make(map[uuid.UUID][]models.GetStandingsRow),
	}

	userIDs := make([]int32, 0, len(categorizedPlayers))
	for _, cp := range categorizedPlayers {
		userIDs = append(userIDs, cp.Registration.UserID)
	}

	if len(userIDs) > 0 {
		histories, err := rm.stores.LeagueStore.GetPlayerHistoriesForUsers(ctx, models.GetPlayerHistoriesForUsersParams{
			UserIds:  userIDs,
			LeagueID: leagueID,
		})
		if err == nil {
			for _, h := range histories {
				cache.playerHistory[h.UserID] = append(cache.playerHistory[h.UserID], h)
			}
		}
	}

	// Pre-fetch division standings from previous season
	if previousSeasonID != uuid.Nil {
		prevDivisions, err := rm.stores.LeagueStore.GetDivisionsBySeason(ctx, previousSeasonID)
		if err == nil {
			for _, div := range prevDivisions {
				standings, err := rm.stores.LeagueStore.GetStandings(ctx, div.Uuid)
				if err == nil {
					cache.divisionStandings[div.Uuid] = standings
				}
			}
		}
	}

	// Create a map for O(1) lookup by user_id
	registrationMap := make(map[int32]models.GetSeasonRegistrationsRow)
	for _, reg := range allRegistrations {
		registrationMap[reg.UserID] = reg
	}

	// Get registrations to access updated placement_status
	for _, catPlayer := range categorizedPlayers {
		regRow, exists := registrationMap[catPlayer.Registration.UserID]
		if !exists {
			return nil, fmt.Errorf("registration not found for user %d in season", catPlayer.Registration.UserID)
		}

		// Convert GetSeasonRegistrationsRow to LeagueRegistration
		reg := models.LeagueRegistration{
			ID:                   regRow.ID,
			UserID:               regRow.UserID,
			SeasonID:             regRow.SeasonID,
			DivisionID:           regRow.DivisionID,
			RegistrationDate:     regRow.RegistrationDate,
			FirstsCount:          regRow.FirstsCount,
			Status:               regRow.Status,
			PlacementStatus:      regRow.PlacementStatus,
			PreviousDivisionRank: regRow.PreviousDivisionRank,
			SeasonsAway:          regRow.SeasonsAway,
			CreatedAt:            regRow.CreatedAt,
			UpdatedAt:            regRow.UpdatedAt,
		}

		status := ipc.PlacementStatus_PLACEMENT_NONE
		if reg.PlacementStatus.Valid {
			status = ipc.PlacementStatus(reg.PlacementStatus.Int32)
		}

		player := PlayerWithVirtualDiv{
			UserID:               catPlayer.Registration.UserUuid.String, // UUID string from JOIN
			UserDBID:             catPlayer.Registration.UserID,          // Database ID
			Username:             catPlayer.Registration.Username.String, // Username from JOIN
			PlacementStatus:      status,
			PreviousRank:         0,
			HiatusSeasons:        0,
			PreviousDivisionSize: 0,
			Rating:               int(catPlayer.Rating),
			RegistrationRow:      reg,
		}

		if reg.PreviousDivisionRank.Valid {
			player.PreviousRank = reg.PreviousDivisionRank.Int32
		}
		if reg.SeasonsAway.Valid {
			player.HiatusSeasons = reg.SeasonsAway.Int32
		}

		// Get previous division size from cache
		if status != ipc.PlacementStatus_PLACEMENT_NEW {
			if history, ok := cache.playerHistory[catPlayer.Registration.UserID]; ok && len(history) > 0 {
				for _, h := range history {
					if h.SeasonID != newSeasonID && h.DivisionID.Valid {
						// Get standings count from cache
						if standings, ok := cache.divisionStandings[h.DivisionID.Bytes]; ok {
							player.PreviousDivisionSize = len(standings)
						}
						break
					}
				}
			}
		}

		players = append(players, player)
	}

	// Now assign virtual divisions based on status
	return rm.calculateVirtualDivisions(ctx, leagueID, previousSeasonID, players, cache)
}

// calculateVirtualDivisions determines virtual division for each player
func (rm *RebalanceManager) calculateVirtualDivisions(
	ctx context.Context,
	leagueID uuid.UUID,
	previousSeasonID uuid.UUID,
	players []PlayerWithVirtualDiv,
	cache *placementCache,
) ([]PlayerWithVirtualDiv, error) {
	// Get previous season divisions to determine virtual division structure
	prevDivisions, err := rm.stores.LeagueStore.GetDivisionsBySeason(ctx, previousSeasonID)
	if err != nil && previousSeasonID != uuid.Nil {
		return nil, fmt.Errorf("failed to get previous season divisions: %w", err)
	}

	// Sort divisions by division number
	sort.Slice(prevDivisions, func(i, j int) bool {
		return prevDivisions[i].DivisionNumber < prevDivisions[j].DivisionNumber
	})

	highestPrevDivNumber := int32(0)
	if len(prevDivisions) > 0 {
		highestPrevDivNumber = prevDivisions[len(prevDivisions)-1].DivisionNumber
	}

	// Assign virtual divisions for all players based on their status
	result := make([]PlayerWithVirtualDiv, len(players))

	for i, p := range players {
		// Get their previous division number from cache
		history, ok := cache.playerHistory[p.UserDBID]
		if !ok {
			// No history, default to division 1
			history = nil
		}

		prevDivNumber := int32(1) // Default
		for _, h := range history {
			if h.DivisionID.Valid {
				// Look up the division to get its number
				div, err := rm.stores.LeagueStore.GetDivision(ctx, h.DivisionID.Bytes)
				if err == nil {
					prevDivNumber = div.DivisionNumber
				}
				break
			}
		}

		// Copy player data to result
		result[i] = p

		// Apply outcome to determine virtual division
		switch p.PlacementStatus {
		case ipc.PlacementStatus_PLACEMENT_PROMOTED:
			result[i].VirtualDivision = prevDivNumber - 1
			if result[i].VirtualDivision < 1 {
				result[i].VirtualDivision = 1
			}
		case ipc.PlacementStatus_PLACEMENT_RELEGATED:
			result[i].VirtualDivision = prevDivNumber + 1
		case ipc.PlacementStatus_PLACEMENT_NEW:
			// NEW players: assign target based on rating match with existing divisions
			// This prevents overrated rookies from dominating bottom divisions
			// and helps rookies find their correct level faster
			if highestPrevDivNumber > 0 && p.Rating > 0 {
				// Calculate target based on rating similarity to returning players
				result[i].VirtualDivision = rm.calculateRatingBasedTarget(
					ctx,
					p.Rating,
					result[:i], // Already-processed returning players
					highestPrevDivNumber,
				)
			} else if highestPrevDivNumber > 0 {
				// Unrated NEW player (rating = 0) goes to bottom division
				result[i].VirtualDivision = highestPrevDivNumber
			} else {
				// First season, no previous divisions
				result[i].VirtualDivision = 1
			}
		case ipc.PlacementStatus_PLACEMENT_STAYED, ipc.PlacementStatus_PLACEMENT_SHORT_HIATUS_RETURNING, ipc.PlacementStatus_PLACEMENT_LONG_HIATUS_RETURNING:
			result[i].VirtualDivision = prevDivNumber
		default:
			result[i].VirtualDivision = prevDivNumber
		}
	}

	return result, nil
}

// calculateRatingBasedTarget assigns a target division for NEW players based on their rating
// compared to the average ratings of returning players in each division
func (rm *RebalanceManager) calculateRatingBasedTarget(
	ctx context.Context,
	newPlayerRating int,
	returningPlayers []PlayerWithVirtualDiv,
	maxDivision int32,
) int32 {
	// Calculate average rating per virtual division from returning players
	divisionRatingSums := make(map[int32]int)
	divisionCounts := make(map[int32]int)

	for _, player := range returningPlayers {
		if player.Rating > 0 {
			divisionRatingSums[player.VirtualDivision] += player.Rating
			divisionCounts[player.VirtualDivision]++
		}
	}

	// Calculate averages
	divisionAverages := make(map[int32]float64)
	for div := int32(1); div <= maxDivision; div++ {
		if divisionCounts[div] > 0 {
			divisionAverages[div] = float64(divisionRatingSums[div]) / float64(divisionCounts[div])
		}
	}

	// If no returning players have ratings, fall back to bottom division
	if len(divisionAverages) == 0 {
		return maxDivision
	}

	// Find division with closest average rating
	// Start from Division 2 unless there's only 1 division (keep Div 1 exclusive)
	startDiv := int32(2)
	if maxDivision == 1 {
		startDiv = 1
	}

	closestDiv := maxDivision
	smallestDiff := math.Inf(1)

	for div := startDiv; div <= maxDivision; div++ {
		if avg, exists := divisionAverages[div]; exists {
			diff := math.Abs(float64(newPlayerRating) - avg)
			if diff < smallestDiff {
				smallestDiff = diff
				closestDiv = div
			}
		}
	}

	// If we didn't find any division with ratings (very unlikely), default to Div 2 or max
	if math.IsInf(smallestDiff, 1) {
		if maxDivision >= 2 {
			closestDiv = 2
		} else {
			closestDiv = maxDivision
		}
	}

	log.Debug().
		Int("newPlayerRating", newPlayerRating).
		Int32("targetDivision", closestDiv).
		Int32("maxDivision", maxDivision).
		Interface("divisionAverages", divisionAverages).
		Msg("Calculated rating-based target for NEW player (Div 1 excluded)")

	return closestDiv
}

// CalculatePriorityScores calculates priority scores for all players
func (rm *RebalanceManager) CalculatePriorityScores(
	players []PlayerWithVirtualDiv,
	numVirtualDivs int32,
	seasonNumber int32,
) []PlayerWithPriority {
	result := make([]PlayerWithPriority, len(players))

	for i, p := range players {
		// Calculate priority bonus based on status
		var priorityBonus int64
		switch p.PlacementStatus {
		case ipc.PlacementStatus_PLACEMENT_STAYED:
			priorityBonus = PriorityBonusStayed
		case ipc.PlacementStatus_PLACEMENT_PROMOTED:
			priorityBonus = PriorityBonusPromoted
		case ipc.PlacementStatus_PLACEMENT_RELEGATED:
			priorityBonus = PriorityBonusRelegated
		case ipc.PlacementStatus_PLACEMENT_SHORT_HIATUS_RETURNING, ipc.PlacementStatus_PLACEMENT_LONG_HIATUS_RETURNING:
			priorityBonus = PriorityBonusHiatusReturning
		case ipc.PlacementStatus_PLACEMENT_NEW:
			priorityBonus = PriorityBonusNew
		default:
			priorityBonus = 0
		}

		// Calculate weight based on hiatus
		weight := 1.0
		if p.HiatusSeasons > 0 {
			weight = math.Pow(HiatusWeightBase, float64(p.HiatusSeasons))
		}

		// Calculate rank component (num_players_in_previous_division - our_rank_in_previous_division)
		rankComponent := int64(0)
		if p.PreviousDivisionSize > 0 && p.PreviousRank > 0 {
			rankComponent = int64(p.PreviousDivisionSize) - int64(p.PreviousRank)
		}

		// Calculate priority score
		// score = ((DivMultiplier * (num_virtual_divs - our_virtual_div)) + PriorityBonus + rankComponent) * Weight
		score := float64(
			(int64(DivisionMultiplier)*int64(numVirtualDivs-p.VirtualDivision+1))+
				priorityBonus+
				rankComponent,
		) * weight

		// For NEW players, add their rating to prioritize by skill.
		if p.PlacementStatus == ipc.PlacementStatus_PLACEMENT_NEW {
			score += float64(p.Rating)
		}

		result[i] = PlayerWithPriority{
			PlayerWithVirtualDiv: p,
			PriorityScore:        score,
		}
	}

	// Sort by priority score (descending - highest priority first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].PriorityScore > result[j].PriorityScore
	})

	return result
}

// CreateDivisionsAndAssign creates real divisions and assigns players sequentially
func (rm *RebalanceManager) CreateDivisionsAndAssign(
	ctx context.Context,
	seasonID uuid.UUID,
	playersWithPriority []PlayerWithPriority,
	numDivisions int,
	idealDivisionSize int32,
) error {
	// Create divisions 1, 2, 3, ..., numDivisions
	createdDivisions := []models.LeagueDivision{}
	manualMgr := NewManualDivisionManager(rm.stores)

	for divNum := 1; divNum <= numDivisions; divNum++ {
		div, err := manualMgr.CreateDivision(ctx, seasonID, int32(divNum), "")
		if err != nil {
			return fmt.Errorf("failed to create division %d: %w", divNum, err)
		}
		createdDivisions = append(createdDivisions, div)
	}

	// Assign players sequentially based on ideal division size
	for i, player := range playersWithPriority {
		// Determine which division this player goes to
		divIndex := i / int(idealDivisionSize)
		if divIndex >= len(createdDivisions) {
			divIndex = len(createdDivisions) - 1 // Put overflow in last division
		}

		targetDiv := createdDivisions[divIndex]

		// Log player placement with priority score details
		log.Info().
			Str("username", player.Username).
			Int("assignmentOrder", i+1).
			Int32("divisionNumber", targetDiv.DivisionNumber).
			Float64("priorityScore", player.PriorityScore).
			Int32("virtualDivision", player.VirtualDivision).
			Str("placementStatus", player.PlacementStatus.String()).
			Int32("previousRank", player.PreviousRank).
			Int("previousDivisionSize", player.PreviousDivisionSize).
			Int32("hiatusSeasons", player.HiatusSeasons).
			Int("rating", player.Rating).
			Msg("Player division assignment")

		// Assign player to this division
		err := rm.stores.LeagueStore.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      player.UserDBID,
			SeasonID:    seasonID,
			DivisionID:  pgtype.UUID{Bytes: targetDiv.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to assign player %s to division %d: %w", player.UserID, targetDiv.DivisionNumber, err)
		}
	}

	return nil
}

// MergeUndersizedFinalDivision merges the final division into the second-to-last if it's too small
func (rm *RebalanceManager) MergeUndersizedFinalDivision(
	ctx context.Context,
	seasonID uuid.UUID,
	numDivisions int,
) (bool, error) {
	if numDivisions <= 1 {
		return false, nil // Can't merge if only one division
	}

	// Get all divisions
	divisions, err := rm.stores.LeagueStore.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return false, fmt.Errorf("failed to get divisions: %w", err)
	}

	// Sort divisions by number
	sort.Slice(divisions, func(i, j int) bool {
		return divisions[i].DivisionNumber < divisions[j].DivisionNumber
	})

	if len(divisions) <= 1 {
		return false, nil
	}

	// Get last division
	lastDiv := divisions[len(divisions)-1]
	secondToLast := divisions[len(divisions)-2]

	// Count players in last division
	lastDivPlayers, err := rm.stores.LeagueStore.GetDivisionRegistrations(ctx, lastDiv.Uuid)
	if err != nil {
		return false, fmt.Errorf("failed to get last division players: %w", err)
	}

	if len(lastDivPlayers) < MinimumFinalDivSize {
		// Merge into second-to-last division
		manualMgr := NewManualDivisionManager(rm.stores)
		_, err := manualMgr.MergeDivisions(ctx, seasonID, secondToLast.Uuid, lastDiv.Uuid)
		if err != nil {
			return false, fmt.Errorf("failed to merge divisions: %w", err)
		}
		return true, nil
	}

	return false, nil
}
