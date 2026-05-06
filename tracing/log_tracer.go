package tracing

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// NewLogTracer returns a new instance of Log tracer.
func NewLogTracer() trace.Tracer {
	return new(Log)
}

// Log is a custom OpenTelemetry Tracer implementation primarily used for debugging.
// It embeds noop.Tracer to satisfy the trace.Tracer interface without performing
// standard OpenTelemetry collection/export logic.
// Instead, it logs the caller's function name and file location when a span is started.
type Log struct {
	noop.Tracer
}

// Start implements the trace.Tracer interface method.
// Instead of creating a real distributed trace span, this method uses reflection
// (runtime.Caller) to determine the function and file line that called the tracer,
// and prints this information to the console for debugging purposes.
func (t *Log) Start(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
	// We use Caller(2) because:
	// 0: runtime.Caller
	// 1: t.Start (this function)
	// 2: The actual function that called t.Start
	pc, file, line, ok := runtime.Caller(2) //nolint:mnd
	if ok {
		// Get the full function name and simplify it to  package/function_name
		names := strings.Split(runtime.FuncForPC(pc).Name(), "/")

		wd, err := os.Getwd()
		if err == nil {
			relPath, err := filepath.Rel(wd, file)
			if err == nil {
				file = relPath
			}
		}

		name := names[len(names)-1]
		name = strings.Replace(name, "middleware.(*Middleware)", "MW", 1)
		name = strings.Replace(name, "service.(*Service)", "Service", 1)
		// Print the function name and file location
		println("> " + name)
		println("@ " + file + ":" + strconv.Itoa(line))
		println("")
	}

	return ctx, noop.Span{}
}
