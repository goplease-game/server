package tracing

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// NewNoOpTracer initializes the OpenTelemetry global TracerProvider
// with a No-Operation (NoOp) provider.
//
// A NoOp Tracer is used when tracing is explicitly disabled in the application
// configuration. It satisfies the OpenTelemetry interface but performs no tracing
// or data collection, ensuring that tracing calls in the application code
// do not cause errors or overhead.
func NewNoOpTracer() trace.Tracer {
	otel.SetTracerProvider(noop.NewTracerProvider())

	return otel.Tracer("noop")
}
