package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func inspectLeague(ctx context.Context, leagueSlugOrUUID string) error {
	log.Info().Str("league", leagueSlugOrUUID).Msg("inspecting league")

	// Initialize stores
	allStores, err := initializeStores(ctx)
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
		fmt.Printf("Season %d - %s\n", season.SeasonNumber, season.Status)
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

			divisionName := "Division"
			if division.DivisionName.Valid {
				divisionName = division.DivisionName.String
			}

			playerCount := int32(0)
			if division.PlayerCount.Valid {
				playerCount = division.PlayerCount.Int32
			}

			fmt.Printf("  %s %d (%d players)\n", divisionName, division.DivisionNumber, playerCount)

			// Get standings
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

				for _, standing := range standings {
					// Get username
					user, err := allStores.UserStore.GetByUUID(ctx, standing.UserID)
					username := "Unknown"
					if err == nil {
						username = user.Username
					}

					rank := int32(0)
					if standing.Rank.Valid {
						rank = standing.Rank.Int32
					}

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
