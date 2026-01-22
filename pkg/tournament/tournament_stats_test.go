package tournament_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

func TestAddTourneyStat(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	stores, cfg := recreateDB()
	defer func() { cleanup(stores) }()

	directors := makeTournamentPersons(map[string]int32{"Kieran": 0, "Vince": 2, "Jennifer": 2})

	ty, err := makeTournament(ctx, stores.TournamentStore, cfg, directors)
	is.NoErr(err)
	tuuid := pgtype.Text{String: ty.UUID, Valid: true}

	err = stores.Queries.AddTourneyStat(ctx, models.AddTourneyStatParams{
		Uuid:         tuuid,
		DivisionName: "foo",
		PlayerID:     "bar",
		Stats:        []byte(`{}`),
	})
	is.NoErr(err)

	err = stores.Queries.AddTourneyStat(ctx, models.AddTourneyStatParams{
		Uuid:         tuuid,
		DivisionName: "foo",
		PlayerID:     "bar",
		Stats:        []byte(`{"foo": "bar"}`),
	})
	is.NoErr(err)

	stats, err := stores.Queries.GetTourneyStatsForPlayer(ctx, models.GetTourneyStatsForPlayerParams{
		Uuid:         tuuid,
		DivisionName: "foo",
		PlayerID:     "bar",
	})
	is.NoErr(err)
	is.Equal(stats, []byte(`{"foo": "bar"}`))
}
