package sockets

import (
	"context"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	ipcTimeout = 10 * time.Second
)

func extendTopic(c *Client, topic string) string {
	// The publish topic should encode the user ID and the login status.
	// This is so we don't have to wastefully unmarshal and remarshal here,
	// and also because NATS plays nicely with hierarchical subject names.
	first := ""
	second := ""
	if !c.authenticated {
		first = "anon"
	} else {
		first = "auth"
	}
	second = c.userID

	return topic + "." + first + "." + second + "." + c.connID
}

func (h *Hub) parseAndExecuteMessage(ctx context.Context, msg []byte, c *Client) error {
	// All socket messages are encoded entity.Events.
	// (or they better be)

	// The type byte is [2] ([0] and [1] are length of the packet)

	topicName := "ipc.pb." + strconv.Itoa(int(msg[2]))
	fullTopic := extendTopic(c, topicName)
	log.Debug().Str("fullTopic", fullTopic).Msg("nats-publish")

	return h.pubsub.natsconn.Publish(fullTopic, msg[3:])
}
