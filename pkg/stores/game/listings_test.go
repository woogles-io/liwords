package game

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
)

// giveLiveGamesRows gives every live fixture game its ongoing row, as every
// live game in production has, and returns their ids. Saved rather than
// backfilled: the fixtures were inserted without games.player_on_turn, which a
// save writes and the bot listings return.
func giveLiveGamesRows(t *testing.T, s *DBStore) []string {
	t.Helper()
	ctx := testCtx()
	ids, err := s.ListLiveGamesWithoutOngoingRow(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		g, err := s.Get(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Set(ctx, g); err != nil {
			t.Fatal(err)
		}
	}
	left, err := s.ListLiveGamesWithoutOngoingRow(ctx)
	if err != nil || len(left) != 0 {
		t.Fatalf("live games still without a row: %v (%v)", left, err)
	}
	return ids
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

// Each listing returns the games it filters for and no ended game. The
// fixture game is cloned into each kind of game a listing filters on --
// correspondence, tournament, league, a bot on turn, and an ended game that
// must appear in none.
func TestListings(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	wctx := testCtx()

	ids := giveLiveGamesRows(t, gstore)
	_, err := gstore.queries.GetOngoingGame(wctx, ongoingGID)
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

	q := gstore.queries

	// same runs one listing and requires the expected number of games (a
	// non-empty result for want < 0, so a filter that matches nothing cannot
	// pass by accident), and that no ended game is listed.
	same := func(name string, list func(ctx context.Context) (any, error), want int) {
		t.Helper()
		b, err := list(wctx)
		is.NoErr(err)
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
	same("ActiveGames", func(ctx context.Context) (any, error) { return q.ListActiveGames(ctx) }, nRealTime)
	same("ActiveRealTimeGamesForAdjudication", func(ctx context.Context) (any, error) {
		return q.ListActiveRealTimeGamesForAdjudication(ctx)
	}, nRealTime)
	same("ActiveTournamentGames", func(ctx context.Context) (any, error) {
		return q.ListActiveTournamentGames(ctx, "listTourney")
	}, 1)
	same("ActiveCorrespondenceGames", func(ctx context.Context) (any, error) {
		return q.ListActiveCorrespondenceGames(ctx)
	}, nCorres)
	same("ActiveCorrespondenceGamesForUser", func(ctx context.Context) (any, error) {
		return q.ListActiveCorrespondenceGamesForUser(ctx, userUUID)
	}, nCorres)
	same("ActiveCorrespondenceGamesForUserAndLeague", func(ctx context.Context) (any, error) {
		return q.ListActiveCorrespondenceGamesForUserAndLeague(ctx, models.ListActiveCorrespondenceGamesForUserAndLeagueParams{
			LeagueID: leagueID, UserUuid: userUUID,
		})
	}, 1)
	same("ActiveCorrespondenceGamesWithBotOnTurn", func(ctx context.Context) (any, error) {
		return q.ListActiveCorrespondenceGamesWithBotOnTurn(ctx)
	}, 2) // corres + league
	same("ActiveRealtimeGamesWithBotOnTurn", func(ctx context.Context) (any, error) {
		return q.ListActiveRealtimeGamesWithBotOnTurn(ctx)
	}, -1) // fixture and both tourneys, plus any other fixture game a bot is on turn in

	n, err := q.CountActiveCorrespondenceGames(wctx)
	is.NoErr(err)
	is.Equal(int(n), nCorres)
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

// The correspondence timeout pre-filter must never drop a game that
// Timers.RanOut would time out, and should drop the ones it would not. Each
// case is a correspondence clone with hand-set timers.
func TestCorrespondenceTimeoutCandidates(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	wctx := testCtx()

	giveLiveGamesRows(t, gstore)

	const now = int64(10_000_000_000)
	timers := func(sinceUpdate, bank int64, rtiat bool, withInc bool) string {
		inc := ""
		if withInc {
			inc = `, "inc": 60`
		}
		return fmt.Sprintf(`{"lu": %d, "ts": 1, "tr": [60000, 60000], "mo": 0, "tb": [%d, %d], "rtiat": %v%s}`,
			now-sinceUpdate, bank, bank, rtiat, inc)
	}
	noBank := func(sinceUpdate int64) string {
		return fmt.Sprintf(`{"lu": %d, "ts": 1, "tr": [60000, 60000], "mo": 0, "rtiat": true, "inc": 60}`, now-sinceUpdate)
	}
	cases := []struct {
		id        string
		timers    string
		candidate bool
	}{
		{"tcInTime00000000000000", timers(30_000, 0, true, true), false},
		{"tcOverNoBank0000000000", timers(120_000, 0, true, true), true},
		{"tcBankCovers0000000000", timers(120_000, 100_000, true, true), false},
		{"tcBankExactly000000000", timers(160_000, 100_000, true, true), false}, // deficit == bank: RanOut says no
		{"tcBankOneMsShort000000", timers(160_001, 100_000, true, true), true},
		{"tcBankExhausted0000000", timers(240_000, 100_000, true, true), true},
		{"tcNoIncKey000000000000", timers(30_000, 0, true, false), true}, // cannot judge: check it
		{"tcNotReset00000000000a", timers(30_000, 0, false, true), true}, // other RanOut branch: check it
		{"tcNoBankAtLimit0000000", noBank(60_000), false},
		{"tcNoBankOver000000000a", noBank(60_001), true},
	}
	// The request agrees with "inc", so a row without the key is filled with
	// the same 60 seconds on load.
	const corres = `game_request = jsonb_set(jsonb_set(game_request, '{game_mode}', '1'), '{increment_seconds}', '60')`
	for _, c := range cases {
		cloneLiveGame(t, gstore, c.id,
			corres+`, started = true, timers = '`+c.timers+`'`,
			`game_mode = 1, started = true, timers = '`+c.timers+`'`)
	}

	all, err := gstore.queries.ListActiveCorrespondenceGames(wctx)
	is.NoErr(err)
	got, err := gstore.ListCorrespondenceTimeoutCandidates(wctx, now)
	is.NoErr(err)
	in := map[string]bool{}
	for _, r := range got {
		in[r.Uuid.String] = true
	}

	for _, c := range cases {
		if in[c.id] != c.candidate {
			t.Errorf("%s: candidate = %v, want %v", c.id, in[c.id], c.candidate)
		}
	}
	// Whatever RanOut times out from the full list is among the candidates.
	timedOut := 0
	for _, r := range all {
		tm := r.Timers
		tm.FillIncrement(r.GameRequest.GameRequest)
		if r.PlayerOnTurn.Valid && tm.RanOut(int(r.PlayerOnTurn.Int32), now) {
			timedOut++
			if !in[r.Uuid.String] {
				t.Errorf("%s runs out of time but is not a candidate", r.Uuid.String)
			}
		}
	}
	is.Equal(timedOut, 4) // over-no-bank, one-ms-short, exhausted, no-bank-over
}
