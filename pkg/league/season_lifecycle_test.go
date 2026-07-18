package league

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	leaguestore "github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func createTestLeagueAndScheduledSeason(t *testing.T, ctx context.Context, store *leaguestore.DBStore) (uuid.UUID, uuid.UUID) {
	is := is.New(t)

	league := uuid.New()
	_, err := store.CreateLeague(ctx, models.CreateLeagueParams{
		Uuid:        league,
		Name:        "Test League",
		Description: pgtype.Text{String: "Test", Valid: true},
		Slug:        uuid.New().String(),
		Settings:    []byte(`{}`),
		IsActive:    pgtype.Bool{Bool: true, Valid: true},
		CreatedBy:   pgtype.Int8{Int64: 1, Valid: true},
	})
	is.NoErr(err)

	seasonID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         seasonID,
		LeagueID:     league,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 1, 0), Valid: true},
		Status:       int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	is.NoErr(err)

	return league, seasonID
}

// TestOpenRegistrationForSeason_FreshSeason verifies a season that has never
// had its divisions prepared can have registration opened normally.
func TestOpenRegistrationForSeason_FreshSeason(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	allStores, store, cleanup := setupTest(t)
	defer cleanup()

	_, seasonID := createTestLeagueAndScheduledSeason(t, ctx, store)

	slm := NewSeasonLifecycleManager(allStores, RealClock{})
	_, err := slm.OpenRegistrationForSeason(ctx, seasonID)
	is.NoErr(err)

	season, err := store.GetSeason(ctx, seasonID)
	is.NoErr(err)
	is.Equal(season.Status, int32(ipc.SeasonStatus_SEASON_REGISTRATION_OPEN))
}

// TestOpenRegistrationForSeason_RejectsAlreadyPreparedSeason is a regression
// test for the incident where clicking "Open Registration" on a season whose
// divisions were already built (status SCHEDULED, awaiting start) reverted it
// to REGISTRATION_OPEN, stranding it: the hourly cron requires SCHEDULED to
// start the season, and skips re-preparing divisions once DivisionsPreparedAt
// is set - so the season could never progress again without manual DB surgery.
func TestOpenRegistrationForSeason_RejectsAlreadyPreparedSeason(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	allStores, store, cleanup := setupTest(t)
	defer cleanup()

	_, seasonID := createTestLeagueAndScheduledSeason(t, ctx, store)

	err := store.MarkDivisionsPrepared(ctx, seasonID)
	is.NoErr(err)

	slm := NewSeasonLifecycleManager(allStores, RealClock{})
	_, err = slm.OpenRegistrationForSeason(ctx, seasonID)
	is.True(err != nil)

	// Status must be untouched - still SCHEDULED, not reverted.
	season, err := store.GetSeason(ctx, seasonID)
	is.NoErr(err)
	is.Equal(season.Status, int32(ipc.SeasonStatus_SEASON_SCHEDULED))
}
