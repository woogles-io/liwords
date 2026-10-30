package sockets

import (
	"errors"
	"testing"
)

// The hub closes a client's send channel when it drops the client, while the
// client's read pump may still be running. What the read pump sends (the lag
// after a pong, an error) must go through the hub, which no longer has the
// client, and not to the closed channel, where it would panic.
func TestReadPumpSendsGoThroughHub(t *testing.T) {
	h := &Hub{sendConnMessage: make(chan ConnMessage, 2)}
	c := &Client{hub: h, connID: "conn", send: make(chan []byte, 256)}
	close(c.send) // as removeClient does

	c.sendLatency()
	c.sendError(errors.New("boom"))

	for i := 0; i < 2; i++ {
		m := <-h.sendConnMessage
		if m.connID != "conn" {
			t.Fatalf("message %d went to %q", i, m.connID)
		}
	}
}
