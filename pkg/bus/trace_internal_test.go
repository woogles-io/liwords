package bus

import (
	"context"
	"testing"

	"github.com/matryer/is"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/natstrace"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// An event made while handling a traced message goes out with that trace in
// its headers, so the socket server can continue it; any other event goes
// out with no headers, as before.
func TestPublishEventCarriesTrace(t *testing.T) {
	is := is.New(t)
	prev := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(prev) })

	var sent []*nats.Msg
	b := &Bus{publish: func(m *nats.Msg) error { sent = append(sent, m); return nil }}

	ctx, span := sdktrace.NewTracerProvider().Tracer("test").Start(context.Background(), "handle")
	defer span.End()
	evt := entity.WrapEvent(&pb.ErrorMessage{Message: "x"}, pb.MessageType_ERROR_MESSAGE)
	is.NoErr(b.publishEvent(evt.WithTrace(ctx), "game.abc", []byte("data")))

	plain := entity.WrapEvent(&pb.ErrorMessage{Message: "y"}, pb.MessageType_ERROR_MESSAGE)
	is.NoErr(b.publishEvent(plain, "lobby.x", []byte("data")))

	is.Equal(len(sent), 2)
	is.Equal(sent[0].Subject, "game.abc")
	is.True(natstrace.Traced(sent[0]))
	is.Equal(sent[0].Header.Get("traceparent")[3:35], span.SpanContext().TraceID().String())
	is.Equal(sent[1].Subject, "lobby.x")
	is.Equal(sent[1].Header, nil)
}
