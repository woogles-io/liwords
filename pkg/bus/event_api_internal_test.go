package bus

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"
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
