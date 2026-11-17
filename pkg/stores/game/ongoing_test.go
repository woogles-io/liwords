package game

// ongoing_games has to track a live game exactly, because a listing that
// disagrees with the game it names is worse than no listing at all.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const ongoingGID = "wJxURccCgSAPivUvj4QdYL"

func ongoingCount(t *testing.T, gstore *DBStore) int {
	t.Helper()
	n, err := gstore.queries.CountOngoingGames(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return int(n)
}

// testCtx carries the default config, as the API's contexts carry its own.
func testCtx() context.Context {
	return DefaultConfig.WithContext(context.Background())
}

func TestOngoingRowFollowsTheGame(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := testCtx()

	is.Equal(ongoingCount(t, gstore), 0)

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.NoErr(gstore.Set(ctx, entGame))

	is.Equal(ongoingCount(t, gstore), 1)
	row, err := gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(int(row.PlayState), int(entGame.Playing()))
	is.Equal(int(row.OnTurn.Int16), entGame.PlayerOnTurn())
	is.Equal(row.Lexicon, entGame.History().Lexicon)

	// A move moves the row with it.
	before := len(entGame.History().Events)
	_, err = entGame.PlayScoringMove("8E", "AGUE", true)
	is.NoErr(err)
	is.NoErr(gstore.StageTurns(entGame, before, entGame.History().Events[before:]))
	is.NoErr(gstore.Set(ctx, entGame))

	row, err = gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(int(row.OnTurn.Int16), entGame.PlayerOnTurn())
}

// A finished game is not ongoing, and stops being listed in the same
// transaction that records that it ended.
func TestOngoingRowGoesWhenTheGameEnds(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := testCtx()

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.NoErr(gstore.Set(ctx, entGame))
	is.Equal(ongoingCount(t, gstore), 1)

	entGame.SetGameEndReason(pb.GameEndReason_STANDARD)
	entGame.SetPlaying(macondopb.PlayState_GAME_OVER)
	is.NoErr(gstore.Set(ctx, entGame))
	is.Equal(ongoingCount(t, gstore), 0)
}

// Readiness is ORed in one bit per player. A save landing between the two
// players readying must not wipe the first -- which is the bug games.ready_flag
// has today, and the reason the upsert leaves the column alone.
func TestOngoingSaveDoesNotClobberReadiness(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := testCtx()

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.NoErr(gstore.Set(ctx, entGame))

	flag, err := gstore.queries.SetOngoingGameReady(ctx, models.SetOngoingGameReadyParams{
		GameUuid: ongoingGID, PlayerIdx: 0,
	})
	is.NoErr(err)
	is.Equal(flag, int64(1))

	// Another save, before the second player is ready.
	is.NoErr(gstore.Set(ctx, entGame))

	row, err := gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(row.ReadyFlag, int64(1))
}

// Readiness is accumulated by the database, one bit per player, so a save
// between the two players readying must leave the first player's bit alone.
//
// UpdateGame used to write a literal 0 to games.ready_flag on every save --
// entity.Game has no such field, so there was nothing else to write. A meta
// event arriving while the first player waited was enough: the second player's
// ready then returned 2 rather than 3, StartGame never fired, and the first
// player had to ready up again.
func TestSaveDoesNotClobberReadiness(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := context.Background()

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)

	// The fixture arrives with bits already set; start from nobody ready.
	_, err = gstore.dbPool.Exec(ctx, "UPDATE games SET ready_flag = 0 WHERE uuid = $1", ongoingGID)
	is.NoErr(err)

	// Player 0 is ready.
	flag, err := gstore.SetReady(ctx, ongoingGID, 0)
	is.NoErr(err)
	is.Equal(flag, 1)

	// Anything at all saves the game while player 1 is still deciding.
	is.NoErr(gstore.Set(ctx, entGame))

	// Player 1 is ready. Both bits must be set, or the game never starts.
	flag, err = gstore.SetReady(ctx, ongoingGID, 1)
	is.NoErr(err)
	is.Equal(flag, 3)
}

