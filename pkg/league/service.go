package league

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"

	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	pb "github.com/woogles-io/liwords/rpc/api/proto/league_service"
)

// LeagueService implements the LeagueService RPC interface
type LeagueService struct {
	store       league.Store
	userStore   user.Store
	cfg         *config.Config
	queries     *models.Queries
	stores      *stores.Stores
	gameCreator GameCreator
}

// NewLeagueService creates a new LeagueService
func NewLeagueService(store league.Store, userStore user.Store, cfg *config.Config, queries *models.Queries,
	allStores *stores.Stores, gameCreator GameCreator) *LeagueService {
	return &LeagueService{
		store:       store,
		userStore:   userStore,
		cfg:         cfg,
		queries:     queries,
		stores:      allStores,
		gameCreator: gameCreator,
	}
}

// BootstrapSeason creates the first season for a league with explicit dates and status.
// This can only be used when the league has zero seasons.
func (ls *LeagueService) BootstrapSeason(
	ctx context.Context,
	req *connect.Request[pb.BootstrapSeasonRequest],
) (*connect.Response[pb.SeasonResponse], error) {
	// Parse league ID (could be UUID or slug)
	var leagueID uuid.UUID
	var err error

	// Try parsing as UUID first
	leagueID, err = uuid.Parse(req.Msg.LeagueId)
	if err != nil {
		// Not a UUID, try as slug
		dbLeague, err := ls.store.GetLeagueBySlug(ctx, req.Msg.LeagueId)
		if err != nil {
			return nil, apiserver.InvalidArg(fmt.Sprintf("league not found: %s", req.Msg.LeagueId))
		}
		leagueID = dbLeague.Uuid
	}

	// Verify league exists
	_, err = ls.store.GetLeagueByUUID(ctx, leagueID)
	if err != nil {
		return nil, apiserver.InvalidArg(fmt.Sprintf("league not found: %s", leagueID))
	}

	// Check that no seasons exist (bootstrap only)
	existingSeasons, err := ls.store.GetSeasonsByLeague(ctx, leagueID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to check existing seasons: %w", err))
	}
	if len(existingSeasons) > 0 {
		return nil, apiserver.InvalidArg("BootstrapSeason can only be used when league has zero seasons")
	}

	// Validate dates
	if req.Msg.StartDate == nil || req.Msg.EndDate == nil {
		return nil, apiserver.InvalidArg("start_date and end_date are required")
	}
	startTime := req.Msg.StartDate.AsTime()
	endTime := req.Msg.EndDate.AsTime()
	if endTime.Before(startTime) {
		return nil, apiserver.InvalidArg("end_date must be after start_date")
	}

	// Validate status
	if req.Msg.Status == ipc.SeasonStatus_SEASON_CANCELLED {
		return nil, apiserver.InvalidArg("cannot bootstrap season with CANCELLED status")
	}

	// Create the first season
	seasonID := uuid.New()
	season, err := ls.store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         seasonID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: startTime, Valid: true},
		EndDate:      pgtype.Timestamptz{Time: endTime, Valid: true},
		Status:       int32(req.Msg.Status),
	})
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to create season: %w", err))
	}

	// If status is ACTIVE, set as current season
	// (REGISTRATION_OPEN and SCHEDULED seasons should not be set as current)
	if req.Msg.Status == ipc.SeasonStatus_SEASON_ACTIVE {
		err = ls.store.SetCurrentSeason(ctx, models.SetCurrentSeasonParams{
			Uuid:            leagueID,
			CurrentSeasonID: pgtype.UUID{Bytes: seasonID, Valid: true},
		})
		if err != nil {
			return nil, apiserver.InternalErr(fmt.Errorf("failed to set current season: %w", err))
		}
	}

	log.Info().
		Str("leagueID", leagueID.String()).
		Str("seasonID", seasonID.String()).
		Str("status", req.Msg.Status.String()).
		Time("startDate", startTime).
		Time("endDate", endTime).
		Msg("bootstrapped-first-season")

	// Convert to proto response
	protoSeason := &ipc.Season{
		Uuid:         season.Uuid.String(),
		LeagueId:     season.LeagueID.String(),
		SeasonNumber: season.SeasonNumber,
		StartDate:    timestamppb.New(season.StartDate.Time),
		EndDate:      timestamppb.New(season.EndDate.Time),
		Status:       ipc.SeasonStatus(season.Status),
		Divisions:    []*ipc.Division{}, // No divisions yet
	}

	return connect.NewResponse(&pb.SeasonResponse{
		Season: protoSeason,
	}), nil
}

