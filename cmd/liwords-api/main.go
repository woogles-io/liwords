package main

import (
	"context"
	"expvar"
	_ "expvar"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/bus"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/mod"
	cfgstore "github.com/domino14/liwords/pkg/stores/config"
	"github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/pkg/stores/session"
	"github.com/domino14/liwords/pkg/stores/soughtgame"
	"github.com/domino14/liwords/pkg/stores/stats"
	"github.com/domino14/liwords/pkg/tournament"
	"github.com/domino14/liwords/pkg/words"

	"github.com/domino14/liwords/pkg/registration"

	"github.com/domino14/liwords/pkg/auth"

	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/config"
	pkgprofile "github.com/domino14/liwords/pkg/profile"
	tournamentstore "github.com/domino14/liwords/pkg/stores/tournament"
	"github.com/domino14/liwords/pkg/stores/user"
	pkguser "github.com/domino14/liwords/pkg/user"
	configservice "github.com/domino14/liwords/rpc/api/proto/config_service"
	gameservice "github.com/domino14/liwords/rpc/api/proto/game_service"
	modservice "github.com/domino14/liwords/rpc/api/proto/mod_service"
	tournamentservice "github.com/domino14/liwords/rpc/api/proto/tournament_service"
	userservice "github.com/domino14/liwords/rpc/api/proto/user_service"
	wordservice "github.com/domino14/liwords/rpc/api/proto/word_service"

	"net/http/pprof"
)

const (
	GracefulShutdownTimeout = 30 * time.Second
)

var (
	// BuildHash is the git hash, set by go build flags
	BuildHash = "unknown"
	// BuildDate is the build date, set by go build flags
	BuildDate = "unknown"
)

func newPool(addr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
		Dial: func() (redis.Conn, error) { return redis.DialURL(addr) },
	}
}

func pingEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(`{"status":"copacetic"}`))
}

