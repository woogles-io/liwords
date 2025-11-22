package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func inspectLeague(ctx context.Context, leagueSlugOrUUID string) error {
	log.Info().Str("league", leagueSlugOrUUID).Msg("inspecting league")

	// Initialize stores
	allStores, err := initStores(ctx)
	if err != nil {
		return err
	}

	// Get league
	var leagueUUID uuid.UUID
	leagueUUID, err = uuid.Parse(leagueSlugOrUUID)
	if err != nil {
		// Not a UUID, try as slug
		league, err := allStores.LeagueStore.GetLeagueBySlug(ctx, leagueSlugOrUUID)
		if err != nil {
			return fmt.Errorf("league not found: %s", leagueSlugOrUUID)
		}
		leagueUUID = league.Uuid
	}

	league, err := allStores.LeagueStore.GetLeagueByUUID(ctx, leagueUUID)
	if err != nil {
		return fmt.Errorf("failed to get league: %w", err)
	}

	fmt.Println("================================================================================")
	fmt.Printf("LEAGUE: %s (%s)\n", league.Name, league.Slug)
	fmt.Println("================================================================================")
	fmt.Println()

	// Get all seasons
	seasons, err := allStores.LeagueStore.GetSeasonsByLeague(ctx, leagueUUID)
	if err != nil {
		return fmt.Errorf("failed to get seasons: %w", err)
	}

	for _, season := range seasons {
		fmt.Printf("Season %d - %s\n", season.SeasonNumber, pb.SeasonStatus(season.Status).String())
		fmt.Printf("  UUID: %s\n", season.Uuid.String())
		fmt.Printf("  Start: %s\n", season.StartDate.Time.Format("2006-01-02"))
		fmt.Printf("  End: %s\n", season.EndDate.Time.Format("2006-01-02"))
		fmt.Println()

		// Get divisions for this season
		divisions, err := allStores.LeagueStore.GetDivisionsBySeason(ctx, season.Uuid)
		if err != nil {
			log.Err(err).Msg("failed to get divisions")
			continue
		}

		if len(divisions) == 0 {
			fmt.Println("  No divisions yet")
			fmt.Println()
			continue
		}

		for _, division := range divisions {
			divisionUUID, err := uuid.FromBytes(division.Uuid[:])
			if err != nil {
				log.Err(err).Msg("failed to parse division UUID")
				continue
			}

			divisionName := fmt.Sprintf("Division %d", division.DivisionNumber)
			if division.DivisionName.Valid && division.DivisionName.String != "" {
				divisionName = division.DivisionName.String
			}

			// Count actual players assigned to this division
			registrations, err := allStores.LeagueStore.GetDivisionRegistrations(ctx, division.Uuid)
			if err != nil {
				log.Err(err).Msg("failed to get division registrations")
				continue
			}

			playerCount := len(registrations)
			fmt.Printf("  %s (%d players)\n", divisionName, playerCount)

			// Get standings for display
			standings, err := allStores.LeagueStore.GetStandings(ctx, divisionUUID)
			if err != nil {
				log.Err(err).Msg("failed to get standings")
				continue
			}

			if len(standings) == 0 {
				fmt.Println("    No standings yet")
			} else {
				// Print standings header
				fmt.Printf("    %-4s %-25s %4s %4s %4s %6s %6s\n",
					"Rank", "Player", "W", "L", "D", "Spread", "Games")
				fmt.Println("    " + "────────────────────────────────────────────────────────────")

				for i, standing := range standings {
					// Get username from the JOIN (no need for separate lookup)
					username := "Unknown"
					if standing.Username.Valid {
						username = standing.Username.String
					}

					// Calculate rank from position (standings are already ordered)
					rank := int32(i + 1)

					wins := int32(0)
					if standing.Wins.Valid {
						wins = standing.Wins.Int32
					}

					losses := int32(0)
					if standing.Losses.Valid {
						losses = standing.Losses.Int32
					}

					draws := int32(0)
					if standing.Draws.Valid {
						draws = standing.Draws.Int32
					}

					spread := int32(0)
					if standing.Spread.Valid {
						spread = standing.Spread.Int32
					}

					gamesPlayed := int32(0)
					if standing.GamesPlayed.Valid {
						gamesPlayed = standing.GamesPlayed.Int32
					}

					fmt.Printf("    %-4d %-25s %4d %4d %4d %+6d %6d\n",
						rank, username, wins, losses, draws, spread, gamesPlayed)
				}
			}

			// Get game completion status
			totalGames, err := allStores.LeagueStore.CountDivisionGamesTotal(ctx, divisionUUID)
			if err != nil {
				log.Err(err).Msg("failed to count total games")
				continue
			}

			completedGames, err := allStores.LeagueStore.CountDivisionGamesComplete(ctx, divisionUUID)
			if err != nil {
				log.Err(err).Msg("failed to count completed games")
				continue
			}

			fmt.Printf("    Games: %d/%d completed\n", completedGames, totalGames)
			fmt.Println()
		}
	}

	return nil
}
