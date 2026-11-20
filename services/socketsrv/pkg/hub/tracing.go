package sockets

import (
	"context"
	"strings"
	"time"

	nats "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/woogles-io/liwords/pkg/natstrace"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// The socket server's spans. A message from a browser starts a trace
// (socket.receive.<type>) that goes to the API in the NATS message's
// headers. What the API publishes back in that trace is forwarded under a
// socket.forward.<subject prefix> span, which ends once the message is
// queued for each of its sockets. Each socket's write of it is then a
// socket.write span under that, unless the message went to more than
// maxTracedWrites sockets. Messages without a trace get no spans.
//
// The tracer is looked up on each use, not kept: the provider is set after
// this package loads (and replaced in tests).
func tracer() trace.Tracer { return otel.Tracer("socketsrv") }

// maxTracedWrites is the most sockets a message can go to and still get a
// socket.write span for each: a lobby broadcast reaches hundreds.
const maxTracedWrites = 20

// receiveSpan starts the span for a message from a browser.
func receiveSpan(ctx context.Context, c *Client, msgType byte) (context.Context, trace.Span) {
	return tracer().Start(ctx, "socket.receive."+pb.MessageType(msgType).String(),
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.Int("msg.type", int(msgType)),
			attribute.String("user.id", c.userID),
			attribute.String("conn.id", c.connID),
			attribute.Bool("user.authenticated", c.authenticated),
		))
}

// forwardContext starts the span for forwarding msg to browsers, in the
// trace msg carries, and returns a context holding it. If msg carries no
// trace, it returns context.Background().
func forwardContext(msg *nats.Msg) context.Context {
	ctx := natstrace.Extract(context.Background(), msg)
	if !trace.SpanContextFromContext(ctx).IsValid() {
		return context.Background()
	}
	prefix, _, _ := strings.Cut(msg.Subject, ".")
	attrs := []attribute.KeyValue{attribute.String("msg.subject", msg.Subject)}
	if len(msg.Data) > 2 {
		attrs = append(attrs, attribute.String("msg.type", pb.MessageType(msg.Data[2]).String()))
	}
	ctx, _ = tracer().Start(ctx, "socket.forward."+prefix,
		trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attrs...))
	return ctx
}

// endForward ends ctx's forward span, if it has one, with the number of
// sockets the message was queued for.
func endForward(ctx context.Context, sockets int) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	span.SetAttributes(attribute.Int("sockets", sockets))
	span.End()
}

// An outMsg is a message queued for a socket. If trace is valid, the write
// loop records its write as a socket.write span under it.
type outMsg struct {
	data   []byte
	trace  trace.SpanContext
	queued time.Time
}

// outgoing returns msg ready to queue for one of `sockets` sockets, with
// ctx's forward span to record the write under if there are few enough.
func outgoing(ctx context.Context, msg []byte, sockets int) outMsg {
	out := outMsg{data: msg}
	if span := trace.SpanFromContext(ctx); sockets <= maxTracedWrites && span.IsRecording() {
		out.trace = span.SpanContext()
		out.queued = time.Now()
	}
	return out
}

// recordWrites records a socket.write span for each traced message in
// msgs, written together in one batch of `batched` messages: from when it
// was queued, through the start of the write at start, to now. err is the
// write's error, if any.
func recordWrites(c *Client, msgs []outMsg, start time.Time, batched int, err error) {
	end := time.Now()
	for _, m := range msgs {
		ctx := trace.ContextWithSpanContext(context.Background(), m.trace)
		_, span := tracer().Start(ctx, "socket.write",
			trace.WithTimestamp(m.queued),
			trace.WithAttributes(
				attribute.String("user.id", c.userID),
				attribute.String("conn.id", c.connID),
				attribute.Int("msg.bytes", len(m.data)),
				attribute.Int("batch.messages", batched),
				attribute.Int64("queue_wait_us", start.Sub(m.queued).Microseconds()),
				attribute.Int64("write_us", end.Sub(start).Microseconds()),
			))
		span.AddEvent("write.start", trace.WithTimestamp(start))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "write failed")
		}
		span.End(trace.WithTimestamp(end))
	}
}
