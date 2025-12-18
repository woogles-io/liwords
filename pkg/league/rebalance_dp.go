package league

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// DP Algorithm Constants
const (
	// Division size bounds
	AbsoluteMinDivSize = 8   // Emergency minimum for very small leagues
	AbsoluteMaxDivSize = 30  // Emergency maximum for very large leagues
	IdealMinDivSize    = 12  // Preferred minimum
	IdealMaxDivSize    = 18  // Preferred maximum

	// Cost function weights
	WeightSize             = 10.0      // Base size deviation penalty
	WeightSizeOutOfRange   = 50.0      // Extra penalty per division outside [12-18]
	WeightNewDeviation     = 100.0     // NEW player deviation from rating-based target
	WeightLuckyPromotion   = 50.0      // Promoting safe player to fill gap (acceptable)
	WeightBadKeep          = 500.0     // Keeping relegated player up (bad)
	WeightForcedRelegation = 1000.0    // Relegating safe player (very bad)
	WeightDoubleRelegation = 100000.0  // Double-relegating (nuclear)
)

// DPPlayer represents a player with all info needed for DP cost calculations
type DPPlayer struct {
	UserID               string
	UserDBID             int32
	Username             string
	TargetDivision       int32 // Where they should go based on outcome
	                            // For RETURNING players: their earned target (e.g., promoted to div 2)
	                            // For NEW players: set to bottom division of PREVIOUS season (used for sorting only)
	                            //   The actual cost function recalculates their target dynamically based on totalDivs
	PlacementStatus      ipc.PlacementStatus
	PreviousDivisionSize int
	PreviousRank         int32
	SeasonsAway          int32
	Rating               int
	WasRelegated         bool // True if status == PLACEMENT_RELEGATED
	PriorityScore        float64 // Calculated priority score (used for sorting before DP)
	RegistrationRow      models.LeagueRegistration
}

// DPSolution represents a complete division assignment solution
type DPSolution struct {
	NumDivisions  int
	TotalCost     float64
	Divisions     []DPDivision
	PlayerToDivision map[string]int32 // UserID -> division number
}

// DPDivision represents a single division in the solution
type DPDivision struct {
	DivisionNumber int32
	Players        []DPPlayer
	Cost           float64
}

// SolveDivisionsDP uses dynamic programming to find the optimal division configuration
func (rm *RebalanceManager) SolveDivisionsDP(
	ctx context.Context,
	players []DPPlayer,
	idealDivisionSize int32,
) (*DPSolution, error) {
	N := len(players)

	if N == 0 {
		return &DPSolution{
			NumDivisions:     0,
			TotalCost:        0,
			Divisions:        []DPDivision{},
			PlayerToDivision: make(map[string]int32),
		}, nil
	}

	// Edge case: very small league - just one division
	if N < AbsoluteMinDivSize {
		div := DPDivision{
			DivisionNumber: 1,
			Players:        players,
			Cost:           rm.calculateDivisionCost(players, 1, 1, int(idealDivisionSize)),
		}

		playerToDivision := make(map[string]int32)
		for _, p := range players {
			playerToDivision[p.UserID] = 1
		}

		return &DPSolution{
			NumDivisions:     1,
			TotalCost:        div.Cost,
			Divisions:        []DPDivision{div},
			PlayerToDivision: playerToDivision,
		}, nil
	}

	// Sort players by priority score before DP (critical for correct assignment)
	// Players must already have PriorityScore calculated before calling this function
	sortPlayersByPriorityScore(players)

	// Estimate viable division range
	estimated := int(math.Round(float64(N) / float64(idealDivisionSize)))
	minDivs := max(1, estimated-2)
	maxDivs := estimated + 2

	log.Info().
		Int("totalPlayers", N).
		Int("estimatedDivisions", estimated).
		Int("minDivs", minDivs).
		Int("maxDivs", maxDivs).
		Int32("idealSize", idealDivisionSize).
		Msg("Starting DP division solver")

	var bestSolution *DPSolution
	minGlobalCost := math.Inf(1)

	// Try different division counts
	for K := minDivs; K <= maxDivs; K++ {
		solution, cost := rm.solveForKDivisions(players, K, N, int(idealDivisionSize))

		log.Debug().
			Int("K", K).
			Float64("cost", cost).
			Msg("Evaluated division count")

		if cost < minGlobalCost {
			minGlobalCost = cost
			bestSolution = solution
		}
	}

	if bestSolution == nil {
		return nil, fmt.Errorf("DP solver failed to find any valid solution")
	}

	log.Info().
		Int("numDivisions", bestSolution.NumDivisions).
		Float64("totalCost", bestSolution.TotalCost).
		Msg("DP solver found optimal solution")

	return bestSolution, nil
}

