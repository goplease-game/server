// Package endpoint ...
package endpoint

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/frontend"
	"github.com/ognev-dev/goplease/server/handler"
	"github.com/ognev-dev/goplease/server/middleware"
)

const exactMatchSuffix = "{$}"

// Router manages the application's routing table, handles middleware application,
// and delegates requests to the standard http.ServeMux.
type Router struct {
	basePath    string
	mux         *http.ServeMux
	mw          *middleware.Middleware
	handler     *handler.Handler
	middlewares []middleware.Fn
}

// NewRouter initializes and returns a new Router instance.
func NewRouter(mw *middleware.Middleware, h *handler.Handler) *Router {
	return &Router{
		basePath:    "/",
		mux:         http.NewServeMux(),
		mw:          mw,
		handler:     h,
		middlewares: []middleware.Fn{},
	}
}

// Group creates a new Router instance with a base path appended to the current router's base path.
// The new router shares the underlying http.ServeMux and middleware stack.
func (r *Router) Group(pattern string, mws ...middleware.Fn) *Router {
	return &Router{
		basePath:    path.Join(r.basePath, pattern),
		mux:         r.mux,
		handler:     r.handler,
		mw:          r.mw,
		middlewares: append(r.middlewares, mws...),
	}
}

// Use appends one or more Middleware functions to the router's middleware stack.
func (r *Router) Use(mw ...middleware.Fn) *Router {
	r.middlewares = append(r.middlewares, mw...)

	return r
}

// GET registers a handler for the HTTP GET method at the specified pattern relative to the base path.
func (r *Router) GET(pattern string, handler handler.Fn) *Router {
	r.register(http.MethodGet, pattern, handler)

	return r
}

// POST registers a handler for the HTTP POST method at the specified pattern relative to the base path.
func (r *Router) POST(pattern string, handler handler.Fn) *Router {
	r.register(http.MethodPost, pattern, handler)

	return r
}

// PUT registers a handler for the HTTP PUT method at the specified pattern relative to the base path.
func (r *Router) PUT(pattern string, handler handler.Fn) *Router {
	r.register(http.MethodPut, pattern, handler)

	return r
}

// DELETE registers a handler for the HTTP DELETE method at the specified pattern relative to the base path.
func (r *Router) DELETE(pattern string, handler handler.Fn) *Router {
	r.register(http.MethodDelete, pattern, handler)

	return r
}

// HandleAssets registers the handler for serving static assets.
// It uses a different approach depending on the environment:
//   - In development (`dev`), it serves files directly from the local disk
//     to allow for live reloading without recompiling the application.
//   - In production (`prod`) or any other environment, it serves files from
//     the embedded filesystem (`embed.FS`) for a self-contained binary.
func (r *Router) HandleAssets() *Router {
	var h http.Handler

	if app.Config().IsDevEnv() {
		localAssets := http.Dir("./frontend/assets/")
		h = http.StripPrefix("/assets/", http.FileServer(localAssets))
	} else {
		h = http.FileServer(http.FS(frontend.AssetsFs))
	}

	r.mux.Handle("GET /assets/", h)
	return r
}

// ServeHTTP implements the http.Handler interface, delegating the request handling
// to the underlying http.ServeMux.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *Router) applyMWsToHandler(h handler.Fn) handler.Fn {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}

	return h
}

func (r *Router) register(method, pattern string, h handler.Fn) {
	h = r.applyMWsToHandler(h)

	withSlash := strings.HasSuffix(pattern, "/")
	pattern = path.Join(r.basePath, pattern)

	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	if withSlash && !strings.HasSuffix(pattern, "/") {
		pattern += "/" + exactMatchSuffix
	}

	if pattern == "/" {
		pattern += exactMatchSuffix
	}

	pattern = method + " " + pattern
	fmt.Println(pattern)

	r.mux.Handle(pattern, http.HandlerFunc(h))
}
