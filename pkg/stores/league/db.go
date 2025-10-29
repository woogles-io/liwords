package league

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// Store is the interface for league database operations
type Store interface {
	// League operations
	CreateLeague(ctx context.Context, arg models.CreateLeagueParams) (models.League, error)
	GetLeagueByUUID(ctx context.Context, uuid uuid.UUID) (models.League, error)
	GetLeagueBySlug(ctx context.Context, slug string) (models.League, error)
	GetAllLeagues(ctx context.Context, activeOnly bool) ([]models.League, error)
	UpdateLeagueSettings(ctx context.Context, arg models.UpdateLeagueSettingsParams) error
	SetCurrentSeason(ctx context.Context, arg models.SetCurrentSeasonParams) error

	// Season operations
	CreateSeason(ctx context.Context, arg models.CreateSeasonParams) (models.LeagueSeason, error)
	GetSeason(ctx context.Context, uuid uuid.UUID) (models.LeagueSeason, error)
	GetCurrentSeason(ctx context.Context, leagueUUID uuid.UUID) (models.LeagueSeason, error)
	GetPastSeasons(ctx context.Context, leagueID uuid.UUID) ([]models.LeagueSeason, error)
	GetSeasonsByLeague(ctx context.Context, leagueID uuid.UUID) ([]models.LeagueSeason, error)
	UpdateSeasonStatus(ctx context.Context, arg models.UpdateSeasonStatusParams) error
	MarkSeasonComplete(ctx context.Context, uuid uuid.UUID) error

	// Division operations
	CreateDivision(ctx context.Context, arg models.CreateDivisionParams) (models.LeagueDivision, error)
	GetDivision(ctx context.Context, uuid uuid.UUID) (models.LeagueDivision, error)
	GetDivisionsBySeason(ctx context.Context, seasonID uuid.UUID) ([]models.LeagueDivision, error)
	UpdateDivisionPlayerCount(ctx context.Context, arg models.UpdateDivisionPlayerCountParams) error
	MarkDivisionComplete(ctx context.Context, uuid uuid.UUID) error

	// Registration operations
	RegisterPlayer(ctx context.Context, arg models.RegisterPlayerParams) (models.LeagueRegistration, error)
	UnregisterPlayer(ctx context.Context, arg models.UnregisterPlayerParams) error
	GetPlayerRegistration(ctx context.Context, arg models.GetPlayerRegistrationParams) (models.LeagueRegistration, error)
	GetSeasonRegistrations(ctx context.Context, seasonID uuid.UUID) ([]models.LeagueRegistration, error)
	GetDivisionRegistrations(ctx context.Context, divisionID uuid.UUID) ([]models.LeagueRegistration, error)
	UpdatePlayerDivision(ctx context.Context, arg models.UpdatePlayerDivisionParams) error
	UpdateRegistrationDivision(ctx context.Context, arg models.UpdateRegistrationDivisionParams) error
	GetPlayerSeasonHistory(ctx context.Context, arg models.GetPlayerSeasonHistoryParams) ([]models.GetPlayerSeasonHistoryRow, error)

	// Standings operations
	UpsertStanding(ctx context.Context, arg models.UpsertStandingParams) error
	GetStandings(ctx context.Context, divisionID uuid.UUID) ([]models.LeagueStanding, error)
	GetPlayerStanding(ctx context.Context, arg models.GetPlayerStandingParams) (models.LeagueStanding, error)
	DeleteDivisionStandings(ctx context.Context, divisionID uuid.UUID) error

	// Game queries
	GetLeagueGames(ctx context.Context, divisionID uuid.UUID) ([]models.Game, error)
	GetLeagueGamesByStatus(ctx context.Context, arg models.GetLeagueGamesByStatusParams) ([]models.Game, error)
	CountDivisionGamesComplete(ctx context.Context, divisionID uuid.UUID) (int64, error)
	CountDivisionGamesTotal(ctx context.Context, divisionID uuid.UUID) (int64, error)
}

