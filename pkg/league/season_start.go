package league

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// GameCreator interface for creating and starting games
// This avoids import cycle with gameplay package by allowing the caller to provide
// an adapter that wraps the gameplay package functions.
//
// Example implementation (in calling code, not in league package):
//
//	type GameplayAdapter struct {
//		stores    *stores.Stores
//		cfg       *config.Config
//		eventChan chan<- *entity.EventWrapper
//	}
//
//	func (a *GameplayAdapter) InstantiateNewGame(ctx context.Context, users [2]*entity.User,
//		req *pb.GameRequest, tdata *entity.TournamentData) (*entity.Game, error) {
//		return gameplay.InstantiateNewGame(ctx, a.stores.GameStore, a.cfg, users, req, tdata)
//	}
//
//	func (a *GameplayAdapter) StartGame(ctx context.Context, game *entity.Game) error {
//		return gameplay.StartGame(ctx, a.stores, a.eventChan, game)
//	}
//
// This is the same pattern used in pkg/bus/gameplay.go for tournaments.
type GameCreator interface {
	InstantiateNewGame(ctx context.Context, users [2]*entity.User, req *pb.GameRequest, tdata *entity.TournamentData) (*entity.Game, error)
	StartGame(ctx context.Context, game *entity.Game) error
}

// SeasonStartManager handles creating games when a season starts
type SeasonStartManager struct {
	store       league.Store
	stores      *stores.Stores
	cfg         *config.Config
	gameCreator GameCreator
}

// NewSeasonStartManager creates a new season start manager
func NewSeasonStartManager(
	store league.Store,
	stores *stores.Stores,
	cfg *config.Config,
	gameCreator GameCreator,
) *SeasonStartManager {
	return &SeasonStartManager{
		store:       store,
		stores:      stores,
		cfg:         cfg,
		gameCreator: gameCreator,
	}
}

// GameCreationResult tracks the outcome of creating games for a season
type GameCreationResult struct {
	TotalGamesCreated int
	GamesPerDivision  map[uuid.UUID]int
	Errors            []string
}

// CreateGamesForSeason creates all games for all divisions in a season
func (ssm *SeasonStartManager) CreateGamesForSeason(
	ctx context.Context,
	leagueID uuid.UUID,
	seasonID uuid.UUID,
	leagueSettings *pb.LeagueSettings,
) (*GameCreationResult, error) {
	result := &GameCreationResult{
		GamesPerDivision: make(map[uuid.UUID]int),
		Errors:           []string{},
	}

	// Get all divisions for this season
	divisions, err := ssm.store.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get divisions: %w", err)
	}

	log.Info().
		Str("seasonID", seasonID.String()).
		Int("divisionCount", len(divisions)).
		Msg("creating-games-for-season")

	// Process each division
	for _, division := range divisions {
		gamesCreated, err := ssm.createGamesForDivision(ctx, leagueID, seasonID, division, leagueSettings)
		if err != nil {
			errMsg := fmt.Sprintf("division %s: %v", division.Uuid.String(), err)
			result.Errors = append(result.Errors, errMsg)
			log.Error().Err(err).
				Str("divisionID", division.Uuid.String()).
				Msg("failed-to-create-games-for-division")
			continue
		}

		result.GamesPerDivision[division.Uuid] = gamesCreated
		result.TotalGamesCreated += gamesCreated
	}

	if len(result.Errors) > 0 {
		return result, fmt.Errorf("encountered %d errors creating games", len(result.Errors))
	}

	return result, nil
}

