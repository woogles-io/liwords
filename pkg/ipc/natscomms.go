// Package ipc implements the inter-process communication between the different
// services of this repo.
package ipc

import (
	"fmt"
	"time"

	"github.com/avast/retry-go"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

const (
	defaultIPCTimeout = 3 * time.Second
	MaxSizeLimit      = 10_000_000
)

// Patterns:
// Services can either send requests to other services, or they can
// send fire-and-forget messages.
// In most cases, receivers should use queue groups.
// For example, a game service should use a queue group to listen
// for game messages. We only want one game service at a time to receive
// these.
// One exception is the socket server. We want to publish any messages that
// are going back to users on sockets to ALL sockets (so the receivers
// in the socket service should not use queue groups).
// This is because users can connect to only one socket and we don't know
// which one they're connected to.

// PublishToUser publishes a message to a specific user. If a channel is passed
// in, it'll attach the channel to the topic. We can use this for example if
// a user is in multiple games, across multiple connections.
func (b *Bus) PublishToUser(userID string, data []byte, channel string) error {
	var fullChannel string
	if channel == "" {
		fullChannel = "user." + userID
	} else {
		fullChannel = "user." + userID + "." + channel
	}

	return b.PublishToTopic(fullChannel, data)
}

// PublishToConnectionID will publish a message to a specific connection ID.
// We can use this function to publish stuff to new connections, for example,
// or in any other situations where we want to send a user a message but it doesn't
// have to be across all their connections.
func (b *Bus) PublishToConnectionID(connID string, data []byte) error {
	return b.PublishToTopic("connid."+connID, data)
}

func (b *Bus) PublishToTopic(topic string, data []byte) error {
	if len(data) > MaxSizeLimit {
		return fmt.Errorf("cannot send data over %d bytes", MaxSizeLimit)
	}
	return b.natsconn.Publish(topic, data)
}

// Request makes a request on the given topic and blocks, waiting for a response.
// It will retry on error.
func (b *Bus) Request(subject string, data []byte, opts ...Option) ([]byte, error) {
	if len(data) > MaxSizeLimit {
		return nil, fmt.Errorf("cannot send data over %d bytes", MaxSizeLimit)
	}
	var resp *nats.Msg
	config := &Config{
		reqTimeout: defaultIPCTimeout,
	}
	for _, opt := range opts {
		opt(config)
	}

	err := retry.Do(
		func() error {
			var err error
			resp, err = b.natsconn.Request(subject, data, config.reqTimeout)
			if err != nil {
				log.Err(err).Str("subject", subject).Msg("nats-request-error")
				return err
			}
			return nil
		},
		retry.Attempts(3),
		retry.OnRetry(func(n uint, err error) {
			log.Err(err).Uint("try", n).Str("subject", subject).Msg("retrying-nats-req")
		}),
	)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

type Config struct {
	reqTimeout time.Duration
}

// default is 10
func RequestTimeout(t time.Duration) Option {
	return func(c *Config) {
		c.reqTimeout = t
	}
}

type Option func(*Config)
