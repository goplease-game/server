package middleware

import (
	"net/http"

	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/server/handler"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// StatusRecorder is a wrapper around http.ResponseWriter
// that records the HTTP status code and potentially the response body error message.
type StatusRecorder struct {
	http.ResponseWriter

	status int
	err    string
}

// WriteHeader intercepts the call to set the status code.
// It records the status and then calls the embedded (original) WriteHeader.
func (rec *StatusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

// Write must also be implemented to ensure that http.ResponseWriter interface is satisfied.
// It sets the default status (200 OK) if no header was written yet.
// If the status is >= 400 (Client Error), it records the response body as a potential error message.
func (rec *StatusRecorder) Write(b []byte) (int, error) {
	if rec.status == 0 {
		rec.status = http.StatusOK
	}

	if rec.status >= http.StatusBadRequest {
		rec.err = string(b)
	}

	return rec.ResponseWriter.Write(b)
}

// Tracing creates a new span and injects it into the request's context to propagate down the chain.
func (mw *Middleware) Tracing(next handler.Fn) handler.Fn {
	return func(w http.ResponseWriter, r *http.Request) {
		if app.Config().TracingDisabled() {
			next(w, r)
			return
		}

		ctx, span := mw.tracer.Start(r.Context(), r.URL.Path)
		defer span.End()

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", r.Pattern),
		)

		rec := &StatusRecorder{ResponseWriter: w}
		r = r.WithContext(ctx)
		next(rec, r)

		span.SetAttributes(
			attribute.Int("http.status_code", rec.status),
		)

		if rec.status >= http.StatusBadRequest {
			span.SetStatus(codes.Error, rec.err)
		}
	}
}
