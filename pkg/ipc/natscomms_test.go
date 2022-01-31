package ipc

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/domino14/liwords/pkg/config"
	"github.com/lithammer/shortuuid"
	"github.com/matryer/is"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// test request error/retry.

func TestRequest(t *testing.T) {
	is := is.New(t)
	msgHandler := func(ctx context.Context, bus Publisher, topic string, data []byte, reply string) error {
		log.Info().Str("data", string(data)).Msg("received msg")
		bus.PublishToTopic(reply, []byte("response to "+string(data)))
		return nil
	}
	topics := []TopicListener{{"foosvc.>", "fooqueue"}}

	b, err := NewBus(&config.Config{NatsURL: NatsUrl}, topics, msgHandler,
		nats.DrainTimeout(2*time.Second))
	is.NoErr(err)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	log.Info().Msg("about to start gofuncs")
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.ProcessMessages(ctx)
		log.Info().Msg("exiting-test-gofunc1")
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		b2, err := NewBus(&config.Config{NatsURL: NatsUrl}, topics, msgHandler,
			nats.DrainTimeout(2*time.Second))
		is.NoErr(err)
		// Don't start loop for this guy.
		resp, err := b2.Request("foosvc.fooreq", []byte("what's up?"))
		is.NoErr(err)
		is.Equal(resp, []byte("response to what's up?"))
		cancel()
	}()
	wg.Wait()
}

func TestRequestRetry(t *testing.T) {
	is := is.New(t)
	i := 0
	msgHandler := func(ctx context.Context, bus Publisher, topic string, data []byte, reply string) error {
		log.Info().Str("data", string(data)).Msg("received msg")
		if i != 2 {
			// This msg handler returns an error the first two times.
			// Since this is supposed to handle a response to a NATS request,
			// it just looks like it timed out on the request side.
			i += 1
			log.Info().Int("i", i).Msg("returning an oops")
			return errors.New("oops!")
		}
		log.Info().Msg("returning an actual response")
		bus.PublishToTopic(reply, []byte("response to "+string(data)))
		return nil
	}
	chars := shortuuid.New()[2:8]

	topics := []TopicListener{{chars + "svc.>", chars + "queue"}}

	b, err := NewBus(&config.Config{NatsURL: NatsUrl}, topics, msgHandler,
		nats.DrainTimeout(1*time.Second))
	is.NoErr(err)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	log.Info().Msg("about to start gofuncs")
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.ProcessMessages(ctx)
		log.Info().Msg("exiting-test-gofunc1")
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Create a new bus only to use its Request function. Make sure
		// it doesn't actually subscribe to any topics.
		b2, err := NewBus(
			&config.Config{NatsURL: NatsUrl},
			[]TopicListener{},
			nil)
		is.NoErr(err)
		// Don't start loop for this guy..
		log.Info().Msg("about to make Request")
		resp, err := b2.Request(chars+"svc.fooreq", []byte("what's up?"),
			RequestTimeout(500*time.Millisecond))
		cancel()
		is.NoErr(err)
		is.Equal(resp, []byte("response to what's up?"))
	}()
	wg.Wait()
}

func TestRequestRetryFail(t *testing.T) {
	is := is.New(t)
	i := 0
	msgHandler := func(ctx context.Context, bus Publisher, topic string, data []byte, reply string) error {
		log.Info().Str("data", string(data)).Msg("received msg")
		if i != 3 {
			// return an error the first three times. By default we only retry
			// three times, so this retry should fail.
			i += 1
			log.Info().Int("i", i).Msg("returning an oops")
			return errors.New("oops!")
		}
		log.Info().Msg("returning an actual response")
		bus.PublishToTopic(reply, []byte("response to "+string(data)))
		return nil
	}
	chars := shortuuid.New()[2:8]

	topics := []TopicListener{{chars + "svc.>", chars + "queue"}}

	b, err := NewBus(&config.Config{NatsURL: NatsUrl}, topics, msgHandler,
		nats.DrainTimeout(1*time.Second))
	is.NoErr(err)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	log.Info().Msg("about to start gofuncs")
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.ProcessMessages(ctx)
		log.Info().Msg("exiting-test-gofunc1")
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Create a new bus only to use its Request function. Make sure
		// it doesn't actually subscribe to any topics.
		b2, err := NewBus(
			&config.Config{NatsURL: NatsUrl},
			[]TopicListener{},
			nil)
		is.NoErr(err)
		// Don't start loop for this guy..
		log.Info().Msg("about to make Request")
		resp, err := b2.Request(chars+"svc.fooreq", []byte("what's up?"),
			RequestTimeout(500*time.Millisecond))
		cancel()
		is.True(strings.HasPrefix(err.Error(), "All attempts fail:"))
		is.Equal(resp, nil)
	}()
	wg.Wait()
}