// Stub implementations for other RPC methods
// These will be implemented in future phases

func (ls *LeagueService) CreateLeague(
	ctx context.Context,
	req *connect.Request[pb.CreateLeagueRequest],
) (*connect.Response[pb.LeagueResponse], error) {
	// Authenticate - requires can_manage_leagues permission
	user, err := apiserver.AuthenticateWithPermission(ctx, ls.userStore, ls.queries, rbac.CanManageLeagues)
	if err != nil {
		return nil, err
	}

	// Validate input
	if req.Msg.Name == "" {
		return nil, apiserver.InvalidArg("name is required")
	}
	if req.Msg.Slug == "" {
		return nil, apiserver.InvalidArg("slug is required")
	}
	if req.Msg.Settings == nil {
		return nil, apiserver.InvalidArg("settings is required")
	}

	// Marshal settings to JSON
	settingsJSON, err := json.Marshal(req.Msg.Settings)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to marshal settings: %w", err))
	}

	// Create league
	leagueID := uuid.New()

	// Convert user UUID to int64 for createdBy
	userIDInt, err := ls.userStore.GetByUUID(ctx, user.UUID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get user: %w", err))
	}

	league, err := ls.store.CreateLeague(ctx, models.CreateLeagueParams{
		Uuid:        leagueID,
		Name:        req.Msg.Name,
		Description: pgtype.Text{String: req.Msg.Description, Valid: req.Msg.Description != ""},
		Slug:        req.Msg.Slug,
		Settings:    settingsJSON,
		IsActive:    pgtype.Bool{Bool: true, Valid: true},
		CreatedBy:   pgtype.Int8{Int64: int64(userIDInt.ID), Valid: true},
	})
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to create league: %w", err))
	}

	log.Info().
		Str("leagueID", leagueID.String()).
		Str("slug", req.Msg.Slug).
		Str("createdBy", user.UUID).
		Msg("league-created")

	// Convert to proto
	description := ""
	if league.Description.Valid {
		description = league.Description.String
	}
	isActive := false
	if league.IsActive.Valid {
		isActive = league.IsActive.Bool
	}

	protoLeague := &ipc.League{
		Uuid:        league.Uuid.String(),
		Name:        league.Name,
		Description: description,
		Slug:        league.Slug,
		Settings:    req.Msg.Settings,
		IsActive:    isActive,
	}

	return connect.NewResponse(&pb.LeagueResponse{League: protoLeague}), nil
}

func (ls *LeagueService) GetLeague(
	ctx context.Context,
	req *connect.Request[pb.LeagueRequest],
) (*connect.Response[pb.LeagueResponse], error) {
	// Parse league ID (UUID or slug)
	var league models.League
	var err error

	leagueID, err := uuid.Parse(req.Msg.LeagueId)
	if err != nil {
		// Try as slug
		league, err = ls.store.GetLeagueBySlug(ctx, req.Msg.LeagueId)
		if err != nil {
			return nil, apiserver.InvalidArg(fmt.Sprintf("league not found: %s", req.Msg.LeagueId))
		}
	} else {
		// Parse as UUID
		league, err = ls.store.GetLeagueByUUID(ctx, leagueID)
		if err != nil {
			return nil, apiserver.InvalidArg(fmt.Sprintf("league not found: %s", req.Msg.LeagueId))
		}
	}

	// Unmarshal settings
	var settings ipc.LeagueSettings
	if err := json.Unmarshal(league.Settings, &settings); err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to unmarshal settings: %w", err))
	}

	// Convert to proto
	description := ""
	if league.Description.Valid {
		description = league.Description.String
	}
	currentSeasonID := ""
	if league.CurrentSeasonID.Valid {
		currentSeasonUUID, err := uuid.FromBytes(league.CurrentSeasonID.Bytes[:])
		if err == nil {
			currentSeasonID = currentSeasonUUID.String()
		}
	}
	isActive := false
	if league.IsActive.Valid {
		isActive = league.IsActive.Bool
	}

	protoLeague := &ipc.League{
		Uuid:            league.Uuid.String(),
		Name:            league.Name,
		Description:     description,
		Slug:            league.Slug,
		Settings:        &settings,
		CurrentSeasonId: currentSeasonID,
		IsActive:        isActive,
	}

	return connect.NewResponse(&pb.LeagueResponse{League: protoLeague}), nil
}

