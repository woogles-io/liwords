package sockets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/matryer/is"
	nats "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

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
	fctx := forwardContext(natstrace.NewMsg(ctx, "game.abc", data))
	is.True(trace.SpanFromContext(fctx).IsRecording())
	endForward(fctx, 2)
	api.End()

	plain := forwardContext(&nats.Msg{Subject: "lobby.x", Data: data})
	is.Equal(plain, context.Background())
	endForward(plain, 5) // no span: nothing to do

	fwd := spanNamed(rec, "socket.forward.game")
	is.True(fwd != nil)
	is.Equal(fwd.Parent().SpanID(), api.SpanContext().SpanID())
	is.Equal(fwd.SpanContext().TraceID(), api.SpanContext().TraceID())
	attrs := attrMap(fwd)
	is.Equal(attrs["sockets"].AsInt64(), int64(2))
	is.Equal(attrs["msg.type"].AsString(), "SERVER_GAMEPLAY_EVENT")
}

// A message queued for few enough sockets carries the forward span, so each
// socket's write can be recorded under it; one for more sockets doesn't.
func TestOutgoingTracesOnlySmallDeliveries(t *testing.T) {
	is := is.New(t)
	recordSpans(t)
	ctx, span := tracer().Start(context.Background(), "socket.forward.game")
	defer span.End()

	is.True(outgoing(ctx, []byte("x"), maxTracedWrites).trace.IsValid())
	is.True(!outgoing(ctx, []byte("x"), maxTracedWrites+1).trace.IsValid())
	is.True(!outgoing(context.Background(), []byte("x"), 1).trace.IsValid())
}

// The write loop records a socket.write span, under the forward span, for
// each traced message it writes, including ones batched together.
func TestWritePumpRecordsWrites(t *testing.T) {
	is := is.New(t)
	rec := recordSpans(t)
	ctx, fwd := tracer().Start(context.Background(), "socket.forward.game")
	queued := []outMsg{
		outgoing(ctx, []byte{0, 1, 1}, 2),
		outgoing(context.Background(), []byte{0, 1, 2}, 2), // not traced
		outgoing(ctx, []byte{0, 1, 3}, 2),
	}
	fwd.End()

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		c := &Client{conn: conn, userID: "u1", connID: "c1", send: make(chan outMsg, len(queued))}
		for _, m := range queued {
			c.send <- m
		}
		go c.writePump()
	}))
	defer srv.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	is.NoErr(err)
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, got, err := conn.ReadMessage() // all three, batched
	is.NoErr(err)
	is.Equal(len(got), 9)

	var writes []sdktrace.ReadOnlySpan
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		writes = writes[:0]
		for _, s := range rec.Ended() {
			if s.Name() == "socket.write" {
				writes = append(writes, s)
			}
		}
		if len(writes) == 2 {
			break
		}
	}
	is.Equal(len(writes), 2)
	for _, w := range writes {
		is.Equal(w.Parent().SpanID(), fwd.SpanContext().SpanID())
		attrs := attrMap(w)
		is.Equal(attrs["conn.id"].AsString(), "c1")
		is.Equal(attrs["batch.messages"].AsInt64(), int64(3))
		is.Equal(attrs["msg.bytes"].AsInt64(), int64(3))
	}
}

func spanNamed(rec *tracetest.SpanRecorder, name string) sdktrace.ReadOnlySpan {
	for _, s := range rec.Ended() {
		if s.Name() == name {
			return s
		}
	}
	return nil
}

func attrMap(s sdktrace.ReadOnlySpan) map[attribute.Key]attribute.Value {
	attrs := map[attribute.Key]attribute.Value{}
	for _, kv := range s.Attributes() {
		attrs[kv.Key] = kv.Value
	}
	return attrs
}
