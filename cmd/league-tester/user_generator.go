package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lithammer/shortuuid/v4"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/auth"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores"
)

type TestUsersOutput struct {
	Users []TestUser `json:"users"`
}

type TestUser struct {
	UUID     string `json:"uuid"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func createTestUsers(ctx context.Context, count int, startID int, outputFile string) error {
	log.Info().Int("count", count).Int("startID", startID).Msg("creating test users")

	// Initialize stores
	allStores, err := initStores(ctx)
	if err != nil {
		return err
	}

	testUsers := []TestUser{}

	// Create Argon2id config (matching default liwords config)
	argonConfig := auth.NewPasswordConfig(1, 64*1024, 4, 32)

	// Create users using the user store
	for i := 0; i < count; i++ {
		userNum := startID + i
		username := fmt.Sprintf("league_test_user_%02d", userNum)
		email := fmt.Sprintf("%s@example.com", username)
		userUUID := shortuuid.New()

		// Hash password using Argon2id (matching production hash algorithm)
		hashedPassword, err := auth.GeneratePassword(argonConfig, "testpassword123")
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		// Create user entity
		user := &entity.User{
			UUID:     userUUID,
			Username: username,
			Email:    email,
			Password: string(hashedPassword),
			Profile: &entity.Profile{
				Ratings: entity.Ratings{
					Data: make(map[entity.VariantKey]entity.SingleRating),
				},
			},
		}

		// Add default rating
		user.Profile.Ratings.Data[entity.VariantKey("correspondence.CSW24.classic")] = entity.SingleRating{
			Rating:          1500,
			RatingDeviation: 350,
			Volatility:      0.06,
		}

		// Create user in database
		err = allStores.UserStore.New(ctx, user)
		if err != nil {
			log.Warn().Err(err).Str("username", username).Msg("user might already exist, trying to get existing user")
			// Try to get existing user
			existingUser, getErr := allStores.UserStore.Get(ctx, username)
			if getErr != nil {
				return fmt.Errorf("failed to create or get user %s: %w", username, err)
			}
			userUUID = existingUser.UUID
		}

		testUsers = append(testUsers, TestUser{
			UUID:     userUUID,
			Username: username,
			Email:    email,
		})

		log.Info().
			Str("uuid", userUUID).
			Str("username", username).
			Msg("created test user")
	}

	// Write output file
	output := TestUsersOutput{Users: testUsers}
	outputJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	err = os.WriteFile(outputFile, outputJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	log.Info().
		Int("count", len(testUsers)).
		Str("outputFile", outputFile).
		Msg("successfully created test users")

	return nil
}

func initStores(ctx context.Context) (*stores.Stores, error) {
	cfg := &config.Config{}
	cfg.Load(nil)

	// Build DSN from environment variables (supports both test and production configs)
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbSSLMode := os.Getenv("DB_SSL_MODE")

	// Defaults for local development
	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "15432"
	}
	if dbName == "" {
		dbName = "liwords"
	}
	if dbUser == "" {
		dbUser = "postgres"
	}
	if dbPassword == "" {
		dbPassword = "pass"
	}
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	dbDSN := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		dbHost, dbPort, dbName, dbUser, dbPassword, dbSSLMode)

	log.Info().
		Str("host", dbHost).
		Str("port", dbPort).
		Str("dbname", dbName).
		Str("user", dbUser).
		Str("sslmode", dbSSLMode).
		Msg("Connecting to database")

	dbPool, err := pgxpool.New(ctx, dbDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize Redis pool (needed for some store operations)
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:16379/1"
	}

	redisPool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(redisURL)
		},
	}

	return stores.NewInitializedStores(dbPool, redisPool, cfg)
}
