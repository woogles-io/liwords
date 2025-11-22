package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/league"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func simulateGames(ctx context.Context, leagueSlugOrUUID string, seasonNumber int32, all bool, count int, seed int64) error {
	log.Info().
		Str("league", leagueSlugOrUUID).
		Int32("seasonNumber", seasonNumber).
		Bool("all", all).
		Int("count", count).
		Int64("seed", seed).
		Msg("simulating games")

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
	seasonUUID := season.Uuid

	// Initialize random number generator
	var rng *rand.Rand
	if seed == 0 {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	} else {
		rng = rand.New(rand.NewSource(seed))
	}

	// Get all divisions for this season
	divisions, err := allStores.LeagueStore.GetDivisionsBySeason(ctx, seasonUUID)
	if err != nil {
		return fmt.Errorf("failed to get divisions: %w", err)
	}

	log.Info().Int("divisionCount", len(divisions)).Msg("found divisions")

	totalGamesCompleted := 0

	for _, division := range divisions {
		divisionUUID, err := uuid.FromBytes(division.Uuid[:])
		if err != nil {
			log.Err(err).Msg("failed to parse division UUID")
			continue
		}

		// Get all games for this division
		games, err := allStores.LeagueStore.GetLeagueGamesByStatus(ctx, models.GetLeagueGamesByStatusParams{
			LeagueDivisionID: pgtype.UUID{Bytes: divisionUUID, Valid: true},
			IncludeFinished:  true,
		})
		if err != nil {
			log.Err(err).Str("division", divisionUUID.String()).Msg("failed to get games")
			continue
		}

		log.Info().
			Str("division", divisionUUID.String()).
			Int("totalGames", len(games)).
			Msg("simulating games for division")

		gamesCompleted := 0
		for _, gameRow := range games {
			// Check if we've reached the count limit (across all divisions)
			if !all && count > 0 && totalGamesCompleted >= count {
				break
			}

			// Check if game is already complete
			if gameRow.GameEndReason.Valid && gameRow.GameEndReason.Int32 != 0 {
				// Game already ended
				continue
			}

			// Load game from store
			gameUUIDStr := gameRow.Uuid.String
			if !gameRow.Uuid.Valid {
				log.Warn().Msg("game has invalid UUID")
				continue
			}

			// Simulate the game
			err = simulateSingleGame(ctx, allStores, gameUUIDStr, rng)
			if err != nil {
				log.Err(err).Str("gameUUID", gameUUIDStr).Msg("failed to simulate game")
				continue
			}

			gamesCompleted++
			totalGamesCompleted++

			log.Info().
				Str("gameUUID", gameUUIDStr).
				Int("completed", gamesCompleted).
				Msg("simulated game")
		}

		log.Info().
			Str("division", divisionUUID.String()).
			Int("gamesCompleted", gamesCompleted).
			Msg("completed division simulation")

		// If we've reached the count limit, stop processing divisions
		if !all && count > 0 && totalGamesCompleted >= count {
			break
		}
	}

	log.Info().
		Int("totalGamesCompleted", totalGamesCompleted).
		Msg("successfully simulated games")

	return nil
}

func simulateSingleGame(ctx context.Context, allStores *stores.Stores, gameUUID string, rng *rand.Rand) error {
	// Load the actual game entity
	entGame, err := allStores.GameStore.Get(ctx, gameUUID)
	if err != nil {
		return fmt.Errorf("failed to get game entity: %w", err)
	}

	// Generate random but realistic scores
	// Typical game scores range from 250-500, with most around 350-450
	baseScore := 300
	variance := 150

	p0Score := baseScore + rng.Intn(variance)
	p1Score := baseScore + rng.Intn(variance)

	// Determine winner
	var winner, loser int
	if p0Score > p1Score {
		winner = 0
		loser = 1
	} else if p1Score > p0Score {
		winner = 1
		loser = 0
	} else {
		// Tie - rare, so nudge one score up by 1
		p0Score++
		winner = 0
		loser = 1
	}

	// Set game results
	entGame.SetPointsFor(0, p0Score)
	entGame.SetPointsFor(1, p1Score)
	entGame.SetWinnerIdx(winner)
	entGame.SetLoserIdx(loser)
	entGame.SetGameEndReason(pb.GameEndReason_STANDARD)

	// Mark game as over
	entGame.History().PlayState = 2 // GAME_OVER
	entGame.History().Winner = int32(winner)
	if len(entGame.History().FinalScores) == 0 {
		entGame.AddFinalScoresToHistory()
	}

	// Copy final scores to Quickdata (like performEndgameDuties does)
	// This is required for the UI to display scores correctly
	entGame.Quickdata.FinalScores = entGame.History().FinalScores

	// Save the game to the database
	err = allStores.GameStore.Set(ctx, entGame)
	if err != nil {
		return fmt.Errorf("failed to save game: %w", err)
	}

	// Insert entries into game_players table for query optimization
	// This is required for standings calculations and game queries
	err = allStores.GameStore.InsertGamePlayers(ctx, entGame)
	if err != nil {
		return fmt.Errorf("failed to insert game_players: %w", err)
	}

	// Manually update league standings
	// This is what normally happens in performEndgameDuties
	if allStores.LeagueStore != nil {
		err = league.UpdateGameStandings(ctx, allStores.LeagueStore, entGame.GameID())
		if err != nil {
			return fmt.Errorf("failed to update standings: %w", err)
		}
	}

	return nil
}