func (ls *LeagueService) GetAllLeagues(
	ctx context.Context,
	req *connect.Request[pb.GetAllLeaguesRequest],
) (*connect.Response[pb.GetAllLeaguesResponse], error) {
	// Get all leagues
	dbLeagues, err := ls.store.GetAllLeagues(ctx, req.Msg.ActiveOnly)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get leagues: %w", err))
	}

	// Convert to proto
	protoLeagues := make([]*ipc.League, len(dbLeagues))
	for i, dbLeague := range dbLeagues {
		var settings ipc.LeagueSettings
		if err := json.Unmarshal(dbLeague.Settings, &settings); err != nil {
			return nil, apiserver.InternalErr(fmt.Errorf("failed to unmarshal settings: %w", err))
		}

		description := ""
		if dbLeague.Description.Valid {
			description = dbLeague.Description.String
		}
		currentSeasonID := ""
		if dbLeague.CurrentSeasonID.Valid {
			currentSeasonUUID, err := uuid.FromBytes(dbLeague.CurrentSeasonID.Bytes[:])
			if err == nil {
				currentSeasonID = currentSeasonUUID.String()
			}
		}
		isActive := false
		if dbLeague.IsActive.Valid {
			isActive = dbLeague.IsActive.Bool
		}

		protoLeagues[i] = &ipc.League{
			Uuid:            dbLeague.Uuid.String(),
			Name:            dbLeague.Name,
			Description:     description,
			Slug:            dbLeague.Slug,
			Settings:        &settings,
			CurrentSeasonId: currentSeasonID,
			IsActive:        isActive,
		}
	}

	return connect.NewResponse(&pb.GetAllLeaguesResponse{Leagues: protoLeagues}), nil
}

func (ls *LeagueService) UpdateLeagueSettings(
	ctx context.Context,
	req *connect.Request[pb.UpdateLeagueSettingsRequest],
) (*connect.Response[pb.LeagueResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("UpdateLeagueSettings not yet implemented"))
}

func (ls *LeagueService) GetSeason(
	ctx context.Context,
	req *connect.Request[pb.SeasonRequest],
) (*connect.Response[pb.SeasonResponse], error) {
	// Parse season ID
	seasonID, err := uuid.Parse(req.Msg.SeasonId)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid season_id")
	}

	// Get season
	season, err := ls.store.GetSeason(ctx, seasonID)
	if err != nil {
		return nil, apiserver.InvalidArg(fmt.Sprintf("season not found: %s", req.Msg.SeasonId))
	}

	// Convert to proto
	protoSeason := &ipc.Season{
		Uuid:         season.Uuid.String(),
		LeagueId:     season.LeagueID.String(),
		SeasonNumber: season.SeasonNumber,
		StartDate:    timestamppb.New(season.StartDate.Time),
		EndDate:      timestamppb.New(season.EndDate.Time),
		Status:       ipc.SeasonStatus(season.Status),
		Divisions:    []*ipc.Division{}, // TODO: optionally load divisions
	}

	if season.ActualEndDate.Valid {
		protoSeason.ActualEndDate = timestamppb.New(season.ActualEndDate.Time)
	}

	return connect.NewResponse(&pb.SeasonResponse{Season: protoSeason}), nil
}

