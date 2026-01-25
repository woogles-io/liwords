package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func setSeasonStatusCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("set-season-status", flag.ExitOnError)
	league := fs.String("league", "", "League slug or UUID (required)")
	season := fs.Int("season", 0, "Season number (required)")
	status := fs.String("status", "", "Status name: SCHEDULED, ACTIVE, REGISTRATION_OPEN, COMPLETED, CANCELLED")
	statusValue := fs.Int("status-value", -1, "Status as integer (0-4)")
	fs.Parse(args)

	if *league == "" {
		return fmt.Errorf("--league is required")
	}
	if *season == 0 {
		return fmt.Errorf("--season is required")
	}

	// Determine the status value
	var newStatus int32
	if *statusValue >= 0 {
		// Use the numeric value
		newStatus = int32(*statusValue)
	} else if *status != "" {
		// Parse the status name
		parsedStatus, err := parseSeasonStatus(*status)
		if err != nil {
			return err
		}
		newStatus = int32(parsedStatus)
	} else {
		return fmt.Errorf("either --status or --status-value must be provided")
	}

	// Validate the status value
	if newStatus < 0 || newStatus > 4 {
		return fmt.Errorf("invalid status value: %d (must be 0-4)", newStatus)
	}

	log.Info().
		Str("league", *league).
		Int("seasonNumber", *season).
		Int32("newStatus", newStatus).
		Str("statusName", pb.SeasonStatus(newStatus).String()).
		Msg("setting season status")

	// Initialize stores
	allStores, err := initStores(ctx)
	if err != nil {
		return err
	}

	// Get league
	leagueUUID, err := getLeagueUUID(ctx, allStores, *league)
	if err != nil {
		return err
	}

	// Get season
	seasonData, err := allStores.LeagueStore.GetSeasonByLeagueAndNumber(ctx, leagueUUID, int32(*season))
	if err != nil {
		return fmt.Errorf("failed to get season %d: %w", *season, err)
	}

	// Update the season status
	err = allStores.LeagueStore.UpdateSeasonStatus(ctx, models.UpdateSeasonStatusParams{
		Uuid:   seasonData.Uuid,
		Status: newStatus,
	})
	if err != nil {
		return fmt.Errorf("failed to update season status: %w", err)
	}

	log.Info().
		Str("seasonUUID", seasonData.Uuid.String()).
		Str("status", pb.SeasonStatus(newStatus).String()).
		Msg("successfully updated season status")

	return nil
}

func parseSeasonStatus(statusName string) (pb.SeasonStatus, error) {
	// Normalize the input
	statusName = strings.ToUpper(strings.TrimSpace(statusName))

	// Add SEASON_ prefix if not present
	if !strings.HasPrefix(statusName, "SEASON_") {
		statusName = "SEASON_" + statusName
	}

	// Map status names to values
	statusMap := map[string]pb.SeasonStatus{
		"SEASON_SCHEDULED":         pb.SeasonStatus_SEASON_SCHEDULED,
		"SEASON_ACTIVE":            pb.SeasonStatus_SEASON_ACTIVE,
		"SEASON_COMPLETED":         pb.SeasonStatus_SEASON_COMPLETED,
		"SEASON_CANCELLED":         pb.SeasonStatus_SEASON_CANCELLED,
		"SEASON_REGISTRATION_OPEN": pb.SeasonStatus_SEASON_REGISTRATION_OPEN,
	}

	if status, ok := statusMap[statusName]; ok {
		return status, nil
	}

	return 0, fmt.Errorf("unknown status name: %s (valid: SCHEDULED, ACTIVE, REGISTRATION_OPEN, COMPLETED, CANCELLED)", statusName)
}
