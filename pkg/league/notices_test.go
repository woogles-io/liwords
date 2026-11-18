package league

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/outbox"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// outboxCounts counts the outbox messages by channel.
func outboxCounts(t *testing.T, ctx context.Context, allStores *stores.Stores) map[string]int {
	t.Helper()
	rows, err := allStores.DBPool.Query(ctx, "SELECT channel, count(*) FROM outbox GROUP BY channel")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var ch string
		var n int
		if err := rows.Scan(&ch, &n); err != nil {
			t.Fatal(err)
		}
		counts[ch] = n
	}
	return counts
}

func registerUsers(t *testing.T, ctx context.Context, allStores *stores.Stores, seasonID uuid.UUID, userIDs ...int32) {
	t.Helper()
	for _, id := range userIDs {
		_, err := allStores.LeagueStore.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:           id,
			SeasonID:         seasonID,
			RegistrationDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			Status:           pgtype.Text{String: "REGISTERED", Valid: true},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// Registration-open messages are written for a season the runner opened,
// once, however often the runner gets to them; a season opened any other way
// gets none.
func TestQueueRegistrationOpenNoticesOnce(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	allStores, store, cleanup := setupTest(t)
	defer cleanup()
	leagueID, otherSeason := createTestLeagueAndScheduledSeason(t, ctx, store)

	seasonID := uuid.New()
	_, err := store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:                      seasonID,
		LeagueID:                  leagueID,
		SeasonNumber:              2,
		StartDate:                 pgtype.Timestamptz{Time: time.Now().Add(48 * time.Hour), Valid: true},
		EndDate:                   pgtype.Timestamptz{Time: time.Now().AddDate(0, 1, 0), Valid: true},
		Status:                    int32(ipc.SeasonStatus_SEASON_REGISTRATION_OPEN),
		RegistrationNoticePending: true,
	})
	is.NoErr(err)
	registerUsers(t, ctx, allStores, seasonID, 1, 2)

	slm := NewSeasonLifecycleManager(allStores, RealClock{})
	written, err := slm.QueueRegistrationOpenNotices(ctx, leagueID, seasonID)
	is.NoErr(err)
	is.True(written)
	written, err = slm.QueueRegistrationOpenNotices(ctx, leagueID, seasonID)
	is.NoErr(err)
	is.True(!written)
	written, err = slm.QueueRegistrationOpenNotices(ctx, leagueID, otherSeason)
	is.NoErr(err)
	is.True(!written) // not opened by the runner

	counts := outboxCounts(t, ctx, allStores)
	is.Equal(counts["email"], 2)
	is.Equal(counts["league_discord"], 1)

	season, err := store.GetSeason(ctx, seasonID)
	is.NoErr(err)
	is.True(!season.RegistrationNoticePending)
	is.True(season.RegistrationOpenedAt.Valid)
}

// Starting-soon messages are written once, with the mark that says so.
func TestQueueSeasonStartingSoonNoticesOnce(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	allStores, store, cleanup := setupTest(t)
	defer cleanup()
	leagueID, seasonID := createTestLeagueAndScheduledSeason(t, ctx, store)
	registerUsers(t, ctx, allStores, seasonID, 1, 2, 3)

	slm := NewSeasonLifecycleManager(allStores, RealClock{})
	for i, want := range []bool{true, false} {
		written, err := slm.QueueSeasonStartingSoonNotices(ctx, leagueID, seasonID)
		is.NoErr(err)
		if written != want {
			t.Fatalf("call %d: written %v, want %v", i+1, written, want)
		}
	}
	is.Equal(outboxCounts(t, ctx, allStores)["email"], 3)
	season, err := store.GetSeason(ctx, seasonID)
	is.NoErr(err)
	is.True(season.StartingSoonNotificationSentAt.Valid)
}

// A cut-off run (ctx cancelled while the messages are being built) writes
// nothing and leaves the season unmarked, so the next run writes them all.
func TestQueueNoticesCutOffWritesNothing(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	allStores, store, cleanup := setupTest(t)
	defer cleanup()
	leagueID, seasonID := createTestLeagueAndScheduledSeason(t, ctx, store)
	registerUsers(t, ctx, allStores, seasonID, 1, 2)

	slm := NewSeasonLifecycleManager(allStores, RealClock{})
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	_, err := slm.QueueSeasonStartingSoonNotices(cancelled, leagueID, seasonID)
	is.True(err != nil)
	is.Equal(len(outboxCounts(t, ctx, allStores)), 0)

	written, err := slm.QueueSeasonStartingSoonNotices(ctx, leagueID, seasonID)
	is.NoErr(err)
	is.True(written)
	is.Equal(outboxCounts(t, ctx, allStores)["email"], 2)
}

// Each player gets each unstarted-game reminder once per season: writing
// the same reminders again adds nothing, and the firm one is a message of
// its own.
func TestUnstartedGameRemindersOnce(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	allStores, store, cleanup := setupTest(t)
	defer cleanup()
	_, seasonID := createTestLeagueAndScheduledSeason(t, ctx, store)

	players := []models.GetSeasonPlayersWithUnstartedGamesRow{{UserUuid: "test-uuid-1", UnstartedGameCount: 2}}
	for _, firm := range []bool{false, false, true, true} {
		msgs, err := unstartedGameReminderMessages(ctx, allStores.UserStore, seasonID, "L", "l", 1, players, firm)
		is.NoErr(err)
		is.Equal(len(msgs), 1)
		_, err = outbox.Add(ctx, allStores.Queries, msgs)
		is.NoErr(err)
	}
	is.Equal(outboxCounts(t, ctx, allStores)["email"], 2)
}
