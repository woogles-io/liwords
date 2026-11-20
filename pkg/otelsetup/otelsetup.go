// Package otelsetup sets up tracing the same way for every liwords process:
// W3C trace context propagation, and spans sent over OTLP/HTTP to the
// endpoint in OTEL_EXPORTER_OTLP_ENDPOINT. The service name and version come
// from OTEL_SERVICE_NAME / OTEL_RESOURCE_ATTRIBUTES.
package otelsetup

import (
	"context"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Tracing sets the global propagator and tracer provider. Call the returned
// shutdown before exiting, to send the spans still buffered.
func Tracing(ctx context.Context) (shutdown func(context.Context) error, err error) {
	otel.SetTextMapPropagator(Propagator())
	tp, err := tracerProvider(ctx)
	if err != nil {
		return nil, err
	}
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// Propagator is how a trace is carried between processes: in HTTP and NATS
// headers.
func Propagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func secure() bool {
	return strings.HasPrefix(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), "https://") ||
		strings.ToLower(os.Getenv("OTEL_EXPORTER_OTLP_INSECURE")) == "false"
}

func tracerProvider(ctx context.Context) (*trace.TracerProvider, error) {
	var opts []otlptracehttp.Option
	if !secure() {
		opts = []otlptracehttp.Option{otlptracehttp.WithInsecure()}
	}
	exporter, err := otlptrace.New(ctx, otlptracehttp.NewClient(opts...))
	if err != nil {
		return nil, err
	}
	res, err := traceResource(ctx)
	if err != nil {
		return nil, err
	}
	return trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	), nil
}

// traceResource describes this process on every span: what the default
// resource has (service name and version from OTEL_SERVICE_NAME and
// OTEL_RESOURCE_ATTRIBUTES, the SDK), plus host.name, the container's
// hostname, so spans from different servers can be told apart.
func traceResource(ctx context.Context) (*resource.Resource, error) {
	host, err := resource.New(ctx, resource.WithHost())
	if err != nil {
		return nil, err
	}
	return resource.Merge(resource.Default(), host)
}
