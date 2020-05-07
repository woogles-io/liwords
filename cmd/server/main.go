package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/domino14/crosswords/rpc/api/proto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/domino14/crosswords/pkg/server"
)

const (
	GracefulShutdownTimeout = 30 * time.Second
)

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	gameService := &server.GameService{}
	handler := pb.NewCrosswordGameServiceServer(gameService, nil)

	srv := &http.Server{Addr: ":8088", Handler: handler}

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

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("")
	}
	<-idleConnsClosed
	log.Info().Msg("server gracefully shutting down")
}
