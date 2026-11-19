// Package natsopts holds the connection options the API and the socket
// server share for NATS.
package natsopts

import (
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// Options keeps a connection reconnecting for as long as the process runs.
//
// nats.go's defaults give up after 60 attempts 2s apart, about two minutes,
// and then close the connection for good: the process keeps serving but
// never hears from NATS again until it restarts. A NATS restart or upgrade
// that takes longer than that would leave every server deaf.
//
// Connect still fails at startup if NATS isn't reachable (no
// RetryOnFailedConnect), so a task with a bad NATS_URL crashes and fails its
// deploy instead of running unconnected.
//
// onReconnect, if given, run after each reconnect, after it is logged.
func Options(name string, onReconnect ...func()) []nats.Option {
	return []nats.Option{
		nats.Name(name),
		nats.MaxReconnects(-1),
		// Messages published while a connection is down are buffered and
		// flushed when it's back, but anyone who resubscribes after that
		// flush misses them. After a NATS restart, how much is lost depends
		// on how long clients wait before reconnecting: about a second's
		// worth of messages at 1s, a handful at 100ms (plus nats.go's default
		// 100ms jitter).
		nats.ReconnectWait(100 * time.Millisecond),
		// A server that disappears without closing the socket (an instance
		// stopped hard, a network fault) is noticed only by missed pings.
		// The default, 2 pings 2 minutes apart, takes up to 6 minutes;
		// this takes about a minute.
		nats.PingInterval(20 * time.Second),
		nats.MaxPingsOutstanding(3),
		// Also called, with a nil error, by Close at shutdown.
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.Warn().Err(err).Msg("nats-disconnected")
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Info().Str("server", nc.ConnectedUrlRedacted()).
				Uint64("reconnects", nc.Stats().Reconnects).Msg("nats-reconnected")
			for _, f := range onReconnect {
				f()
			}
		}),
		nats.DiscoveredServersHandler(func(nc *nats.Conn) {
			log.Info().Strs("servers", nc.DiscoveredServers()).Msg("nats-discovered-servers")
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Info().Err(nc.LastError()).Msg("nats-closed")
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			ev := log.Error().Err(err)
			if sub != nil {
				ev = ev.Str("subject", sub.Subject)
			}
			ev.Msg("nats-async-error")
		}),
	}
}
