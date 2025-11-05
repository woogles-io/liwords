package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lithammer/shortuuid/v4"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

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

func createTestUsers(ctx context.Context, count int, outputFile string) error {
	log.Info().Int("count", count).Msg("creating test users")

	// Initialize stores
	allStores, err := initStores(ctx)
	if err != nil {
		return err
	}

	testUsers := []TestUser{}

	// Create users using the user store
	for i := 1; i <= count; i++ {
		username := fmt.Sprintf("league_test_user_%02d", i)
		email := fmt.Sprintf("%s@example.com", username)
		userUUID := shortuuid.New()

		// Hash a simple password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpassword123"), bcrypt.DefaultCost)
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

	dbPool, err := pgxpool.New(ctx, cfg.DBConnDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// For testing, we don't need Redis
	return stores.NewInitializedStores(dbPool, nil, cfg)
}
