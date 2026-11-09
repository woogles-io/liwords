package game

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
)

// readOngoingCtx turns on both flags, so the listings come from ongoing_games.
func readOngoingCtx() context.Context {
	cfg := *DefaultConfig
	cfg.WriteOngoingGames = true
	cfg.ReadOngoingGames = true
	return cfg.WithContext(context.Background())
}

// readGamesCtx keeps the writer on but the listings on games.
func readGamesCtx() context.Context {
	cfg := *DefaultConfig
	cfg.WriteOngoingGames = true
	return cfg.WithContext(context.Background())
}

// cloneLiveGame copies ongoingGID's games row and ongoing_games row under a
// new uuid, then applies the two updates, so one fixture game becomes one of
// each kind the listings filter on. An empty ongoingSet means the clone has
// no ongoing row, as for a game that has ended.
func cloneLiveGame(t *testing.T, s *DBStore, gid, gamesSet, ongoingSet string) {
	t.Helper()
	ctx := context.Background()
	stmts := []string{
		`CREATE TEMP TABLE g_clone AS SELECT * FROM games WHERE uuid = '` + ongoingGID + `'`,
		`UPDATE g_clone SET id = nextval(pg_get_serial_sequence('games', 'id')), uuid = '` + gid + `'`,
		`INSERT INTO games SELECT * FROM g_clone`,
		`DROP TABLE g_clone`,
	}
	if gamesSet != "" {
		stmts = append(stmts, `UPDATE games SET `+gamesSet+` WHERE uuid = '`+gid+`'`)
	}
	if ongoingSet != "" {
		stmts = append(stmts,
			`CREATE TEMP TABLE o_clone AS SELECT * FROM ongoing_games WHERE game_uuid = '`+ongoingGID+`'`,
			`UPDATE o_clone SET game_uuid = '`+gid+`'`,
			`INSERT INTO ongoing_games SELECT * FROM o_clone`,
			`DROP TABLE o_clone`,
			`UPDATE ongoing_games SET `+ongoingSet+` WHERE game_uuid = '`+gid+`'`,
		)
	}
	for _, q := range stmts {
		if _, err := s.dbPool.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
}

// Every listing must return exactly the same rows from either table. The
// fixture game is cloned into each kind of game a listing filters on --
// correspondence, tournament, league, a bot on turn, and an ended game that
// must appear in none -- and each listing is compared row for row.
func TestListingsAgreeAcrossTables(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	wctx := writeOngoingCtx()

	// Give every live fixture game its row. Saved rather than backfilled: the
	// fixtures were inserted without games.player_on_turn, which a save
	// writes, and the bot listings on games filter on that column. (In
	// production the two agree; the rollout checks compare them.)
	ids, err := gstore.ListLiveGamesWithoutOngoingRow(wctx)
	is.NoErr(err)
	for _, id := range ids {
		g, err := gstore.Get(wctx, id)
		is.NoErr(err)
		is.NoErr(gstore.Set(wctx, g))
	}
	left, err := gstore.ListLiveGamesWithoutOngoingRow(wctx)
	is.NoErr(err)
	is.Equal(len(left), 0)
	_, err = gstore.queries.GetOngoingGame(wctx, ongoingGID)
	is.NoErr(err) // the game every clone is made from

	leagueID := uuid.New()
	const corres = `game_request = jsonb_set(game_request, '{game_mode}', '1')`
	cloneLiveGame(t, gstore, "listCorres000000000000", corres, `game_mode = 1`)
	cloneLiveGame(t, gstore, "listTourney00000000000",
		`tournament_id = 'listTourney'`, `tournament_id = 'listTourney'`)
	cloneLiveGame(t, gstore, "listOtherTourney000000",
		`tournament_id = 'listOtherTourney'`, `tournament_id = 'listOtherTourney'`)
	cloneLiveGame(t, gstore, "listLeague000000000000",
		corres+`, league_id = '`+leagueID.String()+`'`,
		`game_mode = 1, league_id = '`+leagueID.String()+`'`)
	cloneLiveGame(t, gstore, "listOtherTurn000000000",
		`player_on_turn = 1 - player_on_turn`, `on_turn = 1 - on_turn`)
	cloneLiveGame(t, gstore, "listCorresOtherTurn000",
		corres+`, player_on_turn = 1 - player_on_turn`, `game_mode = 1, on_turn = 1 - on_turn`)
	// Ended: its row was deleted with the save that ended it.
	cloneLiveGame(t, gstore, "listEnded0000000000000", `game_end_reason = 1`, "")
	cloneLiveGame(t, gstore, "listEndedCorres0000000", corres+`, game_end_reason = 1`, "")

	// The player on turn in the fixture game is a bot, so the bot listings
	// have something in both modes.
	_, err = gstore.dbPool.Exec(wctx, `
		UPDATE users SET internal_bot = true WHERE id = (
			SELECT CASE WHEN player_on_turn = 0 THEN player0_id ELSE player1_id END
			FROM games WHERE uuid = $1)`, ongoingGID)
	is.NoErr(err)

	var userUUID string
	is.NoErr(gstore.dbPool.QueryRow(wctx,
		`SELECT u.uuid FROM users u JOIN games g ON g.player0_id = u.id WHERE g.uuid = $1`,
		ongoingGID).Scan(&userUUID))

	oldCtx, newCtx := readGamesCtx(), readOngoingCtx()
	q := gstore.queries

	// same runs one listing both ways and requires identical, non-empty
	// results (so a filter that matches nothing cannot pass by accident), and
	// that no ended game is listed.
	same := func(name string, list func(ctx context.Context) (any, error), want int) {
		t.Helper()
		a, err := list(oldCtx)
		is.NoErr(err)
		b, err := list(newCtx)
		is.NoErr(err)
		if !reflect.DeepEqual(a, b) {
			t.Errorf("%s differs:\n games:   %v\n ongoing: %v", name, uuidsOf(a), uuidsOf(b))
		}
		got := uuidsOf(b)
		if (want < 0 && len(got) == 0) || (want >= 0 && len(got) != want) {
			t.Errorf("%s: %d games, want %d: %v", name, len(got), want, got)
		}
		for _, id := range got {
			if id == "listEnded0000000000000" || id == "listEndedCorres0000000" {
				t.Errorf("%s lists an ended game", name)
			}
		}
	}
	nRealTime, nCorres := 4, 3 // fixture, two tourneys, other turn; corres, league, other turn
	nRealTime += len(ids) - 1  // other live fixture games, all real-time
	same("ActiveGames", func(ctx context.Context) (any, error) { return ActiveGames(ctx, q) }, nRealTime)
	same("ActiveRealTimeGamesForAdjudication", func(ctx context.Context) (any, error) {
		return ActiveRealTimeGamesForAdjudication(ctx, q)
	}, nRealTime)
	same("ActiveTournamentGames", func(ctx context.Context) (any, error) {
		return ActiveTournamentGames(ctx, q, "listTourney")
	}, 1)
	same("ActiveCorrespondenceGames", func(ctx context.Context) (any, error) {
		return ActiveCorrespondenceGames(ctx, q)
	}, nCorres)
	same("ActiveCorrespondenceGamesForUser", func(ctx context.Context) (any, error) {
		return ActiveCorrespondenceGamesForUser(ctx, q, userUUID)
	}, nCorres)
	same("ActiveCorrespondenceGamesForUserAndLeague", func(ctx context.Context) (any, error) {
		return ActiveCorrespondenceGamesForUserAndLeague(ctx, q, models.ListActiveCorrespondenceGamesForUserAndLeagueParams{
			LeagueID: leagueID, UserUuid: userUUID,
		})
	}, 1)
	same("ActiveCorrespondenceGamesWithBotOnTurn", func(ctx context.Context) (any, error) {
		return ActiveCorrespondenceGamesWithBotOnTurn(ctx, q)
	}, 2) // corres + league
	same("ActiveRealtimeGamesWithBotOnTurn", func(ctx context.Context) (any, error) {
		return ActiveRealtimeGamesWithBotOnTurn(ctx, q)
	}, -1) // fixture and both tourneys, plus any other fixture game a bot is on turn in

	a, err := CountActiveCorrespondenceGames(oldCtx, q)
	is.NoErr(err)
	b, err := CountActiveCorrespondenceGames(newCtx, q)
	is.NoErr(err)
	is.Equal(a, b)
	is.Equal(int(b), nCorres)
}

// The reads follow the writer: with WRITE_ONGOING_GAMES off the table is not
// kept up to date, so READ_ONGOING_GAMES alone must not use it.
func TestListingsIgnoreReadFlagWithoutWriter(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()

	// A row for a game that has ended: what a stale table looks like.
	wctx := writeOngoingCtx()
	entGame, err := gstore.Get(wctx, ongoingGID)
	is.NoErr(err)
	is.NoErr(gstore.Set(wctx, entGame))
	_, err = gstore.dbPool.Exec(wctx, `UPDATE games SET game_end_reason = 1 WHERE uuid = $1`, ongoingGID)
	is.NoErr(err)

	cfg := *DefaultConfig
	cfg.ReadOngoingGames = true // writer off
	ctx := cfg.WithContext(context.Background())
	rows, err := ActiveGames(ctx, gstore.queries)
	is.NoErr(err)
	for _, r := range rows {
		is.True(r.Uuid.String != ongoingGID) // read from games, not the stale row
	}
	// And with both on, the stale row is what would be read.
	rows, err = ActiveGames(readOngoingCtx(), gstore.queries)
	is.NoErr(err)
	is.True(contains(uuidsOf(rows), ongoingGID))
}

// uuidsOf pulls the game ids out of any listing result, for messages.
func uuidsOf(rows any) []string {
	var out []string
	v := reflect.ValueOf(rows)
	for i := 0; i < v.Len(); i++ {
		e := v.Index(i)
		if t, ok := e.Interface().(pgtype.Text); ok {
			out = append(out, t.String)
			continue
		}
		out = append(out, e.FieldByName("Uuid").Interface().(pgtype.Text).String)
	}
	return out
}
