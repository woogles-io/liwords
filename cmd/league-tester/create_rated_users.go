package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lithammer/shortuuid/v4"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/auth"
	"github.com/woogles-io/liwords/pkg/entity"
)

// createRatedTestUsers creates test users with varying ratings for testing rating-based division assignment
func createRatedTestUsers(ctx context.Context, count int, startID int, outputFile string) error {
	log.Info().Int("count", count).Int("startID", startID).Msg("creating test users with varied ratings")

	// Initialize stores
	allStores, err := initStores(ctx)
	if err != nil {
		return err
	}

	testUsers := []TestUser{}

	// Create Argon2id config (matching default liwords config)
	argonConfig := auth.NewPasswordConfig(1, 64*1024, 4, 32)

	// Rating ranges to distribute users across
	// We'll create users with ratings from 1200 to 2000
	minRating := 1200.0
	maxRating := 2000.0
	ratingStep := (maxRating - minRating) / float64(count-1)

	// Create users using the user store
	for i := 0; i < count; i++ {
		userNum := startID + i
		username := fmt.Sprintf("league_test_user_%02d", userNum)
		email := fmt.Sprintf("%s@example.com", username)
		userUUID := shortuuid.New()

		// Calculate this user's rating (distributed across the range)
		rating := minRating + (float64(i) * ratingStep)

		// Make ratings stable (RD < 90) for most users
		// Every 5th user has unstable rating to test that logic too
		rd := 60.0 // Stable rating
		if i%5 == 0 {
			rd = 120.0 // Unstable rating
		}

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

		// Add multiple ratings to simulate a realistic player
		// NWL18 = North American lexicon
		// CSW19 = International lexicon
		variants := []struct {
			key    entity.VariantKey
			rating float64
			rd     float64
		}{
			{entity.VariantKey("NWL18.classic.rapid"), rating, rd},
			{entity.VariantKey("NWL18.classic.blitz"), rating - 50, rd},
			{entity.VariantKey("CSW19.classic.rapid"), rating + 30, rd},
			{entity.VariantKey("NWL18.classic.corres"), rating + 20, rd},
		}

		for _, v := range variants {
			user.Profile.Ratings.Data[v.key] = entity.SingleRating{
				Rating:          v.rating,
				RatingDeviation: v.rd,
				Volatility:      0.06,
			}
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
			Float64("avgRating", rating).
			Float64("rd", rd).
			Msg("created rated test user")
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
		Msg("successfully created rated test users")

	return nil
}
