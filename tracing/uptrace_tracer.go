package tracing

import (
	"context"
	"log"
	"time"

	"github.com/ognev-dev/goplease/app"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// ShutdownTimeout ...
const ShutdownTimeout = 5 * time.Second

// NewUptraceTracer initializes and configures the OpenTelemetry tracing provider
// to send traces to Uptrace.
func NewUptraceTracer(ctx context.Context) trace.Tracer {
	conf := app.Config()
	uptrace.ConfigureOpentelemetry(
		uptrace.WithDSN(conf.Tracing.UptraceDSN),
		uptrace.WithServiceName(conf.App.Name),
		uptrace.WithServiceVersion(conf.App.Version),
		uptrace.WithDeploymentEnvironment(conf.App.Env),
	)

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()

		// Send buffered spans and free resources.

		// TODO: FIX: (i don't get it)
		// Non-inherited new context, use function like `context.WithXXX` instead (contextcheck)
		err := uptrace.Shutdown(shutdownCtx)
		if err != nil {
			log.Printf("uptrace.Shutdown failed: %v\n", err)
		}
	}()

	return otel.Tracer(conf.App.Name)
}
