package rerate

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const pkg = "rerate"

func setupDB(t *testing.T) (*pgxpool.Pool, context.Context) {
	if err := common.RecreateTestDB(pkg); err != nil {
		t.Fatal(err)
	}
	pool, err := common.OpenTestingDB(pkg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO users (id, uuid, username, email, password, internal_bot) VALUES
			(1, 'ua', 'alice', 'a@x', 'x', false), (2, 'ub', 'bob', 'b@x', 'x', false),
			(3, 'uh', 'HastyBot', 'h@x', 'x', true)`,
		`INSERT INTO profiles (user_id, ratings) VALUES
			(1, '{"Data": {"NWL18.classic.regular": {"r": 1600, "rd": 80, "v": 0.06, "ts": 5}, "NWL18.puzzle.corres": {"r": 1400, "rd": 90, "v": 0.06, "ts": 5}}}'),
			(2, '{"Data": null}')`,
		`INSERT INTO user_actions (user_id, action_type, start_time, email_type) VALUES
			(2, 4, '2024-01-01T00:00:00Z', 0), (2, 1, '2024-01-02T00:00:00Z', 0)`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	return pool, ctx
}

// insertGame adds a game; pi0/pi1 are the players in the game's own order,
// which needn't match player0_id/player1_id.
func insertGame(t *testing.T, pool *pgxpool.Pool, id int, request string, reason, winner int,
	pi0, pi1 string, s0, s1 int, ended string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO games (id, uuid, player0_id, player1_id, game_end_reason, winner_idx, game_request,
			quickdata, created_at, updated_at)
		VALUES ($1::int, 'g' || $1::int, 2, 1, $2, $3, $4::jsonb,
			jsonb_build_object('s', jsonb_build_array($7::int, $8::int),
				'pi', jsonb_build_array(jsonb_build_object('user_id', $5::text), jsonb_build_object('user_id', $6::text))),
			$9::timestamptz - interval '1 hour', $9::timestamptz)`,
		id, reason, winner, request, pi0, pi1, s0, s1, ended)
	if err != nil {
		t.Fatal(err)
	}
}

const (
	snakeRated  = `{"lexicon": "NWL23", "rules": {"variant_name": "classic", "board_layout_name": "CrosswordGame"}, "initial_time_seconds": 1500, "max_overtime_minutes": 1}`
	camelRated  = `{"lexicon": "CSW21", "rules": {"boardLayoutName": "CrosswordGame"}, "initialTimeSeconds": 300}`
	camelCasual = `{"lexicon": "CSW21", "rules": {"boardLayoutName": "CrosswordGame"}, "ratingMode": "CASUAL", "initialTimeSeconds": 300}`
	corres      = `{"lexicon": "NWL23", "rules": {"board_layout_name": "CrosswordGame"}, "game_mode": "CORRESPONDENCE", "initial_time_seconds": 86400}`
)

