package main

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/exaring/otelpgx"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justinas/alice"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	otelredisoption "github.com/signalfx/splunk-otel-go/instrumentation/github.com/gomodule/redigo/splunkredigo/option"
	splunkredis "github.com/signalfx/splunk-otel-go/instrumentation/github.com/gomodule/redigo/splunkredigo/redis"

	"go.akshayshah.org/connectproto"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth"
	"github.com/woogles-io/liwords/pkg/bus"
	"github.com/woogles-io/liwords/pkg/collections"
	"github.com/woogles-io/liwords/pkg/comments"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/embed"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/integrations"
	"github.com/woogles-io/liwords/pkg/league"
	"github.com/woogles-io/liwords/pkg/memento"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/omgwords"
	"github.com/woogles-io/liwords/pkg/pair"
	pkgprofile "github.com/woogles-io/liwords/pkg/profile"
	"github.com/woogles-io/liwords/pkg/puzzles"
	"github.com/woogles-io/liwords/pkg/registration"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/tournament"
	userservices "github.com/woogles-io/liwords/pkg/user/services"
	"github.com/woogles-io/liwords/pkg/utilities"
	"github.com/woogles-io/liwords/pkg/vdowebhook"
	"github.com/woogles-io/liwords/pkg/words"
	"github.com/woogles-io/liwords/rpc/api/proto/collections_service/collections_serviceconnect"
	"github.com/woogles-io/liwords/rpc/api/proto/comments_service/comments_serviceconnect"
	"github.com/woogles-io/liwords/rpc/api/proto/config_service/config_serviceconnect"
	"github.com/woogles-io/liwords/rpc/api/proto/game_service/game_serviceconnect"
	"github.com/woogles-io/liwords/rpc/api/proto/league_service/league_serviceconnect"
	"github.com/woogles-io/liwords/rpc/api/proto/mod_service/mod_serviceconnect"
	"github.com/woogles-io/liwords/rpc/api/proto/omgwords_service/omgwords_serviceconnect"
	"github.com/woogles-io/liwords/rpc/api/proto/pair_service/pair_serviceconnect"
	"github.com/woogles-io/liwords/rpc/api/proto/puzzle_service/puzzle_serviceconnect"
	"github.com/woogles-io/liwords/rpc/api/proto/tournament_service/tournament_serviceconnect"
	"github.com/woogles-io/liwords/rpc/api/proto/user_service/user_serviceconnect"
	"github.com/woogles-io/liwords/rpc/api/proto/word_service/word_serviceconnect"
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
	db := 0
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
		Dial: func() (redis.Conn, error) {
			return splunkredis.DialURL(addr,
				otelredisoption.WithAttributes([]attribute.KeyValue{
					semconv.DBRedisDBIndexKey.Int(db),
				}),
			)
		},
	}
}

func pingEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(`{"status":"copacetic"}`))
}

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
	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	log.Debug().Msg("debug log is on")

	// Run migrations if configured to do so.
	// In production, migrations are run via a separate ECS task before deployment.
	if cfg.RunMigrations {
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
		log.Info().Msg("migrations completed successfully")

		// If configured to quit after migration, exit now
		if cfg.QuitAfterMigration {
			log.Info().Msg("quit-after-migration enabled - exiting")
			return
		}
	} else {
		log.Info().Msg("skipping migrations (run-migrations disabled)")
	}
	ctx, pubsubCancel := context.WithCancel(context.Background())

	// Set up OpenTelemetry.
	otelShutdown, err := setupOTelSDK(ctx)
	if err != nil {
		return
	}

	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	redisPool := newPool(cfg.RedisURL)
	dbCfg, err := pgxpool.ParseConfig(cfg.DBConnUri)
	if err != nil {
		panic(err)
	}
	dbCfg.ConnConfig.Tracer = otelpgx.NewTracer(otelpgx.WithIncludeQueryParameters())
	dbPool, err := pgxpool.NewWithConfig(ctx, dbCfg)

	router := http.NewServeMux()
	stores, err := stores.NewInitializedStores(dbPool, redisPool, cfg)
	if err != nil {
		panic(err)
	}

	middlewares := alice.New(
		hlog.NewHandler(log.With().Str("service", "liwords").Logger()),
		apiserver.ExposeResponseWriterMiddleware,
		apiserver.AuthenticationMiddlewareGenerator(stores.SessionStore, cfg.SecureCookies),
		apiserver.APIKeyMiddlewareGenerator(),
		config.CtxMiddlewareGenerator(cfg),

		hlog.AccessHandler(func(r *http.Request, status int, size int, d time.Duration) {
			path := strings.Split(r.URL.Path, "/")
			method := path[len(path)-1]
			hlog.FromRequest(r).Info().
				Str("method", method).
				Int("status", status).
				Dur("duration", d).
				Msg("")
		}),
		ErrorReqResLoggingMiddleware,
	)

	// s3 config

	awscfg, err := awsconfig.LoadDefaultConfig(
		context.Background(), awsconfig.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(utilities.CustomResolver)))
	if err != nil {
		panic(err)
	}
	otelaws.AppendMiddlewares(&awscfg.APIOptions)
	s3Client := s3.NewFromConfig(awscfg, utilities.CustomClientOptions)
	lambdaClient := lambda.NewFromConfig(awscfg)

	mementoService := memento.NewMementoService(stores.UserStore, stores.GameStore,
		stores.GameDocumentStore, cfg)
	embedService := embed.NewEmbedService(stores.GameDocumentStore)
	oauthIntegrationService := integrations.NewOAuthIntegrationService(stores.SessionStore, stores.Queries, cfg)
	integrationService := integrations.NewIntegrationService(stores.Queries)
	authenticationService := auth.NewAuthenticationService(stores.UserStore, stores.SessionStore, stores.ConfigStore,
		cfg.SecretKey, cfg.MailgunKey, cfg.DiscordToken, cfg.ArgonConfig, cfg.SecureCookies, stores.Queries)
	authorizationService := auth.NewAuthorizationService(stores.UserStore, stores.Queries)
	registrationService := registration.NewRegistrationService(stores.UserStore, cfg.ArgonConfig, cfg.MailgunKey, cfg.SkipEmailVerification)
	gameService := gameplay.NewGameService(stores.UserStore, stores.GameStore, stores.GameDocumentStore, cfg, stores.Queries)
	profileService := pkgprofile.NewProfileService(stores.UserStore, userservices.NewS3Uploader(os.Getenv("AVATAR_UPLOAD_BUCKET"), s3Client), stores.Queries)
	wordService := words.NewWordService(cfg)
	autocompleteService := userservices.NewAutocompleteService(stores.UserStore)
	socializeService := userservices.NewSocializeService(stores.UserStore, stores.ChatStore, stores.PresenceStore, stores.Queries)
	configService := config.NewConfigService(stores.ConfigStore, stores.UserStore, stores.Queries)
	tournamentService := tournament.NewTournamentService(stores.TournamentStore, stores.UserStore, cfg, lambdaClient, stores.Queries)
	gameCreatorAdapter := &GameCreatorAdapter{
		stores:    stores,
		cfg:       cfg,
		eventChan: nil, // Set later after pubsubBus is created
	}
	leagueService := league.NewLeagueService(stores.LeagueStore, stores.UserStore, cfg, stores.Queries, stores, gameCreatorAdapter)
	modService := mod.NewModService(stores.UserStore, stores.ChatStore, stores.Queries)
	puzzleService := puzzles.NewPuzzleService(stores.PuzzleStore, stores.UserStore, cfg.PuzzleGenerationSecretKey, cfg.ECSClusterName, cfg.PuzzleGenerationTaskDefinition, stores.Queries)
	omgwordsService := omgwords.NewOMGWordsService(stores.UserStore, cfg, stores.GameDocumentStore, stores.AnnotatedGameStore)
	commentService := comments.NewCommentsService(stores.UserStore, stores.GameStore, stores.CommentsStore, stores.Queries)
	collectionsService := collections.NewCollectionsService(stores.UserStore, stores.Queries, dbPool)
	pairService := pair.NewPairService(cfg, lambdaClient)
	vdoWebhookService := vdowebhook.NewVDOWebhookService(stores.TournamentStore, cfg.VDOPollingIntervalSeconds)
	router.Handle("/ping", http.HandlerFunc(pingEndpoint))

	otcInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		panic("could not set up otelconnect interceptor")
	}

	customHTTPSpanNameFormatter := func(opName string, r *http.Request) string {
		return r.Method + " " + r.URL.Path
	}

	router.Handle(memento.GameimgPrefix, otelhttp.WithRouteTag(memento.GameimgPrefix, otelhttp.NewHandler(
		middlewares.Then(mementoService),
		"memento-api",
		otelhttp.WithSpanNameFormatter(customHTTPSpanNameFormatter),
	)))
	router.Handle(embed.EmbedServicePrefix, otelhttp.WithRouteTag(embed.EmbedServicePrefix, otelhttp.NewHandler(
		middlewares.Then(embedService),
		"embed-api",
		otelhttp.WithSpanNameFormatter(customHTTPSpanNameFormatter),
	)))
	router.Handle(integrations.OAuthIntegrationServicePrefix,
		otelhttp.WithRouteTag(integrations.OAuthIntegrationServicePrefix, otelhttp.NewHandler(
			middlewares.Then(oauthIntegrationService),
			"oauth-integration-handlers",
			otelhttp.WithSpanNameFormatter(customHTTPSpanNameFormatter),
		)))
	// VDO webhook doesn't need authentication middleware - it handles CORS directly
	router.Handle("/api/vdo-webhook", otelhttp.WithRouteTag("/api/vdo-webhook", otelhttp.NewHandler(
		vdoWebhookService,
		"vdo-webhook-api",
		otelhttp.WithSpanNameFormatter(customHTTPSpanNameFormatter),
	)))

	interceptors := connect.WithInterceptors(otcInterceptor)
	// We want to emit default values for backwards compatibility.
	// see https://github.com/connectrpc/connect-go/issues/684
	// Use this 3rd party package until connectrpc exposes this.
	opt := connectproto.WithJSON(
		protojson.MarshalOptions{EmitDefaultValues: true, UseProtoNames: true},
		protojson.UnmarshalOptions{DiscardUnknown: true},
	)

	options := connect.WithHandlerOptions(opt, interceptors, connect.WithCompressMinBytes(500))

	connectapi := http.NewServeMux()
	connectapi.Handle(
		user_serviceconnect.NewAuthenticationServiceHandler(authenticationService, options),
	)
	connectapi.Handle(
		user_serviceconnect.NewRegistrationServiceHandler(registrationService, options),
	)
	connectapi.Handle(
		user_serviceconnect.NewProfileServiceHandler(profileService, options),
	)
	connectapi.Handle(
		user_serviceconnect.NewAutocompleteServiceHandler(autocompleteService, options),
	)
	connectapi.Handle(
		user_serviceconnect.NewSocializeServiceHandler(socializeService, options),
	)
	connectapi.Handle(
		game_serviceconnect.NewGameMetadataServiceHandler(gameService, options),
	)
	connectapi.Handle(
		omgwords_serviceconnect.NewGameEventServiceHandler(omgwordsService, options),
	)
	connectapi.Handle(
		word_serviceconnect.NewWordServiceHandler(wordService, options),
	)
	connectapi.Handle(
		config_serviceconnect.NewConfigServiceHandler(configService, options),
	)
	connectapi.Handle(
		tournament_serviceconnect.NewTournamentServiceHandler(tournamentService, options),
	)
	connectapi.Handle(
		league_serviceconnect.NewLeagueServiceHandler(leagueService, options),
	)
	connectapi.Handle(
		mod_serviceconnect.NewModServiceHandler(modService, options),
	)
	connectapi.Handle(
		puzzle_serviceconnect.NewPuzzleServiceHandler(puzzleService, options),
	)
	connectapi.Handle(
		comments_serviceconnect.NewGameCommentServiceHandler(commentService, options),
	)
	connectapi.Handle(
		collections_serviceconnect.NewCollectionsServiceHandler(collectionsService, options),
	)
	connectapi.Handle(
		user_serviceconnect.NewIntegrationServiceHandler(integrationService, options),
	)
	connectapi.Handle(
		pair_serviceconnect.NewPairServiceHandler(pairService, options),
	)
	connectapi.Handle(
		user_serviceconnect.NewAuthorizationServiceHandler(authorizationService, options),
	)

	connectapichain := middlewares.Then(connectapi)

	router.Handle("/api/", otelhttp.WithRouteTag("/api/",
		otelhttp.NewHandler(http.StripPrefix("/api", connectapichain),
			"api",
			otelhttp.WithSpanNameFormatter(customHTTPSpanNameFormatter))))

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

	natsconn, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		panic(err)
	}

	// Handle bus.
	pubsubBus, err := bus.NewBus(cfg, natsconn, stores, redisPool)
	if err != nil {
		panic(err)
	}
	tournamentService.SetEventChannel(pubsubBus.TournamentEventChannel())
	omgwordsService.SetEventChannel(pubsubBus.GameEventChannel())
	gameCreatorAdapter.eventChan = pubsubBus.GameEventChannel()

	router.Handle(bus.GameEventStreamPrefix,
		middlewares.Then(pubsubBus.EventAPIServerInstance()))

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      router,
		WriteTimeout: 120 * time.Second,
		ReadTimeout:  10 * time.Second}

	go pubsubBus.ProcessMessages(ctx)
	go vdoWebhookService.Start(ctx)

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

// func customInterceptor() connect.UnaryInterceptorFunc {
// 	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
// 		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
// 			printStackContexts(50) // Adjust the depth as needed to explore the call stack

// 			pc, file, line, ok := runtime.Caller(4)
// 			if !ok {
// 				return next(ctx, req)
// 			}
// 			_, span := tracer.Start(ctx, "custom-context", trace.WithAttributes(
// 				// Capture caller file and line information
// 				attribute.String("code.file", file),
// 				attribute.String("code.func", runtime.FuncForPC(pc).Name()),
// 				attribute.Int("code.line", line),
// 			))
// 			defer span.End()
// 			return next(ctx, req)
// 		})
// 	}
// 	return connect.UnaryInterceptorFunc(interceptor)
// }
