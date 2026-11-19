package gameplay_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores"
)

// responseWithID is a BotResponse as a bot that echoes request_id sends it.
func responseWithID(t *testing.T, id string) *macondopb.BotResponse {
	t.Helper()
	b := protowire.AppendTag(nil, 5, protowire.BytesType)
	b = protowire.AppendString(b, id)
	resp := &macondopb.BotResponse{GameId: "g"}
	resp.ProtoReflect().SetUnknown(protoreflect.RawFields(b))
	data, err := proto.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	got := &macondopb.BotResponse{}
	if err := proto.Unmarshal(data, got); err != nil {
		t.Fatal(err)
	}
	return got
}

func TestBotResponseRequestID(t *testing.T) {
	is := is.New(t)
	is.Equal(gameplay.BotResponseRequestID(responseWithID(t, "12")), "12")
	is.Equal(gameplay.BotResponseRequestID(&macondopb.BotResponse{GameId: "g"}), "")
}

func resend(t *testing.T, ctx context.Context, s *stores.Stores, staleAfter, bestBotStaleAfter time.Duration) []string {
	t.Helper()
	ids, err := gameplay.ResendStaleBotRequests(ctx, s, staleAfter, bestBotStaleAfter)
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

// makeJesseABot makes jesse, who is on turn first in setupNewGame's game, a
// bot.
func makeJesseABot(t *testing.T, gs *gamesetup) {
	t.Helper()
	_, err := gs.stores.DBPool.Exec(context.Background(),
		`UPDATE users SET internal_bot = true WHERE username = 'jesse'`)
	if err != nil {
		t.Fatal(err)
	}
}

func ageBotTurn(t *testing.T, gs *gamesetup, by time.Duration) {
	t.Helper()
	_, err := gs.stores.DBPool.Exec(context.Background(),
		`UPDATE ongoing_games SET updated_at = now() - make_interval(secs => $1) WHERE game_uuid = $2`,
		by.Seconds(), gs.g.GameID())
	if err != nil {
		t.Fatal(err)
	}
}

// botRequests are the bot move requests the game sent.
func botRequests(gs *gamesetup) []*macondopb.BotRequest {
	gs.cancel()
	<-gs.donechan
	var reqs []*macondopb.BotRequest
	for _, e := range gs.consumer.evts {
		if r, ok := e.Event.(*macondopb.BotRequest); ok {
			reqs = append(reqs, r)
		}
	}
	return reqs
}

func botRequestID(t *testing.T, req *macondopb.BotRequest) string {
	t.Helper()
	data, err := proto.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	resp := &macondopb.BotResponse{}
	// Field 5 is request_id in both messages, so the response parser reads it.
	if err := proto.Unmarshal(data, resp); err != nil {
		t.Fatal(err)
	}
	return gameplay.BotResponseRequestID(resp)
}

func TestWatchdogResendsStaleBotTurn(t *testing.T) {
	is := is.New(t)
	gs := setupNewGame()
	defer teardownGame(gs)
	makeJesseABot(t, gs)
	ctx := context.Background()

	// On turn for less than the threshold: left alone.
	is.Equal(len(resend(t, ctx, gs.stores, time.Minute, 5*time.Minute)), 0)

	ageBotTurn(t, gs, 3*time.Minute)
	is.Equal(resend(t, ctx, gs.stores, time.Minute, 5*time.Minute), []string{gs.g.GameID()})
	// Claimed: not resent again until another threshold has passed.
	is.Equal(len(resend(t, ctx, gs.stores, time.Minute, 5*time.Minute)), 0)

	reqs := botRequests(gs)
	is.Equal(len(reqs), 1)
	g := gs.reload(t)
	is.Equal(botRequestID(t, reqs[0]), strconv.Itoa(len(g.History().Events)))
}

// BestBot sims, so it gets longer before its request is resent.
func TestWatchdogWaitsLongerForBestBot(t *testing.T) {
	is := is.New(t)
	gs := setupNewGame()
	defer teardownGame(gs)
	makeJesseABot(t, gs)
	_, err := gs.stores.DBPool.Exec(context.Background(),
		`UPDATE users SET username = 'BestBot' WHERE username = 'jesse'`)
	is.NoErr(err)
	ctx := context.Background()

	ageBotTurn(t, gs, 3*time.Minute)
	is.Equal(len(resend(t, ctx, gs.stores, time.Minute, 5*time.Minute)), 0)

	ageBotTurn(t, gs, 6*time.Minute)
	is.Equal(resend(t, ctx, gs.stores, time.Minute, 5*time.Minute), []string{gs.g.GameID()})
	is.Equal(len(botRequests(gs)), 1)
}

func TestWatchdogIgnoresHumanTurn(t *testing.T) {
	is := is.New(t)
	gs := setupNewGame()
	defer teardownGame(gs)
	ageBotTurn(t, gs, 3*time.Minute)
	is.Equal(len(resend(t, context.Background(), gs.stores, time.Minute, 5*time.Minute)), 0)
	is.Equal(len(botRequests(gs)), 0)
}

func TestStaleBotResponse(t *testing.T) {
	is := is.New(t)
	gs := setupNewGame()
	defer teardownGame(gs)
	g := gs.g
	current := gameplay.BotRequestID(g)
	is.Equal(gameplay.StaleBotResponse(responseWithID(t, current), g, true), "")
	// A bot that doesn't echo an ID is trusted, as before.
	is.Equal(gameplay.StaleBotResponse(&macondopb.BotResponse{}, g, true), "")
	is.Equal(gameplay.StaleBotResponse(responseWithID(t, current+"0"), g, true), "position-changed")
	is.Equal(gameplay.StaleBotResponse(responseWithID(t, current), g, false), "bot-not-on-turn")
	gs.cancel()
	<-gs.donechan
}
