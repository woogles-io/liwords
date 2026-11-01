package sockets

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/woogles-io/liwords/pkg/entity"
)

// Queued frames are batched into one websocket message, except a frame too big
// for a two-byte length, which goes in a message of its own so a client that
// can't read it loses nothing else.
func TestWritePumpSendsLargeFramesAlone(t *testing.T) {
	small := func(b byte) []byte { return []byte{0, 2, 24, b} }
	large := make([]byte, 3+entity.ExtendedFrameLength)
	large[2] = 24
	if !needsExtendedLength(large) || needsExtendedLength(small(1)) {
		t.Fatal("needsExtendedLength misclassifies the test frames")
	}
	queued := [][]byte{small(1), small(2), large, small(3), small(4), large, large}
	want := [][]byte{
		bytes.Join([][]byte{small(1), small(2)}, nil),
		large,
		bytes.Join([][]byte{small(3), small(4)}, nil),
		large,
		large,
	}

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		c := &Client{conn: conn, send: make(chan []byte, len(queued))}
		// Queued before the pump starts, so it finds them all waiting.
		for _, f := range queued {
			c.send <- f
		}
		go c.writePump()
	}))
	defer srv.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for i, w := range want {
		_, got, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("message %d: %v", i, err)
		}
		if !bytes.Equal(got, w) {
			t.Fatalf("message %d: got %d bytes, want %d", i, len(got), len(w))
		}
	}
}
