// Package sockets encapsulates all of the Websocket communication.
// It's part of the framework/drivers layer.
package sockets

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/domino14/liwords/pkg/entity"

	pb "github.com/domino14/liwords/rpc/api/proto"
	"github.com/rs/zerolog/log"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		log.Debug().Msgf("host header %v", r.Header.Get("Host"))
		log.Debug().Msgf("origin %v", r.Header.Get("Origin"))
		// XXX: FIX THIS
		return true
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	username string

	realms map[Realm]bool
}

func (c *Client) sendError(err error) {
	evt := entity.WrapEvent(&pb.ErrorMessage{Message: err.Error()}, pb.MessageType_ERROR_MESSAGE, "")
	bts, err := evt.Serialize()
	if err != nil {
		// This really shouldn't happen.
		log.Err(err).Msg("error serializing error, lol")
		return
	}
	c.send <- bts
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Err(err).Msg("unexpected-close")
			}
			break
		}

		// Here is where we parse the message and send something off to the hub
		// potentially.
		err = c.hub.parseAndExecuteMessage(context.Background(), message, c.username)
		if err != nil {
			log.Err(err).Str("username", c.username).Msg("parse-and-execute-message")
			c.sendError(err)
			continue
		}

		// message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		// c.hub.broadcast <- message
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				log.Info().Msg("hub closed channel")
				return
			}

			w, err := c.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message.
			// XXX: FIX ME, THESE NEED TO BE SENT AS SEPARATE MESSAGES.
			// commenting out for now. There's no delimiter handling
			// on the client side.
			// n := len(c.send)
			// for i := 0; i < n; i++ {
			// 	w.Write(<-c.send)
			// }

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// close connection with an error string.
func closeMessage(ws *websocket.Conn, errStr string) {
	// close code 1008 is used for a generic "policy violation" message.
	msg := websocket.FormatCloseMessage(websocket.ClosePolicyViolation, errStr)
	log.Debug().Str("closemsg", string(msg)).Msg("writing close message")
	err := ws.WriteMessage(websocket.CloseMessage, msg)
	if err != nil {
		log.Err(err).Msg("writing close message back to user")
	}
	ws.Close()
	return
}

func validateWsRequest(v url.Values, now int64) error {
	secretKey := os.Getenv("SECRET_KEY")
	user := v.Get("user")
	timestamp := v.Get("expire")
	token := v.Get("_token")
	// Convert token to an array of bytes. Assume token is hex encoded.

	if secretKey == "" {
		// This should be a panic but let's not go overboard.
		// Maybe it's a too many open files issue.
		return errors.New("no secret key in environment")
	}
	// Convert timestamp to an int.
	tsInt, err := strconv.Atoi(timestamp)
	if err != nil {
		return err
	}
	if int64(tsInt) < now {
		return fmt.Errorf("your token has expired (ts = %v, now = %v)",
			tsInt, now)
	}
	if user == "" {
		return fmt.Errorf("no user was specified")
	}
	tokenHex, err := hex.DecodeString(token)
	if err != nil {
		return err
	}

	// Reconstruct signed string.
	ss := fmt.Sprintf("expire=%v&user=%v", timestamp, user)
	log.Debug().Str("tosign", ss).Msg("signing")
	mac := hmac.New(sha1.New, []byte(secretKey))
	mac.Write([]byte(ss))
	expectedMac := mac.Sum(nil)
	if !hmac.Equal(expectedMac, tokenHex) {
		return fmt.Errorf(
			"token signature was not correct (got %x, expected %x)",
			expectedMac, tokenHex)
	}
	return nil
}

// serveWs handles websocket requests from the peer.
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Err(err).Msg("upgrading socket")
		return
	}

	qvals := r.URL.Query()
	// XXX: figure out validation later
	// err = validateWsRequest(qvals, time.Now().Unix())
	// if err != nil {
	// 	log.Err(err).Msg("validating websocket")
	// 	return
	// }
	// initialRealm := qvals.Get("realm")

	client := &Client{
		hub:      hub,
		conn:     conn,
		username: qvals.Get("user"),
		send:     make(chan []byte, 256),
		realms:   make(map[Realm]bool),
	}
	// log.Info().Str("user", client.username).Str("initRealm", initialRealm).
	// 	Msg("new connection")
	// client.realms[Realm(initialRealm)] = true
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
	// client.sendInitialData()
}
