package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/league"
	leaguestore "github.com/woogles-io/liwords/pkg/stores/league"
)

// LeagueRegistrationOpener opens registration for the next season on Day 15
// This creates a new season with status REGISTRATION_OPEN
func LeagueRegistrationOpener() error {
	log.Info().Msg("starting league registration opener maintenance task")

	ctx := context.Background()
	cfg := &config.Config{}
	cfg.Load(nil)

	dbPool, err := pgxpool.New(ctx, cfg.DBConnDSN)
	if err != nil {
		return err
	}
	defer dbPool.Close()

	leagueStore, err := leaguestore.NewDBStore(cfg, dbPool)
	if err != nil {
		return err
	}

	lifecycleMgr := league.NewSeasonLifecycleManager(leagueStore)

	// Get all active leagues
	leagues, err := leagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	now := time.Now()
	registrationsOpened := 0

	for _, dbLeague := range leagues {
		result, err := lifecycleMgr.OpenRegistrationForNextSeason(ctx, dbLeague.Uuid, now)
		if err != nil {
			log.Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("failed to open registration")
			continue
		}

		if result != nil {
			log.Info().
				Str("leagueID", result.LeagueID.String()).
				Str("seasonID", result.NextSeasonID.String()).
				Int32("seasonNumber", result.NextSeasonNumber).
				Time("startDate", result.StartDate).
				Msg("successfully opened registration for next season")
			registrationsOpened++
		}
	}

	log.Info().Int("registrationsOpened", registrationsOpened).Msg("completed league registration opener")
	return nil
}

// LeagueSeasonCloser closes the current season on Day 20 at midnight
// This force-finishes unfinished games, marks season outcomes, and prepares next season divisions
func LeagueSeasonCloser() error {
	log.Info().Msg("starting league season closer maintenance task")

	ctx := context.Background()
	cfg := &config.Config{}
	cfg.Load(nil)

	dbPool, err := pgxpool.New(ctx, cfg.DBConnDSN)
	if err != nil {
		return err
	}
	defer dbPool.Close()

	leagueStore, err := leaguestore.NewDBStore(cfg, dbPool)
	if err != nil {
		return err
	}

	lifecycleMgr := league.NewSeasonLifecycleManager(leagueStore)

	// Get all active leagues
	leagues, err := leagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	now := time.Now()
	seasonsClosed := 0

	for _, dbLeague := range leagues {
		result, err := lifecycleMgr.CloseCurrentSeason(ctx, dbLeague.Uuid, now)
		if err != nil {
			log.Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("failed to close season")
			continue
		}

		if result != nil {
			log.Info().
				Str("currentSeasonID", result.CurrentSeasonID.String()).
				Str("nextSeasonID", result.NextSeasonID.String()).
				Str("leagueID", result.LeagueID.String()).
				Int("forceFinished", result.ForceFinishedGames).
				Int("totalRegistrations", result.DivisionPreparation.TotalRegistrations).
				Msg("successfully closed season and prepared next season")
			seasonsClosed++
		}
	}

	log.Info().Int("seasonsClosed", seasonsClosed).Msg("completed league season closer")
	return nil
}

// LeagueSeasonStarter starts seasons that are SCHEDULED and past their start date
// This runs at 8 AM ET on Day 21 (or any time after the scheduled start)
func LeagueSeasonStarter() error {
	log.Info().Msg("starting league season starter maintenance task")

	ctx := context.Background()
	cfg := &config.Config{}
	cfg.Load(nil)

	dbPool, err := pgxpool.New(ctx, cfg.DBConnDSN)
	if err != nil {
		return err
	}
	defer dbPool.Close()

	leagueStore, err := leaguestore.NewDBStore(cfg, dbPool)
	if err != nil {
		return err
	}

	lifecycleMgr := league.NewSeasonLifecycleManager(leagueStore)

	// Get all active leagues
	leagues, err := leagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}

	now := time.Now()
	seasonsStarted := 0

	for _, dbLeague := range leagues {
		// Get all seasons for this league
		seasons, err := leagueStore.GetSeasonsByLeague(ctx, dbLeague.Uuid)
		if err != nil {
			log.Warn().Err(err).Str("leagueID", dbLeague.Uuid.String()).Msg("failed to get seasons")
			continue
		}

		for _, season := range seasons {
			result, err := lifecycleMgr.StartScheduledSeason(ctx, dbLeague.Uuid, season.Uuid, now)
			if err != nil {
				log.Err(err).
					Str("seasonID", season.Uuid.String()).
					Str("leagueID", dbLeague.Uuid.String()).
					Msg("failed to start season")
				continue
			}

			if result != nil {
				log.Info().
					Str("leagueID", result.LeagueID.String()).
					Str("seasonID", result.SeasonID.String()).
					Str("leagueName", result.LeagueName).
					Msg("successfully started league season")
				seasonsStarted++
			}
		}
	}

	log.Info().Int("seasonsStarted", seasonsStarted).Msg("completed league season starter")
	return nil
}
