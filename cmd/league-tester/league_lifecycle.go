package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/league"
	"github.com/woogles-io/liwords/pkg/stores"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// GameplayAdapter adapts the gameplay package functions to the league.GameCreator interface
type GameplayAdapter struct {
	stores    *stores.Stores
	cfg       *config.Config
	eventChan chan<- *entity.EventWrapper
}

func (a *GameplayAdapter) InstantiateNewGame(ctx context.Context, users [2]*entity.User,
	req *pb.GameRequest, tdata *entity.TournamentData) (*entity.Game, error) {
	return gameplay.InstantiateNewGame(ctx, a.stores.GameStore, a.cfg, users, req, tdata)
}

func (a *GameplayAdapter) StartGame(ctx context.Context, game *entity.Game) error {
	return gameplay.StartGame(ctx, a.stores, a.eventChan, game)
}

// openRegistration opens registration for a specific season
func openRegistration(ctx context.Context, leagueSlugOrUUID string, seasonNumber int32) error {
	log.Info().
		Str("league", leagueSlugOrUUID).
		Int32("seasonNumber", seasonNumber).
		Msg("opening registration for season")

	// Initialize stores
	allStores, err := initStores(ctx)
	if err != nil {
		return err
	}

	// Get league
	leagueUUID, err := getLeagueUUID(ctx, allStores, leagueSlugOrUUID)
	if err != nil {
		return err
	}

	// Get season
	season, err := allStores.LeagueStore.GetSeasonByLeagueAndNumber(ctx, leagueUUID, seasonNumber)
	if err != nil {
		return fmt.Errorf("failed to get season %d: %w", seasonNumber, err)
	}

	lifecycleMgr := league.NewSeasonLifecycleManager(allStores)
	result, err := lifecycleMgr.OpenRegistrationForSeason(ctx, season.Uuid)
	if err != nil {
		return fmt.Errorf("failed to open registration: %w", err)
	}

	log.Info().
		Str("seasonID", result.NextSeasonID.String()).
		Int32("seasonNumber", result.NextSeasonNumber).
		Time("startDate", result.StartDate).
		Msg("successfully opened registration")

	return nil
}

// closeSeason closes the current active season
func closeSeason(ctx context.Context, leagueSlugOrUUID string) error {
	log.Info().
		Str("league", leagueSlugOrUUID).
		Msg("closing current season")

	// Initialize stores
	allStores, err := initStores(ctx)
	if err != nil {
		return err
	}

	// Get league
	leagueUUID, err := getLeagueUUID(ctx, allStores, leagueSlugOrUUID)
	if err != nil {
		return err
	}

	// Close season
	lifecycleMgr := league.NewSeasonLifecycleManager(allStores)
	result, err := lifecycleMgr.CloseCurrentSeason(ctx, leagueUUID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to close season: %w", err)
	}

	if result == nil {
		log.Info().Msg("season not closed (no active season found)")
		return nil
	}

	log.Info().
		Str("seasonID", result.CurrentSeasonID.String()).
		Int("forceFinishedGames", result.ForceFinishedGames).
		Msg("successfully closed season")

	return nil
}

// prepareDivisions prepares divisions for a season in REGISTRATION_OPEN status
func prepareDivisions(ctx context.Context, leagueSlugOrUUID string, seasonNumber int32) error {
	log.Info().
		Str("league", leagueSlugOrUUID).
		Int32("seasonNumber", seasonNumber).
		Msg("preparing divisions for season")

	// Initialize stores
	allStores, err := initStores(ctx)
	if err != nil {
		return err
	}

	// Get league
	leagueUUID, err := getLeagueUUID(ctx, allStores, leagueSlugOrUUID)
	if err != nil {
		return err
	}

	// Get season
	season, err := allStores.LeagueStore.GetSeasonByLeagueAndNumber(ctx, leagueUUID, seasonNumber)
	if err != nil {
		return fmt.Errorf("failed to get season %d: %w", seasonNumber, err)
	}

	// Prepare divisions
	lifecycleMgr := league.NewSeasonLifecycleManager(allStores)
	result, err := lifecycleMgr.PrepareAndScheduleSeason(ctx, leagueUUID, season.Uuid, time.Now())
	if err != nil {
		return fmt.Errorf("failed to prepare divisions: %w", err)
	}

	if result == nil {
		log.Info().Msg("divisions not prepared (season not in REGISTRATION_OPEN status)")
		return nil
	}

	log.Info().
		Str("seasonID", result.SeasonID.String()).
		Int("totalRegistrations", result.DivisionPreparation.TotalRegistrations).
		Int("newPlayers", result.DivisionPreparation.NewPlayers).
		Int("returningPlayers", result.DivisionPreparation.ReturningPlayers).
		Int("regularDivisions", result.DivisionPreparation.RegularDivisionsUsed).
		Msg("successfully prepared divisions and scheduled season")

	return nil
}

