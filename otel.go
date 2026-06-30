package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	logglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	// otelServiceName is the default service.name reported to OpenTelemetry. It
	// is overridden by the OTEL_SERVICE_NAME environment variable.
	otelServiceName = "whoami"
	// otelInstrumentationName is the instrumentation scope used for the meter,
	// tracer, and logger created by whoami.
	otelInstrumentationName = "github.com/traefik/whoami"
	// consoleExporter is the standard OTEL_*_EXPORTER value for stdout output.
	consoleExporter = "console"
)

// setupOTel configures the global OpenTelemetry tracer, meter, and logger
// providers from the standard OTEL_* environment variables.
//
// Logs are emitted by default to stdout (OTEL_LOGS_EXPORTER=console) so whoami
// always prints structured application and access logs. Traces and metrics are
// opt-in: set OTEL_TRACES_EXPORTER and/or OTEL_METRICS_EXPORTER (e.g. to "otlp")
// to ship them. Endpoint, protocol, headers, and resource attributes are read
// from the usual OTEL_EXPORTER_OTLP_* and OTEL_RESOURCE_ATTRIBUTES variables.
//
// The returned function flushes and shuts down every provider that was started.
func setupOTel(ctx context.Context) (func(context.Context) error, error) {
	// Logs default to stdout; traces and metrics are opt-in (disabled until an
	// exporter is requested). "stdout" is accepted as an alias for "console".
	setEnvDefault("OTEL_TRACES_EXPORTER", "none")
	setEnvDefault("OTEL_METRICS_EXPORTER", "none")
	setEnvDefault("OTEL_LOGS_EXPORTER", consoleExporter)
	if os.Getenv("OTEL_LOGS_EXPORTER") == "stdout" {
		_ = os.Setenv("OTEL_LOGS_EXPORTER", consoleExporter)
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	res, err := newResource(ctx)
	if err != nil {
		return nil, err
	}

	var shutdownFuncs []func(context.Context) error
	shutdown := func(sctx context.Context) error {
		var errs error
		for _, fn := range shutdownFuncs {
			errs = errors.Join(errs, fn(sctx))
		}

		return errs
	}

	setups := []func(context.Context, *resource.Resource) ([]func(context.Context) error, error){
		setupTraces,
		setupMetrics,
		setupLogs,
	}
	for _, setup := range setups {
		var fns []func(context.Context) error

		fns, err = setup(ctx, res)
		if err != nil {
			return nil, errors.Join(err, shutdown(ctx))
		}

		shutdownFuncs = append(shutdownFuncs, fns...)
	}

	initLogger()

	return shutdown, nil
}

// newResource describes the running whoami instance. service.name defaults to
// "whoami" and service.version to the build version; both are overridden by
// OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES.
func newResource(ctx context.Context) (*resource.Resource, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", otelServiceName),
			attribute.String("service.version", version),
		),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("building OpenTelemetry resource: %w", err)
	}

	return res, nil
}

// setupTraces wires a tracer provider when OTEL_TRACES_EXPORTER is enabled.
func setupTraces(ctx context.Context, res *resource.Resource) ([]func(context.Context) error, error) {
	exporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating span exporter: %w", err)
	}

	if autoexport.IsNoneSpanExporter(exporter) {
		return nil, nil
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)

	return []func(context.Context) error{provider.Shutdown}, nil
}

// setupMetrics wires a meter provider when OTEL_METRICS_EXPORTER is enabled.
func setupMetrics(ctx context.Context, res *resource.Resource) ([]func(context.Context) error, error) {
	reader, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating metric reader: %w", err)
	}

	if autoexport.IsNoneMetricReader(reader) {
		return nil, nil
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(provider)

	// Emit Go runtime metrics (memory, GC, goroutines) alongside the
	// auto-instrumented HTTP and gRPC metrics.
	err = runtime.Start(runtime.WithMeterProvider(provider))
	if err != nil {
		return nil, fmt.Errorf("starting runtime metrics: %w", err)
	}

	return []func(context.Context) error{provider.Shutdown}, nil
}

// setupLogs wires a logger provider, defaulting to stdout. The console exporter
// uses a synchronous processor so records print immediately and in order, while
// OTLP batches. The provider is published globally for the log helpers to use.
func setupLogs(ctx context.Context, res *resource.Resource) ([]func(context.Context) error, error) {
	exporter, err := autoexport.NewLogExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating log exporter: %w", err)
	}

	if autoexport.IsNoneLogExporter(exporter) {
		return nil, nil
	}

	var processor sdklog.Processor = sdklog.NewBatchProcessor(exporter)
	if os.Getenv("OTEL_LOGS_EXPORTER") == consoleExporter {
		processor = sdklog.NewSimpleProcessor(exporter)
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(processor),
		sdklog.WithResource(res),
	)
	logglobal.SetLoggerProvider(provider)

	return []func(context.Context) error{provider.Shutdown}, nil
}

// setEnvDefault sets an environment variable only when it is not already set.
func setEnvDefault(key, value string) {
	if _, ok := os.LookupEnv(key); !ok {
		_ = os.Setenv(key, value)
	}
}
