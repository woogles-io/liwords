package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/domino14/crosswords/pkg/config"
	"github.com/domino14/crosswords/pkg/sockets"
	"github.com/domino14/crosswords/pkg/stores/game"
	"github.com/domino14/crosswords/pkg/stores/soughtgame"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	GracefulShutdownTimeout = 30 * time.Second
)

func main() {
	var dir, addr string
	flag.StringVar(&addr, "addr", ":8087", "http service address")
	flag.StringVar(&dir, "dir", ".", "the directory to serve files from. Defaults to the current dir")
	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Msgf("Loaded config: %v", cfg)

	hub := sockets.NewHub(game.NewMemoryStore(), soughtgame.NewMemoryStore(), cfg)
	go hub.Run()
	go hub.RunGameEventHandler()

	// http.HandleFunc("/")

	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	r.URL.Path = "/"
	// 	staticHandler.ServeHTTP(w, r)
	// })

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		sockets.ServeWS(hub, w, r)
	})
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("ListenAndServe")
	}

	// srv := &http.Server{Addr: ":8088", Handler: handler}

	// idleConnsClosed := make(chan struct{})
	// sig := make(chan os.Signal, 1)

	// go func() {
	// 	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	// 	<-sig
	// 	log.Info().Msg("got quit signal...")
	// 	ctx, cancel := context.WithTimeout(context.Background(), GracefulShutdownTimeout)
	// 	if err := srv.Shutdown(ctx); err != nil {
	// 		// Error from closing listeners, or context timeout:
	// 		log.Error().Msgf("HTTP server Shutdown: %v", err)
	// 	}
	// 	cancel()
	// 	close(idleConnsClosed)
	// }()

	// if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	// 	log.Fatal().Err(err).Msg("")
	// }
	// <-idleConnsClosed
	// log.Info().Msg("server gracefully shutting down")
}
