package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/domino14/macondo/gaddag"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/bus"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/pkg/stores/session"
	"github.com/domino14/liwords/pkg/stores/soughtgame"
	"github.com/domino14/macondo/alphabet"

	"github.com/domino14/liwords/pkg/registration"

	"github.com/domino14/liwords/pkg/auth"

	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/stores/user"
	pkguser "github.com/domino14/liwords/pkg/user"
	gameservice "github.com/domino14/liwords/rpc/api/proto/game_service"
	userservice "github.com/domino14/liwords/rpc/api/proto/user_service"
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

	tmpGameStore, err := game.NewDBStore(cfg, userStore)
	if err != nil {
		panic(err)
	}

	gameStore := game.NewCache(tmpGameStore)
	soughtGameStore := soughtgame.NewMemoryStore()

	authenticationService := auth.NewAuthenticationService(userStore, sessionStore, cfg.SecretKey, cfg.MailgunKey)
	registrationService := registration.NewRegistrationService(userStore)
	gameService := gameplay.NewGameService(userStore, gameStore)
	profileService := pkguser.NewProfileService(userStore)

	router.Handle(userservice.AuthenticationServicePathPrefix,
		middlewares.Then(userservice.NewAuthenticationServiceServer(authenticationService, nil)))

	router.Handle(userservice.RegistrationServicePathPrefix,
		middlewares.Then(userservice.NewRegistrationServiceServer(registrationService, nil)))

	router.Handle(gameservice.GameMetadataServicePathPrefix,
		middlewares.Then(gameservice.NewGameMetadataServiceServer(gameService, nil)))

	router.Handle(userservice.ProfileServicePathPrefix,
		middlewares.Then(userservice.NewProfileServiceServer(profileService, nil)))

	// Create any caches
	alphabet.CreateLetterDistributionCache()
	gaddag.CreateGaddagCache()

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

	ctx, pubsubCancel := context.WithCancel(context.Background())

	go pubsubBus.ProcessMessages(ctx)

	go func() {
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Info().Msg("got quit signal...")
		ctx, shutdownCancel := context.WithTimeout(context.Background(), GracefulShutdownTimeout)
		if err := srv.Shutdown(ctx); err != nil {
			// Error from closing listeners, or context timeout:
			log.Error().Msgf("HTTP server Shutdown: %v", err)
		}
		shutdownCancel()
		pubsubCancel()
		close(idleConnsClosed)
	}()
	log.Info().Msg("starting listening...")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("")
	}
	<-idleConnsClosed
	log.Info().Msg("server gracefully shutting down")
}