func (ls *LeagueService) GetCurrentSeason(
	ctx context.Context,
	req *connect.Request[pb.LeagueRequest],
) (*connect.Response[pb.SeasonResponse], error) {
	// Parse league ID (UUID or slug)
	var leagueID uuid.UUID
	var err error

	leagueID, err = uuid.Parse(req.Msg.LeagueId)
	if err != nil {
		// Try as slug
		league, err := ls.store.GetLeagueBySlug(ctx, req.Msg.LeagueId)
		if err != nil {
			return nil, apiserver.InvalidArg(fmt.Sprintf("league not found: %s", req.Msg.LeagueId))
		}
		leagueID = league.Uuid
	}

	// Get current season
	season, err := ls.store.GetCurrentSeason(ctx, leagueID)
	if err != nil {
		return nil, apiserver.InvalidArg(fmt.Sprintf("no current season found for league: %s", req.Msg.LeagueId))
	}

	// Convert to proto
	protoSeason := &ipc.Season{
		Uuid:         season.Uuid.String(),
		LeagueId:     season.LeagueID.String(),
		SeasonNumber: season.SeasonNumber,
		StartDate:    timestamppb.New(season.StartDate.Time),
		EndDate:      timestamppb.New(season.EndDate.Time),
		Status:       ipc.SeasonStatus(season.Status),
		Divisions:    []*ipc.Division{}, // TODO: optionally load divisions
	}

	if season.ActualEndDate.Valid {
		protoSeason.ActualEndDate = timestamppb.New(season.ActualEndDate.Time)
	}

	return connect.NewResponse(&pb.SeasonResponse{Season: protoSeason}), nil
}

func (ls *LeagueService) GetPastSeasons(
	ctx context.Context,
	req *connect.Request[pb.LeagueRequest],
) (*connect.Response[pb.PastSeasonsResponse], error) {
	// Parse league ID (UUID or slug)
	var leagueID uuid.UUID
	var err error

	leagueID, err = uuid.Parse(req.Msg.LeagueId)
	if err != nil {
		// Try as slug
		league, err := ls.store.GetLeagueBySlug(ctx, req.Msg.LeagueId)
		if err != nil {
			return nil, apiserver.InvalidArg(fmt.Sprintf("league not found: %s", req.Msg.LeagueId))
		}
		leagueID = league.Uuid
	}

	// Get past seasons
	dbSeasons, err := ls.store.GetPastSeasons(ctx, leagueID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get past seasons: %w", err))
	}

	// Convert to proto
	protoSeasons := make([]*ipc.Season, len(dbSeasons))
	for i, season := range dbSeasons {
		protoSeasons[i] = &ipc.Season{
			Uuid:         season.Uuid.String(),
			LeagueId:     season.LeagueID.String(),
			SeasonNumber: season.SeasonNumber,
			StartDate:    timestamppb.New(season.StartDate.Time),
			EndDate:      timestamppb.New(season.EndDate.Time),
			Status:       ipc.SeasonStatus(season.Status),
			Divisions:    []*ipc.Division{},
		}

		if season.ActualEndDate.Valid {
			protoSeasons[i].ActualEndDate = timestamppb.New(season.ActualEndDate.Time)
		}
	}

	return connect.NewResponse(&pb.PastSeasonsResponse{Seasons: protoSeasons}), nil
}

// OpenRegistration opens registration for the next season (Day 15 workflow)
func (ls *LeagueService) OpenRegistration(
	ctx context.Context,
	req *connect.Request[pb.LeagueRequest],
) (*connect.Response[pb.SeasonResponse], error) {
	// Authenticate - requires can_manage_leagues permission
	_, err := apiserver.AuthenticateWithPermission(ctx, ls.userStore, ls.queries, rbac.CanManageLeagues)
	if err != nil {
		return nil, err
	}

	// Parse league ID (UUID or slug)
	var leagueID uuid.UUID
	leagueID, err = uuid.Parse(req.Msg.LeagueId)
	if err != nil {
		// Not a UUID, try as slug
		dbLeague, err := ls.store.GetLeagueBySlug(ctx, req.Msg.LeagueId)
		if err != nil {
			return nil, apiserver.InvalidArg(fmt.Sprintf("league not found: %s", req.Msg.LeagueId))
		}
		leagueID = dbLeague.Uuid
	}

	// Open registration for next season
	lifecycleMgr := NewSeasonLifecycleManager(ls.store, nil)
	result, err := lifecycleMgr.OpenRegistrationForNextSeason(ctx, leagueID, time.Now())
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to open registration: %w", err))
	}

	if result == nil {
		// Next season already exists or couldn't be created
		return nil, apiserver.InvalidArg("registration already open for next season or conditions not met")
	}

	// Get the created season
	season, err := ls.store.GetSeason(ctx, result.NextSeasonID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get created season: %w", err))
	}

	// Convert to proto response
	protoSeason := &ipc.Season{
		Uuid:         season.Uuid.String(),
		LeagueId:     season.LeagueID.String(),
		SeasonNumber: season.SeasonNumber,
		StartDate:    timestamppb.New(season.StartDate.Time),
		EndDate:      timestamppb.New(season.EndDate.Time),
		Status:       ipc.SeasonStatus(season.Status),
		Divisions:    []*ipc.Division{}, // No divisions created yet
	}

	log.Info().
		Str("leagueID", leagueID.String()).
		Str("seasonID", result.NextSeasonID.String()).
		Int32("seasonNumber", result.NextSeasonNumber).
		Msg("registration-opened-for-next-season")

	return connect.NewResponse(&pb.SeasonResponse{Season: protoSeason}), nil
}

