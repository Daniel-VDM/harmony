package server

import (
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/pkg/errors"

	"github.com/harmony-one/harmony/internal/utils"
)

// Export existing useful middlewares
var (
	// LoggerMiddleware to log all rosetta requests
	LoggerMiddleware = server.LoggerMiddleware
	// CorsMiddleware handles CORS
	CorsMiddleware = server.CorsMiddleware
)

func RecoverMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		defer func() {
			r := recover()
			if r != nil {
				switch t := r.(type) {
				case string:
					err = errors.New(t)
				case error:
					err = t
				default:
					err = errors.New("unknown error")
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				utils.Logger().Error().Err(err).Msg("Rosetta Error")
				// Print to stderr for quick check of rosetta activity
				debug.PrintStack()
				_, _ = fmt.Fprintf(
					os.Stderr, "%s PANIC: %s\n", time.Now().Format("2006-01-02 15:04:05"), err.Error(),
				)
			}
		}()
		h.ServeHTTP(w, r)
	})
}
