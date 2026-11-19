package sockets

import (
	"sync"
	"testing"
	"time"

	nats "github.com/nats-io/nats.go"
)

func testPubSub() *PubSub {
	ps := &PubSub{subchans: map[string]chan *nats.Msg{}}
	for _, topic := range []string{
		"lobby.>", "connid.>", "user.>", "usertv.>", "gametv.>", "game.>",
		"tournament.>", "league.>", "chat.>", "channel.>", "broadcasts.>",
		"presence.changed.>", "seek.followed.>", "sessionend.>",
	} {
		ps.subchans[topic] = make(chan *nats.Msg, subChanSize)
	}
	return ps
}

// A follower lookup that hangs (the API is busy) must not hold up messages
// for a connection, which used to be dropped behind them.
func TestFollowerLookupsDontHoldUpForwarding(t *testing.T) {
	h := &Hub{pubsub: testPubSub(), sendConnMessage: make(chan ConnMessage)}
	release := make(chan struct{})
	defer close(release)
	started := make(chan string, 1)
	presence := func(userID string, data []byte) {
		started <- userID
		<-release
	}
	go h.PubsubProcess()
	go h.followerProcess(presence, presence)

	h.pubsub.subchans["presence.changed.>"] <- &nats.Msg{Subject: "presence.changed.u1", Data: []byte{0, 0, 0}}
	select {
	case got := <-started:
		if got != "u1" {
			t.Fatalf("presence handled for %q, want u1", got)
		}
	case <-time.After(time.Second):
		t.Fatal("presence change not handled")
	}

	h.pubsub.subchans["connid.>"] <- &nats.Msg{Subject: "connid.abc", Data: []byte{0, 0, 7}}
	select {
	case m := <-h.sendConnMessage:
		if m.connID != "abc" {
			t.Fatalf("forwarded to %q, want abc", m.connID)
		}
	case <-time.After(time.Second):
		t.Fatal("connid message held up behind a follower lookup")
	}
}

func TestShardedRunner(t *testing.T) {
	s := newShardedRunner(4, 16)

	// One key's functions run in order.
	var mu sync.Mutex
	var got []int
	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		s.run("u1", func() {
			mu.Lock()
			got = append(got, i)
			mu.Unlock()
			if i == 99 {
				close(done)
			}
		})
	}
	<-done
	for i, v := range got {
		if v != i {
			t.Fatalf("ran out of order: %v", got)
		}
	}

	// A blocked key holds up only its own shard.
	blocked := "u1"
	other := ""
	for i := 0; other == ""; i++ {
		if k := string(rune('a' + i)); s.shardFor(k) != s.shardFor(blocked) {
			other = k
		}
	}
	release := make(chan struct{})
	s.run(blocked, func() { <-release })
	ran := make(chan struct{})
	s.run(other, func() { close(ran) })
	select {
	case <-ran:
	case <-time.After(time.Second):
		t.Fatal("another shard waited for a blocked one")
	}
	close(release)
}
