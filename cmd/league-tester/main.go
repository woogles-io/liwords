package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	command := os.Args[1]
	ctx := context.Background()

	var err error
	switch command {
	case "create-users":
		err = createUsersCommand(ctx, os.Args[2:])
	case "create-league":
		err = createLeagueCommand(ctx, os.Args[2:])
	case "register-users":
		err = registerUsersCommand(ctx, os.Args[2:])
	case "open-registration":
		err = openRegistrationCommand(ctx, os.Args[2:])
	case "close-season":
		err = closeSeasonCommand(ctx, os.Args[2:])
	case "prepare-divisions":
		err = prepareDivisionsCommand(ctx, os.Args[2:])
	case "start-season":
		err = startSeasonCommand(ctx, os.Args[2:])
	case "simulate-games":
		err = simulateGamesCommand(ctx, os.Args[2:])
	case "inspect":
		err = inspectCommand(ctx, os.Args[2:])
	case "run-full-season":
		err = runFullSeasonCommand(ctx, os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatal().Err(err).Msg("command failed")
	}
}

func printUsage() {
	fmt.Println("League Tester - Test tool for league functionality")
	fmt.Println()
	fmt.Println("Usage: go run cmd/league-tester <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  create-users       Create fake test users")
	fmt.Println("  create-league      Create a test league")
	fmt.Println("  register-users     Register users for a league season")
	fmt.Println("  start-season       Start a season (creates games)")
	fmt.Println("  simulate-games     Simulate game completions with random results")
	fmt.Println("  inspect            Inspect current league state")
	fmt.Println("  run-full-season    Run complete season(s) end-to-end")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/league-tester create-users --count 20")
	fmt.Println("  go run cmd/league-tester create-league --slug test-league")
	fmt.Println("  go run cmd/league-tester register-users --league test-league --season 1")
	fmt.Println("  go run cmd/league-tester simulate-games --season <uuid> --all")
	fmt.Println()
	fmt.Println("Run 'go run cmd/league-tester <command> --help' for command-specific options")
}

func createUsersCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("create-users", flag.ExitOnError)
	count := fs.Int("count", 20, "Number of test users to create")
	output := fs.String("output", "test_users.json", "Output file for user UUIDs")
	fs.Parse(args)

	return createTestUsers(ctx, *count, *output)
}

func createLeagueCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("create-league", flag.ExitOnError)
	name := fs.String("name", "Test League", "League name")
	slug := fs.String("slug", "test-league", "League slug")
	divisionSize := fs.Int("division-size", 15, "Ideal division size (target players per division)")
	output := fs.String("output", "test_league.json", "Output file for league info")
	fs.Parse(args)

	return createTestLeague(ctx, *name, *slug, int32(*divisionSize), *output)
}

func registerUsersCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("register-users", flag.ExitOnError)
	league := fs.String("league", "", "League slug or UUID (required)")
	season := fs.Int("season", 0, "Season number (required)")
	usersFile := fs.String("users-file", "test_users.json", "JSON file with user UUIDs")
	fs.Parse(args)

	if *league == "" {
		return fmt.Errorf("--league is required")
	}

	if *season == 0 {
		return fmt.Errorf("--season is required")
	}

	return registerTestUsers(ctx, *league, int32(*season), *usersFile)
}

func openRegistrationCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("open-registration", flag.ExitOnError)
	league := fs.String("league", "", "League slug or UUID (required)")
	season := fs.Int("season", 0, "Season number (required)")
	fs.Parse(args)

	if *league == "" {
		return fmt.Errorf("--league is required")
	}

	if *season == 0 {
		return fmt.Errorf("--season is required")
	}

	return openRegistration(ctx, *league, int32(*season))
}

func closeSeasonCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("close-season", flag.ExitOnError)
	league := fs.String("league", "", "League slug or UUID (required)")
	fs.Parse(args)

	if *league == "" {
		return fmt.Errorf("--league is required")
	}

	return closeSeason(ctx, *league)
}

func prepareDivisionsCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("prepare-divisions", flag.ExitOnError)
	league := fs.String("league", "", "League slug or UUID (required)")
	season := fs.Int("season", 0, "Season number (required)")
	fs.Parse(args)

	if *league == "" {
		return fmt.Errorf("--league is required")
	}

	if *season == 0 {
		return fmt.Errorf("--season is required")
	}

	return prepareDivisions(ctx, *league, int32(*season))
}

func startSeasonCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("start-season", flag.ExitOnError)
	league := fs.String("league", "", "League slug or UUID (required)")
	season := fs.Int("season", 0, "Season number (required)")
	fs.Parse(args)

	if *league == "" {
		return fmt.Errorf("--league is required")
	}

	if *season == 0 {
		return fmt.Errorf("--season is required")
	}

	return startSeason(ctx, *league, int32(*season))
}

func simulateGamesCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("simulate-games", flag.ExitOnError)
	season := fs.String("season", "", "Season UUID (required)")
	all := fs.Bool("all", true, "Simulate all games at once")
	rounds := fs.Int("rounds", 0, "Number of rounds to simulate (0 = all)")
	seed := fs.Int64("seed", 0, "Random seed for reproducibility (0 = random)")
	fs.Parse(args)

	if *season == "" {
		return fmt.Errorf("--season is required")
	}

	return simulateGames(ctx, *season, *all, *rounds, *seed)
}

func inspectCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	league := fs.String("league", "", "League slug or UUID (required)")
	fs.Parse(args)

	if *league == "" {
		return fmt.Errorf("--league is required")
	}

	return inspectLeague(ctx, *league)
}

func runFullSeasonCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("run-full-season", flag.ExitOnError)
	league := fs.String("league", "", "League slug or UUID (required)")
	seasons := fs.Int("seasons", 1, "Number of seasons to run")
	seed := fs.Int64("seed", 0, "Random seed for reproducibility (0 = random)")
	fs.Parse(args)

	if *league == "" {
		return fmt.Errorf("--league is required")
	}

	log.Info().
		Str("league", *league).
		Int("seasons", *seasons).
		Int64("seed", *seed).
		Msg("run-full-season not yet implemented")
	return fmt.Errorf("not implemented yet")
}
