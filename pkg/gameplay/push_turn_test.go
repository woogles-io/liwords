package gameplay_test

import (
	"encoding/json"
	"slices"
	"sync"
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/gameplay"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type pushCall struct {
	user    string
	payload []byte
}

// recordingPusher stands in for the web push sender.
type recordingPusher struct {
	mu    sync.Mutex
	calls []pushCall
}

func (r *recordingPusher) Notify(userUUID string, payload []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, pushCall{userUUID, payload})
}

func (r *recordingPusher) got() []pushCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.calls)
}

func TestCorrespondenceMovePushesTheTurn(t *testing.T) {
	is := is.New(t)
	gs := setupNewGame(withCorrespondence, WithInitialTimeSeconds(3*24*3600))
	defer teardownGame(gs)
	pusher := &recordingPusher{}
	gs.stores.PushSender = pusher
	ctx := ctxForTests()

	g := gs.reload(t)
	mover := g.Game.PlayerIDOnTurn()
	var next, moverName string
	for _, p := range g.History().Players {
		if p.UserId == mover {
			moverName = p.Nickname
		} else {
			next = p.UserId
		}
	}
	_, err := gameplay.HandleEvent(ctx, gs.stores, mover, &pb.ClientGameplayEvent{
		Type: pb.ClientGameplayEvent_PASS, GameId: g.GameID()})
	is.NoErr(err)

	calls := pusher.got()
	is.Equal(len(calls), 1)
	is.Equal(calls[0].user, next)
	is.Equal(turnPush(t, calls[0]), turnMsg{
		GameID:   g.GameID(),
		Title:    "Your turn vs " + moverName,
		Body:     "It's your turn to play",
		Deadline: deadlineOnTurn(t, gs),
	})
}

type turnMsg struct {
	GameID   string `json:"gameId"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Deadline int64  `json:"deadline"`
}

func turnPush(t *testing.T, c pushCall) turnMsg {
	var m turnMsg
	if err := json.Unmarshal(c.payload, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// deadlineOnTurn is the last moment the player now on turn can move.
func deadlineOnTurn(t *testing.T, gs *gamesetup) int64 {
	g := gs.reload(t)
	return g.Timers.Deadline(g.Game.PlayerOnTurn())
}

func TestRealTimeMoveDoesNotPush(t *testing.T) {
	is := is.New(t)
	gs := setupNewGame()
	defer teardownGame(gs)
	pusher := &recordingPusher{}
	gs.stores.PushSender = pusher
	ctx := ctxForTests()

	g := gs.reload(t)
	_, err := gameplay.HandleEvent(ctx, gs.stores, g.Game.PlayerIDOnTurn(), &pb.ClientGameplayEvent{
		Type: pb.ClientGameplayEvent_PASS, GameId: g.GameID()})
	is.NoErr(err)
	is.Equal(len(pusher.got()), 0)
}

func TestCorrespondenceStartPushesTheFirstTurn(t *testing.T) {
	is := is.New(t)
	_, stores, cfg := recreateDB()
	defer stores.Disconnect()
	pusher := &recordingPusher{}
	stores.PushSender = pusher
	g, _, cancel, _, _ := makeGame(cfg, stores, withCorrespondence, WithInitialTimeSeconds(3*24*3600))
	defer cancel()

	calls := pusher.got()
	is.Equal(len(calls), 1)
	is.Equal(calls[0].user, g.Game.PlayerIDOnTurn())
	m := turnPush(t, calls[0])
	is.Equal(m.GameID, g.GameID())
	loaded, err := stores.GameStore.Get(ctxForTests(), g.GameID())
	is.NoErr(err)
	is.Equal(m.Deadline, loaded.Timers.Deadline(loaded.Game.PlayerOnTurn()))
}

func TestRealTimeStartDoesNotPush(t *testing.T) {
	is := is.New(t)
	_, stores, cfg := recreateDB()
	defer stores.Disconnect()
	pusher := &recordingPusher{}
	stores.PushSender = pusher
	_, _, cancel, _, _ := makeGame(cfg, stores)
	defer cancel()
	is.Equal(len(pusher.got()), 0)
}

func TestCorrespondenceChallengeKeepingTheTurnPushesNothing(t *testing.T) {
	is := is.New(t)
	gs := setupNewGame(withCorrespondence, WithInitialTimeSeconds(3*24*3600))
	defer teardownGame(gs)
	pusher := &recordingPusher{}
	gs.stores.PushSender = pusher
	ctx := ctxForTests()

	g := gs.reload(t)
	g.SetChallengeRule(macondopb.ChallengeRule_DOUBLE)
	racks := []*tilemapping.Rack{
		tilemapping.RackFromString("AGLSYYZ", g.Alphabet()),
		tilemapping.RackFromString("AGLSYYZ", g.Alphabet()),
	}
	racks[g.Game.PlayerOnTurn()] = tilemapping.RackFromString("ABEJNOR", g.Alphabet())
	g.SetRacksForBoth(racks)
	is.NoErr(gs.stores.GameStore.Set(ctx, g))
	mover := g.Game.PlayerIDOnTurn()
	_, err := gameplay.HandleEvent(ctx, gs.stores, mover, &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         g.GameID(),
		PositionCoords: "8D",
		MachineLetters: []byte{2, 1, 14, 10, 15, 5, 18}, // BANJOER, a phony
	})
	is.NoErr(err)
	challenger := gs.reload(t).Game.PlayerIDOnTurn()
	is.True(challenger != mover)
	is.Equal(len(pusher.got()), 1)

	// The phony comes off and the challenger keeps the turn they were pushed.
	_, err = gameplay.HandleEvent(ctx, gs.stores, challenger, &pb.ClientGameplayEvent{
		Type: pb.ClientGameplayEvent_CHALLENGE_PLAY, GameId: g.GameID()})
	is.NoErr(err)
	is.Equal(gs.reload(t).Game.PlayerIDOnTurn(), challenger)
	is.Equal(len(pusher.got()), 1)
}
