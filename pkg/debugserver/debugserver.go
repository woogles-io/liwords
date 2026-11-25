// Package debugserver serves pprof and expvar on their own listener, apart
// from the public HTTP server, so a load balancer rule can never expose them.
package debugserver

import (
	"expvar"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/rs/zerolog/log"
)

// DefaultAddr only accepts connections from inside the container. In
// production, reach it with `aws ecs execute-command` or an SSM port forward.
const DefaultAddr = "127.0.0.1:6060"

// Handler serves /debug/pprof/ and /debug/vars.
func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index) // also heap, goroutine, allocs...
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/vars", expvar.Handler())
	return mux
}

// Start serves Handler on addr in the background. An empty addr turns it off.
func Start(addr string) {
	if addr == "" {
		return
	}
	srv := &http.Server{Addr: addr, Handler: Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		log.Info().Str("addr", addr).Msg("debug-server-listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Err(err).Str("addr", addr).Msg("debug-server-stopped")
		}
	}()
}