// StartNextSeason starts a scheduled season and creates all games (Day 21 workflow)
func (ls *LeagueService) StartNextSeason(
	ctx context.Context,
	req *connect.Request[pb.StartSeasonRequest],
) (*connect.Response[pb.SeasonResponse], error) {
	// Authenticate - requires can_manage_leagues permission
	_, err := apiserver.AuthenticateWithPermission(ctx, ls.userStore, ls.queries, rbac.CanManageLeagues)
	if err != nil {
		return nil, err
	}

	// Parse league ID (UUID or slug)
	var leagueID uuid.UUID
	leagueID, err = uuid.Parse(req.Msg.LeagueId)
	if err != nil {
		// Not a UUID, try as slug
		dbLeague, err := ls.store.GetLeagueBySlug(ctx, req.Msg.LeagueId)
		if err != nil {
			return nil, apiserver.InvalidArg(fmt.Sprintf("league not found: %s", req.Msg.LeagueId))
		}
		leagueID = dbLeague.Uuid
	}

	// Get the league to access settings
	dbLeague, err := ls.store.GetLeagueByUUID(ctx, leagueID)
	if err != nil {
		return nil, apiserver.InvalidArg(fmt.Sprintf("league not found: %s", leagueID))
	}

	// Parse league settings
	var leagueSettings ipc.LeagueSettings
	err = json.Unmarshal(dbLeague.Settings, &leagueSettings)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to parse league settings: %w", err))
	}

	// Get current season to find the next scheduled season
	currentSeason, err := ls.store.GetCurrentSeason(ctx, leagueID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get current season: %w", err))
	}

	// Get next season (current season number + 1)
	nextSeasonNumber := currentSeason.SeasonNumber + 1
	nextSeason, err := ls.store.GetSeasonByLeagueAndNumber(ctx, leagueID, nextSeasonNumber)
	if err != nil {
		return nil, apiserver.InvalidArg(fmt.Sprintf("next season not found: %v", err))
	}

	// Step 1: Start the season (changes status to ACTIVE)
	lifecycleMgr := NewSeasonLifecycleManager(ls.store, ls.stores.GameStore)
	result, err := lifecycleMgr.StartScheduledSeason(ctx, leagueID, nextSeason.Uuid, time.Now())
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to start season: %w", err))
	}

	if result == nil {
		return nil, apiserver.InvalidArg("season not ready to start (not scheduled or start time not reached)")
	}

	// Step 2: Create ALL games for the season
	startMgr := NewSeasonStartManager(ls.store, ls.stores, ls.cfg, ls.gameCreator)
	gameResult, err := startMgr.CreateGamesForSeason(ctx, leagueID, nextSeason.Uuid, &leagueSettings)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to create games: %w", err))
	}

	// Get the updated season
	updatedSeason, err := ls.store.GetSeason(ctx, nextSeason.Uuid)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get updated season: %w", err))
	}

	// Convert to proto response
	protoSeason := &ipc.Season{
		Uuid:         updatedSeason.Uuid.String(),
		LeagueId:     updatedSeason.LeagueID.String(),
		SeasonNumber: updatedSeason.SeasonNumber,
		StartDate:    timestamppb.New(updatedSeason.StartDate.Time),
		EndDate:      timestamppb.New(updatedSeason.EndDate.Time),
		Status:       ipc.SeasonStatus(updatedSeason.Status),
		Divisions:    []*ipc.Division{}, // TODO: load divisions if needed
	}

	log.Info().
		Str("leagueID", leagueID.String()).
		Str("seasonID", nextSeason.Uuid.String()).
		Int32("seasonNumber", nextSeasonNumber).
		Int("totalGames", gameResult.TotalGamesCreated).
		Msg("season-started-and-games-created")

	return connect.NewResponse(&pb.SeasonResponse{Season: protoSeason}), nil
}

