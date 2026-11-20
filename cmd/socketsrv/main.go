package main

import (
	"context"
	"expvar"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/otelsetup"
	"github.com/woogles-io/liwords/services/socketsrv/pkg/config"
	sockets "github.com/woogles-io/liwords/services/socketsrv/pkg/hub"
)

const (
	GracefulShutdownTimeout = 30 * time.Second
)

func pingEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(`{"status":"copacetic"}`))
}

var (
	// BuildHash is the git hash, set by go build flags
	BuildHash = "unknown"
	// BuildDate is the build date, set by go build flags
	BuildDate = "unknown"
)

func main() {

	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Interface("config", cfg).
		Str("build-date", BuildDate).Str("build-hash", BuildHash).Msg("started")

	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Debug().Msg("debug log is on")

	// Tracing, when there's somewhere to send the spans. Without it the
	// tracer is a no-op and messages go out with no trace headers.
	shutdownTracing := func(context.Context) error { return nil }
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		if shutdown, err := otelsetup.Tracing(context.Background()); err != nil {
			log.Err(err).Msg("tracing-setup-failed")
		} else {
			shutdownTracing = shutdown
			log.Info().Msg("tracing-on")
		}
	}

	h, err := sockets.NewHub(cfg)
	if err != nil {
		panic(err)
	}
	go h.Run()

	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return fmt.Sprintf("%d", runtime.NumGoroutine())
	}))

	router := http.NewServeMux() // here you could also go with third party packages to create a router

	router.Handle("/ping", http.HandlerFunc(pingEndpoint))

	router.Handle("/ws", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sockets.ServeWS(h, w, r)
	}))

	router.Handle("/debug/vars", http.DefaultServeMux)

	srv := &http.Server{
		Addr:         cfg.WebsocketAddress,
		Handler:      router,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  90 * time.Second}

	idleConnsClosed := make(chan struct{})
	sig := make(chan os.Signal, 1)

	go func() {
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Info().Msg("got quit signal...")
		ctx, cancel := context.WithTimeout(context.Background(), GracefulShutdownTimeout)
		if err := srv.Shutdown(ctx); err != nil {
			// Error from closing listeners, or context timeout:
			log.Error().Msgf("HTTP server Shutdown: %v", err)
		}
		cancel()
		close(idleConnsClosed)
	}()

	log.Info().Msg("starting listening...")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("")
	}
	<-idleConnsClosed
	log.Info().Msg("server gracefully shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := shutdownTracing(ctx); err != nil {
		log.Err(err).Msg("tracing-shutdown")
	}
}
