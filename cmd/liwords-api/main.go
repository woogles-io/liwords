package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/domino14/liwords/bus"
	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/pkg/stores/session"
	"github.com/domino14/liwords/pkg/stores/soughtgame"

	"github.com/domino14/liwords/pkg/registration"

	"github.com/domino14/liwords/pkg/auth"

	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/stores/user"
	pb "github.com/domino14/liwords/rpc/api/proto"
)

const (
	GracefulShutdownTimeout = 30 * time.Second
)

func main() {

	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Msgf("Loaded config: %v", cfg)

	if cfg.SecretKey == "" {
		panic("secret key must be non blank")
	}
	if cfg.MacondoConfig.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	log.Debug().Msg("debug log is on")

	router := http.NewServeMux() // here you could also go with third party packages to create a router

	userStore, err := user.NewDBStore(cfg.DBConnString)
	if err != nil {
		panic(err)
	}
	sessionStore, err := session.NewDBStore(cfg.DBConnString)

	middlewares := alice.New(
		hlog.NewHandler(log.With().Str("service", "liwords").Logger()),
		apiserver.WithCookiesMiddleware,
		apiserver.AuthenticationMiddlewareGenerator(sessionStore),
		hlog.AccessHandler(func(r *http.Request, status int, size int, d time.Duration) {
			path := strings.Split(r.URL.Path, "/")
			method := path[len(path)-1]
			hlog.FromRequest(r).Info().Str("method", method).Int("status", status).Dur("duration", d).Msg("")
		}),
	)
	gameStore := game.NewMemoryStore()
	soughtGameStore := soughtgame.NewMemoryStore()

	authenticationService := auth.NewAuthenticationService(userStore, sessionStore, cfg.SecretKey)
	registrationService := registration.NewRegistrationService(userStore)
	gameService := gameplay.NewGameService(userStore, gameStore)

	router.Handle(pb.AuthenticationServicePathPrefix,
		middlewares.Then(pb.NewAuthenticationServiceServer(authenticationService, nil)))

	router.Handle(pb.RegistrationServicePathPrefix,
		middlewares.Then(pb.NewRegistrationServiceServer(registrationService, nil)))

	router.Handle(pb.GameMetadataServicePathPrefix,
		middlewares.Then(pb.NewGameMetadataServiceServer(gameService, nil)))

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      router,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second}

	idleConnsClosed := make(chan struct{})
	sig := make(chan os.Signal, 1)

	// Handle bus.
	pubsubBus, err := bus.NewBus(cfg, userStore, gameStore, soughtGameStore)
	if err != nil {
		panic(err)
	}
	go pubsubBus.ProcessMessages(context.Background())

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
}
