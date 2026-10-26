package tournament_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/tournament"
)

func TestEnteredTournamentQueries(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	stores, _ := recreateDB()
	defer stores.Disconnect()
	pool, err := common.OpenTestingDB(pkg)
	is.NoErr(err)
	defer pool.Close()
	tstore, us, q := stores.TournamentStore, stores.UserStore, stores.Queries

	directors := makeTournamentPersons(map[string]int32{"Kieran:Kieran": 0})
	newTournament := func(slug string, irl bool, players ...string) *entity.Tournament {
		ty, err := tournament.NewTournament(ctx, tstore, slug, "", directors,
			entity.TypeStandard, "", "/tournament/"+slug, nil, nil, 0, irl)
		is.NoErr(err)
		is.NoErr(tournament.AddDivision(ctx, tstore, ty.UUID, divOneName))
		persons := map[string]int32{}
		for _, p := range players {
			persons[p] = 1500
		}
		is.NoErr(tournament.AddPlayers(ctx, tstore, us, ty.UUID, divOneName, makeTournamentPersons(persons)))
		return ty
	}
	open := newTournament("entered-open", false, "Will", "Josh")
	done := newTournament("entered-done", true, "Will")
	_, err = pool.Exec(ctx, `UPDATE tournaments SET is_finished = true WHERE uuid = $1`, done.UUID)
	is.NoErr(err)

	will, err := us.Get(ctx, "Will")
	is.NoErr(err)
	josh, err := us.Get(ctx, "Josh")
	is.NoErr(err)
	// Will and Josh played twice in the finished one, the second an hour
	// later; a third game never started (CANCELLED, 7).
	for i, reason := range []int{2, 2, 7} {
		_, err = pool.Exec(ctx, `INSERT INTO games
			(uuid, player0_id, player1_id, tournament_id, game_end_reason, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, now(), now() + make_interval(hours => $6))`,
			fmt.Sprintf("entered%d", i), will.ID, josh.ID, done.UUID, reason, i)
		is.NoErr(err)
	}

	for _, u := range []*entity.User{will, josh} {
		played, err := q.ListTournamentsPlayedByUser(ctx, models.ListTournamentsPlayedByUserParams{
			UserID: pgtype.Int4{Int32: int32(u.ID), Valid: true},
			Lim:    10,
		})
		is.NoErr(err)
		is.Equal(len(played), 1)
		is.Equal(played[0].TournamentID, done.UUID)
		is.Equal(played[0].Games, int32(2))
		is.True(played[0].LastPlayed.Valid)
	}

	// Registered: only in the one not finished.
	for _, u := range []*entity.User{will, josh} {
		registered, err := q.ListUnfinishedTournamentsWithPlayer(ctx, models.ListUnfinishedTournamentsWithPlayerParams{
			UserUuid: u.UUID,
			Lim:      10,
		})
		is.NoErr(err)
		is.Equal(len(registered), 1)
		is.Equal(registered[0].String, open.UUID)
	}
	kieran, err := us.Get(ctx, "Kieran")
	is.NoErr(err)
	registered, err := q.ListUnfinishedTournamentsWithPlayer(ctx, models.ListUnfinishedTournamentsWithPlayerParams{
		UserUuid: kieran.UUID,
		Lim:      10,
	})
	is.NoErr(err)
	is.Equal(len(registered), 0)

	listings, err := q.GetTournamentListings(ctx, []string{open.UUID, done.UUID})
	is.NoErr(err)
	is.Equal(len(listings), 2)
	for _, l := range listings {
		switch l.Uuid.String {
		case open.UUID:
			is.Equal(l.Slug.String, "/tournament/entered-open")
			is.Equal(l.Type.String, "tournament")
			is.True(!l.IrlMode)
		case done.UUID:
			is.Equal(l.Name.String, "entered-done")
			is.True(l.IrlMode)
		default:
			t.Fatalf("unexpected listing %v", l.Uuid.String)
		}
	}
}
