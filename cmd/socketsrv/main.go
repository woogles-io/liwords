package main

import (
	"context"
	"expvar"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	h, err := sockets.NewHub(cfg)
	if err != nil {
		panic(err)
	}
	go h.Run()

	if cfg.TargetGroupARN != "" {
		go startDrainWatcher(cfg, h)
	}

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
}

// startDrainWatcher drains h once the load balancer starts draining this
// task. If it can't watch, the server works as before: the load balancer cuts
// every socket at the end of the deregistration delay.
func startDrainWatcher(cfg *config.Config, h *sockets.Hub) {
	ctx := context.Background()
	parsed, err := arn.Parse(cfg.TargetGroupARN)
	if err != nil {
		log.Err(err).Msg("drain-watcher: bad target group ARN")
		return
	}
	ip, err := taskIP(ctx)
	if err != nil {
		log.Err(err).Msg("drain-watcher: no task IP")
		return
	}
	_, portStr, err := net.SplitHostPort(cfg.WebsocketAddress)
	if err != nil {
		log.Err(err).Msg("drain-watcher: bad ws address")
		return
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Err(err).Msg("drain-watcher: bad ws port")
		return
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(parsed.Region))
	if err != nil {
		log.Err(err).Msg("drain-watcher: aws config")
		return
	}
	target := elbtypes.TargetDescription{Id: aws.String(ip), Port: aws.Int32(int32(port))}
	log.Info().Str("target", ip).Int("port", port).Msg("drain-watcher: watching")
	watchForDraining(ctx, elb.NewFromConfig(awsCfg), cfg.TargetGroupARN, target,
		targetHealthPollPeriod, func() { h.Drain(cfg.DrainPeriod) })
}
