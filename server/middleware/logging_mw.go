package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/ognev-dev/goplease/server/handler"
)

// Logging is a middleware that logs the start time, HTTP method, and path of an incoming request,
// and then logs the completion time and total duration after the next handler has executed.
func (mw *Middleware) Logging(next handler.Fn) handler.Fn {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next(w, r)
		log.Printf("[%s] %s %s", time.Since(start), r.Method, r.URL.Path) //nolint:gosec
	}
}