// DBStore is a postgres-backed store for leagues.
type DBStore struct {
	cfg     *config.Config
	dbPool  *pgxpool.Pool
	queries *models.Queries
}

// NewDBStore creates a new DB store for leagues.
func NewDBStore(cfg *config.Config, dbPool *pgxpool.Pool) (*DBStore, error) {
	return &DBStore{
		cfg:     cfg,
		dbPool:  dbPool,
		queries: models.New(dbPool),
	}, nil
}

// All methods below just delegate to the sqlc-generated queries

// League operations

func (s *DBStore) CreateLeague(ctx context.Context, arg models.CreateLeagueParams) (models.League, error) {
	return s.queries.CreateLeague(ctx, arg)
}

func (s *DBStore) GetLeagueByUUID(ctx context.Context, uuid uuid.UUID) (models.League, error) {
	return s.queries.GetLeagueByUUID(ctx, uuid)
}

func (s *DBStore) GetLeagueBySlug(ctx context.Context, slug string) (models.League, error) {
	return s.queries.GetLeagueBySlug(ctx, slug)
}

func (s *DBStore) GetAllLeagues(ctx context.Context, activeOnly bool) ([]models.League, error) {
	return s.queries.GetAllLeagues(ctx, activeOnly)
}

func (s *DBStore) UpdateLeagueSettings(ctx context.Context, arg models.UpdateLeagueSettingsParams) error {
	return s.queries.UpdateLeagueSettings(ctx, arg)
}

func (s *DBStore) SetCurrentSeason(ctx context.Context, arg models.SetCurrentSeasonParams) error {
	return s.queries.SetCurrentSeason(ctx, arg)
}

// Season operations

func (s *DBStore) CreateSeason(ctx context.Context, arg models.CreateSeasonParams) (models.LeagueSeason, error) {
	return s.queries.CreateSeason(ctx, arg)
}

func (s *DBStore) GetSeason(ctx context.Context, uuid uuid.UUID) (models.LeagueSeason, error) {
	return s.queries.GetSeason(ctx, uuid)
}

func (s *DBStore) GetCurrentSeason(ctx context.Context, leagueUUID uuid.UUID) (models.LeagueSeason, error) {
	return s.queries.GetCurrentSeason(ctx, leagueUUID)
}

func (s *DBStore) GetPastSeasons(ctx context.Context, leagueID uuid.UUID) ([]models.LeagueSeason, error) {
	return s.queries.GetPastSeasons(ctx, leagueID)
}

func (s *DBStore) GetSeasonsByLeague(ctx context.Context, leagueID uuid.UUID) ([]models.LeagueSeason, error) {
	return s.queries.GetSeasonsByLeague(ctx, leagueID)
}

func (s *DBStore) UpdateSeasonStatus(ctx context.Context, arg models.UpdateSeasonStatusParams) error {
	return s.queries.UpdateSeasonStatus(ctx, arg)
}

func (s *DBStore) MarkSeasonComplete(ctx context.Context, uuid uuid.UUID) error {
	return s.queries.MarkSeasonComplete(ctx, uuid)
}

// Division operations

func (s *DBStore) CreateDivision(ctx context.Context, arg models.CreateDivisionParams) (models.LeagueDivision, error) {
	return s.queries.CreateDivision(ctx, arg)
}

func (s *DBStore) GetDivision(ctx context.Context, uuid uuid.UUID) (models.LeagueDivision, error) {
	return s.queries.GetDivision(ctx, uuid)
}

func (s *DBStore) GetDivisionsBySeason(ctx context.Context, seasonID uuid.UUID) ([]models.LeagueDivision, error) {
	return s.queries.GetDivisionsBySeason(ctx, seasonID)
}

func (s *DBStore) UpdateDivisionPlayerCount(ctx context.Context, arg models.UpdateDivisionPlayerCountParams) error {
	return s.queries.UpdateDivisionPlayerCount(ctx, arg)
}

