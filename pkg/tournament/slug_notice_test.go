package tournament_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/tournament"
	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

// A director's change of a tournament's slug is posted to the webhook that
// hears of new tournaments, and a change that keeps the slug is not.
func TestSlugChangeIsAnnounced(t *testing.T) {
	is := is.New(t)
	stores, cfg := recreateDB()
	defer stores.Disconnect()
	tstore, us := stores.TournamentStore, stores.UserStore

	var mu sync.Mutex
	posted := []string{}
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding a posted notice: %v", err)
		}
		mu.Lock()
		posted = append(posted, body.Content)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer hook.Close()
	postedSoFar := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string{}, posted...)
	}

	withHook := *cfg
	withHook.DiscordToken = hook.URL
	ctx := withHook.WithContext(context.Background())
	ctx = apiserver.PlaceInContext(ctx, &entity.Session{Username: "Kieran", UserUUID: "Kieran"})

	ty, err := tournament.NewTournament(ctx, tstore, tournamentName, "desc",
		makeTournamentPersons(map[string]int32{"Kieran:Kieran": 0}),
		entity.TypeStandard, "", "/tournament/old-slug", nil, nil, 0, false)
	is.NoErr(err)
	svc := tournament.NewTournamentService(tstore, us, &withHook, nil, stores.Queries)

	set := func(meta *pb.TournamentMetadata) {
		_, err := svc.SetTournamentMetadata(ctx, connect.NewRequest(&pb.SetTournamentMetadataRequest{
			Metadata: meta, SetOnlySpecified: true}))
		is.NoErr(err)
	}

	set(&pb.TournamentMetadata{Id: ty.UUID, Description: "a new description"})
	is.Equal(postedSoFar(), []string{})

	set(&pb.TournamentMetadata{Id: ty.UUID, Slug: "/tournament/new-slug"})
	is.Equal(postedSoFar(), []string{
		"A tournament's slug has been changed by Kieran: testTournament (from /tournament/old-slug to /tournament/new-slug)"})

	set(&pb.TournamentMetadata{Id: ty.UUID, Slug: "/tournament/new-slug"})
	is.Equal(len(postedSoFar()), 1)
}
