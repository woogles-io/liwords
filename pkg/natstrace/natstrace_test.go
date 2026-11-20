package natstrace

import (
	"context"
	"testing"

	"github.com/matryer/is"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func setup(t *testing.T) trace.Tracer {
	prev := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(prev) })
	return sdktrace.NewTracerProvider().Tracer("test")
}

func TestRoundTrip(t *testing.T) {
	is := is.New(t)
	tracer := setup(t)
	ctx, span := tracer.Start(context.Background(), "send")
	defer span.End()

	msg := NewMsg(ctx, "ipc.pb.1", []byte("data"))
	is.Equal(msg.Subject, "ipc.pb.1")
	is.Equal(string(msg.Data), "data")
	is.True(msg.Header.Get("traceparent") != "")
	is.True(Traced(msg))

	got := trace.SpanContextFromContext(Extract(context.Background(), msg))
	is.Equal(got.TraceID(), span.SpanContext().TraceID())
	is.Equal(got.SpanID(), span.SpanContext().SpanID())
	is.True(got.IsRemote())
}

// A message from a context without a trace gets no headers, so it is
// published exactly as before.
func TestNoTraceNoHeaders(t *testing.T) {
	is := is.New(t)
	setup(t)
	msg := NewMsg(context.Background(), "lobby.x", []byte("data"))
	is.Equal(msg.Header, nil)
	is.True(!Traced(msg))

	ctx := context.WithValue(context.Background(), struct{}{}, 1)
	is.Equal(Extract(ctx, msg), ctx) // nothing to extract: ctx unchanged
}
