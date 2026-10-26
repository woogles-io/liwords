package game

import (
	"context"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/user"
)

// A profile's game list names the league or the tournament each game was
// played in, from the same query.
func TestRecentGamesNameTheirEvent(t *testing.T) {
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
	const leagueID = "6d1f0b9e-0000-4000-8000-000000000001"
	exec(`INSERT INTO tournaments (uuid, name, slug, type)
		VALUES ('t1', 'Philly Mondays', '/club/philly', 'club')`)
	exec(`INSERT INTO leagues (uuid, name, slug, settings)
		VALUES ($1, 'Smoke League', 'smoke', '{}')`, leagueID)
	// Three finished games of cesar's, an hour apart, newest first: one in
	// the club, one in the league, one in the open.
	for i, g := range []struct {
		id, tournament string
		league         any
	}{
		{"gclub", "t1", nil},
		{"gleague", "", leagueID},
		{"gopen", "", nil},
	} {
		exec(`INSERT INTO games (uuid, player0_id, player1_id, tournament_id,
			league_id, game_end_reason, created_at, updated_at, quickdata,
			game_request)
			VALUES ($1, $2, $3, NULLIF($4, ''), $5, 2, now(),
			now() - make_interval(hours => $6),
			'{"pi":[{"nickname":"cesar"},{"nickname":"mina"}]}',
			'{"rules": {"boardLayoutName": "CrosswordGame",
			  "letterDistributionName": "english"}, "initialTimeSeconds": 60}')`,
			g.id, cesar.ID, mina.ID, g.tournament, g.league, i)
		exec(`INSERT INTO game_players (game_uuid, player_id, player_index,
			score, game_end_reason, created_at, updated_at, game_type,
			opponent_id, opponent_score)
			VALUES ($1, $2, 0, 400, 2, now(), now() - make_interval(hours => $3),
			0, $4, 300)`, g.id, cesar.ID, i, mina.ID)
	}

	resp, err := gstore.GetRecentGames(ctx, "cesar", 10, 0)
	is.NoErr(err)
	is.Equal(len(resp.GameInfo), 3)
	club, league, open := resp.GameInfo[0], resp.GameInfo[1], resp.GameInfo[2]
	is.Equal(club.GameId, "gclub")
	is.Equal(club.TournamentName, "Philly Mondays")
	is.Equal(club.TournamentSlug, "/club/philly")
	is.Equal(club.LeagueName, "")
	is.Equal(league.LeagueSlug, "smoke")
	is.Equal(league.LeagueName, "Smoke League")
	is.Equal(league.TournamentName, "")
	is.Equal(open.TournamentName, "")
	is.Equal(open.LeagueName, "")
}
