// Package middleware ...
package middleware

import (
	"net/http"

	"github.com/ognev-dev/goplease/app/service"
	"github.com/ognev-dev/goplease/server/handler"
	"go.opentelemetry.io/otel/trace"
)

// Fn is the function signature for a middleware.
type Fn func(handler.Fn) handler.Fn

// Middleware holds dependencies required by middlewares.
type Middleware struct {
	service *service.Service
	tracer  trace.Tracer
}

// New creates and returns a new Middleware instance.
func New(service *service.Service, t trace.Tracer) *Middleware {
	return &Middleware{
		service: service,
		tracer:  t,
	}
}

// ServeJSON forces JSON responses for all handlers in the chain.
// It sets the response Content-Type and marks the request context.
func (mw *Middleware) ServeJSON(next handler.Fn) handler.Fn {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		next(w, handler.SetServerJSON(r))
	}
}