// CompleteSeason closes the current season (Day 20 workflow):
// - Force-finishes unfinished games
// - Marks season outcomes (promoted/relegated/stayed)
// - Prepares divisions for next season
func (ls *LeagueService) CompleteSeason(
	ctx context.Context,
	req *connect.Request[pb.SeasonRequest],
) (*connect.Response[pb.SeasonResponse], error) {
	// Authenticate - requires can_manage_leagues permission
	_, err := apiserver.AuthenticateWithPermission(ctx, ls.userStore, ls.queries, rbac.CanManageLeagues)
	if err != nil {
		return nil, err
	}

	// Parse season ID
	seasonID, err := uuid.Parse(req.Msg.SeasonId)
	if err != nil {
		return nil, apiserver.InvalidArg(fmt.Sprintf("invalid season ID: %s", req.Msg.SeasonId))
	}

	// Get the season to find the league
	season, err := ls.store.GetSeason(ctx, seasonID)
	if err != nil {
		return nil, apiserver.InvalidArg(fmt.Sprintf("season not found: %s", seasonID))
	}

	leagueID := season.LeagueID

	// Close the current season (force-finishes games, marks outcomes, prepares next season)
	lifecycleMgr := NewSeasonLifecycleManager(ls.store, ls.stores.GameStore)
	result, err := lifecycleMgr.CloseCurrentSeason(ctx, leagueID, time.Now())
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to close season: %w", err))
	}

	if result == nil {
		return nil, apiserver.InvalidArg("season could not be closed (may not be active or conditions not met)")
	}

	// Get the updated season
	updatedSeason, err := ls.store.GetSeason(ctx, seasonID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get updated season: %w", err))
	}

	// Convert to proto response
	protoSeason := &ipc.Season{
		Uuid:         updatedSeason.Uuid.String(),
		LeagueId:     updatedSeason.LeagueID.String(),
		SeasonNumber: updatedSeason.SeasonNumber,
		StartDate:    timestamppb.New(updatedSeason.StartDate.Time),
		EndDate:      timestamppb.New(updatedSeason.EndDate.Time),
		Status:       ipc.SeasonStatus(updatedSeason.Status),
		Divisions:    []*ipc.Division{}, // TODO: load divisions if needed
	}

	log.Info().
		Str("leagueID", leagueID.String()).
		Str("seasonID", seasonID.String()).
		Int("forceFinishedGames", result.ForceFinishedGames).
		Int("totalRegistrations", result.DivisionPreparation.TotalRegistrations).
		Msg("season-closed-and-next-season-prepared")

	return connect.NewResponse(&pb.SeasonResponse{Season: protoSeason}), nil
}

