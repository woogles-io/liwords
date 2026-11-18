package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// A webhook send is a child span that records the status, and nothing in
// the trace holds the webhook URL, whose path is the webhook's secret.
func TestSendSpanHasNoWebhookURL(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	status := http.StatusNoContent
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
	defer srv.Close()
	const secret = "s3cr3t-webhook-token"
	webhook := srv.URL + "/api/webhooks/123/" + secret

	ctx, parent := tp.Tracer("test").Start(context.Background(), "job.run")
	if err := Send(ctx, "hello", webhook); err != nil {
		t.Fatal(err)
	}
	status = http.StatusBadRequest
	if err := Send(ctx, "hello", webhook); err == nil {
		t.Fatal("want an error for a 400")
	} else if strings.Contains(err.Error(), secret) {
		t.Fatalf("error holds the webhook secret: %v", err)
	}
	if err := Send(ctx, "hello", "http://127.0.0.1:1/api/webhooks/123/"+secret); err == nil {
		t.Fatal("want an error for a refused connection")
	} else if strings.Contains(err.Error(), secret) {
		t.Fatalf("error holds the webhook secret: %v", err)
	}
	parent.End()

	sends := 0
	for _, s := range rec.Ended() {
		for _, a := range s.Attributes() {
			if strings.Contains(a.Value.Emit(), secret) {
				t.Fatalf("span %q attribute %s holds the webhook secret", s.Name(), a.Key)
			}
		}
		for _, e := range s.Events() {
			for _, a := range e.Attributes {
				if strings.Contains(a.Value.Emit(), secret) {
					t.Fatalf("span %q event %q holds the webhook secret", s.Name(), e.Name)
				}
			}
		}
		if s.Name() == "discord.webhook" {
			sends++
			if s.Parent().SpanID() != parent.SpanContext().SpanID() {
				t.Fatal("discord.webhook is not a child of job.run")
			}
		}
	}
	if sends != 3 {
		t.Fatalf("got %d discord.webhook spans, want 3", sends)
	}
}
