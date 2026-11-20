// Package natstrace carries a trace across NATS, in message headers, so one
// trace can follow a message from the socket server through the API and back.
//
// It uses the process's global OpenTelemetry propagator (W3C traceparent).
// A message is given headers only when its context has a trace; every other
// message is published exactly as before. NATS servers older than 2.2 can't
// carry headers.
package natstrace

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// NewMsg returns a message for subject carrying data and, if ctx has a
// trace, that trace in its headers.
func NewMsg(ctx context.Context, subject string, data []byte) *nats.Msg {
	msg := &nats.Msg{Subject: subject, Data: data}
	if trace.SpanContextFromContext(ctx).IsValid() {
		msg.Header = nats.Header{}
		otel.GetTextMapPropagator().Inject(ctx, carrier(msg.Header))
	}
	return msg
}

// Publish publishes data to subject, with ctx's trace if it has one.
func Publish(ctx context.Context, nc *nats.Conn, subject string, data []byte) error {
	if !trace.SpanContextFromContext(ctx).IsValid() {
		return nc.Publish(subject, data)
	}
	return nc.PublishMsg(NewMsg(ctx, subject, data))
}

// Extract returns ctx with the trace from msg's headers, if it has one.
func Extract(ctx context.Context, msg *nats.Msg) context.Context {
	if len(msg.Header) == 0 {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, carrier(msg.Header))
}

// Traced reports whether msg carries a trace.
func Traced(msg *nats.Msg) bool {
	return trace.SpanContextFromContext(Extract(context.Background(), msg)).IsValid()
}

// carrier reads and writes NATS headers. Unlike HTTP headers, NATS header
// names are case-sensitive, so the keys are written exactly as the
// propagator names them ("traceparent"), not canonicalized.
type carrier nats.Header

var _ propagation.TextMapCarrier = carrier{}

func (c carrier) Get(key string) string {
	if v := c[key]; len(v) > 0 {
		return v[0]
	}
	return ""
}

func (c carrier) Set(key, value string) { c[key] = []string{value} }

func (c carrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
