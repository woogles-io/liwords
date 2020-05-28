package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/domino14/crosswords/pkg/config"
	"github.com/domino14/crosswords/pkg/sockets"
	"github.com/domino14/crosswords/pkg/stores/game"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	GracefulShutdownTimeout = 30 * time.Second
)

var addr = flag.String("addr", ":8087", "http service address")

// func sendSeek(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != "POST" {
// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 	}
// 	body, err := ioutil.ReadAll(r.Body)
// 	if err != nil {
// 		http.Error(w, "can't read body", http.StatusBadRequest)
// 		return
// 	}

// 	cs := &sockets.ClientSeek{}

// 	err = json.Unmarshal(body, cs)
// 	if err != nil {
// 		http.Error(w, "bad request", http.StatusBadRequest)
// 		return
// 	}
// 	// broadcast seek
// 	hub.NewSeekRequest(cs)

// 	// players := []*macondopb.PlayerInfo{
// 	// 	{Nickname: evt.Acceptor, RealName: evt.Acceptor},
// 	// 	{Nickname: csa.Seeker, RealName: csa.Seeker},
// 	// }

// 	fmt.Fprintf(w, "OK")
// }

// func acceptedSeek(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != "POST" {
// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 	}
// 	body, err := ioutil.ReadAll(r.Body)
// 	if err != nil {
// 		http.Error(w, "can't read body", http.StatusBadRequest)
// 		return
// 	}

// 	csa := &clientSeekAcceptance{}

// 	err = json.Unmarshal(body, csa)
// 	if err != nil {
// 		http.Error(w, "bad request", http.StatusBadRequest)
// 		return
// 	}
// 	// players := []*macondopb.PlayerInfo{
// 	// 	{Nickname: evt.Acceptor, RealName: evt.Acceptor},
// 	// 	{Nickname: csa.Seeker, RealName: csa.Seeker},
// 	// }

// 	// fmt.Fprintf(w, seeker.Seeker)
// }

func main() {

	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Msgf("Loaded config: %v", cfg)

	hub := sockets.NewHub(game.NewMemoryStore(), cfg)
	go hub.Run()
	go hub.RunGameEventHandler()

	// http.HandleFunc("/api/acceptedseek", acceptedSeek)
	// http.HandleFunc("/api/sendseek", sendSeek)
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