func (ls *LeagueService) GetDivisionStandings(
	ctx context.Context,
	req *connect.Request[pb.DivisionRequest],
) (*connect.Response[pb.DivisionStandingsResponse], error) {
	// Parse division ID
	divisionID, err := uuid.Parse(req.Msg.DivisionId)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid division_id")
	}

	// Get division
	division, err := ls.store.GetDivision(ctx, divisionID)
	if err != nil {
		return nil, apiserver.InvalidArg(fmt.Sprintf("division not found: %s", req.Msg.DivisionId))
	}

	// Get standings
	standings, err := ls.store.GetStandings(ctx, divisionID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get standings: %w", err))
	}

	// Convert standings to proto
	protoStandings := make([]*ipc.LeaguePlayerStanding, len(standings))
	for i, standing := range standings {
		// Get user info
		user, err := ls.userStore.GetByUUID(ctx, standing.UserID)
		if err != nil {
			// If user not found, use placeholder
			user = &entity.User{UUID: standing.UserID, Username: "Unknown"}
		}

		resultValue := ipc.StandingResult_RESULT_NONE
		if standing.Result.Valid {
			resultValue = ipc.StandingResult(standing.Result.Int32)
		}

		protoStandings[i] = &ipc.LeaguePlayerStanding{
			UserId:         standing.UserID,
			Username:       user.Username,
			Rank:           standing.Rank.Int32,
			Wins:           standing.Wins.Int32,
			Losses:         standing.Losses.Int32,
			Draws:          standing.Draws.Int32,
			Spread:         standing.Spread.Int32,
			GamesPlayed:    standing.GamesPlayed.Int32,
			GamesRemaining: standing.GamesRemaining.Int32,
			Result:         resultValue,
		}
	}

	// Build division proto
	divisionName := ""
	if division.DivisionName.Valid {
		divisionName = division.DivisionName.String
	}
	isComplete := false
	if division.IsComplete.Valid {
		isComplete = division.IsComplete.Bool
	}

	protoDivision := &ipc.Division{
		Uuid:           division.Uuid.String(),
		SeasonId:       division.SeasonID.String(),
		DivisionNumber: division.DivisionNumber,
		DivisionName:   divisionName,
		Standings:      protoStandings,
		IsComplete:     isComplete,
	}

	return connect.NewResponse(&pb.DivisionStandingsResponse{Division: protoDivision}), nil
}

func (ls *LeagueService) GetAllDivisionStandings(
	ctx context.Context,
	req *connect.Request[pb.SeasonRequest],
) (*connect.Response[pb.AllDivisionStandingsResponse], error) {
	// Parse season ID
	seasonID, err := uuid.Parse(req.Msg.SeasonId)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid season_id")
	}

	// Get all divisions for season
	divisions, err := ls.store.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get divisions: %w", err))
	}

	// Get standings for each division
	protoDivisions := make([]*ipc.Division, len(divisions))
	for i, division := range divisions {
		divisionUUID, err := uuid.FromBytes(division.Uuid[:])
		if err != nil {
			return nil, apiserver.InternalErr(fmt.Errorf("failed to parse division UUID: %w", err))
		}

		standings, err := ls.store.GetStandings(ctx, divisionUUID)
		if err != nil {
			return nil, apiserver.InternalErr(fmt.Errorf("failed to get standings: %w", err))
		}

		// Convert standings to proto
		protoStandings := make([]*ipc.LeaguePlayerStanding, len(standings))
		for j, standing := range standings {
			// Get user info
			user, err := ls.userStore.GetByUUID(ctx, standing.UserID)
			if err != nil {
				user = &entity.User{UUID: standing.UserID, Username: "Unknown"}
			}

			resultValue := ipc.StandingResult_RESULT_NONE
			if standing.Result.Valid {
				resultValue = ipc.StandingResult(standing.Result.Int32)
			}

			protoStandings[j] = &ipc.LeaguePlayerStanding{
				UserId:         standing.UserID,
				Username:       user.Username,
				Rank:           standing.Rank.Int32,
				Wins:           standing.Wins.Int32,
				Losses:         standing.Losses.Int32,
				Draws:          standing.Draws.Int32,
				Spread:         standing.Spread.Int32,
				GamesPlayed:    standing.GamesPlayed.Int32,
				GamesRemaining: standing.GamesRemaining.Int32,
				Result:         resultValue,
			}
		}

		divisionName := ""
		if division.DivisionName.Valid {
			divisionName = division.DivisionName.String
		}
		isComplete := false
		if division.IsComplete.Valid {
			isComplete = division.IsComplete.Bool
		}

		protoDivisions[i] = &ipc.Division{
			Uuid:           divisionUUID.String(),
			SeasonId:       division.SeasonID.String(),
			DivisionNumber: division.DivisionNumber,
			DivisionName:   divisionName,
			Standings:      protoStandings,
			IsComplete:     isComplete,
		}
	}

	return connect.NewResponse(&pb.AllDivisionStandingsResponse{Divisions: protoDivisions}), nil
}