// solveForKDivisions solves the DP problem for exactly K divisions
func (rm *RebalanceManager) solveForKDivisions(
	players []DPPlayer,
	K, N int,
	idealSize int,
) (*DPSolution, float64) {
	// Initialize DP table: dp[k][i] = min cost to organize first i players into k divisions
	dp := make([][]float64, K+1)
	cuts := make([][]int, K+1) // Track where divisions start for backtracking

	for k := 0; k <= K; k++ {
		dp[k] = make([]float64, N+1)
		cuts[k] = make([]int, N+1)
		for i := 0; i <= N; i++ {
			dp[k][i] = math.Inf(1)
		}
	}

	dp[0][0] = 0

	// Fill DP table
	for k := 1; k <= K; k++ {
		for i := 1; i <= N; i++ {
			// Try different division sizes
			for size := AbsoluteMinDivSize; size <= min(AbsoluteMaxDivSize, i); size++ {
				prevI := i - size
				if prevI < 0 {
					continue
				}

				// Extract players for this potential division
				divPlayers := players[prevI:i]

				// Calculate cost for this specific division
				divCost := rm.calculateDivisionCost(divPlayers, k, K, idealSize)

				totalCost := dp[k-1][prevI] + divCost

				if totalCost < dp[k][i] {
					dp[k][i] = totalCost
					cuts[k][i] = prevI
				}
			}
		}
	}

	// Reconstruct solution by backtracking through cuts
	solution := rm.reconstructSolution(players, cuts, K, N, idealSize)
	solution.TotalCost = dp[K][N]

	return solution, dp[K][N]
}

// calculateDivisionCost calculates the total cost for a specific division
func (rm *RebalanceManager) calculateDivisionCost(
	players []DPPlayer,
	divNum int,
	totalDivs int,
	idealSize int,
) float64 {
	cost := 0.0
	size := len(players)

	// 1. Size penalty (quadratic with extra penalty outside ideal range)
	cost += calculateSizePenalty(size, idealSize)

	// 2. Individual player placement penalties
	for _, p := range players {
		if p.PlacementStatus == ipc.PlacementStatus_PLACEMENT_NEW {
			// NEW player - use simple deviation penalty from their target
			// (target is calculated based on rating in calculateVirtualDivisions)
			cost += calculateNewPlayerPenalty(p, divNum)
		} else {
			// RETURNING player - use earned outcome penalties
			cost += calculatePlacementPenalty(p, divNum)
		}
	}

	return cost
}

// calculateSizePenalty calculates the penalty for deviating from ideal division size
func calculateSizePenalty(size, idealSize int) float64 {
	diff := size - idealSize
	basePenalty := WeightSize * float64(diff*diff)

	// Extra penalty for being outside ideal range [12-18]
	if size < IdealMinDivSize {
		extraPenalty := float64(IdealMinDivSize-size) * WeightSizeOutOfRange
		basePenalty += extraPenalty
	}
	if size > IdealMaxDivSize {
		extraPenalty := float64(size-IdealMaxDivSize) * WeightSizeOutOfRange
		basePenalty += extraPenalty
	}

	return basePenalty
}