func TestExtract(t *testing.T) {
	is := is.New(t)
	pool, ctx := setupDB(t)
	insertGame(t, pool, 1, snakeRated, int(pb.GameEndReason_STANDARD), 0, "ua", "ub", 400, 350, "2024-02-01T00:00:00Z")
	insertGame(t, pool, 2, camelRated, int(pb.GameEndReason_RESIGNED), 1, "ub", "uh", 100, 200, "2024-02-02T00:00:00Z")
	insertGame(t, pool, 3, camelCasual, int(pb.GameEndReason_STANDARD), 0, "ua", "ub", 400, 350, "2024-02-03T00:00:00Z")
	insertGame(t, pool, 4, snakeRated, int(pb.GameEndReason_ABORTED), -1, "ua", "ub", 0, 0, "2024-02-04T00:00:00Z")
	insertGame(t, pool, 5, corres, int(pb.GameEndReason_NONE), -1, "ua", "ub", 0, 0, "2024-02-05T00:00:00Z")

	// A rated game with a player we don't know.
	insertGame(t, pool, 8, snakeRated, int(pb.GameEndReason_STANDARD), 0, "ua", "gone", 400, 350, "2024-02-06T00:00:00Z")

	d, err := Extract(ctx, pool, nil, ExtractOptions{BatchSize: 2})
	is.NoErr(err)
	is.Equal(len(d.Games), 2) // casual, aborted, unfinished and unreadable games left out
	is.Equal(d.SkippedIDs, map[string][]int64{"players or scores missing": {8}})
	is.Equal(len(d.Penalties), 0)
	g := d.Games[0]
	is.Equal(g.ID, int64(1))
	is.Equal(d.Keys[g.Key], entity.VariantKey("NWL18.classic.regular"))
	is.Equal(g.Players, [2]int64{1, 2}) // from the game's own order, not player0_id
	is.Equal(g.Scores, [2]int32{400, 350})
	is.Equal(g.Reason, pb.GameEndReason_STANDARD)
	is.Equal(g.End, int64(1706745600))
	g = d.Games[1]
	is.Equal(d.Keys[g.Key], entity.VariantKey("CSW19.classic.blitz")) // camelCase request
	is.Equal(g.Players, [2]int64{2, 3})
	is.Equal(g.Winner, int8(1))
	is.Equal(d.Unfinished, []int64{5})
	is.Equal(d.MaxID, int64(8))
	is.True(d.Users[3].Bot)
	is.Equal(d.Resets, []Reset{{UserID: 2, Time: 1704067200}})
	is.Equal(d.Current[1]["NWL18.classic.regular"].Rating, 1600.0)
	is.Equal(len(d.Current[2]), 0)

	// Catching up: the corres game ends and a new game is played.
	_, err = pool.Exec(ctx, `UPDATE games SET game_end_reason = 2, winner_idx = 1,
		quickdata = jsonb_set(quickdata, '{s}', '[300, 410]'), updated_at = '2024-03-01T00:00:00Z' WHERE id = 5`)
	is.NoErr(err)
	insertGame(t, pool, 10, snakeRated, int(pb.GameEndReason_STANDARD), 1, "ua", "ub", 300, 350, "2024-03-02T00:00:00Z")
	insertGame(t, pool, 11, snakeRated, int(pb.GameEndReason_NONE), -1, "ua", "ub", 0, 0, "2024-03-03T00:00:00Z")
	// A league cancels alice's results in game 1, which she won 400-350,
	// after it was read. (The season itself doesn't matter here, so skip the
	// foreign key checks.)
	_, err = pool.Exec(ctx, `
		SET session_replication_role = replica;
		UPDATE games SET winner_idx = 1, timers = '{"lu": 1706745600000}', updated_at = '2024-03-04T00:00:00Z' WHERE id = 1;
		INSERT INTO game_players (game_uuid, player_id, player_index, score, won, game_end_reason, created_at,
			game_type, opponent_id, opponent_score, league_season_id)
		VALUES ('g1', 1, 0, 250, false, 2, now(), 0, 2, 350, gen_random_uuid()),
		       ('g1', 2, 1, 350, true, 2, now(), 0, 1, 250, gen_random_uuid());
		SET session_replication_role = DEFAULT`)
	is.NoErr(err)
	d2, err := Extract(ctx, pool, d, ExtractOptions{BatchSize: 100})
	is.NoErr(err)
	is.Equal(len(d2.Games), 4)
	is.Equal(d2.Games[2].ID, int64(5))
	is.Equal(d2.Keys[d2.Games[2].Key], entity.VariantKey("NWL18.classic.corres"))
	is.Equal(d2.Games[3].ID, int64(10))
	is.Equal(d2.Unfinished, []int64{11})
	is.Equal(d2.MaxID, int64(11))
	is.Equal(d2.SkippedIDs, map[string][]int64{"players or scores missing": {8}})
	is.Equal(d2.Penalties, map[int64]Penalty{1: {Loser: 0, Scores: [2]int32{250, 350}, End: 1706745600}})
}