func (s *DBStore) MarkDivisionComplete(ctx context.Context, uuid uuid.UUID) error {
	return s.queries.MarkDivisionComplete(ctx, uuid)
}

// Registration operations

func (s *DBStore) RegisterPlayer(ctx context.Context, arg models.RegisterPlayerParams) (models.LeagueRegistration, error) {
	return s.queries.RegisterPlayer(ctx, arg)
}

func (s *DBStore) UnregisterPlayer(ctx context.Context, arg models.UnregisterPlayerParams) error {
	return s.queries.UnregisterPlayer(ctx, arg)
}

func (s *DBStore) GetPlayerRegistration(ctx context.Context, arg models.GetPlayerRegistrationParams) (models.LeagueRegistration, error) {
	return s.queries.GetPlayerRegistration(ctx, arg)
}

func (s *DBStore) GetSeasonRegistrations(ctx context.Context, seasonID uuid.UUID) ([]models.LeagueRegistration, error) {
	return s.queries.GetSeasonRegistrations(ctx, seasonID)
}

func (s *DBStore) GetDivisionRegistrations(ctx context.Context, divisionID uuid.UUID) ([]models.LeagueRegistration, error) {
	return s.queries.GetDivisionRegistrations(ctx, pgtype.UUID{Bytes: divisionID, Valid: true})
}

func (s *DBStore) UpdatePlayerDivision(ctx context.Context, arg models.UpdatePlayerDivisionParams) error {
	return s.queries.UpdatePlayerDivision(ctx, arg)
}

func (s *DBStore) UpdateRegistrationDivision(ctx context.Context, arg models.UpdateRegistrationDivisionParams) error {
	return s.queries.UpdateRegistrationDivision(ctx, arg)
}

func (s *DBStore) GetPlayerSeasonHistory(ctx context.Context, arg models.GetPlayerSeasonHistoryParams) ([]models.GetPlayerSeasonHistoryRow, error) {
	return s.queries.GetPlayerSeasonHistory(ctx, arg)
}

// Standings operations

func (s *DBStore) UpsertStanding(ctx context.Context, arg models.UpsertStandingParams) error {
	return s.queries.UpsertStanding(ctx, arg)
}

func (s *DBStore) GetStandings(ctx context.Context, divisionID uuid.UUID) ([]models.LeagueStanding, error) {
	return s.queries.GetStandings(ctx, divisionID)
}

func (s *DBStore) GetPlayerStanding(ctx context.Context, arg models.GetPlayerStandingParams) (models.LeagueStanding, error) {
	return s.queries.GetPlayerStanding(ctx, arg)
}

func (s *DBStore) DeleteDivisionStandings(ctx context.Context, divisionID uuid.UUID) error {
	return s.queries.DeleteDivisionStandings(ctx, divisionID)
}

// Game queries

func (s *DBStore) GetLeagueGames(ctx context.Context, divisionID uuid.UUID) ([]models.Game, error) {
	return s.queries.GetLeagueGames(ctx, pgtype.UUID{Bytes: divisionID, Valid: true})
}

func (s *DBStore) GetLeagueGamesByStatus(ctx context.Context, arg models.GetLeagueGamesByStatusParams) ([]models.Game, error) {
	return s.queries.GetLeagueGamesByStatus(ctx, arg)
}

func (s *DBStore) CountDivisionGamesComplete(ctx context.Context, divisionID uuid.UUID) (int64, error) {
	return s.queries.CountDivisionGamesComplete(ctx, pgtype.UUID{Bytes: divisionID, Valid: true})
}

func (s *DBStore) CountDivisionGamesTotal(ctx context.Context, divisionID uuid.UUID) (int64, error) {
	return s.queries.CountDivisionGamesTotal(ctx, pgtype.UUID{Bytes: divisionID, Valid: true})
}
