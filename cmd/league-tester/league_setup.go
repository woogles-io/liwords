package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type TestLeagueOutput struct {
	LeagueUUID  string `json:"league_uuid"`
	LeagueSlug  string `json:"league_slug"`
	SeasonUUID  string `json:"season_uuid"`
	SeasonNumber int32 `json:"season_number"`
}

func createTestLeague(ctx context.Context, name, slug string, divisionSize int32, outputFile string) error {
	log.Info().
		Str("name", name).
		Str("slug", slug).
		Int32("divisionSize", divisionSize).
		Msg("creating test league")

	// Load config
	cfg := &config.Config{}
	cfg.Load(nil)

	// Connect to database
	dbPool, err := pgxpool.New(ctx, cfg.DBConnDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer dbPool.Close()

	queries := models.New(dbPool)

	// Create league settings
	settings := &ipc.LeagueSettings{
		SeasonLengthDays: 30,
		TimeControl: &ipc.TimeControl{
			IncrementSeconds: 28800, // 8 hours
			TimeBankMinutes:  4320,  // 72 hours
		},
		Lexicon:           "CSW24",
		Variant:           "classic",
		IdealDivisionSize: divisionSize,
		ChallengeRule:     ipc.ChallengeRule_ChallengeRule_FIVE_POINT,
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Create league
	leagueUUID := uuid.New()
	league, err := queries.CreateLeague(ctx, models.CreateLeagueParams{
		Uuid:        leagueUUID,
		Name:        name,
		Description: pgtype.Text{String: "Test league for development and testing", Valid: true},
		Slug:        slug,
		Settings:    settingsJSON,
		IsActive:    pgtype.Bool{Bool: true, Valid: true},
		CreatedBy:   pgtype.Int8{Valid: false},
	})
	if err != nil {
		return fmt.Errorf("failed to create league: %w", err)
	}

	log.Info().
		Str("leagueUUID", league.Uuid.String()).
		Str("slug", slug).
		Msg("created league")

	// Bootstrap first season
	// Set dates: start in 1 day, end in 31 days
	now := time.Now()
	startDate := now.Add(24 * time.Hour)
	endDate := startDate.Add(30 * 24 * time.Hour)

	seasonUUID := uuid.New()
	season, err := queries.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         seasonUUID,
		LeagueID:     leagueUUID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: startDate, Valid: true},
		EndDate:      pgtype.Timestamptz{Time: endDate, Valid: true},
		Status:       int32(ipc.SeasonStatus_SEASON_REGISTRATION_OPEN),
	})
	if err != nil {
		return fmt.Errorf("failed to create season: %w", err)
	}

	// Update league to set current season
	err = queries.SetCurrentSeason(ctx, models.SetCurrentSeasonParams{
		Uuid:            leagueUUID,
		CurrentSeasonID: pgtype.UUID{Bytes: seasonUUID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update league current season: %w", err)
	}

	log.Info().
		Str("seasonUUID", season.Uuid.String()).
		Int32("seasonNumber", season.SeasonNumber).
		Str("status", ipc.SeasonStatus(season.Status).String()).
		Msg("bootstrapped first season")

	// Write output file
	output := TestLeagueOutput{
		LeagueUUID:   league.Uuid.String(),
		LeagueSlug:   slug,
		SeasonUUID:   season.Uuid.String(),
		SeasonNumber: season.SeasonNumber,
	}
	outputJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	err = os.WriteFile(outputFile, outputJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	log.Info().
		Str("outputFile", outputFile).
		Msg("successfully created test league")

	return nil
}

func registerTestUsers(ctx context.Context, leagueSlugOrUUID string, seasonNumber int32, usersFile string) error {
	log.Info().
		Str("league", leagueSlugOrUUID).
		Int32("seasonNumber", seasonNumber).
		Str("usersFile", usersFile).
		Msg("registering test users")

	// Load config
	cfg := &config.Config{}
	cfg.Load(nil)

	// Connect to database
	dbPool, err := pgxpool.New(ctx, cfg.DBConnDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer dbPool.Close()

	queries := models.New(dbPool)

	// Read users file
	usersData, err := os.ReadFile(usersFile)
	if err != nil {
		return fmt.Errorf("failed to read users file: %w", err)
	}

	var usersOutput TestUsersOutput
	err = json.Unmarshal(usersData, &usersOutput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal users file: %w", err)
	}

	// Get league
	var leagueUUID uuid.UUID
	leagueUUID, err = uuid.Parse(leagueSlugOrUUID)
	if err != nil {
		// Not a UUID, try as slug
		league, err := queries.GetLeagueBySlug(ctx, leagueSlugOrUUID)
		if err != nil {
			return fmt.Errorf("league not found: %s", leagueSlugOrUUID)
		}
		leagueUUID = league.Uuid
	}

	// Get season by number
	season, err := queries.GetSeasonByLeagueAndNumber(ctx, models.GetSeasonByLeagueAndNumberParams{
		LeagueID:     leagueUUID,
		SeasonNumber: seasonNumber,
	})
	if err != nil {
		return fmt.Errorf("failed to get season %d: %w", seasonNumber, err)
	}

	// Validate that registration is open (status = 4)
	if season.Status != int32(ipc.SeasonStatus_SEASON_REGISTRATION_OPEN) {
		return fmt.Errorf("registration is not open for season %d (current status: %s)",
			seasonNumber, ipc.SeasonStatus(season.Status).String())
	}

	log.Info().
		Str("seasonUUID", season.Uuid.String()).
		Int("userCount", len(usersOutput.Users)).
		Msg("registering users for season")

	// Register each user
	registered := 0
	for _, user := range usersOutput.Users {
		// Check if already registered
		existing, err := queries.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
			SeasonID: season.Uuid,
			UserID:   user.UUID,
		})
		if err == nil && existing.ID > 0 {
			log.Info().Str("username", user.Username).Msg("already registered, skipping")
			continue
		}

		// Create registration using RegisterPlayer
		_, err = queries.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:               user.UUID,
			SeasonID:             season.Uuid,
			DivisionID:           pgtype.UUID{Valid: false}, // No division yet
			RegistrationDate:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
			FirstsCount:          pgtype.Int4{Int32: 0, Valid: true},
			Status:               pgtype.Text{String: "REGISTERED", Valid: true},
			PlacementStatus:      pgtype.Int4{Int32: 0, Valid: true}, // 0 = PLACEMENT_NONE
			PreviousDivisionRank: pgtype.Int4{Valid: false},
			SeasonsAway:          pgtype.Int4{Int32: 0, Valid: true},
		})
		if err != nil {
			log.Warn().Err(err).Str("username", user.Username).Msg("failed to register user")
			continue
		}

		registered++
		log.Info().Str("username", user.Username).Msg("registered user")
	}

	log.Info().
		Int("registered", registered).
		Int("total", len(usersOutput.Users)).
		Msg("successfully registered users")

	return nil
}
