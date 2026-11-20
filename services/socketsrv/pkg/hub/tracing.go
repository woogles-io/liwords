package sockets

import (
	"context"
	"strings"

	nats "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/woogles-io/liwords/pkg/natstrace"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// The socket server's spans. A message from a browser starts a trace
// (socket.receive.<type>) that goes to the API in the NATS message's
// headers. What the API publishes back in that trace is forwarded under a
// socket.forward.<subject prefix> span, which ends once the message is
// queued for each of its sockets. Messages without a trace get no span.
//
// The tracer is looked up on each use, not kept: the provider is set after
// this package loads (and replaced in tests).
func tracer() trace.Tracer { return otel.Tracer("socketsrv") }

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

// forwardSpan starts the span for forwarding msg to browsers, in the trace
// msg carries; nil if it carries none.
func forwardSpan(msg *nats.Msg) trace.Span {
	ctx := natstrace.Extract(context.Background(), msg)
	if !trace.SpanContextFromContext(ctx).IsValid() {
		return nil
	}
	prefix, _, _ := strings.Cut(msg.Subject, ".")
	attrs := []attribute.KeyValue{attribute.String("msg.subject", msg.Subject)}
	if len(msg.Data) > 2 {
		attrs = append(attrs, attribute.String("msg.type", pb.MessageType(msg.Data[2]).String()))
	}
	_, span := tracer().Start(ctx, "socket.forward."+prefix,
		trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attrs...))
	return span
}

// endForward ends a forward span, if there is one, with the number of
// sockets the message was queued for.
func endForward(span trace.Span, sockets int) {
	if span == nil {
		return
	}
	span.SetAttributes(attribute.Int("sockets", sockets))
	span.End()
}
