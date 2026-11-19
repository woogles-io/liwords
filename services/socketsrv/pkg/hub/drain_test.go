package sockets

import (
	"fmt"
	"testing"
	"time"
)

func drainTestHub(sockets int) *Hub {
	h := &Hub{
		clients:         map[*Client][]Realm{},
		clientsByUserID: map[string]map[*Client]bool{},
		clientsByConnID: map[string]*Client{},
		realms:          map[Realm]map[*Client]bool{},
		// removeClient publishes leave messages; with no connection they
		// fail, which doesn't matter here.
		pubsub: &PubSub{},
	}
	for i := 0; i < sockets; i++ {
		c := &Client{userID: fmt.Sprintf("u%d", i), connID: fmt.Sprintf("c%d", i),
			send: make(chan []byte, 1)}
		h.clientsByUserID[c.userID] = map[*Client]bool{c: true}
		h.clientsByConnID[c.connID] = c
		h.addToRealm([]string{"lobby"}, c)
	}
	return h
}

// Sockets close evenly over the drain period, and all of them by its end.
func TestDrainPacing(t *testing.T) {
	h := drainTestHub(100)
	start := time.Unix(1000, 0)

	h.drainStep(start) // not draining yet
	if len(h.clients) != 100 {
		t.Fatalf("closed sockets before draining: %d left", len(h.clients))
	}

	h.startDrain(start, 10*time.Second)
	var left []int
	for s := 1; s <= 10; s++ {
		h.drainStep(start.Add(time.Duration(s) * time.Second))
		left = append(left, len(h.clients))
	}
	want := []int{90, 80, 70, 60, 50, 40, 30, 20, 10, 0}
	for i := range want {
		if left[i] != want[i] {
			t.Fatalf("sockets left after each second = %v, want %v", left, want)
		}
	}
	if !h.draining.drained {
		t.Fatal("not marked drained")
	}
}

// A late tick, or sockets that connect while draining, still finish by the
// end of the period.
func TestDrainCatchesUp(t *testing.T) {
	h := drainTestHub(100)
	start := time.Unix(1000, 0)
	h.startDrain(start, 10*time.Second)

	// 6 steps left, counting this one: 100/6, rounded up.
	h.drainStep(start.Add(5 * time.Second))
	if got := len(h.clients); got != 83 {
		t.Fatalf("after the first step 5s in, %d left, want 83", got)
	}
	h.drainStep(start.Add(30 * time.Second))
	if got := len(h.clients); got != 0 {
		t.Fatalf("after the period, %d left, want 0", got)
	}
}

// Only the first request starts a drain.
func TestDrainOnce(t *testing.T) {
	h := drainTestHub(10)
	start := time.Unix(1000, 0)
	h.startDrain(start, 10*time.Second)
	h.startDrain(start.Add(5*time.Second), time.Hour)
	if want := start.Add(10 * time.Second); !h.draining.until.Equal(want) {
		t.Fatalf("drain ends at %v, want %v", h.draining.until, want)
	}

	h.drainRequests = make(chan time.Duration, 1)
	h.Drain(time.Minute)
	h.Drain(time.Minute) // must not block
}
