// Package telemetry wires OpenTelemetry for every entry point of the system.
//
// Instrumentation is always installed; exporting only happens when an OTLP
// endpoint is configured, so a developer without a collector still gets a
// working process.
package telemetry

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// Shutdown releases the telemetry pipeline. It is safe to call it once.
type Shutdown func(context.Context) error

// Setup installs the global tracer and meter providers.
//
// When endpoint is empty the providers are still installed, but nothing is
// exported and the returned Shutdown is a no-op.
func Setup(ctx context.Context, serviceName, environment, endpoint string) (Shutdown, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.DeploymentEnvironmentName(environment),
	))
	if err != nil {
		return nil, fmt.Errorf("telemetry: cannot build the resource: %w", err)
	}

	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("telemetry: cannot create the trace exporter: %w", err)
	}

	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("telemetry: cannot create the metric exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)

	return func(shutdownCtx context.Context) error {
		return errors.Join(
			tracerProvider.Shutdown(shutdownCtx),
			meterProvider.Shutdown(shutdownCtx),
		)
	}, nil
}
