package config

import (
	"time"
	
	"github.com/namsral/flag"
)

type Config struct {
	Debug            bool
	WebsocketAddress string
	NatsURL          string
	SecretKey        string
	// Feature flags
	NewPresenceSystem bool
	// Cache settings
	FollowersCacheSize int
	FollowersCacheTTL  time.Duration
}

// Load loads the configs from the given arguments
func (c *Config) Load(args []string) error {
	fs := flag.NewFlagSet("liwords-socket", flag.ContinueOnError)

	fs.StringVar(&c.WebsocketAddress, "ws-address", ":8087", "WS server listens on this address")
	fs.BoolVar(&c.Debug, "debug", false, "debug logging on")
	fs.StringVar(&c.NatsURL, "nats-url", "nats://localhost:4222", "the NATS server URL")
	fs.StringVar(&c.SecretKey, "secret-key", "", "secret key must be a random unguessable string")
	fs.BoolVar(&c.NewPresenceSystem, "new-presence-system", false, "Enable new efficient presence notification system")
	
	// Cache configuration
	fs.IntVar(&c.FollowersCacheSize, "followers-cache-size", 1000, "Maximum number of follower lists to cache")
	fs.DurationVar(&c.FollowersCacheTTL, "followers-cache-ttl", 5*time.Minute, "TTL for cached follower lists")

	err := fs.Parse(args)
	return err
}
