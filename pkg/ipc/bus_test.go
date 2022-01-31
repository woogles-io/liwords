package ipc

import (
	"context"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/domino14/liwords/pkg/config"
	"github.com/matryer/is"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var NatsUrl = os.Getenv("NATS_URL")

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	os.Exit(m.Run())
}

func TestBusLoop(t *testing.T) {
	is := is.New(t)
	topics := []TopicListener{{"foosvc.>", "fooqueue"},
		{"barsvc.>", "barqueue"}}
	receivedMessages := []string{}
	var mu sync.Mutex
	msgHandler := func(ctx context.Context, bus Publisher, topic string, data []byte, reply string) error {
		mu.Lock()
		defer mu.Unlock()
		receivedMessages = append(receivedMessages, string(data))
		log.Info().Str("data", string(data)).Msg("received msg")
		return nil
	}

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
		natsconn, err := nats.Connect(NatsUrl)
		is.NoErr(err)
		natsconn.Publish("bazsvc.huh", []byte("This message won't be received"))
		natsconn.Publish("foosvc.something", []byte("Hello"))
		natsconn.Publish("barsvc.somethingelse", []byte("Goodbye"))
		// Wait a little bit of time before canceling. Otherwise, the
		// subscriptions will immediately begin draining and no messages
		// will be received.
		time.AfterFunc(100*time.Millisecond, func() {
			log.Info().Msg("going to cancel")
			cancel()
		})
		log.Info().Msg("exiting-test-gofunc2")
	}()

	// Wait till both goroutines die.
	wg.Wait()
	sort.Strings(receivedMessages)
	is.Equal(receivedMessages, []string{"Goodbye", "Hello"})

}

func TestBusDrain(t *testing.T) {
	is := is.New(t)
	topics := []TopicListener{{"foosvc.>", "fooqueue"},
		{"barsvc.>", "barqueue"}}
	receivedMessages := []string{}
	var mu sync.Mutex
	msgHandler := func(ctx context.Context, bus Publisher, topic string, data []byte, reply string) error {
		mu.Lock()
		defer mu.Unlock()
		// This is a slow message handler.
		time.Sleep(250 * time.Millisecond)
		receivedMessages = append(receivedMessages, string(data))
		log.Info().Str("data", string(data)).Msg("received msg")
		return nil
	}

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
		natsconn, err := nats.Connect(NatsUrl)
		is.NoErr(err)
		natsconn.Publish("bazsvc.huh", []byte("This message won't be received"))
		natsconn.Publish("foosvc.something", []byte("Hello"))
		natsconn.Publish("barsvc.somethingelse", []byte("Goodbye"))
		natsconn.Publish("foosvc.something", []byte("How do"))
		natsconn.Publish("barsvc.somethingelse", []byte("you do"))
		time.AfterFunc(10*time.Millisecond, func() {
			log.Info().Msg("going to cancel")
			cancel()
		})
		log.Info().Msg("exiting-test-gofunc2")
	}()

	// Wait till both goroutines die.
	wg.Wait()
	sort.Strings(receivedMessages)

	is.Equal(receivedMessages, []string{
		"Goodbye",
		"Hello",
		"How do",
		"you do",
	})
}

