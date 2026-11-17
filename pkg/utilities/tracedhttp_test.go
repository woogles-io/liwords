package utilities

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// A request made with ctx is a span named after its method and host, a
// child of the span in ctx.
func TestTracedTransportSpanIsChild(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	ctx, parent := tp.Tracer("test").Start(context.Background(), "job.run")
	client := &http.Client{Transport: TracedTransport()}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/feed", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	parent.End()

	host := mustHost(t, srv.URL)
	for _, s := range rec.Ended() {
		if s.Name() == "GET "+host {
			if s.Parent().SpanID() != parent.SpanContext().SpanID() {
				t.Fatalf("span %q is not a child of job.run", s.Name())
			}
			return
		}
	}
	t.Fatalf("no span named %q", "GET "+host)
}

func mustHost(t *testing.T, raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u.Host
}
