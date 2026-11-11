package tournament

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	commondb "github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
	tl "github.com/woogles-io/liwords/pkg/tournament"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

var pkg = "tournament_store"

func recreateDB(t *testing.T) (*pgxpool.Pool, *DBStore) {
	t.Helper()
	if err := commondb.RecreateTestDB(pkg); err != nil {
		t.Fatal(err)
	}
	pool, err := commondb.OpenTestingDB(pkg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	store, err := NewDBStore(pool, nil)
	if err != nil {
		t.Fatal(err)
	}
	return pool, store
}

func testTournament(t *testing.T, creator uint) *entity.Tournament {
	t.Helper()
	raw, err := os.ReadFile("../../tournament/testdata/wtf5.json")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := (&DBStore{}).dbObjToEntity(&models.Tournament{Divisions: raw, Directors: []byte("{}")})
	if err != nil {
		t.Fatal(err)
	}
	// Ended yesterday, so it shows in both date-window listings.
	start := time.Now().Add(-72 * time.Hour).Truncate(time.Second)
	end := start.Add(48 * time.Hour)
	return &entity.Tournament{
		UUID:        "tid1",
		Name:        "Some Tournament",
		Description: "a description",
		Directors: &ipc.TournamentPersons{Persons: []*ipc.TournamentPerson{
			{Id: "director-uuid:director", Rating: 0},
		}},
		Divisions:          decoded.Divisions,
		Type:               entity.TypeStandard,
		Slug:               "/tournament/Some-Tournament",
		ExtraMeta:          &entity.TournamentMeta{PrivateAnalysis: true, Logo: "logo.png"},
		ScheduledStartTime: &start,
		ScheduledEndTime:   &end,
		CreatedBy:          creator,
	}
}

func encodedDivisions(t *testing.T, tm *entity.Tournament) string {
	t.Helper()
	dbt, err := (&DBStore{}).toDBObj(tm)
	if err != nil {
		t.Fatal(err)
	}
	return string(dbt.Divisions)
}

func TestCreateGetSet(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	pool, store := recreateDB(t)

	ustore, err := user.NewDBStore(pool)
	is.NoErr(err)
	is.NoErr(ustore.New(ctx, &entity.User{Username: "creator", Email: "creator@example.com", UUID: "creator-uuid"}))
	creator, err := ustore.GetByUUID(ctx, "creator-uuid")
	is.NoErr(err)

	tm := testTournament(t, creator.ID)
	is.NoErr(store.Create(ctx, tm))

	got, err := store.Get(ctx, "tid1")
	is.NoErr(err)
	is.Equal(got.Name, tm.Name)
	is.Equal(got.Description, tm.Description)
	is.Equal(got.Slug, tm.Slug)
	is.Equal(got.Type, tm.Type)
	is.Equal(got.Directors.Persons[0].Id, "director-uuid:director")
	is.Equal(got.ExtraMeta.Logo, "logo.png")
	is.True(got.ScheduledStartTime.Equal(*tm.ScheduledStartTime))
	is.True(got.ScheduledEndTime.Equal(*tm.ScheduledEndTime))
	is.True(!got.IsStarted)
	is.Equal(encodedDivisions(t, got), encodedDivisions(t, tm))

	// Slugs are case-insensitive.
	bySlug, err := store.GetBySlug(ctx, "/TOURNAMENT/some-tournament")
	is.NoErr(err)
	is.Equal(bySlug.UUID, "tid1")

	header, err := store.GetHeader(ctx, "tid1")
	is.NoErr(err)
	is.Equal(header.Name, tm.Name)
	is.Equal(header.Slug, tm.Slug)
	is.Equal(header.Type, tm.Type)
	is.Equal(header.Directors.Persons[0].Id, "director-uuid:director")
	is.True(header.ExtraMeta.PrivateAnalysis)
	is.True(header.ScheduledEndTime.Equal(*tm.ScheduledEndTime))

	id, err := store.GetUUIDBySlug(ctx, "/tournament/SOME-tournament")
	is.NoErr(err)
	is.Equal(id, "tid1")

	summaryByID, err := store.GetSummary(ctx, "tid1")
	is.NoErr(err)
	summaryBySlug, err := store.GetSummaryBySlug(ctx, "/tournament/some-TOURNAMENT")
	is.NoErr(err)
	for _, summary := range []*entity.Tournament{summaryByID, summaryBySlug} {
		is.Equal(summary.UUID, "tid1")
		is.Equal(len(summary.Divisions), len(tm.Divisions))
		for name, div := range summary.Divisions {
			full := tm.Divisions[name].DivisionManager.(*tl.ClassicDivision)
			stripped := div.DivisionManager.(*tl.ClassicDivision)
			is.Equal(len(stripped.PairingMap), 0)
			is.Equal(len(stripped.Players.Persons), len(full.Players.Persons))
			is.Equal(len(stripped.RoundControls), len(full.RoundControls))
		}
	}

	_, err = store.GetHeader(ctx, "nonexistent")
	is.True(err != nil)
	_, err = store.GetUUIDBySlug(ctx, "/tournament/nonexistent")
	is.True(err != nil)

	private, err := store.PrivateAnalysis(ctx, "tid1")
	is.NoErr(err)
	is.True(private)

	// Listings decode divisions without pairings and standings.
	for _, list := range []func(context.Context) ([]*entity.Tournament, error){
		store.GetRecentAndUpcomingTournaments,
		func(ctx context.Context) ([]*entity.Tournament, error) { return store.GetPastTournaments(ctx, 10) },
	} {
		listed, err := list(ctx)
		is.NoErr(err)
		is.Equal(len(listed), 1)
		is.Equal(listed[0].UUID, "tid1")
		for name, div := range listed[0].Divisions {
			full := tm.Divisions[name].DivisionManager.(*tl.ClassicDivision)
			stripped := div.DivisionManager.(*tl.ClassicDivision)
			is.True(len(full.PairingMap) > 0)
			is.Equal(len(stripped.PairingMap), 0)
			is.Equal(len(stripped.Players.Persons), len(full.Players.Persons))
		}
	}

	_, err = store.Get(ctx, "nonexistent")
	is.True(err != nil)

	// Clearing fields is saved. Gorm's Updates skipped empty strings and nil
	// times, so these used to stay in the database.
	got.Description = ""
	got.ScheduledStartTime = nil
	got.ScheduledEndTime = nil
	got.IsStarted = true
	is.NoErr(store.Set(ctx, got))

	again, err := store.Get(ctx, "tid1")
	is.NoErr(err)
	is.Equal(again.Description, "")
	is.True(again.ScheduledStartTime == nil)
	is.True(again.ScheduledEndTime == nil)
	is.True(again.IsStarted)
	is.Equal(encodedDivisions(t, again), encodedDivisions(t, tm))

	// The entity never loads created_by, so saving must not clear it.
	n, err := store.CountRecentTournamentsByUser(ctx, creator.ID)
	is.NoErr(err)
	is.Equal(n, int64(1))
}

func TestMonitoringStreams(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	_, store := recreateDB(t)

	tid, uid, err := store.FindTournamentByStreamKey(ctx, "nokey", "camera")
	is.NoErr(err)
	is.Equal(tid, "")
	is.Equal(uid, "")

	is.NoErr(store.InsertMonitoringStream(ctx, "tid1", "u1", "user1", "camera", "camkey"))
	is.NoErr(store.InsertMonitoringStream(ctx, "tid1", "u1", "user1", "screenshot", "shotkey"))

	tid, uid, err = store.FindTournamentByStreamKey(ctx, "camkey", "camera")
	is.NoErr(err)
	is.Equal(tid, "tid1")
	is.Equal(uid, "u1:user1")

	is.NoErr(store.UpdateMonitoringStreamStatus(ctx, "camkey", 2, 1762700000))
	data, err := store.GetMonitoringStream(ctx, "tid1", "u1")
	is.NoErr(err)
	is.Equal(data.CameraKey, "camkey")
	is.Equal(data.CameraStatus, ipc.StreamStatus(2))
	is.Equal(data.CameraTimestamp.AsTime().Unix(), int64(1762700000))
	is.Equal(data.ScreenshotKey, "shotkey")
	is.True(data.ScreenshotTimestamp == nil)

	active, err := store.GetActiveMonitoringStreams(ctx)
	is.NoErr(err)
	is.Equal(len(active), 1)
	is.Equal(active[0].StreamKey, "camkey")

	all, err := store.GetMonitoringStreams(ctx, "tid1")
	is.NoErr(err)
	is.Equal(len(all), 1)

	is.NoErr(store.DeleteMonitoringStreamsForTournament(ctx, "tid1"))
	data, err = store.GetMonitoringStream(ctx, "tid1", "u1")
	is.NoErr(err)
	is.True(data == nil)
}

func TestReadyStates(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	_, store := recreateDB(t)

	k := entity.TournamentPairingKey{TournamentID: "tid1", Division: "A", Round: 2,
		Players: [2]string{"u1:alice", "u2:bob"}}

	flag, conns, already, err := store.SetReady(ctx, k, 1, "bobconn")
	is.NoErr(err)
	is.True(!already)
	is.Equal(flag, 2)
	is.Equal(conns, [2]string{"", "bobconn"})

	// Readying twice changes nothing.
	_, _, already, err = store.SetReady(ctx, k, 1, "otherconn")
	is.NoErr(err)
	is.True(already)

	waiting, err := store.WaitingReady(ctx, "tid1")
	is.NoErr(err)
	is.Equal(len(waiting), 1)
	is.Equal(waiting[0].TournamentPairingKey, k)
	is.Equal(waiting[0].Flag, 2)

	// Closing another tab does nothing; closing Bob's unreadies him.
	left, err := store.UnreadyConnection(ctx, "someoneelse")
	is.NoErr(err)
	is.Equal(len(left), 0)
	left, err = store.UnreadyConnection(ctx, "bobconn")
	is.NoErr(err)
	is.Equal(len(left), 1)
	is.Equal(left[0].TournamentPairingKey, k)
	is.Equal(left[0].Flag, 2)
	waiting, err = store.WaitingReady(ctx, "tid1")
	is.NoErr(err)
	is.Equal(len(waiting), 0)

	// Unready only changes a player who is ready.
	changed, _, err := store.Unready(ctx, k, 0)
	is.NoErr(err)
	is.True(!changed)
	_, _, _, err = store.SetReady(ctx, k, 0, "aliceconn")
	is.NoErr(err)
	changed, conns, err = store.Unready(ctx, k, 0)
	is.NoErr(err)
	is.True(changed)
	is.Equal(conns, [2]string{"", ""})

	// Both ready: the game starts, and after that nothing unreadies them or
	// readies them again.
	_, _, _, err = store.SetReady(ctx, k, 0, "aliceconn")
	is.NoErr(err)
	flag, conns, _, err = store.SetReady(ctx, k, 1, "bobconn")
	is.NoErr(err)
	is.Equal(flag, 3)
	is.Equal(conns, [2]string{"aliceconn", "bobconn"})
	changed, _, err = store.Unready(ctx, k, 0)
	is.NoErr(err)
	is.True(!changed)
	left, err = store.UnreadyConnection(ctx, "aliceconn")
	is.NoErr(err)
	is.Equal(len(left), 0)
	_, _, already, err = store.SetReady(ctx, k, 0, "aliceconn")
	is.NoErr(err)
	is.True(already)

	// A cancelled game clears the pairing, found through either player.
	is.NoErr(store.ClearReady(ctx, "tid1", "A", 2, "u2:bob"))
	flag, _, already, err = store.SetReady(ctx, k, 0, "aliceconn")
	is.NoErr(err)
	is.True(!already)
	is.Equal(flag, 1)
}

// Players readying at the same moment, on any server: exactly one of them
// sees both ready, so exactly one game starts.
func TestReadyRace(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	_, store := recreateDB(t)

	for round := range 50 {
		k := entity.TournamentPairingKey{TournamentID: "tid1", Division: "A", Round: round,
			Players: [2]string{"u1:alice", "u2:bob"}}
		var wg sync.WaitGroup
		flags := make([]int, 2)
		errs := make([]error, 2)
		for i := range 2 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				flags[i], _, _, errs[i] = store.SetReady(ctx, k, i, fmt.Sprintf("conn%d", i))
			}()
		}
		wg.Wait()
		is.NoErr(errs[0])
		is.NoErr(errs[1])
		started := 0
		for _, f := range flags {
			if f == 3 {
				started++
			}
		}
		is.Equal(started, 1)
	}
}