func TestQueueSubscriptions(t *testing.T) {
	is := is.New(t)
	topics := []TopicListener{{"foosvc.>", "fooqueue"},
		{"barsvc.>", "barqueue"}}
	receivedMessages := []string{}

	var mu sync.Mutex
	msgHandler := func(ctx context.Context, bus Publisher, topic string, data []byte, reply string) error {
		mu.Lock()
		defer mu.Unlock()
		// This is a slow message handler.
		time.Sleep(100 * time.Millisecond)
		receivedMessages = append(receivedMessages, string(data))
		log.Info().Str("data", string(data)).Msg("received msg")
		return nil
	}

	var cancel1, cancel2 context.CancelFunc
	var ctx1, ctx2 context.Context
	wg := sync.WaitGroup{}

	// Create two buses, which will create subscriptions
	for i := 0; i < 2; i++ {
		if i == 0 {
			ctx1, cancel1 = context.WithCancel(context.Background())
		} else {
			ctx2, cancel2 = context.WithCancel(context.Background())
		}
		b, err := NewBus(&config.Config{NatsURL: NatsUrl}, topics, msgHandler,
			nats.DrainTimeout(2*time.Second))
		is.NoErr(err)

		log.Info().Int("bus", i).Msg("about to start gofuncs")
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i == 0 {
				b.ProcessMessages(ctx1)
			} else {
				b.ProcessMessages(ctx2)
			}
			log.Info().Int("bus", i).Msg("exiting-test-gofunc1")
		}(i)
	}
	wg.Add(1)
	// Publish a bunch of messages.
	go func() {
		defer wg.Done()
		natsconn, err := nats.Connect(NatsUrl)
		is.NoErr(err)
		natsconn.Publish("bazsvc.huh", []byte("This message won't be received"))
		natsconn.Publish("foosvc.something", []byte("Hello"))
		natsconn.Publish("barsvc.somethingelse", []byte("Goodbye"))
		natsconn.Publish("foosvc.something", []byte("How do"))
		natsconn.Publish("barsvc.somethingelse", []byte("you do"))
		natsconn.Publish("foosvc.something", []byte("Cookies"))
		natsconn.Publish("barsvc.somethingelse", []byte("Ice cream"))
		natsconn.Publish("foosvc.something", []byte("How do"))
		natsconn.Publish("barsvc.somethingelse", []byte("you do"))
		time.AfterFunc(10*time.Millisecond, func() {
			log.Info().Msg("going to cancel both buses")
			cancel1()
			cancel2()
		})
		log.Info().Msg("exiting-test-gofunc2")
	}()

	wg.Wait()
	sort.Strings(receivedMessages)

	// There should be no extra messages others than those we specifically
	// duplicated, since this is a QueueSubscribe.
	is.Equal(receivedMessages, []string{
		"Cookies",
		"Goodbye",
		"Hello",
		"How do",
		"How do",
		"Ice cream",
		"you do",
		"you do",
	})
}

func TestPubsubSubscriptions(t *testing.T) {
	is := is.New(t)
	topics := []TopicListener{{"foosvc.>", ""},
		{"barsvc.>", ""}}
	receivedMessages := []string{}

	var mu sync.Mutex
	msgHandler := func(ctx context.Context, bus Publisher, topic string, data []byte, reply string) error {
		mu.Lock()
		defer mu.Unlock()
		// This is a slow message handler.
		time.Sleep(100 * time.Millisecond)
		receivedMessages = append(receivedMessages, string(data))
		log.Info().Str("data", string(data)).Msg("received msg")
		return nil
	}

	var cancel1, cancel2 context.CancelFunc
	var ctx1, ctx2 context.Context
	wg := sync.WaitGroup{}

	// Create two buses, which will create subscriptions
	for i := 0; i < 2; i++ {
		if i == 0 {
			ctx1, cancel1 = context.WithCancel(context.Background())
		} else {
			ctx2, cancel2 = context.WithCancel(context.Background())
		}
		b, err := NewBus(&config.Config{NatsURL: NatsUrl}, topics, msgHandler,
			nats.DrainTimeout(2*time.Second))
		is.NoErr(err)

		log.Info().Int("bus", i).Msg("about to start gofuncs")
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i == 0 {
				b.ProcessMessages(ctx1)
			} else {
				b.ProcessMessages(ctx2)
			}
			log.Info().Int("bus", i).Msg("exiting-test-gofunc1")
		}(i)
	}
	wg.Add(1)
	// Publish a bunch of messages.
	go func() {
		defer wg.Done()
		natsconn, err := nats.Connect(NatsUrl)
		is.NoErr(err)
		natsconn.Publish("bazsvc.huh", []byte("This message won't be received"))
		natsconn.Publish("foosvc.something", []byte("Hello"))
		natsconn.Publish("barsvc.somethingelse", []byte("Goodbye"))
		natsconn.Publish("foosvc.something", []byte("How do"))
		natsconn.Publish("barsvc.somethingelse", []byte("you do"))
		natsconn.Publish("foosvc.something", []byte("Cookies"))
		natsconn.Publish("barsvc.somethingelse", []byte("Ice cream"))
		time.AfterFunc(10*time.Millisecond, func() {
			log.Info().Msg("going to cancel both buses")
			cancel1()
			cancel2()
		})
		log.Info().Msg("exiting-test-gofunc2")
	}()

	wg.Wait()
	sort.Strings(receivedMessages)

	// All messages are duplicated since this is not a queue subscribe.
	is.Equal(receivedMessages, []string{
		"Cookies",
		"Cookies",
		"Goodbye",
		"Goodbye",
		"Hello",
		"Hello",
		"How do",
		"How do",
		"Ice cream",
		"Ice cream",
		"you do",
		"you do",
	})
}
