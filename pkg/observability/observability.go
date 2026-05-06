// Package observability wires structured logging and OpenTelemetry tracing.
//
// One Init call per service main(): returns a zerolog logger plus a shutdown
// func that flushes pending traces. Logs are JSON; trace_id and span_id are
// injected when a span is active so log lines correlate to Jaeger traces.
package observability

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Shutdown flushes any pending telemetry. Call from main() defer.
type Shutdown func(context.Context) error

// Init configures global tracer + returns a logger.
// otlpEndpoint is e.g. "http://localhost:4318" (Jaeger OTLP HTTP).
// If otlpEndpoint is empty, tracing is disabled and Shutdown is a no-op.
func Init(ctx context.Context, serviceName, otlpEndpoint string) (zerolog.Logger, Shutdown, error) {
	logger := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("service", serviceName).
		Logger()

	if otlpEndpoint == "" {
		return logger, func(context.Context) error { return nil }, nil
	}

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(otlpEndpoint+"/v1/traces"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return logger, nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		return logger, nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp, sdktrace.WithBatchTimeout(2*time.Second)),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return logger, tp.Shutdown, nil
}
