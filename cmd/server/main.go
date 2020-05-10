package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/domino14/crosswords/pkg/sockets"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	GracefulShutdownTimeout = 30 * time.Second
)

var addr = flag.String("addr", ":8087", "http service address")

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Debug().Interface("rURL", r.URL).Msg("")
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func main() {
	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	hub := sockets.NewHub()
	go hub.Run()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		sockets.ServeWS(hub, w, r)
	})
	err := http.ListenAndServe(*addr, nil)
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