// calculatePlacementPenalty calculates penalties for RETURNING players based on their outcomes
func calculatePlacementPenalty(p DPPlayer, divNum int) float64 {
	diff := int32(divNum) - p.TargetDivision

	if diff == 0 {
		return 0 // Perfect placement
	}

	if diff > 0 {
		// Player pushed DOWN from target division
		if p.WasRelegated {
			// DOUBLE RELEGATION - nuclear penalty
			return WeightDoubleRelegation * float64(diff)
		}

		// FORCED RELEGATION - with hiatus decay
		penalty := WeightForcedRelegation
		if p.SeasonsAway > 0 {
			// Decay: base / (seasons_away + 1)
			decayFactor := 1.0 / float64(p.SeasonsAway+1)
			penalty = penalty * decayFactor
		}
		return penalty * float64(diff)
	}

	// Player pulled UP from target division
	if p.WasRelegated {
		// BAD KEEP - keeping relegated player up
		return WeightBadKeep * float64(-diff)
	}

	// LUCKY PROMOTION - safe player bumped up to fill gap (acceptable)
	return WeightLuckyPromotion * float64(-diff)
}

// calculateNewPlayerPenalty calculates simple deviation penalty for NEW players
func calculateNewPlayerPenalty(p DPPlayer, divNum int) float64 {
	diff := abs(divNum - int(p.TargetDivision))
	return WeightNewDeviation * float64(diff)
}

// ConvertToDPPlayers converts PlayerWithPriority to DPPlayer format
func ConvertToDPPlayers(playersWithPriority []PlayerWithPriority) []DPPlayer {
	dpPlayers := make([]DPPlayer, len(playersWithPriority))

	for i, p := range playersWithPriority {
		dpPlayers[i] = DPPlayer{
			UserID:               p.UserID,
			UserDBID:             p.UserDBID,
			Username:             p.Username,
			TargetDivision:       p.VirtualDivision,
			PlacementStatus:      p.PlacementStatus,
			PreviousDivisionSize: p.PreviousDivisionSize,
			PreviousRank:         p.PreviousRank,
			SeasonsAway:          p.HiatusSeasons,
			Rating:               p.Rating,
			WasRelegated:         p.PlacementStatus == ipc.PlacementStatus_PLACEMENT_RELEGATED,
			PriorityScore:        p.PriorityScore,
			RegistrationRow:      p.RegistrationRow,
		}
	}

	return dpPlayers
}

// sortPlayersByPriorityScore sorts players by their pre-calculated priority score
// Higher priority score = placed first (gets first choice of division spots)
// The priority score already factors in:
//   - Target division
//   - Placement status (promoted/relegated/stayed/hiatus/new)
//   - Previous rank
//   - Hiatus decay (weight = 0.933^seasons_away)
//   - Rating (for NEW players)
func sortPlayersByPriorityScore(players []DPPlayer) {
	sort.Slice(players, func(i, j int) bool {
		return players[i].PriorityScore > players[j].PriorityScore
	})
}

// reconstructSolution backtraces through the DP cuts to build the final solution
func (rm *RebalanceManager) reconstructSolution(
	players []DPPlayer,
	cuts [][]int,
	K, N int,
	idealSize int,
) *DPSolution {
	solution := &DPSolution{
		NumDivisions:     K,
		Divisions:        make([]DPDivision, 0, K),
		PlayerToDivision: make(map[string]int32),
	}

	// Backtrack through cuts to find division boundaries
	divisionBoundaries := make([]int, K+1)
	divisionBoundaries[K] = N
	pos := N

	for k := K; k > 0; k-- {
		pos = cuts[k][pos]
		divisionBoundaries[k-1] = pos
	}

	// Build divisions
	for k := 1; k <= K; k++ {
		start := divisionBoundaries[k-1]
		end := divisionBoundaries[k]
		divPlayers := players[start:end]
		cost := rm.calculateDivisionCost(divPlayers, k, K, idealSize)

		div := DPDivision{
			DivisionNumber: int32(k),
			Players:        divPlayers,
			Cost:           cost,
		}

		solution.Divisions = append(solution.Divisions, div)

		// Update player-to-division mapping
		for _, p := range divPlayers {
			solution.PlayerToDivision[p.UserID] = int32(k)
		}
	}

	return solution
}

// Note: SortStandingsByRank is defined in standings.go
// It sorts by points (wins*2 + draws) desc, spread desc, then username asc

// Helper functions
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
