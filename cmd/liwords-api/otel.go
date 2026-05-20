package main

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

// setupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
//
// NOTE: metric pipeline is currently disabled. The stdoutmetric exporter with
// cumulative temporality retains one record per unique attribute combination in
// memory forever. The db.statement view filter was a no-op (otelpgx registers a
// tracer, not metrics). Root cause of the OOM has not been heap-profiled, but
// disabling metrics is the conservative kill-switch.
// TODO(metrics-leak): re-enable after heap-profiling identifies the offender.
func setupOTelSDK(ctx context.Context) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider.
	tracerProvider, err := newTraceProvider()
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// TODO(metrics-leak): meter provider disabled — see comment on setupOTelSDK.
	// meterProvider, err := newMeterProvider()
	// if err != nil {
	// 	handleErr(err)
	// 	return
	// }
	// shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	// otel.SetMeterProvider(meterProvider)

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func withSecure() bool {
	return strings.HasPrefix(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), "https://") ||
		strings.ToLower(os.Getenv("OTEL_EXPORTER_OTLP_INSECURE")) == "false"
}

func newTraceProvider() (*trace.TracerProvider, error) {
	var opts []otlptracehttp.Option
	if !withSecure() {
		opts = []otlptracehttp.Option{otlptracehttp.WithInsecure()}
	}

	traceExporter, err := otlptrace.New(context.Background(),
		otlptracehttp.NewClient(opts...))
	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
	)
	return traceProvider, nil
}

func newMeterProvider() (*metric.MeterProvider, error) {
	// Exports metrics as JSON to stdout → CloudWatch Logs → queryable via Logs Insights.
	metricExporter, err := stdoutmetric.New()
	if err != nil {
		return nil, err
	}

	// Drop db.statement from histogram cardinality: query parameter values
	// are included in trace spans (where cardinality is fine) but embedding
	// them in metric attributes would create an unbounded number of time series
	// (one per unique game UUID / event JSON combination) and OOM the process.
	// Traces keep full parameter visibility; histograms are grouped only by
	// query template (db.operation.name, db.collection.name, etc.).
	noDB := metric.NewView(
		metric.Instrument{Name: "db.client.operation.duration"},
		metric.Stream{
			AttributeFilter: func(kv attribute.KeyValue) bool {
				return kv.Key != "db.statement"
			},
		},
	)

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter)),
		metric.WithView(noDB),
	)
	return meterProvider, nil
}

// registerRuntimeMetrics registers observable gauges for Go runtime memory stats.
// Call once after setupOTelSDK so the global MeterProvider is set.
// Samples runtime.MemStats on each metric collection interval (default 60s for stdout).
func registerRuntimeMetrics() {
	meter := otel.GetMeterProvider().Meter("liwords-api")

	var memStats runtime.MemStats

	_, _ = meter.Int64ObservableGauge(
		"go.runtime.heap_alloc_bytes",
		otelmetric.WithInt64Callback(func(_ context.Context, o otelmetric.Int64Observer) error {
			runtime.ReadMemStats(&memStats)
			o.Observe(int64(memStats.HeapAlloc))
			return nil
		}),
		otelmetric.WithDescription("Bytes of allocated heap objects"),
		otelmetric.WithUnit("By"),
	)
	_, _ = meter.Int64ObservableGauge(
		"go.runtime.heap_inuse_bytes",
		otelmetric.WithInt64Callback(func(_ context.Context, o otelmetric.Int64Observer) error {
			o.Observe(int64(memStats.HeapInuse))
			return nil
		}),
		otelmetric.WithDescription("Bytes in in-use heap spans"),
		otelmetric.WithUnit("By"),
	)
	_, _ = meter.Int64ObservableGauge(
		"go.runtime.num_gc",
		otelmetric.WithInt64Callback(func(_ context.Context, o otelmetric.Int64Observer) error {
			o.Observe(int64(memStats.NumGC))
			return nil
		}),
		otelmetric.WithDescription("Number of completed GC cycles"),
	)
	_, _ = meter.Int64ObservableGauge(
		"go.runtime.pause_total_ns",
		otelmetric.WithInt64Callback(func(_ context.Context, o otelmetric.Int64Observer) error {
			o.Observe(int64(memStats.PauseTotalNs))
			return nil
		}),
		otelmetric.WithDescription("Cumulative nanoseconds in GC stop-the-world pauses"),
		otelmetric.WithUnit("ns"),
	)
}
