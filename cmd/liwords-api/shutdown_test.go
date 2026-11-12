package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matryer/is"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// A server with a stream route that stays open until its request ends, as
// the event-stream API and OBS overlays do, and a slow ordinary route.
func streamServer(t *testing.T) (*httptest.Server, chan struct{}) {
	streamStarted := make(chan struct{})
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	streamRoute := endStreamsOnShutdown(ts.Config)
	mux.Handle("/stream", streamRoute(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		close(streamStarted)
		<-r.Context().Done()
	})))
	mux.Handle("/slow", streamRoute(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(300 * time.Millisecond):
			io.WriteString(w, "done")
		case <-r.Context().Done():
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})))
	ts.Start()
	t.Cleanup(ts.Close)
	return ts, streamStarted
}

// An open stream doesn't hold the shutdown: it ends streamGrace after
// Shutdown starts, and a request in progress on the same kind of route
// still finishes normally.
func TestShutdownEndsStreams(t *testing.T) {
	is := is.New(t)
	ts, streamStarted := streamServer(t)

	resp, err := http.Get(ts.URL + "/stream")
	is.NoErr(err)
	defer resp.Body.Close()
	<-streamStarted

	slow := make(chan string, 1)
	go func() {
		r, err := http.Get(ts.URL + "/slow")
		if err != nil {
			slow <- err.Error()
			return
		}
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		slow <- string(b)
	}()
	time.Sleep(50 * time.Millisecond) // the slow request is in progress

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	is.NoErr(shutdownHTTP(ctx, ts.Config))
	took := time.Since(start)
	is.True(took >= streamGrace)
	is.True(took < streamGrace+time.Second)
	is.Equal(<-slow, "done")
}

// An open stream on a route that isn't wrapped would hold Shutdown forever;
// shutdownHTTP gives up at its deadline.
func TestShutdownHTTPDeadline(t *testing.T) {
	is := is.New(t)
	started := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	defer ts.Close()
	resp, err := http.Get(ts.URL)
	is.NoErr(err)
	defer resp.Body.Close()
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	is.True(errors.Is(shutdownHTTP(ctx, ts.Config), context.DeadlineExceeded))
}

// Each shutdown step is a span under api.shutdown, marked when it fails.
func TestShutdownSpans(t *testing.T) {
	is := is.New(t)
	rec := tracetest.NewSpanRecorder()
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	defer otel.SetTracerProvider(prev)
	shutdownTracer = otel.Tracer("shutdown")

	ctx, root := startShutdownSpan(context.Background())
	is.NoErr(shutdownStep(ctx, "ok.step", func(context.Context) error { return nil }))
	is.True(shutdownStep(ctx, "bad.step", func(context.Context) error { return errors.New("x") }) != nil)
	root.End()

	byName := map[string]sdktrace.ReadOnlySpan{}
	for _, s := range rec.Ended() {
		byName[s.Name()] = s
	}
	is.Equal(len(byName), 3)
	rootID := byName["api.shutdown"].SpanContext().SpanID()
	is.Equal(byName["ok.step"].Parent().SpanID(), rootID)
	is.Equal(byName["bad.step"].Parent().SpanID(), rootID)
	is.Equal(byName["ok.step"].Status().Code, codes.Unset)
	is.Equal(byName["bad.step"].Status().Code, codes.Error)
}