// createGamesForDivision creates all games for a single division
func (ssm *SeasonStartManager) createGamesForDivision(
	ctx context.Context,
	leagueID uuid.UUID,
	seasonID uuid.UUID,
	division models.LeagueDivision,
	leagueSettings *pb.LeagueSettings,
) (int, error) {
	// Get players in division
	registrations, err := ssm.store.GetDivisionRegistrations(ctx, division.Uuid)
	if err != nil {
		return 0, fmt.Errorf("failed to get players: %w", err)
	}

	numPlayers := len(registrations)
	if numPlayers < 2 {
		log.Warn().
			Str("divisionID", division.Uuid.String()).
			Int("playerCount", numPlayers).
			Msg("skipping-division-with-insufficient-players")
		return 0, nil
	}

	// Calculate max rounds based on player count
	maxRounds := calculateMaxRounds(numPlayers)

	// Generate pairings
	seed := generatePairingSeed(seasonID, division.Uuid)
	pairings, err := GenerateAllLeaguePairings(numPlayers, seed, maxRounds)
	if err != nil {
		return 0, fmt.Errorf("failed to generate pairings: %w", err)
	}

	log.Info().
		Str("divisionID", division.Uuid.String()).
		Int("playerCount", numPlayers).
		Int("maxRounds", maxRounds).
		Int("pairingsCount", len(pairings)).
		Msg("generated-pairings-for-division")

	// Build game request from league settings
	gameReq, err := ssm.buildGameRequest(leagueSettings)
	if err != nil {
		return 0, fmt.Errorf("failed to build game request: %w", err)
	}

	// Create tournament data for league games
	tdata := &entity.TournamentData{
		Division: division.Uuid.String(),
	}

	// Create games for each pairing
	gamesCreated := 0
	for _, pairing := range pairings {
		// Get player registrations
		player1Reg := registrations[pairing.Player1Index]
		player2Reg := registrations[pairing.Player2Index]

		// Look up user entities
		user1, err := ssm.stores.UserStore.GetByUUID(ctx, player1Reg.UserID)
		if err != nil {
			return gamesCreated, fmt.Errorf("failed to get user %s: %w", player1Reg.UserID, err)
		}

		user2, err := ssm.stores.UserStore.GetByUUID(ctx, player2Reg.UserID)
		if err != nil {
			return gamesCreated, fmt.Errorf("failed to get user %s: %w", player2Reg.UserID, err)
		}

		// Determine order based on who goes first
		var users [2]*entity.User
		if pairing.IsPlayer1First {
			users = [2]*entity.User{user1, user2}
		} else {
			users = [2]*entity.User{user2, user1}
		}

		// Create the game
		game, err := ssm.gameCreator.InstantiateNewGame(ctx, users, gameReq, tdata)
		if err != nil {
			return gamesCreated, fmt.Errorf("failed to create game: %w", err)
		}

		// Start the game (starts timer for correspondence games)
		err = ssm.gameCreator.StartGame(ctx, game)
		if err != nil {
			return gamesCreated, fmt.Errorf("failed to start game %s: %w", game.Uid(), err)
		}

		gamesCreated++
	}

	log.Info().
		Str("divisionID", division.Uuid.String()).
		Int("gamesCreated", gamesCreated).
		Msg("created-games-for-division")

	return gamesCreated, nil
}

// calculateMaxRounds determines the max rounds based on player count
func calculateMaxRounds(numPlayers int) int {
	if numPlayers >= 15 {
		return 14 // Cap at 14 rounds for large divisions
	}
	// For smaller divisions, do full round-robin (N-1 rounds)
	return 0 // 0 means no limit, will use N-1
}

// generatePairingSeed creates a consistent seed from seasonID and divisionID
func generatePairingSeed(seasonID uuid.UUID, divisionID uuid.UUID) uint64 {
	// Combine season and division UUIDs
	combined := seasonID.String() + divisionID.String()

	// Hash to get consistent seed
	hash := sha256.Sum256([]byte(combined))

	// Convert first 8 bytes to uint64
	seed := binary.BigEndian.Uint64(hash[:8])

	return seed
}

// buildGameRequest builds a GameRequest from league settings
func (ssm *SeasonStartManager) buildGameRequest(settings *pb.LeagueSettings) (*pb.GameRequest, error) {
	// Use settings or defaults
	lexicon := settings.Lexicon
	if lexicon == "" {
		lexicon = "CSW24"
	}

	variant := settings.Variant
	if variant == "" {
		variant = "classic"
	}

	// Build time control
	var timeControl *pb.TimeControl
	if settings.TimeControl != nil {
		timeControl = settings.TimeControl
	} else {
		// Default: 8 hours per turn (28800 seconds), 72 hour bank (4320 minutes)
		timeControl = &pb.TimeControl{
			IncrementSeconds: 28800, // 8 hours
			TimeBankMinutes:  4320,  // 72 hours
		}
	}

	// Determine challenge rule
	challengeRule := settings.ChallengeRule
	if challengeRule == pb.ChallengeRule_ChallengeRule_VOID {
		// Use default based on lexicon
		challengeRule = determineChallengeRule(lexicon)
	}

	req := &pb.GameRequest{
		Lexicon:       lexicon,
		ChallengeRule: macondo.ChallengeRule(challengeRule),
		Rules: &pb.GameRules{
			BoardLayoutName:         "CrosswordGame",
			LetterDistributionName:  lexicon,
			VariantName:             variant,
		},
		InitialTimeSeconds: 0, // Correspondence games don't use this
		IncrementSeconds:   timeControl.IncrementSeconds,
		MaxOvertimeMinutes: int32(timeControl.TimeBankMinutes),
		RatingMode:         pb.RatingMode_RATED,
		RequestId:          uuid.NewString(),
		OriginalRequestId:  uuid.NewString(),
	}

	return req, nil
}

// determineChallengeRule returns appropriate challenge rule based on lexicon
func determineChallengeRule(lexicon string) pb.ChallengeRule {
	// CSW lexicons use FIVE_POINT, NWL lexicons use DOUBLE
	lexiconUpper := strings.ToUpper(lexicon)
	if strings.HasPrefix(lexiconUpper, "CSW") {
		return pb.ChallengeRule_ChallengeRule_FIVE_POINT
	}
	return pb.ChallengeRule_ChallengeRule_DOUBLE
}
