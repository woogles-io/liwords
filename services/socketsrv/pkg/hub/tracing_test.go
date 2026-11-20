package sockets

import (
	"context"
	"testing"

	"github.com/matryer/is"
	nats "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/woogles-io/liwords/pkg/natstrace"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func recordSpans(t *testing.T) *tracetest.SpanRecorder {
	rec := tracetest.NewSpanRecorder()
	prevTP, prevProp := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
	})
	return rec
}

// A message from a browser starts a trace, and the message sent to the API
// carries it.
func TestReceiveSpanStartsTrace(t *testing.T) {
	is := is.New(t)
	rec := recordSpans(t)
	c := &Client{userID: "u1", connID: "c1", authenticated: true}

	ctx, span := receiveSpan(context.Background(), c, byte(pb.MessageType_CLIENT_GAMEPLAY_EVENT))
	msg := natstrace.NewMsg(ctx, "ipc.pb.x", []byte("data"))
	span.End()

	is.True(natstrace.Traced(msg))
	ended := rec.Ended()
	is.Equal(len(ended), 1)
	is.Equal(ended[0].Name(), "socket.receive.CLIENT_GAMEPLAY_EVENT")
	is.True(!ended[0].Parent().IsValid()) // a new trace
}

// What the API sends back in that trace is forwarded under a span in the
// same trace, which records how many sockets got it. Messages without a
// trace get no span.
func TestForwardSpanContinuesTrace(t *testing.T) {
	is := is.New(t)
	rec := recordSpans(t)

	ctx, api := otel.Tracer("api").Start(context.Background(), "nats.publish.CLIENT_GAMEPLAY_EVENT")
	data := []byte{0, 1, byte(pb.MessageType_SERVER_GAMEPLAY_EVENT)}
	span := forwardSpan(natstrace.NewMsg(ctx, "game.abc", data))
	is.True(span != nil)
	endForward(span, 2)
	api.End()

	is.Equal(forwardSpan(&nats.Msg{Subject: "lobby.x", Data: data}), nil)
	endForward(nil, 5) // no span: nothing to do

	var fwd sdktrace.ReadOnlySpan
	for _, s := range rec.Ended() {
		if s.Name() == "socket.forward.game" {
			fwd = s
		}
	}
	is.True(fwd != nil)
	is.Equal(fwd.Parent().SpanID(), api.SpanContext().SpanID())
	is.Equal(fwd.SpanContext().TraceID(), api.SpanContext().TraceID())
	attrs := map[attribute.Key]attribute.Value{}
	for _, kv := range fwd.Attributes() {
		attrs[kv.Key] = kv.Value
	}
	is.Equal(attrs["sockets"].AsInt64(), int64(2))
	is.Equal(attrs["msg.type"].AsString(), "SERVER_GAMEPLAY_EVENT")
}
