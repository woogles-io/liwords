package main

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Shutdown is traced as one "api.shutdown" trace, from SIGTERM to the last
// step, with a child span per step ("bus.drain" has its own events, see
// pkg/bus/drain.go). The trace is exported when main returns, by the
// deferred OTel shutdown.

var shutdownTracer = otel.Tracer("shutdown")

// shutdownStep runs one step of the shutdown in its own span.
func shutdownStep(ctx context.Context, name string, step func(ctx context.Context) error) error {
	ctx, span := shutdownTracer.Start(ctx, name)
	defer span.End()
	err := step(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func startShutdownSpan(ctx context.Context) (context.Context, trace.Span) {
	return shutdownTracer.Start(ctx, "api.shutdown")
}

// Streaming endpoints (the event-stream API, OBS overlays) keep a request
// open until the client leaves, and srv.Shutdown waits for every request, so
// one connected stream would hold the shutdown until its deadline. Once
// Shutdown starts, streams get streamGrace, then their request context ends,
// which is what each stream loop waits for; the clients reconnect, to the
// new server. Ordinary requests on these routes finish well within the grace.
const streamGrace = 2 * time.Second

// httpShutdownBudget caps srv.Shutdown, so the archive wait after it still
// gets its time.
const httpShutdownBudget = 5 * time.Second

// endStreamsOnShutdown makes srv's shutdown end the requests handled by the
// handlers wrap returns, streamGrace after it starts.
func endStreamsOnShutdown(srv *http.Server) (wrap func(http.Handler) http.Handler) {
	stop, end := context.WithCancel(context.Background())
	srv.RegisterOnShutdown(func() { time.AfterFunc(streamGrace, end) })
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithCancel(r.Context())
			defer cancel()
			defer context.AfterFunc(stop, cancel)()
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// shutdownHTTP runs srv.Shutdown for at most httpShutdownBudget.
func shutdownHTTP(ctx context.Context, srv *http.Server) error {
	ctx, cancel := context.WithTimeout(ctx, httpShutdownBudget)
	defer cancel()
	return srv.Shutdown(ctx)
}
