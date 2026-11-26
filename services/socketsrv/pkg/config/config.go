package config

import (
	"github.com/namsral/flag"

	"github.com/woogles-io/liwords/pkg/debugserver"
)

type Config struct {
	Debug            bool
	WebsocketAddress string
	DebugAddr        string
	NatsURL          string
	SecretKey        string
}

// Load loads the configs from the given arguments
func (c *Config) Load(args []string) error {
	fs := flag.NewFlagSet("socketsrv", flag.ContinueOnError)

	fs.StringVar(&c.WebsocketAddress, "ws-address", ":8087", "WS server listens on this address")
	fs.StringVar(&c.DebugAddr, "debug-addr", debugserver.DefaultAddr, "serve pprof and expvar on this address (empty: off)")
	fs.BoolVar(&c.Debug, "debug", false, "debug logging on")
	fs.StringVar(&c.NatsURL, "nats-url", "nats://localhost:4222", "the NATS server URL")
	fs.StringVar(&c.SecretKey, "secret-key", "", "secret key must be a random unguessable string")

	err := fs.Parse(args)
	return err
}
