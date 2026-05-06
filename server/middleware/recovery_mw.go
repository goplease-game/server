package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/ognev-dev/goplease/server/handler"
)

// Recovery is a middleware that wraps the execution of the next handler in a defer function
// with recover(), catching any runtime panics that occur during request processing.
func (mw *Middleware) Recovery(next handler.Fn) handler.Fn {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Printf("[%s %s] Recovered from panic: %s", r.Method, r.URL.Path, err) //nolint:gosec
				log.Println(string(debug.Stack()))
			}
		}()

		next(w, r)
	}
}
