package game

import (
	"context"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/user"
)

// A rematch streak lists its finished games newest first, with each winner
// given as an index into the players sorted by username descending, whichever
// seat they sat in.
func TestRematchStreak(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer func() {
		ustore.(*user.DBStore).Disconnect()
		gstore.Disconnect()
	}()
	ctx := context.Background()
	cesar, err := ustore.Get(ctx, "cesar")
	is.NoErr(err)
	mina, err := ustore.Get(ctx, "mina")
	is.NoErr(err)

	exec := func(sql string, args ...any) {
		_, err := gstore.dbPool.Exec(ctx, sql, args...)
		is.NoErr(err)
	}
	// won is NULL for both players in a tie or an aborted game.
	addGame := func(id, request string, hoursAgo, endReason int, p0, p1 uint, p0Won, p1Won any) {
		for _, p := range []struct {
			player, opponent uint
			index            int
			won              any
		}{{p0, p1, 0, p0Won}, {p1, p0, 1, p1Won}} {
			exec(`INSERT INTO game_players (game_uuid, player_id, player_index,
				score, won, game_end_reason, created_at, updated_at, game_type,
				opponent_id, opponent_score, original_request_id)
				VALUES ($1, $2, $3, 400, $4, $5, now() - make_interval(hours => $6),
				now(), 0, $7, 300, $8)`,
				id, p.player, p.index, p.won, endReason, hoursAgo, p.opponent, request)
		}
	}
	addGame("gnewest", "req", 1, 2, cesar.ID, mina.ID, true, false)
	addGame("gaborted", "req", 2, 5, cesar.ID, mina.ID, nil, nil)
	addGame("gmiddle", "req", 3, 2, mina.ID, cesar.ID, true, false)
	addGame("gtie", "req", 4, 2, cesar.ID, mina.ID, nil, nil)
	addGame("gother", "other", 5, 2, cesar.ID, mina.ID, true, false)

	resp, err := gstore.GetRematchStreak(ctx, "req")
	is.NoErr(err)
	is.Equal(len(resp.PlayersInfo), 2)
	is.Equal(resp.PlayersInfo[0].Nickname, "mina")
	is.Equal(resp.PlayersInfo[0].Uuid, mina.UUID)
	is.Equal(resp.PlayersInfo[1].Nickname, "cesar")
	is.Equal(len(resp.Streak), 3)
	is.Equal(resp.Streak[0].GameId, "gnewest")
	is.Equal(resp.Streak[0].Winner, int32(1))
	is.Equal(resp.Streak[1].GameId, "gmiddle")
	is.Equal(resp.Streak[1].Winner, int32(0))
	is.Equal(resp.Streak[2].GameId, "gtie")
	is.Equal(resp.Streak[2].Winner, int32(-1))

	resp, err = gstore.GetRematchStreak(ctx, "none")
	is.NoErr(err)
	is.Equal(len(resp.Streak), 0)
	is.Equal(len(resp.PlayersInfo), 0)
}