func (ls *LeagueService) RegisterForSeason(
	ctx context.Context,
	req *connect.Request[pb.RegisterRequest],
) (*connect.Response[pb.RegisterResponse], error) {
	// Authenticate - requires user to be logged in
	user, err := apiserver.AuthUser(ctx, ls.userStore)
	if err != nil {
		return nil, err
	}

	// Parse season ID (required)
	if req.Msg.SeasonId == "" {
		return nil, apiserver.InvalidArg("season_id is required")
	}
	seasonID, err := uuid.Parse(req.Msg.SeasonId)
	if err != nil {
		return nil, apiserver.InvalidArg(fmt.Sprintf("invalid season_id: %s", req.Msg.SeasonId))
	}

	// Get the season
	season, err := ls.store.GetSeason(ctx, seasonID)
	if err != nil {
		return nil, apiserver.InvalidArg(fmt.Sprintf("season not found: %s", req.Msg.SeasonId))
	}

	// Validate that registration is open
	if season.Status != int32(ipc.SeasonStatus_SEASON_REGISTRATION_OPEN) {
		return nil, apiserver.InvalidArg(fmt.Sprintf("registration is not open for this season (current status: %s)", ipc.SeasonStatus(season.Status).String()))
	}

	// Register the player
	regMgr := NewRegistrationManager(ls.store)
	err = regMgr.RegisterPlayer(ctx, user.UUID, season.Uuid)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to register player: %w", err))
	}

	log.Info().
		Str("userID", user.UUID).
		Str("seasonID", season.Uuid.String()).
		Str("leagueID", season.LeagueID.String()).
		Msg("player-registered-for-season")

	return connect.NewResponse(&pb.RegisterResponse{
		Success:  true,
		SeasonId: season.Uuid.String(),
	}), nil
}

func (ls *LeagueService) UnregisterFromSeason(
	ctx context.Context,
	req *connect.Request[pb.UnregisterRequest],
) (*connect.Response[pb.UnregisterResponse], error) {
	// Authenticate - requires user to be logged in
	user, err := apiserver.AuthUser(ctx, ls.userStore)
	if err != nil {
		return nil, err
	}

	// Parse season ID
	seasonID, err := uuid.Parse(req.Msg.SeasonId)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid season_id")
	}

	// Get the season to check status
	season, err := ls.store.GetSeason(ctx, seasonID)
	if err != nil {
		return nil, apiserver.InvalidArg(fmt.Sprintf("season not found: %s", req.Msg.SeasonId))
	}

	// Validate that season is not active
	if season.Status == int32(ipc.SeasonStatus_SEASON_ACTIVE) {
		return nil, apiserver.InvalidArg("cannot unregister from an active season")
	}

	// Allow user to specify different user_id only if they have manage permission
	userIDToUnregister := user.UUID
	if req.Msg.UserId != "" && req.Msg.UserId != user.UUID {
		// Check if user has manage permission
		_, err := apiserver.AuthenticateWithPermission(ctx, ls.userStore, ls.queries, rbac.CanManageLeagues)
		if err != nil {
			return nil, apiserver.PermissionDenied("cannot unregister other users")
		}
		userIDToUnregister = req.Msg.UserId
	}

	// Unregister the player
	err = ls.store.UnregisterPlayer(ctx, models.UnregisterPlayerParams{
		SeasonID: seasonID,
		UserID:   userIDToUnregister,
	})
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to unregister player: %w", err))
	}

	log.Info().
		Str("userID", userIDToUnregister).
		Str("seasonID", seasonID.String()).
		Msg("player-unregistered-from-season")

	return connect.NewResponse(&pb.UnregisterResponse{Success: true}), nil
}

func (ls *LeagueService) GetPlayerLeagueHistory(
	ctx context.Context,
	req *connect.Request[pb.PlayerHistoryRequest],
) (*connect.Response[pb.PlayerHistoryResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("GetPlayerLeagueHistory not yet implemented"))
}

func (ls *LeagueService) GetLeagueStatistics(
	ctx context.Context,
	req *connect.Request[pb.LeagueRequest],
) (*connect.Response[pb.LeagueStatisticsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("GetLeagueStatistics not yet implemented"))
}