func contains(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// A game that was live before the table existed, and is not saved again, gets
// its row from the backfill -- the same row a save would have written.
func TestOngoingBackfill(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := testCtx()

	// One player readied before the row existed.
	_, err := gstore.dbPool.Exec(ctx, "UPDATE games SET ready_flag = 1 WHERE uuid = $1", ongoingGID)
	is.NoErr(err)

	ids, err := gstore.ListLiveGamesWithoutOngoingRow(ctx)
	is.NoErr(err)
	is.True(contains(ids, ongoingGID))

	wrote, err := gstore.BackfillOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.True(wrote)

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	row, err := gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(int(row.PlayState), int(entGame.Playing()))
	is.Equal(int(row.OnTurn.Int16), entGame.PlayerOnTurn())
	is.Equal(row.Lexicon, entGame.History().Lexicon)
	is.Equal(row.Player0ID, int32(entGame.PlayerDBIDs[0]))
	is.Equal(row.ReadyFlag, int64(1)) // copied, not reset

	// Idempotent: the second run finds the row and leaves it.
	wrote, err = gstore.BackfillOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.True(!wrote)
	ids, err = gstore.ListLiveGamesWithoutOngoingRow(ctx)
	is.NoErr(err)
	is.True(!contains(ids, ongoingGID))
}

// A game that ended is not ongoing, so the backfill must not give it a row --
// whether the listing was read before it ended or after.
func TestOngoingBackfillSkipsEndedGames(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := testCtx()

	ids, err := gstore.ListLiveGamesWithoutOngoingRow(ctx)
	is.NoErr(err)
	is.True(contains(ids, ongoingGID))

	// It ends after the listing was read. It never had a row, so the save
	// that ends it has none to delete.
	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	entGame.SetGameEndReason(pb.GameEndReason_STANDARD)
	entGame.SetPlaying(macondopb.PlayState_GAME_OVER)
	is.NoErr(gstore.Set(ctx, entGame))

	wrote, err := gstore.BackfillOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.True(!wrote)
	is.Equal(ongoingCount(t, gstore), 0)

	ids, err = gstore.ListLiveGamesWithoutOngoingRow(ctx)
	is.NoErr(err)
	is.True(!contains(ids, ongoingGID))
}

// The first save of a game without a row copies readiness from the games row,
// and from then on SetReady keeps both in step.
func TestOngoingReadinessFollowsTheGame(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := testCtx()

	_, err := gstore.dbPool.Exec(ctx, "UPDATE games SET ready_flag = 0 WHERE uuid = $1", ongoingGID)
	is.NoErr(err)

	// Player 0 readies before the game has a row.
	flag, err := gstore.SetReady(ctx, ongoingGID, 0)
	is.NoErr(err)
	is.Equal(flag, 1)
	is.Equal(ongoingCount(t, gstore), 0)

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.NoErr(gstore.Set(ctx, entGame))
	row, err := gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(row.ReadyFlag, int64(1))

	// Player 1 readies once the row exists: both rows get the bit.
	flag, err = gstore.SetReady(ctx, ongoingGID, 1)
	is.NoErr(err)
	is.Equal(flag, 3)
	row, err = gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(row.ReadyFlag, int64(3))

	// Readying again changes no bits and reports the whole flag, so a start
	// that failed after both readied can be retried.
	flag, err = gstore.SetReady(ctx, ongoingGID, 1)
	is.NoErr(err)
	is.Equal(flag, 3)
	row, err = gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(row.ReadyFlag, int64(3))
}

// A game is ongoing from the moment it is created, not from its first save:
// that is StartGame, which waits for both players to ready.
func TestOngoingRowFromCreation(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := testCtx()

	const uid = "ongoingCreate000000000"
	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	entGame.History().Uid = uid
	entGame.Started = false
	is.NoErr(gstore.Create(ctx, entGame))

	row, err := gstore.queries.GetOngoingGame(ctx, uid)
	is.NoErr(err)
	is.Equal(row.Started, false)
	is.Equal(row.ReadyFlag, int64(0))
	is.Equal(row.Player0ID, int32(entGame.PlayerDBIDs[0]))
}

// The fixture's stored timers predate the increment. A load fills it from the
// request, and the next save writes it to both games and ongoing_games, so
// the adjudicator's SQL can read it.
func TestTimersIncrementFilledOnLoadAndSaved(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := testCtx()

	// Strip the key, as every pre-existing row is.
	_, err := gstore.dbPool.Exec(ctx, `UPDATE games SET timers = timers - 'inc',
		game_request = jsonb_set(game_request, '{increment_seconds}', '7') WHERE uuid = $1`, ongoingGID)
	is.NoErr(err)

	g, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(g.Timers.IncrementSeconds, int32(7))
	is.NoErr(gstore.Set(ctx, g))

	var gi, oi int
	is.NoErr(gstore.dbPool.QueryRow(ctx, `SELECT (timers->>'inc')::int FROM games WHERE uuid = $1`, ongoingGID).Scan(&gi))
	is.NoErr(gstore.dbPool.QueryRow(ctx, `SELECT (timers->>'inc')::int FROM ongoing_games WHERE game_uuid = $1`, ongoingGID).Scan(&oi))
	is.Equal(gi, 7)
	is.Equal(oi, 7)
}

// Adding time bank is a direct SQL update of games.timers. It must move the
// ongoing_games row with it, or the two disagree until the next move.
func TestAddTimeBankKeepsOngoingTimers(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := testCtx()

	season := uuid.New()
	_, err := gstore.dbPool.Exec(ctx, `UPDATE games SET season_id = $2,
		timers = jsonb_set(timers, '{tb}', '[1000, 2000]') WHERE uuid = $1`, ongoingGID, season)
	is.NoErr(err)
	g, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.NoErr(gstore.Set(ctx, g)) // creates the row with these timers

	var p0 int32
	is.NoErr(gstore.dbPool.QueryRow(ctx, `SELECT player0_id FROM games WHERE uuid = $1`, ongoingGID).Scan(&p0))
	sid := pgtype.UUID{Bytes: season, Valid: true}

	tbs := func() (string, string) {
		var a, b string
		is.NoErr(gstore.dbPool.QueryRow(ctx, `SELECT g.timers->>'tb', o.timers->>'tb'
			FROM games g JOIN ongoing_games o ON o.game_uuid = g.uuid WHERE g.uuid = $1`, ongoingGID).Scan(&a, &b))
		return a, b
	}

	n, err := gstore.queries.AddTimeBankAllPlayers(ctx, models.AddTimeBankAllPlayersParams{AdditionalMs: 5, SeasonID: sid})
	is.NoErr(err)
	is.Equal(n, int64(1))
	a, b := tbs()
	is.Equal(a, "[1005, 2005]")
	is.Equal(b, a)

	n, err = gstore.queries.AddTimeBankSinglePlayer(ctx, models.AddTimeBankSinglePlayerParams{
		PlayerID: p0, AdditionalMs: 10, SeasonID: sid})
	is.NoErr(err)
	is.Equal(n, int64(1))
	a, b = tbs()
	is.Equal(a, "[1015, 2005]")
	is.Equal(b, a)

	n, err = gstore.queries.AddTimeBankPlayerAndOpponent(ctx, models.AddTimeBankPlayerAndOpponentParams{
		PlayerID: p0, AdditionalMs: 1, SeasonID: sid})
	is.NoErr(err)
	is.Equal(n, int64(1))
	a, b = tbs()
	is.Equal(a, "[1016, 2006]")
	is.Equal(b, a)

	// The games to update are picked from ongoing_games: a game with no row is
	// not in progress, whatever its games row says, and is left alone.
	_, err = gstore.dbPool.Exec(ctx, `DELETE FROM ongoing_games`)
	is.NoErr(err)
	n, err = gstore.queries.AddTimeBankAllPlayers(ctx, models.AddTimeBankAllPlayersParams{AdditionalMs: 1, SeasonID: sid})
	is.NoErr(err)
	is.Equal(n, int64(0))
	var tb string
	is.NoErr(gstore.dbPool.QueryRow(ctx, `SELECT timers->>'tb' FROM games WHERE uuid = $1`, ongoingGID).Scan(&tb))
	is.Equal(tb, "[1016, 2006]")
}

// Create writes the ongoing row before season start tags the game with its
// league; the StartGame save must fill those columns in, or the game is
// missing from every league listing for its whole life.
func TestOngoingRowPicksUpLeagueAfterCreate(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := testCtx()

	g, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.True(g.SeasonID == nil)
	is.NoErr(gstore.Set(ctx, g)) // the row exists, with no league

	league, season, division := uuid.New(), uuid.New(), uuid.New()
	g.LeagueID, g.SeasonID, g.LeagueDivisionID = &league, &season, &division
	is.NoErr(gstore.Set(ctx, g))

	var l, s, d pgtype.UUID
	is.NoErr(gstore.dbPool.QueryRow(ctx, `SELECT league_id, season_id, league_division_id
		FROM ongoing_games WHERE game_uuid = $1`, ongoingGID).Scan(&l, &s, &d))
	is.Equal(uuid.UUID(l.Bytes), league)
	is.Equal(uuid.UUID(s.Bytes), season)
	is.Equal(uuid.UUID(d.Bytes), division)

	// Once set, a later save does not move the game to another league.
	other := uuid.New()
	g.SeasonID = &other
	is.NoErr(gstore.Set(ctx, g))
	is.NoErr(gstore.dbPool.QueryRow(ctx, `SELECT season_id FROM ongoing_games WHERE game_uuid = $1`,
		ongoingGID).Scan(&s))
	is.Equal(uuid.UUID(s.Bytes), season)
}