// startSeason starts a season in SCHEDULED status
func startSeason(ctx context.Context, leagueSlugOrUUID string, seasonNumber int32) error {
	log.Info().
		Str("league", leagueSlugOrUUID).
		Int32("seasonNumber", seasonNumber).
		Msg("starting season")

	// Initialize stores
	allStores, err := initStores(ctx)
	if err != nil {
		return err
	}

	// Get league
	leagueUUID, err := getLeagueUUID(ctx, allStores, leagueSlugOrUUID)
	if err != nil {
		return err
	}

	// Get season
	season, err := allStores.LeagueStore.GetSeasonByLeagueAndNumber(ctx, leagueUUID, seasonNumber)
	if err != nil {
		return fmt.Errorf("failed to get season %d: %w", seasonNumber, err)
	}

	// Load config
	cfg := &config.Config{}
	cfg.Load(nil)

	// Start season
	lifecycleMgr := league.NewSeasonLifecycleManager(allStores)
	result, err := lifecycleMgr.StartScheduledSeason(ctx, leagueUUID, season.Uuid, time.Now())
	if err != nil {
		return fmt.Errorf("failed to start season: %w", err)
	}

	// Create event channel for game events
	eventChan := make(chan *entity.EventWrapper, 100)
	defer close(eventChan)

	// Drain events in background
	go func() {
		for range eventChan {
			// Discard events in testing context
		}
	}()

	// Create game creator adapter
	gameCreator := &GameplayAdapter{
		stores:    allStores,
		cfg:       cfg,
		eventChan: eventChan,
	}

	// Parse league settings
	dbLeague, err := allStores.LeagueStore.GetLeagueByUUID(ctx, leagueUUID)
	if err != nil {
		return fmt.Errorf("failed to get league: %w", err)
	}

	leagueSettings, err := parseLeagueSettings(dbLeague.Settings)
	if err != nil {
		return fmt.Errorf("failed to parse league settings: %w", err)
	}

	// Create all games for the season
	startMgr := league.NewSeasonStartManager(allStores.LeagueStore, allStores, cfg, gameCreator)
	gameResult, err := startMgr.CreateGamesForSeason(ctx, leagueUUID, season.Uuid, leagueSettings)
	if err != nil {
		// Roll back the season status to SCHEDULED since game creation failed
		rollbackErr := lifecycleMgr.RollbackSeasonToScheduled(ctx, season.Uuid)
		if rollbackErr != nil {
			log.Err(rollbackErr).
				Str("seasonID", season.Uuid.String()).
				Msg("failed to rollback season status after game creation failure")
		}
		return fmt.Errorf("failed to create games (season rolled back to SCHEDULED): %w", err)
	}

	log.Info().
		Str("seasonID", result.SeasonID.String()).
		Int("totalGamesCreated", gameResult.TotalGamesCreated).
		Interface("gamesPerDivision", gameResult.GamesPerDivision).
		Msg("successfully started season and created games")

	return nil
}

// getLeagueUUID gets a league UUID by slug or UUID string
func getLeagueUUID(ctx context.Context, allStores *stores.Stores, leagueSlugOrUUID string) (uuid.UUID, error) {
	leagueUUID, err := uuid.Parse(leagueSlugOrUUID)
	if err != nil {
		// Not a UUID, try as slug
		league, err := allStores.LeagueStore.GetLeagueBySlug(ctx, leagueSlugOrUUID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("league not found: %s", leagueSlugOrUUID)
		}
		leagueUUID = league.Uuid
	}
	return leagueUUID, nil
}

// parseLeagueSettings parses the JSONB settings from the database
func parseLeagueSettings(settingsJSON []byte) (*pb.LeagueSettings, error) {
	var settings pb.LeagueSettings
	err := json.Unmarshal(settingsJSON, &settings)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}
