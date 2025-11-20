package league

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	// Authenticate - requires can_manage_leagues permission
	_, err := apiserver.AuthenticateWithPermission(ctx, ls.userStore, ls.queries, rbac.CanManageLeagues)
	if err != nil {
		return nil, err
	}

	// Validate input
	if req.Msg.LeagueId == "" {
		return nil, apiserver.InvalidArg("league_id is required")
	}
	if req.Msg.Settings == nil {
		return nil, apiserver.InvalidArg("settings is required")
	}

	// Parse league ID
	leagueID, err := uuid.Parse(req.Msg.LeagueId)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid league_id")
	}

	// Marshal settings to JSON
	settingsJSON, err := json.Marshal(req.Msg.Settings)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to marshal settings: %w", err))
	}

	// Update league settings
	err = ls.queries.UpdateLeagueSettings(ctx, models.UpdateLeagueSettingsParams{
		Uuid:     leagueID,
		Settings: settingsJSON,
	})
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to update league settings: %w", err))
	}

	log.Info().
		Str("leagueID", leagueID.String()).
		Msg("league-settings-updated")

	// Fetch and return updated league
	league, err := ls.store.GetLeagueByUUID(ctx, leagueID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to fetch updated league: %w", err))
	}

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

func (ls *LeagueService) UpdateLeagueMetadata(
	ctx context.Context,
	req *connect.Request[pb.UpdateLeagueMetadataRequest],
) (*connect.Response[pb.LeagueResponse], error) {
	// Authenticate - requires can_manage_leagues permission
	_, err := apiserver.AuthenticateWithPermission(ctx, ls.userStore, ls.queries, rbac.CanManageLeagues)
	if err != nil {
		return nil, err
	}

	// Validate input
	if req.Msg.LeagueId == "" {
		return nil, apiserver.InvalidArg("league_id is required")
	}
	if req.Msg.Name == "" {
		return nil, apiserver.InvalidArg("name is required")
	}

	// Parse league ID
	leagueID, err := uuid.Parse(req.Msg.LeagueId)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid league_id")
	}

	// Update league metadata
	err = ls.queries.UpdateLeagueMetadata(ctx, models.UpdateLeagueMetadataParams{
		Uuid:        leagueID,
		Name:        req.Msg.Name,
		Description: pgtype.Text{String: req.Msg.Description, Valid: req.Msg.Description != ""},
	})
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to update league metadata: %w", err))
	}

	log.Info().
		Str("leagueID", leagueID.String()).
		Str("name", req.Msg.Name).
		Msg("league-metadata-updated")

	// Fetch and return updated league
	league, err := ls.store.GetLeagueByUUID(ctx, leagueID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to fetch updated league: %w", err))
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
	isActive := false
	if league.IsActive.Valid {
		isActive = league.IsActive.Bool
	}

	protoLeague := &ipc.League{
		Uuid:        league.Uuid.String(),
		Name:        league.Name,
		Description: description,
		Slug:        league.Slug,
		Settings:    &settings,
		IsActive:    isActive,
	}

	return connect.NewResponse(&pb.LeagueResponse{League: protoLeague}), nil
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

func (ls *LeagueService) GetAllSeasons(
	ctx context.Context,
	req *connect.Request[pb.LeagueRequest],
) (*connect.Response[pb.AllSeasonsResponse], error) {
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

	// Get all seasons (regardless of status)
	dbSeasons, err := ls.store.GetSeasonsByLeague(ctx, leagueID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get all seasons: %w", err))
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

	return connect.NewResponse(&pb.AllSeasonsResponse{Seasons: protoSeasons}), nil
}

// OpenRegistration opens registration for a specific season
func (ls *LeagueService) OpenRegistration(
	ctx context.Context,
	req *connect.Request[pb.OpenRegistrationRequest],
) (*connect.Response[pb.SeasonResponse], error) {
	// Authenticate - requires can_manage_leagues permission
	_, err := apiserver.AuthenticateWithPermission(ctx, ls.userStore, ls.queries, rbac.CanManageLeagues)
	if err != nil {
		return nil, err
	}

	// season_id is required
	if req.Msg.SeasonId == "" {
		return nil, apiserver.InvalidArg("season_id is required")
	}

	seasonID, err := uuid.Parse(req.Msg.SeasonId)
	if err != nil {
		return nil, apiserver.InvalidArg(fmt.Sprintf("invalid season_id: %s", req.Msg.SeasonId))
	}

	lifecycleMgr := NewSeasonLifecycleManager(ls.stores, RealClock{})
	result, err := lifecycleMgr.OpenRegistrationForSeason(ctx, seasonID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to open registration: %w", err))
	}

	// Get the season
	season, err := ls.store.GetSeason(ctx, result.NextSeasonID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get season: %w", err))
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
		Str("leagueID", result.LeagueID.String()).
		Str("seasonID", result.NextSeasonID.String()).
		Int32("seasonNumber", result.NextSeasonNumber).
		Msg("registration-opened")

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
		// Use user info from the JOIN (no need to query separately)
		userUUID := standing.UserUuid.String
		username := standing.Username.String
		if username == "" {
			username = "Unknown"
		}

		resultValue := ipc.StandingResult_RESULT_NONE
		if standing.Result.Valid {
			resultValue = ipc.StandingResult(standing.Result.Int32)
		}

		protoStandings[i] = &ipc.LeaguePlayerStanding{
			UserId:         userUUID,
			Username:       username,
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

		// If no standings exist yet (season not started), create initial standings from registrations
		if len(standings) == 0 {
			registrations, err := ls.store.GetDivisionRegistrations(ctx, divisionUUID)
			if err != nil {
				return nil, apiserver.InternalErr(fmt.Errorf("failed to get registrations: %w", err))
			}

			// Convert registrations to "fake standings" with all zeros
			standings = make([]models.GetStandingsRow, len(registrations))
			for j, reg := range registrations {
				standings[j] = models.GetStandingsRow{
					UserID:         reg.UserID,
					UserUuid:       reg.UserUuid,                          // From JOIN
					Username:       pgtype.Text{String: "", Valid: false}, // Not included in registration JOIN, will fetch later
					Rank:           pgtype.Int4{Int32: int32(j + 1), Valid: true},
					Wins:           pgtype.Int4{Int32: 0, Valid: true},
					Losses:         pgtype.Int4{Int32: 0, Valid: true},
					Draws:          pgtype.Int4{Int32: 0, Valid: true},
					Spread:         pgtype.Int4{Int32: 0, Valid: true},
					GamesPlayed:    pgtype.Int4{Int32: 0, Valid: true},
					GamesRemaining: pgtype.Int4{Int32: 0, Valid: true},
					Result:         pgtype.Int4{Valid: false},
				}
			}
		}

		// Convert standings to proto
		protoStandings := make([]*ipc.LeaguePlayerStanding, len(standings))
		for j, standing := range standings {
			// Get user info (either from standings JOIN or lookup)
			userUUID := standing.UserUuid.String
			username := "Unknown"
			if standing.Username.Valid && standing.Username.String != "" {
				username = standing.Username.String
			} else if userUUID != "" {
				// Fallback: lookup by UUID
				user, err := ls.userStore.GetByUUID(ctx, userUUID)
				if err == nil {
					username = user.Username
				}
			}

			resultValue := ipc.StandingResult_RESULT_NONE
			if standing.Result.Valid {
				resultValue = ipc.StandingResult(standing.Result.Int32)
			}

			protoStandings[j] = &ipc.LeaguePlayerStanding{
				UserId:         userUUID,
				Username:       username,
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

	// Check if user has can_play_leagues permission
	hasPermission, err := rbac.HasPermission(ctx, ls.queries, uint(user.ID), rbac.CanPlayLeagues)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to check league permissions: %w", err))
	}
	if !hasPermission {
		return nil, apiserver.PermissionDenied("You need permission to play in leagues. Please contact a League Promoter for access.")
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

	// Validate that registration is open (only REGISTRATION_OPEN status)
	if season.Status != int32(ipc.SeasonStatus_SEASON_REGISTRATION_OPEN) {
		return nil, apiserver.InvalidArg(fmt.Sprintf("registration is not open for this season (current status: %s)", ipc.SeasonStatus(season.Status).String()))
	}

	// Register the player
	regMgr := NewRegistrationManager(ls.store, RealClock{}, ls.stores)
	err = regMgr.RegisterPlayer(ctx, int32(user.ID), season.Uuid)
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
	userDBIDToUnregister := int32(user.ID)
	userUUIDForLogging := user.UUID
	if req.Msg.UserId != "" && req.Msg.UserId != user.UUID {
		// Check if user has manage permission
		_, err := apiserver.AuthenticateWithPermission(ctx, ls.userStore, ls.queries, rbac.CanManageLeagues)
		if err != nil {
			return nil, apiserver.PermissionDenied("cannot unregister other users")
		}

		// Look up the target user to get their database ID
		targetUser, err := ls.userStore.GetByUUID(ctx, req.Msg.UserId)
		if err != nil {
			return nil, apiserver.InvalidArg(fmt.Sprintf("user not found: %s", req.Msg.UserId))
		}
		userDBIDToUnregister = int32(targetUser.ID)
		userUUIDForLogging = targetUser.UUID
	}

	// Unregister the player
	err = ls.store.UnregisterPlayer(ctx, models.UnregisterPlayerParams{
		SeasonID: seasonID,
		UserID:   userDBIDToUnregister,
	})
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to unregister player: %w", err))
	}

	log.Info().
		Str("userID", userUUIDForLogging).
		Str("seasonID", seasonID.String()).
		Msg("player-unregistered-from-season")

	return connect.NewResponse(&pb.UnregisterResponse{Success: true}), nil
}

func (ls *LeagueService) GetSeasonRegistrations(
	ctx context.Context,
	req *connect.Request[pb.SeasonRequest],
) (*connect.Response[pb.SeasonRegistrationsResponse], error) {
	// Parse season ID
	seasonID, err := uuid.Parse(req.Msg.SeasonId)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid season_id")
	}

	// Get all registrations for the season
	rm := NewRegistrationManager(ls.store, RealClock{}, ls.stores)
	registrations, err := rm.GetSeasonRegistrations(ctx, seasonID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get season registrations: %w", err))
	}

	// Convert to proto
	protoRegistrations := make([]*pb.SeasonRegistration, len(registrations))
	for i, reg := range registrations {
		// Get user info
		user, err := ls.userStore.GetByUUID(ctx, reg.UserUuid.String)
		if err != nil {
			// If user not found, use placeholder
			user = &entity.User{UUID: reg.UserUuid.String, Username: "Unknown"}
		}

		divisionID := ""
		divisionNumber := int32(0)
		if reg.DivisionID.Valid {
			divUUID, err := uuid.FromBytes(reg.DivisionID.Bytes[:])
			if err == nil {
				divisionID = divUUID.String()
				// Get division to find division number
				division, err := ls.store.GetDivision(ctx, divUUID)
				if err == nil {
					divisionNumber = division.DivisionNumber
				}
			}
		}

		protoRegistrations[i] = &pb.SeasonRegistration{
			UserId:         reg.UserUuid.String,
			Username:       user.Username,
			SeasonId:       reg.SeasonID.String(),
			DivisionId:     divisionID,
			DivisionNumber: divisionNumber,
		}
	}

	return connect.NewResponse(&pb.SeasonRegistrationsResponse{
		Registrations: protoRegistrations,
	}), nil
}

func (ls *LeagueService) InviteUserToLeagues(
	ctx context.Context,
	req *connect.Request[pb.InviteUserRequest],
) (*connect.Response[pb.InviteUserResponse], error) {
	// Authenticate and check for can_invite_to_leagues permission
	inviter, err := apiserver.AuthenticateWithPermission(ctx, ls.userStore, ls.queries, rbac.CanInviteToLeagues)
	if err != nil {
		return nil, err
	}

	// Validate user_id
	if req.Msg.UserId == "" {
		return nil, apiserver.InvalidArg("user_id is required")
	}

	// Get the target user
	targetUser, err := ls.userStore.GetByUUID(ctx, req.Msg.UserId)
	if err != nil {
		return nil, apiserver.InvalidArg(fmt.Sprintf("user not found: %s", req.Msg.UserId))
	}

	// Assign the League Player role (which grants can_play_leagues permission)
	err = ls.queries.AssignRole(ctx, models.AssignRoleParams{
		Username: targetUser.Username,
		RoleName: string(rbac.LeaguePlayer),
	})
	if err != nil {
		// Check if this is a duplicate key error (user already has the role)
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") ||
			strings.Contains(err.Error(), "user_roles_pkey") {
			return connect.NewResponse(&pb.InviteUserResponse{
				Success: true,
				Message: fmt.Sprintf("%s already has league access", targetUser.Username),
			}), nil
		}
		return nil, apiserver.InternalErr(fmt.Errorf("failed to assign league player role: %w", err))
	}

	log.Info().
		Str("inviterID", inviter.UUID).
		Str("inviterUsername", inviter.Username).
		Str("invitedUserID", targetUser.UUID).
		Str("invitedUsername", targetUser.Username).
		Msg("user-invited-to-leagues")

	return connect.NewResponse(&pb.InviteUserResponse{
		Success: true,
		Message: fmt.Sprintf("%s has been granted access to leagues", targetUser.Username),
	}), nil
}

func (ls *LeagueService) GetPlayerLeagueHistory(
	ctx context.Context,
	req *connect.Request[pb.PlayerHistoryRequest],
) (*connect.Response[pb.PlayerHistoryResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("GetPlayerLeagueHistory not yet implemented"))
}

func (ls *LeagueService) GetPlayerSeasonGames(
	ctx context.Context,
	req *connect.Request[pb.GetPlayerSeasonGamesRequest],
) (*connect.Response[pb.GetPlayerSeasonGamesResponse], error) {
	// Parse season ID
	seasonID, err := uuid.Parse(req.Msg.SeasonId)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid season_id")
	}

	// User ID is required
	userID := req.Msg.UserId
	if userID == "" {
		return nil, apiserver.InvalidArg("user_id is required")
	}

	// Use the new query that joins with game_players table
	gameRows, err := ls.queries.GetPlayerSeasonGames(ctx, models.GetPlayerSeasonGamesParams{
		SeasonID: pgtype.UUID{Bytes: seasonID, Valid: true},
		UserUuid: pgtype.Text{String: userID, Valid: true},
	})
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get player season games: %w", err))
	}

	// Convert to proto
	allGames := make([]*pb.PlayerSeasonGame, 0, len(gameRows))
	for _, row := range gameRows {
		// Determine result from the won field and game_end_reason
		result := "in_progress"
		playerScore := int32(0)
		opponentScore := int32(0)

		if row.GameEndReason != 0 { // Game is finished
			playerScore = row.PlayerScore
			opponentScore = row.OpponentScore

			if row.Won.Valid {
				if row.Won.Bool {
					result = "win"
				} else {
					result = "loss"
				}
			} else {
				result = "draw"
			}
		}

		allGames = append(allGames, &pb.PlayerSeasonGame{
			GameId:           row.GameUuid.String,
			OpponentUserId:   row.OpponentUuid.String,
			OpponentUsername: row.OpponentUsername.String,
			PlayerScore:      playerScore,
			OpponentScore:    opponentScore,
			Result:           result,
			GameDate:         timestamppb.New(row.CreatedAt.Time), // Game creation date
			Round:            0,                                    // TODO: Add round info if available
		})
	}

	return connect.NewResponse(&pb.GetPlayerSeasonGamesResponse{
		Games: allGames,
	}), nil
}

func (ls *LeagueService) GetLeagueStatistics(
	ctx context.Context,
	req *connect.Request[pb.LeagueRequest],
) (*connect.Response[pb.LeagueStatisticsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("GetLeagueStatistics not yet implemented"))
}

func (ls *LeagueService) MovePlayerToDivision(
	ctx context.Context,
	req *connect.Request[pb.MovePlayerToDivisionRequest],
) (*connect.Response[pb.MovePlayerToDivisionResponse], error) {
	// Authenticate - requires can_manage_leagues permission (admin only)
	_, err := apiserver.AuthenticateWithPermission(ctx, ls.userStore, ls.queries, rbac.CanManageLeagues)
	if err != nil {
		return nil, err
	}

	// Validate input
	if req.Msg.UserId == "" {
		return nil, apiserver.InvalidArg("user_id is required")
	}
	if req.Msg.SeasonId == "" {
		return nil, apiserver.InvalidArg("season_id is required")
	}
	if req.Msg.FromDivisionId == "" {
		return nil, apiserver.InvalidArg("from_division_id is required")
	}
	if req.Msg.ToDivisionId == "" {
		return nil, apiserver.InvalidArg("to_division_id is required")
	}

	// Parse UUIDs
	seasonID, err := uuid.Parse(req.Msg.SeasonId)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid season_id")
	}
	fromDivID, err := uuid.Parse(req.Msg.FromDivisionId)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid from_division_id")
	}
	toDivID, err := uuid.Parse(req.Msg.ToDivisionId)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid to_division_id")
	}

	// Get the season to check status
	season, err := ls.store.GetSeason(ctx, seasonID)
	if err != nil {
		return nil, apiserver.InvalidArg(fmt.Sprintf("season not found: %s", req.Msg.SeasonId))
	}

	// Only allow moving players when season is SCHEDULED
	if season.Status != int32(ipc.SeasonStatus_SEASON_SCHEDULED) {
		return nil, apiserver.InvalidArg(fmt.Sprintf("can only move players when season is SCHEDULED (current status: %s)", ipc.SeasonStatus(season.Status).String()))
	}

	// Use the ManualDivisionManager to move the player
	mdm := NewManualDivisionManager(ls.stores)
	result, err := mdm.MovePlayer(ctx, req.Msg.UserId, seasonID, fromDivID, toDivID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to move player: %w", err))
	}

	log.Info().
		Str("userID", result.UserID).
		Str("seasonID", seasonID.String()).
		Str("fromDivisionID", result.PreviousDivisionID.String()).
		Str("toDivisionID", result.NewDivisionID.String()).
		Msg("player-moved-to-division")

	return connect.NewResponse(&pb.MovePlayerToDivisionResponse{
		Success: true,
		Message: fmt.Sprintf("Player successfully moved to new division"),
	}), nil
}