func main() {

	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Interface("config", cfg).
		Str("build-date", BuildDate).Str("build-hash", BuildHash).Msg("started")

	if cfg.SecretKey == "" {
		panic("secret key must be non blank")
	}
	if cfg.MacondoConfig.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	log.Debug().Msg("debug log is on")

	redisPool := newPool(cfg.RedisURL)

	router := http.NewServeMux() // here you could also go with third party packages to create a router

	tmpUserStore, err := user.NewDBStore(cfg.DBConnString)
	if err != nil {
		panic(err)
	}
	stores := bus.Stores{}

	stores.UserStore = user.NewCache(tmpUserStore)

	stores.SessionStore, err = session.NewDBStore(cfg.DBConnString)

	middlewares := alice.New(
		hlog.NewHandler(log.With().Str("service", "liwords").Logger()),
		apiserver.WithCookiesMiddleware,
		apiserver.AuthenticationMiddlewareGenerator(stores.SessionStore),
		apiserver.APIKeyMiddlewareGenerator(),
		hlog.AccessHandler(func(r *http.Request, status int, size int, d time.Duration) {
			path := strings.Split(r.URL.Path, "/")
			method := path[len(path)-1]
			hlog.FromRequest(r).Info().Str("method", method).Int("status", status).Dur("duration", d).Msg("")
		}),
	)

	tmpGameStore, err := game.NewDBStore(cfg, stores.UserStore)
	if err != nil {
		panic(err)
	}

	stores.GameStore = game.NewCache(tmpGameStore)

	tmpTournamentStore, err := tournamentstore.NewDBStore(cfg, stores.GameStore)
	if err != nil {
		panic(err)
	}
	stores.TournamentStore = tournamentstore.NewCache(tmpTournamentStore)

	stores.SoughtGameStore, err = soughtgame.NewDBStore(cfg)
	if err != nil {
		panic(err)
	}
	stores.ConfigStore = cfgstore.NewRedisConfigStore(redisPool)
	stores.ListStatStore, err = stats.NewListStatStore(cfg.DBConnString)
	if err != nil {
		panic(err)
	}
	stores.PresenceStore = user.NewRedisPresenceStore(redisPool)
	stores.ChatStore = user.NewRedisChatStore(redisPool, stores.PresenceStore, stores.TournamentStore)

	authenticationService := auth.NewAuthenticationService(stores.UserStore, stores.SessionStore, stores.ConfigStore,
		cfg.SecretKey, cfg.MailgunKey, cfg.ArgonConfig)
	registrationService := registration.NewRegistrationService(stores.UserStore, cfg.ArgonConfig)
	gameService := gameplay.NewGameService(stores.UserStore, stores.GameStore)
	profileService := pkgprofile.NewProfileService(stores.UserStore, pkguser.NewS3Uploader(os.Getenv("AVATAR_UPLOAD_BUCKET")))
	wordService := words.NewWordService(&cfg.MacondoConfig)
	autocompleteService := pkguser.NewAutocompleteService(stores.UserStore)
	socializeService := pkguser.NewSocializeService(stores.UserStore, stores.ChatStore)
	configService := config.NewConfigService(stores.ConfigStore, stores.UserStore)
	tournamentService := tournament.NewTournamentService(stores.TournamentStore, stores.UserStore)
	modService := mod.NewModService(stores.UserStore, stores.ChatStore, cfg.MailgunKey, cfg.DiscordToken)

	router.Handle("/ping", http.HandlerFunc(pingEndpoint))

	router.Handle(userservice.AuthenticationServicePathPrefix,
		middlewares.Then(userservice.NewAuthenticationServiceServer(authenticationService, nil)))

	router.Handle(userservice.RegistrationServicePathPrefix,
		middlewares.Then(userservice.NewRegistrationServiceServer(registrationService, nil)))

	router.Handle(gameservice.GameMetadataServicePathPrefix,
		middlewares.Then(gameservice.NewGameMetadataServiceServer(gameService, nil)))

	router.Handle(wordservice.WordServicePathPrefix,
		middlewares.Then(wordservice.NewWordServiceServer(wordService, nil)))

	router.Handle(userservice.ProfileServicePathPrefix,
		middlewares.Then(userservice.NewProfileServiceServer(profileService, nil)))

	router.Handle(userservice.AutocompleteServicePathPrefix,
		middlewares.Then(userservice.NewAutocompleteServiceServer(autocompleteService, nil)))

	router.Handle(userservice.SocializeServicePathPrefix,
		middlewares.Then(userservice.NewSocializeServiceServer(socializeService, nil)))

	router.Handle(configservice.ConfigServicePathPrefix,
		middlewares.Then(configservice.NewConfigServiceServer(configService, nil)))

	router.Handle(tournamentservice.TournamentServicePathPrefix,
		middlewares.Then(tournamentservice.NewTournamentServiceServer(tournamentService, nil)))

	router.Handle(modservice.ModServicePathPrefix,
		middlewares.Then(modservice.NewModServiceServer(modService, nil)))

	router.Handle(
		"/debug/pprof/goroutine", pprof.Handler("goroutine"),
	)
	router.Handle(
		"/debug/pprof/heap", pprof.Handler("heap"),
	)
	router.Handle(
		"/debug/vars", http.DefaultServeMux,
	)

	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return fmt.Sprintf("%d", runtime.NumGoroutine())
	}))

	expvar.Publish("gameCacheSize", expvar.Func(func() interface{} {
		ct := stores.GameStore.CachedCount(context.Background())
		return fmt.Sprintf("%d", ct)
	}))

	expvar.Publish("userCacheSize", expvar.Func(func() interface{} {
		ct := stores.UserStore.CachedCount(context.Background())
		return fmt.Sprintf("%d", ct)
	}))

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      router,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second}

	idleConnsClosed := make(chan struct{})
	sig := make(chan os.Signal, 1)

	// Handle bus.
	pubsubBus, err := bus.NewBus(cfg, stores, redisPool)
	if err != nil {
		panic(err)
	}
	tournamentService.SetEventChannel(pubsubBus.TournamentEventChannel())

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
	// XXX: We need to wait until all goroutines end. Not just the pubsub but possibly the bot,
	// etc.
	<-idleConnsClosed
	log.Info().Msg("server gracefully shutting down")
}
