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
	// Cache settings
	FollowersCacheSize int
	FollowersCacheTTL  time.Duration
	// TargetGroupARN is the load balancer target group this server is in.
	// When set, the server closes its sockets gradually, over DrainPeriod,
	// once the target group starts draining it.
	TargetGroupARN string
	DrainPeriod    time.Duration
}

// Load loads the configs from the given arguments
func (c *Config) Load(args []string) error {
	fs := flag.NewFlagSet("socketsrv", flag.ContinueOnError)

	fs.StringVar(&c.WebsocketAddress, "ws-address", ":8087", "WS server listens on this address")
	fs.BoolVar(&c.Debug, "debug", false, "debug logging on")
	fs.StringVar(&c.NatsURL, "nats-url", "nats://localhost:4222", "the NATS server URL")
	fs.StringVar(&c.SecretKey, "secret-key", "", "secret key must be a random unguessable string")

	// Cache configuration
	fs.IntVar(&c.FollowersCacheSize, "followers-cache-size", 1000, "Maximum number of follower lists to cache")
	fs.DurationVar(&c.FollowersCacheTTL, "followers-cache-ttl", 5*time.Minute, "TTL for cached follower lists")

	fs.StringVar(&c.TargetGroupARN, "target-group-arn", "", "the load balancer target group this server is in; enables gradual draining")
	fs.DurationVar(&c.DrainPeriod, "drain-period", 2*time.Minute, "how long to take closing every socket once draining; keep it under the target group's deregistration delay")

	err := fs.Parse(args)
	return err
}
