package league

import (
	"context"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// mockLeagueStore implements league.Store for testing
type mockLeagueStore struct {
	divisions     map[uuid.UUID]models.LeagueDivision
	standings     map[uuid.UUID][]models.LeagueStanding
	gameResults   map[uuid.UUID][]models.GetDivisionGameResultsRow
	registrations map[uuid.UUID][]models.LeagueRegistration
}

func newMockLeagueStore() *mockLeagueStore {
	return &mockLeagueStore{
		divisions:     make(map[uuid.UUID]models.LeagueDivision),
		standings:     make(map[uuid.UUID][]models.LeagueStanding),
		gameResults:   make(map[uuid.UUID][]models.GetDivisionGameResultsRow),
		registrations: make(map[uuid.UUID][]models.LeagueRegistration),
	}
}

// Implement only the methods we need for testing

func (m *mockLeagueStore) GetDivisionsBySeason(ctx context.Context, seasonID uuid.UUID) ([]models.LeagueDivision, error) {
	var divs []models.LeagueDivision
	for _, div := range m.divisions {
		divs = append(divs, div)
	}
	return divs, nil
}

func (m *mockLeagueStore) GetDivisionRegistrations(ctx context.Context, divisionID uuid.UUID) ([]models.LeagueRegistration, error) {
	return m.registrations[divisionID], nil
}

func (m *mockLeagueStore) GetDivisionGameResults(ctx context.Context, divisionID uuid.UUID) ([]models.GetDivisionGameResultsRow, error) {
	return m.gameResults[divisionID], nil
}

func (m *mockLeagueStore) UpsertStanding(ctx context.Context, arg models.UpsertStandingParams) error {
	// Find existing or create new
	divStandings := m.standings[arg.DivisionID]
	found := false
	for i := range divStandings {
		if divStandings[i].UserID == arg.UserID {
			// Update existing
			divStandings[i].Rank = arg.Rank
			divStandings[i].Wins = arg.Wins
			divStandings[i].Losses = arg.Losses
			divStandings[i].Draws = arg.Draws
			divStandings[i].Spread = arg.Spread
			divStandings[i].GamesPlayed = arg.GamesPlayed
			found = true
			break
		}
	}
	if !found {
		// Add new
		divStandings = append(divStandings, models.LeagueStanding{
			DivisionID:  arg.DivisionID,
			UserID:      arg.UserID,
			Rank:        arg.Rank,
			Wins:        arg.Wins,
			Losses:      arg.Losses,
			Draws:       arg.Draws,
			Spread:      arg.Spread,
			GamesPlayed: arg.GamesPlayed,
			Result:      arg.Result,
		})
	}
	m.standings[arg.DivisionID] = divStandings
	return nil
}

func (m *mockLeagueStore) GetStandings(ctx context.Context, divisionID uuid.UUID) ([]models.LeagueStanding, error) {
	return m.standings[divisionID], nil
}

func (m *mockLeagueStore) GetGameLeagueInfo(ctx context.Context, gameUUID string) (models.GetGameLeagueInfoRow, error) {
	return models.GetGameLeagueInfoRow{}, nil
}

// Stubs for other interface methods
func (m *mockLeagueStore) CreateLeague(ctx context.Context, arg models.CreateLeagueParams) (models.League, error) {
	return models.League{}, nil
}
func (m *mockLeagueStore) GetLeagueByUUID(ctx context.Context, uuid uuid.UUID) (models.League, error) {
	return models.League{}, nil
}
func (m *mockLeagueStore) GetLeagueBySlug(ctx context.Context, slug string) (models.League, error) {
	return models.League{}, nil
}
func (m *mockLeagueStore) GetAllLeagues(ctx context.Context, activeOnly bool) ([]models.League, error) {
	return nil, nil
}
func (m *mockLeagueStore) UpdateLeagueSettings(ctx context.Context, arg models.UpdateLeagueSettingsParams) error {
	return nil
}
func (m *mockLeagueStore) SetCurrentSeason(ctx context.Context, arg models.SetCurrentSeasonParams) error {
	return nil
}
func (m *mockLeagueStore) CreateSeason(ctx context.Context, arg models.CreateSeasonParams) (models.LeagueSeason, error) {
	return models.LeagueSeason{}, nil
}
func (m *mockLeagueStore) GetSeason(ctx context.Context, uuid uuid.UUID) (models.LeagueSeason, error) {
	return models.LeagueSeason{}, nil
}
func (m *mockLeagueStore) GetCurrentSeason(ctx context.Context, leagueUUID uuid.UUID) (models.LeagueSeason, error) {
	return models.LeagueSeason{}, nil
}
func (m *mockLeagueStore) GetPastSeasons(ctx context.Context, leagueID uuid.UUID) ([]models.LeagueSeason, error) {
	return nil, nil
}
func (m *mockLeagueStore) GetSeasonsByLeague(ctx context.Context, leagueID uuid.UUID) ([]models.LeagueSeason, error) {
	return nil, nil
}
func (m *mockLeagueStore) GetSeasonByLeagueAndNumber(ctx context.Context, leagueID uuid.UUID, seasonNumber int32) (models.LeagueSeason, error) {
	return models.LeagueSeason{}, nil
}
func (m *mockLeagueStore) UpdateSeasonStatus(ctx context.Context, arg models.UpdateSeasonStatusParams) error {
	return nil
}
func (m *mockLeagueStore) MarkSeasonComplete(ctx context.Context, uuid uuid.UUID) error { return nil }
func (m *mockLeagueStore) CreateDivision(ctx context.Context, arg models.CreateDivisionParams) (models.LeagueDivision, error) {
	return models.LeagueDivision{}, nil
}
func (m *mockLeagueStore) GetDivision(ctx context.Context, uuid uuid.UUID) (models.LeagueDivision, error) {
	return models.LeagueDivision{}, nil
}
func (m *mockLeagueStore) UpdateDivisionPlayerCount(ctx context.Context, arg models.UpdateDivisionPlayerCountParams) error {
	return nil
}
func (m *mockLeagueStore) MarkDivisionComplete(ctx context.Context, uuid uuid.UUID) error { return nil }
func (m *mockLeagueStore) DeleteDivision(ctx context.Context, uuid uuid.UUID) error       { return nil }
func (m *mockLeagueStore) UpdateDivisionNumber(ctx context.Context, arg models.UpdateDivisionNumberParams) error {
	return nil
}
func (m *mockLeagueStore) RegisterPlayer(ctx context.Context, arg models.RegisterPlayerParams) (models.LeagueRegistration, error) {
	return models.LeagueRegistration{}, nil
}
func (m *mockLeagueStore) UnregisterPlayer(ctx context.Context, arg models.UnregisterPlayerParams) error {
	return nil
}
func (m *mockLeagueStore) GetPlayerRegistration(ctx context.Context, arg models.GetPlayerRegistrationParams) (models.LeagueRegistration, error) {
	return models.LeagueRegistration{}, nil
}
func (m *mockLeagueStore) GetSeasonRegistrations(ctx context.Context, seasonID uuid.UUID) ([]models.LeagueRegistration, error) {
	return nil, nil
}
func (m *mockLeagueStore) UpdatePlayerDivision(ctx context.Context, arg models.UpdatePlayerDivisionParams) error {
	return nil
}
func (m *mockLeagueStore) UpdateRegistrationDivision(ctx context.Context, arg models.UpdateRegistrationDivisionParams) error {
	return nil
}
func (m *mockLeagueStore) UpdatePlacementStatus(ctx context.Context, arg models.UpdatePlacementStatusParams) error {
	return nil
}
func (m *mockLeagueStore) UpdatePlacementStatusWithSeasonsAway(ctx context.Context, arg models.UpdatePlacementStatusWithSeasonsAwayParams) error {
	return nil
}
func (m *mockLeagueStore) GetPlayerSeasonHistory(ctx context.Context, arg models.GetPlayerSeasonHistoryParams) ([]models.GetPlayerSeasonHistoryRow, error) {
	return nil, nil
}
func (m *mockLeagueStore) GetPlayerStanding(ctx context.Context, arg models.GetPlayerStandingParams) (models.LeagueStanding, error) {
	return models.LeagueStanding{}, nil
}
func (m *mockLeagueStore) DeleteDivisionStandings(ctx context.Context, divisionID uuid.UUID) error {
	return nil
}
func (m *mockLeagueStore) GetLeagueGames(ctx context.Context, divisionID uuid.UUID) ([]models.Game, error) {
	return nil, nil
}
func (m *mockLeagueStore) GetLeagueGamesByStatus(ctx context.Context, arg models.GetLeagueGamesByStatusParams) ([]models.Game, error) {
	return nil, nil
}
func (m *mockLeagueStore) CountDivisionGamesComplete(ctx context.Context, divisionID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockLeagueStore) CountDivisionGamesTotal(ctx context.Context, divisionID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockLeagueStore) GetUnfinishedLeagueGames(ctx context.Context, seasonID uuid.UUID) ([]models.GetUnfinishedLeagueGamesRow, error) {
	return nil, nil
}
func (m *mockLeagueStore) ForceFinishGame(ctx context.Context, arg models.ForceFinishGameParams) error {
	return nil
}

func (m *mockLeagueStore) IncrementStandingsAtomic(ctx context.Context, arg models.IncrementStandingsAtomicParams) error {
	// Find existing or create new standing
	divStandings := m.standings[arg.DivisionID]
	found := false
	for i := range divStandings {
		if divStandings[i].UserID == arg.UserID {
			// Atomically increment values
			divStandings[i].Wins.Int32 += arg.Wins.Int32
			divStandings[i].Losses.Int32 += arg.Losses.Int32
			divStandings[i].Draws.Int32 += arg.Draws.Int32
			divStandings[i].Spread.Int32 += arg.Spread.Int32
			divStandings[i].GamesPlayed.Int32++
			found = true
			break
		}
	}
	if !found {
		// Create new standing
		divStandings = append(divStandings, models.LeagueStanding{
			DivisionID:     arg.DivisionID,
			UserID:         arg.UserID,
			Rank:           pgtype.Int4{Int32: 0, Valid: true},
			Wins:           arg.Wins,
			Losses:         arg.Losses,
			Draws:          arg.Draws,
			Spread:         arg.Spread,
			GamesPlayed:    pgtype.Int4{Int32: 1, Valid: true},
			GamesRemaining: arg.GamesRemaining,
		})
	}
	m.standings[arg.DivisionID] = divStandings
	return nil
}

func (m *mockLeagueStore) RecalculateRanks(ctx context.Context, divisionID uuid.UUID) error {
	divStandings := m.standings[divisionID]
	if len(divStandings) == 0 {
		return nil
	}

	// Sort by wins (desc), then spread (desc)
	sort.Slice(divStandings, func(i, j int) bool {
		// Calculate effective wins (wins + 0.5*draws)
		iWins := float64(divStandings[i].Wins.Int32) + 0.5*float64(divStandings[i].Draws.Int32)
		jWins := float64(divStandings[j].Wins.Int32) + 0.5*float64(divStandings[j].Draws.Int32)

		if iWins != jWins {
			return iWins > jWins
		}
		return divStandings[i].Spread.Int32 > divStandings[j].Spread.Int32
	})

	// Assign ranks
	for i := range divStandings {
		divStandings[i].Rank.Int32 = int32(i + 1)
		divStandings[i].Rank.Valid = true
	}

	m.standings[divisionID] = divStandings
	return nil
}

var _ league.Store = (*mockLeagueStore)(nil) // Compile-time interface check

func TestStandingsCalculation_SimpleWinsLosses(t *testing.T) {
	ctx := context.Background()
	store := newMockLeagueStore()
	mgr := NewStandingsManager(store)

	divisionID := uuid.New()
	seasonID := uuid.New()

	// Setup: 4 players in a division
	store.divisions[divisionID] = models.LeagueDivision{
		Uuid:           divisionID,
		SeasonID:       seasonID,
		DivisionNumber: 1,
	}

	// Players: Alice (1), Bob (2), Carol (3), Dave (4)
	store.registrations[divisionID] = []models.LeagueRegistration{
		{UserID: "1", DivisionID: pgtype.UUID{Bytes: divisionID, Valid: true}},
		{UserID: "2", DivisionID: pgtype.UUID{Bytes: divisionID, Valid: true}},
		{UserID: "3", DivisionID: pgtype.UUID{Bytes: divisionID, Valid: true}},
		{UserID: "4", DivisionID: pgtype.UUID{Bytes: divisionID, Valid: true}},
	}

	// Game results:
	// Alice beats Bob 450-400
	// Alice beats Carol 460-380
	// Carol beats Bob 420-410
	// Dave beats Bob 430-400
	// Alice beats Dave 470-420
	// Carol beats Dave 440-410
	store.gameResults[divisionID] = []models.GetDivisionGameResultsRow{
		{Player0ID: pgtype.Int4{Int32: 1, Valid: true}, Player1ID: pgtype.Int4{Int32: 2, Valid: true},
			Player0Score: 450, Player1Score: 400, Player0Won: pgtype.Bool{Bool: true, Valid: true}},
		{Player0ID: pgtype.Int4{Int32: 1, Valid: true}, Player1ID: pgtype.Int4{Int32: 3, Valid: true},
			Player0Score: 460, Player1Score: 380, Player0Won: pgtype.Bool{Bool: true, Valid: true}},
		{Player0ID: pgtype.Int4{Int32: 3, Valid: true}, Player1ID: pgtype.Int4{Int32: 2, Valid: true},
			Player0Score: 420, Player1Score: 410, Player0Won: pgtype.Bool{Bool: true, Valid: true}},
		{Player0ID: pgtype.Int4{Int32: 4, Valid: true}, Player1ID: pgtype.Int4{Int32: 2, Valid: true},
			Player0Score: 430, Player1Score: 400, Player0Won: pgtype.Bool{Bool: true, Valid: true}},
		{Player0ID: pgtype.Int4{Int32: 1, Valid: true}, Player1ID: pgtype.Int4{Int32: 4, Valid: true},
			Player0Score: 470, Player1Score: 420, Player0Won: pgtype.Bool{Bool: true, Valid: true}},
		{Player0ID: pgtype.Int4{Int32: 3, Valid: true}, Player1ID: pgtype.Int4{Int32: 4, Valid: true},
			Player0Score: 440, Player1Score: 410, Player0Won: pgtype.Bool{Bool: true, Valid: true}},
	}

	// Calculate standings
	err := mgr.CalculateAndSaveStandings(ctx, seasonID)
	require.NoError(t, err)

	// Verify standings
	standings := store.standings[divisionID]
	require.Len(t, standings, 4)

	// Expected: Alice 3-0 (+160), Carol 2-1 (+60), Dave 1-2 (-40), Bob 0-3 (-180)
	standingsMap := make(map[string]models.LeagueStanding)
	for _, s := range standings {
		standingsMap[s.UserID] = s
	}

	// Alice should be rank 1
	alice := standingsMap["1"]
	assert.Equal(t, int32(1), alice.Rank.Int32)
	assert.Equal(t, int32(3), alice.Wins.Int32)
	assert.Equal(t, int32(0), alice.Losses.Int32)
	assert.Equal(t, int32(180), alice.Spread.Int32) // (450-400)=50 + (460-380)=80 + (470-420)=50
	assert.Equal(t, int32(3), alice.GamesPlayed.Int32)

	// Carol should be rank 2
	carol := standingsMap["3"]
	assert.Equal(t, int32(2), carol.Rank.Int32)
	assert.Equal(t, int32(2), carol.Wins.Int32)
	assert.Equal(t, int32(1), carol.Losses.Int32)
	assert.Equal(t, int32(-40), carol.Spread.Int32) // (380-460)=-80 + (420-410)=10 + (440-410)=30
	assert.Equal(t, int32(3), carol.GamesPlayed.Int32)

	// Dave should be rank 3
	dave := standingsMap["4"]
	assert.Equal(t, int32(3), dave.Rank.Int32)
	assert.Equal(t, int32(1), dave.Wins.Int32)
	assert.Equal(t, int32(2), dave.Losses.Int32)
	assert.Equal(t, int32(-50), dave.Spread.Int32) // (430-400)=30 + (420-470)=-50 + (410-440)=-30
	assert.Equal(t, int32(3), dave.GamesPlayed.Int32)

	// Bob should be rank 4
	bob := standingsMap["2"]
	assert.Equal(t, int32(4), bob.Rank.Int32)
	assert.Equal(t, int32(0), bob.Wins.Int32)
	assert.Equal(t, int32(3), bob.Losses.Int32)
	assert.Equal(t, int32(-90), bob.Spread.Int32) // (400-450)=-50 + (410-420)=-10 + (400-430)=-30
	assert.Equal(t, int32(3), bob.GamesPlayed.Int32)
}

func TestStandingsCalculation_WithTies(t *testing.T) {
	ctx := context.Background()
	store := newMockLeagueStore()
	mgr := NewStandingsManager(store)

	divisionID := uuid.New()
	seasonID := uuid.New()

	store.divisions[divisionID] = models.LeagueDivision{
		Uuid:           divisionID,
		SeasonID:       seasonID,
		DivisionNumber: 1,
	}

	// 3 players
	store.registrations[divisionID] = []models.LeagueRegistration{
		{UserID: "1", DivisionID: pgtype.UUID{Bytes: divisionID, Valid: true}},
		{UserID: "2", DivisionID: pgtype.UUID{Bytes: divisionID, Valid: true}},
		{UserID: "3", DivisionID: pgtype.UUID{Bytes: divisionID, Valid: true}},
	}

	// Game results with one tie:
	// Alice beats Bob 450-400
	// Alice ties Carol 430-430
	// Carol beats Bob 440-410
	store.gameResults[divisionID] = []models.GetDivisionGameResultsRow{
		{Player0ID: pgtype.Int4{Int32: 1, Valid: true}, Player1ID: pgtype.Int4{Int32: 2, Valid: true},
			Player0Score: 450, Player1Score: 400, Player0Won: pgtype.Bool{Bool: true, Valid: true}},
		{Player0ID: pgtype.Int4{Int32: 1, Valid: true}, Player1ID: pgtype.Int4{Int32: 3, Valid: true},
			Player0Score: 430, Player1Score: 430, Player0Won: pgtype.Bool{Valid: false}}, // Tie (won = null)
		{Player0ID: pgtype.Int4{Int32: 3, Valid: true}, Player1ID: pgtype.Int4{Int32: 2, Valid: true},
			Player0Score: 440, Player1Score: 410, Player0Won: pgtype.Bool{Bool: true, Valid: true}},
	}

	err := mgr.CalculateAndSaveStandings(ctx, seasonID)
	require.NoError(t, err)

	standings := store.standings[divisionID]
	require.Len(t, standings, 3)

	standingsMap := make(map[string]models.LeagueStanding)
	for _, s := range standings {
		standingsMap[s.UserID] = s
	}

	// Alice: 1 win, 1 tie = 1.5 wins, spread +50
	alice := standingsMap["1"]
	assert.Equal(t, int32(1), alice.Rank.Int32)
	assert.Equal(t, int32(1), alice.Wins.Int32) // Actual wins
	assert.Equal(t, int32(0), alice.Losses.Int32)
	assert.Equal(t, int32(1), alice.Draws.Int32)
	assert.Equal(t, int32(50), alice.Spread.Int32) // (450-400) + (430-430)

	// Carol: 1 win, 1 tie = 1.5 wins, spread +30
	carol := standingsMap["3"]
	assert.Equal(t, int32(2), carol.Rank.Int32)
	assert.Equal(t, int32(1), carol.Wins.Int32)
	assert.Equal(t, int32(0), carol.Losses.Int32)
	assert.Equal(t, int32(1), carol.Draws.Int32)
	assert.Equal(t, int32(30), carol.Spread.Int32) // (430-430) + (440-410)

	// Bob: 0 wins, 2 losses = 0 wins, spread -80
	bob := standingsMap["2"]
	assert.Equal(t, int32(3), bob.Rank.Int32)
	assert.Equal(t, int32(0), bob.Wins.Int32)
	assert.Equal(t, int32(2), bob.Losses.Int32)
	assert.Equal(t, int32(0), bob.Draws.Int32)
	assert.Equal(t, int32(-80), bob.Spread.Int32) // (400-450) + (410-440)
}

func TestStandingsCalculation_TimeoutLoss(t *testing.T) {
	ctx := context.Background()
	store := newMockLeagueStore()
	mgr := NewStandingsManager(store)

	divisionID := uuid.New()
	seasonID := uuid.New()

	store.divisions[divisionID] = models.LeagueDivision{
		Uuid:           divisionID,
		SeasonID:       seasonID,
		DivisionNumber: 1,
	}

	store.registrations[divisionID] = []models.LeagueRegistration{
		{UserID: "1", DivisionID: pgtype.UUID{Bytes: divisionID, Valid: true}},
		{UserID: "2", DivisionID: pgtype.UUID{Bytes: divisionID, Valid: true}},
	}

	// Alice has higher score but timed out (won=false)
	// This tests that we use the "won" column correctly
	store.gameResults[divisionID] = []models.GetDivisionGameResultsRow{
		{Player0ID: pgtype.Int4{Int32: 1, Valid: true}, Player1ID: pgtype.Int4{Int32: 2, Valid: true},
			Player0Score: 480, Player1Score: 400,
			Player0Won: pgtype.Bool{Bool: false, Valid: true}}, // Alice timed out despite higher score
	}

	err := mgr.CalculateAndSaveStandings(ctx, seasonID)
	require.NoError(t, err)

	standings := store.standings[divisionID]
	standingsMap := make(map[string]models.LeagueStanding)
	for _, s := range standings {
		standingsMap[s.UserID] = s
	}

	// Bob should win despite lower score
	bob := standingsMap["2"]
	assert.Equal(t, int32(1), bob.Rank.Int32)
	assert.Equal(t, int32(1), bob.Wins.Int32)
	assert.Equal(t, int32(0), bob.Losses.Int32)
	assert.Equal(t, int32(-80), bob.Spread.Int32) // Spread still reflects actual scores

	// Alice should lose despite higher score
	alice := standingsMap["1"]
	assert.Equal(t, int32(2), alice.Rank.Int32)
	assert.Equal(t, int32(0), alice.Wins.Int32)
	assert.Equal(t, int32(1), alice.Losses.Int32)
	assert.Equal(t, int32(80), alice.Spread.Int32) // Spread still reflects actual scores
}

func TestIncrementalStandings_FirstGame(t *testing.T) {
	ctx := context.Background()
	store := newMockLeagueStore()
	mgr := NewStandingsManager(store)

	divisionID := uuid.New()

	// No existing standings, first game of the season
	// Alice beats Bob 450-400
	err := mgr.UpdateStandingsIncremental(ctx, divisionID, 1, 2, 0, 450, 400)
	require.NoError(t, err)

	standings := store.standings[divisionID]
	require.Len(t, standings, 2)

	standingsMap := make(map[string]models.LeagueStanding)
	for _, s := range standings {
		standingsMap[s.UserID] = s
	}

	// Alice should be rank 1
	alice := standingsMap["1"]
	assert.Equal(t, int32(1), alice.Rank.Int32)
	assert.Equal(t, int32(1), alice.Wins.Int32)
	assert.Equal(t, int32(0), alice.Losses.Int32)
	assert.Equal(t, int32(50), alice.Spread.Int32)
	assert.Equal(t, int32(1), alice.GamesPlayed.Int32)

	// Bob should be rank 2
	bob := standingsMap["2"]
	assert.Equal(t, int32(2), bob.Rank.Int32)
	assert.Equal(t, int32(0), bob.Wins.Int32)
	assert.Equal(t, int32(1), bob.Losses.Int32)
	assert.Equal(t, int32(-50), bob.Spread.Int32)
	assert.Equal(t, int32(1), bob.GamesPlayed.Int32)
}

func TestIncrementalStandings_RankChange(t *testing.T) {
	ctx := context.Background()
	store := newMockLeagueStore()
	mgr := NewStandingsManager(store)

	divisionID := uuid.New()

	// Setup initial standings after 2 games
	// Alice beat Bob, Bob beat Carol
	// Alice: 1-0, Bob: 1-1, Carol: 0-1
	store.standings[divisionID] = []models.LeagueStanding{
		{DivisionID: divisionID, UserID: "1", Rank: pgtype.Int4{Int32: 1, Valid: true},
			Wins: pgtype.Int4{Int32: 1, Valid: true}, Losses: pgtype.Int4{Int32: 0, Valid: true},
			Spread: pgtype.Int4{Int32: 50, Valid: true}, GamesPlayed: pgtype.Int4{Int32: 1, Valid: true}},
		{DivisionID: divisionID, UserID: "2", Rank: pgtype.Int4{Int32: 2, Valid: true},
			Wins: pgtype.Int4{Int32: 1, Valid: true}, Losses: pgtype.Int4{Int32: 1, Valid: true},
			Spread: pgtype.Int4{Int32: 0, Valid: true}, GamesPlayed: pgtype.Int4{Int32: 2, Valid: true}},
		{DivisionID: divisionID, UserID: "3", Rank: pgtype.Int4{Int32: 3, Valid: true},
			Wins: pgtype.Int4{Int32: 0, Valid: true}, Losses: pgtype.Int4{Int32: 1, Valid: true},
			Spread: pgtype.Int4{Int32: -50, Valid: true}, GamesPlayed: pgtype.Int4{Int32: 1, Valid: true}},
	}

	// Carol beats Alice 460-400
	// This should change the rankings!
	err := mgr.UpdateStandingsIncremental(ctx, divisionID, 3, 1, 0, 460, 400)
	require.NoError(t, err)

	standings := store.standings[divisionID]
	standingsMap := make(map[string]models.LeagueStanding)
	for _, s := range standings {
		standingsMap[s.UserID] = s
	}

	// Now all three have 1-1, ranked by spread:
	// Carol: 1-1, +10 (rank 1)
	// Bob: 1-1, +0 (rank 2)
	// Alice: 1-1, -10 (rank 3)
	carol := standingsMap["3"]
	assert.Equal(t, int32(1), carol.Rank.Int32, "Carol should be rank 1 (best spread)")
	assert.Equal(t, int32(1), carol.Wins.Int32)
	assert.Equal(t, int32(1), carol.Losses.Int32)
	assert.Equal(t, int32(10), carol.Spread.Int32) // -50 + 60

	bob := standingsMap["2"]
	assert.Equal(t, int32(2), bob.Rank.Int32, "Bob should be rank 2")
	assert.Equal(t, int32(1), bob.Wins.Int32)
	assert.Equal(t, int32(1), bob.Losses.Int32)
	assert.Equal(t, int32(0), bob.Spread.Int32)

	alice := standingsMap["1"]
	assert.Equal(t, int32(3), alice.Rank.Int32, "Alice should be rank 3 (worst spread)")
	assert.Equal(t, int32(1), alice.Wins.Int32)
	assert.Equal(t, int32(1), alice.Losses.Int32)
	assert.Equal(t, int32(-10), alice.Spread.Int32) // 50 - 60
}

func TestIncrementalStandings_TieGame(t *testing.T) {
	ctx := context.Background()
	store := newMockLeagueStore()
	mgr := NewStandingsManager(store)

	divisionID := uuid.New()

	// Alice and Bob tie 420-420
	err := mgr.UpdateStandingsIncremental(ctx, divisionID, 1, 2, -1, 420, 420)
	require.NoError(t, err)

	standings := store.standings[divisionID]
	standingsMap := make(map[string]models.LeagueStanding)
	for _, s := range standings {
		standingsMap[s.UserID] = s
	}

	// Both should have 0 wins, 0 losses, 1 draw
	alice := standingsMap["1"]
	assert.Equal(t, int32(0), alice.Wins.Int32)
	assert.Equal(t, int32(0), alice.Losses.Int32)
	assert.Equal(t, int32(1), alice.Draws.Int32)
	assert.Equal(t, int32(0), alice.Spread.Int32)

	bob := standingsMap["2"]
	assert.Equal(t, int32(0), bob.Wins.Int32)
	assert.Equal(t, int32(0), bob.Losses.Int32)
	assert.Equal(t, int32(1), bob.Draws.Int32)
	assert.Equal(t, int32(0), bob.Spread.Int32)
}
