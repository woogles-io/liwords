package ipc

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/config"
)

// This file should handle utility functions for setting up a bus
// to subscribe to NATS channels and respond. It is meant to be modular
// so the different services can use it.

// TopicListener is a struct that includes a NATS topic, and a name
// of a queue. The queue name should be unique per NATS topic across
// the entire liwords. If the queue name is left blank, it will do a
// regular subscribe instead of a queue subscribe.
type TopicListener struct {
	Topic     string
	QueueName string
}

const DrainTimeout = 10 * time.Second

// A Publisher sends messages to topics.
type Publisher interface {
	Request(subject string, data []byte, opts ...Option) ([]byte, error)
	PublishToTopic(topic string, data []byte) error
	PublishToUser(user string, data []byte, optionalChannel string) error
}

// MsgHandler defines a function signature for handling NATS messages.
// Note that the `reply` parameter can be an empty string, in which case
// the handler should not publish to this channel.
type MsgHandler func(ctx context.Context, bus Publisher, topic string, data []byte, reply string) error

type Bus struct {
	natsconn *nats.Conn
	config   *config.Config

	subscriptions []*nats.Subscription
	subchans      map[string]chan *nats.Msg

	msgHandler MsgHandler
}

func NewBus(cfg *config.Config, topics []TopicListener, msgHandler MsgHandler,
	opts ...nats.Option) (*Bus, error) {
	natsconn, err := nats.Connect(cfg.NatsURL, opts...)

	if err != nil {
		return nil, err
	}
	bus := &Bus{
		natsconn:      natsconn,
		subscriptions: []*nats.Subscription{},
		subchans:      map[string]chan *nats.Msg{},
		config:        cfg,
	}

	for _, t := range topics {
		ch := make(chan *nats.Msg, 64)
		var err error
		var sub *nats.Subscription
		if t.QueueName != "" {
			sub, err = natsconn.ChanQueueSubscribe(t.Topic, t.QueueName, ch)
		} else {
			sub, err = natsconn.ChanSubscribe(t.Topic, ch)
		}
		if err != nil {
			return nil, err
		}
		log.Info().Str("topic", t.Topic).Str("queueName", t.QueueName).Msg("subscribed-to-topic")

		bus.subscriptions = append(bus.subscriptions, sub)
		bus.subchans[t.Topic] = ch
	}
	bus.msgHandler = msgHandler
	return bus, nil
}

func (b *Bus) ProcessMessages(ctx context.Context) {
	ctx = context.WithValue(ctx, config.CtxKeyword, b.config)
	ctx = log.Logger.WithContext(ctx)
	log := zerolog.Ctx(ctx)

	agg := make(chan *nats.Msg)
	for topic, ch := range b.subchans {
		go func(c chan *nats.Msg, topic string) {
			log.Info().Str("topic", topic).Msg("start-chan-aggregation")
			for msg := range c {
				agg <- msg
			}
			log.Info().Str("topic", topic).Msg("stop-chan-aggregation")
		}(ch, topic)
	}

	exit := make(chan struct{})
	done := ctx.Done()
	log.Info().Msg("starting process loop")
outerfor:
	for {
		select {
		case msg := <-agg:

			log := log.With().Interface("msg-subject", msg.Subject).Logger()
			log.Debug().Msg("got-msg")

			go func(topic string, data []byte, reply string) {
				err := b.msgHandler(log.WithContext(ctx), b, topic, data, reply)
				if err != nil {
					log.Err(err).Msg("msg-handler-error")
					// TODO: publish error back to user.
				}
			}(msg.Subject, msg.Data, msg.Reply)

		case <-done:
			log.Info().Msg("pubsub context done, draining subscriptions")
			for _, sub := range b.subscriptions {
				if err := sub.Drain(); err != nil {
					log.Err(err).Str("sub", sub.Subject).Msg("error-draining")
				}
			}
			time.AfterFunc(b.natsconn.Opts.DrainTimeout, func() {
				log.Info().Msg("Exiting loop after timeout...")
				exit <- struct{}{}
			})
			// Take done out of the select so that it stops triggering once
			// we close it.
			done = nil

		case <-exit:
			break outerfor
		}

	}

	log.Info().Msg("exiting processMessages loop")

}

// Function should be called with a waitgroup to ensure it has time to end.
// for example
// in caller
// wg := sync.WaitGroup{}
// go func() {
//		defer wg.Done()
//		ProcessMessages(ctx)
// }()
// wg.Wait()
