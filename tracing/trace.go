// Package tracing provides OpenTelemetry tracer initialization utilities.
package tracing

import (
	"context"
	"errors"
	"fmt"

	"github.com/ognev-dev/goplease/app"
	"go.opentelemetry.io/otel/trace"
)

var (
	// ErrInvalidDriver indicates that the configured tracing driver name is not recognized or supported.
	ErrInvalidDriver = errors.New("invalid trace driver")
)

const (
	// UptraceDriver is used to identify the Uptrace OpenTelemetry implementation.
	// See: https://uptrace.dev/get/opentelemetry-go
	UptraceDriver = "uptrace"

	// LogDriver identifies a basic tracer that writes span information to logs.
	LogDriver = "log"
)

// New initializes and returns the configured OpenTelemetry Tracer based on the application settings.
// If tracing is disabled in the configuration, it returns a no-op tracer.
func New(ctx context.Context) (t trace.Tracer, err error) {
	conf := app.Config()
	if conf.TracingDisabled() {
		return NewNoOpTracer(), nil
	}

	switch app.Config().Tracing.Driver {
	case UptraceDriver:
		t = NewUptraceTracer(ctx)
	case LogDriver:
		t = NewLogTracer()
	default:
		err = fmt.Errorf("driver '%s': %w", conf.Tracing.Driver, ErrInvalidDriver)
	}

	return
}
