package main

import (
	"context"
	"expvar"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/twitchtv/twirp"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/bus"
	"github.com/domino14/liwords/pkg/comments"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/memento"
	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/notify"
	"github.com/domino14/liwords/pkg/omgwords"
	omgstores "github.com/domino14/liwords/pkg/omgwords/stores"
	"github.com/domino14/liwords/pkg/puzzles"
	commentsstore "github.com/domino14/liwords/pkg/stores/comments"
	cfgstore "github.com/domino14/liwords/pkg/stores/config"
	"github.com/domino14/liwords/pkg/stores/game"
	modstore "github.com/domino14/liwords/pkg/stores/mod"
	puzzlestore "github.com/domino14/liwords/pkg/stores/puzzles"
	"github.com/domino14/liwords/pkg/stores/session"
	"github.com/domino14/liwords/pkg/stores/soughtgame"
	"github.com/domino14/liwords/pkg/stores/stats"
	"github.com/domino14/liwords/pkg/tournament"
	"github.com/domino14/liwords/pkg/utilities"
	"github.com/domino14/liwords/pkg/words"

	"github.com/domino14/liwords/pkg/registration"

	"github.com/domino14/liwords/pkg/auth"

	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/config"
	pkgprofile "github.com/domino14/liwords/pkg/profile"
	pkgredis "github.com/domino14/liwords/pkg/stores/redis"
	tournamentstore "github.com/domino14/liwords/pkg/stores/tournament"
	"github.com/domino14/liwords/pkg/stores/user"
	userservices "github.com/domino14/liwords/pkg/user/services"
	"github.com/domino14/liwords/rpc/api/proto/comments_service"
	configservice "github.com/domino14/liwords/rpc/api/proto/config_service"
	gameservice "github.com/domino14/liwords/rpc/api/proto/game_service"
	modservice "github.com/domino14/liwords/rpc/api/proto/mod_service"
	omgwordsservice "github.com/domino14/liwords/rpc/api/proto/omgwords_service"
	puzzleservice "github.com/domino14/liwords/rpc/api/proto/puzzle_service"
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

// NewLoggingServerHooks logs request and errors to stdout in the service
func NewLoggingServerHooks() *twirp.ServerHooks {
	return &twirp.ServerHooks{
		Error: func(ctx context.Context, twerr twirp.Error) context.Context {
			method, _ := twirp.MethodName(ctx)
			log.Err(twerr).
				Str("code", string(twerr.Code())).
				Str("method", method).
				Msg("api-error")
			// Currently the only Woogles Errors are tournament errors
			// so this will need to be changed later.
			if len(twerr.Msg()) > 0 &&
				string(twerr.Msg()[0]) == entity.WooglesErrorDelimiter &&
				os.Getenv("TournamentDiscordToken") != "" {
				notify.Post(fmt.Sprintf("%s\n%s", twerr.Msg(), time.Now().Format(time.RFC3339)), os.Getenv("TournamentDiscordToken"))
			}
			return ctx
		},
	}
}

// this CORS middleware is needed for SharedArrayBuffer to be enabled!
// func corsMiddlewareGenerator() (mw func(http.Handler) http.Handler) {
// 	mw = func(h http.Handler) http.Handler {
// 		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.Header().Add("Cross-Origin-Opener-Policy", "same-origin")
// 			w.Header().Add("Cross-Origin-Embedder-Policy", "require-corp")
// 			h.ServeHTTP(w, r)
// 		})
// 	}
// 	return
// }

// func NewInterceptorCustomError() twirp.Interceptor {
// 	return func(next twirp.Method) twirp.Method {
// 		return func(ctx context.Context, req interface{}) (interface{}, error) {

// 			resp, err := next(ctx, req)
// 			if err != nil {
// 				switch err.(type) {
// 				case twirp.Error:

// 				}
// 			}
// 		}
// 	}
// }

func main() {
	log.Info().Msg("before load")
	cfg := &config.Config{}
	log.Info().Msg("after cfg")
	cfg.Load(os.Args[1:])
	log.Info().Msg("after load")
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

	if os.Getenv("USE_LOCALSTACK_S3") == "1" {
		// pre-create buckets
		precreateLocalStackBuckets()
	}
	log.Info().Msg("setting up migration")
	m, err := migrate.New(cfg.DBMigrationsPath, cfg.DBConnUri)
	if err != nil {
		panic(err)
	}
	log.Info().Msg("bringing up migration")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		panic(err)
	}
	e1, e2 := m.Close()
	log.Err(e1).Msg("close-source")
	log.Err(e2).Msg("close-database")

	redisPool := newPool(cfg.RedisURL)
	dbPool, err := pgxpool.Connect(context.Background(), cfg.DBConnUri)
	if err != nil {
		panic(err)
	}

	router := http.NewServeMux() // here you could also go with third party packages to create a router

	stores := bus.Stores{}

	stores.UserStore, err = user.NewDBStore(dbPool)
	if err != nil {
		panic(err)
	}

	stores.SessionStore, err = session.NewDBStore(dbPool)
	if err != nil {
		panic(err)
	}

	middlewares := alice.New(
		hlog.NewHandler(log.With().Str("service", "liwords").Logger()),
		apiserver.ExposeResponseWriterMiddleware,
		apiserver.AuthenticationMiddlewareGenerator(stores.SessionStore),
		apiserver.APIKeyMiddlewareGenerator(),
		config.CtxMiddlewareGenerator(cfg),
		// corsMiddlewareGenerator(),
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

	stores.SoughtGameStore, err = soughtgame.NewDBStore(dbPool)
	if err != nil {
		panic(err)
	}
	stores.ConfigStore = cfgstore.NewRedisConfigStore(redisPool)
	stores.ListStatStore, err = stats.NewDBStore(dbPool)
	if err != nil {
		panic(err)
	}

	stores.NotorietyStore, err = modstore.NewDBStore(dbPool)
	if err != nil {
		panic(err)
	}
	stores.PresenceStore = pkgredis.NewRedisPresenceStore(redisPool)
	stores.ChatStore = pkgredis.NewRedisChatStore(redisPool, stores.PresenceStore, stores.TournamentStore)

	stores.PuzzleStore, err = puzzlestore.NewDBStore(dbPool)
	if err != nil {
		panic(err)
	}

	// s3 config

	awscfg, err := awsconfig.LoadDefaultConfig(
		context.Background(), awsconfig.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(utilities.CustomResolver)))
	if err != nil {
		panic(err)
	}
	s3Client := s3.NewFromConfig(awscfg, utilities.CustomClientOptions)

	stores.GameDocumentStore, err = omgstores.NewGameDocumentStore(cfg, redisPool, dbPool)
	if err != nil {
		panic(err)
	}
	stores.AnnotatedGameStore, err = omgstores.NewDBStore(dbPool)
	if err != nil {
		panic(err)
	}
	commentsStore, err := commentsstore.NewDBStore(dbPool)
	if err != nil {
		panic(err)
	}

	mementoService := memento.NewMementoService(stores.UserStore, stores.GameStore,
		stores.GameDocumentStore, cfg)
	authenticationService := auth.NewAuthenticationService(stores.UserStore, stores.SessionStore, stores.ConfigStore,
		cfg.SecretKey, cfg.MailgunKey, cfg.DiscordToken, cfg.ArgonConfig)
	registrationService := registration.NewRegistrationService(stores.UserStore, cfg.ArgonConfig)
	gameService := gameplay.NewGameService(stores.UserStore, stores.GameStore, stores.GameDocumentStore, cfg)
	profileService := pkgprofile.NewProfileService(stores.UserStore, userservices.NewS3Uploader(os.Getenv("AVATAR_UPLOAD_BUCKET"), s3Client))
	wordService := words.NewWordService(&cfg.MacondoConfig)
	autocompleteService := userservices.NewAutocompleteService(stores.UserStore)
	socializeService := userservices.NewSocializeService(stores.UserStore, stores.ChatStore, stores.PresenceStore)
	configService := config.NewConfigService(stores.ConfigStore, stores.UserStore)
	tournamentService := tournament.NewTournamentService(stores.TournamentStore, stores.UserStore)
	modService := mod.NewModService(stores.UserStore, stores.ChatStore)
	puzzleService := puzzles.NewPuzzleService(stores.PuzzleStore, stores.UserStore, cfg.PuzzleGenerationSecretKey, cfg.ECSClusterName, cfg.PuzzleGenerationTaskDefinition)
	omgwordsService := omgwords.NewOMGWordsService(stores.UserStore, cfg, stores.GameDocumentStore, stores.AnnotatedGameStore)
	commentService := comments.NewCommentsService(stores.UserStore, stores.GameStore, commentsStore)

	router.Handle("/ping", http.HandlerFunc(pingEndpoint))

	router.Handle(memento.GameimgPrefix, middlewares.Then(mementoService))

	router.Handle(userservice.AuthenticationServicePathPrefix,
		middlewares.Then(userservice.NewAuthenticationServiceServer(authenticationService, NewLoggingServerHooks())))

	router.Handle(userservice.RegistrationServicePathPrefix,
		middlewares.Then(userservice.NewRegistrationServiceServer(registrationService, NewLoggingServerHooks())))

	router.Handle(gameservice.GameMetadataServicePathPrefix,
		middlewares.Then(gameservice.NewGameMetadataServiceServer(gameService, NewLoggingServerHooks())))

	router.Handle(omgwordsservice.GameEventServicePathPrefix,
		middlewares.Then(omgwordsservice.NewGameEventServiceServer(omgwordsService, NewLoggingServerHooks())))

	router.Handle(wordservice.WordServicePathPrefix,
		middlewares.Then(wordservice.NewWordServiceServer(wordService, NewLoggingServerHooks())))

	router.Handle(userservice.ProfileServicePathPrefix,
		middlewares.Then(userservice.NewProfileServiceServer(profileService, NewLoggingServerHooks())))

	router.Handle(userservice.AutocompleteServicePathPrefix,
		middlewares.Then(userservice.NewAutocompleteServiceServer(autocompleteService, NewLoggingServerHooks())))

	router.Handle(userservice.SocializeServicePathPrefix,
		middlewares.Then(userservice.NewSocializeServiceServer(socializeService, NewLoggingServerHooks())))

	router.Handle(configservice.ConfigServicePathPrefix,
		middlewares.Then(configservice.NewConfigServiceServer(configService, NewLoggingServerHooks())))

	router.Handle(tournamentservice.TournamentServicePathPrefix,
		middlewares.Then(tournamentservice.NewTournamentServiceServer(tournamentService, NewLoggingServerHooks())))

	router.Handle(modservice.ModServicePathPrefix,
		middlewares.Then(modservice.NewModServiceServer(modService, NewLoggingServerHooks())))

	router.Handle(puzzleservice.PuzzleServicePathPrefix,
		middlewares.Then(puzzleservice.NewPuzzleServiceServer(puzzleService, NewLoggingServerHooks())))

	router.Handle(comments_service.GameCommentServicePathPrefix,
		middlewares.Then(comments_service.NewGameCommentServiceServer(commentService, NewLoggingServerHooks())))

	router.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	router.Handle(
		"/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	router.Handle(
		"/debug/pprof/trace", pprof.Handler("trace"),
	)
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

	idleConnsClosed := make(chan struct{})
	sig := make(chan os.Signal, 1)

	// Handle bus.
	pubsubBus, err := bus.NewBus(cfg, stores, redisPool)
	if err != nil {
		panic(err)
	}
	tournamentService.SetEventChannel(pubsubBus.TournamentEventChannel())
	omgwordsService.SetEventChannel(pubsubBus.GameEventChannel())

	router.Handle(bus.GameEventStreamPrefix,
		middlewares.Then(pubsubBus.EventAPIServerInstance()))

	ctx, pubsubCancel := context.WithCancel(context.Background())

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      router,
		WriteTimeout: 120 * time.Second,
		ReadTimeout:  10 * time.Second}

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

func precreateLocalStackBuckets() {
	ctx := context.Background()
	cfg, err := awsconfig.LoadDefaultConfig(
		ctx, awsconfig.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(utilities.CustomResolver)))
	if err != nil {
		log.Err(err).Msg("unable-to-load-awsconfig")
		return
	}

	client := s3.NewFromConfig(cfg, utilities.CustomClientOptions)

	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(os.Getenv("AVATAR_UPLOAD_BUCKET")),
	})
	log.Err(err).Msg("trying to create avatar upload bucket")

	// 	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
	// 		Bucket: aws.String(os.Getenv("GAMEDOC_UPLOAD_BUCKET")),
	// 	})
	// 	log.Err(err).Msg("trying to create gamedoc upload bucket")

}
