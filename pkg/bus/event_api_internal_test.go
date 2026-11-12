package bus

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Events reach a subscriber in the order the bus loop handed them over.
func TestEventAPIKeepsOrder(t *testing.T) {
	is := is.New(t)
	s := NewEventApiServer(nil, nil)
	sub, unsubscribe := s.subscribe("r1", []string{"game.g1"})
	defer unsubscribe()

	for i := range eventSubBuffer {
		is.NoErr(s.processEvent("game.g1", []byte(fmt.Sprint(i))))
	}
	for i := range eventSubBuffer {
		is.Equal(string(<-sub.ch), fmt.Sprint(i))
	}
}

// A client that falls behind is cut off rather than holding up the bus
// loop, and one that has left gets nothing.
func TestEventAPIDoesNotWaitForClients(t *testing.T) {
	is := is.New(t)
	s := NewEventApiServer(nil, nil)
	slow, unsubscribeSlow := s.subscribe("slow", []string{"game.g1"})
	defer unsubscribeSlow()
	_, unsubscribeGone := s.subscribe("gone", []string{"game.g1"})
	unsubscribeGone()

	done := make(chan struct{})
	go func() {
		for range eventSubBuffer + 10 {
			s.processEvent("game.g1", []byte("e"))
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("processEvent waited for a client")
	}
	select {
	case <-slow.tooSlow:
	default:
		is.Fail() // the slow client should have been cut off
	}
}

// Clients connecting and leaving while events flow: go test -race fails on
// unsynchronized access to the subscription maps.
func TestEventAPIConcurrentSubscriptions(t *testing.T) {
	s := NewEventApiServer(nil, nil)
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				s.processEvent("game.g1", []byte("e"))
			}
		}
	}()
	for i := range 200 {
		sub, unsubscribe := s.subscribe(requestID(fmt.Sprint(i)), []string{"game.g1", "user.u.game.g1"})
		select {
		case <-sub.ch:
		case <-time.After(10 * time.Millisecond):
		}
		unsubscribe()
	}
	close(stop)
	wg.Wait()
}

type apiKeyUsers struct{ user.Store }

func (apiKeyUsers) GetByAPIKey(ctx context.Context, apiKey string) (*entity.User, error) {
	return &entity.User{UUID: "u1"}, nil
}

type historyGames struct{ gameplay.GameStore }

func (historyGames) GetHistory(ctx context.Context, id string) (*macondopb.GameHistory, error) {
	return &macondopb.GameHistory{Uid: id}, nil
}

// A stream outlives the server's WriteTimeout: events written after it
// still reach the client.
func TestEventStreamOutlivesWriteTimeout(t *testing.T) {
	is := is.New(t)
	s := NewEventApiServer(apiKeyUsers{}, historyGames{})
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.ServeHTTP(w, r.WithContext(apiserver.StoreAPIKeyInContext(r.Context(), "key")))
	}))
	ts.Config.WriteTimeout = 200 * time.Millisecond
	ts.Start()
	defer ts.Close()

	resp, err := http.Get(ts.URL + GameEventStreamPrefix + "g1")
	is.NoErr(err)
	defer resp.Body.Close()
	lines := bufio.NewScanner(resp.Body)
	is.True(lines.Scan()) // the game's history

	time.Sleep(400 * time.Millisecond) // past the WriteTimeout
	evt := entity.WrapEvent(&pb.ServerGameplayEvent{GameId: "late"}, pb.MessageType_SERVER_GAMEPLAY_EVENT)
	data, err := evt.Serialize()
	is.NoErr(err)
	is.NoErr(s.processEvent("game.g1", data))

	got := make(chan string, 1)
	go func() {
		if lines.Scan() {
			got <- lines.Text()
		}
		close(got)
	}()
	select {
	case line, ok := <-got:
		is.True(ok) // the stream is still open
		is.True(strings.Contains(line, "late"))
	case <-time.After(2 * time.Second):
		t.Fatal("no event after the write timeout")
	}
}
